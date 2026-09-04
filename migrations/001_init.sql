CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    modified_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

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

CREATE UNIQUE INDEX idx_one_default_template ON templates(user_id) WHERE is_default = TRUE;
CREATE INDEX idx_templates_user_id ON templates(user_id);

CREATE TABLE resumes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    s3_path TEXT NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_one_resume_per_user ON resumes(user_id);

CREATE TABLE xlsx_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    storage_path TEXT NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_xlsx_files_user_id ON xlsx_files(user_id);

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

CREATE UNIQUE INDEX idx_campaign_number_per_user ON campaigns(user_id, campaign_number);
CREATE INDEX idx_campaigns_user_id ON campaigns(user_id);

ALTER TABLE campaigns
ADD CONSTRAINT campaigns_status_check
CHECK (status IN ('DRAFT', 'SCHEDULED', 'RUNNING', 'COMPLETED', 'CANCELLED'));

ALTER TABLE campaigns
ADD CONSTRAINT campaigns_counter_check
CHECK (total >= 0 AND sent >= 0 AND failed >= 0 AND sent + failed <= total);

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

CREATE UNIQUE INDEX idx_unique_campaign_recipient ON outreach(campaign_id, recipient_email);
CREATE INDEX idx_outreach_scheduler ON outreach(status, scheduled_at);
CREATE INDEX idx_outreach_user_id ON outreach(user_id);
CREATE INDEX idx_outreach_campaign_id ON outreach(campaign_id);

ALTER TABLE outreach
ADD CONSTRAINT outreach_status_check
CHECK (status IN ('PENDING', 'QUEUED', 'PROCESSING', 'SENT', 'FAILED', 'CANCELLED'));

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
