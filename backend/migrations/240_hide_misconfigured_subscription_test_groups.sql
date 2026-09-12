-- 240: 下架本次误建的测试订阅分组。
--
-- 订阅商品由 239_seed_subscription_catalog_v2.sql 创建；以下名称是此前
-- 本地/线上验证时误建的临时分组，不属于正式目录。只在确认没有用户订阅、
-- API Key、用量或用户白名单引用时软删除，避免影响任何已产生业务数据。

DO $$
DECLARE
    group_id_value BIGINT;
    group_name_value TEXT;
    protected_count BIGINT;
BEGIN
    FOR group_id_value, group_name_value IN
        SELECT g.id, g.name
        FROM groups g
        WHERE g.deleted_at IS NULL
          AND g.name IN (
              '【本地测试】CCMax 订阅',
              '【本地测试】Kiro 订阅',
              '【本地测试】GPT 订阅',
              'GPT 订阅',
              'CCMax 订阅',
              'Kiro 订阅',
              '国模 gm x0.35'
          )
    LOOP
        -- 239 使用这三个名称作为正式目录组。只要迁移已经写入在售套餐，
        -- 必须跳过，避免把正式资源组误当成旧测试组下架。
        IF group_name_value IN ('GPT 订阅', 'CCMax 订阅', 'Kiro 订阅') AND EXISTS (
            SELECT 1 FROM subscription_plans p
            WHERE p.group_id = group_id_value AND p.for_sale = TRUE
        ) THEN
            CONTINUE;
        END IF;

        SELECT COUNT(*) INTO protected_count
        FROM (
            SELECT 1 FROM api_keys k WHERE k.group_id = group_id_value AND k.deleted_at IS NULL
            UNION ALL
            SELECT 1 FROM user_subscriptions s WHERE s.group_id = group_id_value AND s.deleted_at IS NULL
            UNION ALL
            SELECT 1 FROM usage_logs u WHERE u.group_id = group_id_value
            UNION ALL
            SELECT 1 FROM user_allowed_groups ug WHERE ug.group_id = group_id_value
        ) protected_rows;

        IF protected_count > 0 THEN
            -- 历史测试数据可能已经产生订阅或用量。保护这些数据，跳过
            -- 下架当前分组，但不要让整批正式订阅目录迁移失败。
            RAISE NOTICE 'keeping subscription test group with live references: % (id=%)', group_name_value, group_id_value;
            CONTINUE;
        END IF;

        UPDATE subscription_plans
        SET for_sale = FALSE, updated_at = NOW()
        WHERE group_id = group_id_value;

        -- 239 已把这些测试组的账号复制到正式资源组；删除旧的中间表绑定，
        -- 避免软删除后的测试组继续出现在账号统计或调度候选中。
        DELETE FROM account_groups WHERE group_id = group_id_value;

        UPDATE groups
        SET status = 'inactive', deleted_at = NOW(), updated_at = NOW()
        WHERE id = group_id_value AND deleted_at IS NULL;
    END LOOP;
END $$;
