-- 订阅重置卡：用户可自助把「日 / 周」用量清零（月度额度不受影响）。
--
-- 语义：
--   * 1 张卡 = 1 次重置，只清 daily_usage_usd / weekly_usage_usd 及其窗口锚点，
--     月度用量与月度窗口完全不动。
--   * 余额放在独立表而不是 users 上：不需要改 ent 生成代码，且能独立审计。
--   * 默认 1 张只赠给**有订阅**的用户：无行时由代码按「是否存在未删除订阅」判定
--     （有则 1、无则 0）；这里给存量有订阅的用户回填 1 张，避免首次消耗时
--     在热路径上判定 + 抢写。
-- 审计：发放 / 设置 / 消耗 / 退回 全部写 user_subscription_reset_card_logs。

CREATE TABLE IF NOT EXISTS user_subscription_reset_cards (
    user_id    BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    balance    INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_subscription_reset_cards_balance_non_negative CHECK (balance >= 0)
);

CREATE TABLE IF NOT EXISTS user_subscription_reset_card_logs (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id BIGINT NULL,
    action          VARCHAR(20) NOT NULL, -- grant / set / consume / refund
    delta           INTEGER NOT NULL,
    balance_after   INTEGER NOT NULL,
    operator_id     BIGINT NULL,
    note            TEXT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_subscription_reset_card_logs_user
    ON user_subscription_reset_card_logs (user_id, created_at DESC);

-- 只给「有订阅（未删除）」的存量用户回填 1 张；没订阅的用户不送卡。
INSERT INTO user_subscription_reset_cards (user_id, balance)
SELECT DISTINCT s.user_id, 1
FROM user_subscriptions s
WHERE s.deleted_at IS NULL
ON CONFLICT (user_id) DO NOTHING;
