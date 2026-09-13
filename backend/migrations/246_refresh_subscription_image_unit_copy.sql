-- 246: Explain the per-image quota for the two image subscription series.
--
-- Sale prices remain stored in the plan currency (CNY). The dollar amounts
-- below describe the USD quota consumed by one generated image and do not
-- change plan prices, subscription balances, orders, or model routing.

-- Publish the customer-facing names requested for the two image products.
-- Refuse an ambiguous rename instead of merging two resource groups.
DO $$
DECLARE
    rename_pair RECORD;
    old_group_id BIGINT;
    new_group_id BIGINT;
BEGIN
    FOR rename_pair IN
        SELECT * FROM (VALUES
            ('GPT Image 2.5 订阅'::TEXT, 'GPT Image订阅'::TEXT),
            ('香蕉生图订阅'::TEXT, 'nano banana订阅'::TEXT)
        ) AS pairs(old_name, new_name)
    LOOP
        SELECT id INTO old_group_id
          FROM groups
         WHERE name = rename_pair.old_name
           AND subscription_type = 'subscription'
           AND deleted_at IS NULL
         ORDER BY id
         LIMIT 1;
        SELECT id INTO new_group_id
          FROM groups
         WHERE name = rename_pair.new_name
           AND subscription_type = 'subscription'
           AND deleted_at IS NULL
         ORDER BY id
         LIMIT 1;

        IF old_group_id IS NOT NULL AND new_group_id IS NOT NULL THEN
            RAISE EXCEPTION '246: both old and new image subscription groups exist: % / %',
                rename_pair.old_name, rename_pair.new_name;
        END IF;
        IF old_group_id IS NOT NULL THEN
            UPDATE groups
               SET name = rename_pair.new_name,
                   updated_at = NOW()
             WHERE id = old_group_id;
        END IF;
    END LOOP;
END $$;

CREATE TEMP TABLE _subscription_image_unit_copy (
    group_name  TEXT NOT NULL,
    tier_name   TEXT NOT NULL,
    description TEXT NOT NULL,
    features    TEXT NOT NULL,
    PRIMARY KEY (group_name, tier_name)
);

INSERT INTO _subscription_image_unit_copy (group_name, tier_name, description, features)
VALUES
    (
        'GPT Image订阅',
        'Basic',
        '轻量图片创作入门档，适合灵感草图、头像尝试和偶发生图；每张生成消耗 $0.08 额度。',
        E'适用场景：灵感草图、头像尝试与偶发创作\n核心优势：两个 GPT Image 模型可选，创作更灵活\n支持模型：GPT Image 2.5 系列\n单张生成额度：$0.08 / 张；每月 500 张，日 / 周 / 月独立限额'
    ),
    (
        'GPT Image订阅',
        'Plus',
        '稳定图片创作进阶档，适合社交内容、产品概念和日常视觉素材；每张生成消耗 $0.08 额度。',
        E'适用场景：社交内容、产品概念与日常视觉素材\n核心优势：适合规律创作，满足日常素材需求\n支持模型：GPT Image 2.5 系列\n单张生成额度：$0.08 / 张；每月 1,500 张，日 / 周 / 月独立限额'
    ),
    (
        'GPT Image订阅',
        'Standard',
        '日常批量生图主力档，适合内容团队、营销素材和连续创作；每张生成消耗 $0.08 额度。',
        E'适用场景：营销素材、内容团队与日常批量生图\n核心优势：生成空间充足，适合稳定批量生产\n支持模型：GPT Image 2.5 系列\n单张生成额度：$0.08 / 张；每月 3,000 张，日 / 周 / 月独立限额'
    ),
    (
        'GPT Image订阅',
        'Pro',
        '专业图片生产档，适合活动物料、产品批次和高频内容团队；每张生成消耗 $0.06 额度。',
        E'适用场景：活动物料、产品批次与高频内容生产\n核心优势：适合专业批量工作，连续产出更从容\n支持模型：GPT Image 2.5 系列\n单张生成额度：$0.06 / 张；每月 8,000 张，日 / 周 / 月独立限额'
    ),
    (
        'GPT Image订阅',
        'Ultra',
        '商业与团队级生图旗舰档，适合广告批量制作、素材库和持续项目；每张生成消耗 $0.05 额度。',
        E'适用场景：广告批量制作、素材库与持续商业项目\n核心优势：生成空间充足，适合大规模视觉生产\n支持模型：GPT Image 2.5 系列\n单张生成额度：$0.05 / 张；每月 15,000 张，日 / 周 / 月独立限额'
    ),
    (
        'nano banana订阅',
        'Basic',
        'Nano Banana 轻量创作入门档，适合灵感尝试、社交配图和偶发图片生成；每张生成消耗 $0.15 额度。',
        E'适用场景：灵感尝试、社交配图与偶发创作\n核心优势：轻量灵活，适合快速验证视觉想法\n支持模型：Nano Banana 生图系列\n单张生成额度：$0.15 / 张；每月 500 张，日 / 周 / 月独立限额'
    ),
    (
        'nano banana订阅',
        'Plus',
        '稳定多模态创作进阶档，适合日常内容、文章配图和小批量素材；每张生成消耗 $0.15 额度。',
        E'适用场景：日常内容、文章配图与小批量素材\n核心优势：生成节奏稳定，覆盖常规创作需求\n支持模型：Nano Banana 生图系列\n单张生成额度：$0.15 / 张；每月 1,500 张，日 / 周 / 月独立限额'
    ),
    (
        'nano banana订阅',
        'Standard',
        '主力内容生产档，适合账号运营、营销配图和批量创作；每张生成消耗 $0.13 额度。',
        E'适用场景：账号运营、营销配图与批量创作\n核心优势：生成空间充足，适合持续内容生产\n支持模型：Nano Banana 生图系列\n单张生成额度：$0.13 / 张；每月 3,000 张，日 / 周 / 月独立限额'
    ),
    (
        'nano banana订阅',
        'Pro',
        '专业视觉生产档，适合活动系列、品牌内容和高频素材制作；每张生成消耗 $0.12 额度。',
        E'适用场景：活动系列、品牌内容与高频素材制作\n核心优势：适合专业创作，连续产出更高效\n支持模型：Nano Banana 生图系列\n单张生成额度：$0.12 / 张；每月 5,000 张，日 / 周 / 月独立限额'
    ),
    (
        'nano banana订阅',
        'Ultra',
        '团队与规模化生图旗舰档，适合广告素材库、矩阵运营和持续商业项目；每张生成消耗 $0.10 额度。',
        E'适用场景：广告素材库、矩阵运营与持续商业项目\n核心优势：生成空间充足，适合规模化视觉生产\n支持模型：Nano Banana 生图系列\n单张生成额度：$0.10 / 张；每月 10,000 张，日 / 周 / 月独立限额'
    );

DO $$
DECLARE
    matched_count  INTEGER;
    updated_count  INTEGER;
    verified_count INTEGER;
BEGIN
    SELECT COUNT(DISTINCT p.id)
      INTO matched_count
      FROM subscription_plans p
      JOIN _subscription_image_unit_copy d
        ON d.tier_name = p.name
      JOIN groups g
        ON g.id = p.group_id
       AND g.name = d.group_name
       AND g.subscription_type = 'subscription'
       AND g.deleted_at IS NULL;

    IF matched_count = 0 THEN
        RAISE NOTICE '246: image subscription catalog not present; no-op';
        RETURN;
    END IF;

    IF matched_count <> 10 THEN
        RAISE EXCEPTION '246: expected 10 image subscription plans, found %', matched_count;
    END IF;

    UPDATE subscription_plans p
       SET description = d.description,
           features = d.features,
           product_name = d.group_name,
           updated_at = NOW()
      FROM _subscription_image_unit_copy d
      JOIN groups g
        ON g.name = d.group_name
       AND g.subscription_type = 'subscription'
       AND g.deleted_at IS NULL
     WHERE p.group_id = g.id
       AND p.name = d.tier_name;

    GET DIAGNOSTICS updated_count = ROW_COUNT;

    SELECT COUNT(DISTINCT p.id)
      INTO verified_count
      FROM subscription_plans p
      JOIN _subscription_image_unit_copy d
        ON d.tier_name = p.name
       AND p.description = d.description
       AND p.features = d.features
      JOIN groups g
        ON g.id = p.group_id
       AND g.name = d.group_name
       AND g.subscription_type = 'subscription'
       AND g.deleted_at IS NULL;

    IF updated_count <> 10 OR verified_count <> 10 THEN
        RAISE EXCEPTION '246: expected 10 refreshed image plans, updated %, verified %', updated_count, verified_count;
    END IF;
END $$;
