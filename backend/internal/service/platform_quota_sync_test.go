package service

import (
	"os"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestAllowedQuotaPlatformsMatchEntSchemaValidator 是一条**同步守卫**测试。
//
// 背景：平台白名单历史上曾有三处副本 ——
//   1. service.AllowedQuotaPlatforms（配额接口可见的平台）
//   2. ent/schema/user_platform_quota.go 的构建期 Validate（hand-written schema）
//   3. 数据库 CHECK 约束（由迁移维护）
//
// 2026-10-06 实际出过事故：本地视频平台 xiaoapi 只在 (1) 里，导致前端配额弹窗能选、
// 保存时被 ent 校验拒绝。三处之间的漂移不会让任何既有测试变红，所以补这条守卫。
//
// 平台清单收敛为单一数据源（domain/platforms.go）之后，副本结构已改变：
// 数据库 CHECK 约束被上游迁移 242 显式 DROP，(3) 已不存在；ent 的 Validate 也不再
// 逐平台硬编码，而是委派给 domain.IsConcretePlatform。因此这里断言两件事：
//   - ent 构建期校验确实走平台清单（不能退回硬编码字面量，否则又会漂移）；
//   - service 侧的平台集合与平台清单逐一相同（新增平台只登记清单即可）。
func TestAllowedQuotaPlatformsMatchEntSchemaValidator(t *testing.T) {
	const schemaPath = "../../ent/schema/user_platform_quota.go"

	src, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "读取 ent schema 失败：%s", schemaPath)
	content := string(src)

	require.NotEmpty(t, AllowedQuotaPlatforms, "AllowedQuotaPlatforms 不应为空")

	require.Contains(t, content, "domain.IsConcretePlatform",
		"ent/schema/user_platform_quota.go 的平台校验必须委派给 domain.IsConcretePlatform（平台清单）；"+
			"改回逐平台硬编码字面量就会重新引入与平台清单漂移的风险。")

	// 与平台清单逐一对齐：任何一边新增/删除平台而另一边没跟上，这里立刻变红。
	require.ElementsMatch(t, domain.ConcretePlatformIDs(), AllowedQuotaPlatforms,
		"service.AllowedQuotaPlatforms 必须与平台清单（domain.ConcretePlatformIDs）完全一致；"+
			"新增平台只需要登记 domain/platforms.go 的 platformList。")

	// 具体平台必须都在校验器放行的集合内 —— 这是当年 xiaoapi 事故的直接复现点。
	for _, platform := range AllowedQuotaPlatforms {
		require.True(t, domain.IsConcretePlatform(platform),
			"平台 %q 出现在 AllowedQuotaPlatforms 中，但 domain.IsConcretePlatform 不放行它，"+
				"会导致该平台配额保存失败。", platform)
	}
}
