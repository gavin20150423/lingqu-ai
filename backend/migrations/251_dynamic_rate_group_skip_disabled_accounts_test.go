package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMigration251DynamicRateGroupSkipsDisabledAccounts 锁定「动态速率分组不得自动绑定
// 被操作员禁用的账号」这条规则。
//
// 背景（2026-10-08，与 v0.2.14-lingqu.2 的单账号强制路径修复配套）：
// sync_dynamic_rate_group / sync_dynamic_rate_groups_for_account 原本只按
// 「平台一致 + 费率不超过上限 + 未删除」筛选，不看账号状态，于是
// status <> 'active' 或 schedulable = false 的账号仍被自动绑进动态分组。
// 正常调度会过滤它们所以平时不显形，但任何"只看分组归属"的旁路都会把禁用账号捞回来调度
//（2026-10-06 由 804/kiro-95 事故定案）。
func TestMigration251DynamicRateGroupSkipsDisabledAccounts(t *testing.T) {
	content, err := FS.ReadFile("251_dynamic_rate_group_skip_disabled_accounts.sql")
	require.NoError(t, err)
	sql := string(content)

	// 两个函数都要重定义：分组维度（sync_dynamic_rate_group）与账号维度
	// （sync_dynamic_rate_groups_for_account，accounts UPDATE 触发）。
	require.Contains(t, sql, "CREATE OR REPLACE FUNCTION sync_dynamic_rate_group(p_group_id BIGINT)")
	require.Contains(t, sql, "CREATE OR REPLACE FUNCTION sync_dynamic_rate_groups_for_account()")

	// INSERT 侧：两条都必须在费率条件之后追加可用性过滤。
	require.Contains(t, sql, "AND a.rate_multiplier <= g.auto_assign_max_rate\n      AND a.status = 'active'\n      AND a.schedulable = TRUE")
	require.Contains(t, sql, "AND NEW.rate_multiplier <= g.auto_assign_max_rate\n      AND NEW.status = 'active'\n      AND NEW.schedulable = TRUE")

	// DELETE 侧：存量错误绑定要能在下次同步时被清掉，否则状态会一直撒谎。
	require.Contains(t, sql, "OR a.status IS DISTINCT FROM 'active'\n                   OR a.schedulable = FALSE)")
	require.Contains(t, sql, "OR NEW.status IS DISTINCT FROM 'active'\n                   OR NEW.schedulable = FALSE)")

	// 立刻重算存量绑定：只加 INSERT 过滤的话，存量要等下次账号/分组变更才自愈。
	require.Contains(t, sql, "PERFORM sync_dynamic_rate_group(gid);")
}

// TestMigration251DoesNotTouchManualBindings 确认清理只作用于规则自动创建的绑定：
// DELETE 的第一个分支仍然要求 auto_managed = TRUE，手工加的绑定不受影响。
func TestMigration251DoesNotTouchManualBindings(t *testing.T) {
	content, err := FS.ReadFile("251_dynamic_rate_group_skip_disabled_accounts.sql")
	require.NoError(t, err)
	sql := string(content)

	require.Contains(t, sql, "(ag.auto_managed = TRUE AND NOT g.auto_assign_accounts_by_rate)")
	require.Contains(t, sql, "(ag.auto_managed = TRUE AND NOT g.auto_assign_accounts_by_rate)")
	// 不得出现"无视 auto_managed 直接删"的写法。
	require.NotContains(t, sql, "DELETE FROM account_groups ag\n    USING groups g, accounts a\n    WHERE ag.group_id = g.id\n      AND ag.account_id = a.id\n      AND g.id = p_group_id\n      AND a.status")
}