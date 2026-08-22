package store

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// CollectionSchema defines a PocketBase collection declaratively.
type CollectionSchema struct {
	Name        string
	Type        string // "base" or "auth"
	ListRule    *string
	ViewRule    *string
	CreateRule  *string
	UpdateRule  *string
	DeleteRule  *string
	Fields      []FieldSchema
	Indexes     []IndexSchema
}

// FieldSchema defines a field in a collection.
type FieldSchema struct {
	Name        string
	Type        string // "text", "number", "select", "relation", "date", "autodate", "json", "file", "email", "bool"
	Required    bool
	Max         int
	MaxSelect   int
	Values      []string // for select fields
	Collection  string   // for relation fields
	CascadeDel  bool     // for relation fields
	OnCreate    bool     // for autodate
	OnUpdate    bool     // for autodate
	MaxSize     int64    // for file fields
	MimeTypes   []string // for file fields
}

// IndexSchema defines an index on a collection.
type IndexSchema struct {
	Name    string
	Unique  bool
	Fields  string
	Exp     string
}

// schemaRegistry holds all collection schemas.
var schemaRegistry []CollectionSchema

// RegisterSchema registers a collection schema.
func RegisterSchema(schema CollectionSchema) {
	schemaRegistry = append(schemaRegistry, schema)
}

// EnsureSchema creates or updates all registered collections using a two-pass approach:
// 1. Create all collections with non-relation fields
// 2. Add relation fields after all collections exist
// 3. Resolve relation CollectionIds
func EnsureSchema(app core.App) error {
	// Pass 1: Create collections with non-relation fields
	for _, schema := range schemaRegistry {
		if err := ensureCollectionNonRelations(app, schema); err != nil {
			return err
		}
	}

	// Pass 2: Add relation fields now that all collections exist
	for _, schema := range schemaRegistry {
		if err := addRelationFields(app, schema); err != nil {
			return err
		}
	}

	// Pass 3: Resolve CollectionIds for relation fields
	if err := ResolveRelationFields(app); err != nil {
		return err
	}

	return nil
}

func ensureCollectionNonRelations(app core.App, schema CollectionSchema) error {
	// Find or create collection
	var col *core.Collection
	if existing, err := app.FindCollectionByNameOrId(schema.Name); err == nil {
		col = existing
	} else {
		var newCol *core.Collection
		if schema.Type == "auth" {
			newCol = core.NewAuthCollection(schema.Name)
		} else {
			newCol = core.NewBaseCollection(schema.Name)
		}
		col = newCol
	}

	// Add non-relation fields
	existingFieldNames := make(map[string]bool)
	for _, f := range col.Fields {
		existingFieldNames[f.GetName()] = true
	}

	for _, fs := range schema.Fields {
		if existingFieldNames[fs.Name] {
			continue // field already exists
		}
		if strings.ToLower(fs.Type) == "relation" {
			continue // skip relation fields in pass 1
		}
		field := createFieldFromSchema(fs)
		if field != nil {
			col.Fields.Add(field)
		}
	}

	// Save collection (without rules or indexes for now - will be set in pass 2)
	if err := app.Save(col); err != nil {
		return fmt.Errorf("failed to save collection %s: %w", schema.Name, err)
	}
	return nil
}

func addRelationFields(app core.App, schema CollectionSchema) error {
	col, err := app.FindCollectionByNameOrId(schema.Name)
	if err != nil {
		return fmt.Errorf("collection %s not found: %w", schema.Name, err)
	}

	existingFieldNames := make(map[string]bool)
	for _, f := range col.Fields {
		existingFieldNames[f.GetName()] = true
	}

	hasNewFields := false
	for _, fs := range schema.Fields {
		if existingFieldNames[fs.Name] {
			continue
		}
		if strings.ToLower(fs.Type) != "relation" {
			continue
		}
		field := createFieldFromSchema(fs)
		if field != nil {
			// Try to resolve CollectionId immediately if target collection exists
			if rel, ok := field.(*core.RelationField); ok && fs.Collection != "" {
				if refCol, err := app.FindCollectionByNameOrId(fs.Collection); err == nil {
					rel.CollectionId = refCol.Id
				}
			}
			col.Fields.Add(field)
			hasNewFields = true
		}
	}

	// Add indexes after all fields are present
	for _, is := range schema.Indexes {
		hasIndex := false
		for _, idx := range col.Indexes {
			if strings.Contains(idx, "`"+is.Name+"`") {
				hasIndex = true
				break
			}
		}
		if !hasIndex {
			col.AddIndex(is.Name, is.Unique, is.Fields, is.Exp)
			hasNewFields = true
		}
	}

	// Now set rules after all fields are present
	col.ListRule = schema.ListRule
	col.ViewRule = schema.ViewRule
	col.CreateRule = schema.CreateRule
	col.UpdateRule = schema.UpdateRule
	col.DeleteRule = schema.DeleteRule

	if hasNewFields || col.ListRule != nil || col.ViewRule != nil || col.CreateRule != nil || col.UpdateRule != nil || col.DeleteRule != nil {
		if err := app.Save(col); err != nil {
			return fmt.Errorf("failed to save collection %s after adding relation fields, indexes and rules: %w", schema.Name, err)
		}
	}
	return nil
}

func createFieldFromSchema(fs FieldSchema) core.Field {
	switch strings.ToLower(fs.Type) {
	case "text":
		f := &core.TextField{Name: fs.Name, Required: fs.Required}
		if fs.Max > 0 {
			f.Max = fs.Max
		}
		return f
	case "number":
		f := &core.NumberField{Name: fs.Name, Required: fs.Required}
		return f
	case "select":
		f := &core.SelectField{Name: fs.Name, Required: fs.Required, MaxSelect: fs.MaxSelect, Values: fs.Values}
		if fs.MaxSelect == 0 {
			f.MaxSelect = 1
		}
		return f
	case "relation":
		f := &core.RelationField{
			Name:          fs.Name,
			Required:      fs.Required,
			MaxSelect:     fs.MaxSelect,
			CascadeDelete: fs.CascadeDel,
		}
		if fs.MaxSelect == 0 {
			f.MaxSelect = 1
		}
		// Store the target collection name for later resolution
		// We can't set CollectionId yet because the target collection may not exist
		f.CollectionId = fs.Collection // temporarily store name, will be replaced with ID in ResolveRelationFields
		return f
	case "date":
		return &core.DateField{Name: fs.Name, Required: fs.Required}
	case "autodate":
		return &core.AutodateField{Name: fs.Name, OnCreate: fs.OnCreate, OnUpdate: fs.OnUpdate}
	case "json":
		return &core.JSONField{Name: fs.Name, Required: fs.Required}
	case "file":
		f := &core.FileField{Name: fs.Name, Required: fs.Required}
		if fs.MaxSize > 0 {
			f.MaxSize = fs.MaxSize
		}
		if len(fs.MimeTypes) > 0 {
			f.MimeTypes = fs.MimeTypes
		}
		return f
	case "email":
		return &core.EmailField{Name: fs.Name, Required: fs.Required}
	case "bool":
		return &core.BoolField{Name: fs.Name, Required: fs.Required}
	default:
		return nil
	}
}

// ResolveRelationFields resolves relation field CollectionIds by looking up collections.
// This should be called after all collections are created.
func ResolveRelationFields(app core.App) error {
	for _, schema := range schemaRegistry {
		col, err := app.FindCollectionByNameOrId(schema.Name)
		if err != nil {
			return fmt.Errorf("collection %s not found: %w", schema.Name, err)
		}
		for _, fs := range schema.Fields {
			if strings.ToLower(fs.Type) == "relation" && fs.Collection != "" {
				refCol, err := app.FindCollectionByNameOrId(fs.Collection)
				if err != nil {
					return fmt.Errorf("referenced collection %s not found for field %s.%s: %w", fs.Collection, schema.Name, fs.Name, err)
				}
				field := col.Fields.GetByName(fs.Name)
				if rel, ok := field.(*core.RelationField); ok {
					rel.CollectionId = refCol.Id
				}
			}
		}
		if err := app.Save(col); err != nil {
			return fmt.Errorf("failed to save collection %s after relation resolution: %w", schema.Name, err)
		}
	}
	return nil
}

// CollectionSchemas returns all registered schemas (for testing/inspection).
func CollectionSchemas() []CollectionSchema {
	return schemaRegistry
}

// SchemaBuilder provides a fluent API for building collection schemas.
type SchemaBuilder struct {
	schema CollectionSchema
}

func NewSchemaBuilder(name string) *SchemaBuilder {
	return &SchemaBuilder{schema: CollectionSchema{Name: name}}
}

func (b *SchemaBuilder) Type(t string) *SchemaBuilder {
	b.schema.Type = t
	return b
}

func (b *SchemaBuilder) ListRule(rule string) *SchemaBuilder {
	b.schema.ListRule = types.Pointer(rule)
	return b
}

func (b *SchemaBuilder) ViewRule(rule string) *SchemaBuilder {
	b.schema.ViewRule = types.Pointer(rule)
	return b
}

func (b *SchemaBuilder) CreateRule(rule string) *SchemaBuilder {
	b.schema.CreateRule = types.Pointer(rule)
	return b
}

func (b *SchemaBuilder) UpdateRule(rule string) *SchemaBuilder {
	b.schema.UpdateRule = types.Pointer(rule)
	return b
}

func (b *SchemaBuilder) DeleteRule(rule string) *SchemaBuilder {
	b.schema.DeleteRule = types.Pointer(rule)
	return b
}

func (b *SchemaBuilder) HiddenFromAPI() *SchemaBuilder {
	b.schema.ListRule = nil
	b.schema.ViewRule = nil
	b.schema.CreateRule = nil
	b.schema.UpdateRule = nil
	b.schema.DeleteRule = nil
	return b
}

func (b *SchemaBuilder) Field(name, ftype string, opts ...FieldOption) *SchemaBuilder {
	fs := FieldSchema{Name: name, Type: ftype}
	for _, opt := range opts {
		opt(&fs)
	}
	b.schema.Fields = append(b.schema.Fields, fs)
	return b
}

func (b *SchemaBuilder) Index(name string, unique bool, fields, exp string) *SchemaBuilder {
	b.schema.Indexes = append(b.schema.Indexes, IndexSchema{
		Name:   name,
		Unique: unique,
		Fields: fields,
		Exp:    exp,
	})
	return b
}

func (b *SchemaBuilder) Build() CollectionSchema {
	return b.schema
}

// FieldOption configures a field schema.
type FieldOption func(*FieldSchema)

func Required() FieldOption {
	return func(fs *FieldSchema) { fs.Required = true }
}

func Max(max int) FieldOption {
	return func(fs *FieldSchema) { fs.Max = max }
}

func MaxSelect(max int) FieldOption {
	return func(fs *FieldSchema) { fs.MaxSelect = max }
}

func Values(vals ...string) FieldOption {
	return func(fs *FieldSchema) { fs.Values = vals }
}

func Relation(collection string, cascadeDelete bool) FieldOption {
	return func(fs *FieldSchema) {
		fs.Collection = collection
		fs.CascadeDel = cascadeDelete
	}
}

func OnCreate(onCreate bool) FieldOption {
	return func(fs *FieldSchema) { fs.OnCreate = onCreate }
}

func OnUpdate(onUpdate bool) FieldOption {
	return func(fs *FieldSchema) { fs.OnUpdate = onUpdate }
}

func MaxSize(size int64) FieldOption {
	return func(fs *FieldSchema) { fs.MaxSize = size }
}

func MimeTypes(types ...string) FieldOption {
	return func(fs *FieldSchema) { fs.MimeTypes = types }
}

// AutodateFields adds created and updated autodate fields.
func (b *SchemaBuilder) AutodateFields() *SchemaBuilder {
	b.Field("created", "autodate", OnCreate(true))
	b.Field("updated", "autodate", OnCreate(true), OnUpdate(true))
	return b
}