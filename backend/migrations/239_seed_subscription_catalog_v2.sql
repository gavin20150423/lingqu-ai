-- 239: 正式用户订阅目录（8 个产品系列 × 5 个按月档位）。
--
-- 这里的 groups 只是订阅商品背后的资源路由组，用户侧商品始终来自
-- subscription_plans；不要把每个档位再建成一个普通分组。
--
-- 文本套餐的 plan quota 使用上游原始 USD 成本作为扣减单位，资源组倍率固定为
-- 1x；plan.price 是用户实际支付价，因此可同时满足「日/周/月额度」和每个系列
-- 独立的用户单价。图片套餐同样以资源组的基础图片单价换算内部 quota，features
-- 明确展示真实张数和用户单价。
--
-- 迁移只管理本文件创建的“订阅资源”名称，重复执行会修复半完成的目录并补齐
-- 缺失套餐；不会修改现有普通分组、账号、用户余额或已购买的订阅。

DROP TABLE IF EXISTS _subscription_catalog_groups;
DROP TABLE IF EXISTS _subscription_catalog_plans;

CREATE TEMP TABLE _subscription_catalog_groups (
    product_key              TEXT PRIMARY KEY,
    target_name              TEXT NOT NULL,
    source_name              TEXT NOT NULL,
    fallback_source_id       BIGINT NOT NULL,
    target_platform          TEXT NOT NULL,
    target_image             BOOLEAN NOT NULL,
    target_image_price       NUMERIC(20,8),
    target_claude_code_only  BOOLEAN NOT NULL,
    target_allowlist         JSONB NOT NULL,
    target_sort_order        INTEGER NOT NULL,
    source_group_id          BIGINT,
    target_group_id          BIGINT
);

INSERT INTO _subscription_catalog_groups (
    product_key, target_name, source_name, fallback_source_id,
    target_platform, target_image, target_image_price,
    target_claude_code_only, target_allowlist, target_sort_order
)
VALUES
    (
        'ccmax', 'CCMax Claude 订阅', '下游-ccmax-claude', 25,
        'anthropic', FALSE, NULL, FALSE,
        '{"enabled":true,"models":["claude-fable-5","claude-opus-4-5-20251101","claude-opus-4-6","claude-opus-4-7","claude-opus-4-8","claude-sonnet-5","claude-sonnet-4-6","claude-sonnet-4-5-20250929","claude-haiku-4-5-20251001"]}'::jsonb,
        10
    ),
    (
        'kiro', 'Kiro Claude 订阅', '下游-kiro-claude-90%缓', 23,
        'anthropic', FALSE, NULL, FALSE,
        '{"enabled":true,"models":["claude-fable-5","claude-opus-4-5-20251101","claude-opus-4-6","claude-opus-4-7","claude-opus-4-8","claude-sonnet-5","claude-sonnet-4-6","claude-sonnet-4-5-20250929","claude-haiku-4-5-20251001"]}'::jsonb,
        20
    ),
    (
        'gpt', 'GPT 订阅', 'openai-plus', 3,
        'openai', FALSE, NULL, FALSE,
        '{"enabled":true,"models":["gpt-5.6-sol","gpt-5.6","gpt-5.6-terra","gpt-5.6-luna","gpt-5.5","gpt-5.4","gpt-5.4-mini","gpt-5.3-codex","gpt-5.3-codex-spark","gpt-5.2","gpt-5.2-pro","codex-auto-review","gpt-4o-audio-preview","gpt-4o-realtime-preview"]}'::jsonb,
        30
    ),
    (
        'gemini', 'Gemini 订阅', 'Gemini', 72,
        'gemini', FALSE, NULL, FALSE,
        '{"enabled":true,"models":["gemini-2.0-flash","gemini-2.5-flash","gemini-2.5-pro","gemini-3.5-flash","gemini-3-flash-preview","gemini-3-pro-preview","gemini-3.1-pro-preview","gemini-3.1-flash"]}'::jsonb,
        40
    ),
    (
        'domestic', '国模订阅', '国模 gm x0.35', 81,
        'openai', FALSE, NULL, FALSE,
        '{"enabled":true,"models":["glm-5.3","glm-5.3-flash","deepseek-v4-flash","deepseek-v4-pro","DeepSeek-V4.1-Flash","kimi-k3","MiniMax-M3","qwen3.8-max","qwen3.8-flash"]}'::jsonb,
        50
    ),
    (
        'grok', 'Grok 订阅', 'Grok', 28,
        'grok', FALSE, NULL, FALSE,
        '{"enabled":true,"models":["grok-4.6","grok-4.6-latest","grok-4.5","grok-4.5-latest","grok-4.3","grok-4.20-0309-reasoning","grok-4.20-0309-non-reasoning","grok-4.20-multi-agent-0309","grok-build-0.1","grok-composer-2.5-fast","grok-3-mini","grok-3-mini-fast"]}'::jsonb,
        60
    ),
    (
        'gpt_image', 'GPT Image 2.5 订阅', 'jp-image', 29,
        'openai', TRUE, 0.08, FALSE,
        '{"enabled":true,"models":["gpt-image-2.5-flare","gpt-image-2.5-sunburst","gpt-image-2.5-flare-2026-09-08","gpt-image-2.5-sunburst-2026-09-08"]}'::jsonb,
        70
    ),
    (
        'banana', '香蕉生图订阅', 'jp-gemini-image', 36,
        'gemini', TRUE, 0.15, FALSE,
        '{"enabled":true,"models":["gemini-3.1-flash-image","gemini-2.5-flash-image","gemini-3-pro-image","gemini-3-pro-image-preview"]}'::jsonb,
        80
    );

-- 优先按名称找源分组，只有名称在某个灾备数据集不存在时才使用当前实例的
-- 已知 ID；这样同一份迁移在恢复库和当前线上库都能复用。
UPDATE _subscription_catalog_groups c
SET source_group_id = (
    SELECT g.id
    FROM groups g
    WHERE g.name = c.source_name
      AND g.deleted_at IS NULL
    ORDER BY g.id
    LIMIT 1
);

UPDATE _subscription_catalog_groups c
SET source_group_id = c.fallback_source_id
WHERE c.source_group_id IS NULL
  AND EXISTS (
      SELECT 1 FROM groups g
      WHERE g.id = c.fallback_source_id
        AND g.deleted_at IS NULL
  );

DO $$
DECLARE
    missing_product TEXT;
BEGIN
    SELECT product_key INTO missing_product
    FROM _subscription_catalog_groups
    WHERE source_group_id IS NULL
    LIMIT 1;
    IF missing_product IS NOT NULL THEN
        RAISE EXCEPTION 'subscription catalog source group is missing: %', missing_product;
    END IF;
END $$;

DO $$
DECLARE
    item RECORD;
    source_id BIGINT;
    new_id BIGINT;
BEGIN
    FOR item IN SELECT * FROM _subscription_catalog_groups ORDER BY target_sort_order LOOP
        source_id := item.source_group_id;

        SELECT g.id INTO new_id
        FROM groups g
        WHERE g.name = item.target_name
          AND g.deleted_at IS NULL
        ORDER BY g.id
        LIMIT 1;

        IF new_id IS NULL THEN
            INSERT INTO groups (
                name, description, rate_multiplier, is_exclusive, status,
                platform, subscription_type, default_validity_days,
                allow_image_generation, allow_batch_image_generation,
                image_rate_independent, image_rate_multiplier,
                image_price_1k, image_price_2k, image_price_4k,
                claude_code_only, supported_model_scopes, model_allowlist,
                model_pricing, scope, sort_order, long_context_pricing_enabled
            )
            SELECT
                item.target_name,
                format('用户订阅资源组：%s。账号路由由源分组复制，额度由套餐独立快照控制。', item.target_name),
                1.0,
                FALSE,
                'active',
                item.target_platform,
                'subscription',
                30,
                item.target_image,
                FALSE,
                item.target_image,
                1.0,
                CASE WHEN item.target_image THEN item.target_image_price ELSE NULL END,
                CASE WHEN item.target_image THEN item.target_image_price ELSE NULL END,
                CASE WHEN item.target_image THEN item.target_image_price ELSE NULL END,
                item.target_claude_code_only,
                COALESCE(src.supported_model_scopes, '["claude","gemini_text","gemini_image"]'::jsonb),
                item.target_allowlist,
                src.model_pricing,
                'public',
                item.target_sort_order,
                COALESCE(src.long_context_pricing_enabled, TRUE)
            FROM groups src
            WHERE src.id = source_id
            RETURNING id INTO new_id;
        ELSE
            -- 这些名称由本迁移独占；修复重复执行或上次中断留下的半成品，
            -- 不触碰普通分组和任何用户数据。
            UPDATE groups
            SET description = format('用户订阅资源组：%s。账号路由由源分组复制，额度由套餐独立快照控制。', item.target_name),
                rate_multiplier = 1.0,
                status = 'active',
                platform = item.target_platform,
                subscription_type = 'subscription',
                default_validity_days = 30,
                allow_image_generation = item.target_image,
                allow_batch_image_generation = FALSE,
                image_rate_independent = item.target_image,
                image_rate_multiplier = 1.0,
                image_price_1k = CASE WHEN item.target_image THEN item.target_image_price ELSE NULL END,
                image_price_2k = CASE WHEN item.target_image THEN item.target_image_price ELSE NULL END,
                image_price_4k = CASE WHEN item.target_image THEN item.target_image_price ELSE NULL END,
                claude_code_only = item.target_claude_code_only,
                model_allowlist = item.target_allowlist,
                scope = 'public',
                sort_order = item.target_sort_order,
                updated_at = NOW()
            WHERE id = new_id;
        END IF;

        IF new_id IS NULL THEN
            RAISE EXCEPTION 'failed to create subscription resource group: %', item.target_name;
        END IF;

        UPDATE _subscription_catalog_groups
        SET target_group_id = new_id
        WHERE product_key = item.product_key;

        INSERT INTO account_groups (account_id, group_id, priority)
        SELECT ag.account_id, new_id, ag.priority
        FROM account_groups ag
        WHERE ag.group_id = source_id
        ON CONFLICT (account_id, group_id) DO NOTHING;

        IF NOT EXISTS (SELECT 1 FROM account_groups WHERE group_id = new_id) THEN
            RAISE EXCEPTION 'subscription resource group has no accounts: %', item.target_name;
        END IF;
    END LOOP;
END $$;

CREATE TEMP TABLE _subscription_catalog_plans (
    product_key       TEXT NOT NULL,
    tier_name         TEXT NOT NULL,
    description       TEXT NOT NULL,
    price             NUMERIC(20,2) NOT NULL,
    original_price    NUMERIC(20,2),
    daily_limit       NUMERIC(20,8) NOT NULL,
    weekly_limit      NUMERIC(20,8) NOT NULL,
    monthly_limit     NUMERIC(20,8) NOT NULL,
    features          TEXT NOT NULL,
    tier_sort_order   INTEGER NOT NULL,
    PRIMARY KEY (product_key, tier_name)
);

-- 文本套餐额度沿用参考方案：周额度约为日额度 3.5 倍，月额度显著高于
-- 周额度，避免用户在不到两周时就耗尽月卡。
INSERT INTO _subscription_catalog_plans (
    product_key, tier_name, description, price, original_price,
    daily_limit, weekly_limit, monthly_limit, features, tier_sort_order
)
VALUES
    ('ccmax', 'Basic', '适合日常 Claude Code 轻量使用。', 270.00, NULL, 30, 105, 300, 'Claude / Claude Code 模型\n用户价 $0.90 / $1 原始调用成本\n日 $30 · 周 $105 · 月 $300', 10),
    ('ccmax', 'Plus', '适合持续开发与多轮对话。', 576.00, NULL, 65, 225, 640, 'Claude / Claude Code 模型\n用户价 $0.90 / $1 原始调用成本\n日 $65 · 周 $225 · 月 $640', 20),
    ('ccmax', 'Standard', '适合个人开发者的主力档位。', 1350.00, NULL, 120, 420, 1500, 'Claude / Claude Code 模型\n用户价 $0.90 / $1 原始调用成本\n日 $120 · 周 $420 · 月 $1500', 30),
    ('ccmax', 'Pro', '适合高频开发和长上下文任务。', 2475.00, NULL, 200, 700, 2750, 'Claude / Claude Code 模型\n用户价 $0.90 / $1 原始调用成本\n日 $200 · 周 $700 · 月 $2750', 40),
    ('ccmax', 'Ultra', '适合重度团队与持续 Agent 任务。', 5400.00, NULL, 400, 1400, 6000, 'Claude / Claude Code 模型\n用户价 $0.90 / $1 原始调用成本\n日 $400 · 周 $1400 · 月 $6000', 50),

    ('kiro', 'Basic', '适合轻量 Kiro Claude 编程辅助。', 51.00, NULL, 30, 105, 300, 'Kiro Claude 模型\n用户价 $0.17 / $1 原始调用成本\n日 $30 · 周 $105 · 月 $300', 10),
    ('kiro', 'Plus', '适合日常编码与持续迭代。', 108.80, NULL, 65, 225, 640, 'Kiro Claude 模型\n用户价 $0.17 / $1 原始调用成本\n日 $65 · 周 $225 · 月 $640', 20),
    ('kiro', 'Standard', '适合个人开发者的稳定主力档位。', 255.00, NULL, 120, 420, 1500, 'Kiro Claude 模型\n用户价 $0.17 / $1 原始调用成本\n日 $120 · 周 $420 · 月 $1500', 30),
    ('kiro', 'Pro', '适合高频 Agent 与大型项目。', 467.50, NULL, 200, 700, 2750, 'Kiro Claude 模型\n用户价 $0.17 / $1 原始调用成本\n日 $200 · 周 $700 · 月 $2750', 40),
    ('kiro', 'Ultra', '适合重度 Agent 和团队协作。', 1020.00, NULL, 400, 1400, 6000, 'Kiro Claude 模型\n用户价 $0.17 / $1 原始调用成本\n日 $400 · 周 $1400 · 月 $6000', 50),

    ('gpt', 'Basic', '适合 GPT 日常问答与轻量开发。', 54.00, NULL, 30, 105, 300, 'GPT / Codex 模型\n用户价 $0.18 / $1 原始调用成本\n日 $30 · 周 $105 · 月 $300', 10),
    ('gpt', 'Plus', '适合日常开发与稳定调用。', 115.20, NULL, 65, 225, 640, 'GPT / Codex 模型\n用户价 $0.18 / $1 原始调用成本\n日 $65 · 周 $225 · 月 $640', 20),
    ('gpt', 'Standard', '适合个人开发者的主力档位。', 270.00, NULL, 120, 420, 1500, 'GPT / Codex 模型\n用户价 $0.18 / $1 原始调用成本\n日 $120 · 周 $420 · 月 $1500', 30),
    ('gpt', 'Pro', '适合高频推理和大型项目。', 495.00, NULL, 200, 700, 2750, 'GPT / Codex 模型\n用户价 $0.18 / $1 原始调用成本\n日 $200 · 周 $700 · 月 $2750', 40),
    ('gpt', 'Ultra', '高额度 GPT 旗舰档位，最高档 9 折。', 972.00, 1080.00, 400, 1400, 6000, 'GPT / Codex 模型\n用户价 $0.18 / $1 原始调用成本\n最高档单价 9 折\n日 $400 · 周 $1400 · 月 $6000', 50),

    ('gemini', 'Basic', '适合 Gemini 轻量体验与日常问答。', 120.00, NULL, 30, 105, 300, 'Gemini 模型\n用户价 $0.40 / $1 原始调用成本\n日 $30 · 周 $105 · 月 $300', 10),
    ('gemini', 'Plus', '适合日常开发与多模态任务。', 256.00, NULL, 65, 225, 640, 'Gemini 模型\n用户价 $0.40 / $1 原始调用成本\n日 $65 · 周 $225 · 月 $640', 20),
    ('gemini', 'Standard', '适合个人开发者的主力档位。', 600.00, NULL, 120, 420, 1500, 'Gemini 模型\n用户价 $0.40 / $1 原始调用成本\n日 $120 · 周 $420 · 月 $1500', 30),
    ('gemini', 'Pro', '适合高频多模态与长上下文任务。', 1100.00, NULL, 200, 700, 2750, 'Gemini 模型\n用户价 $0.40 / $1 原始调用成本\n日 $200 · 周 $700 · 月 $2750', 40),
    ('gemini', 'Ultra', '高额度 Gemini 旗舰档位，最高档 9 折。', 2160.00, 2400.00, 400, 1400, 6000, 'Gemini 模型\n用户价 $0.40 / $1 原始调用成本\n最高档单价 9 折\n日 $400 · 周 $1400 · 月 $6000', 50),

    ('domestic', 'Basic', '适合国产模型轻量调用。', 90.00, NULL, 30, 105, 300, 'GLM / DeepSeek / Kimi / MiniMax / Qwen\n用户价为上游成本 3 折\n日 $30 · 周 $105 · 月 $300', 10),
    ('domestic', 'Plus', '适合日常中文问答与开发。', 192.00, NULL, 65, 225, 640, 'GLM / DeepSeek / Kimi / MiniMax / Qwen\n用户价为上游成本 3 折\n日 $65 · 周 $225 · 月 $640', 20),
    ('domestic', 'Standard', '适合个人开发者的国产模型主力档位。', 450.00, NULL, 120, 420, 1500, 'GLM / DeepSeek / Kimi / MiniMax / Qwen\n用户价为上游成本 3 折\n日 $120 · 周 $420 · 月 $1500', 30),
    ('domestic', 'Pro', '适合高频国产模型与 Agent 任务。', 825.00, NULL, 200, 700, 2750, 'GLM / DeepSeek / Kimi / MiniMax / Qwen\n用户价为上游成本 3 折\n日 $200 · 周 $700 · 月 $2750', 40),
    ('domestic', 'Ultra', '高额度国产模型旗舰档位，最高档 9 折。', 1620.00, 1800.00, 400, 1400, 6000, 'GLM / DeepSeek / Kimi / MiniMax / Qwen\n用户价为上游成本 3 折\n最高档单价 9 折\n日 $400 · 周 $1400 · 月 $6000', 50),

    ('grok', 'Basic', '适合 Grok 轻量问答与探索。', 60.00, NULL, 30, 105, 300, 'Grok 文本模型\n用户价 $0.20 / $1 原始调用成本\n日 $30 · 周 $105 · 月 $300', 10),
    ('grok', 'Plus', '适合日常开发与持续对话。', 128.00, NULL, 65, 225, 640, 'Grok 文本模型\n用户价 $0.20 / $1 原始调用成本\n日 $65 · 周 $225 · 月 $640', 20),
    ('grok', 'Standard', '适合个人开发者的主力档位。', 300.00, NULL, 120, 420, 1500, 'Grok 文本模型\n用户价 $0.20 / $1 原始调用成本\n日 $120 · 周 $420 · 月 $1500', 30),
    ('grok', 'Pro', '适合高频推理和 Agent 任务。', 550.00, NULL, 200, 700, 2750, 'Grok 文本模型\n用户价 $0.20 / $1 原始调用成本\n日 $200 · 周 $700 · 月 $2750', 40),
    ('grok', 'Ultra', '高额度 Grok 旗舰档位，最高档 9 折。', 1080.00, 1200.00, 400, 1400, 6000, 'Grok 文本模型\n用户价 $0.20 / $1 原始调用成本\n最高档单价 9 折\n日 $400 · 周 $1400 · 月 $6000', 50),

    ('gpt_image', 'Basic', '适合偶尔使用 GPT Image 2.5。', 40.00, NULL, 4, 14, 40, 'GPT Image 2.5 两个模型\n月 500 张，单价 $0.08 / 张\n日 50 张 · 周 175 张 · 月 500 张', 10),
    ('gpt_image', 'Plus', '适合稳定的图片生成需求。', 120.00, NULL, 12, 42, 120, 'GPT Image 2.5 两个模型\n月 1500 张，单价 $0.08 / 张\n日 150 张 · 周 525 张 · 月 1500 张', 20),
    ('gpt_image', 'Standard', '适合内容创作与日常批量生图。', 240.00, NULL, 24, 84, 240, 'GPT Image 2.5 两个模型\n月 3000 张，单价 $0.08 / 张\n日 300 张 · 周 1050 张 · 月 3000 张', 30),
    ('gpt_image', 'Pro', '超过 3000 张后享受阶梯单价。', 480.00, NULL, 64, 224, 640, 'GPT Image 2.5 两个模型\n月 8000 张，单价 $0.06 / 张\n日 800 张 · 周 2800 张 · 月 8000 张', 40),
    ('gpt_image', 'Ultra', '重度图片生产与团队使用。', 750.00, NULL, 120, 420, 1200, 'GPT Image 2.5 两个模型\n月 15000 张，单价 $0.05 / 张\n日 1500 张 · 周 5250 张 · 月 15000 张', 50),

    ('banana', 'Basic', '适合偶尔使用香蕉生图。', 75.00, NULL, 7.5, 26.25, 75, 'Nano Banana 生图模型\n月 500 张，单价 $0.15 / 张\n日 50 张 · 周 175 张 · 月 500 张', 10),
    ('banana', 'Plus', '适合稳定的多模态创作。', 225.00, NULL, 22.5, 78.75, 225, 'Nano Banana 生图模型\n月 1500 张，单价 $0.15 / 张\n日 150 张 · 周 525 张 · 月 1500 张', 20),
    ('banana', 'Standard', '适合日常内容创作与批量生图。', 390.00, NULL, 45, 157.5, 450, 'Nano Banana 生图模型\n月 3000 张，单价 $0.13 / 张\n日 300 张 · 周 1050 张 · 月 3000 张', 30),
    ('banana', 'Pro', '超过 5000 张后享受阶梯单价。', 600.00, NULL, 75, 262.5, 750, 'Nano Banana 生图模型\n月 5000 张，单价 $0.12 / 张\n日 500 张 · 周 1750 张 · 月 5000 张', 40),
    ('banana', 'Ultra', '重度图片生产与团队使用。', 1000.00, NULL, 150, 525, 1500, 'Nano Banana 生图模型\n月 10000 张，单价 $0.10 / 张\n日 1000 张 · 周 3500 张 · 月 10000 张', 50);

-- 套餐名称在每个资源组内唯一。对本迁移拥有的目标组执行覆盖式修复，
-- 这样即使上一次发布只写入了部分计划，重新跑迁移也会得到完整配置。
UPDATE subscription_plans p
SET description = c.description,
    price = c.price,
    original_price = c.original_price,
    validity_days = 30,
    validity_unit = 'days',
    features = c.features,
    product_name = g.target_name,
    for_sale = TRUE,
    sort_order = g.target_sort_order * 100 + c.tier_sort_order,
    currency = 'USD',
    entitlements = '{}'::jsonb,
    daily_limit_usd = c.daily_limit,
    weekly_limit_usd = c.weekly_limit,
    monthly_limit_usd = c.monthly_limit,
    updated_at = NOW()
FROM _subscription_catalog_plans c
JOIN _subscription_catalog_groups g ON g.product_key = c.product_key
WHERE p.group_id = g.target_group_id
  AND p.name = c.tier_name;

INSERT INTO subscription_plans (
    group_id, name, description, price, original_price,
    validity_days, validity_unit, features, product_name,
    for_sale, sort_order, currency, entitlements,
    daily_limit_usd, weekly_limit_usd, monthly_limit_usd
)
SELECT
    g.target_group_id,
    c.tier_name,
    c.description,
    c.price,
    c.original_price,
    30,
    'days',
    c.features,
    g.target_name,
    TRUE,
    g.target_sort_order * 100 + c.tier_sort_order,
    'USD',
    '{}'::jsonb,
    c.daily_limit,
    c.weekly_limit,
    c.monthly_limit
FROM _subscription_catalog_plans c
JOIN _subscription_catalog_groups g ON g.product_key = c.product_key
WHERE NOT EXISTS (
    SELECT 1
    FROM subscription_plans p
    WHERE p.group_id = g.target_group_id
      AND p.name = c.tier_name
);

-- 旧版本曾在这几个同名资源组里写入过其他套餐名称。保留历史订单和
-- 已购买订阅所需的数据，但将非本目录套餐下架，确保用户侧每个系列只
-- 展示五个统一档位。
UPDATE subscription_plans p
SET for_sale = FALSE,
    updated_at = NOW()
FROM _subscription_catalog_groups g
WHERE p.group_id = g.target_group_id
  AND p.name NOT IN ('Basic', 'Plus', 'Standard', 'Pro', 'Ultra')
  AND p.for_sale = TRUE;

DO $$
DECLARE
    plan_count INTEGER;
    group_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO group_count
    FROM _subscription_catalog_groups
    WHERE target_group_id IS NOT NULL;
    IF group_count <> 8 THEN
        RAISE EXCEPTION 'subscription catalog expected 8 resource groups, got %', group_count;
    END IF;

    SELECT COUNT(*) INTO plan_count
    FROM subscription_plans p
    JOIN _subscription_catalog_groups g ON g.target_group_id = p.group_id
    WHERE p.for_sale = TRUE
      AND p.name IN ('Basic', 'Plus', 'Standard', 'Pro', 'Ultra');
    IF plan_count <> 40 THEN
        RAISE EXCEPTION 'subscription catalog expected 40 plans, got %', plan_count;
    END IF;
END $$;
