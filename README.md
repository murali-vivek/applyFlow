# ApplyFlow

Cold email by hand was tedious, so a campaign (roles, resume, schedule) sends personalized mail in the background through the user's own Gmail.

Sign in with Google, upload a spreadsheet, attach a resume, watch sent / failed / in-progress on a dashboard. Cancel a campaign and pending mail stops. The spreadsheet must have Company Name, Role Name, and Company Mail ID. Headers are checked, emails are validated, the file is capped at 400 rows, and duplicate recipients are skipped.

## Stack

- **React** — campaigns, uploads, and progress. The production build is static files in a private S3 bucket, served over HTTPS by CloudFront.
- **Go API** — users, campaigns, templates, and outreach state. It runs as a container on ECS Fargate.
- **Go worker** — a second Fargate service. It pulls from SQS and sends the mail, so a slow Gmail call never sits on the HTTP request. Failed sends can be retried from the queue.
- **PostgreSQL on RDS** — private in the VPC. Progress survives a refresh. The API reaches it through security groups, not a public database port.
- **S3** — resumes and spreadsheets, separate from the database.
- **SQS** — the handoff between "this email is due" and "send it."
- **In-process scheduler** — campaign start times. The API periodically claims due rows and enqueues them. The queue is not the clock.
- **Google OAuth and the Gmail API** — mail goes out from the user's Gmail. Sign-in returns a token to the React app. Access and refresh tokens stay on the server.

The API sits behind an Application Load Balancer. Google rejects a plain HTTP redirect for Gmail access, and the domain's DNS is not in Route 53, so a second CloudFront distribution terminates HTTPS in front of that load balancer. The browser app and the API are separate CloudFront URLs. Images are built for Linux on Fargate. GitHub Actions assumes an AWS role with OIDC (no saved access keys), pushes the API and worker images to ECR, and rolls the ECS services. The React app is published to S3 on its own.

Docker, ECS Fargate, an Application Load Balancer, CloudFront, private RDS, security groups, and GitHub Actions.

## Local Development

Run everything locally with Docker Compose and LocalStack:

```bash
docker compose up -d
bash scripts/setup-local-aws.sh
cp .env.example .env
# Edit .env with Google OAuth credentials
make dev-api  # Terminal 1
make dev-worker  # Terminal 2
cd frontend && npm install && npm run dev  # Terminal 3
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

## Constraints

- One resume per user (PDF)
- Max 5 email templates
- One email every 3 minutes per campaign
- Campaign cancellation is state-based (no SQS purge)
- Frontend polls campaign status every ~4 seconds
- At-least-once email delivery (not exactly-once)

See `DEPLOYMENT.md` for the production deployment record.
