-- 242: 完善正式订阅目录的套餐简介与权益文案。
--
-- 套餐卡片的 description 和 features 都来自数据库。features 的前三行会
-- 在用户侧优先展示，因此按“适用场景 / 核心优势 / 权益内容”组织文案。
-- 本迁移只更新这两个文案字段，不修改名称、售价、币种、额度、有效期、
-- 分组关系、上下游配置或支付逻辑。

CREATE TEMP TABLE _subscription_plan_copy_refresh (
    group_name  TEXT NOT NULL,
    tier_name   TEXT NOT NULL,
    description TEXT NOT NULL,
    features    TEXT NOT NULL,
    PRIMARY KEY (group_name, tier_name)
);

INSERT INTO _subscription_plan_copy_refresh (group_name, tier_name, description, features)
VALUES
    (
        'CCMax Claude 订阅',
        'Basic',
        '面向轻量 Claude Code 使用：适合小型修复、脚本编写和日常问答，按 $0.90/$ 计费，较参考市场 $1.20/$ 约省 25%。',
        E'适用场景：小型修复、脚本编写与日常问答\n核心优势：Claude Code 稳定接入，轻量任务快速完成\n权益内容：Claude / Claude Code 模型，按 $0.90/$ 计费'
    ),
    (
        'CCMax Claude 订阅',
        'Plus',
        '面向持续开发：适合多轮编码、代码解释和中小项目迭代，在日常效率与额度之间保持平衡，较参考市场价约省 25%。',
        E'适用场景：持续编码、代码解释与中小项目迭代\n核心优势：兼顾响应频率与使用空间，适合日常开发\n权益内容：Claude / Claude Code 模型，按 $0.90/$ 计费'
    ),
    (
        'CCMax Claude 订阅',
        'Standard',
        '面向个人主力开发：适合连续编程、复杂修改和长上下文协作，覆盖大多数日常工程任务，较参考市场价约省 25%。',
        E'适用场景：个人主力开发、复杂修改与长上下文协作\n核心优势：覆盖大多数工程任务，额度与连续性更均衡\n权益内容：Claude / Claude Code 模型，按 $0.90/$ 计费'
    ),
    (
        'CCMax Claude 订阅',
        'Pro',
        '面向高频工程工作：适合大型项目重构、长上下文分析和连续 Agent 任务，减少中途断档，较参考市场价约省 25%。',
        E'适用场景：大型项目重构、高频开发与 Agent 流程\n核心优势：更高月度额度，适合长时间连续工作\n权益内容：Claude / Claude Code 模型，按 $0.90/$ 计费'
    ),
    (
        'CCMax Claude 订阅',
        'Ultra',
        '面向重度与团队使用：适合并行项目、持续 Agent 流程和高峰期调用，提供最高月度额度，较参考市场价约省 25%。',
        E'适用场景：团队协作、并行项目与持续 Agent 任务\n核心优势：最高月度额度，适合重度调用和高峰期使用\n权益内容：Claude / Claude Code 模型，按 $0.90/$ 计费'
    ),

    (
        'Kiro Claude 订阅',
        'Basic',
        '面向轻量 Kiro Claude 辅助：适合代码补全、简单修复和日常问答，按 $0.17/$ 计费，较参考市场 $0.25/$ 约省 32%。',
        E'适用场景：代码补全、简单修复与日常问答\n核心优势：低成本获得稳定的 Kiro Claude 编程辅助\n权益内容：Kiro Claude 模型，按 $0.17/$ 计费'
    ),
    (
        'Kiro Claude 订阅',
        'Plus',
        '面向日常编码：适合功能开发、代码解释和持续迭代，兼顾使用频率与预算，较参考市场价约省 32%。',
        E'适用场景：功能开发、代码解释与日常迭代\n核心优势：适合高频 IDE 辅助，工作节奏更连贯\n权益内容：Kiro Claude 模型，按 $0.17/$ 计费'
    ),
    (
        'Kiro Claude 订阅',
        'Standard',
        '面向个人稳定开发：适合中型项目、跨文件修改和多轮协作，作为日常主力档位，较参考市场价约省 32%。',
        E'适用场景：中型项目、跨文件修改与多轮协作\n核心优势：在连续开发和额度空间之间取得平衡\n权益内容：Kiro Claude 模型，按 $0.17/$ 计费'
    ),
    (
        'Kiro Claude 订阅',
        'Pro',
        '面向专业工程任务：适合大型仓库、复杂重构和 Agent 工作流，减少上下文切换，较参考市场价约省 32%。',
        E'适用场景：大型仓库、复杂重构与 Agent 工作流\n核心优势：更充足的月度空间，支持长时间连续编码\n权益内容：Kiro Claude 模型，按 $0.17/$ 计费'
    ),
    (
        'Kiro Claude 订阅',
        'Ultra',
        '面向重度开发与团队协作：适合并行项目、持续 Agent 任务和高峰期使用，较参考市场价约省 32%。',
        E'适用场景：团队协作、并行项目与持续 Agent 任务\n核心优势：最高额度，适合重度开发和稳定高频调用\n权益内容：Kiro Claude 模型，按 $0.17/$ 计费'
    ),

    (
        'GPT 订阅',
        'Basic',
        '面向 GPT / Codex 入门使用：适合日常问答、代码审阅和轻量开发，按 $0.18/$ 计费，较参考市场 $0.31/$ 约省 42%。',
        E'适用场景：日常问答、代码审阅与轻量开发\n核心优势：用较低成本覆盖常见 GPT / Codex 任务\n权益内容：GPT / Codex 模型，按 $0.18/$ 计费'
    ),
    (
        'GPT 订阅',
        'Plus',
        '面向持续编码与推理：适合功能开发、调试和多轮技术对话，保持稳定调用节奏，较参考市场价约省 42%。',
        E'适用场景：功能开发、调试与多轮技术对话\n核心优势：适合日常连续使用，响应与额度更均衡\n权益内容：GPT / Codex 模型，按 $0.18/$ 计费'
    ),
    (
        'GPT 订阅',
        'Standard',
        '面向个人主力使用：适合完整项目迭代、方案分析和中长上下文任务，覆盖大多数开发需求，较参考市场价约省 42%。',
        E'适用场景：项目迭代、方案分析与中长上下文任务\n核心优势：个人开发者的均衡主力档，适用范围更广\n权益内容：GPT / Codex 模型，按 $0.18/$ 计费'
    ),
    (
        'GPT 订阅',
        'Pro',
        '面向高频推理与复杂工程：适合大型项目、深度分析和 Agent 协作，减少额度中断，较参考市场价约省 42%。',
        E'适用场景：大型项目、深度推理与 Agent 协作\n核心优势：更高月度空间，适合高强度连续工作\n权益内容：GPT / Codex 模型，按 $0.18/$ 计费'
    ),
    (
        'GPT 订阅',
        'Ultra',
        '面向重度 GPT / Codex 使用：适合团队协作和持续自动化任务，最高档额外 9 折，约 $0.162/$，较参考市场价约省 48%。',
        E'适用场景：团队协作、并行项目与持续自动化任务\n核心优势：最高额度，最高档额外 9 折，适合重度调用\n权益内容：GPT / Codex 模型，约 $0.162/$，较参考市场价约省 48%'
    ),

    (
        'Gemini 订阅',
        'Basic',
        '面向 Gemini 入门使用：适合轻量问答、文本整理和偶发多模态任务，按 $0.40/$ 计费，较参考市场 $0.60/$ 约省 33%。',
        E'适用场景：轻量问答、文本整理与偶发多模态任务\n核心优势：低门槛体验 Gemini 的文本与多模态能力\n权益内容：Gemini 模型，按 $0.40/$ 计费'
    ),
    (
        'Gemini 订阅',
        'Plus',
        '面向日常多模态创作：适合文档理解、图片分析和常规开发，使用频率与额度更协调，较参考市场价约省 33%。',
        E'适用场景：文档理解、图片分析与常规开发\n核心优势：文本、视觉与代码任务一体化处理\n权益内容：Gemini 模型，按 $0.40/$ 计费'
    ),
    (
        'Gemini 订阅',
        'Standard',
        '面向个人主力创作：适合持续分析、内容生成和多模态工作流，覆盖日常生产需求，较参考市场价约省 33%。',
        E'适用场景：持续分析、内容生成与多模态工作流\n核心优势：个人使用的均衡档位，适合稳定产出\n权益内容：Gemini 模型，按 $0.40/$ 计费'
    ),
    (
        'Gemini 订阅',
        'Pro',
        '面向高频多模态任务：适合长上下文分析、批量内容处理和复杂研究，减少工作流中断，较参考市场价约省 33%。',
        E'适用场景：长上下文分析、批量处理与复杂研究\n核心优势：更充足的额度，适合高频多模态生产\n权益内容：Gemini 模型，按 $0.40/$ 计费'
    ),
    (
        'Gemini 订阅',
        'Ultra',
        '面向团队与重度多模态使用：适合并行创作、持续分析和高峰调用，最高档额外 9 折，约 $0.36/$，较参考市场价约省 40%。',
        E'适用场景：团队创作、并行分析与高峰期调用\n核心优势：最高额度，最高档额外 9 折，适合重度使用\n权益内容：Gemini 模型，约 $0.36/$，较参考市场价约省 40%'
    ),

    (
        '国模订阅',
        'Basic',
        '面向国产模型轻量使用：适合中文问答、翻译和简单代码任务，覆盖 GLM、DeepSeek、Kimi、MiniMax、Qwen，按上游成本 3 折。',
        E'适用场景：中文问答、翻译与简单代码任务\n核心优势：五大国产模型统一入口，按需灵活切换\n权益内容：GLM / DeepSeek / Kimi / MiniMax / Qwen，按上游成本 3 折'
    ),
    (
        '国模订阅',
        'Plus',
        '面向日常中文工作：适合内容整理、代码辅助和常规推理，多模型协作更顺手，按上游成本 3 折。',
        E'适用场景：内容整理、代码辅助与常规推理\n核心优势：覆盖多家国产模型，中文任务适配更自然\n权益内容：GLM / DeepSeek / Kimi / MiniMax / Qwen，按上游成本 3 折'
    ),
    (
        '国模订阅',
        'Standard',
        '面向个人主力使用：适合中文创作、模型对比和持续开发，在不同国产模型之间平滑切换，按上游成本 3 折。',
        E'适用场景：中文创作、模型对比与持续开发\n核心优势：主力额度更均衡，适合长期多模型使用\n权益内容：GLM / DeepSeek / Kimi / MiniMax / Qwen，按上游成本 3 折'
    ),
    (
        '国模订阅',
        'Pro',
        '面向高频国产模型任务：适合深度推理、Agent 和批量处理，支持复杂工作流持续运行，按上游成本 3 折。',
        E'适用场景：深度推理、Agent 与批量处理\n核心优势：更大的额度空间，适合高频复杂工作流\n权益内容：GLM / DeepSeek / Kimi / MiniMax / Qwen，按上游成本 3 折'
    ),
    (
        '国模订阅',
        'Ultra',
        '面向团队与重度国产模型使用：适合并行项目和持续自动化任务，基础 3 折再享 9 折，约为上游成本 2.7 折。',
        E'适用场景：团队协作、并行项目与持续自动化任务\n核心优势：最高额度，基础 3 折再享 9 折，约 2.7 折\n权益内容：GLM / DeepSeek / Kimi / MiniMax / Qwen，约为上游成本 2.7 折'
    ),

    (
        'Grok 订阅',
        'Basic',
        '面向 Grok 轻量探索：适合日常问答、实时信息整理和灵感 brainstorming，常规单价 $0.20/$。',
        E'适用场景：日常问答、实时信息整理与灵感探索\n核心优势：快速进入 Grok 的实时对话与分析场景\n权益内容：Grok 文本模型，常规单价 $0.20/$'
    ),
    (
        'Grok 订阅',
        'Plus',
        '面向持续对话与内容辅助：适合资料梳理、写作协作和日常开发，常规单价 $0.20/$。',
        E'适用场景：资料梳理、写作协作与日常开发\n核心优势：适合连续对话，保持上下文与创作节奏\n权益内容：Grok 文本模型，常规单价 $0.20/$'
    ),
    (
        'Grok 订阅',
        'Standard',
        '面向个人主力使用：适合研究分析、内容创作和多轮任务协作，常规单价 $0.20/$。',
        E'适用场景：研究分析、内容创作与多轮任务协作\n核心优势：个人使用的均衡档，覆盖更多实际工作流\n权益内容：Grok 文本模型，常规单价 $0.20/$'
    ),
    (
        'Grok 订阅',
        'Pro',
        '面向高频推理与 Agent 任务：适合长对话、复杂分析和持续生产，常规单价 $0.20/$。',
        E'适用场景：长对话、复杂分析与 Agent 任务\n核心优势：更高额度，适合高强度连续调用\n权益内容：Grok 文本模型，常规单价 $0.20/$'
    ),
    (
        'Grok 订阅',
        'Ultra',
        '面向团队与重度 Grok 使用：适合并行研究、批量内容和高峰期调用，最高档额外 9 折，单价 $0.18/$。',
        E'适用场景：团队研究、批量内容与高峰期调用\n核心优势：最高额度，最高档额外 9 折，适合重度使用\n权益内容：Grok 文本模型，最高档单价 $0.18/$'
    ),

    (
        'GPT Image 2.5 订阅',
        'Basic',
        '面向 GPT Image 轻量创作：适合灵感草图、头像尝试和偶发生图，支持两个模型选择，每月 500 张。',
        E'适用场景：灵感草图、头像尝试与偶发创作\n核心优势：两个 GPT Image 模型可选，按需生成更灵活\n权益内容：每月 500 张，基础单价 $0.08/张'
    ),
    (
        'GPT Image 2.5 订阅',
        'Plus',
        '面向稳定图片创作：适合社交内容、产品概念和日常视觉素材，每月 1,500 张，保持持续产出。',
        E'适用场景：社交内容、产品概念与日常视觉素材\n核心优势：适合稳定创作，减少临时补充额度的需要\n权益内容：每月 1,500 张，基础单价 $0.08/张'
    ),
    (
        'GPT Image 2.5 订阅',
        'Standard',
        '面向日常批量生图：适合内容团队、营销素材和连续创作，每月 3,000 张，覆盖常规生产需求。',
        E'适用场景：营销素材、内容团队与日常批量生图\n核心优势：主力张数更充足，适合稳定批量生产\n权益内容：每月 3,000 张，基础单价 $0.08/张'
    ),
    (
        'GPT Image 2.5 订阅',
        'Pro',
        '面向专业图片生产：适合活动物料、产品批次和高频内容团队，每月 8,000 张，阶梯单价 $0.06/张，较基础价省 25%。',
        E'适用场景：活动物料、产品批次与高频内容生产\n核心优势：超过 3,000 张进入阶梯价，单张成本更低\n权益内容：每月 8,000 张，$0.06/张，较基础价省 25%'
    ),
    (
        'GPT Image 2.5 订阅',
        'Ultra',
        '面向商业与团队级生图：适合大批量广告、素材库和持续项目，每月 15,000 张，阶梯单价 $0.05/张，较基础价省 37.5%。',
        E'适用场景：广告批量制作、素材库与持续商业项目\n核心优势：最高张数与最低阶梯单价，适合规模化生产\n权益内容：每月 15,000 张，$0.05/张，较基础价省 37.5%'
    ),

    (
        '香蕉生图订阅',
        'Basic',
        '面向 nano banana 轻量创作：适合灵感尝试、社交配图和偶发图片生成，每月 500 张，基础单价 $0.15/张。',
        E'适用场景：灵感尝试、社交配图与偶发创作\n核心优势：轻量灵活，适合快速验证视觉想法\n权益内容：每月 500 张，基础单价 $0.15/张'
    ),
    (
        '香蕉生图订阅',
        'Plus',
        '面向稳定多模态创作：适合日常内容、配图和小批量素材，每月 1,500 张，保持持续产出。',
        E'适用场景：日常内容、文章配图与小批量素材\n核心优势：稳定张数覆盖常规创作节奏\n权益内容：每月 1,500 张，基础单价 $0.15/张'
    ),
    (
        '香蕉生图订阅',
        'Standard',
        '面向主力内容生产：适合账号运营、营销配图和批量创作，每月 3,000 张，阶梯单价 $0.13/张，较基础价省约 13%。',
        E'适用场景：账号运营、营销配图与批量创作\n核心优势：达到 3,000 张即进入阶梯价，适合持续生产\n权益内容：每月 3,000 张，$0.13/张，较基础价省约 13%'
    ),
    (
        '香蕉生图订阅',
        'Pro',
        '面向专业视觉生产：适合活动系列、品牌内容和高频素材制作，每月 5,000 张，阶梯单价 $0.12/张，较基础价省 20%。',
        E'适用场景：活动系列、品牌内容与高频素材制作\n核心优势：更高张数与更低单张成本，适合专业创作\n权益内容：每月 5,000 张，$0.12/张，较基础价省 20%'
    ),
    (
        '香蕉生图订阅',
        'Ultra',
        '面向团队与规模化生图：适合广告素材库、矩阵运营和持续商业项目，每月 10,000 张，阶梯单价 $0.10/张，较基础价省约 33%。',
        E'适用场景：广告素材库、矩阵运营与持续商业项目\n核心优势：最高张数与最低阶梯单价，适合规模化产出\n权益内容：每月 10,000 张，$0.10/张，较基础价省约 33%'
    );

DO $$
DECLARE
    matched_count INTEGER;
    updated_count INTEGER;
    refreshed_count INTEGER;
BEGIN
    SELECT COUNT(DISTINCT p.id)
    INTO matched_count
    FROM subscription_plans p
    JOIN _subscription_plan_copy_refresh d
      ON d.tier_name = p.name
     AND d.group_name = p.product_name
    JOIN groups g
      ON g.id = p.group_id
     AND g.name = d.group_name
     AND g.deleted_at IS NULL
    WHERE g.subscription_type = 'subscription';

    -- 旧的本地测试库可能尚未生成正式订阅目录。安全跳过，避免阻断
    -- 本地启动；正式目录存在时会一次性更新当前匹配到的全部套餐。
    IF matched_count = 0 THEN
        RAISE NOTICE 'subscription plan copy refresh skipped: formal catalog is not present';
        RETURN;
    END IF;

    UPDATE subscription_plans p
    SET description = d.description,
        features = d.features,
        updated_at = NOW()
    FROM _subscription_plan_copy_refresh d
    JOIN groups g
      ON g.name = d.group_name
     AND g.deleted_at IS NULL
     AND g.subscription_type = 'subscription'
    WHERE p.group_id = g.id
      AND p.name = d.tier_name
      AND p.product_name = d.group_name;

    GET DIAGNOSTICS updated_count = ROW_COUNT;

    SELECT COUNT(DISTINCT p.id)
    INTO refreshed_count
    FROM subscription_plans p
    JOIN _subscription_plan_copy_refresh d
      ON d.tier_name = p.name
     AND d.group_name = p.product_name
    JOIN groups g
      ON g.id = p.group_id
     AND g.name = d.group_name
     AND g.deleted_at IS NULL
    WHERE g.subscription_type = 'subscription'
      AND p.description = d.description
      AND p.features = d.features;

    IF refreshed_count <> updated_count THEN
        RAISE EXCEPTION 'subscription plan copy refresh expected % updated rows, got %', updated_count, refreshed_count;
    END IF;

    RAISE NOTICE 'subscription plan copy refresh updated % of % matched plans', updated_count, matched_count;
END $$;
