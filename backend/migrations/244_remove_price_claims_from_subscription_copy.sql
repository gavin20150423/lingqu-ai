-- 244: Remove inline price, market-price and discount claims from subscription copy.
--
-- Sale prices and quota amounts are rendered by dedicated fields. Keep the
-- description/features copy focused on use cases, advantages and benefits.
-- This migration changes only description/features for the eight formal
-- subscription series; plan prices, currencies, quota columns, validity,
-- group relationships and payment state are intentionally untouched.

CREATE TEMP TABLE _subscription_plan_copy_without_prices (
    group_name  TEXT NOT NULL,
    tier_name   TEXT NOT NULL,
    description TEXT NOT NULL,
    features    TEXT NOT NULL,
    PRIMARY KEY (group_name, tier_name)
);

INSERT INTO _subscription_plan_copy_without_prices (group_name, tier_name, description, features)
VALUES
    (
        'CCMax Claude 订阅',
        'Basic',
        '适合轻量 Claude Code 使用：小型修复、脚本编写和日常问答，快速完成日常开发任务。',
        E'适用场景：小型修复、脚本编写与日常问答\n核心优势：Claude Code 稳定接入，轻量任务响应直接\n套餐权益：Claude / Claude Code 模型，日 / 周 / 月独立额度'
    ),
    (
        'CCMax Claude 订阅',
        'Plus',
        '适合持续编码：多轮代码解释、中小项目迭代和日常功能开发，保持连贯的工作节奏。',
        E'适用场景：持续编码、代码解释与中小项目迭代\n核心优势：兼顾响应频率与使用空间，适合日常开发\n套餐权益：Claude / Claude Code 模型，日 / 周 / 月独立额度'
    ),
    (
        'CCMax Claude 订阅',
        'Standard',
        '适合个人主力开发：连续编程、复杂修改和长上下文协作，覆盖大多数工程任务。',
        E'适用场景：个人主力开发、复杂修改与长上下文协作\n核心优势：额度与连续性更均衡，适合稳定推进项目\n套餐权益：Claude / Claude Code 模型，日 / 周 / 月独立额度'
    ),
    (
        'CCMax Claude 订阅',
        'Pro',
        '适合高频工程工作：大型项目重构、长上下文分析和连续 Agent 任务，减少工作中断。',
        E'适用场景：大型项目重构、高频开发与 Agent 流程\n核心优势：更充足的使用空间，支持长时间连续工作\n套餐权益：Claude / Claude Code 模型，日 / 周 / 月独立额度'
    ),
    (
        'CCMax Claude 订阅',
        'Ultra',
        '适合重度与团队使用：并行项目、持续 Agent 流程和高峰期调用，满足规模化开发需求。',
        E'适用场景：团队协作、并行项目与持续 Agent 任务\n核心优势：最大使用空间，适合重度调用和高峰期工作\n套餐权益：Claude / Claude Code 模型，日 / 周 / 月独立额度'
    ),

    (
        'Kiro Claude 订阅',
        'Basic',
        '适合轻量 Kiro Claude 辅助：代码补全、简单修复和日常问答，快速获得开发建议。',
        E'适用场景：代码补全、简单修复与日常问答\n核心优势：适合快速获得稳定的编程辅助\n套餐权益：Kiro Claude 模型，日 / 周 / 月独立额度'
    ),
    (
        'Kiro Claude 订阅',
        'Plus',
        '适合日常编码：功能开发、代码解释和持续迭代，让 IDE 辅助融入工作流程。',
        E'适用场景：功能开发、代码解释与日常迭代\n核心优势：适合高频 IDE 辅助，工作节奏更连贯\n套餐权益：Kiro Claude 模型，日 / 周 / 月独立额度'
    ),
    (
        'Kiro Claude 订阅',
        'Standard',
        '适合个人稳定开发：中型项目、跨文件修改和多轮协作，覆盖常见工程任务。',
        E'适用场景：中型项目、跨文件修改与多轮协作\n核心优势：连续开发与使用空间保持平衡\n套餐权益：Kiro Claude 模型，日 / 周 / 月独立额度'
    ),
    (
        'Kiro Claude 订阅',
        'Pro',
        '适合专业工程任务：大型仓库、复杂重构和 Agent 工作流，减少上下文切换。',
        E'适用场景：大型仓库、复杂重构与 Agent 工作流\n核心优势：更充足的使用空间，支持长时间连续编码\n套餐权益：Kiro Claude 模型，日 / 周 / 月独立额度'
    ),
    (
        'Kiro Claude 订阅',
        'Ultra',
        '适合重度开发与团队协作：并行项目、持续 Agent 任务和高峰期使用，支撑密集开发。',
        E'适用场景：团队协作、并行项目与持续 Agent 任务\n核心优势：最大使用空间，适合重度开发与高频调用\n套餐权益：Kiro Claude 模型，日 / 周 / 月独立额度'
    ),

    (
        'GPT 订阅',
        'Basic',
        '适合 GPT / Codex 入门使用：日常问答、代码审阅和轻量开发，覆盖常见工作需求。',
        E'适用场景：日常问答、代码审阅与轻量开发\n核心优势：快速覆盖常见 GPT / Codex 任务\n套餐权益：GPT / Codex 模型，日 / 周 / 月独立额度'
    ),
    (
        'GPT 订阅',
        'Plus',
        '适合持续编码与推理：功能开发、调试和多轮技术对话，保持稳定的调用节奏。',
        E'适用场景：功能开发、调试与多轮技术对话\n核心优势：响应与使用空间更均衡，适合连续使用\n套餐权益：GPT / Codex 模型，日 / 周 / 月独立额度'
    ),
    (
        'GPT 订阅',
        'Standard',
        '适合个人主力使用：完整项目迭代、方案分析和中长上下文任务，满足日常开发需要。',
        E'适用场景：项目迭代、方案分析与中长上下文任务\n核心优势：适用范围广，适合作为个人主力档位\n套餐权益：GPT / Codex 模型，日 / 周 / 月独立额度'
    ),
    (
        'GPT 订阅',
        'Pro',
        '适合高频推理与复杂工程：大型项目、深度分析和 Agent 协作，支持持续推进工作流。',
        E'适用场景：大型项目、深度推理与 Agent 协作\n核心优势：更大的使用空间，适合高强度连续工作\n套餐权益：GPT / Codex 模型，日 / 周 / 月独立额度'
    ),
    (
        'GPT 订阅',
        'Ultra',
        '适合重度 GPT / Codex 使用：团队协作、并行项目和持续自动化任务，支撑高峰期需求。',
        E'适用场景：团队协作、并行项目与持续自动化任务\n核心优势：最大使用空间，适合重度调用\n套餐权益：GPT / Codex 模型，日 / 周 / 月独立额度'
    ),

    (
        'Gemini 订阅',
        'Basic',
        '适合 Gemini 入门使用：轻量问答、文本整理和偶发多模态任务，快速体验综合能力。',
        E'适用场景：轻量问答、文本整理与偶发多模态任务\n核心优势：覆盖文本、视觉与基础创作需求\n套餐权益：Gemini 模型，日 / 周 / 月独立额度'
    ),
    (
        'Gemini 订阅',
        'Plus',
        '适合日常多模态创作：文档理解、图片分析和常规开发，满足稳定的内容处理需求。',
        E'适用场景：文档理解、图片分析与常规开发\n核心优势：文本、视觉与代码任务一体化处理\n套餐权益：Gemini 模型，日 / 周 / 月独立额度'
    ),
    (
        'Gemini 订阅',
        'Standard',
        '适合个人主力创作：持续分析、内容生成和多模态工作流，覆盖日常生产场景。',
        E'适用场景：持续分析、内容生成与多模态工作流\n核心优势：适合稳定产出，兼顾多种创作任务\n套餐权益：Gemini 模型，日 / 周 / 月独立额度'
    ),
    (
        'Gemini 订阅',
        'Pro',
        '适合高频多模态任务：长上下文分析、批量内容处理和复杂研究，减少工作流中断。',
        E'适用场景：长上下文分析、批量处理与复杂研究\n核心优势：更大的使用空间，适合高频多模态生产\n套餐权益：Gemini 模型，日 / 周 / 月独立额度'
    ),
    (
        'Gemini 订阅',
        'Ultra',
        '适合团队与重度多模态使用：并行创作、持续分析和高峰调用，支撑规模化内容生产。',
        E'适用场景：团队创作、并行分析与高峰期调用\n核心优势：最大使用空间，适合重度多模态工作\n套餐权益：Gemini 模型，日 / 周 / 月独立额度'
    ),

    (
        '国模订阅',
        'Basic',
        '适合国产模型轻量使用：中文问答、翻译和简单代码任务，灵活进入多模型工作流。',
        E'适用场景：中文问答、翻译与简单代码任务\n核心优势：统一接入 GLM、DeepSeek、Kimi、MiniMax、Qwen\n套餐权益：GLM / DeepSeek / Kimi / MiniMax / Qwen，日 / 周 / 月独立额度'
    ),
    (
        '国模订阅',
        'Plus',
        '适合日常中文工作：内容整理、代码辅助和常规推理，在不同国产模型间顺畅切换。',
        E'适用场景：内容整理、代码辅助与常规推理\n核心优势：多家国产模型统一入口，中文任务适配自然\n套餐权益：GLM / DeepSeek / Kimi / MiniMax / Qwen，日 / 周 / 月独立额度'
    ),
    (
        '国模订阅',
        'Standard',
        '适合个人主力使用：中文创作、模型对比和持续开发，覆盖长期多模型使用场景。',
        E'适用场景：中文创作、模型对比与持续开发\n核心优势：额度与模型选择更均衡，适合长期使用\n套餐权益：GLM / DeepSeek / Kimi / MiniMax / Qwen，日 / 周 / 月独立额度'
    ),
    (
        '国模订阅',
        'Pro',
        '适合高频国产模型任务：深度推理、Agent 和批量处理，支持复杂工作流持续运行。',
        E'适用场景：深度推理、Agent 与批量处理\n核心优势：更大的使用空间，适合高频复杂工作流\n套餐权益：GLM / DeepSeek / Kimi / MiniMax / Qwen，日 / 周 / 月独立额度'
    ),
    (
        '国模订阅',
        'Ultra',
        '适合团队与重度国产模型使用：并行项目和持续自动化任务，满足集中调用需求。',
        E'适用场景：团队协作、并行项目与持续自动化任务\n核心优势：最大使用空间，支持重度多模型调用\n套餐权益：GLM / DeepSeek / Kimi / MiniMax / Qwen，日 / 周 / 月独立额度'
    ),

    (
        'Grok 订阅',
        'Basic',
        '适合 Grok 轻量探索：日常问答、实时信息整理和灵感发散，快速开始对话。',
        E'适用场景：日常问答、实时信息整理与灵感探索\n核心优势：适合快速获取观点与信息整理结果\n套餐权益：Grok 模型，日 / 周 / 月独立额度'
    ),
    (
        'Grok 订阅',
        'Plus',
        '适合持续对话与内容辅助：资料梳理、写作协作和日常开发，保持上下文连续。',
        E'适用场景：资料梳理、写作协作与日常开发\n核心优势：支持连续对话，创作与分析衔接自然\n套餐权益：Grok 模型，日 / 周 / 月独立额度'
    ),
    (
        'Grok 订阅',
        'Standard',
        '适合个人主力使用：研究分析、内容创作和多轮任务协作，覆盖更多实际工作流。',
        E'适用场景：研究分析、内容创作与多轮任务协作\n核心优势：适用范围均衡，适合稳定推进任务\n套餐权益：Grok 模型，日 / 周 / 月独立额度'
    ),
    (
        'Grok 订阅',
        'Pro',
        '适合高频推理与 Agent 任务：长对话、复杂分析和持续生产，支持高强度工作。',
        E'适用场景：长对话、复杂分析与 Agent 任务\n核心优势：更大的使用空间，适合连续调用\n套餐权益：Grok 模型，日 / 周 / 月独立额度'
    ),
    (
        'Grok 订阅',
        'Ultra',
        '适合团队与重度 Grok 使用：并行研究、批量内容和高峰期调用，支撑规模化工作流。',
        E'适用场景：团队研究、批量内容与高峰期调用\n核心优势：最大使用空间，适合重度使用\n套餐权益：Grok 模型，日 / 周 / 月独立额度'
    ),

    (
        'GPT Image 2.5 订阅',
        'Basic',
        '适合轻量图片创作：灵感草图、头像尝试和偶发生图，快速验证视觉方向。',
        E'适用场景：灵感草图、头像尝试与偶发创作\n核心优势：两个 GPT Image 模型可选，创作更灵活\n套餐权益：每月 500 张生成额度，日 / 周 / 月独立限额'
    ),
    (
        'GPT Image 2.5 订阅',
        'Plus',
        '适合稳定图片创作：社交内容、产品概念和日常视觉素材，保持持续产出。',
        E'适用场景：社交内容、产品概念与日常视觉素材\n核心优势：适合稳定创作，减少临时补充额度的需要\n套餐权益：每月 1,500 张生成额度，日 / 周 / 月独立限额'
    ),
    (
        'GPT Image 2.5 订阅',
        'Standard',
        '适合日常批量生图：内容团队、营销素材和连续创作，覆盖常规生产需求。',
        E'适用场景：营销素材、内容团队与日常批量生图\n核心优势：主力张数更充足，适合稳定批量生产\n套餐权益：每月 3,000 张生成额度，日 / 周 / 月独立限额'
    ),
    (
        'GPT Image 2.5 订阅',
        'Pro',
        '适合专业图片生产：活动物料、产品批次和高频内容团队，支持连续视觉产出。',
        E'适用场景：活动物料、产品批次与高频内容生产\n核心优势：更充足的生成空间，适合专业批量工作\n套餐权益：每月 8,000 张生成额度，日 / 周 / 月独立限额'
    ),
    (
        'GPT Image 2.5 订阅',
        'Ultra',
        '适合商业与团队级生图：大批量广告、素材库和持续项目，支撑规模化视觉生产。',
        E'适用场景：广告批量制作、素材库与持续商业项目\n核心优势：最大生成空间，适合规模化生产\n套餐权益：每月 15,000 张生成额度，日 / 周 / 月独立限额'
    ),

    (
        '香蕉生图订阅',
        'Basic',
        '适合 nano banana 轻量创作：灵感尝试、社交配图和偶发图片生成，快速验证想法。',
        E'适用场景：灵感尝试、社交配图与偶发创作\n核心优势：轻量灵活，适合快速验证视觉想法\n套餐权益：每月 500 张生成额度，日 / 周 / 月独立限额'
    ),
    (
        '香蕉生图订阅',
        'Plus',
        '适合稳定多模态创作：日常内容、文章配图和小批量素材，保持规律产出。',
        E'适用场景：日常内容、文章配图与小批量素材\n核心优势：生成空间适中，覆盖常规创作节奏\n套餐权益：每月 1,500 张生成额度，日 / 周 / 月独立限额'
    ),
    (
        '香蕉生图订阅',
        'Standard',
        '适合主力内容生产：账号运营、营销配图和批量创作，满足日常运营需求。',
        E'适用场景：账号运营、营销配图与批量创作\n核心优势：主力生成空间更充足，适合持续生产\n套餐权益：每月 3,000 张生成额度，日 / 周 / 月独立限额'
    ),
    (
        '香蕉生图订阅',
        'Pro',
        '适合专业视觉生产：活动系列、品牌内容和高频素材制作，支持连续创作项目。',
        E'适用场景：活动系列、品牌内容与高频素材制作\n核心优势：更大的生成空间，适合专业创作\n套餐权益：每月 5,000 张生成额度，日 / 周 / 月独立限额'
    ),
    (
        '香蕉生图订阅',
        'Ultra',
        '适合团队与规模化生图：广告素材库、矩阵运营和持续商业项目，支撑高峰期产出。',
        E'适用场景：广告素材库、矩阵运营与持续商业项目\n核心优势：最大生成空间，适合规模化视觉生产\n套餐权益：每月 10,000 张生成额度，日 / 周 / 月独立限额'
    );

DO $$
DECLARE
    matched_count   INTEGER;
    updated_count   INTEGER;
    refreshed_count INTEGER;
    remaining_count INTEGER;
BEGIN
    SELECT COUNT(DISTINCT p.id)
      INTO matched_count
      FROM subscription_plans p
      JOIN _subscription_plan_copy_without_prices d
        ON d.tier_name = p.name
       AND replace(
               replace(d.group_name, 'GPT Image 2.5 订阅', 'GPT Image订阅'),
               '香蕉生图订阅', 'nano banana订阅'
           ) = replace(
               replace(p.product_name, 'GPT Image 2.5 订阅', 'GPT Image订阅'),
               '香蕉生图订阅', 'nano banana订阅'
           )
      JOIN groups g
        ON g.id = p.group_id
       AND replace(
               replace(g.name, 'GPT Image 2.5 订阅', 'GPT Image订阅'),
               '香蕉生图订阅', 'nano banana订阅'
           ) = replace(
               replace(d.group_name, 'GPT Image 2.5 订阅', 'GPT Image订阅'),
               '香蕉生图订阅', 'nano banana订阅'
           )
       AND g.deleted_at IS NULL
     WHERE g.subscription_type = 'subscription';

    -- Older local fixtures may not contain the formal catalog yet.
    IF matched_count = 0 THEN
        RAISE NOTICE '244: formal subscription catalog not present; no-op';
        RETURN;
    END IF;

    IF matched_count <> 40 THEN
        RAISE EXCEPTION '244: expected 40 formal subscription plans, found %', matched_count;
    END IF;

    UPDATE subscription_plans p
       SET description = d.description,
           features = d.features,
           updated_at = NOW()
      FROM _subscription_plan_copy_without_prices d
      JOIN groups g
        ON replace(
               replace(g.name, 'GPT Image 2.5 订阅', 'GPT Image订阅'),
               '香蕉生图订阅', 'nano banana订阅'
           ) = replace(
               replace(d.group_name, 'GPT Image 2.5 订阅', 'GPT Image订阅'),
               '香蕉生图订阅', 'nano banana订阅'
           )
       AND g.deleted_at IS NULL
       AND g.subscription_type = 'subscription'
     WHERE p.group_id = g.id
       AND p.name = d.tier_name
       AND replace(
               replace(p.product_name, 'GPT Image 2.5 订阅', 'GPT Image订阅'),
               '香蕉生图订阅', 'nano banana订阅'
           ) = replace(
               replace(d.group_name, 'GPT Image 2.5 订阅', 'GPT Image订阅'),
               '香蕉生图订阅', 'nano banana订阅'
           );

    GET DIAGNOSTICS updated_count = ROW_COUNT;

    SELECT COUNT(DISTINCT p.id)
      INTO refreshed_count
      FROM subscription_plans p
      JOIN _subscription_plan_copy_without_prices d
        ON d.tier_name = p.name
       AND replace(
               replace(d.group_name, 'GPT Image 2.5 订阅', 'GPT Image订阅'),
               '香蕉生图订阅', 'nano banana订阅'
           ) = replace(
               replace(p.product_name, 'GPT Image 2.5 订阅', 'GPT Image订阅'),
               '香蕉生图订阅', 'nano banana订阅'
           )
      JOIN groups g
        ON g.id = p.group_id
       AND replace(
               replace(g.name, 'GPT Image 2.5 订阅', 'GPT Image订阅'),
               '香蕉生图订阅', 'nano banana订阅'
           ) = replace(
               replace(d.group_name, 'GPT Image 2.5 订阅', 'GPT Image订阅'),
               '香蕉生图订阅', 'nano banana订阅'
           )
       AND g.deleted_at IS NULL
     WHERE g.subscription_type = 'subscription'
       AND p.description = d.description
       AND p.features = d.features;

    IF refreshed_count <> updated_count OR updated_count <> 40 THEN
        RAISE EXCEPTION '244: expected 40 refreshed plans, updated %, verified %', updated_count, refreshed_count;
    END IF;

    SELECT COUNT(*)
      INTO remaining_count
      FROM subscription_plans p
      JOIN _subscription_plan_copy_without_prices d
        ON d.tier_name = p.name
       AND replace(
               replace(d.group_name, 'GPT Image 2.5 订阅', 'GPT Image订阅'),
               '香蕉生图订阅', 'nano banana订阅'
           ) = replace(
               replace(p.product_name, 'GPT Image 2.5 订阅', 'GPT Image订阅'),
               '香蕉生图订阅', 'nano banana订阅'
           )
      JOIN groups g
        ON g.id = p.group_id
       AND replace(
               replace(g.name, 'GPT Image 2.5 订阅', 'GPT Image订阅'),
               '香蕉生图订阅', 'nano banana订阅'
           ) = replace(
               replace(d.group_name, 'GPT Image 2.5 订阅', 'GPT Image订阅'),
               '香蕉生图订阅', 'nano banana订阅'
           )
       AND g.deleted_at IS NULL
     WHERE g.subscription_type = 'subscription'
       AND (
           p.description ~ '(参考市场|上游成本|单价|计费|折|省[约 ]*[0-9]|[$¥￥][0-9])'
           OR p.features ~ '(参考市场|上游成本|单价|计费|折|省[约 ]*[0-9]|[$¥￥][0-9])'
       );

    IF remaining_count <> 0 THEN
        RAISE EXCEPTION '244: % subscription plans still contain inline price claims', remaining_count;
    END IF;

    RAISE NOTICE '244: removed inline price claims from % subscription plans', updated_count;
END $$;
