-- 为国模订阅创建独立渠道定价。
-- 渠道定价表的价格字段按每 token 保存；当前业务约定 1 USD = 1 CNY，
-- 因此沿用上游公开的数值口径。国模上游是 OpenAI-compatible，Claude Code
-- 通过本系统 /v1/messages 进入后，再由 composite 路由转换到该上游。

DO $$
DECLARE
    domestic_group_id BIGINT;
    domestic_channel_id BIGINT;
BEGIN
    SELECT id INTO domestic_group_id
      FROM groups
     WHERE name = '国模订阅'
       AND subscription_type = 'subscription'
       AND deleted_at IS NULL
     ORDER BY id
     LIMIT 1;

    IF domestic_group_id IS NULL THEN
        RAISE NOTICE '248: domestic subscription group not present; no-op';
        RETURN;
    END IF;

    UPDATE groups
       SET platform = 'composite', allow_messages_dispatch = TRUE, updated_at = NOW()
     WHERE id = domestic_group_id;

    INSERT INTO channels (
        name, description, status, billing_model_source, restrict_models,
        features, features_config
    ) VALUES (
        '国模', 'GLM / DeepSeek / Kimi / MiniMax OpenAI 兼容渠道',
        'active', 'channel_mapped', TRUE, '', '{}'::jsonb
    )
    ON CONFLICT (name) DO UPDATE SET
        description = EXCLUDED.description,
        status = 'active',
        billing_model_source = 'channel_mapped',
        restrict_models = TRUE,
        updated_at = NOW();

    SELECT id INTO domestic_channel_id FROM channels WHERE name = '国模';

    INSERT INTO channel_groups (channel_id, group_id)
    VALUES (domestic_channel_id, domestic_group_id)
    ON CONFLICT (group_id) DO UPDATE SET channel_id = EXCLUDED.channel_id;

    DELETE FROM channel_model_pricing WHERE channel_id = domestic_channel_id;

    INSERT INTO channel_model_pricing (
        channel_id, platform, models, billing_mode,
        input_price, output_price, cache_read_price
    ) VALUES
        (domestic_channel_id, 'openai', '["glm-5.3"]'::jsonb, 'token', 0.0000014, 0.0000044, 0.00000026),
        (domestic_channel_id, 'openai', '["glm-5.3-flash"]'::jsonb, 'token', 0.00000015, 0.00000050, 0.00000003),
        (domestic_channel_id, 'openai', '["deepseek-v4-flash"]'::jsonb, 'token', 0.00000022, 0.00000066, 0.000000007),
        (domestic_channel_id, 'openai', '["deepseek-v4-pro"]'::jsonb, 'token', 0.00000066, 0.00000198, 0.000000022),
        (domestic_channel_id, 'openai', '["DeepSeek-V4.1-Flash"]'::jsonb, 'token', 0.00000022, 0.00000066, 0.000000007),
        (domestic_channel_id, 'openai', '["kimi-k3"]'::jsonb, 'token', 0.00000300, 0.00001500, 0.00000030),
        (domestic_channel_id, 'openai', '["MiniMax-M3"]'::jsonb, 'token', 0.00000060, 0.00000240, 0.00000012);
END $$;
