-- Subscription plan entitlements and time-boxed subscription promo codes.
ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS entitlements JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE user_subscriptions
    ADD COLUMN IF NOT EXISTS entitlements JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE promo_codes
    ADD COLUMN IF NOT EXISTS discount_percent NUMERIC(5,2) NOT NULL DEFAULT 0;
ALTER TABLE promo_codes
    ADD COLUMN IF NOT EXISTS applies_to_subscriptions BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE promo_codes
    ADD COLUMN IF NOT EXISTS starts_at TIMESTAMPTZ NULL;

ALTER TABLE payment_orders
    ADD COLUMN IF NOT EXISTS promo_code VARCHAR(32) NULL;

CREATE INDEX IF NOT EXISTS promo_codes_subscription_window_idx
    ON promo_codes (applies_to_subscriptions, starts_at, expires_at);
