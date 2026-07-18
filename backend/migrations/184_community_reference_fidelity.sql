-- Additional constraints and roles observed in the reference community flows.
ALTER TABLE support_ticket_messages
    DROP CONSTRAINT IF EXISTS support_ticket_messages_author_role_check;
ALTER TABLE support_ticket_messages
    ADD CONSTRAINT support_ticket_messages_author_role_check
        CHECK (author_role IN ('user', 'admin', 'system'));

ALTER TABLE store_orders
    ADD COLUMN IF NOT EXISTS payment_order_id BIGINT REFERENCES payment_orders(id),
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
CREATE UNIQUE INDEX IF NOT EXISTS idx_store_orders_payment_order
    ON store_orders(payment_order_id)
    WHERE payment_order_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_store_orders_expiry
    ON store_orders(expires_at)
    WHERE status = 'pending' AND payment_source = 'platform';
