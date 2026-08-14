# WaGo

Self-hosted **WhatsApp Business API** server for small businesses doing marketing and support. Multi-user team support built in, with a React SPA served from `/`.

## Stack

- Backend: Go + [PocketBase](https://pocketbase.io) (custom API routes, webhook handlers, org-scoped data)
- Frontend: React + Vite + TanStack Router + TanStack Query (Tailwind + React Aria Components), installable as a PWA

## PWA & phone notifications

WaGo is an installable **PWA** (manifest + service worker). Install it to the home screen on your phone, then allow notifications — you'll get real **Web Push** notifications for new chats even when the PWA is closed.

Requirements:

- **HTTPS** in production. Web Push and service workers require a secure context (use a reverse proxy with TLS, or a tunnel like Cloudflare Tunnel/ngrok for local testing).
- VAPID keys are **auto-generated on first boot** and stored in PocketBase; no extra setup needed. Set `VAPID_SUBJECT` (defaults to `ADMIN_EMAIL`) so push services can contact you.

Behavior:
- **Active agent** (pinged presence within the last 5 minutes): notification shows in-app (bell in the sidebar) and, if the tab is foregrounded, as a browser banner.
- **Inactive agent**: the OS shows the Web Push banner regardless of whether the app is open; email and WhatsApp alerts also fire (see [Notifications](#notifications)).

## Quickstart

1. Copy the sample env and fill it in:

   ```sh
   cp .env.sample .env
   ```

2. Required variables:

   | Variable | Purpose |
   | --- | --- |
   | `ADMIN_PASSWORD` | Seeds the PocketBase superuser on first boot |

3. (Optional) Set up SMTP (notification emails) and WhatsApp notification template — see [Environment variables](#environment-variables).
4. Run the server (dev):

   ```sh
   go run .
   ```

   The SPA and the Wago API are served from `/`.

> All variables except `ADMIN_PASSWORD` are optional. Values set in the
> environment seed the config on first boot and can then be edited at runtime by
> a superadmin from **Settings → Instance Config** (stored in the SQLite DB,
> applied immediately — no restart needed).

## Environment variables

All variables are read from the process environment and a `.env` file (see `.env.sample`).

### Required

| Variable | Default | Notes |
| --- | --- | --- |
| `ADMIN_EMAIL` | `admin@wago.local` | Superuser email seeded at boot |
| `ADMIN_PASSWORD` | — | Required; seeds the superuser at boot |

> Every other setting is optional and runtime-editable by a superadmin
> (Settings → Instance Config).

### Optional

| Variable | Default | Notes |
| --- | --- | --- |
| `WA_WEBHOOK_VERIFY_TOKEN` | *(empty)* | WhatsApp Cloud API webhook handshake. Optional; also editable at runtime. |
| `META_APP_SECRET` | *(empty)* | Validates inbound webhook signatures when set |
| `SMTP_HOST` | *(empty → email off)* | SMTP relay host; enabling this turns on notification emails |
| `SMTP_PORT` | `587` | SMTP submit port |
| `SMTP_USERNAME` | *(empty)* | SMTP user |
| `SMTP_PASSWORD` | *(empty)* | SMTP password |
| `SMTP_TLS` | `false` | Implicit TLS (e.g. port 465) |
| `SMTP_FROM_ADDRESS` | `ADMIN_EMAIL` | From-address for outgoing email |
| `SMTP_FROM_NAME` | `WaGo` | Display name for outgoing email |
| `VAPID_SUBJECT` | `ADMIN_EMAIL` | Contact for Web Push VAPID tokens (email or URL) |
| `WA_NOTIFICATION_TEMPLATE` | *(empty → off)* | Approved Meta template for WhatsApp alerts to away agents |

## Notifications

When a customer message arrives, the assigned agent is notified:

- **Active agents** (have pinged presence in the last 5 minutes) get an **in-app notification** and a **desktop push** (auto-prompted after login, visible in the bell in the top-left sidebar).
- **Inactive agents** are alerted by **email** (requires SMTP config) and, best-effort, by **WhatsApp** (requires `WA_NOTIFICATION_TEMPLATE` and a `phone` number on the user's profile).

Notification emails are sent through PocketBase's built-in SMTP mailer, enabled automatically at startup when `SMTP_HOST` is set.

## Webhooks

- `POST /api/wa/webhook` — inbound WhatsApp messages (Meta Cloud API)
- `GET  /api/wa/webhook` — Meta subscription verification

## License

[MIT](LICENSE.md)