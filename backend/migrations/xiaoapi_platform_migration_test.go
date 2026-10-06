package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestXiaoAPIPlatformMigration 锁定 250 号迁移把 xiaoapi 补进两个平台 CHECK 约束，
// 且新列表是 241（typesafe，11 平台）的严格超集。
func TestXiaoAPIPlatformMigration(t *testing.T) {
	content, err := FS.ReadFile("250_add_xiaoapi_quota_platform.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "DROP CONSTRAINT IF EXISTS user_platform_quotas_platform_check")
	require.Contains(t, sql, "DROP CONSTRAINT IF EXISTS composite_model_routes_target_platform_check")
	require.Contains(t, sql,
		"CHECK (platform IN ('anthropic', 'openai', 'gemini', 'antigravity', 'grok', 'kimi', 'zhipu', 'deepseek', 'xiaoapi', 'minimax', 'opencode_go', 'typesafe'))")
	require.Contains(t, sql,
		"CHECK (target_platform IN ('anthropic', 'openai', 'gemini', 'antigravity', 'grok', 'kimi', 'zhipu', 'deepseek', 'xiaoapi', 'minimax', 'opencode_go', 'typesafe'))")
}
