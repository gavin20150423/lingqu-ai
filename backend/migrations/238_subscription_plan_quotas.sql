-- 订阅套餐独立额度：额度属于用户购买的套餐，不属于共享资源分组。
--
-- 这些列全部允许 NULL：NULL 表示该维度不限制；这样旧套餐和旧订阅仍保持
-- 原有的分组额度兼容行为。UserSubscription 上保存购买时的快照，避免管理员
-- 修改/删除套餐后影响已经购买的订阅。

ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS daily_limit_usd NUMERIC(20,8) NULL;

ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS weekly_limit_usd NUMERIC(20,8) NULL;

ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS monthly_limit_usd NUMERIC(20,8) NULL;

ALTER TABLE user_subscriptions
    ADD COLUMN IF NOT EXISTS plan_id BIGINT NULL;

ALTER TABLE user_subscriptions
    ADD COLUMN IF NOT EXISTS daily_limit_usd NUMERIC(20,8) NULL;

ALTER TABLE user_subscriptions
    ADD COLUMN IF NOT EXISTS weekly_limit_usd NUMERIC(20,8) NULL;

ALTER TABLE user_subscriptions
    ADD COLUMN IF NOT EXISTS monthly_limit_usd NUMERIC(20,8) NULL;

CREATE INDEX IF NOT EXISTS user_subscriptions_plan_id_idx
    ON user_subscriptions (plan_id);
