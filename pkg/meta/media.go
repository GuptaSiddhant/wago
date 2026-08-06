package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/piusalfred/whatsapp/message"
)

// Media kinds (must match the Cloud API message type strings).
const (
	KindImage    = "image"
	KindVideo    = "video"
	KindAudio    = "audio"
	KindDocument = "document"
	KindSticker  = "sticker"
)

type uploadMediaResponse struct {
	ID string `json:"id"`
}

type mediaRetrieveResponse struct {
	URL       string `json:"url"`
	MimeType  string `json:"mime_type"`
	FileSize  int    `json:"file_size"`
	Messaging string `json:"messaging_product"`
}

// UploadMedia uploads a local file to the phone number's media store and
// returns the Meta media id. Use the returned id with SendMediaByID.
func (c *Client) UploadMedia(ctx context.Context, accessToken, phoneNumberID, filename, mimeType string, data []byte) (string, error) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	_ = w.WriteField("messaging_product", "whatsapp")
	if mimeType != "" {
		_ = w.WriteField("type", mimeType)
	}
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("meta: create upload form file: %w", err)
	}
	if _, err := fw.Write(data); err != nil {
		return "", fmt.Errorf("meta: write upload payload: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("meta: close upload form: %w", err)
	}

	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s/media", graphVersion, phoneNumberID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("meta: media upload: %w", err)
	}
	defer resp.Body.Close()

	raw, err := readResponse(resp)
	if err != nil {
		return "", err
	}

	var parsed uploadMediaResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("meta: parse media upload response: %w", err)
	}
	if parsed.ID == "" {
		return "", fmt.Errorf("meta: no media id in upload response")
	}
	return parsed.ID, nil
}

// SendMediaByID sends a previously uploaded media message by its Meta media id
// and returns the wamid. kind is one of KindImage, KindVideo, KindAudio,
// KindDocument or KindSticker.
func (c *Client) SendMediaByID(ctx context.Context, accessToken, phoneNumberID, to, kind, mediaID, caption, filename string) (string, error) {
	ctx = withCredentials(ctx, accessToken, phoneNumberID)

	var (
		resp *message.Response
		err  error
	)
	switch kind {
	case KindImage:
		resp, err = c.base.SendImage(ctx, message.NewRequest(to, &message.Image{ID: mediaID, Caption: caption}))
	case KindVideo:
		resp, err = c.base.SendVideo(ctx, message.NewRequest(to, &message.Video{ID: mediaID, Caption: caption}))
	case KindAudio:
		resp, err = c.base.SendAudio(ctx, message.NewRequest(to, &message.Audio{ID: mediaID}))
	case KindDocument:
		resp, err = c.base.SendDocument(ctx, message.NewRequest(to, &message.Document{ID: mediaID, Caption: caption, Filename: filename}))
	case KindSticker:
		resp, err = c.base.SendSticker(ctx, message.NewRequest(to, &message.Sticker{ID: mediaID}))
	default:
		return "", fmt.Errorf("meta: unsupported media kind %q", kind)
	}
	if err != nil {
		return "", err
	}
	if len(resp.Messages) == 0 {
		return "", fmt.Errorf("meta: no message id in response")
	}
	return resp.Messages[0].ID, nil
}

// GetMediaRetrieve returns the retrievable URL and MIME type for a Meta media
// id. The URL is signed and can be fetched with the account access token in the
// Authorization header.
func (c *Client) GetMediaRetrieve(ctx context.Context, accessToken, mediaID string) (string, string, error) {
	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s", graphVersion, mediaID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("meta: media retrieve: %w", err)
	}
	defer resp.Body.Close()

	raw, err := readResponse(resp)
	if err != nil {
		return "", "", err
	}

	var parsed mediaRetrieveResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", "", fmt.Errorf("meta: parse media retrieve response: %w", err)
	}
	if parsed.URL == "" {
		return "", "", fmt.Errorf("meta: no download url in retrieve response")
	}
	return parsed.URL, parsed.MimeType, nil
}

// DownloadMedia fetches the bytes of a Meta media file using the account token.
func (c *Client) DownloadMedia(ctx context.Context, downloadURL, accessToken string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("meta: media download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("meta: media download returned %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("meta: read media download: %w", err)
	}
	return data, nil
}
