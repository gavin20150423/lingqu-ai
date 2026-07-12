-- 邀请返利增加用户级线下结算控制和结算审计理由。

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS transfer_disabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN user_affiliates.transfer_disabled IS '是否禁止将邀请返利可用额度转入账户余额；开启后由管理员线下结算';

ALTER TABLE user_affiliate_ledger
    ADD COLUMN IF NOT EXISTS reason TEXT NULL;

COMMENT ON COLUMN user_affiliate_ledger.reason IS '流水操作理由；offline_settlement 固定记录为已线下结算';
COMMENT ON COLUMN user_affiliate_ledger.action IS 'accrue|transfer|offline_settlement';

CREATE INDEX IF NOT EXISTS idx_user_affiliates_transfer_disabled
    ON user_affiliates(transfer_disabled)
    WHERE transfer_disabled = TRUE;

CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_offline_settlement
    ON user_affiliate_ledger(user_id, created_at DESC)
    WHERE action = 'offline_settlement';
