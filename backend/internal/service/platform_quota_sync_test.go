package service

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAllowedQuotaPlatformsMatchEntSchemaValidator 是一条**同步守卫**测试。
//
// 背景：平台白名单有三处副本 ——
//   1. service.AllowedQuotaPlatforms（单一权威源）
//   2. ent/schema/user_platform_quota.go 的构建期 Validate（hand-written schema）
//   3. 数据库 CHECK 约束（由迁移维护）
//
// 2026-10-06 实际出过事故：本地视频平台 xiaoapi 只在 (1) 里，导致前端配额弹窗能选、
// 保存时被 ent 校验拒绝。三处之间的漂移不会让任何既有测试变红，所以补这条守卫。
//
// 这里只把 (1) 与 (2) 对齐（(3) 由 migrations 包的迁移测试覆盖）。
// 用读源码文本的方式断言：Validate 的 switch 里每个平台字面量必须出现，
// 比反射取 validators 更直接，也与仓库既有做法（读自身源码断言 SQL）一致。
func TestAllowedQuotaPlatformsMatchEntSchemaValidator(t *testing.T) {
	const schemaPath = "../../ent/schema/user_platform_quota.go"

	src, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "读取 ent schema 失败：%s", schemaPath)
	content := string(src)

	require.NotEmpty(t, AllowedQuotaPlatforms, "AllowedQuotaPlatforms 不应为空")

	for _, platform := range AllowedQuotaPlatforms {
		require.Containsf(t, content, `"`+platform+`"`,
			"ent/schema/user_platform_quota.go 的 Validate 缺少平台 %q —— "+
				"service.AllowedQuotaPlatforms 与 ent 构建期校验已漂移，会导致该平台配额保存失败。", platform)
	}
}
