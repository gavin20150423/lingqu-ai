package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMigration252TriggerWatchesAvailability 锁住「操作员禁用账号时绑定立即消失」。
//
// 背景（2026-10-09，线上实测）：251 给规则函数加了 status/schedulable 过滤，
// 但账号级触发器仍只监听 `UPDATE OF platform, rate_multiplier, deleted_at`，
// 于是手动禁用账号（status / schedulable）时触发器不跑，绑定残留。
// 实测账号 787 关掉 schedulable 后绑定数不变，手工调 sync_dynamic_rate_group() 才清掉。
func TestMigration252TriggerWatchesAvailability(t *testing.T) {
	content, err := FS.ReadFile("252_dynamic_rate_group_trigger_watches_availability.sql")
	require.NoError(t, err)
	sql := string(content)

	// 必须先 DROP 再重建（CREATE OR REPLACE TRIGGER 不存在）。
	require.Contains(t, sql, "DROP TRIGGER IF EXISTS trg_sync_dynamic_rate_groups_for_account ON accounts;")
	require.Contains(t, sql, "CREATE TRIGGER trg_sync_dynamic_rate_groups_for_account")

	// 监听列必须包含 availability 两项，且保留原有的三项。
	require.Contains(t, sql, "AFTER INSERT OR UPDATE OF platform, rate_multiplier, deleted_at, status, schedulable")

	// 重建后仍指向同一个函数。
	require.Contains(t, sql, "EXECUTE FUNCTION sync_dynamic_rate_groups_for_account();")
}

// TestMigration213OriginalTriggerWasNarrowerThanTheRule guards 回归：213 定义的旧触发器
// 只监听三列，正是 252 要修的那个洞。这里钉住"251/252 之后监听列不少于旧定义"。
func TestMigration213OriginalTriggerWasNarrowerThanTheRule(t *testing.T) {
	oldContent, err := FS.ReadFile("213_dynamic_rate_groups.sql")
	require.NoError(t, err)
	oldSQL := string(oldContent)
	require.Contains(t, oldSQL, "AFTER INSERT OR UPDATE OF platform, rate_multiplier, deleted_at ON")

	newContent, err := FS.ReadFile("252_dynamic_rate_group_trigger_watches_availability.sql")
	require.NoError(t, err)
	newSQL := string(newContent)

	// 旧的窄监听不应再出现在 252 之后（否则重建等于没改）。
	require.NotContains(t, newSQL, "UPDATE OF platform, rate_multiplier, deleted_at ON")
	require.Contains(t, newSQL, "deleted_at, status, schedulable")
}