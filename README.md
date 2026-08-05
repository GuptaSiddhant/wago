# WaGo

Self-hosted **WhatsApp Business API** server for small businesses doing marketing and support. Multi-user team support built in, with a React SPA served from `/`.

## Stack

- Backend: Go + [PocketBase](https://pocketbase.io) (custom API routes, webhook handlers, org-scoped data)
- Frontend: React + Vite + TanStack Router + TanStack Query (Tailwind + React Aria Components)

## Quickstart

1. Copy the sample env and fill it in:

   ```sh
   cp .env.sample .env
   ```

2. Required variables:

   | Variable | Purpose |
   | --- | --- |
   | `ADMIN_PASSWORD` | Seeds the PocketBase superuser on first boot |
   | `WA_WEBHOOK_VERIFY_TOKEN` | WhatsApp Cloud API webhook verification |

3. Set up SMTP (notification emails) and WhatsApp notification template — see [Environment variables](#environment-variables).
4. Run the server (dev):

   ```sh
   go run .
   ```

   The SPA and the Wago API are served from `/`.

## Environment variables

All variables are read from the process environment and a `.env` file (see `.env.sample`).

### Required

| Variable | Default | Notes |
| --- | --- | --- |
| `ADMIN_EMAIL` | `admin@wago.local` | Superuser email seeded at boot |
| `ADMIN_PASSWORD` | — | Required; seeds the superuser at boot |
| `WA_WEBHOOK_VERIFY_TOKEN` | — | Required; WhatsApp webhook handshake |

### Optional

| Variable | Default | Notes |
| --- | --- | --- |
| `META_APP_SECRET` | *(empty)* | Validates inbound webhook signatures when set |
| `SMTP_HOST` | *(empty → email off)* | SMTP relay host; enabling this turns on notification emails |
| `SMTP_PORT` | `587` | SMTP submit port |
| `SMTP_USERNAME` | *(empty)* | SMTP user |
| `SMTP_PASSWORD` | *(empty)* | SMTP password |
| `SMTP_TLS` | `false` | Implicit TLS (e.g. port 465) |
| `SMTP_FROM_ADDRESS` | `ADMIN_EMAIL` | From-address for outgoing email |
| `SMTP_FROM_NAME` | `WaGo` | Display name for outgoing email |
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