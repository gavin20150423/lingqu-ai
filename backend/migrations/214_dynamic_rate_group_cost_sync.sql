-- Dynamic groups use the effective channel cost shown by SubPilot.
-- Keep accounts.rate_multiplier as a fallback for installations without SubPilot tables.

CREATE OR REPLACE FUNCTION dynamic_rate_group_effective_rate(p_account_id BIGINT, p_default NUMERIC)
RETURNS NUMERIC
LANGUAGE plpgsql
AS $$
DECLARE
    configured_rate NUMERIC;
BEGIN
    IF to_regclass('public.subpilot_channel_configs') IS NULL THEN
        RETURN p_default;
    END IF;

    EXECUTE '
        SELECT NULLIF(cost_cny_per_official_usd, 0)::numeric
        FROM subpilot_channel_configs
        WHERE account_id = $1
        LIMIT 1'
    INTO configured_rate
    USING p_account_id::text;

    RETURN COALESCE(configured_rate, p_default);
END;
$$;

CREATE OR REPLACE FUNCTION sync_dynamic_rate_group(p_group_id BIGINT)
RETURNS VOID
LANGUAGE plpgsql
AS $$
BEGIN
    DELETE FROM account_groups ag
    USING groups g, accounts a
    WHERE ag.group_id = g.id
      AND ag.account_id = a.id
      AND g.id = p_group_id
      AND (
          (ag.auto_managed = TRUE AND NOT g.auto_assign_accounts_by_rate)
          OR (
              g.auto_assign_accounts_by_rate = TRUE
              AND (g.auto_assign_max_rate IS NULL
                   OR g.deleted_at IS NOT NULL
                   OR a.deleted_at IS NOT NULL
                   OR a.platform IS DISTINCT FROM g.platform
                   OR dynamic_rate_group_effective_rate(a.id, a.rate_multiplier) > g.auto_assign_max_rate)
          )
      );

    INSERT INTO account_groups (account_id, group_id, priority, created_at, auto_managed)
    SELECT a.id, g.id, 50, NOW(), TRUE
    FROM accounts a
    CROSS JOIN groups g
    WHERE g.id = p_group_id
      AND g.auto_assign_accounts_by_rate = TRUE
      AND g.auto_assign_max_rate IS NOT NULL
      AND g.deleted_at IS NULL
      AND a.deleted_at IS NULL
      AND a.platform = g.platform
      AND dynamic_rate_group_effective_rate(a.id, a.rate_multiplier) <= g.auto_assign_max_rate
    ON CONFLICT (account_id, group_id)
    DO UPDATE SET auto_managed = TRUE;
END;
$$;

CREATE OR REPLACE FUNCTION sync_dynamic_rate_group_account(p_account_id BIGINT)
RETURNS VOID
LANGUAGE plpgsql
AS $$
BEGIN
    DELETE FROM account_groups ag
    USING groups g
    WHERE ag.account_id = p_account_id
      AND ag.group_id = g.id
      AND (
          (ag.auto_managed = TRUE AND NOT g.auto_assign_accounts_by_rate)
          OR (
              g.auto_assign_accounts_by_rate = TRUE
              AND (g.auto_assign_max_rate IS NULL
                   OR g.deleted_at IS NOT NULL
                   OR NOT EXISTS (SELECT 1 FROM accounts a WHERE a.id = p_account_id AND a.deleted_at IS NULL)
                   OR (SELECT a.platform FROM accounts a WHERE a.id = p_account_id) IS DISTINCT FROM g.platform
                   OR dynamic_rate_group_effective_rate(
                         p_account_id,
                         COALESCE((SELECT a.rate_multiplier FROM accounts a WHERE a.id = p_account_id), 1)
                      ) > g.auto_assign_max_rate)
          )
      );

    INSERT INTO account_groups (account_id, group_id, priority, created_at, auto_managed)
    SELECT p_account_id, g.id, 50, NOW(), TRUE
    FROM groups g
    JOIN accounts a ON a.id = p_account_id
    WHERE g.auto_assign_accounts_by_rate = TRUE
      AND g.auto_assign_max_rate IS NOT NULL
      AND g.deleted_at IS NULL
      AND a.deleted_at IS NULL
      AND a.platform = g.platform
      AND dynamic_rate_group_effective_rate(a.id, a.rate_multiplier) <= g.auto_assign_max_rate
    ON CONFLICT (account_id, group_id)
    DO UPDATE SET auto_managed = TRUE;
END;
$$;

CREATE OR REPLACE FUNCTION sync_dynamic_rate_groups_for_account()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM sync_dynamic_rate_group_account(NEW.id);
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION sync_dynamic_rate_groups_for_channel_config()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    target_account_id BIGINT;
BEGIN
    SELECT a.id INTO target_account_id
    FROM accounts a
    WHERE a.id::text = NEW.account_id;
    IF target_account_id IS NOT NULL THEN
        PERFORM sync_dynamic_rate_group_account(target_account_id);
    END IF;
    RETURN NEW;
END;
$$;

DO $$
DECLARE
    group_id BIGINT;
BEGIN
    IF to_regclass('public.subpilot_channel_configs') IS NOT NULL THEN
        EXECUTE 'DROP TRIGGER IF EXISTS trg_sync_dynamic_rate_groups_for_channel_config ON subpilot_channel_configs';
        EXECUTE '
            CREATE TRIGGER trg_sync_dynamic_rate_groups_for_channel_config
            AFTER INSERT OR UPDATE OF cost_cny_per_official_usd
            ON subpilot_channel_configs
            FOR EACH ROW
            EXECUTE FUNCTION sync_dynamic_rate_groups_for_channel_config()';

        FOR group_id IN SELECT id FROM groups WHERE auto_assign_accounts_by_rate AND deleted_at IS NULL LOOP
            PERFORM sync_dynamic_rate_group(group_id);
        END LOOP;
    END IF;
END;
$$;
