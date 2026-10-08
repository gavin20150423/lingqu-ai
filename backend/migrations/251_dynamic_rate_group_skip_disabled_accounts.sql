-- 动态速率分组自动绑定：不得把「操作员显式禁用」的账号绑进分组。
--
-- 背景（2026-10-08，与 v0.2.14-lingqu.2 的单账号强制路径修复配套）：
-- sync_dynamic_rate_group / sync_dynamic_rate_groups_for_account 的 INSERT 只按
-- 「平台一致 + 费率不超过上限 + 未删除」筛选，**不看账号状态**。于是
-- status <> 'active'（禁用/错误）或 schedulable = false（操作员手动关了调度开关）
-- 的账号仍被自动绑进动态分组——而这两者正是操作员明确表达"别用它"的开关。
-- 正常调度路径会过滤掉它们，所以平时不显形；但任何"只看分组归属"的旁路都会
-- 把禁用账号捞回来调度（2026-10-06 由 804/kiro-95 事故定案）。
--
-- 口径与 Go 侧保持一致：只有 active + schedulable 的账号可被规则自动绑定。
-- 账号侧触发器 sync_dynamic_rate_groups_for_account 在 accounts UPDATE 时触发，
-- 所以操作员重新启用账号后绑定会被自动补回 —— 本规则是自愈的，不需要额外补偿任务。
--
-- DELETE 分支同步补上"账号已不可用"这一条，让存量错误绑定在下一次同步时被清掉，
-- 而不是只对新绑定生效（否则状态会一直撒谎）。

CREATE OR REPLACE FUNCTION sync_dynamic_rate_group(p_group_id BIGINT)
RETURNS VOID
LANGUAGE plpgsql
AS $$
BEGIN
    -- Once enabled, a dynamic group is fully rule-managed. When the rule is
    -- disabled, only bindings created by the rule are removed.
    DELETE FROM account_groups ag
    USING groups g, accounts a
    WHERE ag.group_id = g.id
      AND ag.account_id = a.id
      AND g.id = p_group_id
      AND (
          (ag.auto_managed = TRUE AND NOT g.auto_assign_accounts_by_rate)
          OR (
              g.auto_assign_accounts_by_rate = TRUE
              AND (g.auto_assign_max_rate IS NULL
                   OR g.deleted_at IS NOT NULL
                   OR a.deleted_at IS NOT NULL
                   OR a.platform IS DISTINCT FROM g.platform
                   OR a.rate_multiplier > g.auto_assign_max_rate
                   OR a.status IS DISTINCT FROM 'active'
                   OR a.schedulable = FALSE)
          )
      );

    INSERT INTO account_groups (account_id, group_id, priority, created_at, auto_managed)
    SELECT a.id, g.id, 50, NOW(), TRUE
    FROM accounts a
    CROSS JOIN groups g
    WHERE g.id = p_group_id
      AND g.auto_assign_accounts_by_rate = TRUE
      AND g.auto_assign_max_rate IS NOT NULL
      AND g.deleted_at IS NULL
      AND a.deleted_at IS NULL
      AND a.platform = g.platform
      AND a.rate_multiplier <= g.auto_assign_max_rate
      AND a.status = 'active'
      AND a.schedulable = TRUE
    ON CONFLICT (account_id, group_id)
    DO UPDATE SET auto_managed = TRUE;
END;
$$;

CREATE OR REPLACE FUNCTION sync_dynamic_rate_groups_for_account()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    DELETE FROM account_groups ag
    USING groups g
    WHERE ag.account_id = NEW.id
      AND ag.group_id = g.id
      AND (
          (ag.auto_managed = TRUE AND NOT g.auto_assign_accounts_by_rate)
          OR (
              g.auto_assign_accounts_by_rate = TRUE
              AND (g.auto_assign_max_rate IS NULL
                   OR g.deleted_at IS NOT NULL
                   OR NEW.deleted_at IS NOT NULL
                   OR NEW.platform IS DISTINCT FROM g.platform
                   OR NEW.rate_multiplier > g.auto_assign_max_rate
                   OR NEW.status IS DISTINCT FROM 'active'
                   OR NEW.schedulable = FALSE)
          )
      );

    INSERT INTO account_groups (account_id, group_id, priority, created_at, auto_managed)
    SELECT NEW.id, g.id, 50, NOW(), TRUE
    FROM groups g
    WHERE g.auto_assign_accounts_by_rate = TRUE
      AND g.auto_assign_max_rate IS NOT NULL
      AND g.deleted_at IS NULL
      AND NEW.deleted_at IS NULL
      AND NEW.platform = g.platform
      AND NEW.rate_multiplier <= g.auto_assign_max_rate
      AND NEW.status = 'active'
      AND NEW.schedulable = TRUE
    ON CONFLICT (account_id, group_id)
    DO UPDATE SET auto_managed = TRUE;
    RETURN NEW;
END;
$$;

-- 立刻按新规则重算一遍存量绑定（否则要等下次账号/分组变更才会自愈）。
DO $$
DECLARE
    gid BIGINT;
BEGIN
    FOR gid IN
        SELECT id FROM groups
        WHERE auto_assign_accounts_by_rate = TRUE
          AND deleted_at IS NULL
    LOOP
        PERFORM sync_dynamic_rate_group(gid);
    END LOOP;
END;
$$;