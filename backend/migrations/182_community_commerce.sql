-- User-owned OAuth accounts, sharing marketplace, support, withdrawals, and store.
ALTER TABLE users ADD COLUMN IF NOT EXISTS points DECIMAL(20,8) NOT NULL DEFAULT 0 CHECK (points >= 0);

CREATE TABLE IF NOT EXISTS community_accounts (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES users(id),
    name VARCHAR(120) NOT NULL,
    provider VARCHAR(20) NOT NULL CHECK (provider IN ('openai', 'anthropic')),
    credential_encrypted TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'paused', 'invalid', 'rejected')),
    review_status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (review_status IN ('pending', 'approved', 'rejected')),
    review_note TEXT NOT NULL DEFAULT '',
    reviewed_by BIGINT REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    scheduler_account_id BIGINT REFERENCES accounts(id),
    share_mode VARCHAR(20) NOT NULL DEFAULT 'private' CHECK (share_mode IN ('private', 'public')),
    account_tier VARCHAR(20) NOT NULL DEFAULT 'oauth',
    capacity INTEGER NOT NULL DEFAULT 1 CHECK (capacity BETWEEN 1 AND 1000),
    concurrency INTEGER NOT NULL DEFAULT 1 CHECK (concurrency BETWEEN 1 AND 1000),
    schedulable BOOLEAN NOT NULL DEFAULT FALSE,
    priority INTEGER NOT NULL DEFAULT 50 CHECK (priority BETWEEN 0 AND 1000),
    today_requests BIGINT NOT NULL DEFAULT 0,
    today_tokens BIGINT NOT NULL DEFAULT 0,
    usage_5h_percent DECIMAL(6,2) NOT NULL DEFAULT 0 CHECK (usage_5h_percent BETWEEN 0 AND 100),
    usage_7d_percent DECIMAL(6,2) NOT NULL DEFAULT 0 CHECK (usage_7d_percent BETWEEN 0 AND 100),
    provider_options JSONB NOT NULL DEFAULT '{}',
    tags TEXT[] NOT NULL DEFAULT '{}',
    supported_models TEXT[] NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_community_accounts_owner ON community_accounts(owner_user_id, created_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS community_listings (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL UNIQUE REFERENCES community_accounts(id),
    owner_user_id BIGINT NOT NULL REFERENCES users(id),
    title VARCHAR(160) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    seat_limit INTEGER NOT NULL DEFAULT 1 CHECK (seat_limit BETWEEN 1 AND 1000),
    per_user_concurrency INTEGER NOT NULL DEFAULT 1 CHECK (per_user_concurrency BETWEEN 1 AND 1000),
    minimum_balance DECIMAL(20,8) NOT NULL DEFAULT 0 CHECK (minimum_balance >= 0),
    hourly_price DECIMAL(20,8) NOT NULL DEFAULT 0 CHECK (hourly_price >= 0),
    hourly_minimum_spend DECIMAL(20,8) NOT NULL DEFAULT 0 CHECK (hourly_minimum_spend >= 0),
    usage_multiplier DECIMAL(12,4) NOT NULL DEFAULT 1 CHECK (usage_multiplier > 0),
    idle_timeout_minutes INTEGER NOT NULL DEFAULT 30 CHECK (idle_timeout_minutes BETWEEN 1 AND 10080),
    commission_rate DECIMAL(8,4) NOT NULL DEFAULT 10 CHECK (commission_rate BETWEEN 0 AND 100),
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'paused', 'sold_out', 'rejected')),
    score DECIMAL(5,2) NOT NULL DEFAULT 5,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_community_listings_market ON community_listings(status, updated_at DESC);

CREATE TABLE IF NOT EXISTS community_memberships (
    id BIGSERIAL PRIMARY KEY,
    listing_id BIGINT NOT NULL REFERENCES community_listings(id),
    member_user_id BIGINT NOT NULL REFERENCES users(id),
    api_key_id BIGINT REFERENCES api_keys(id),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('reserved', 'active', 'left', 'expired', 'suspended')),
    reservation_order INTEGER NOT NULL DEFAULT 1 CHECK (reservation_order BETWEEN 1 AND 5),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_community_memberships_active ON community_memberships(listing_id, member_user_id) WHERE status IN ('reserved', 'active');
CREATE INDEX IF NOT EXISTS idx_community_memberships_key_queue ON community_memberships(api_key_id, reservation_order) WHERE status IN ('reserved', 'active');
CREATE TABLE IF NOT EXISTS community_settlements (
    id BIGSERIAL PRIMARY KEY,
    listing_id BIGINT NOT NULL REFERENCES community_listings(id),
    membership_id BIGINT NOT NULL REFERENCES community_memberships(id),
    payer_user_id BIGINT NOT NULL REFERENCES users(id),
    owner_user_id BIGINT NOT NULL REFERENCES users(id),
    gross_amount DECIMAL(20,8) NOT NULL,
    commission_rate DECIMAL(8,4) NOT NULL,
    platform_fee DECIMAL(20,8) NOT NULL,
    owner_amount DECIMAL(20,8) NOT NULL,
    settlement_type VARCHAR(20) NOT NULL DEFAULT 'initial_hour',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_community_settlements_owner ON community_settlements(owner_user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS support_tickets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    subject VARCHAR(180) NOT NULL,
    category VARCHAR(40) NOT NULL DEFAULT 'general',
    priority VARCHAR(20) NOT NULL DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    status VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'waiting_user', 'waiting_admin', 'resolved', 'closed')),
    assigned_admin_id BIGINT REFERENCES users(id),
    user_unread INTEGER NOT NULL DEFAULT 0,
    admin_unread INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_support_tickets_user ON support_tickets(user_id, updated_at DESC);
CREATE TABLE IF NOT EXISTS support_ticket_messages (
    id BIGSERIAL PRIMARY KEY,
    ticket_id BIGINT NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    author_user_id BIGINT NOT NULL REFERENCES users(id),
    author_role VARCHAR(20) NOT NULL CHECK (author_role IN ('user', 'admin')),
    content TEXT NOT NULL,
    attachment_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_support_messages_ticket ON support_ticket_messages(ticket_id, created_at);

CREATE TABLE IF NOT EXISTS payout_methods (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    method VARCHAR(20) NOT NULL CHECK (method IN ('alipay', 'wechat')),
    qr_code_data TEXT NOT NULL,
    display_name VARCHAR(120) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, method)
);
CREATE TABLE IF NOT EXISTS withdrawals (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    payout_method VARCHAR(20) NOT NULL CHECK (payout_method IN ('alipay', 'wechat')),
    payout_snapshot TEXT NOT NULL,
    amount DECIMAL(20,8) NOT NULL CHECK (amount >= 1),
    fee DECIMAL(20,8) NOT NULL DEFAULT 0 CHECK (fee >= 0),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled', 'paid')),
    user_note TEXT NOT NULL DEFAULT '',
    admin_note TEXT NOT NULL DEFAULT '',
    payment_reference VARCHAR(180) NOT NULL DEFAULT '',
    reviewed_by BIGINT REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_withdrawals_one_pending ON withdrawals(user_id) WHERE status IN ('pending', 'approved');
CREATE INDEX IF NOT EXISTS idx_withdrawals_admin ON withdrawals(status, created_at DESC);

CREATE TABLE IF NOT EXISTS store_products (
    id BIGSERIAL PRIMARY KEY,
    category VARCHAR(80) NOT NULL DEFAULT '兑换码',
    name VARCHAR(160) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price DECIMAL(20,8) NOT NULL CHECK (price >= 0),
    points_price DECIMAL(20,8) CHECK (points_price IS NULL OR points_price >= 0),
    fulfillment_type VARCHAR(30) NOT NULL DEFAULT 'card' CHECK (fulfillment_type IN ('card', 'redeem_code', 'balance', 'entitlement')),
    fulfillment_value DECIMAL(20,8) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'inactive')),
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS store_inventory (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES store_products(id) ON DELETE CASCADE,
    secret_value TEXT NOT NULL, -- AES-GCM ciphertext; decrypted only during fulfillment.
    status VARCHAR(20) NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'reserved', 'sold', 'disabled')),
    order_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sold_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_store_inventory_available ON store_inventory(product_id, id) WHERE status = 'available';
CREATE TABLE IF NOT EXISTS store_orders (
    id BIGSERIAL PRIMARY KEY,
    order_no VARCHAR(48) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id),
    product_id BIGINT NOT NULL REFERENCES store_products(id),
    quantity INTEGER NOT NULL DEFAULT 1 CHECK (quantity BETWEEN 1 AND 100),
    unit_price DECIMAL(20,8) NOT NULL,
    total_amount DECIMAL(20,8) NOT NULL,
    payment_source VARCHAR(20) NOT NULL DEFAULT 'balance' CHECK (payment_source IN ('balance', 'points', 'platform')),
    payment_method VARCHAR(20) NOT NULL DEFAULT 'balance',
    status VARCHAR(20) NOT NULL DEFAULT 'paid' CHECK (status IN ('pending', 'paid', 'fulfilled', 'failed', 'refunded')),
    delivery JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paid_at TIMESTAMPTZ,
    fulfilled_at TIMESTAMPTZ
);
ALTER TABLE store_inventory DROP CONSTRAINT IF EXISTS store_inventory_order_fk;
ALTER TABLE store_inventory ADD CONSTRAINT store_inventory_order_fk FOREIGN KEY (order_id) REFERENCES store_orders(id);
CREATE INDEX IF NOT EXISTS idx_store_orders_user ON store_orders(user_id, created_at DESC);

INSERT INTO settings (key, value, updated_at)
VALUES ('community_marketplace_commission_percent', '10', NOW())
ON CONFLICT (key) DO NOTHING;
