-- Align account sharing with the reference account-mode workflow.
ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS account_mode_platform VARCHAR(20)
        CHECK (account_mode_platform IS NULL OR account_mode_platform IN ('openai', 'anthropic'));
CREATE INDEX IF NOT EXISTS idx_api_keys_account_mode
    ON api_keys(user_id, account_mode_platform)
    WHERE deleted_at IS NULL AND account_mode_platform IS NOT NULL;

ALTER TABLE proxies
    ADD COLUMN IF NOT EXISTS owner_user_id BIGINT REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS ip_type VARCHAR(10) NOT NULL DEFAULT 'ipv4'
        CHECK (ip_type IN ('ipv4', 'ipv6'));
CREATE INDEX IF NOT EXISTS idx_proxies_user_owned
    ON proxies(owner_user_id, created_at DESC)
    WHERE deleted_at IS NULL AND owner_user_id IS NOT NULL;

ALTER TABLE community_accounts
    ADD COLUMN IF NOT EXISTS proxy_id BIGINT REFERENCES proxies(id) ON DELETE SET NULL;

ALTER TABLE community_listings
    ADD COLUMN IF NOT EXISTS feature_tags TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS codex_cli_only BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS usage_updated_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS health_status VARCHAR(20) NOT NULL DEFAULT 'healthy'
        CHECK (health_status IN ('healthy', 'degraded', 'unavailable'));
ALTER TABLE community_listings
    DROP CONSTRAINT IF EXISTS community_listings_usage_multiplier_check;
ALTER TABLE community_listings
    ADD CONSTRAINT community_listings_usage_multiplier_check
        CHECK (usage_multiplier >= 0);

ALTER TABLE community_memberships
    ADD COLUMN IF NOT EXISTS idle_timeout_minutes INTEGER NOT NULL DEFAULT 10
        CHECK (idle_timeout_minutes BETWEEN 1 AND 1440),
    ADD COLUMN IF NOT EXISTS activated_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_request_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS idle_expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS paid_until TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS billed_until TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS dispatch_cooldown_until TIMESTAMPTZ;

-- A user may use the same listing with different account-mode keys. The key,
-- not the user alone, owns the ordered reservation.
DROP INDEX IF EXISTS idx_community_memberships_active;
CREATE UNIQUE INDEX IF NOT EXISTS idx_community_memberships_listing_key_active
    ON community_memberships(listing_id, api_key_id)
    WHERE status IN ('reserved', 'active') AND api_key_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_community_memberships_idle
    ON community_memberships(idle_expires_at)
    WHERE status = 'active';

CREATE TABLE IF NOT EXISTS community_billing_windows (
    id BIGSERIAL PRIMARY KEY,
    membership_id BIGINT NOT NULL REFERENCES community_memberships(id),
    listing_id BIGINT NOT NULL REFERENCES community_listings(id),
    payer_user_id BIGINT NOT NULL REFERENCES users(id),
    owner_user_id BIGINT NOT NULL REFERENCES users(id),
    window_started_at TIMESTAMPTZ NOT NULL,
    window_ends_at TIMESTAMPTZ NOT NULL,
    active_seconds BIGINT NOT NULL DEFAULT 0 CHECK (active_seconds >= 0),
    request_spend DECIMAL(20,8) NOT NULL DEFAULT 0 CHECK (request_spend >= 0),
    hourly_fee_precharged DECIMAL(20,8) NOT NULL DEFAULT 0 CHECK (hourly_fee_precharged >= 0),
    hourly_fee_refunded DECIMAL(20,8) NOT NULL DEFAULT 0 CHECK (hourly_fee_refunded >= 0),
    commission_rate DECIMAL(8,4) NOT NULL CHECK (commission_rate BETWEEN 0 AND 100),
    status VARCHAR(20) NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'settled', 'cancelled')),
    settled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(membership_id, window_started_at)
);
CREATE INDEX IF NOT EXISTS idx_community_billing_windows_open
    ON community_billing_windows(window_ends_at)
    WHERE status = 'open';

CREATE TABLE IF NOT EXISTS community_reviews (
    id BIGSERIAL PRIMARY KEY,
    listing_id BIGINT NOT NULL REFERENCES community_listings(id),
    membership_id BIGINT NOT NULL REFERENCES community_memberships(id),
    reviewer_user_id BIGINT NOT NULL REFERENCES users(id),
    owner_user_id BIGINT NOT NULL REFERENCES users(id),
    score DECIMAL(4,2) NOT NULL CHECK (score BETWEEN 1 AND 10),
    content TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(membership_id, reviewer_user_id)
);
CREATE INDEX IF NOT EXISTS idx_community_reviews_owner
    ON community_reviews(owner_user_id, created_at DESC);

ALTER TABLE community_settlements
    ADD COLUMN IF NOT EXISTS request_id VARCHAR(160),
    ADD COLUMN IF NOT EXISTS usage_amount DECIMAL(20,8) NOT NULL DEFAULT 0
        CHECK (usage_amount >= 0);
CREATE UNIQUE INDEX IF NOT EXISTS idx_community_settlements_request
    ON community_settlements(request_id)
    WHERE request_id IS NOT NULL;
