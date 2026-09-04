ALTER TABLE outreach ADD COLUMN IF NOT EXISTS queued_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_outreach_stale_queued
ON outreach(status, queued_at)
WHERE status = 'QUEUED';
