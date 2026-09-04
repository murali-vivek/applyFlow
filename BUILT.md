# ApplyFlow V1 — What Has Been Built

This document describes everything implemented in the ApplyFlow project as of September 2026.

---

## Why is the frontend inside the Go project folder?

The React app lives at `applyflow/frontend/` — **sibling to** `cmd/` and `internal/`, not inside the Go backend code.

This is a **monorepo** layout, chosen for V1 simplicity:

| Reason | Detail |
|--------|--------|
| Single project root | One repo, one README, one `.env`, one `docker-compose.yml` |
| Shared local dev | Postgres + LocalStack serve both API and UI |
| One Makefile | `make dev-api`, `make dev-worker`, `make dev-frontend` from the same place |
| Faster iteration | No separate deployment wiring needed for a personal tool |

The frontend is **not compiled into Go**. It runs as a separate Vite dev server (`localhost:5173`) and talks to the Go API (`localhost:8080`) over HTTP. They are independent processes that happen to share a folder tree.

A split layout (`applyflow-api/` + `applyflow-web/`) would make sense later if teams, CI, or deployment pipelines diverge.

---

## High-Level Architecture

```
Browser (React SPA, :5173)
        │
        ▼ HTTP + JWT
Go API (:8080) ──────────────────────────────┐
        │                                   │
        ├── PostgreSQL (users, campaigns,   │
        │    outreach, templates, oauth)    │
        ├── S3 via LocalStack (resume PDF,  │
        │    XLSX files)                    │
        └── Embedded Scheduler              │
                │                           │
                ▼                           │
            SQS (LocalStack)                │
                │                           │
                ▼                           │
        Email Worker (separate process)     │
                │                           │
                ▼                           │
        Gmail API (OAuth send) ◄────────────┘
```

**Source of truth:** PostgreSQL for all scheduled outreach state.  
**SQS:** Carries only `{ outreachId }` — not email content.

---

## Repository Layout

```
applyflow/
├── cmd/
│   ├── api/                 # HTTP API + embedded scheduler
│   └── worker/              # SQS consumer → Gmail send
├── internal/
│   ├── auth/                # JWT issue/verify
│   ├── config/              # .env loading (godotenv)
│   ├── errors/              # Standard API error responses
│   ├── gmail/               # OAuth + MIME email send
│   ├── handler/             # HTTP handlers
│   ├── middleware/          # JWT auth middleware
│   ├── migrate/             # SQL migrations (auto-run on startup)
│   ├── model/               # Domain structs
│   ├── queue/               # SQS client (LocalStack-compatible)
│   ├── render/              # Template variable substitution
│   ├── repository/          # PostgreSQL data access
│   ├── service/             # Business logic
│   ├── storage/             # S3 client
│   └── xlsx/                # Spreadsheet parse + validate
├── frontend/                # React + Vite SPA
│   └── src/
│       ├── pages/           # Route screens
│       ├── components/      # Reusable UI
│       ├── hooks/           # useDialog, etc.
│       └── utils/           # uploadSummary helper
├── scripts/
│   └── setup-local-aws.sh   # Creates S3 bucket + SQS queue in LocalStack
├── docker-compose.yml       # Postgres 16 + LocalStack
├── Makefile
├── .env.example
└── README.md
```

---

## Infrastructure (Docker)

| Service | Purpose | Port |
|---------|---------|------|
| **postgres** | Primary database | 5432 |
| **localstack** | Fake S3 + SQS (no real AWS account needed) | 4566 |

Run: `docker compose up -d` then `bash scripts/setup-local-aws.sh`

---

## Database Schema

Migrations: `internal/migrate/migrations/`

| Table | Purpose |
|-------|---------|
| `users` | Google OAuth users |
| `oauth_credentials` | Google access/refresh tokens per user |
| `templates` | Email templates (max 5 per user) |
| `resumes` | One PDF resume per user (S3 path) |
| `xlsx_files` | Uploaded company-list spreadsheets (S3 path) |
| `campaigns` | Outreach campaigns with status + counters |
| `outreach` | One row per recipient email in a campaign |

**Migration 002:** Added `queued_at` on outreach for correct stale-queue recovery in the scheduler.

---

## Backend — API (`cmd/api`)

### Authentication
- Google OAuth 2.0 login with Gmail send scope
- JWT stored in browser `localStorage`, sent as `Authorization: Bearer`
- Auto-creates user + default template on first login
- Endpoints: `/auth/google`, `/auth/google/callback`, `/auth/me`, `/auth/logout`

### Templates
- CRUD for email templates (max 5 per user)
- Variables: `{{user_name}}`, `{{company_name}}`, `{{role}}`
- One default template per user

### Resume
- Upload PDF (max 10 MB), stored in S3
- One active resume per user
- Required before starting a campaign

### Company List (XLSX)
- Upload `.xlsx` spreadsheets
- Required columns: `Company Name`, `Role`, `Company Mail`
- Max 400 data rows per file
- **Smart cleaning** (does not fail on bad rows):
  - Skips rows missing company, role, or email
  - Skips invalid emails
  - Removes duplicate emails (keeps first occurrence)
  - Ignores completely empty rows
  - Returns skip metrics in `skipped` object
  - Upload succeeds if ≥ 1 valid row remains
- Preview: first 5 cleaned rows returned to UI

### Campaigns
- Create from: XLSX file ID + template ID + `startAt` timestamp
- Each recipient scheduled 3 minutes apart
- Status flow: `SCHEDULED` → `RUNNING` → `COMPLETED` / `CANCELLED`
- Cancel: marks pending outreach as cancelled (state-based, no SQS purge)
- Requires resume + Google OAuth connected

### Embedded Scheduler
- Runs inside the API process
- Polls for due outreach (`scheduled_at <= NOW()`)
- Enqueues outreach IDs to SQS
- Recovers stale `QUEUED` items after 5 minutes (uses `queued_at`, not `created_at`)

---

## Backend — Email Worker (`cmd/worker`)

Separate process — **must be running** for emails to send.

1. Polls SQS for `{ outreachId }`
2. Loads outreach, campaign, template, resume, OAuth creds
3. Renders subject/body from template
4. Downloads resume PDF from S3
5. Sends via Gmail API (`users.messages.send`)
6. Marks outreach `SENT` or `FAILED`, updates campaign counters

---

## Frontend (React + Vite)

### Pages

| Route | Page | Features |
|-------|------|----------|
| `/` | Login | Google OAuth button |
| `/auth/callback` | Auth callback | Stores JWT, redirects to dashboard |
| `/dashboard` | Dashboard | Stats cards, campaign table, **View Campaign** button |
| `/resume` | Resume | PDF upload with validation dialogs |
| `/templates` | Templates | Create/edit/delete with confirm dialogs |
| `/campaigns/new` | New Campaign | Company list upload, template picker, schedule / start immediately, existing campaigns list |
| `/campaigns/:id` | Campaign Detail | Progress bar, outreach table, cancel button, 4s polling |

### UI Components
- `Dialog` / `ConfirmDialog` — replaces browser `alert()` / `confirm()`
- `DateTimePicker` — custom calendar + time picker (disables when "Start immediately" checked)
- `XlsxPreview` — table preview of uploaded company list
- `CampaignList` — shared campaign table (dashboard + new campaign page)

### UX Details
- Navy / emerald / gold color scheme
- Upload summary dialog after spreadsheet cleaning (skip counts + final row count)
- "Upload Company List" button dulls to "Change File" after successful upload
- All API errors shown in modal dialogs

---

## API Reference

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | No | Health check |
| GET | `/auth/google` | No | Start OAuth |
| GET | `/auth/google/callback` | No | OAuth callback → JWT |
| GET | `/auth/me` | Yes | Current user |
| GET | `/auth/logout` | No | Clear session cookie |
| GET | `/templates` | Yes | List templates |
| POST | `/templates` | Yes | Create template |
| PUT | `/templates/{id}` | Yes | Update template |
| DELETE | `/templates/{id}` | Yes | Delete template |
| GET | `/resume` | Yes | Resume metadata |
| POST | `/resume` | Yes | Upload PDF |
| POST | `/files/xlsx` | Yes | Upload + validate spreadsheet |
| GET | `/campaigns` | Yes | List campaigns |
| POST | `/campaigns` | Yes | Create campaign |
| GET | `/campaigns/{id}` | Yes | Campaign detail + outreach |
| POST | `/campaigns/{id}/cancel` | Yes | Cancel campaign |

---

## Spreadsheet Validation Rules

### Hard failures (upload rejected)
- Not a `.xlsx` file
- File over 10 MB
- Missing required column headers
- No data rows (header only)
- More than 400 data rows
- No valid rows after cleaning

### Soft skips (upload succeeds, metrics reported)
| Skip reason | JSON field |
|-------------|------------|
| Missing Company Name | `skipped.missingCompanyName` |
| Missing Role | `skipped.missingRole` |
| Missing Company Mail | `skipped.missingEmail` |
| Invalid email format | `skipped.invalidEmail` |
| Duplicate email | `skipped.duplicateEmail` |
| Completely empty row | `skipped.emptyRows` |

---

## Local Development

```bash
# 1. Infrastructure
docker compose up -d
bash scripts/setup-local-aws.sh

# 2. Environment
cp .env.example .env   # add Google OAuth + JWT_SECRET

# 3. Backend (two terminals)
make dev-api           # API + scheduler on :8080
make dev-worker        # Email worker (required for sending)

# 4. Frontend
cd frontend && npm install && npm run dev   # :5173
```

**Important:** Restart `make dev-api` after any Go code change — it does not hot-reload.

---

## V1 Constraints

- One resume per user (PDF only)
- Max 5 email templates per user
- Max 400 spreadsheet rows per file
- One email every 3 minutes per campaign
- Emails sent from the Google account used to log in
- At-least-once delivery (not exactly-once)
- First spreadsheet sheet only
- Campaign cancellation is state-based (in-flight SQS messages may still process)

---

## Tests

```bash
make test   # Go unit tests (xlsx parser, template render, etc.)
make vet
make build  # bin/api, bin/worker
```

Automated tests cover: valid parse, missing columns, duplicate dedup, mixed invalid rows, all-invalid rows.

---

## Related Docs

- `README.md` — Quick start guide
- `APPLYFLOW_DESIGN.md` — Full engineering specification (parent folder)
