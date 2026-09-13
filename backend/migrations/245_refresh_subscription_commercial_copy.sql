-- 245: Refresh subscription copy with commercial positioning and remove Qwen
-- from the formal domestic-model subscription.
--
-- This migration updates only subscription copy and the model allowlist of
-- the domestic subscription group. Sale prices, currencies, quota columns,
-- validity, account routing and payment state are not changed.

CREATE TEMP TABLE _subscription_plan_commercial_copy (
    group_name  TEXT NOT NULL,
    tier_name   TEXT NOT NULL,
    description TEXT NOT NULL,
    features    TEXT NOT NULL,
    PRIMARY KEY (group_name, tier_name)
);

INSERT INTO _subscription_plan_commercial_copy (group_name, tier_name, description, features)
VALUES
    (
        'CCMax Claude 订阅',
        'Basic',
        '轻量入门 Claude Code，适合日常问答、代码小修和脚本编写，快速开启稳定的 AI 编程辅助。',
        E'适合场景：日常问答、代码小修与脚本编写\n服务亮点：轻量高效，快速完成常见开发任务\n支持模型：Claude / Claude Code\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'CCMax Claude 订阅',
        'Plus',
        '日常开发主力档，适合功能开发、代码解释和中小项目迭代，让连续编码更顺畅。',
        E'适合场景：功能开发、代码解释与中小项目迭代\n服务亮点：响应稳定，适合高频日常开发\n支持模型：Claude / Claude Code\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'CCMax Claude 订阅',
        'Standard',
        '个人开发主力档，适合连续编程、复杂修改和长上下文协作，覆盖大多数工程任务。',
        E'适合场景：连续编程、复杂修改与长上下文协作\n服务亮点：使用空间均衡，适合长期稳定推进项目\n支持模型：Claude / Claude Code\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'CCMax Claude 订阅',
        'Pro',
        '专业工程档，面向大型项目重构、深度分析和连续 Agent 工作流，减少开发过程中的中断。',
        E'适合场景：大型项目重构、深度分析与 Agent 工作流\n服务亮点：更适合长时间连续工作，减少上下文切换\n支持模型：Claude / Claude Code\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'CCMax Claude 订阅',
        'Ultra',
        '团队与重度使用档，适合并行项目、持续 Agent 任务和高峰期调用，支撑规模化开发协作。',
        E'适合场景：团队协作、并行项目与持续 Agent 任务\n服务亮点：使用空间充足，适合高强度开发工作流\n支持模型：Claude / Claude Code\n额度权益：日 / 周 / 月独立额度'
    ),

    (
        'Kiro Claude 订阅',
        'Basic',
        '轻量入门 Kiro Claude，适合代码补全、简单修复和日常问答，快速获得可靠的编程建议。',
        E'适合场景：代码补全、简单修复与日常问答\n服务亮点：上手简单，适合轻量 IDE 辅助\n支持模型：Kiro Claude\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'Kiro Claude 订阅',
        'Plus',
        '日常编码进阶档，适合功能开发、代码解释和持续迭代，让 AI 辅助自然融入 IDE 工作流。',
        E'适合场景：功能开发、代码解释与日常迭代\n服务亮点：适合高频使用，保持稳定开发节奏\n支持模型：Kiro Claude\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'Kiro Claude 订阅',
        'Standard',
        '个人稳定开发档，适合中型项目、跨文件修改和多轮协作，覆盖常见的工程开发需求。',
        E'适合场景：中型项目、跨文件修改与多轮协作\n服务亮点：连续开发与使用空间保持平衡\n支持模型：Kiro Claude\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'Kiro Claude 订阅',
        'Pro',
        '专业工程档，适合大型仓库、复杂重构和 Agent 工作流，帮助团队减少上下文切换。',
        E'适合场景：大型仓库、复杂重构与 Agent 工作流\n服务亮点：支持长时间连续编码，适合专业开发\n支持模型：Kiro Claude\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'Kiro Claude 订阅',
        'Ultra',
        '团队重度开发档，适合并行项目、持续 Agent 任务和高峰期使用，支撑密集型研发协作。',
        E'适合场景：团队协作、并行项目与持续 Agent 任务\n服务亮点：使用空间充足，适合重度 IDE 辅助\n支持模型：Kiro Claude\n额度权益：日 / 周 / 月独立额度'
    ),

    (
        'GPT 订阅',
        'Basic',
        'GPT / Codex 轻量入门档，适合日常问答、代码审阅和简单开发，覆盖常见工作需求。',
        E'适合场景：日常问答、代码审阅与轻量开发\n服务亮点：快速覆盖常见 GPT / Codex 工作\n支持模型：GPT / Codex\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'GPT 订阅',
        'Plus',
        '持续编码进阶档，适合功能开发、调试和多轮技术对话，保持稳定且连贯的工作节奏。',
        E'适合场景：功能开发、调试与多轮技术对话\n服务亮点：适合日常连续使用，开发体验更顺畅\n支持模型：GPT / Codex\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'GPT 订阅',
        'Standard',
        '个人主力 GPT / Codex 档，适合项目迭代、方案分析和中长上下文任务，覆盖日常开发全流程。',
        E'适合场景：项目迭代、方案分析与中长上下文任务\n服务亮点：应用范围广，适合作为个人主力方案\n支持模型：GPT / Codex\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'GPT 订阅',
        'Pro',
        '高频推理与复杂工程档，适合大型项目、深度分析和 Agent 协作，持续推进高强度工作流。',
        E'适合场景：大型项目、深度推理与 Agent 协作\n服务亮点：更适合连续推理和复杂工程任务\n支持模型：GPT / Codex\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'GPT 订阅',
        'Ultra',
        '团队与重度 GPT / Codex 档，适合并行项目、持续自动化和高峰期调用，支撑规模化生产。',
        E'适合场景：团队协作、并行项目与持续自动化任务\n服务亮点：使用空间充足，适合重度模型调用\n支持模型：GPT / Codex\n额度权益：日 / 周 / 月独立额度'
    ),

    (
        'Gemini 订阅',
        'Basic',
        'Gemini 轻量入门档，适合日常问答、文本整理和偶发多模态任务，快速体验综合能力。',
        E'适合场景：轻量问答、文本整理与偶发多模态任务\n服务亮点：覆盖文本、视觉与基础创作需求\n支持模型：Gemini\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'Gemini 订阅',
        'Plus',
        '日常多模态进阶档，适合文档理解、图片分析和常规开发，满足稳定的内容处理需求。',
        E'适合场景：文档理解、图片分析与常规开发\n服务亮点：文本、视觉与代码任务一体化处理\n支持模型：Gemini\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'Gemini 订阅',
        'Standard',
        '个人主力创作档，适合持续分析、内容生成和多模态工作流，覆盖日常生产场景。',
        E'适合场景：持续分析、内容生成与多模态工作流\n服务亮点：兼顾多种创作任务，适合稳定产出\n支持模型：Gemini\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'Gemini 订阅',
        'Pro',
        '高频多模态专业档，适合长上下文分析、批量内容处理和复杂研究，减少工作流中断。',
        E'适合场景：长上下文分析、批量处理与复杂研究\n服务亮点：更适合高频多模态生产与研究\n支持模型：Gemini\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'Gemini 订阅',
        'Ultra',
        '团队级多模态旗舰档，适合并行创作、持续分析和高峰期调用，支撑规模化内容生产。',
        E'适合场景：团队创作、并行分析与高峰期调用\n服务亮点：使用空间充足，适合重度多模态工作\n支持模型：Gemini\n额度权益：日 / 周 / 月独立额度'
    ),

    (
        '国模订阅',
        'Basic',
        '国产模型轻量入门档，适合中文问答、翻译和简单代码任务，灵活开启多模型工作流。',
        E'适合场景：中文问答、翻译与简单代码任务\n服务亮点：一个入口接入多家国产模型，使用灵活\n支持模型：GLM / DeepSeek / Kimi / MiniMax\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        '国模订阅',
        'Plus',
        '日常中文工作进阶档，适合内容整理、代码辅助和常规推理，在主流国产模型间顺畅切换。',
        E'适合场景：内容整理、代码辅助与常规推理\n服务亮点：中文任务适配自然，模型切换更高效\n支持模型：GLM / DeepSeek / Kimi / MiniMax\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        '国模订阅',
        'Standard',
        '个人主力国模档，适合中文创作、模型对比和持续开发，覆盖长期多模型使用场景。',
        E'适合场景：中文创作、模型对比与持续开发\n服务亮点：模型选择与使用空间更均衡，适合长期使用\n支持模型：GLM / DeepSeek / Kimi / MiniMax\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        '国模订阅',
        'Pro',
        '高频国产模型专业档，适合深度推理、Agent 和批量处理，支持复杂工作流持续运行。',
        E'适合场景：深度推理、Agent 与批量处理\n服务亮点：更适合高频复杂任务和连续工作流\n支持模型：GLM / DeepSeek / Kimi / MiniMax\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        '国模订阅',
        'Ultra',
        '团队与重度国模旗舰档，适合并行项目和持续自动化任务，满足集中调用与规模化协作需求。',
        E'适合场景：团队协作、并行项目与持续自动化任务\n服务亮点：使用空间充足，适合重度多模型调用\n支持模型：GLM / DeepSeek / Kimi / MiniMax\n额度权益：日 / 周 / 月独立额度'
    ),

    (
        'Grok 订阅',
        'Basic',
        'Grok 轻量探索档，适合日常问答、信息整理和灵感发散，快速开始高质量对话。',
        E'适合场景：日常问答、信息整理与灵感探索\n服务亮点：快速获取观点，适合轻量信息工作\n支持模型：Grok\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'Grok 订阅',
        'Plus',
        '持续对话进阶档，适合资料梳理、写作协作和日常开发，让分析与创作保持连贯。',
        E'适合场景：资料梳理、写作协作与日常开发\n服务亮点：支持连续对话，适合稳定内容工作\n支持模型：Grok\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'Grok 订阅',
        'Standard',
        '个人主力 Grok 档，适合研究分析、内容创作和多轮任务协作，覆盖更多实际工作流。',
        E'适合场景：研究分析、内容创作与多轮任务协作\n服务亮点：场景覆盖均衡，适合稳定推进任务\n支持模型：Grok\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'Grok 订阅',
        'Pro',
        '高频推理与 Agent 专业档，适合长对话、复杂分析和持续生产，支持高强度工作。',
        E'适合场景：长对话、复杂分析与 Agent 任务\n服务亮点：更适合连续推理和高频内容生产\n支持模型：Grok\n额度权益：日 / 周 / 月独立额度'
    ),
    (
        'Grok 订阅',
        'Ultra',
        '团队与重度 Grok 旗舰档，适合并行研究、批量内容和高峰期调用，支撑规模化工作流。',
        E'适合场景：团队研究、批量内容与高峰期调用\n服务亮点：使用空间充足，适合重度持续使用\n支持模型：Grok\n额度权益：日 / 周 / 月独立额度'
    ),

    (
        'GPT Image 2.5 订阅',
        'Basic',
        '轻量图片创作入门档，适合灵感草图、头像尝试和偶发生图，快速验证视觉方向。',
        E'适合场景：灵感草图、头像尝试与偶发创作\n服务亮点：适合快速试错，轻松开启视觉创作\n支持模型：GPT Image 2.5 系列\n额度权益：每月 500 张生成额度，日 / 周 / 月独立限额'
    ),
    (
        'GPT Image 2.5 订阅',
        'Plus',
        '稳定图片创作进阶档，适合社交内容、产品概念和日常视觉素材，保持持续产出。',
        E'适合场景：社交内容、产品概念与日常视觉素材\n服务亮点：适合规律创作，满足日常素材需求\n支持模型：GPT Image 2.5 系列\n额度权益：每月 1,500 张生成额度，日 / 周 / 月独立限额'
    ),
    (
        'GPT Image 2.5 订阅',
        'Standard',
        '日常批量生图主力档，适合内容团队、营销素材和连续创作，覆盖常规生产需求。',
        E'适合场景：营销素材、内容团队与日常批量生图\n服务亮点：生成空间充足，适合稳定批量生产\n支持模型：GPT Image 2.5 系列\n额度权益：每月 3,000 张生成额度，日 / 周 / 月独立限额'
    ),
    (
        'GPT Image 2.5 订阅',
        'Pro',
        '专业图片生产档，适合活动物料、产品批次和高频内容团队，支持连续视觉产出。',
        E'适合场景：活动物料、产品批次与高频内容生产\n服务亮点：适合专业批量工作，连续产出更从容\n支持模型：GPT Image 2.5 系列\n额度权益：每月 8,000 张生成额度，日 / 周 / 月独立限额'
    ),
    (
        'GPT Image 2.5 订阅',
        'Ultra',
        '商业与团队级生图旗舰档，适合广告批量制作、素材库和持续项目，支撑规模化视觉生产。',
        E'适合场景：广告批量制作、素材库与持续商业项目\n服务亮点：生成空间充足，适合大规模视觉生产\n支持模型：GPT Image 2.5 系列\n额度权益：每月 15,000 张生成额度，日 / 周 / 月独立限额'
    ),

    (
        '香蕉生图订阅',
        'Basic',
        'Nano Banana 轻量创作入门档，适合灵感尝试、社交配图和偶发图片生成，快速验证想法。',
        E'适合场景：灵感尝试、社交配图与偶发创作\n服务亮点：轻量灵活，适合快速验证视觉想法\n支持模型：Nano Banana 生图系列\n额度权益：每月 500 张生成额度，日 / 周 / 月独立限额'
    ),
    (
        '香蕉生图订阅',
        'Plus',
        '稳定多模态创作进阶档，适合日常内容、文章配图和小批量素材，保持规律产出。',
        E'适合场景：日常内容、文章配图与小批量素材\n服务亮点：生成节奏稳定，覆盖常规创作需求\n支持模型：Nano Banana 生图系列\n额度权益：每月 1,500 张生成额度，日 / 周 / 月独立限额'
    ),
    (
        '香蕉生图订阅',
        'Standard',
        '主力内容生产档，适合账号运营、营销配图和批量创作，满足日常运营的持续需求。',
        E'适合场景：账号运营、营销配图与批量创作\n服务亮点：生成空间充足，适合持续内容生产\n支持模型：Nano Banana 生图系列\n额度权益：每月 3,000 张生成额度，日 / 周 / 月独立限额'
    ),
    (
        '香蕉生图订阅',
        'Pro',
        '专业视觉生产档，适合活动系列、品牌内容和高频素材制作，支持连续创作项目。',
        E'适合场景：活动系列、品牌内容与高频素材制作\n服务亮点：适合专业创作，连续产出更高效\n支持模型：Nano Banana 生图系列\n额度权益：每月 5,000 张生成额度，日 / 周 / 月独立限额'
    ),
    (
        '香蕉生图订阅',
        'Ultra',
        '团队与规模化生图旗舰档，适合广告素材库、矩阵运营和持续商业项目，支撑高峰期产出。',
        E'适合场景：广告素材库、矩阵运营与持续商业项目\n服务亮点：生成空间充足，适合规模化视觉生产\n支持模型：Nano Banana 生图系列\n额度权益：每月 10,000 张生成额度，日 / 周 / 月独立限额'
    );

DO $$
DECLARE
    matched_count          INTEGER;
    updated_count          INTEGER;
    verified_count         INTEGER;
    domestic_group_count   INTEGER;
    domestic_qwen_before  INTEGER;
    domestic_qwen_after   INTEGER;
BEGIN
    SELECT COUNT(DISTINCT p.id)
      INTO matched_count
      FROM subscription_plans p
      JOIN _subscription_plan_commercial_copy d
        ON d.tier_name = p.name
       AND d.group_name = p.product_name
      JOIN groups g
        ON g.id = p.group_id
       AND g.name = d.group_name
       AND g.deleted_at IS NULL
     WHERE g.subscription_type = 'subscription';

    IF matched_count = 0 THEN
        RAISE NOTICE '245: formal subscription catalog not present; no-op';
        RETURN;
    END IF;

    IF matched_count <> 40 THEN
        RAISE EXCEPTION '245: expected 40 formal subscription plans, found %', matched_count;
    END IF;

    UPDATE subscription_plans p
       SET description = d.description,
           features = d.features,
           updated_at = NOW()
      FROM _subscription_plan_commercial_copy d
      JOIN groups g
        ON g.name = d.group_name
       AND g.deleted_at IS NULL
       AND g.subscription_type = 'subscription'
     WHERE p.group_id = g.id
       AND p.name = d.tier_name
       AND p.product_name = d.group_name;

    GET DIAGNOSTICS updated_count = ROW_COUNT;

    SELECT COUNT(DISTINCT p.id)
      INTO verified_count
      FROM subscription_plans p
      JOIN _subscription_plan_commercial_copy d
        ON d.tier_name = p.name
       AND d.group_name = p.product_name
      JOIN groups g
        ON g.id = p.group_id
       AND g.name = d.group_name
       AND g.deleted_at IS NULL
     WHERE g.subscription_type = 'subscription'
       AND p.description = d.description
       AND p.features = d.features;

    IF updated_count <> 40 OR verified_count <> 40 THEN
        RAISE EXCEPTION '245: expected 40 refreshed plans, updated %, verified %', updated_count, verified_count;
    END IF;

    SELECT COUNT(*)
      INTO domestic_group_count
      FROM groups g
     WHERE g.name = '国模订阅'
       AND g.subscription_type = 'subscription'
       AND g.deleted_at IS NULL;

    IF domestic_group_count <> 1 THEN
        RAISE EXCEPTION '245: expected one active domestic subscription group, found %', domestic_group_count;
    END IF;

    IF EXISTS (
        SELECT 1
          FROM groups g
         WHERE g.name = '国模订阅'
           AND g.subscription_type = 'subscription'
           AND g.deleted_at IS NULL
           AND jsonb_typeof(g.model_allowlist) <> 'object'
    ) OR EXISTS (
        SELECT 1
          FROM groups g
         WHERE g.name = '国模订阅'
           AND g.subscription_type = 'subscription'
           AND g.deleted_at IS NULL
           AND jsonb_typeof(g.model_allowlist -> 'models') <> 'array'
    ) THEN
        RAISE EXCEPTION '245: domestic subscription model allowlist is not an object with a models array';
    END IF;

    SELECT COUNT(*)
      INTO domestic_qwen_before
      FROM groups g
      CROSS JOIN LATERAL jsonb_array_elements_text(g.model_allowlist -> 'models') AS m(model)
     WHERE g.name = '国模订阅'
       AND g.subscription_type = 'subscription'
       AND g.deleted_at IS NULL
       AND lower(m.model) LIKE 'qwen%';

    IF domestic_qwen_before = 0 THEN
        RAISE NOTICE '245: domestic subscription already has no Qwen models';
    END IF;

    UPDATE groups g
       SET model_allowlist = jsonb_set(
               g.model_allowlist,
               '{models}',
               COALESCE(
                   (
                       SELECT jsonb_agg(m.value ORDER BY m.ordinality)
                         FROM jsonb_array_elements(g.model_allowlist -> 'models') WITH ORDINALITY AS m(value, ordinality)
                        WHERE lower(m.value #>> '{}') NOT LIKE 'qwen%'
                   ),
                   '[]'::jsonb
               ),
               TRUE
           ),
           updated_at = NOW()
     WHERE g.name = '国模订阅'
       AND g.subscription_type = 'subscription'
       AND g.deleted_at IS NULL;

    SELECT COUNT(*)
      INTO domestic_qwen_after
      FROM groups g
      CROSS JOIN LATERAL jsonb_array_elements_text(g.model_allowlist -> 'models') AS m(model)
     WHERE g.name = '国模订阅'
       AND g.subscription_type = 'subscription'
       AND g.deleted_at IS NULL
       AND lower(m.model) LIKE 'qwen%';

    IF domestic_qwen_after <> 0 THEN
        RAISE EXCEPTION '245: domestic subscription still contains % Qwen models', domestic_qwen_after;
    END IF;

    IF EXISTS (
        SELECT 1
          FROM subscription_plans p
          JOIN groups g ON g.id = p.group_id
         WHERE g.subscription_type = 'subscription'
           AND g.name IN (
               'CCMax Claude 订阅', 'Kiro Claude 订阅', 'GPT 订阅', 'Gemini 订阅',
               '国模订阅', 'Grok 订阅', 'GPT Image 2.5 订阅', '香蕉生图订阅'
           )
           AND (
               p.description ~ '(参考市场|上游成本|单价|计费|折|省[约 ]*[0-9]|[$¥￥][0-9])'
               OR p.features ~ '(参考市场|上游成本|单价|计费|折|省[约 ]*[0-9]|[$¥￥][0-9])'
           )
    ) THEN
        RAISE EXCEPTION '245: commercial subscription copy still contains price claims';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM subscription_plans p
          JOIN groups g ON g.id = p.group_id
         WHERE g.subscription_type = 'subscription'
           AND g.name = '国模订阅'
           AND (p.description ILIKE '%qwen%' OR p.features ILIKE '%qwen%')
    ) THEN
        RAISE EXCEPTION '245: domestic subscription copy still contains Qwen';
    END IF;

    RAISE NOTICE '245: refreshed % commercial plan copies and removed % Qwen models from domestic subscription', updated_count, domestic_qwen_before;
END $$;
