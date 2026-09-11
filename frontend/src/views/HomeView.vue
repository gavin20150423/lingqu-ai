<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="hasHomeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Compact Home Page -->
  <div
    v-else-if="compactHomeEnabled"
    data-testid="compact-home"
    class="flex min-h-screen flex-col bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white"
  >
    <header class="border-b border-gray-200 px-4 py-4 sm:px-6 dark:border-dark-800">
      <nav class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-9 w-9 shrink-0 rounded-lg object-contain" />
          <span class="min-w-0 truncate text-base font-semibold">{{ siteName }}</span>
        </div>
        <div class="flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2">
          <LocaleSwitcher />
          <router-link
            v-if="canAccessModelPlaza"
            to="/model-plaza"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg px-3 py-2 text-sm font-medium text-gray-600 hover:bg-gray-100 dark:text-dark-300 dark:hover:bg-dark-800"
          >
            模型广场
          </router-link>
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <button
            type="button"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            :aria-label="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
          <router-link
            :to="entryPath"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="flex min-w-0 flex-1 items-center justify-center px-4 py-16 sm:px-6">
      <div class="min-w-0 max-w-2xl text-center">
        <img :src="siteLogo || '/logo.svg'" alt="Logo" class="mx-auto mb-6 h-20 w-20 rounded-2xl object-contain" />
        <h1 class="[overflow-wrap:anywhere] text-3xl font-bold md:text-4xl">{{ siteName }}</h1>
        <p class="mt-4 whitespace-pre-wrap [overflow-wrap:anywhere] text-base text-gray-600 dark:text-dark-300">{{ siteSubtitle }}</p>
        <router-link
          :to="entryPath"
          class="mt-8 inline-flex min-h-10 items-center justify-center rounded-lg bg-primary-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-primary-700"
        >
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.login') }}
        </router-link>
      </div>
    </main>

    <footer class="min-w-0 border-t border-gray-200 px-4 py-5 text-center text-sm text-gray-500 [overflow-wrap:anywhere] sm:px-6 dark:border-dark-800 dark:text-dark-400">
      &copy; {{ currentYear }} {{ siteName }}
    </footer>
  </div>

  <!-- Default product home page -->
  <div v-else class="lingqu-home" data-testid="default-home">
    <div class="home-background" aria-hidden="true">
      <span class="home-background__beam home-background__beam--one"></span>
      <span class="home-background__beam home-background__beam--two"></span>
      <span class="home-background__dot home-background__dot--one"></span>
      <span class="home-background__dot home-background__dot--two"></span>
    </div>

    <header class="home-header">
      <nav class="home-nav home-shell" aria-label="首页导航">
        <router-link to="/home" class="home-brand" aria-label="灵渠AI 首页">
          <span class="home-brand__mark">
            <img :src="siteLogo || DEFAULT_SITE_LOGO" alt="" />
          </span>
          <span class="home-brand__copy">
            <strong>{{ siteName }}</strong>
            <small>One Key · All Models</small>
          </span>
        </router-link>

        <div class="home-nav__links" aria-label="页面分区">
          <a v-for="item in navItems" :key="item.href" :href="item.href">{{ item.label }}</a>
        </div>

        <div class="home-nav__actions">
          <LocaleSwitcher />
          <router-link v-if="canAccessModelPlaza" to="/model-plaza" class="home-nav__model-link">模型广场</router-link>
          <router-link :to="entryPath" class="home-button home-button--nav">
            {{ isAuthenticated ? t('home.dashboard') : homeCopy.primaryCta }}
            <Icon name="arrowRight" size="sm" />
          </router-link>
        </div>
      </nav>
    </header>

    <main>
      <section id="capabilities" class="home-hero home-shell" aria-labelledby="home-title">
        <div class="home-hero__copy">
          <div class="home-eyebrow">
            <span class="home-eyebrow__pulse"></span>
            {{ homeCopy.eyebrow }}
          </div>
          <h1 id="home-title">
            {{ homeCopy.heroTitle }}
            <span>{{ homeCopy.heroTitleAccent }}</span>
          </h1>
          <p class="home-hero__lead">
            {{ homeCopy.heroLead }}
          </p>
          <div class="home-hero__actions">
            <router-link :to="entryPath" class="home-button home-button--primary">
              <Icon name="key" size="sm" />
              {{ isAuthenticated ? t('home.dashboard') : homeCopy.primaryCta }}
              <Icon name="arrowRight" size="sm" />
            </router-link>
            <a
              :href="docUrl || '#models'"
              :target="docUrl ? '_blank' : undefined"
              :rel="docUrl ? 'noopener noreferrer' : undefined"
              class="home-button home-button--secondary"
            >
              {{ homeCopy.secondaryCta }}
            </a>
          </div>
          <div class="home-hero__proof" :aria-label="homeCopy.proofLabel">
            <span v-for="item in heroProof" :key="item">
              <Icon name="checkCircle" size="sm" />
              {{ item }}
            </span>
          </div>
        </div>

        <div class="home-hero__visual" :aria-label="homeCopy.visualLabel">
          <div class="home-hero__visual-glow"></div>
          <div class="home-hero__artboard">
            <img src="/brand/home/gateway-hero.png" :alt="homeCopy.visualAlt" />
          </div>
          <div class="home-route-card">
            <div class="home-route-card__topline">
              <span class="home-status-dot"></span>
              <span>{{ homeCopy.gatewayOnline }}</span>
              <Icon name="externalLink" size="xs" />
            </div>
            <code>POST /v1/chat/completions</code>
            <div class="home-route-card__bottomline">
              <span>{{ homeCopy.routeLabel }}</span>
              <strong>{{ homeCopy.routeValue }}</strong>
            </div>
          </div>
          <div class="home-model-float home-model-float--one">
            <ModelIcon model="gpt-4o" size="25px" />
            <span>GPT</span>
          </div>
          <div class="home-model-float home-model-float--two">
            <ModelIcon model="claude-3-5-sonnet" size="25px" />
            <span>Claude</span>
          </div>
          <div class="home-model-float home-model-float--three">
            <ModelIcon model="deepseek-v3" size="25px" />
            <span>DeepSeek</span>
          </div>
        </div>
      </section>

      <section class="home-model-band home-shell" aria-labelledby="model-band-title">
        <div class="home-model-band__intro">
          <span class="home-overline">{{ homeCopy.modelBandOverline }}</span>
          <h2 id="model-band-title">{{ homeCopy.modelBandTitle }}</h2>
        </div>
        <div class="home-model-band__logos" :aria-label="homeCopy.modelsAriaLabel">
          <div v-for="model in modelCards" :key="model.key" class="home-model-chip">
            <span class="home-model-chip__icon"><ModelIcon :model="model.iconModel" size="26px" /></span>
            <span>{{ model.name }}</span>
          </div>
          <div class="home-model-chip home-model-chip--more">
            <span class="home-model-chip__more" aria-hidden="true">+</span>
            <span>{{ homeCopy.moreModels }}</span>
          </div>
        </div>
      </section>

      <section id="advantages" class="home-section home-shell" aria-labelledby="advantages-title">
        <div class="home-section__heading">
          <span class="home-overline">{{ homeCopy.advantagesOverline }}</span>
          <h2 id="advantages-title">{{ homeCopy.advantagesTitle }}</h2>
          <p>{{ homeCopy.advantagesDesc }}</p>
        </div>
        <div class="home-feature-grid">
          <article v-for="(item, index) in featureCards" :key="item.title" class="home-feature-card" :class="`home-feature-card--${index + 1}`">
            <span class="home-feature-card__icon"><Icon :name="item.icon" size="lg" /></span>
            <span class="home-feature-card__index">0{{ index + 1 }}</span>
            <h3>{{ item.title }}</h3>
            <p>{{ item.desc }}</p>
          </article>
        </div>
      </section>

      <section id="models" class="home-section home-shell home-section--models" aria-labelledby="models-title">
        <div class="home-section__heading home-section__heading--row">
          <div>
            <span class="home-overline">{{ homeCopy.modelCatalogOverline }}</span>
            <h2 id="models-title">{{ homeCopy.modelCatalogTitle }}</h2>
          </div>
          <router-link v-if="canAccessModelPlaza" to="/model-plaza" class="home-text-link">
            {{ homeCopy.browseModels }}
            <Icon name="arrowRight" size="sm" />
          </router-link>
        </div>
        <div class="home-model-grid">
          <article v-for="model in modelCards" :key="model.key" class="home-model-card">
            <div class="home-model-card__topline">
              <span class="home-model-card__logo"><ModelIcon :model="model.iconModel" size="36px" /></span>
              <span class="home-model-card__availability"><span></span> {{ homeCopy.available }}</span>
            </div>
            <h3>{{ model.name }}</h3>
            <p>{{ model.desc }}</p>
            <div class="home-model-card__meta">
              <span>{{ model.category }}</span>
              <Icon name="arrowRight" size="sm" />
            </div>
          </article>
        </div>
      </section>

      <section id="pricing" class="home-pricing home-shell" aria-labelledby="pricing-title">
        <div class="home-pricing__aside">
          <span class="home-overline home-overline--light">{{ homeCopy.pricingOverline }}</span>
          <h2 id="pricing-title">{{ homeCopy.pricingTitle }}</h2>
          <p>{{ homeCopy.pricingDesc }}</p>
          <div class="home-pricing__actions">
            <router-link :to="rechargePath" class="home-button home-button--light">
              {{ homeCopy.recharge }}
              <Icon name="arrowRight" size="sm" />
            </router-link>
            <router-link :to="pricingPath" class="home-pricing__link">
              {{ homeCopy.subscription }}
              <Icon name="arrowRight" size="sm" />
            </router-link>
          </div>
        </div>
        <div class="home-pricing__plans">
          <article v-for="plan in pricingPlans" :key="plan.title" class="home-plan-card" :class="{ 'home-plan-card--featured': plan.featured }">
            <span v-if="plan.featured" class="home-plan-card__badge">{{ homeCopy.featuredPlan }}</span>
            <span class="home-plan-card__label">{{ plan.label }}</span>
            <h3>{{ plan.title }}</h3>
            <p>{{ plan.desc }}</p>
            <strong>{{ plan.price }}</strong>
            <small>{{ plan.unit }}</small>
            <ul>
              <li v-for="feature in plan.features" :key="feature"><Icon name="check" size="sm" />{{ feature }}</li>
            </ul>
          </article>
        </div>
      </section>

      <section id="integration" class="home-section home-shell" aria-labelledby="quickstart-title">
        <div class="home-section__heading">
          <span class="home-overline">{{ homeCopy.quickStartOverline }}</span>
          <h2 id="quickstart-title">{{ homeCopy.quickStartTitle }}</h2>
          <p>{{ homeCopy.quickStartDesc }}</p>
        </div>
        <div class="home-steps">
          <article v-for="(item, index) in stepItems" :key="item.title" class="home-step">
            <div class="home-step__number">0{{ index + 1 }}</div>
            <div class="home-step__icon"><Icon :name="item.icon" size="lg" /></div>
            <h3>{{ item.title }}</h3>
            <p>{{ item.desc }}</p>
            <span v-if="index < stepItems.length - 1" class="home-step__connector" aria-hidden="true"><Icon name="arrowRight" size="md" /></span>
          </article>
        </div>
      </section>

      <section class="home-final home-shell" aria-labelledby="final-title">
        <div>
          <span class="home-overline">{{ homeCopy.finalOverline }}</span>
          <h2 id="final-title">{{ homeCopy.finalTitle }}</h2>
          <p>{{ homeCopy.finalDesc }}</p>
        </div>
        <router-link :to="entryPath" class="home-button home-button--primary">
          {{ homeCopy.createKey }}
          <Icon name="arrowRight" size="sm" />
        </router-link>
      </section>
    </main>

    <footer class="home-footer">
      <div class="home-footer__inner home-shell">
        <div class="home-footer__brand">
          <img :src="siteLogo || DEFAULT_SITE_LOGO" alt="" />
          <div>
            <strong>{{ siteName }}</strong>
            <span>{{ homeCopy.footerTagline }}</span>
          </div>
        </div>
        <div class="home-footer__links">
          <a v-for="item in navItems" :key="`footer-${item.href}`" :href="item.href">{{ item.label }}</a>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ homeCopy.docs }}</a>
        </div>
        <span class="home-footer__copyright">© {{ currentYear }} {{ siteName }}</span>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import { DEFAULT_SITE_LOGO, resolveBrandLogo, resolveBrandName } from '@/constants/brand'
import { sanitizeUrl } from '@/utils/url'
import i18n from '@/i18n'

const authStore = useAuthStore()
const appStore = useAppStore()
const { t } = useI18n()
const locale = computed(() => String(i18n.global.locale.value))

const siteName = computed(() => resolveBrandName(appStore.cachedPublicSettings?.site_name || appStore.siteName))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const siteLogo = computed(() =>
  sanitizeUrl(
    resolveBrandLogo(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo) || DEFAULT_SITE_LOGO,
    { allowRelative: true, allowDataUrl: true }
  )
)
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const isDark = ref(document.documentElement.classList.contains('dark'))
const currentYear = computed(() => new Date().getFullYear())
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/dashboard'))
const entryPath = computed(() => (isAuthenticated.value ? dashboardPath.value : '/login'))
const pricingPath = computed(() => (isAuthenticated.value ? '/subscription-plans' : '/login?redirect=/subscription-plans'))
const rechargePath = computed(() => (isAuthenticated.value ? '/purchase' : '/login?redirect=/purchase'))
const modelPlazaEnabled = computed(() => appStore.cachedPublicSettings?.model_plaza_enabled === true)
const modelPlazaRequiresAuth = computed(() => appStore.cachedPublicSettings?.model_plaza_require_auth === true)
const canAccessModelPlaza = computed(() => modelPlazaEnabled.value && (!modelPlazaRequiresAuth.value || isAuthenticated.value))

const homeCopy = computed(() => {
  if (locale.value === 'en') {
    return {
      eyebrow: 'AI gateway for developers and teams',
      heroTitle: 'One Key, connect',
      heroTitleAccent: 'every leading model',
      heroLead: 'Lingqu AI brings GPT, Claude, Gemini and leading Chinese models behind one OpenAI-compatible endpoint. Ship faster with unified keys, routing, usage and billing.',
      primaryCta: 'Start integrating',
      secondaryCta: 'See how it works',
      proofLabel: 'Product benefits',
      visualLabel: 'Lingqu AI multi-model gateway',
      visualAlt: 'Multiple AI models connected through one API gateway',
      gatewayOnline: 'Gateway online',
      routeLabel: 'Auto-routed to',
      routeValue: 'Best available model',
      modelBandOverline: 'One API · Many Models',
      modelBandTitle: 'The models you already use, in one place.',
      modelsAriaLabel: 'Supported mainstream models',
      moreModels: 'More models',
      advantagesOverline: 'Built for shipping',
      advantagesTitle: 'Let one stable layer handle complex model access.',
      advantagesDesc: 'Compatibility, reliability, observability and cost control, right where your team needs them.',
      modelCatalogOverline: 'Model catalog',
      modelCatalogTitle: 'Choose by use case, call through one endpoint.',
      browseModels: 'Browse model plaza',
      available: 'Available',
      pricingOverline: 'Flexible billing',
      pricingTitle: 'Choose what fits, keep costs in control.',
      pricingDesc: 'Keep subscriptions and account top-ups separate, so teams can budget around real usage.',
      recharge: 'Top up account',
      subscription: 'Plans',
      featuredPlan: 'Popular with teams',
      quickStartOverline: 'Quick start',
      quickStartTitle: 'Get your first request running in three steps.',
      quickStartDesc: 'Use familiar SDKs and request formats without learning a new platform.',
      finalOverline: 'Ready when you are',
      finalTitle: 'Start with one Key and put your model service into production.',
      finalDesc: 'Create a Key, set its quota in the console, then add the Base URL to your existing app.',
      createKey: 'Create my Key',
      footerTagline: 'One gateway for leading AI models',
      docs: 'Docs'
    }
  }

  return {
    eyebrow: '面向开发者和团队的 AI 网关',
    heroTitle: '一个 Key，接入',
    heroTitleAccent: '所有主流模型',
    heroLead: '灵渠AI 把 GPT、Claude、Gemini 以及国内主流模型汇聚到一个 OpenAI 兼容入口。少改代码，统一管理 Key、路由、用量和费用。',
    primaryCta: '免费开始接入',
    secondaryCta: '查看接入方式',
    proofLabel: '产品特性',
    visualLabel: '灵渠AI 多模型网关',
    visualAlt: '多个 AI 模型围绕统一 API 网关连接',
    gatewayOnline: 'Gateway online',
    routeLabel: '自动路由至',
    routeValue: '最佳可用模型',
    modelBandOverline: 'One API · Many Models',
    modelBandTitle: '你熟悉的模型，都在这里。',
    modelsAriaLabel: '支持的主流模型',
    moreModels: '更多模型',
    advantagesOverline: 'Built for shipping',
    advantagesTitle: '复杂的模型接入，交给一个稳定的中间层。',
    advantagesDesc: '把开发者真正关心的事情放到台前：兼容性、稳定性、可观测和成本控制。',
    modelCatalogOverline: 'Model catalog',
    modelCatalogTitle: '按场景选择模型，按一个入口调用。',
    browseModels: '浏览模型广场',
    available: '可调用',
    pricingOverline: 'Flexible billing',
    pricingTitle: '按需选择，成本更可控。',
    pricingDesc: '订阅套餐和账户充值分开管理，团队可以按实际调用节奏灵活安排预算。',
    recharge: '账户充值',
    subscription: '订阅套餐',
    featuredPlan: '适合大多数团队',
    quickStartOverline: 'Quick start',
    quickStartTitle: '三步，把第一个请求跑起来。',
    quickStartDesc: '使用熟悉的 SDK 和调用方式，接入过程不需要重新学习一套平台。',
    finalOverline: 'Ready when you are',
    finalTitle: '从一个 Key 开始，让模型服务真正上线。',
    finalDesc: '控制台里创建 Key、配置额度，然后把 Base URL 放进现有应用。',
    createKey: '创建我的 Key',
    footerTagline: '统一接入主流大模型',
    docs: '文档'
  }
})

const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

const navItems = computed(() => locale.value === 'en'
  ? [
      { label: 'Capabilities', href: '#capabilities' },
      { label: 'Advantages', href: '#advantages' },
      { label: 'Models', href: '#models' },
      { label: 'Integration', href: '#integration' },
    ]
  : [
      { label: '能力', href: '#capabilities' },
      { label: '优势', href: '#advantages' },
      { label: '模型', href: '#models' },
      { label: '接入', href: '#integration' },
    ])

const heroProof = computed(() => locale.value === 'en'
  ? ['OpenAI compatible', 'Unified key management', 'Clear usage visibility']
  : ['OpenAI 兼容', '统一 Key 管理', '用量清晰可见'])

const featureCards = computed(() => locale.value === 'en'
  ? [
      { icon: 'globe', title: 'Unified API', desc: 'Use the familiar OpenAI-compatible format and connect existing apps with minimal changes.' },
      { icon: 'bolt', title: 'Smart routing', desc: 'Route requests to the right available resource by model, quota and channel status.' },
      { icon: 'refresh', title: 'Automatic failover', desc: 'Switch channels when an upstream fails, so one provider does not stop your workflow.' },
      { icon: 'chartBar', title: 'Usage visibility', desc: 'Keep requests, tokens and costs in one place so team budgets stay visible.' },
      { icon: 'users', title: 'Team collaboration', desc: 'Manage keys, quotas and access levels for shared projects and teams.' },
      { icon: 'shield', title: 'Secure by design', desc: 'Keep provider credentials on the server and reduce exposure in your applications.' },
    ] as const
  : [
      { icon: 'globe', title: '统一 API', desc: '沿用熟悉的 OpenAI 调用格式，接入现有应用更轻量。' },
      { icon: 'bolt', title: '智能路由', desc: '按模型、额度和通道状态，把请求送到合适的可用资源。' },
      { icon: 'refresh', title: '失败切换', desc: '通道异常时自动切换，减少业务被单一供应商卡住。' },
      { icon: 'chartBar', title: '用量可观测', desc: '请求、Token 和费用集中记录，团队预算一眼可见。' },
      { icon: 'users', title: '团队协作', desc: 'Key、额度和访问权限分层管理，适合多人共用项目。' },
      { icon: 'shield', title: '安全可控', desc: '把密钥和供应商账号留在服务端，降低应用侧泄露风险。' },
    ] as const)

const modelCards = computed(() => locale.value === 'en'
  ? [
      { key: 'gpt', iconModel: 'gpt-4o', name: 'GPT', desc: 'General creation, code and reasoning', category: 'OpenAI' },
      { key: 'claude', iconModel: 'claude-3-5-sonnet', name: 'Claude', desc: 'Long context and structured tasks', category: 'Anthropic' },
      { key: 'gemini', iconModel: 'gemini-1.5-pro', name: 'Gemini', desc: 'Multimodal work and long context', category: 'Google' },
      { key: 'kimi', iconModel: 'kimi-k2', name: 'Kimi', desc: 'Chinese documents and research', category: 'Moonshot' },
      { key: 'glm', iconModel: 'glm-4-plus', name: 'GLM', desc: 'Chinese understanding and tool use', category: 'Zhipu' },
      { key: 'qwen', iconModel: 'qwen-max', name: 'Qwen', desc: 'Chinese workflows and code generation', category: 'Alibaba' },
      { key: 'deepseek', iconModel: 'deepseek-v3', name: 'DeepSeek', desc: 'Efficient reasoning and coding', category: 'DeepSeek' },
    ] as const
  : [
      { key: 'gpt', iconModel: 'gpt-4o', name: 'GPT', desc: '通用创作、代码与复杂推理', category: 'OpenAI' },
      { key: 'claude', iconModel: 'claude-3-5-sonnet', name: 'Claude', desc: '长文本理解与结构化任务', category: 'Anthropic' },
      { key: 'gemini', iconModel: 'gemini-1.5-pro', name: 'Gemini', desc: '多模态处理与超长上下文', category: 'Google' },
      { key: 'kimi', iconModel: 'kimi-k2', name: 'Kimi', desc: '中文长文档与深度研究', category: 'Moonshot' },
      { key: 'glm', iconModel: 'glm-4-plus', name: 'GLM', desc: '中文理解、推理与工具调用', category: '智谱' },
      { key: 'qwen', iconModel: 'qwen-max', name: 'Qwen', desc: '中文场景与代码生成', category: '通义千问' },
      { key: 'deepseek', iconModel: 'deepseek-v3', name: 'DeepSeek', desc: '高性价比推理与代码能力', category: 'DeepSeek' },
    ] as const)

const pricingPlans = computed(() => locale.value === 'en'
  ? [
      { label: 'PAY AS YOU GO', title: 'Usage billing', desc: 'For individual developers and small projects', price: '¥0.02', unit: '/ 1K tokens from', features: ['No prepaid package', 'Pay for actual usage'], featured: false },
      { label: 'SUBSCRIPTION', title: 'Subscription', desc: 'For stable usage across people and teams', price: '¥19.90', unit: '/ from 30 days', features: ['Clear quota and validity', 'Unified subscription benefits'], featured: true },
      { label: 'TEAM', title: 'Team plan', desc: 'For teams that need shared control', price: 'Flexible', unit: 'Configured for your team', features: ['Member and key roles', 'Dedicated support'], featured: false },
    ]
  : [
      { label: 'PAY AS YOU GO', title: '按量计费', desc: '适合个人开发者和小型项目', price: '¥0.02', unit: '/ 1K Token 起', features: ['无需预付套餐', '按实际用量结算'], featured: false },
      { label: 'SUBSCRIPTION', title: '订阅套餐', desc: '适合稳定调用的个人和团队', price: '¥19.90', unit: '/ 30 天起', features: ['额度和有效期清晰', '统一管理订阅权益'], featured: true },
      { label: 'TEAM', title: '团队方案', desc: '适合需要协作和预算控制的团队', price: '灵活配置', unit: '按团队需求定制', features: ['成员与 Key 分权', '专属技术支持'], featured: false },
    ])

const stepItems = computed(() => locale.value === 'en'
  ? [
      { icon: 'userPlus', title: 'Create an account', desc: 'Create your Lingqu AI account and enter the console.' },
      { icon: 'key', title: 'Get an API Key', desc: 'Create a Key and set its quota and access scope.' },
      { icon: 'terminal', title: 'Start calling', desc: 'Replace the Base URL and use your existing SDK.' },
    ] as const
  : [
      { icon: 'userPlus', title: '注册账号', desc: '创建你的灵渠AI 账号，进入控制台。' },
      { icon: 'key', title: '获取 API Key', desc: '创建 Key 并设置额度和访问范围。' },
      { icon: 'terminal', title: '开始调用', desc: '替换 Base URL，用原有 SDK 发起请求。' },
    ] as const)

onMounted(() => {
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
.lingqu-home {
  --home-ink: #0c1d3d;
  --home-muted: #5e6d85;
  --home-primary: #2764e8;
  --home-primary-dark: #1746b3;
  --home-line: #dce7f4;
  --home-card: rgba(255, 255, 255, 0.86);
  position: relative;
  min-height: 100vh;
  padding-top: 22px;
  overflow: visible;
  color: var(--home-ink);
  background: #f6f9fd;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Noto Sans SC", sans-serif;
  font-synthesis: none;
}

.home-background {
  position: absolute;
  inset: 0 0 auto;
  height: 49rem;
  overflow: hidden;
  pointer-events: none;
  background: radial-gradient(circle at 80% 12%, rgba(133, 192, 255, 0.38), transparent 27rem), linear-gradient(180deg, #edf7ff 0%, #f7fbff 58%, #f6f9fd 100%);
}

.home-background::before {
  position: absolute;
  inset: 0;
  content: '';
  opacity: 0.56;
  background-image: radial-gradient(rgba(41, 101, 197, 0.16) 1px, transparent 1px);
  background-size: 25px 25px;
  mask-image: linear-gradient(180deg, black, transparent 75%);
}

.home-background__beam,
.home-background__dot { position: absolute; display: block; border-radius: 999px; }
.home-background__beam { height: 1px; background: rgba(45, 104, 204, 0.16); transform: rotate(-13deg); }
.home-background__beam--one { top: 19rem; left: -7rem; width: 40rem; }
.home-background__beam--two { top: 13rem; right: -8rem; width: 33rem; transform: rotate(16deg); }
.home-background__dot { width: 0.55rem; height: 0.55rem; background: #4e83ea; box-shadow: 0 0 0 0.4rem rgba(78, 131, 234, 0.12); }
.home-background__dot--one { top: 9rem; left: 9%; }
.home-background__dot--two { top: 30rem; right: 12%; background: #3bc4b2; box-shadow: 0 0 0 0.4rem rgba(59, 196, 178, 0.13); }

.home-shell { width: min(1320px, calc(100% - 64px)); margin-right: auto; margin-left: auto; }
.home-header { position: sticky; top: 12px; z-index: 20; }
.home-nav { display: flex; min-height: 64px; align-items: center; justify-content: space-between; gap: 20px; padding: 8px 10px 8px 14px; border: 1px solid rgba(207, 222, 241, 0.96); border-radius: 17px; background: rgba(255, 255, 255, 0.9); box-shadow: 0 13px 34px rgba(58, 103, 164, 0.08); backdrop-filter: blur(18px); }
.home-brand, .home-footer__brand { display: inline-flex; min-width: 0; align-items: center; color: inherit; }
.home-brand { gap: 11px; }
.home-brand__mark { display: grid; width: 40px; height: 40px; flex: none; place-items: center; overflow: hidden; border: 1px solid #cbdcf2; border-radius: 11px; background: #f8fbff; }
.home-brand__mark img { width: 32px; height: 32px; object-fit: contain; }
.home-brand__copy { display: grid; min-width: 0; gap: 3px; }
.home-brand__copy strong { overflow: hidden; font-size: 15px; font-weight: 700; line-height: 1; text-overflow: ellipsis; white-space: nowrap; }
.home-brand__copy small { color: #8390a4; font-size: 9px; font-weight: 600; letter-spacing: 0; white-space: nowrap; }
.home-nav__links { display: flex; align-items: center; gap: clamp(20px, 3vw, 38px); margin-left: auto; }
.home-nav__links a, .home-nav__model-link { color: #56647a; font-size: 12px; font-weight: 600; transition: color 160ms ease; }
.home-nav__links a:hover, .home-nav__model-link:hover { color: var(--home-primary); }
.home-nav__actions { display: flex; align-items: center; gap: 12px; }
.home-nav__actions :deep(button) { color: #58677d; }
.home-nav__model-link { padding: 10px 2px; white-space: nowrap; }
.lingqu-home :is(a, button):focus-visible { outline: 3px solid rgba(39, 100, 232, 0.42); outline-offset: 3px; }

.home-button { display: inline-flex; min-height: 42px; align-items: center; justify-content: center; gap: 7px; border: 1px solid transparent; border-radius: 9px; padding: 0 16px; color: var(--home-ink); font-size: 12px; font-weight: 700; transition: transform 160ms ease, box-shadow 160ms ease, background 160ms ease, border-color 160ms ease; }
.home-button:hover { transform: translateY(-2px); }
.home-button:active { transform: translateY(0); }
.home-button--nav { min-height: 40px; background: var(--home-ink); color: #fff; box-shadow: 0 7px 16px rgba(12, 29, 61, 0.17); }
.home-button--nav:hover { background: #18315e; box-shadow: 0 10px 20px rgba(12, 29, 61, 0.23); }
.home-button--primary { background: var(--home-primary); color: #fff; box-shadow: 0 12px 22px rgba(39, 100, 232, 0.22); }
.home-button--primary:hover { background: var(--home-primary-dark); box-shadow: 0 15px 26px rgba(39, 100, 232, 0.27); }
.home-button--secondary { border-color: #d4e0ef; background: rgba(255, 255, 255, 0.76); color: #41536f; }
.home-button--secondary:hover { border-color: #aac5e8; background: #fff; }
.home-button--light { background: #fff; color: #1850c2; box-shadow: 0 9px 20px rgba(9, 42, 111, 0.2); }
.home-button--light:hover { background: #eff5ff; }
.home-pricing__actions { position: relative; z-index: 1; display: flex; flex-wrap: wrap; align-items: center; gap: 18px; margin-top: 27px; }
.home-pricing__link { display: inline-flex; align-items: center; gap: 6px; color: #dbe8ff; font-size: 12px; font-weight: 800; }
.home-pricing__link:hover { color: #fff; }

.home-hero { position: relative; z-index: 1; display: grid; min-height: 560px; grid-template-columns: minmax(0, 0.98fr) minmax(0, 1.02fr); align-items: center; gap: clamp(24px, 4vw, 52px); padding-top: 54px; padding-bottom: 58px; }
.home-hero__copy { position: relative; z-index: 2; max-width: 540px; }
.home-eyebrow { display: inline-flex; align-items: center; gap: 8px; padding: 6px 11px; border: 1px solid #d4e6fb; border-radius: 999px; background: rgba(255, 255, 255, 0.76); color: #2a67d0; font-size: 11px; font-weight: 700; }
.home-eyebrow__pulse { width: 7px; height: 7px; border-radius: 50%; background: #27b89d; box-shadow: 0 0 0 4px rgba(39, 184, 157, 0.12); }
.home-hero h1 { max-width: 590px; margin: 21px 0 0; color: var(--home-ink); font-size: clamp(42px, 5vw, 62px); font-weight: 700; letter-spacing: 0; line-height: 1.16; }
.home-hero h1 span { display: block; color: var(--home-primary); }
.home-hero__lead { max-width: 500px; margin-top: 20px; color: var(--home-muted); font-size: 14px; line-height: 1.82; }
.home-hero__actions { display: flex; flex-wrap: wrap; gap: 10px; margin-top: 27px; }
.home-hero__proof { display: flex; flex-wrap: wrap; gap: 15px; margin-top: 20px; }
.home-hero__proof span { display: inline-flex; align-items: center; gap: 5px; color: #748198; font-size: 11px; font-weight: 600; }
.home-hero__proof :deep(svg) { color: #2ab497; }

.home-hero__visual { position: relative; min-height: 450px; }
.home-hero__visual-glow { position: absolute; inset: 12% 2% 8% 9%; border-radius: 50%; background: rgba(130, 190, 255, 0.26); filter: blur(46px); }
.home-hero__artboard { position: absolute; inset: 5% 2% 10%; display: grid; place-items: center; overflow: hidden; border: 1px solid rgba(209, 226, 245, 0.92); border-radius: 26px; background: rgba(255, 255, 255, 0.48); box-shadow: 0 20px 46px rgba(50, 100, 165, 0.1); }
.home-hero__artboard::before { position: absolute; inset: 14px; border: 1px solid rgba(105, 155, 219, 0.14); border-radius: 19px; content: ''; }
.home-hero__artboard img { position: relative; z-index: 1; width: 106%; max-width: none; height: 106%; object-fit: contain; object-position: center; filter: saturate(1.04); }
.home-route-card { position: absolute; z-index: 3; right: 4%; bottom: 0; width: min(278px, 64%); padding: 14px 16px; border: 1px solid #d4e1f0; border-radius: 13px; background: rgba(255, 255, 255, 0.94); box-shadow: 0 13px 30px rgba(40, 83, 143, 0.14); }
.home-route-card__topline, .home-route-card__bottomline { display: flex; align-items: center; gap: 7px; color: #7890ad; font-size: 10px; font-weight: 700; }
.home-route-card__topline :deep(svg) { margin-left: auto; color: #92a7c0; }
.home-status-dot { width: 7px; height: 7px; border-radius: 50%; background: #20b997; box-shadow: 0 0 0 4px rgba(32, 185, 151, 0.1); }
.home-route-card code { display: block; margin: 11px 0; color: #1d4ba4; font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: 11px; font-weight: 700; }
.home-route-card__bottomline strong { margin-left: auto; color: #2c5db5; font-size: 11px; }
.home-model-float { position: absolute; z-index: 3; display: inline-flex; align-items: center; gap: 6px; padding: 7px 10px 7px 7px; border: 1px solid rgba(205, 220, 239, 0.96); border-radius: 10px; background: rgba(255, 255, 255, 0.92); box-shadow: 0 8px 18px rgba(50, 95, 155, 0.11); color: #2f405d; font-size: 10px; font-weight: 700; }
.home-model-float--one { top: 12%; left: 4%; }
.home-model-float--two { top: 49%; right: 0; }
.home-model-float--three { bottom: 15%; left: 4%; }

.home-model-band { position: relative; z-index: 2; display: grid; grid-template-columns: 240px minmax(0, 1fr); align-items: center; gap: 25px; padding: 20px 24px; border: 1px solid var(--home-line); border-radius: 16px; background: rgba(255, 255, 255, 0.88); box-shadow: 0 13px 30px rgba(58, 103, 164, 0.07); }
.home-overline { display: block; color: #3370d1; font-size: 9px; font-weight: 700; letter-spacing: 0.1em; text-transform: uppercase; }
.home-overline--light { color: #b8d3ff; }
.home-model-band h2 { margin-top: 6px; font-size: 18px; font-weight: 700; letter-spacing: 0; }
.home-model-band__logos { display: grid; grid-template-columns: repeat(8, minmax(0, 1fr)); gap: 8px; }
.home-model-chip { display: flex; min-width: 0; align-items: center; justify-content: center; gap: 7px; padding: 8px 5px; border-left: 1px solid #e4edf7; color: #44546e; font-size: 11px; font-weight: 700; white-space: nowrap; }
.home-model-chip:first-child { border-left: 0; }
.home-model-chip__icon { display: grid; width: 30px; height: 30px; place-items: center; flex: none; }
.home-model-chip--more { color: #3971d0; }
.home-model-chip__more { display: grid; width: 26px; height: 26px; place-items: center; border: 1px dashed #8db0e5; border-radius: 50%; color: #3971d0; font-size: 19px; font-weight: 400; line-height: 1; }

.home-section { position: relative; z-index: 1; padding-top: 88px; }
.home-section__heading { max-width: 620px; margin-bottom: 27px; }
.home-section__heading h2, .home-pricing h2, .home-final h2 { margin-top: 9px; color: var(--home-ink); font-size: clamp(27px, 3.4vw, 37px); font-weight: 700; letter-spacing: 0; line-height: 1.28; }
.home-section__heading p { margin-top: 11px; color: var(--home-muted); font-size: 13px; line-height: 1.75; }
.home-section__heading--row { display: flex; align-items: end; justify-content: space-between; gap: 22px; max-width: none; }
.home-text-link { display: inline-flex; align-items: center; gap: 6px; flex: none; padding-bottom: 5px; color: var(--home-primary); font-size: 13px; font-weight: 800; }
.home-text-link:hover { color: var(--home-primary-dark); }
.home-feature-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.home-feature-card { position: relative; min-height: 178px; overflow: hidden; padding: 20px; border: 1px solid var(--home-line); border-radius: 14px; background: var(--home-card); box-shadow: 0 9px 24px rgba(51, 91, 143, 0.055); transition: transform 180ms ease, box-shadow 180ms ease, border-color 180ms ease; }
.home-feature-card:hover { transform: translateY(-4px); border-color: #bbd2ef; box-shadow: 0 17px 34px rgba(51, 91, 143, 0.12); }
.home-feature-card::after { position: absolute; right: -35px; bottom: -42px; width: 130px; height: 130px; border: 1px solid rgba(88, 137, 211, 0.12); border-radius: 50%; content: ''; }
.home-feature-card__icon { display: grid; width: 40px; height: 40px; place-items: center; border-radius: 11px; background: #eaf2ff; color: #2b69d7; }
.home-feature-card:nth-child(2) .home-feature-card__icon { background: #eafaf7; color: #18a68d; }
.home-feature-card:nth-child(3) .home-feature-card__icon { background: #fff5e8; color: #d7862b; }
.home-feature-card:nth-child(4) .home-feature-card__icon { background: #f1ecff; color: #7658cd; }
.home-feature-card:nth-child(5) .home-feature-card__icon { background: #eaf4ff; color: #4175c8; }
.home-feature-card:nth-child(6) .home-feature-card__icon { background: #edf7f4; color: #2b9d7e; }
.home-feature-card__index { position: absolute; top: 22px; right: 21px; color: #b1bfd1; font-family: ui-monospace, monospace; font-size: 10px; font-weight: 700; }
.home-feature-card h3 { margin-top: 18px; font-size: 17px; font-weight: 700; }
.home-feature-card p { max-width: 285px; margin-top: 7px; color: var(--home-muted); font-size: 12px; line-height: 1.7; }

.home-section--models { padding-bottom: 8px; }
.home-model-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; }
.home-model-card { min-height: 170px; padding: 18px; border: 1px solid var(--home-line); border-radius: 14px; background: rgba(255, 255, 255, 0.9); box-shadow: 0 9px 23px rgba(51, 91, 143, 0.055); transition: transform 180ms ease, box-shadow 180ms ease; }
.home-model-card:hover { transform: translateY(-4px); box-shadow: 0 16px 32px rgba(51, 91, 143, 0.12); }
.home-model-card__topline { display: flex; align-items: start; justify-content: space-between; gap: 10px; }
.home-model-card__logo { display: grid; width: 46px; height: 46px; place-items: center; border: 1px solid #e0eaf5; border-radius: 12px; background: #fbfdff; }
.home-model-card__availability { display: inline-flex; align-items: center; gap: 4px; color: #40a88d; font-size: 10px; font-weight: 750; }
.home-model-card__availability span { width: 5px; height: 5px; border-radius: 50%; background: #33bd9b; }
.home-model-card h3 { margin-top: 14px; font-size: 16px; font-weight: 700; }
.home-model-card p { min-height: 35px; margin-top: 5px; color: var(--home-muted); font-size: 11px; line-height: 1.65; }
.home-model-card__meta { display: flex; align-items: center; justify-content: space-between; margin-top: 10px; padding-top: 10px; border-top: 1px solid #edf2f8; color: #91a0b4; font-size: 9px; font-weight: 600; }
.home-model-card__meta :deep(svg) { color: #5585d4; }

.home-pricing { display: grid; grid-template-columns: 0.76fr 1.74fr; gap: 0; margin-top: 88px; overflow: hidden; border-radius: 19px; background: #245bd6; box-shadow: 0 18px 42px rgba(31, 82, 183, 0.16); }
.home-pricing__aside { position: relative; overflow: hidden; padding: 36px 32px; color: #fff; background: linear-gradient(145deg, #245bd6, #3978ed); }
.home-pricing__aside::after { position: absolute; right: -70px; bottom: -72px; width: 220px; height: 220px; border: 1px solid rgba(255, 255, 255, 0.2); border-radius: 50%; content: ''; box-shadow: 0 0 0 22px rgba(255, 255, 255, 0.04), 0 0 0 44px rgba(255, 255, 255, 0.035); }
.home-pricing__aside h2 { position: relative; z-index: 1; color: #fff; font-size: clamp(27px, 3vw, 36px); }
.home-pricing__aside p { position: relative; z-index: 1; max-width: 270px; margin-top: 15px; color: rgba(255, 255, 255, 0.74); font-size: 13px; line-height: 1.8; }
.home-pricing__plans { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); padding: 12px; background: #fff; }
.home-plan-card { position: relative; min-width: 0; padding: 24px 18px 20px; border-right: 1px solid #e7eef7; }
.home-plan-card:last-child { border-right: 0; }
.home-plan-card--featured { background: #f8fbff; }
.home-plan-card__badge { position: absolute; top: 9px; right: 13px; padding: 4px 7px; border-radius: 5px; background: #e7f5f1; color: #29977f; font-size: 9px; font-weight: 800; }
.home-plan-card__label { color: #6683ab; font-size: 9px; font-weight: 850; letter-spacing: 0.08em; }
.home-plan-card h3 { margin-top: 8px; font-size: 17px; font-weight: 700; }
.home-plan-card p { min-height: 40px; margin-top: 6px; color: var(--home-muted); font-size: 11px; line-height: 1.6; }
.home-plan-card strong { display: inline-block; margin-top: 15px; color: #163b84; font-size: 23px; font-weight: 700; }
.home-plan-card > small { margin-left: 3px; color: #7e8da3; font-size: 10px; }
.home-plan-card ul { display: grid; gap: 8px; margin-top: 20px; padding-top: 15px; border-top: 1px solid #e8eef6; }
.home-plan-card li { display: flex; align-items: center; gap: 6px; color: #66778f; font-size: 11px; font-weight: 650; }
.home-plan-card li :deep(svg) { color: #25ad8d; }

.home-steps { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.home-step { position: relative; min-height: 160px; padding: 20px; border: 1px solid var(--home-line); border-radius: 14px; background: rgba(255, 255, 255, 0.88); }
.home-step__number { color: #9bb0cc; font-family: ui-monospace, monospace; font-size: 12px; font-weight: 800; }
.home-step__icon { display: grid; width: 40px; height: 40px; margin-top: 18px; place-items: center; border-radius: 50%; background: #eaf2ff; color: #2869db; }
.home-step:nth-child(2) .home-step__icon { background: #e8faf5; color: #20a68d; }
.home-step:nth-child(3) .home-step__icon { background: #f1ebff; color: #7458d0; }
.home-step h3 { margin-top: 13px; font-size: 17px; font-weight: 700; }
.home-step p { margin-top: 6px; color: var(--home-muted); font-size: 12px; line-height: 1.65; }
.home-step__connector { position: absolute; top: 45%; right: -25px; z-index: 2; display: grid; width: 36px; height: 36px; place-items: center; border: 1px solid #d4e2f2; border-radius: 50%; background: #fff; color: #7194ca; }

.home-final { display: flex; align-items: center; justify-content: space-between; gap: 28px; margin-top: 88px; padding: 30px 34px; border: 1px solid #cfe0f5; border-radius: 18px; background: linear-gradient(105deg, #eaf4ff, #f7fbff 66%, #edf9f8); }
.home-final h2 { max-width: 650px; font-size: clamp(27px, 3.7vw, 39px); }
.home-final p { margin-top: 10px; color: var(--home-muted); font-size: 13px; line-height: 1.7; }
.home-final .home-button { flex: none; }

.home-footer { margin-top: 76px; padding: 31px 0 36px; background: #0d1c37; color: #fff; }
.home-footer__inner { display: flex; align-items: center; gap: 40px; }
.home-footer__brand { gap: 11px; }
.home-footer__brand img { width: 39px; height: 39px; object-fit: contain; border-radius: 11px; background: #fff; }
.home-footer__brand div { display: grid; gap: 5px; }
.home-footer__brand strong { font-size: 14px; font-weight: 700; }
.home-footer__brand span { color: #91a2bd; font-size: 10px; }
.home-footer__links { display: flex; align-items: center; gap: 25px; margin-left: auto; }
.home-footer__links a { color: #acbad0; font-size: 11px; font-weight: 700; }
.home-footer__links a:hover { color: #fff; }
.home-footer__copyright { color: #687b9b; font-size: 10px; }

:global(.dark) .lingqu-home { --home-ink: #eef5ff; --home-muted: #a8b7cc; --home-line: #2a3d5e; --home-card: rgba(18, 36, 65, 0.84); background: #0b1830; }
:global(.dark) .lingqu-home .home-background { background: linear-gradient(180deg, #112a50, #0b1830 90%); }
:global(.dark) .lingqu-home .home-nav,
:global(.dark) .lingqu-home .home-model-band,
:global(.dark) .lingqu-home .home-model-card,
:global(.dark) .lingqu-home .home-step,
:global(.dark) .lingqu-home .home-final { border-color: #294263; background: rgba(16, 35, 65, 0.86); }
:global(.dark) .lingqu-home .home-nav__links a,
:global(.dark) .lingqu-home .home-nav__model-link,
:global(.dark) .lingqu-home .home-model-chip,
:global(.dark) .lingqu-home .home-hero__proof span { color: #b4c2d5; }
:global(.dark) .lingqu-home .home-button--secondary { border-color: #355178; background: #142b4e; color: #dce9fb; }
:global(.dark) .lingqu-home .home-hero__artboard { border-color: #29496e; background: rgba(22, 47, 83, 0.58); }
:global(.dark) .lingqu-home .home-model-chip { border-color: #294263; }
:global(.dark) .lingqu-home .home-model-card__logo { border-color: #2d4668; background: #152b4d; }
:global(.dark) .lingqu-home .home-model-card__meta,
:global(.dark) .lingqu-home .home-plan-card ul { border-color: #294263; }

@media (max-width: 960px) {
  .home-nav__links { gap: 17px; }
  .home-nav__model-link { display: none; }
  .home-hero { grid-template-columns: minmax(0, 0.96fr) minmax(0, 1.04fr); gap: 20px; }
  .home-hero h1 { font-size: clamp(38px, 5vw, 54px); }
  .home-hero__visual { min-height: 400px; }
  .home-model-band { grid-template-columns: 1fr; gap: 18px; }
  .home-model-band__logos { grid-template-columns: repeat(8, minmax(100px, 1fr)); overflow-x: auto; padding-bottom: 3px; }
  .home-model-chip { border-left: 0; }
  .home-feature-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .home-model-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .home-pricing { grid-template-columns: 1fr; }
  .home-pricing__aside { padding: 36px; }
}

@media (max-width: 720px) {
  .home-shell { width: min(100% - 28px, 560px); }
  .lingqu-home { padding-top: 12px; }
  .home-header { top: 10px; }
  .home-nav { min-height: 58px; border-radius: 15px; padding: 7px 8px 7px 10px; }
  .home-brand__mark { width: 36px; height: 36px; border-radius: 10px; }
  .home-brand__mark img { width: 29px; height: 29px; }
  .home-brand__copy strong { font-size: 14px; }
  .home-brand__copy small { font-size: 9px; }
  .home-nav__links { display: none; }
  .home-nav__actions { gap: 7px; }
  .home-nav__actions :deep(button) { display: none; }
  .home-button--nav { min-height: 38px; padding: 0 11px; font-size: 11px; }
  .home-hero { display: flex; min-height: 0; flex-direction: column; align-items: stretch; gap: 26px; padding-top: 44px; padding-bottom: 38px; }
  .home-hero h1 { margin-top: 18px; font-size: clamp(38px, 11vw, 50px); line-height: 1.18; }
  .home-hero__lead { margin-top: 17px; font-size: 13px; line-height: 1.78; }
  .home-hero__actions { margin-top: 22px; }
  .home-hero__actions .home-button { flex: 1; min-width: 145px; padding: 0 10px; }
  .home-hero__proof { gap: 10px; margin-top: 17px; }
  .home-hero__proof span { font-size: 10px; }
  .home-hero__visual { min-height: 320px; }
  .home-hero__artboard { inset: 0 0 31px; border-radius: 20px; }
  .home-hero__artboard img { width: 118%; height: 116%; }
  .home-route-card { right: 4%; bottom: 0; width: 82%; padding: 12px 13px; }
  .home-route-card code { margin: 9px 0; font-size: 10px; }
  .home-model-float { padding: 6px 8px 6px 6px; font-size: 9px; }
  .home-model-float--one { top: 8%; left: 2%; }
  .home-model-float--two { top: 38%; right: -1%; }
  .home-model-float--three { bottom: 21%; left: 1%; }
  .home-model-band { padding: 19px 18px; border-radius: 16px; }
  .home-model-band h2 { font-size: 18px; }
  .home-model-band__logos { margin-right: -8px; margin-left: -8px; grid-template-columns: repeat(8, minmax(86px, 1fr)); }
  .home-model-chip { padding: 7px 5px; font-size: 10px; }
  .home-model-chip__icon { width: 26px; height: 26px; }
  .home-section { padding-top: 68px; }
  .home-section__heading { margin-bottom: 23px; }
  .home-section__heading h2, .home-pricing h2, .home-final h2 { font-size: 27px; line-height: 1.3; }
  .home-section__heading p { font-size: 13px; }
  .home-section__heading--row { display: block; }
  .home-text-link { margin-top: 17px; }
  .home-feature-grid, .home-model-grid, .home-steps { grid-template-columns: 1fr; }
  .home-feature-card { min-height: 164px; padding: 18px; }
  .home-feature-card h3 { margin-top: 17px; }
  .home-model-card { min-height: 154px; }
  .home-model-card p { min-height: 0; }
  .home-pricing { margin-top: 68px; border-radius: 17px; }
  .home-pricing__aside { padding: 30px 24px; }
  .home-pricing__actions { gap: 15px; }
  .home-pricing__plans { grid-template-columns: 1fr; padding: 7px; }
  .home-plan-card { min-height: 0; padding: 22px 17px; border-right: 0; border-bottom: 1px solid #e7eef7; }
  .home-plan-card:last-child { border-bottom: 0; }
  .home-plan-card p { min-height: 0; }
  .home-plan-card ul { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .home-step { min-height: 142px; padding: 18px; }
  .home-step__icon { position: absolute; top: 24px; right: 22px; margin-top: 0; }
  .home-step h3 { margin-top: 22px; }
  .home-step__connector { top: auto; right: auto; bottom: -25px; left: calc(50% - 18px); transform: rotate(90deg); }
  .home-final { display: block; margin-top: 68px; padding: 24px 20px; }
  .home-final .home-button { margin-top: 23px; }
  .home-footer { margin-top: 72px; padding: 28px 0; }
  .home-footer__inner { display: grid; gap: 22px; }
  .home-footer__links { flex-wrap: wrap; gap: 14px 20px; margin-left: 0; }
  .home-footer__copyright { order: 3; }
}

@media (prefers-reduced-motion: reduce) {
  .lingqu-home *,
  .lingqu-home *::before,
  .lingqu-home *::after { scroll-behavior: auto !important; transition-duration: 0.01ms !important; animation-duration: 0.01ms !important; animation-iteration-count: 1 !important; }
}
</style>
