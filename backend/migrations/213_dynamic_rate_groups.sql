-- Dynamic account groups: keep system-managed bindings in sync with account pricing.
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS auto_assign_accounts_by_rate BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS auto_assign_max_rate DECIMAL(10, 4);
ALTER TABLE account_groups
    ADD COLUMN IF NOT EXISTS auto_managed BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_groups_auto_rate
    ON groups (platform, auto_assign_accounts_by_rate, auto_assign_max_rate)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_account_groups_auto_managed
    ON account_groups (group_id, account_id)
    WHERE auto_managed = TRUE;

CREATE OR REPLACE FUNCTION sync_dynamic_rate_group(p_group_id BIGINT)
RETURNS VOID
LANGUAGE plpgsql
AS $$
BEGIN
    -- Once enabled, a dynamic group is fully rule-managed. When the rule is
    -- disabled, only bindings created by the rule are removed.
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
                   OR a.rate_multiplier > g.auto_assign_max_rate)
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
      AND a.rate_multiplier <= g.auto_assign_max_rate
    ON CONFLICT (account_id, group_id)
    DO UPDATE SET auto_managed = TRUE;
END;
$$;

CREATE OR REPLACE FUNCTION sync_dynamic_rate_group_trigger()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM sync_dynamic_rate_group(NEW.id);
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION sync_dynamic_rate_groups_for_account()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    DELETE FROM account_groups ag
    USING groups g
    WHERE ag.account_id = NEW.id
      AND ag.group_id = g.id
      AND (
          (ag.auto_managed = TRUE AND NOT g.auto_assign_accounts_by_rate)
          OR (
              g.auto_assign_accounts_by_rate = TRUE
              AND (g.auto_assign_max_rate IS NULL
                   OR g.deleted_at IS NOT NULL
                   OR NEW.deleted_at IS NOT NULL
                   OR NEW.platform IS DISTINCT FROM g.platform
                   OR NEW.rate_multiplier > g.auto_assign_max_rate)
          )
      );

    INSERT INTO account_groups (account_id, group_id, priority, created_at, auto_managed)
    SELECT NEW.id, g.id, 50, NOW(), TRUE
    FROM groups g
    WHERE g.auto_assign_accounts_by_rate = TRUE
      AND g.auto_assign_max_rate IS NOT NULL
      AND g.deleted_at IS NULL
      AND NEW.deleted_at IS NULL
      AND NEW.platform = g.platform
      AND NEW.rate_multiplier <= g.auto_assign_max_rate
    ON CONFLICT (account_id, group_id)
    DO UPDATE SET auto_managed = TRUE;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION resync_dynamic_rate_group_binding()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    target_group_id BIGINT;
BEGIN
    -- sync_dynamic_rate_group itself writes account_groups. Avoid recursive
    -- resync while still enforcing the rule for external/manual mutations.
    IF pg_trigger_depth() > 1 THEN
        IF TG_OP = 'DELETE' THEN
            RETURN OLD;
        END IF;
        RETURN NEW;
    END IF;
    target_group_id := CASE WHEN TG_OP = 'DELETE' THEN OLD.group_id ELSE NEW.group_id END;
    IF TG_OP = 'UPDATE' AND OLD.group_id IS DISTINCT FROM NEW.group_id THEN
        PERFORM sync_dynamic_rate_group(OLD.group_id);
    END IF;
    PERFORM sync_dynamic_rate_group(target_group_id);
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_sync_dynamic_rate_group ON groups;
CREATE TRIGGER trg_sync_dynamic_rate_group
AFTER INSERT OR UPDATE OF platform, auto_assign_accounts_by_rate, auto_assign_max_rate, deleted_at
ON groups
FOR EACH ROW
EXECUTE FUNCTION sync_dynamic_rate_group_trigger();

DROP TRIGGER IF EXISTS trg_sync_dynamic_rate_groups_for_account ON accounts;
CREATE TRIGGER trg_sync_dynamic_rate_groups_for_account
AFTER INSERT OR UPDATE OF platform, rate_multiplier, deleted_at
ON accounts
FOR EACH ROW
EXECUTE FUNCTION sync_dynamic_rate_groups_for_account();

DROP TRIGGER IF EXISTS trg_resync_dynamic_rate_group_binding ON account_groups;
CREATE TRIGGER trg_resync_dynamic_rate_group_binding
AFTER INSERT OR UPDATE OF account_id, group_id OR DELETE
ON account_groups
FOR EACH ROW
EXECUTE FUNCTION resync_dynamic_rate_group_binding();

-- Backfill groups that were configured before this migration was installed.
DO $$
DECLARE g_id BIGINT;
BEGIN
    FOR g_id IN SELECT id FROM groups WHERE auto_assign_accounts_by_rate LOOP
        PERFORM sync_dynamic_rate_group(g_id);
    END LOOP;
END;
$$;
