package api

import (
	"io"
	"net/http"
	"strings"

	"github.com/guptasiddhant/wago/pkg/meta"
	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

// HandleMediaUpload uploads a file to a WhatsApp number's media store on Meta
// and returns the media id plus its kind. The uploaded media can be referenced
// later as a template media header or a send-time header override.
func HandleMediaUpload(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		if err := e.Request.ParseMultipartForm(100 << 20); err != nil {
			return e.BadRequestError("Invalid multipart body", err)
		}
		accountID := strings.TrimSpace(e.Request.FormValue("account_id"))
		if accountID == "" {
			return e.BadRequestError("account_id is required", nil)
		}

		account, err := store.FindOrgRecord(app, access.OrgID, "whatsapp_accounts", accountID)
		if err != nil {
			return e.BadRequestError("whatsapp account not found", nil)
		}
		token := account.GetString("access_token")
		phoneID := account.GetString("phone_number_id")
		if token == "" || phoneID == "" {
			return e.BadRequestError("this number needs an access token and phone number id to upload media", nil)
		}

		file, header, err := e.Request.FormFile("file")
		if err != nil {
			return e.BadRequestError("file is required", nil)
		}
		defer file.Close()

		kind := mediaKindForMime(header.Header.Get("Content-Type"))
		if kind == "" || kind == meta.KindAudio || kind == meta.KindSticker {
			return e.BadRequestError("unsupported media type for a template header; use an image, video or document", nil)
		}

		data, err := io.ReadAll(file)
		if err != nil {
			return e.InternalServerError("Failed to read file", err)
		}
		if len(data) == 0 {
			return e.BadRequestError("file is empty", nil)
		}

		client := meta.NewClient()
		mediaID, err := client.UploadMedia(e.Request.Context(), token, phoneID,
			header.Filename, header.Header.Get("Content-Type"), data)
		if err != nil {
			return e.BadRequestError("Failed to upload media to Meta", err)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"media_id":   mediaID,
			"media_type": strings.ToUpper(kind),
			"filename":   header.Filename,
		})
	}
}

// HandleMessageMedia streams the locally-stored media file of a message out to
// the browser. Media bytes were downloaded from Meta at ingest time and stored
// on the message's "media" file field, so no live Meta call or token is needed
// to render an attachment in the inbox. Access is org-scoped via wamid lookup.
// Passing ?download=1 forces a download instead of an inline preview.
func HandleMessageMedia(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		msg, err := store.FindMessageByWamid(app, access.OrgID, e.Request.PathValue("wamid"))
		if err != nil {
			return e.NotFoundError("message not found", nil)
		}

		filename := msg.GetString("media")
		if filename == "" {
			return e.NotFoundError("message has no media", nil)
		}

		fsys, err := app.NewFilesystem()
		if err != nil {
			return e.InternalServerError("Filesystem initialization failure", err)
		}
		defer fsys.Close()

		return fsys.Serve(e.Response, e.Request, msg.BaseFilesPath()+"/"+filename, filename)
	}
}
