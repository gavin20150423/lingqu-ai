-- Persist generated video copies in the admin-selected private OSS bucket.
-- Empty defaults preserve the existing upstream-proxy behavior for old jobs.
ALTER TABLE video_jobs
    ADD COLUMN IF NOT EXISTS storage_provider VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS storage_key TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS storage_content_type VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS storage_requested BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_video_jobs_missing_storage
    ON video_jobs (finished_at ASC NULLS LAST)
    WHERE status = 'completed' AND storage_requested = TRUE AND storage_provider = '';
