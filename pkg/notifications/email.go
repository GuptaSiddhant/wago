package notifications

import (
	"fmt"
	"html"
	"net/mail"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

const notifEmailTemplate = `<!DOCTYPE html>
<html lang="en">
<body style="margin:0;padding:0;background:#18181b;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Arial,sans-serif;">
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#18181b;padding:24px;">
    <tr>
      <td align="center">
        <table role="presentation" width="480" cellpadding="0" cellspacing="0" style="background:#27272a;border-radius:12px;border:1px solid #3f3f46;overflow:hidden;">
          <tr><td style="padding:20px 24px;border-bottom:1px solid #3f3f46;">
            <span style="font-size:16px;font-weight:600;color:#f4f4f5;">WaGo</span>
          </td></tr>
          <tr><td style="padding:20px 24px;">
            <p style="margin:0 0 8px;color:#a1a1aa;font-size:13px;">New message</p>
            <p style="margin:0 0 4px;color:#f4f4f5;font-size:16px;font-weight:600;">%s</p>
            <p style="margin:0;color:#d4d4d8;font-size:14px;">%s</p>
          </td></tr>
          <tr><td style="padding:0 24px 20px;">
            <p style="margin:0;color:#71717a;font-size:12px;">You have a pending chat in WaGo.&nbsp;&nbsp;Sign in and open the inbox to reply.</p>
          </td></tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`

// sendEmail sends a plain notification email using the PocketBase mailer
// (SMTP is enabled from config at startup). Returns an error on failure.
func (n *Notifier) sendEmail(app core.App, toEmail, contact, preview string) error {
	if n.cfg.SMTPHost == "" {
		return nil // no SMTP configured — skip silently
	}

	subject := fmt.Sprintf("WaGo — new message from %s", contact)
	html := fmt.Sprintf(notifEmailTemplate, html.EscapeString(contact), html.EscapeString(preview))

	msg := &mailer.Message{
		From:    mail.Address{Name: n.cfg.SMTPFromName, Address: n.cfg.SMTPFromAddress},
		To:      []mail.Address{{Address: toEmail}},
		Subject: subject,
		HTML:    html,
		Text:    fmt.Sprintf("New message from %s:\n\n%s\n\nYou have a pending chat in WaGo.", contact, preview),
	}

	return app.NewMailClient().Send(msg)
}