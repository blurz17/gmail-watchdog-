# 🐕 Gmail Watchdog

A production-grade Gmail monitoring and notification service built in Go. Watches multiple Gmail accounts for emails from configurable senders and delivers real-time notifications through Telegram.

## Features

- **Web Dashboard** — Manage accounts, providers, and rules from a beautiful dark-themed UI
- **Multi-Account** — Monitor multiple Gmail accounts simultaneously
- **Provider Rules** — Configure sender matching per provider (exact email or domain)
- **Real-time Notifications** — Rich Telegram messages with email details + direct Gmail link
- **Bot Commands** — `/status`, `/count`, `/latest`, `/providers`, `/accounts`
- **Zero Duplicates** — Database-enforced deduplication survives restarts
- **Outbox Pattern** — No notification lost, even if Telegram is temporarily down
- **Auto Token Refresh** — OAuth tokens refresh automatically
- **Retry with Backoff** — Exponential backoff + jitter on failures

## Architecture

```
Gmail API ──→ Sync Worker ──→ PostgreSQL ──→ Delivery Worker ──→ Telegram
                                  ↑                                  ↑
                            Outbox Pattern                    Bot Commands
```

Built with **Hexagonal Architecture** — domain logic is fully decoupled from infrastructure.

## Tech Stack

| Component | Technology |
|---|---|
| Language | Go 1.23+ |
| Database | PostgreSQL (Neon free tier) |
| Gmail | Official Gmail API + OAuth 2.0 (readonly) |
| Notifications | Telegram Bot API |
| Hosting | Heroku (Eco dyno) |
| Logging | `log/slog` (structured JSON) |

## Quick Start

### Prerequisites

- Go 1.23+
- PostgreSQL (local or [Neon](https://neon.tech))
- Telegram bot token (from [@BotFather](https://t.me/BotFather))
- Google Cloud OAuth credentials ([setup guide](#google-oauth-setup))

### 1. Clone & Configure

```bash
git clone https://github.com/blurz17/gmail-watchdog-.git
cd gmail-watchdog-
cp .env.example .env
# Edit .env with your credentials
```

### 2. Run Locally

```bash
# Start local PostgreSQL
docker compose up -d postgres

# Run the service
go run ./cmd/server
```

### 3. Add Gmail Accounts & Providers

1. Open your browser to `http://localhost:8080/dashboard`
2. Log in using your `HTTP_API_KEY`
3. Click **+ Connect Gmail Account** to link your emails via OAuth
4. Add providers and sender rules directly from the UI!

## Google OAuth Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com) → create project
2. Enable **Gmail API**
3. Configure OAuth consent screen → External → add your email as test user
4. Scope: `gmail.readonly` (read-only, cannot modify your email)
5. Create OAuth Client ID → Desktop app → copy Client ID & Secret

> **⚠️ This uses only `gmail.readonly` scope — the service cannot send, delete, or modify any emails.**

## Bot Commands

| Command | Description |
|---|---|
| `/help` | Show available commands |
| `/status` | Monitoring status, account health, pending notifications |
| `/providers` | List all configured providers |
| `/accounts` | List monitored Gmail accounts with sync status |
| `/count <provider>` | Count unread matching emails per account |
| `/latest <provider>` | Show the most recent matching emails |

## Deployment (Heroku + Neon)

```bash
heroku create your-app-name --stack container
heroku config:set DATABASE_URL="<neon-connection-string>" \
  TELEGRAM_BOT_TOKEN="<token>" TELEGRAM_CHAT_ID="<id>" \
  GMAIL_CLIENT_ID="<id>" GMAIL_CLIENT_SECRET="<secret>" \
  APP_ENV=production APP_TIMEZONE=Asia/Riyadh
git push heroku main
```

**Monthly cost: ~$5** (Heroku Eco $5 + Neon free $0)

## Project Structure

```
├── cmd/
│   ├── server/          # Main service entry point
│   └── cli/             # CLI tool (auth, add-provider)
├── internal/
│   ├── domain/          # Entities, interfaces, business logic
│   ├── application/     # Use cases (sync, deliver, count, status)
│   ├── infrastructure/  # Gmail, Telegram, PostgreSQL adapters
│   └── interfaces/      # HTTP API (health endpoints)
├── migrations/          # SQL schema migrations
├── Dockerfile           # Multi-stage Docker build
├── docker-compose.yml   # Local development
└── Procfile             # Heroku process definition
```

## Health Endpoints

| Endpoint | Auth | Description |
|---|---|---|
| `GET /health` | None | Returns 200 if service is running |
| `GET /ready` | None | Returns 200 if database is reachable |
| `GET /status` | Bearer token | Full system status JSON |

## License

[MIT](LICENSE)
