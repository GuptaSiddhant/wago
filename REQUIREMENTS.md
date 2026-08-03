# WaGo — Functional Requirements (MVP)

Self-hosted WhatsApp Business API server for small businesses doing marketing and support chats. Multi-user team support built in. Users primarily interact with the Wago UI.

## Stack

- Backend: Go / PocketBase
- Frontend: React + Vite + TanStack Router, PocketBase JS SDK
- The SPA is always served from `/` (no headless mode).

## Auth & multi-user teams

- In-app login via PocketBase `users` collection with a custom Wago login page; PB admin UI is never exposed to regular users.
- PocketBase superusers are super admins of the Wago UI as well — they can sign in and get full access to everything.
- Orgs (workspaces): `orgs` ↔ `org_members`, with roles: `owner`, `admin`, `agent`, `viewer`.
- Invite flow: email invite, role assignment, join.
- Data isolation: every collection scoped to `org_id`; record rules enforce org membership via `@request.auth`.

## WhatsApp accounts

- Multiple business numbers per org (Meta Cloud API): `phone_number_id` + access token + verify token per account.
- Per-account status (connected/disconnected), used for webhook routing.
- Numbers are org-owned and shared by the team; any agent can handle any conversation.

## Inbox (support chat)

- Unified inbox: conversation list (grouped by contact), unread counts, assignee filters, search.
- Thread view: send text + view inbound; message status (sent/delivered/read/failed) from status webhooks.
- Agent assignment (manual + round-robin), assignee filters.
- Contact sidebar: name, phone, tags, notes, conversation history.

## Contacts

- Auto-created from inbound messages, dedupe by phone.
- Tags, notes, custom attributes; searchable list.
- Conversation history per contact.

## Messaging policy (Meta compliance)

- Free-form replies only within the 24-hour customer service window.
- Outside the window, outbound messages require an approved Meta template (enforced server-side and surfaced in the UI).

## Settings

- Org settings, WhatsApp number management, team management.

## Backend collections

- `users` (built-in)
- `orgs`, `org_members`
- `whatsapp_accounts`
- `contacts` (scoped to `org_id`, tags, notes, last activity)
- `messages` (scoped to `org_id`, conversation id, status)
- `conversations` (contact, whatsapp account, assignee, unread count, last message at)

## Frontend routes

- `/login` — custom auth page.
- App shell: sidebar (Inbox, Contacts, Team, Settings) + auth guards + org context provider.
- `/inbox` — conversation list + thread view with composer, status ticks, contact sidebar.
- `/contacts` — searchable list, contact detail (info, tags, notes, history).
- `/settings/team` — members, roles, invites.
- `/settings/numbers` — WhatsApp account management.

## Out of scope for MVP

Campaigns/templates UI, segments, analytics, media messages. Deferred until the inbox loop is solid.
