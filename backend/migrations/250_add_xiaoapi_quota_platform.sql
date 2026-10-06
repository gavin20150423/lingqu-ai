-- 把本地视频平台 xiaoapi 补进平台白名单，修「合并上游时被漏掉」造成的不一致。
--
-- 背景：service.AllowedQuotaPlatforms 一直含 xiaoapi（本地扩展），但
--   ent/schema/user_platform_quota.go 的构建期校验与本库 CHECK 约束都不含它
--   （237/238/241 的平台列表照抄的是各自当时的列表，从未包含 xiaoapi）。
-- 后果：前端平台配额弹窗会给出 xiaoapi 一行，但保存时 ent 校验先拒、即使绕过也会撞 DB CHECK。
--
-- 1. user_platform_quotas.platform CHECK
-- 2. composite_model_routes.target_platform CHECK
--
-- 与 241 同型：DROP ... IF EXISTS 后重建**超集**约束，存量行瞬时校验通过。
-- 新列表 = 241 的 11 平台 + xiaoapi = service.AllowedQuotaPlatforms 的 12 平台。

ALTER TABLE user_platform_quotas
    DROP CONSTRAINT IF EXISTS user_platform_quotas_platform_check;

ALTER TABLE user_platform_quotas
    ADD CONSTRAINT user_platform_quotas_platform_check
    CHECK (platform IN ('anthropic', 'openai', 'gemini', 'antigravity', 'grok',
                        'kimi', 'zhipu', 'deepseek', 'xiaoapi', 'minimax', 'opencode_go', 'typesafe'));

ALTER TABLE composite_model_routes
    DROP CONSTRAINT IF EXISTS composite_model_routes_target_platform_check;

ALTER TABLE composite_model_routes
    ADD CONSTRAINT composite_model_routes_target_platform_check
    CHECK (target_platform IN ('anthropic', 'openai', 'gemini', 'antigravity', 'grok',
                               'kimi', 'zhipu', 'deepseek', 'xiaoapi', 'minimax', 'opencode_go', 'typesafe'));
