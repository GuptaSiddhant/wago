package store

import "github.com/pocketbase/pocketbase/core"

func EnsureCollections(app core.App) error {

	if err := EnsureContactsCollection(app); err != nil {
		return err
	}
	if err := EnsureMessagesCollection(app); err != nil {
		return err
	}

	return nil
}
