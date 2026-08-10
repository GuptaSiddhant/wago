package api

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/guptasiddhant/wago/pkg/meta"
	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

var metaClient = meta.NewClient()

// HandleAccountMeta fetches the per-phone-number health/status from Meta. These
// require no extra permissions (same token used for messaging). The response
// is always 200 so per-number fetches degrade gracefully; ok=false carries a
// human-readable reason (bad token, missing number, etc.).
func HandleAccountMeta(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		acc, err := store.FindOrgRecord(app, access.OrgID, "whatsapp_accounts", e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("number not found", nil)
		}

		token := acc.GetString("access_token")
		phoneID := acc.GetString("phone_number_id")
		if token == "" || phoneID == "" {
			return e.JSON(http.StatusOK, map[string]any{
				"ok":    false,
				"error": "number is missing an access token or phone_number_id",
			})
		}

		info, err := metaClient.GetPhoneNumberInfo(e.Request.Context(), token, phoneID)
		if err != nil {
			return e.JSON(http.StatusOK, map[string]any{
				"ok":    false,
				"error": err.Error(),
			})
		}

		return e.JSON(http.StatusOK, map[string]any{"ok": true, "info": info})
	}
}

type analyticsTotals struct {
	Conversations int64   `json:"conversations"`
	Cost          float64 `json:"cost"`
}

type analyticsAccount struct {
	ID            string  `json:"id"`
	DisplayName   string  `json:"display_name"`
	PhoneNumberID string  `json:"phone_number_id"`
	Conversations int64   `json:"conversations"`
	Cost          float64 `json:"cost"`
}

type analyticsCategory struct {
	Category      string  `json:"category"`
	Conversations int64   `json:"conversations"`
	Cost          float64 `json:"cost"`
}

// HandleAnalytics returns org-level WhatsApp usage and cost for a time range,
// aggregated from Meta's conversation_analytics across every connected WABA.
// Requires each number to have a waba_id and a token with the
// whatsapp_business_management permission; failing numbers are reported in
// "errors" so the UI can explain missing setup.
func HandleAnalytics(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		rangeKey := e.Request.URL.Query().Get("range")
		var days time.Duration
		switch rangeKey {
		case "7d":
			days = 7 * 24 * time.Hour
		case "90d":
			days = 90 * 24 * time.Hour
		default:
			rangeKey = "30d"
			days = 30 * 24 * time.Hour
		}

		now := time.Now().UTC()
		start := now.Add(-days)
		granularity := "DAILY"
		if days > 31*24*time.Hour {
			granularity = "MONTHLY"
		}

		records, err := app.FindRecordsByFilter("whatsapp_accounts", "org = {:org}", "-created", 200, 0,
			store.DbxParams(map[string]any{"org": access.OrgID}))
		if err != nil {
			return e.InternalServerError("Failed to list WhatsApp accounts", err)
		}

		totals := analyticsTotals{}
		accounts := make([]analyticsAccount, 0)
		categories := map[string]*analyticsCategory{}
		var errorsOut []string

		ctx := e.Request.Context()
		for _, acc := range records {
			wabaID := acc.GetString("waba_id")
			token := acc.GetString("access_token")
			if wabaID == "" {
				errorsOut = append(errorsOut, "Number "+peerLabel(acc)+" has no WABA ID — add it to see analytics")
				continue
			}
			if token == "" {
				errorsOut = append(errorsOut, "Number "+peerLabel(acc)+" has no access token")
				continue
			}

			points, aerr := metaClient.FetchConversationAnalytics(ctx, token, wabaID, start.Unix(), now.Unix(), granularity)
			if aerr != nil {
				errorsOut = append(errorsOut, peerLabel(acc)+": "+aerr.Error())
				continue
			}

			accEntry := analyticsAccount{ID: acc.Id, DisplayName: acc.GetString("display_name"), PhoneNumberID: acc.GetString("phone_number_id")}
			for _, p := range points {
				totals.Conversations += p.Conversations
				totals.Cost += p.Cost
				accEntry.Conversations += p.Conversations
				accEntry.Cost += p.Cost

				cat := strings.Title(strings.ToLower(p.ConversationCategory))
				cat = strings.TrimSpace(cat)
				if cat == "" {
					cat = "Unknown"
				}
				if categories[cat] == nil {
					categories[cat] = &analyticsCategory{Category: cat}
				}
				categories[cat].Conversations += p.Conversations
				categories[cat].Cost += p.Cost
			}
			accounts = append(accounts, accEntry)
		}

		cats := make([]analyticsCategory, 0, len(categories))
		for _, c := range categories {
			cats = append(cats, *c)
		}
		slices.SortFunc(cats, func(a, b analyticsCategory) int {
			if b.Cost > a.Cost {
				return 1
			}
			if b.Cost < a.Cost {
				return -1
			}
			return 0
		})

		return e.JSON(http.StatusOK, map[string]any{
			"range":      rangeKey,
			"start":      start.Unix(),
			"end":        now.Unix(),
			"totals":     totals,
			"accounts":   accounts,
			"categories": cats,
			"errors":     errorsOut,
		})
	}
}

func peerLabel(acc *core.Record) string {
	if n := acc.GetString("display_name"); n != "" {
		return n
	}
	return acc.GetString("phone_number_id")
}

// HandleAccountWebhookConnect subscribes the Meta app to the webhook events of
// a WhatsApp Business Account. The account's token must have
// whatsapp_business_messaging scope. Returns the callback URL and whether it
// was configured, plus a human-readable error when it failed.
func HandleAccountWebhookConnect(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		if !access.CanManageData() {
			return e.ForbiddenError("only the owner or a superadmin can manage numbers", nil)
		}

		acc, err := store.FindOrgRecord(app, access.OrgID, "whatsapp_accounts", e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("number not found", nil)
		}

		callback, err := webhookCallbackURL()
		if err != nil {
			return e.BadRequestError(err.Error(), nil)
		}
		if webhookCfg.VerifyToken == "" {
			return e.BadRequestError("this instance has no webhook verify token configured", nil)
		}

		token := acc.GetString("access_token")
		wabaID := acc.GetString("waba_id")
		if token == "" {
			return e.BadRequestError("number is missing an access token", nil)
		}
		if wabaID == "" {
			return e.BadRequestError("number is missing a WABA ID — add it to connect the webhook", nil)
		}

		if err := metaClient.SubscribeWebhook(e.Request.Context(), token, wabaID, callback, webhookCfg.VerifyToken); err != nil {
			return e.JSON(http.StatusOK, map[string]any{
				"ok":           false,
				"error":        err.Error(),
				"callback_url": callback,
			})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"ok":           true,
			"callback_url": callback,
			"message":      "Webhook connected. Meta will deliver messages for this number to Wago.",
		})
	}
}

// HandleAccountWebhookStatus reports whether the account's WABA is subscribed
// to this app's webhooks, along with the callback URL it should be using.
func HandleAccountWebhookStatus(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		acc, err := store.FindOrgRecord(app, access.OrgID, "whatsapp_accounts", e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("number not found", nil)
		}

		callback, _ := webhookCallbackURL()
		return e.JSON(http.StatusOK, map[string]any{
			"ok":           acc.GetString("status") == "connected",
			"callback_url": callback,
			"verify_token": webhookCfg.VerifyToken != "",
		})
	}
}

// webhookCallbackURL builds the public URL Meta delivers events to. It requires
// the instance to be publicly reachable and have PUBLIC_BASE_URL configured.
func webhookCallbackURL() (string, error) {
	base := strings.TrimRight(webhookCfg.PublicBaseURL, "/")
	if base == "" {
		return "", errors.New("this instance has no PUBLIC_BASE_URL configured — set it to connect webhooks")
	}
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		return "", errors.New("PUBLIC_BASE_URL must include http(s)://")
	}
	return base + "/api/wa/webhook", nil
}
