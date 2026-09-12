<template>
  <article
    :class="[
      'subscription-plan-card',
      { 'subscription-plan-card--featured': featured, 'subscription-plan-card--renewal': isRenewal },
      platformClass,
    ]"
  >
    <div v-if="featured" class="subscription-plan-card__ribbon">
      <Icon name="sparkles" size="xs" /> 推荐档位
    </div>

    <div class="subscription-plan-card__head">
      <div class="subscription-plan-card__identity">
        <span class="subscription-plan-card__dot" aria-hidden="true" />
        <span class="subscription-plan-card__platform">{{ pLabel }}</span>
      </div>
      <span v-if="isRenewal" class="subscription-plan-card__renewal">可续费</span>
      <h3 :title="plan.name">{{ plan.name }}</h3>
      <p v-if="plan.description">{{ plan.description }}</p>
    </div>

    <div class="subscription-plan-card__price">
      <span class="subscription-plan-card__currency">{{ planCurrencySymbol }}</span>
      <strong>{{ formattedPrice }}</strong>
      <span class="subscription-plan-card__period">/ {{ validitySuffix }}</span>
    </div>
    <div v-if="plan.original_price" class="subscription-plan-card__original">
      原价 {{ planCurrencySymbol }}{{ formatNumber(plan.original_price) }}
      <span>{{ discountText }}</span>
    </div>

    <div class="subscription-plan-card__metrics">
      <div>
        <span>月可用额度</span>
        <strong>{{ quotaTotal > 0 ? `$${quotaTotal.toFixed(2)}` : '按量使用' }}</strong>
      </div>
      <div>
        <span>有效期</span>
        <strong>{{ validitySuffix }}</strong>
      </div>
    </div>

    <div class="subscription-plan-card__limits" aria-label="额度限制">
      <div v-for="limit in quotaLimits" :key="limit.label">
        <span>{{ limit.label }}</span>
        <strong>{{ limit.value }}</strong>
      </div>
    </div>

    <div class="subscription-plan-card__entitlement">
      <div class="subscription-plan-card__entitlement-title">
        <Icon name="gift" size="sm" />
        <span>套餐权益</span>
      </div>
      <div class="subscription-plan-card__entitlement-value">
        {{ entitlementSummary }}
      </div>
    </div>

    <div v-if="modelScopeLabels.length > 0" class="subscription-plan-card__scopes">
      <span>模型范围</span>
      <div>
        <span v-for="scope in modelScopeLabels" :key="scope">{{ scope }}</span>
      </div>
    </div>

    <ul class="subscription-plan-card__features">
      <li v-for="feature in featureItems" :key="feature">
        <Icon name="check" size="xs" />
        <span>{{ feature }}</span>
      </li>
    </ul>

    <div class="subscription-plan-card__footer">
      <div class="subscription-plan-card__availability">
        <span>{{ plan.for_sale ? '可购买' : '暂停售卖' }}</span>
        <strong>{{ plan.for_sale ? '立即开通' : '暂不可用' }}</strong>
      </div>
      <div class="subscription-plan-card__progress" aria-hidden="true">
        <span :class="{ 'subscription-plan-card__progress--off': !plan.for_sale }" />
      </div>
      <button type="button" :disabled="!plan.for_sale" @click="emit('select', plan)">
        <Icon name="arrowRight" size="sm" />
        {{ isRenewal ? t('payment.renewNow') : '立即订阅' }}
      </button>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SubscriptionPlan } from '@/types/payment'
import type { UserSubscription } from '@/types'
import { currencySymbol } from '@/components/payment/currency'
import { planValiditySuffix } from './validity'
import { platformLabel } from '@/utils/platformColors'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(defineProps<{
  plan: SubscriptionPlan
  activeSubscriptions?: UserSubscription[]
  featured?: boolean
}>(), {
  activeSubscriptions: () => [],
  featured: false,
})

const emit = defineEmits<{ select: [plan: SubscriptionPlan] }>()
const { t } = useI18n()

const platform = computed(() => props.plan.group_platform || '')
const productTone = computed(() => {
  const name = `${props.plan.group_name || ''} ${props.plan.name || ''}`.toLowerCase()
  if (name.includes('kiro')) return 'kiro'
  return platform.value || 'default'
})
const platformClass = computed(() => `subscription-plan-card--${productTone.value}`)
const pLabel = computed(() => {
  const groupName = props.plan.group_name?.replace(/^【[^】]+】\s*/, '').replace(/\s*订阅\s*$/, '').trim()
  return groupName || platformLabel(platform.value)
})
// Subscription prices are stored as USD. The checkout page converts them only
// when the selected gateway uses CNY and the admin has enabled that rate.
const planCurrencySymbol = computed(() => currencySymbol(props.plan.currency || 'USD'))
const formattedPrice = computed(() => formatNumber(props.plan.price))
const validitySuffix = computed(() => planValiditySuffix(props.plan, t))
const isRenewal = computed(() => props.activeSubscriptions.some(s => s.group_id === props.plan.group_id && s.status === 'active'))

const quotaEntries = computed(() => Object.entries(props.plan.entitlements || {}).filter(([, value]) => Number(value) > 0))
// New plans use the plan-level rolling limits. Legacy plans may still expose
// gifted entitlements instead, so retain that display as a fallback.
const quotaTotal = computed(() => {
  const entitlementTotal = quotaEntries.value.reduce((sum, [, value]) => sum + Number(value), 0)
  if (entitlementTotal > 0) return entitlementTotal
  const monthlyLimit = Number(props.plan.monthly_limit_usd ?? 0)
  return Number.isFinite(monthlyLimit) && monthlyLimit > 0 ? monthlyLimit : 0
})
const hasPlanQuota = computed(() => [
  props.plan.daily_limit_usd,
  props.plan.weekly_limit_usd,
  props.plan.monthly_limit_usd,
].some(value => value != null && Number(value) > 0))
const quotaLimits = computed(() => [
  { label: '日限额', value: formatQuota(props.plan.daily_limit_usd) },
  { label: '周限额', value: formatQuota(props.plan.weekly_limit_usd) },
  { label: '月限额', value: formatQuota(props.plan.monthly_limit_usd) },
])
const entitlementSummary = computed(() => {
  if (quotaEntries.value.length === 0) {
    return hasPlanQuota.value ? '日 / 周 / 月独立限额，按套餐额度扣减' : '按分组倍率计费'
  }
  return quotaEntries.value.map(([key, value]) => `${key} $${Number(value).toFixed(2)}`).join(' · ')
})

const MODEL_SCOPE_LABELS: Record<string, string> = {
  claude: 'Claude',
  gemini_text: 'Gemini',
  gemini_image: 'Imagen',
}

const modelScopeLabels = computed(() => {
  if (platform.value !== 'antigravity') return []
  return (props.plan.supported_model_scopes || [])
    .map(scope => MODEL_SCOPE_LABELS[scope] || scope)
    .filter(Boolean)
})

const featureItems = computed(() => {
  const generated = [
    `支持 ${pLabel.value} 模型`,
    hasPlanQuota.value
      ? '套餐额度独立扣减'
      : `调用倍率 ×${Number(props.plan.rate_multiplier ?? 1).toPrecision(4).replace(/\.0+$/, '')}`,
    `有效期 ${validitySuffix.value}`,
  ]
  const supplied = (props.plan.features || [])
    .flatMap(feature => String(feature).split(/\\n|\n/))
    .map(feature => feature.trim())
    .filter(Boolean)
  return [...generated, ...supplied].filter((item, index, all) => all.indexOf(item) === index).slice(0, 6)
})

const discountText = computed(() => {
  if (!props.plan.original_price || props.plan.original_price <= props.plan.price) return ''
  return `省 ${Math.round((1 - props.plan.price / props.plan.original_price) * 100)}%`
})

function formatNumber(value: number): string {
  return Number(value).toFixed(2)
}

function formatQuota(value: number | null | undefined): string {
  return value != null && Number(value) > 0 ? `$${formatNumber(value)}` : '不限'
}
</script>

<style scoped>
.subscription-plan-card {
  position: relative;
  display: flex;
  min-width: 0;
  min-height: 33rem;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid #ddd9d2;
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 7px 20px rgba(56, 49, 41, 0.06);
  transition: transform 180ms ease, box-shadow 180ms ease, border-color 180ms ease;
}

.subscription-plan-card:hover {
  border-color: var(--plan-accent, #f97316);
  box-shadow: 0 14px 30px rgba(56, 49, 41, 0.12);
  transform: translateY(-3px);
}

.subscription-plan-card--featured {
  border: 2px solid var(--plan-accent, #f97316);
  box-shadow: 0 13px 32px color-mix(in srgb, var(--plan-accent, #f97316) 17%, transparent);
}

.subscription-plan-card--featured::before {
  position: absolute;
  inset: 0 0 auto;
  height: 4px;
  background: var(--plan-accent, #f97316);
  content: '';
}

.subscription-plan-card--anthropic { --plan-accent: #f97316; --plan-soft: #fff2e8; --plan-text: #c2410c; }
.subscription-plan-card--kiro { --plan-accent: #6366f1; --plan-soft: #eef0ff; --plan-text: #4338ca; }
.subscription-plan-card--openai { --plan-accent: #22a866; --plan-soft: #eaf8f0; --plan-text: #167344; }
.subscription-plan-card--antigravity { --plan-accent: #8b5cf6; --plan-soft: #f2edff; --plan-text: #6d28d9; }
.subscription-plan-card--gemini { --plan-accent: #3b82f6; --plan-soft: #edf4ff; --plan-text: #1d4ed8; }
.subscription-plan-card--default { --plan-accent: #0f9f9a; --plan-soft: #e7f8f6; --plan-text: #08736e; }

.subscription-plan-card__ribbon {
  position: absolute;
  top: 0.75rem;
  right: 0.75rem;
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  border-radius: 999px;
  background: var(--plan-accent, #f97316);
  color: #fff;
  font-size: 0.66rem;
  font-weight: 800;
  padding: 0.28rem 0.55rem;
}

.subscription-plan-card__head { padding: 1.25rem 1.25rem 0.5rem; }
.subscription-plan-card__identity { display: flex; align-items: center; gap: 0.45rem; color: var(--plan-text, #08736e); font-size: 0.72rem; font-weight: 800; letter-spacing: 0.02em; text-transform: uppercase; }
.subscription-plan-card__dot { width: 0.48rem; height: 0.48rem; flex: 0 0 auto; border-radius: 50%; background: var(--plan-accent, #0f9f9a); }
.subscription-plan-card__renewal { float: right; border-radius: 999px; background: #f0fdf4; color: #15803d; font-size: 0.66rem; font-weight: 700; padding: 0.2rem 0.45rem; }
.subscription-plan-card__head h3 { margin-top: 0.65rem; color: #292622; font-size: 1.13rem; font-weight: 850; line-height: 1.25; }
.subscription-plan-card__head p { min-height: 2.55rem; margin-top: 0.35rem; color: #817a72; font-size: 0.76rem; line-height: 1.55; }
.subscription-plan-card__price { display: flex; align-items: baseline; gap: 0.18rem; padding: 0.4rem 1.25rem 0; color: #292622; }
.subscription-plan-card__price strong { font-size: 1.9rem; font-weight: 900; letter-spacing: -0.02em; }
.subscription-plan-card__currency { font-size: 1rem; font-weight: 850; }
.subscription-plan-card__period { color: #928b83; font-size: 0.73rem; }
.subscription-plan-card__original { min-height: 1.3rem; padding: 0.2rem 1.25rem 0; color: #a39b92; font-size: 0.69rem; text-decoration: line-through; }
.subscription-plan-card__original span { margin-left: 0.35rem; color: var(--plan-text, #08736e); font-weight: 750; text-decoration: none; }
.subscription-plan-card__metrics { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0.5rem; margin: 0.95rem 1.25rem 0; }
.subscription-plan-card__metrics > div { border: 1px solid color-mix(in srgb, var(--plan-accent, #0f9f9a) 22%, #fff); border-radius: 8px; background: var(--plan-soft, #e7f8f6); padding: 0.58rem 0.65rem; }
.subscription-plan-card__metrics span { display: block; color: #7c756d; font-size: 0.66rem; }
.subscription-plan-card__metrics strong { display: block; margin-top: 0.2rem; color: var(--plan-text, #08736e); font-size: 0.9rem; font-weight: 850; }
.subscription-plan-card__limits { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0.35rem; margin: 0.55rem 1.25rem 0; }
.subscription-plan-card__limits > div { min-width: 0; border-bottom: 1px solid #eee9e2; padding: 0.35rem 0.2rem 0.45rem; text-align: center; }
.subscription-plan-card__limits span { display: block; color: #8a8279; font-size: 0.62rem; }
.subscription-plan-card__limits strong { display: block; margin-top: 0.18rem; overflow: hidden; color: #443d36; font-size: 0.7rem; font-weight: 800; text-overflow: ellipsis; white-space: nowrap; }
.subscription-plan-card__entitlement { margin: 0.85rem 1.25rem 0; border: 1px solid color-mix(in srgb, var(--plan-accent, #0f9f9a) 28%, #fff); border-radius: 8px; background: #fffdfa; padding: 0.65rem 0.75rem; }
.subscription-plan-card__entitlement-title { display: flex; align-items: center; gap: 0.38rem; color: #675f57; font-size: 0.71rem; font-weight: 800; }
.subscription-plan-card__entitlement-title :deep(svg) { color: var(--plan-accent, #0f9f9a); }
.subscription-plan-card__entitlement-value { margin-top: 0.34rem; overflow: hidden; color: #332e29; font-size: 0.7rem; font-weight: 720; text-overflow: ellipsis; white-space: nowrap; }
.subscription-plan-card__scopes { display: flex; align-items: flex-start; justify-content: space-between; gap: 0.6rem; margin: 0.65rem 1.25rem 0; color: #817a72; font-size: 0.68rem; }
.subscription-plan-card__scopes > div { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 0.3rem; }
.subscription-plan-card__scopes > div span { border-radius: 999px; background: var(--plan-soft, #e7f8f6); color: var(--plan-text, #08736e); font-size: 0.63rem; font-weight: 750; padding: 0.2rem 0.4rem; }
.subscription-plan-card__features { display: grid; gap: 0.42rem; margin: 0.95rem 1.25rem 1rem; padding: 0; list-style: none; }
.subscription-plan-card__features li { display: flex; align-items: flex-start; gap: 0.42rem; color: #635c54; font-size: 0.73rem; line-height: 1.45; }
.subscription-plan-card__features li :deep(svg) { flex: 0 0 auto; margin-top: 0.08rem; color: var(--plan-accent, #0f9f9a); }
.subscription-plan-card__footer { margin-top: auto; padding: 0 1.25rem 1.25rem; }
.subscription-plan-card__availability { display: flex; justify-content: space-between; color: #a29a91; font-size: 0.68rem; }
.subscription-plan-card__availability strong { color: #4d463f; font-weight: 800; }
.subscription-plan-card__progress { height: 4px; margin: 0.42rem 0 0.85rem; overflow: hidden; border-radius: 999px; background: #eeeae5; }
.subscription-plan-card__progress span { display: block; width: 100%; height: 100%; border-radius: inherit; background: var(--plan-accent, #0f9f9a); }
.subscription-plan-card__progress span.subscription-plan-card__progress--off { width: 20%; background: #beb8b0; }
.subscription-plan-card__footer button { display: inline-flex; width: 100%; align-items: center; justify-content: center; gap: 0.4rem; border: 0; border-radius: 8px; background: var(--plan-accent, #0f9f9a); color: #fff; cursor: pointer; font-size: 0.83rem; font-weight: 800; min-height: 2.65rem; padding: 0.65rem 0.9rem; transition: filter 160ms ease, transform 160ms ease; }
.subscription-plan-card__footer button:hover:not(:disabled) { filter: brightness(0.94); transform: translateY(-1px); }
.subscription-plan-card__footer button:disabled { background: #c9c4be; cursor: not-allowed; }
.subscription-plan-card__footer button:focus-visible { outline: 3px solid color-mix(in srgb, var(--plan-accent, #0f9f9a) 45%, #fff); outline-offset: 2px; }

@media (max-width: 640px) {
  .subscription-plan-card { min-height: auto; }
  .subscription-plan-card__head { padding-top: 1rem; }
}

@media (prefers-reduced-motion: reduce) {
  .subscription-plan-card, .subscription-plan-card__footer button { transition: none; }
}
</style>
