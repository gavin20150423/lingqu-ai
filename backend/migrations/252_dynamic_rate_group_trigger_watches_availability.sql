-- 让账号级动态分组同步触发器也监听 status / schedulable。
--
-- 背景（2026-10-09，紧随 251 之后）：251 给 sync_dynamic_rate_groups_for_account()
-- 加上了"账号必须 active + schedulable"的过滤，但**触发器本身**仍然是
--   AFTER INSERT OR UPDATE OF platform, rate_multiplier, deleted_at ON accounts
-- 也就是说操作员在后台把账号禁用（status）或关掉调度开关（schedulable）时，
-- 触发器根本不触发，那条按新规则本该消失的绑定会一直留着，直到下一次分组级同步
-- （改分组配置、跑 251 的存量重算）才被清掉。实测：账号 787 关掉 schedulable 后，
-- 绑定数仍是 3；手工调用 sync_dynamic_rate_group() 才降到 0。
--
-- 这不是纯洁癖：绑定留着，就仍然存在"只看分组归属"的旁路把禁用账号捞回来的可能，
-- 而这正是 251 / v0.2.14-lingqu.2 要消除的那类绕过。
--
-- 修法：重建触发器，把 status、schedulable 纳入监听的列。
-- 监听列变多只会让同步更频繁，函数内部是幂等的（DELETE + INSERT ... ON CONFLICT DO UPDATE）。

DROP TRIGGER IF EXISTS trg_sync_dynamic_rate_groups_for_account ON accounts;

CREATE TRIGGER trg_sync_dynamic_rate_groups_for_account
AFTER INSERT OR UPDATE OF platform, rate_multiplier, deleted_at, status, schedulable
ON accounts
FOR EACH ROW
EXECUTE FUNCTION sync_dynamic_rate_groups_for_account();