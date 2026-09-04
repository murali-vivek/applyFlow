# ApplyFlow V1

Personal job-outreach automation: upload a resume, XLSX of target companies, and an email template. ApplyFlow schedules one Gmail email every 3 minutes via Google OAuth.

## Architecture

```
Frontend (React) → Go API → PostgreSQL
                    ↓
              Go Scheduler → SQS → Email Worker → Gmail API
                    ↓
                   S3 (resume, XLSX)
```

## Prerequisites

- Go 1.24+
- Node.js 18+
- Docker & Docker Compose
- AWS CLI (only for the LocalStack setup script — **no real AWS account needed**)
- Google Cloud OAuth credentials with Gmail send scope

**You do not need an AWS account for local development.** S3 and SQS run via LocalStack inside Docker. The setup script uses fake `test` credentials against `localhost:4566`.

## Quick Start

### 1. Start infrastructure

```bash
docker compose up -d
bash scripts/setup-local-aws.sh
```

### 2. Configure environment

```bash
cp .env.example .env
```

Edit `.env` with your Google OAuth credentials:

- Create a project in [Google Cloud Console](https://console.cloud.google.com/)
- Enable Gmail API
- Create OAuth 2.0 credentials (Web application)
- Authorized redirect URI: `http://localhost:8080/auth/google/callback`
- Set `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, and a random `JWT_SECRET`

### 3. Run the backend

```bash
# Terminal 1 — API + scheduler
make dev-api

# Terminal 2 — email worker
make dev-worker
```

### 4. Run the frontend

```bash
cd frontend && npm install && npm run dev
```

Open http://localhost:5173

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/auth/google` | Start Google OAuth |
| GET | `/auth/google/callback` | OAuth callback |
| GET | `/templates` | List templates |
| POST | `/templates` | Create template |
| PUT | `/templates/{id}` | Update template |
| DELETE | `/templates/{id}` | Delete template |
| GET | `/resume` | Get resume metadata |
| POST | `/resume` | Upload PDF resume |
| POST | `/files/xlsx` | Upload & validate XLSX |
| GET | `/campaigns` | List campaigns |
| POST | `/campaigns` | Create campaign |
| GET | `/campaigns/{id}` | Get campaign (polling) |
| POST | `/campaigns/{id}/cancel` | Cancel campaign |

## XLSX Format

Required columns (first sheet):

| Company Name | Role | Company Mail |
|--------------|------|--------------|

- Max 400 rows
- No duplicate emails within a file

## Development

```bash
make test    # Run Go tests
make vet     # Run go vet
make build   # Build api and worker binaries
```

## Project Structure

```
applyflow/
├── cmd/api/          # HTTP API + embedded scheduler
├── cmd/worker/       # SQS email worker
├── internal/         # Business logic, handlers, repos
├── frontend/         # React SPA
├── migrations/       # SQL schema reference
└── scripts/          # Local AWS setup
```

## V1 Constraints

- One resume per user (PDF)
- Max 5 email templates
- One email every 3 minutes per campaign
- Campaign cancellation is state-based (no SQS purge)
- Frontend polls campaign status every ~4 seconds
- At-least-once email delivery (not exactly-once)

See `APPLYFLOW_DESIGN.md` for the full engineering specification.
