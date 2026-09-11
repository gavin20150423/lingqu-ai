-- 首发订阅商品目录：GPT / CCMax / Kiro。
--
-- price 和 entitlements 均以 USD 计价：
--   - price 是用户支付的基础价格；
--   - entitlements 是支付成功后发放的独立模型权益余额；
--   - CNY 网关由 SUBSCRIPTION_USD_TO_CNY_RATE 统一换算。
--
-- 只创建不存在的分组和套餐，不修改管理员已有的同名配置。

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM groups
        WHERE name = 'GPT 订阅' AND deleted_at IS NULL
    ) THEN
        INSERT INTO groups (
            name, description, rate_multiplier, status, platform,
            subscription_type, default_validity_days, sort_order
        )
        VALUES (
            'GPT 订阅',
            'GPT / Codex 模型订阅，按独立 GPT 权益余额计费。',
            1.0,
            'active',
            'openai',
            'subscription',
            30,
            10
        );
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM groups
        WHERE name = 'CCMax 订阅' AND deleted_at IS NULL
    ) THEN
        INSERT INTO groups (
            name, description, rate_multiplier, status, platform,
            subscription_type, default_validity_days, claude_code_only, sort_order
        )
        VALUES (
            'CCMax 订阅',
            'Claude Code Max 订阅，按独立 CCMax 权益余额计费。',
            1.0,
            'active',
            'anthropic',
            'subscription',
            30,
            TRUE,
            20
        );
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM groups
        WHERE name = 'Kiro 订阅' AND deleted_at IS NULL
    ) THEN
        INSERT INTO groups (
            name, description, rate_multiplier, status, platform,
            subscription_type, default_validity_days, sort_order
        )
        VALUES (
            'Kiro 订阅',
            'Kiro 模型订阅，按独立 Kiro 权益余额计费。',
            1.0,
            'active',
            'antigravity',
            'subscription',
            30,
            30
        );
    END IF;
END $$;

WITH catalog (
    group_name, plan_name, description, price, original_price,
    sort_order, features, entitlements
) AS (
    VALUES
        ('GPT 订阅', '入门', '适合轻量体验与日常问答。', 5.00, 6.00, 10,
         concat('GPT / Codex 模型', chr(10), '独立 GPT 权益 $6', chr(10), '适合轻量使用'),
         '{"gpt": 6}'::jsonb),
        ('GPT 订阅', '标准', '平衡价格与调用量的主力档位。', 12.00, 15.00, 20,
         concat('GPT / Codex 模型', chr(10), '独立 GPT 权益 $15', chr(10), '适合日常开发'),
         '{"gpt": 15}'::jsonb),
        ('GPT 订阅', '专业', '面向高频开发和长上下文任务。', 24.00, 32.00, 30,
         concat('GPT / Codex 模型', chr(10), '独立 GPT 权益 $32', chr(10), '适合高频开发'),
         '{"gpt": 32}'::jsonb),
        ('GPT 订阅', '旗舰', '为重度调用和大型项目准备的高额度档位。', 48.00, 72.00, 40,
         concat('GPT / Codex 模型', chr(10), '独立 GPT 权益 $72', chr(10), '适合重度调用'),
         '{"gpt": 72}'::jsonb),

        ('CCMax 订阅', 'Max 5x', 'Claude Code 5x 级别的入门订阅档位。', 10.00, 12.00, 10,
         concat('Claude Code 专用', chr(10), '独立 CCMax 权益 $12', chr(10), '5x 使用档位'),
         '{"ccmax": 12}'::jsonb),
        ('CCMax 订阅', 'Max 5x Pro', 'Claude Code 5x 级别的高额度档位。', 20.00, 26.00, 20,
         concat('Claude Code 专用', chr(10), '独立 CCMax 权益 $26', chr(10), '5x 使用档位'),
         '{"ccmax": 26}'::jsonb),
        ('CCMax 订阅', 'Max 20x', 'Claude Code 20x 级别的主力档位。', 35.00, 48.00, 30,
         concat('Claude Code 专用', chr(10), '独立 CCMax 权益 $48', chr(10), '20x 使用档位'),
         '{"ccmax": 48}'::jsonb),
        ('CCMax 订阅', 'Max 20x Pro', '面向重度 Claude Code 用户的高额度档位。', 60.00, 90.00, 40,
         concat('Claude Code 专用', chr(10), '独立 CCMax 权益 $90', chr(10), '20x 使用档位'),
         '{"ccmax": 90}'::jsonb),

        ('Kiro 订阅', 'Starter', '适合轻量编程辅助与功能体验。', 8.00, 10.00, 10,
         concat('Kiro 模型', chr(10), '独立 Kiro 权益 $10', chr(10), '适合轻量开发'),
         '{"kiro": 10}'::jsonb),
        ('Kiro 订阅', 'Standard', '适合日常编码和持续迭代。', 15.00, 20.00, 20,
         concat('Kiro 模型', chr(10), '独立 Kiro 权益 $20', chr(10), '适合日常开发'),
         '{"kiro": 20}'::jsonb),
        ('Kiro 订阅', 'Pro', '面向高频开发和多轮 Agent 任务。', 30.00, 45.00, 30,
         concat('Kiro 模型', chr(10), '独立 Kiro 权益 $45', chr(10), '适合高频开发'),
         '{"kiro": 45}'::jsonb),
        ('Kiro 订阅', 'Max', '为重度 Agent 使用和大型项目准备。', 60.00, 100.00, 40,
         concat('Kiro 模型', chr(10), '独立 Kiro 权益 $100', chr(10), '适合重度调用'),
         '{"kiro": 100}'::jsonb)
)
INSERT INTO subscription_plans (
    group_id, name, description, price, original_price, currency,
    validity_days, validity_unit, features, entitlements, product_name,
    for_sale, sort_order
)
SELECT
    g.id,
    c.plan_name,
    c.description,
    c.price,
    c.original_price,
    'USD',
    30,
    'days',
    c.features,
    c.entitlements,
    c.group_name,
    TRUE,
    c.sort_order
FROM catalog c
JOIN groups g
  ON g.name = c.group_name
 AND g.deleted_at IS NULL
 AND g.subscription_type = 'subscription'
WHERE NOT EXISTS (
    SELECT 1
    FROM subscription_plans p
    WHERE p.group_id = g.id
      AND p.name = c.plan_name
);
