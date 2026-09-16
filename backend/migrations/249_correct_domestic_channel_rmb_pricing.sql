-- 修正国模渠道定价：美元数值按 1 USD = 1 CNY 直接使用，不做汇率除法。
-- DeepSeek 采用官方峰谷规则：工作日 UTC 01:00-04:00、06:00-10:00
-- 为高峰时段，倍率 2；周末全天低谷。基础价格单位为每 token。

DO $$
DECLARE
    domestic_channel_id BIGINT;
    deepseek_time_pricing JSONB := '{
      "timezone": "Asia/Shanghai",
      "weekdays_only": true,
      "periods": [
        {"start_time": "09:00", "end_time": "12:00", "multiplier": 2},
        {"start_time": "14:00", "end_time": "18:00", "multiplier": 2}
      ]
    }'::jsonb;
BEGIN
    SELECT id INTO domestic_channel_id FROM channels WHERE name = '国模' ORDER BY id LIMIT 1;
    IF domestic_channel_id IS NULL THEN
        RAISE NOTICE '249: domestic channel not present; no-op';
        RETURN;
    END IF;

    -- 统一所有已配置模型的人民币口径；平台保持 openai，因 composite 路由的
    -- target_platform 为 openai，Claude Code 请求会先由 messages dispatch 转换。
    UPDATE channel_model_pricing
       SET input_price = CASE
             WHEN models @> '["glm-5.3"]'::jsonb THEN 0.0000014
             WHEN models @> '["glm-5.3-flash"]'::jsonb THEN 0.00000015
             WHEN models @> '["deepseek-v4-flash"]'::jsonb OR models @> '["DeepSeek-V4.1-Flash"]'::jsonb THEN 0.00000015
             WHEN models @> '["deepseek-v4-pro"]'::jsonb THEN 0.00000066
             WHEN models @> '["kimi-k3"]'::jsonb THEN 0.000003
             WHEN models @> '["MiniMax-M3"]'::jsonb THEN 0.00000060
             ELSE input_price END,
           output_price = CASE
             WHEN models @> '["glm-5.3"]'::jsonb THEN 0.0000044
             WHEN models @> '["glm-5.3-flash"]'::jsonb THEN 0.00000050
             WHEN models @> '["deepseek-v4-flash"]'::jsonb OR models @> '["DeepSeek-V4.1-Flash"]'::jsonb THEN 0.00000060
             WHEN models @> '["deepseek-v4-pro"]'::jsonb THEN 0.00000198
             WHEN models @> '["kimi-k3"]'::jsonb THEN 0.000015
             WHEN models @> '["MiniMax-M3"]'::jsonb THEN 0.00000240
             ELSE output_price END,
           cache_read_price = CASE
             WHEN models @> '["glm-5.3"]'::jsonb THEN 0.00000026
             WHEN models @> '["glm-5.3-flash"]'::jsonb THEN 0.00000003
             WHEN models @> '["deepseek-v4-flash"]'::jsonb OR models @> '["DeepSeek-V4.1-Flash"]'::jsonb THEN 0.000000003
             WHEN models @> '["deepseek-v4-pro"]'::jsonb THEN 0.000000022
             WHEN models @> '["kimi-k3"]'::jsonb THEN 0.00000030
             WHEN models @> '["MiniMax-M3"]'::jsonb THEN 0.00000012
             ELSE cache_read_price END,
           time_pricing = CASE
             WHEN models @> '["deepseek-v4-flash"]'::jsonb
               OR models @> '["DeepSeek-V4.1-Flash"]'::jsonb
               OR models @> '["deepseek-v4-pro"]'::jsonb
             THEN deepseek_time_pricing
             ELSE NULL END,
           updated_at = NOW()
     WHERE channel_id = domestic_channel_id;
END $$;
