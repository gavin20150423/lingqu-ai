-- 243: Treat the formal subscription catalog prices as RMB.
--
-- The price field is the sale price of a plan. The quota fields retain their
-- USD meaning and are intentionally not changed here. This migration only
-- touches the eight formal subscription series created by the catalog setup.
-- A zero-match no-op keeps older local fixtures bootable; a partial catalog
-- fails closed instead of updating an incomplete production catalog.

DO $$
DECLARE
    target_count INTEGER;
    cny_count    INTEGER;
BEGIN
    SELECT COUNT(*)
      INTO target_count
      FROM subscription_plans p
      JOIN groups g ON g.id = p.group_id
     WHERE g.subscription_type = 'subscription'
       AND g.name IN (
           'CCMax Claude 订阅',
           'Kiro Claude 订阅',
           'GPT 订阅',
           'Gemini 订阅',
           '国模订阅',
           'Grok 订阅',
           'GPT Image 2.5 订阅',
           '香蕉生图订阅'
       );

    IF target_count = 0 THEN
        RAISE NOTICE '243: formal subscription catalog not present; no-op';
        RETURN;
    END IF;

    IF target_count <> 40 THEN
        RAISE EXCEPTION '243: expected 40 formal subscription plans, found %', target_count;
    END IF;

    UPDATE subscription_plans p
       SET currency = 'CNY', updated_at = NOW()
      FROM groups g
     WHERE g.id = p.group_id
       AND g.subscription_type = 'subscription'
       AND g.name IN (
           'CCMax Claude 订阅',
           'Kiro Claude 订阅',
           'GPT 订阅',
           'Gemini 订阅',
           '国模订阅',
           'Grok 订阅',
           'GPT Image 2.5 订阅',
           '香蕉生图订阅'
       )
       AND p.currency IS DISTINCT FROM 'CNY';

    SELECT COUNT(*)
      INTO cny_count
      FROM subscription_plans p
      JOIN groups g ON g.id = p.group_id
     WHERE g.subscription_type = 'subscription'
       AND g.name IN (
           'CCMax Claude 订阅',
           'Kiro Claude 订阅',
           'GPT 订阅',
           'Gemini 订阅',
           '国模订阅',
           'Grok 订阅',
           'GPT Image 2.5 订阅',
           '香蕉生图订阅'
       )
       AND p.currency = 'CNY';

    IF cny_count <> 40 THEN
        RAISE EXCEPTION '243: expected 40 CNY subscription plans after update, found %', cny_count;
    END IF;
END $$;
