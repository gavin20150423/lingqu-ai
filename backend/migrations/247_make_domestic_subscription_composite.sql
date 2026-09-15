-- 国模订阅由多个国产模型组成，应使用 composite 分组并显式路由到
-- OpenAI-compatible 国模上游。套餐售价继续使用 CNY；额度字段仍按 USD
-- 计量，汇率由现有的 1 USD = 1 CNY 约定处理。

DO $$
DECLARE
    target_group_id BIGINT;
BEGIN
    SELECT id
      INTO target_group_id
      FROM groups
     WHERE name = '国模订阅'
       AND subscription_type = 'subscription'
       AND deleted_at IS NULL
     ORDER BY id
     LIMIT 1;

    IF target_group_id IS NULL THEN
        RAISE NOTICE '247: domestic subscription group not present; no-op';
        RETURN;
    END IF;

    UPDATE groups
       SET platform = 'composite', updated_at = NOW()
     WHERE id = target_group_id
       AND platform IS DISTINCT FROM 'composite';

    INSERT INTO composite_model_routes (
        group_id, public_model, match_type, target_platform, upstream_model,
        endpoint, priority, enabled, notes, created_at, updated_at
    )
    SELECT target_group_id, model_name, 'exact', 'openai', model_name,
           'any', 100, TRUE, '国模订阅：OpenAI 兼容国模上游', NOW(), NOW()
      FROM unnest(ARRAY[
          'glm-5.3',
          'glm-5.3-flash',
          'deepseek-v4-flash',
          'deepseek-v4-pro',
          'DeepSeek-V4.1-Flash',
          'kimi-k3',
          'MiniMax-M3'
      ]) AS models(model_name)
     WHERE NOT EXISTS (
         SELECT 1
           FROM composite_model_routes r
          WHERE r.group_id = target_group_id
            AND r.public_model = model_name
            AND r.match_type = 'exact'
            AND r.target_platform = 'openai'
            AND r.deleted_at IS NULL
     );
END $$;
