-- CCMax 订阅单价：0.90 元/刀 → 1.30 元/刀（只调售价，日/周/月额度上限不变）。
--
-- 口径：订阅单价 = price / monthly_limit_usd（元/刀），原有 5 档全部精确为 0.90：
--   Basic 270/300、Plus 576/640、Standard 1350/1500、Pro 2475/2750、Ultra 5400/6000。
-- 调整后：390/300、832/640、1950/1500、3575/2750、7800/6000，单价均 1.30。
--
-- 幂等：仅当当前单价仍为 0.90 时才改，避免本迁移在后续人工再次调价后被重放覆盖。
UPDATE subscription_plans p
SET price = round(p.monthly_limit_usd * 1.30, 2),
    updated_at = NOW()
WHERE p.monthly_limit_usd > 0
  AND round(p.price / p.monthly_limit_usd, 4) = 0.9000
  AND p.group_id IN (
      SELECT g.id FROM groups g WHERE g.name = 'CCMax Claude 订阅'
  );
