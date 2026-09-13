-- 241: 为正式订阅目录补充分档说明。
--
-- 本迁移只更新 subscription_plans.description，不能修改套餐名称、售价、
-- 币种、日/周/月额度、上下游分组或支付逻辑。按产品组和档位精确匹配，
-- 并在提交前确认线上正好命中 40 个正式套餐。

CREATE TEMP TABLE _subscription_plan_description_refresh (
    group_name  TEXT NOT NULL,
    tier_name   TEXT NOT NULL,
    description TEXT NOT NULL,
    PRIMARY KEY (group_name, tier_name)
);

INSERT INTO _subscription_plan_description_refresh (group_name, tier_name, description)
VALUES
    ('CCMax Claude 订阅', 'Basic', '入门 Claude Code 档：适合轻量修复与日常开发，$0.90/$，较参考市场 $1.20/$ 约省 25%。'),
    ('CCMax Claude 订阅', 'Plus', '进阶 Claude Code 档：适合持续编码与多轮协作，$0.90/$，较参考市场价约省 25%。'),
    ('CCMax Claude 订阅', 'Standard', '主力 Claude Code 档：适合个人稳定迭代，$0.90/$，较参考市场价约省 25%。'),
    ('CCMax Claude 订阅', 'Pro', '专业 Claude Code 档：适合高频开发、重构与长上下文，$0.90/$，较参考市场价约省 25%。'),
    ('CCMax Claude 订阅', 'Ultra', '旗舰 Claude Code 档：适合团队与持续 Agent 任务，$0.90/$，较参考市场价约省 25%。'),

    ('Kiro Claude 订阅', 'Basic', '入门 Kiro Claude 档：适合轻量编程辅助，$0.17/$，较参考市场 $0.25/$ 约省 32%。'),
    ('Kiro Claude 订阅', 'Plus', '进阶 Kiro Claude 档：适合日常编码与持续迭代，$0.17/$，较参考市场价约省 32%。'),
    ('Kiro Claude 订阅', 'Standard', '主力 Kiro Claude 档：适合个人稳定开发，$0.17/$，较参考市场价约省 32%。'),
    ('Kiro Claude 订阅', 'Pro', '专业 Kiro Claude 档：适合高频 Agent 与大型项目，$0.17/$，较参考市场价约省 32%。'),
    ('Kiro Claude 订阅', 'Ultra', '旗舰 Kiro Claude 档：适合重度 Agent 与团队协作，$0.17/$，较参考市场价约省 32%。'),

    ('GPT 订阅', 'Basic', '入门 GPT / Codex 档：适合日常问答与轻量开发，$0.18/$，较参考市场 $0.31/$ 约省 42%。'),
    ('GPT 订阅', 'Plus', '进阶 GPT / Codex 档：适合持续编码与常规推理，$0.18/$，较参考市场价约省 42%。'),
    ('GPT 订阅', 'Standard', '主力 GPT / Codex 档：适合个人稳定使用，$0.18/$，较参考市场价约省 42%。'),
    ('GPT 订阅', 'Pro', '专业 GPT / Codex 档：适合高频推理与大型项目，$0.18/$，较参考市场价约省 42%。'),
    ('GPT 订阅', 'Ultra', '旗舰 GPT / Codex 档：最高档额外 9 折，约 $0.162/$，较参考市场价约省 48%。'),

    ('Gemini 订阅', 'Basic', '入门 Gemini 档：适合轻量体验与日常问答，$0.40/$，较参考市场 $0.60/$ 约省 33%。'),
    ('Gemini 订阅', 'Plus', '进阶 Gemini 档：适合日常开发与多模态任务，$0.40/$，较参考市场价约省 33%。'),
    ('Gemini 订阅', 'Standard', '主力 Gemini 档：适合个人稳定创作与分析，$0.40/$，较参考市场价约省 33%。'),
    ('Gemini 订阅', 'Pro', '专业 Gemini 档：适合高频多模态与长上下文任务，$0.40/$，较参考市场价约省 33%。'),
    ('Gemini 订阅', 'Ultra', '旗舰 Gemini 档：最高档额外 9 折，约 $0.36/$，较参考市场价约省 40%。'),

    ('国模订阅', 'Basic', '入门国模档：适合轻量中文问答，覆盖 GLM、DeepSeek、Kimi、MiniMax、Qwen，按上游成本 3 折。'),
    ('国模订阅', 'Plus', '进阶国模档：适合日常中文问答与开发，五大国产模型统一按上游成本 3 折。'),
    ('国模订阅', 'Standard', '主力国模档：适合个人稳定调用与多模型切换，按上游成本 3 折。'),
    ('国模订阅', 'Pro', '专业国模档：适合高频推理、Agent 与批量任务，按上游成本 3 折。'),
    ('国模订阅', 'Ultra', '旗舰国模档：适合团队与重度调用，基础 3 折再享 9 折，约为上游成本 2.7 折。'),

    ('Grok 订阅', 'Basic', '入门 Grok 档：适合轻量问答与实时探索，常规单价 $0.20/$。'),
    ('Grok 订阅', 'Plus', '进阶 Grok 档：适合日常对话与持续检索，常规单价 $0.20/$。'),
    ('Grok 订阅', 'Standard', '主力 Grok 档：适合个人稳定使用与内容创作，常规单价 $0.20/$。'),
    ('Grok 订阅', 'Pro', '专业 Grok 档：适合高频推理、长对话与 Agent 任务，常规单价 $0.20/$。'),
    ('Grok 订阅', 'Ultra', '旗舰 Grok 档：适合团队重度使用，最高档额外 9 折，单价 $0.18/$。'),

    ('GPT Image 2.5 订阅', 'Basic', '入门 GPT Image 双模型档：适合偶尔生图，每月 500 张，单价 $0.08/张。'),
    ('GPT Image 2.5 订阅', 'Plus', '进阶 GPT Image 双模型档：适合稳定内容创作，每月 1,500 张，单价 $0.08/张。'),
    ('GPT Image 2.5 订阅', 'Standard', '主力 GPT Image 双模型档：适合日常批量生图，每月 3,000 张，单价 $0.08/张。'),
    ('GPT Image 2.5 订阅', 'Pro', '专业 GPT Image 双模型档：每月 8,000 张，阶梯单价 $0.06/张，较基础价省 25%。'),
    ('GPT Image 2.5 订阅', 'Ultra', '旗舰 GPT Image 双模型档：每月 15,000 张，阶梯单价 $0.05/张，较基础价省 37.5%。'),

    ('香蕉生图订阅', 'Basic', '入门 nano banana 档：适合偶尔创作，每月 500 张，单价 $0.15/张。'),
    ('香蕉生图订阅', 'Plus', '进阶 nano banana 档：适合稳定多模态创作，每月 1,500 张，单价 $0.15/张。'),
    ('香蕉生图订阅', 'Standard', '主力 nano banana 档：每月 3,000 张，阶梯单价 $0.13/张，较基础价省约 13%。'),
    ('香蕉生图订阅', 'Pro', '专业 nano banana 档：每月 5,000 张，阶梯单价 $0.12/张，较基础价省 20%。'),
    ('香蕉生图订阅', 'Ultra', '旗舰 nano banana 档：每月 10,000 张，阶梯单价 $0.10/张，较基础价省约 33%。');

DO $$
DECLARE
    matched_count INTEGER;
BEGIN
    SELECT COUNT(DISTINCT p.id)
    INTO matched_count
    FROM subscription_plans p
    JOIN _subscription_plan_description_refresh d
      ON d.tier_name = p.name
     AND d.group_name = p.product_name
    JOIN groups g
      ON g.id = p.group_id
     AND g.name = d.group_name
     AND g.deleted_at IS NULL
    WHERE g.subscription_type = 'subscription';

    IF matched_count <> 40 THEN
        RAISE EXCEPTION 'subscription plan description refresh expected 40 matches, got %', matched_count;
    END IF;
END $$;

UPDATE subscription_plans p
SET description = d.description
FROM _subscription_plan_description_refresh d
JOIN groups g
  ON g.name = d.group_name
 AND g.deleted_at IS NULL
 AND g.subscription_type = 'subscription'
WHERE p.group_id = g.id
  AND p.name = d.tier_name
  AND p.product_name = d.group_name;

DO $$
DECLARE
    refreshed_count INTEGER;
BEGIN
    SELECT COUNT(DISTINCT p.id)
    INTO refreshed_count
    FROM subscription_plans p
    JOIN _subscription_plan_description_refresh d
      ON d.tier_name = p.name
     AND d.group_name = p.product_name
    JOIN groups g
      ON g.id = p.group_id
     AND g.name = d.group_name
     AND g.deleted_at IS NULL
    WHERE g.subscription_type = 'subscription'
      AND p.description = d.description;

    IF refreshed_count <> 40 THEN
        RAISE EXCEPTION 'subscription plan description refresh expected 40 updated rows, got %', refreshed_count;
    END IF;
END $$;
