# ApplyFlow — V1 Engineering Design & Cursor Build Specification

## 1. Purpose

ApplyFlow is a personal job-outreach automation product for early-career technology professionals.

The user uploads:
- one standard resume PDF
- one XLSX file containing target companies and recipient emails
- one email template

ApplyFlow then creates a campaign, schedules one outreach email every 3 minutes, and sends the email through the user's Gmail account using Google OAuth and the Gmail API.

This document is the source of truth for building the V1.

---

# 2. V1 Product Scope

## Core user flow

1. User signs in with Google.
2. User account is created in ApplyFlow.
3. A default email template is automatically created.
4. User uploads one standard resume PDF.
5. User creates/selects an email template.
6. User uploads an XLSX file.
7. Backend validates the XLSX.
8. User previews the campaign.
9. User chooses:
   - Start now
   - Schedule for later
10. Backend creates:
   - Campaign record
   - Outreach records
11. Scheduler identifies due Outreach records.
12. Scheduler sends work to SQS.
13. Worker consumes SQS messages.
14. Worker sends email through Gmail API.
15. Worker updates Outreach and Campaign state.
16. Frontend polls Campaign API to display progress.
17. User may cancel an active campaign.

---

# 3. Explicit V1 Constraints

## Resume

- Exactly one active standard resume per user.
- Resume must be a PDF.
- No custom resume per company/job in V1.
- Resume binary is stored in S3.
- PostgreSQL stores only metadata and S3 path/key.

## Templates

- A default template always exists.
- Maximum 5 templates per user.
- Only one template may be the default.
- Supported variables:
  - `{{company_name}}`
  - `{{role}}`
  - `{{user_name}}`
- At minimum, require `{{company_name}}` in a user-created template.
- `{{role}}` and `{{user_name}}` are supported but may be optional.
- Subject and body are both required.

## XLSX

Maximum 400 rows.

Required columns:

- `Company Name`
- `Role`
- `Company Mail`

Validation:
- correct file type
- required columns exist
- row count <= 400
- required values are non-empty
- email syntax is valid
- duplicate recipient email addresses within the same file/campaign are rejected

Do not perform SMTP mailbox existence probing.

## Sending

- One email every 3 minutes.
- "Send now" means begin the campaign now; it does NOT mean send all messages immediately.
- For N rows:
  - row 1 -> startAt
  - row 2 -> startAt + 3 minutes
  - row 3 -> startAt + 6 minutes
  - etc.
- 400 rows therefore span roughly 20 hours.
- Gmail/API/provider limits must still be respected. Do not assume 400 messages is universally safe.

## Cancellation

Campaign cancellation is required.

Do NOT attempt to solve cancellation by deleting messages from SQS.

Instead:
- mark Campaign as `CANCELLED`
- scheduler must stop enqueueing new Outreach
- worker must check Campaign status before sending
- queued but not-yet-sent messages should safely become no-ops

## Dashboard

V1 uses frontend polling.

Example:

`GET /campaigns/{id}` every few seconds.

Do not implement WebSockets unless there is extra time after V1 works.

---

# 4. Out of Scope for V1

Do NOT build:

- AI-generated email copy
- recruiter discovery
- job scraping
- custom resume generation
- automatic follow-ups
- browser extensions
- sophisticated analytics
- recommendation engine
- CRM features
- multi-channel outreach
- WebSockets
- complex admin panel

Keep V1 focused.

---

# 5. High-Level Architecture

```text
                         ┌───────────────┐
                         │ Google OAuth  │
                         └───────┬───────┘
                                 │
                                 ▼
Frontend ───────────────► Go API
                            │
               ┌────────────┼─────────────┐
               ▼            ▼             ▼
           PostgreSQL      S3        Gmail OAuth
               │
        ┌──────┴──────┐
        ▼             ▼
     Campaign      Outreach
                      │
                 scheduledAt
                      │
                      ▼
                  Scheduler
                      │
                  due Outreach
                      │
                      ▼
                     SQS
                      │
                      ▼
                 Email Worker
                      │
                      ├── PostgreSQL
                      ├── S3 (resume)
                      └── Gmail API
```

Simplified:

```text
Frontend
   ↓
Go API
   ↓
PostgreSQL
   ↓
Go Scheduler
   ↓
SQS
   ↓
Email Worker
   ↓
Gmail API
```

S3 is used for uploaded PDF/XLSX files.

---

# 6. Key Architectural Decisions

## 6.1 PostgreSQL is the source of truth for scheduled work

Do NOT put all future emails directly into SQS.

SQS is a queue, not a long-term scheduler.

Future Outreach records live in PostgreSQL with `scheduled_at`.

The scheduler periodically finds due records.

Example query concept:

```sql
SELECT id
FROM outreach
WHERE status = 'PENDING'
  AND scheduled_at <= NOW()
ORDER BY scheduled_at
LIMIT 100;
```

---

## 6.2 SQS messages must remain small

Preferred payload:

```json
{
  "outreachId": "uuid"
}
```

Do NOT put:
- XLSX contents
- template body
- resume
- OAuth credentials
- user object

into the queue.

The worker resolves all required information using the Outreach ID.

---

## 6.3 XLSX is input, not runtime state

Lifecycle:

```text
Upload XLSX
→ Validate
→ Store XLSX in S3
→ Create XLSX metadata record
→ Create Campaign
→ Create Outreach records
```

Each XLSX row becomes one Outreach record.

---

## 6.4 Campaign vs Outreach

Campaign means:

> I want to run this outreach operation.

Outreach means:

> This specific email needs to be / was sent to this recipient.

Example:

```text
Campaign #4
 ├── Outreach 1 -> company A
 ├── Outreach 2 -> company B
 └── Outreach 3 -> company C
```

---

## 6.5 User ownership

Every important user-owned entity contains `user_id`.

This includes:

- templates
- resumes
- xlsx_files
- campaigns
- outreach
- oauth_credentials

Campaign and Outreach both explicitly contain `user_id`.

This simplifies:
- authorization
- tenant isolation
- filtering
- indexing

Every user-facing lookup should include authenticated `user_id`.

Example:

```sql
SELECT *
FROM campaigns
WHERE id = $1
  AND user_id = $2;
```

---

# 7. Domain Models

## User

```text
User {
    id
    name
    email
    createdAt
    modifiedAt
}
```

## Template

```text
Template {
    id
    userId
    name
    subject
    body
    variables
    isDefault
    createdAt
    modifiedAt
}
```

## Resume

```text
Resume {
    id
    userId
    name
    s3Path
    uploadedAt
}
```

## XLSXFile

```text
XLSXFile {
    id
    userId
    fileName
    storagePath
    uploadedAt
}
```

## Campaign

```text
Campaign {
    id
    campaignNumber
    userId
    xlsxFileId
    templateId
    status
    total
    sent
    failed
    createdAt
    scheduledAt
    startedAt
    finishedAt
}
```

Campaign states:

```text
DRAFT
SCHEDULED
RUNNING
COMPLETED
CANCELLED
```

A `FAILED` campaign state may be added later if useful.

## Outreach

```text
Outreach {
    id
    userId
    campaignId
    companyName
    role
    recipientEmail
    status
    scheduledAt
    sentAt
    failedAt
    gmailMessageId
    errorMessage
    createdAt
}
```

Outreach states:

```text
PENDING
QUEUED
PROCESSING
SENT
FAILED
CANCELLED
```

`QUEUED` is useful for scheduler ownership.

---

# 8. PostgreSQL Schema

Use UUID primary keys.

Recommended extension:

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;
```

Then UUID defaults can use `gen_random_uuid()`.

## users

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    modified_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## templates

```sql
CREATE TABLE templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    variables JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    modified_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Only one default template per user:

```sql
CREATE UNIQUE INDEX idx_one_default_template
ON templates(user_id)
WHERE is_default = TRUE;
```

User template listing:

```sql
CREATE INDEX idx_templates_user_id
ON templates(user_id);
```

Maximum 5 templates is enforced in the Go service in V1.

## resumes

```sql
CREATE TABLE resumes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    s3_path TEXT NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Exactly one active standard resume per user:

```sql
CREATE UNIQUE INDEX idx_one_resume_per_user
ON resumes(user_id);
```

## xlsx_files

```sql
CREATE TABLE xlsx_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    storage_path TEXT NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

```sql
CREATE INDEX idx_xlsx_files_user_id
ON xlsx_files(user_id);
```

## campaigns

```sql
CREATE TABLE campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    campaign_number INTEGER NOT NULL,
    xlsx_file_id UUID NOT NULL REFERENCES xlsx_files(id),
    template_id UUID NOT NULL REFERENCES templates(id),

    status VARCHAR(20) NOT NULL,

    total INTEGER NOT NULL DEFAULT 0,
    sent INTEGER NOT NULL DEFAULT 0,
    failed INTEGER NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    scheduled_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);
```

Campaign number unique per user:

```sql
CREATE UNIQUE INDEX idx_campaign_number_per_user
ON campaigns(user_id, campaign_number);
```

```sql
CREATE INDEX idx_campaigns_user_id
ON campaigns(user_id);
```

Optional status validation:

```sql
ALTER TABLE campaigns
ADD CONSTRAINT campaigns_status_check
CHECK (status IN ('DRAFT', 'SCHEDULED', 'RUNNING', 'COMPLETED', 'CANCELLED'));
```

Counters should never be negative:

```sql
ALTER TABLE campaigns
ADD CONSTRAINT campaigns_counter_check
CHECK (
    total >= 0
    AND sent >= 0
    AND failed >= 0
    AND sent + failed <= total
);
```

## outreach

```sql
CREATE TABLE outreach (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,

    company_name VARCHAR(255) NOT NULL,
    role VARCHAR(255) NOT NULL,
    recipient_email VARCHAR(255) NOT NULL,

    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',

    scheduled_at TIMESTAMPTZ NOT NULL,
    sent_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,

    gmail_message_id VARCHAR(255),
    error_message TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Prevent duplicate recipient inside a campaign:

```sql
CREATE UNIQUE INDEX idx_unique_campaign_recipient
ON outreach(campaign_id, recipient_email);
```

Scheduler index:

```sql
CREATE INDEX idx_outreach_scheduler
ON outreach(status, scheduled_at);
```

User ownership lookup:

```sql
CREATE INDEX idx_outreach_user_id
ON outreach(user_id);
```

Campaign outreach listing:

```sql
CREATE INDEX idx_outreach_campaign_id
ON outreach(campaign_id);
```

Status constraint:

```sql
ALTER TABLE outreach
ADD CONSTRAINT outreach_status_check
CHECK (
    status IN (
        'PENDING',
        'QUEUED',
        'PROCESSING',
        'SENT',
        'FAILED',
        'CANCELLED'
    )
);
```

## oauth_credentials

```sql
CREATE TABLE oauth_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,

    provider VARCHAR(50) NOT NULL DEFAULT 'google',

    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    modified_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Important:
OAuth credentials are sensitive.

For local V1, environment/application-level encryption is acceptable.

For production:
- encrypt tokens at rest
- use AWS KMS or equivalent
- never log tokens
- never return tokens to frontend

---

# 9. API Design

Authentication determines current user.

Do not accept arbitrary `userId` from frontend for ownership-sensitive operations.

## User

### POST /users

Called after Google OAuth identity has been obtained.

Creates the user if necessary.

Also creates the default template.

Possible request:

```json
{
  "name": "Murali",
  "email": "user@example.com"
}
```

In the final OAuth flow, name/email should ideally come from verified Google identity rather than trusting arbitrary frontend input.

---

# 10. Template APIs

## GET /templates

Returns authenticated user's templates.

## POST /templates

Example:

```json
{
  "name": "Backend Role",
  "subject": "Application for {{role}} at {{company_name}}",
  "body": "Hi,\n\nI am reaching out regarding opportunities at {{company_name}}...",
  "isDefault": false
}
```

Rules:
- maximum 5 templates
- require non-empty name/subject/body
- require supported variable syntax
- require `{{company_name}}`
- if creating a new default:
  - inside one DB transaction, unset existing default
  - create/set new default

## PUT /templates/{id}

Ownership check required.

If setting it as default:
- transactionally unset existing default first

## DELETE /templates/{id}

Do not allow deletion of the only/default template unless another template is first promoted to default.

Keep behavior simple.

---

# 11. Resume API

## POST /resume

Multipart upload.

Validation:
- PDF only
- reasonable file-size limit, e.g. 5 MB or 10 MB

Flow:

```text
receive file
→ validate
→ upload to S3
→ replace existing resume record if needed
→ return metadata
```

Suggested S3 key:

```text
users/{userId}/resume/{uuid}.pdf
```

Do not store PDF bytes in PostgreSQL.

---

# 12. XLSX API

## POST /files/xlsx

Multipart upload.

Flow:

```text
receive file
→ verify XLSX format
→ parse first sheet
→ validate columns
→ validate <= 400 rows
→ validate fields
→ validate emails
→ detect duplicate recipients
→ upload original XLSX to S3
→ create xlsx_files record
→ return metadata + validation summary
```

Suggested S3 key:

```text
users/{userId}/xlsx/{uuid}.xlsx
```

Expected columns:

```text
Company Name | Role | Company Mail
```

Return example:

```json
{
  "id": "uuid",
  "fileName": "companies.xlsx",
  "rowCount": 100,
  "valid": true
}
```

---

# 13. Campaign APIs

## POST /campaigns

Example:

```json
{
  "xlsxFileId": "uuid",
  "templateId": "uuid",
  "startAt": "2026-09-04T09:00:00Z"
}
```

For immediate start, frontend may send current UTC timestamp.

Avoid special string `"now"` if a normal timestamp is simpler.

Backend validates:
- XLSX belongs to user
- template belongs to user
- resume exists
- OAuth credentials exist
- XLSX has already passed validation

Inside transaction:

1. generate user's next campaign number
2. create Campaign
3. parse/load validated XLSX data
4. create one Outreach per row
5. assign scheduled times at 3-minute intervals
6. set Campaign total

Scheduling formula:

```text
outreach[i].scheduled_at =
    campaign.start_at + i * 3 minutes
```

Where first row has `i = 0`.

Campaign initial status:

- if startAt <= now -> `RUNNING` or `SCHEDULED` until scheduler starts
- simplest V1: always create as `SCHEDULED`
- scheduler may transition campaign to `RUNNING` on first due Outreach

## GET /campaigns

Return authenticated user's campaigns.

## GET /campaigns/{id}

Used by dashboard polling.

Example response:

```json
{
  "id": "uuid",
  "campaignNumber": 3,
  "status": "RUNNING",
  "total": 100,
  "sent": 37,
  "failed": 2,
  "scheduledAt": "2026-09-04T09:00:00Z",
  "startedAt": "2026-09-04T09:00:10Z",
  "finishedAt": null
}
```

## POST /campaigns/{id}/cancel

Flow:

```text
verify ownership
→ update campaign status = CANCELLED
→ optionally mark future PENDING/QUEUED outreach CANCELLED
```

Do not depend on purging SQS.

---

# 14. Scheduler Design

Implement a Go scheduler.

It may initially run as:
- a goroutine inside the API process for local development
- later as a dedicated service/process/container

Recommended polling interval:
- 15 to 30 seconds

Scheduler responsibility:

```text
Find due PENDING Outreach
→ claim them
→ enqueue Outreach ID to SQS
→ mark QUEUED
```

## Important concurrency rule

Multiple scheduler loops must not claim the same row.

Use a database transaction with row locking.

Example approach:

```sql
SELECT id
FROM outreach
WHERE status = 'PENDING'
  AND scheduled_at <= NOW()
ORDER BY scheduled_at
FOR UPDATE SKIP LOCKED
LIMIT 50;
```

Then update selected rows to `QUEUED`.

However there is a DB-to-SQS dual-write failure window:

```text
DB says QUEUED
→ application crashes
→ message never reaches SQS
```

For V1:
- keep implementation simple
- add a recovery job that finds stale QUEUED items and resets/requeues them

Production-hardening option:
- transactional outbox pattern

Do not implement the full outbox unless V1 is already working.

---

# 15. SQS Design

Queue message:

```json
{
  "outreachId": "uuid"
}
```

Recommended:
- standard SQS queue
- visibility timeout greater than worker's expected processing time
- dead-letter queue after several failed receives

Worker must delete message only after processing is complete.

SQS provides at-least-once delivery.

Therefore duplicate delivery is possible.

---

# 16. Email Worker Design

Worker flow:

```text
Receive SQS message
→ parse outreachId
→ load Outreach
→ load Campaign
→ check states
→ claim Outreach
→ load User
→ load Template
→ load Resume metadata
→ download resume from S3
→ load OAuth credentials
→ render template
→ send Gmail message
→ update Outreach
→ update Campaign counters
→ delete SQS message
```

## State checks before send

If Campaign is `CANCELLED`:
- mark Outreach CANCELLED if appropriate
- delete SQS message
- do not send

If Outreach is already:
- SENT
- FAILED
- CANCELLED

then treat message as duplicate/no-op and delete it.

To claim:

```sql
UPDATE outreach
SET status = 'PROCESSING'
WHERE id = $1
  AND status = 'QUEUED';
```

Check affected row count.

If zero:
- another worker may have claimed it
- do not send

---

# 17. Duplicate Send / Exactly-Once Reality

SQS is at-least-once.

A difficult failure case exists:

```text
Worker sends Gmail email successfully
→ worker crashes
→ DB still says PROCESSING
→ SQS message becomes visible again
→ retry may send duplicate
```

Gmail send does not provide a simple guaranteed exactly-once idempotency primitive for this use case.

Therefore:

- do not claim exactly-once delivery
- design for at-least-once processing
- reduce duplicates using state transitions
- save Gmail message ID when possible
- set a deterministic RFC Message-ID if practical, but do not assume this provides hard exactly-once guarantees
- later consider reconciliation/recovery strategies

For V1, document this known limitation.

This is an important system-design tradeoff, not a reason to block the build.

---

# 18. Gmail Integration

Do NOT use:
- Gmail password
- passkey
- App Password
- SMTP credentials stored in ApplyFlow

Use:
- Google OAuth
- Gmail API
- scope:

```text
https://www.googleapis.com/auth/gmail.send
```

Email worker responsibility:
- build MIME email
- To
- Subject
- body
- PDF attachment
- base64url encode message
- send through Gmail API

Store returned Gmail message ID in:

```text
outreach.gmail_message_id
```

Do not log access tokens or refresh tokens.

---

# 19. Template Rendering

Create a small renderer.

Inputs:

```text
User.name
Outreach.company_name
Outreach.role
```

Replacement map:

```text
{{user_name}}    -> user.name
{{company_name}} -> outreach.company_name
{{role}}         -> outreach.role
```

Render both:
- subject
- body

Keep rendering deterministic and simple.

Do not add a full template language engine unless needed.

---

# 20. Failure Handling

Distinguish temporary and permanent failures.

## Permanent examples

- invalid recipient formatting that somehow escaped validation
- malformed email payload
- OAuth permission revoked and requires user intervention

Possible result:

```text
Outreach -> FAILED
failed_at = now
error_message = sanitized error
Campaign.failed += 1
```

## Temporary examples

- Gmail API timeout
- transient 5xx
- temporary AWS/network failure

Allow SQS retry.

Do not immediately mark Outreach permanently failed for retryable errors.

After maximum retries / DLQ handling:
- mark FAILED
- increment failed count

---

# 21. Campaign Counter Updates

Campaign contains denormalized counters:

```text
total
sent
failed
```

These exist so dashboard reads are cheap.

When Outreach transitions to `SENT` for the first time:

```sql
UPDATE campaigns
SET sent = sent + 1
WHERE id = $campaign_id;
```

When final failure occurs:

```sql
UPDATE campaigns
SET failed = failed + 1
WHERE id = $campaign_id;
```

Do these transitions carefully so duplicate messages do not increment counters twice.

Recommended:
- update Outreach state with a conditional WHERE clause
- increment Campaign counter only if that state transition succeeded

Use a DB transaction where possible.

Campaign completes when:

```text
sent + failed + cancelled == total
```

Since V1 Campaign does not store a cancelled counter, simplest rule for normal non-cancelled campaigns:

```text
sent + failed == total
```

Then set:

```text
status = COMPLETED
finished_at = NOW()
```

For cancelled campaigns, leave Campaign status `CANCELLED`.

---

# 22. Authentication & Authorization

Authentication:
- Google OAuth

Authorization rule:
- every request acts only on resources belonging to authenticated user

Never use frontend-provided user IDs as trust boundaries.

Examples:

```sql
SELECT *
FROM templates
WHERE id = $templateID
  AND user_id = $authenticatedUserID;
```

```sql
SELECT *
FROM campaigns
WHERE id = $campaignID
  AND user_id = $authenticatedUserID;
```

---

# 23. Suggested Go Project Structure

```text
applyflow/
│
├── cmd/
│   ├── api/
│   │   └── main.go
│   └── worker/
│       └── main.go
│
├── internal/
│   ├── config/
│   │   └── config.go
│   │
│   ├── model/
│   │   ├── user.go
│   │   ├── template.go
│   │   ├── resume.go
│   │   ├── xlsx_file.go
│   │   ├── campaign.go
│   │   └── outreach.go
│   │
│   ├── repository/
│   │   ├── user_repository.go
│   │   ├── template_repository.go
│   │   ├── resume_repository.go
│   │   ├── xlsx_repository.go
│   │   ├── campaign_repository.go
│   │   ├── outreach_repository.go
│   │   └── oauth_repository.go
│   │
│   ├── service/
│   │   ├── auth_service.go
│   │   ├── template_service.go
│   │   ├── resume_service.go
│   │   ├── xlsx_service.go
│   │   ├── campaign_service.go
│   │   ├── scheduler_service.go
│   │   └── email_service.go
│   │
│   ├── handler/
│   │   ├── auth_handler.go
│   │   ├── template_handler.go
│   │   ├── resume_handler.go
│   │   ├── xlsx_handler.go
│   │   └── campaign_handler.go
│   │
│   ├── middleware/
│   │   └── auth.go
│   │
│   ├── storage/
│   │   └── s3.go
│   │
│   ├── queue/
│   │   └── sqs.go
│   │
│   ├── gmail/
│   │   └── client.go
│   │
│   └── xlsx/
│       └── parser.go
│
├── migrations/
│   └── 001_init.sql
│
├── .env.example
├── go.mod
├── go.sum
└── README.md
```

Do not create abstractions merely for abstraction's sake.

Interfaces are useful where external dependencies or testing boundaries exist, especially:
- repositories
- S3
- SQS
- Gmail sender

---

# 24. Recommended Go Libraries

Prefer standard library where possible.

Likely dependencies:

```text
github.com/jackc/pgx/v5
github.com/xuri/excelize/v2
github.com/google/uuid
github.com/aws/aws-sdk-go-v2
golang.org/x/oauth2
google.golang.org/api/gmail/v1
```

Router options:
- Go 1.22+ `http.ServeMux` is enough
- Chi is also acceptable

Prefer simple standard `net/http` unless a framework is already desired.

---

# 25. Configuration

Environment variables:

```text
APP_ENV
HTTP_PORT

DATABASE_URL

AWS_REGION
S3_BUCKET
SQS_QUEUE_URL

GOOGLE_CLIENT_ID
GOOGLE_CLIENT_SECRET
GOOGLE_REDIRECT_URL

FRONTEND_URL
```

Never commit:
- OAuth client secret
- DB passwords
- access tokens
- refresh tokens
- AWS secrets

Use AWS SDK default credential chain rather than hardcoding keys.

---

# 26. Default Template

When a new User is created, create a default template automatically.

Example:

Subject:

```text
Application for {{role}} at {{company_name}}
```

Body:

```text
Hi,

I am reaching out regarding {{role}} opportunities at {{company_name}}.

I have attached my resume for your consideration.

Regards,
{{user_name}}
```

This content is only a V1 placeholder and may later be edited by the user.

---

# 27. Campaign Creation Transaction

Campaign creation should be atomic.

Pseudo flow:

```text
BEGIN

verify user owns template
verify user owns xlsx
verify user has resume

get next campaign_number

INSERT campaign

for every parsed XLSX row:
    scheduledAt = startAt + index * 3 minutes
    INSERT outreach

UPDATE campaign total

COMMIT
```

If anything fails:
- rollback
- no partial campaign

For race-safe campaign numbering, avoid:

```text
SELECT MAX(campaign_number) + 1
```

without locking.

Simplest V1 alternatives:
- drop human campaign number entirely and use UUID
- or lock user/campaign numbering path transactionally

If retaining `campaign_number`, implement it safely enough for one-user V1.

---

# 28. Scheduler Pseudocode

```go
ticker := time.NewTicker(15 * time.Second)

for range ticker.C {
    rows := claimDueOutreach(ctx, 50)

    for _, row := range rows {
        err := queue.Send(ctx, row.ID)

        if err != nil {
            // restore or leave recoverable
            markPendingAgain(ctx, row.ID)
        }
    }
}
```

Do not hold a DB transaction open while making slow network calls to SQS.

A practical V1 pattern:

1. transactionally claim PENDING -> QUEUED
2. commit
3. publish to SQS
4. if publish fails, reset to PENDING
5. stale QUEUED recovery protects crash window

---

# 29. Worker Pseudocode

```go
func ProcessMessage(ctx context.Context, outreachID uuid.UUID) error {
    outreach, err := repo.GetOutreach(ctx, outreachID)
    if err != nil {
        return err
    }

    campaign, err := repo.GetCampaign(ctx, outreach.CampaignID)
    if err != nil {
        return err
    }

    if campaign.Status == "CANCELLED" {
        repo.CancelOutreach(...)
        return nil
    }

    claimed, err := repo.MarkProcessingIfQueued(ctx, outreachID)
    if err != nil {
        return err
    }

    if !claimed {
        return nil
    }

    // load user/template/resume/oauth
    // render subject/body
    // download resume
    // send Gmail

    // success:
    // transition PROCESSING -> SENT
    // increment campaign.sent exactly once

    return nil
}
```

---

# 30. Observability

At minimum use structured logging.

Log fields like:

```text
request_id
user_id
campaign_id
outreach_id
operation
status
duration
error
```

Never log:
- OAuth tokens
- full email body unless explicitly safe
- resume binary
- secrets

Recommended key events:

```text
campaign_created
outreach_claimed
outreach_enqueued
email_send_started
email_sent
email_failed
campaign_cancelled
campaign_completed
```

---

# 31. HTTP Error Shape

Use one consistent error format.

Example:

```json
{
  "error": {
    "code": "INVALID_XLSX",
    "message": "Company Mail is missing in row 8"
  }
}
```

Do not expose raw DB/AWS/Google errors directly to users.

Log technical error internally and return safe client message.

---

# 32. Validation Rules

## Email

Use reasonable syntax validation.

Do not overbuild RFC-perfect validation.

## Upload sizes

Suggested:
- resume <= 10 MB
- XLSX <= 10 MB

## File types

Check:
- filename extension
- MIME type where useful
- parser success

Do not trust extension alone.

---

# 33. Testing Priorities

Do not chase high test coverage first.

Critical tests:

## Template
- max 5 templates
- only one default
- variable rendering

## XLSX
- missing columns
- >400 rows
- blank fields
- invalid email
- duplicate recipient

## Campaign
- correct number of Outreach rows
- correct 3-minute spacing
- ownership checks

## Scheduler
- only due items claimed
- cancelled campaign not enqueued
- same row not claimed twice concurrently

## Worker
- cancelled campaign does not send
- already SENT outreach is no-op
- successful send updates state once
- retryable failure does not permanently fail immediately

---

# 34. Local Development Strategy

To move fast:

1. PostgreSQL via Docker
2. Go API locally
3. Scheduler in API process initially if desired
4. Worker as separate `cmd/worker`
5. AWS S3/SQS can be real dev resources
6. Gmail API can use user's test Google account
7. frontend can be minimal

Example PostgreSQL Docker compose is acceptable.

---

# 35. Frontend V1

Keep frontend minimal.

Screens:

```text
Login
Dashboard
Resume
Templates
Upload XLSX / Preview
Campaign Detail
```

Campaign detail displays:

```text
Campaign #3
RUNNING

37 / 100 sent
2 failed

[Cancel Campaign]
```

Poll backend every 3-5 seconds.

Do not build a complex design system.

---

# 36. Deployment Evolution

Do not block local V1 on perfect AWS deployment.

Later target could be:

```text
Frontend -> static hosting
Go API -> ECS/Fargate
Worker -> ECS/Fargate or Lambda
Scheduler -> ECS service / scheduled process
PostgreSQL -> RDS
Files -> S3
Queue -> SQS
Secrets -> Parameter Store / Secrets Manager
Logs -> CloudWatch
```

EventBridge Scheduler may later replace or complement the Go scheduler, but V1 intentionally uses a Go scheduler so the scheduling logic is explicit and understandable.

---

# 37. Security Checklist

Must:
- authenticated routes
- ownership check on every user resource
- no OAuth token logging
- no secrets in Git
- validate uploads
- sanitize user-facing errors
- parameterized SQL only
- do not build SQL through string concatenation
- keep Gmail scope limited to `gmail.send`
- store OAuth credentials separately from normal User fields
- use HTTPS in deployed environments

---

# 38. Important V1 Tradeoffs

These are deliberate.

## We accept:
- polling instead of WebSockets
- application-enforced max template count
- PostgreSQL scheduler state
- at-least-once queue semantics
- imperfect exactly-once email guarantee
- denormalized Campaign counters
- one standard resume
- one email every 3 minutes

## We avoid:
- premature microservices
- Kafka
- Redis unless a real need appears
- event sourcing
- complex distributed locks
- full outbox pattern before V1 works
- unnecessary abstractions

---

# 39. Definition of Done for V1

V1 is DONE when this works end-to-end:

```text
Google Login
     ↓
User exists
     ↓
Default template exists
     ↓
Upload Resume
     ↓
Upload validated XLSX
     ↓
Select Template
     ↓
Start Campaign
     ↓
Campaign + Outreach records created
     ↓
Scheduler finds due Outreach
     ↓
SQS receives outreachId
     ↓
Worker receives job
     ↓
Worker sends Gmail email with PDF attachment
     ↓
Outreach -> SENT
     ↓
Campaign.sent increases
     ↓
Frontend polling shows updated progress
     ↓
Campaign eventually -> COMPLETED
```

Also verify:

```text
Cancel Campaign
     ↓
No further emails are sent
```

---

# 40. Recommended Build Order for Cursor

Cursor should implement in this sequence.

## Phase 1 — Foundation

1. initialize Go module
2. project structure
3. config loader
4. PostgreSQL connection
5. migration
6. health endpoint

Expected:

```text
GET /health -> 200 OK
```

## Phase 2 — Data layer

7. models
8. repositories
9. basic repository tests

## Phase 3 — User/Auth

10. Google OAuth
11. user creation
12. default template creation
13. auth middleware

## Phase 4 — Template/Resume

14. template CRUD
15. max 5 rule
16. default template behavior
17. resume upload to S3

## Phase 5 — XLSX

18. XLSX parser
19. validation
20. upload to S3
21. xlsx_files metadata

## Phase 6 — Campaign

22. campaign creation
23. Outreach creation
24. scheduled_at generation
25. campaign GET APIs
26. cancellation

## Phase 7 — Queue pipeline

27. SQS wrapper
28. scheduler
29. worker
30. state transitions
31. stale QUEUED recovery

## Phase 8 — Gmail

32. OAuth token loading/refresh
33. MIME builder
34. PDF attachment
35. Gmail API send
36. persistence of Gmail message ID

## Phase 9 — Dashboard

37. minimal frontend
38. polling
39. campaign progress
40. cancel action

## Phase 10 — Hardening

41. retry behavior
42. DLQ
43. structured logging
44. input/error cleanup
45. integration tests

---

# 41. Instructions to Cursor

When implementing this spec:

1. Build the smallest working V1.
2. Do not invent additional product features.
3. Prefer readable Go over clever Go.
4. Keep handlers thin.
5. Put business logic in services.
6. Put SQL/persistence in repositories.
7. Use context.Context consistently.
8. Return errors rather than panic.
9. Use parameterized SQL.
10. Use transactions around operations requiring atomicity.
11. Keep SQS payload to `outreachId`.
12. Do not expose OAuth credentials.
13. Do not use Gmail SMTP/App Passwords.
14. Use Google OAuth + Gmail API.
15. Do not send emails directly from HTTP handlers.
16. Do not schedule 20 hours of messages inside SQS.
17. Keep future work in PostgreSQL.
18. Make all authenticated queries tenant-safe using `user_id`.
19. Avoid giant files; split by responsibility.
20. After each phase, ensure the application still builds and tests pass.

---

# 42. Cursor Initial Prompt

Use this with Cursor:

```text
Build the ApplyFlow V1 described in APPLYFLOW_DESIGN.md.

Treat this document as the source of truth.

Start with Phase 1 only:
- initialize/verify project structure
- config loader
- PostgreSQL connection using pgx
- database migration
- GET /health

Do not implement later phases yet.

Before writing code:
1. inspect the existing repository
2. preserve good existing code
3. tell me which files you will add/change
4. implement Phase 1
5. run gofmt
6. run go test ./...
7. run go vet ./...
8. show me any errors and fix them
9. summarize what was implemented and what remains

Important:
- readable idiomatic Go
- no unnecessary framework
- use net/http
- context-aware DB operations
- environment-based config
- never hardcode secrets
```

Then continue phase-by-phase rather than asking Cursor to generate the entire system in one enormous pass.

---

# 43. Engineering Principles for This Project

The purpose of ApplyFlow is not just to generate code.

The implementation should make these architectural decisions visible:

```text
HTTP handlers
      ↓
services
      ↓
repositories / external clients
```

and:

```text
PostgreSQL = durable application state
S3        = file storage
SQS       = work delivery
Scheduler = determines when work becomes due
Worker    = performs side effects
Gmail API = email provider
```

A developer working on this project should be able to explain:

- why SQS is not the scheduler
- why Outlook rows are stored in PostgreSQL
- why only IDs go into SQS
- why cancellation is state-based
- why worker operations must be idempotent where possible
- why exactly-once Gmail sending cannot simply be assumed
- why Campaign counters are denormalized
- why `user_id` exists on Campaign and Outreach
- why OAuth tokens are stored separately
- why files live in S3 rather than PostgreSQL

That architectural ownership matters more than manually typing every line of code.

---

# 44. Final V1 Principle

**AI may write the code. The developer must own the decisions.**

ApplyFlow should remain simple enough that the developer can explain every major component and failure mode in an interview.
