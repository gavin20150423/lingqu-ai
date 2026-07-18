<template>
  <aside
    class="sidebar"
    :class="[
      sidebarCollapsed ? 'w-[72px]' : 'w-64',
      { '-translate-x-full lg:translate-x-0': !mobileOpen }
    ]"
  >
    <div class="sidebar-header" :class="{ 'sidebar-header-collapsed': sidebarCollapsed }">
      <div class="sidebar-logo flex h-9 w-9 items-center justify-center overflow-hidden rounded-xl bg-white ring-1 ring-slate-200">
        <img
          v-if="settingsLoaded"
          :src="siteLogo || '/logo.png'"
          alt="Logo"
          class="h-full w-full object-contain"
        />
      </div>
      <div
        class="sidebar-brand"
        :class="{ 'sidebar-brand-collapsed': sidebarCollapsed }"
        :aria-hidden="sidebarCollapsed ? 'true' : 'false'"
      >
        <span class="sidebar-brand-title text-lg font-bold text-gray-900 dark:text-white">
          {{ siteName }}
        </span>
        <VersionBadge :version="siteVersion" />
      </div>
    </div>

    <!-- Navigation -->
    <nav ref="sidebarNavRef" class="sidebar-nav scrollbar-hide">
      <!-- Admin View: Admin menu first, then personal menu -->
      <template v-if="isAdmin">
        <div class="sidebar-section">
          <template v-for="item in adminNavItems" :key="item.path">
            <template v-if="item.children?.length">
              <button
                type="button"
                class="sidebar-link mb-1 w-full"
                :class="{
                  'sidebar-link-active': isGroupActive(item) && !isGroupExpanded(item),
                  'sidebar-link-collapsed': sidebarCollapsed
                }"
                :title="sidebarCollapsed ? item.label : undefined"
                @click="handleGroupClick(item)"
              >
                <SidebarItemIcon :item="item" />
                <span
                  class="sidebar-label sidebar-label-flex"
                  :class="{ 'sidebar-label-collapsed': sidebarCollapsed }"
                  :aria-hidden="sidebarCollapsed ? 'true' : 'false'"
                >
                  <span class="min-w-0 truncate">{{ item.label }}</span>
                  <Icon
                    name="chevronDown"
                    size="sm"
                    class="flex-shrink-0 transition-transform duration-200"
                    :class="isGroupExpanded(item) ? 'rotate-180' : ''"
                  />
                </span>
              </button>

              <div
                v-if="!sidebarCollapsed && isGroupExpanded(item)"
                class="mb-1 ml-4 border-l border-gray-200 pl-2 dark:border-dark-600"
              >
                <router-link
                  v-for="child in item.children"
                  :key="child.path"
                  :to="child.path"
                  class="sidebar-link mb-0.5 py-1.5 text-sm"
                  :class="{ 'sidebar-link-active': route.path === child.path }"
                  @click="handleMenuItemClick(child.path)"
                >
                  <SidebarItemIcon :item="child" small />
                  <span>{{ child.label }}</span>
                </router-link>
              </div>
            </template>

            <router-link
              v-else
              :to="item.path"
              class="sidebar-link mb-1"
              :class="{ 'sidebar-link-active': isActive(item.path), 'sidebar-link-collapsed': sidebarCollapsed }"
              :title="sidebarCollapsed ? item.label : undefined"
              :id="
                item.path === '/admin/accounts'
                  ? 'sidebar-channel-manage'
                  : item.path === '/admin/groups'
                    ? 'sidebar-group-manage'
                    : item.path === '/admin/redeem'
                      ? 'sidebar-wallet'
                      : undefined
              "
              @click="handleMenuItemClick(item.path)"
            >
              <SidebarItemIcon :item="item" />
              <span
                class="sidebar-label"
                :class="{ 'sidebar-label-collapsed': sidebarCollapsed }"
                :aria-hidden="sidebarCollapsed ? 'true' : 'false'"
              >
                {{ item.label }}
              </span>
            </router-link>
          </template>
        </div>

        <router-link
          v-if="!authStore.isSimpleMode"
          to="/dashboard"
          class="sidebar-link mx-3 mb-4 mt-2"
          :class="{ 'sidebar-link-collapsed': sidebarCollapsed }"
          :title="sidebarCollapsed ? t('nav.userPortal') : undefined"
          @click="handleMenuItemClick('/dashboard')"
        >
          <SidebarItemIcon :item="{ path: '/dashboard', label: t('nav.userPortal'), icon: 'user' }" />
          <span
            class="sidebar-label"
            :class="{ 'sidebar-label-collapsed': sidebarCollapsed }"
            :aria-hidden="sidebarCollapsed ? 'true' : 'false'"
          >
            {{ t('nav.userPortal') }}
          </span>
        </router-link>
      </template>

      <template v-else-if="!appStore.backendModeEnabled">
        <div class="sidebar-section">
          <router-link
            v-for="item in userNavItems"
            :key="item.path"
            :to="item.path"
            class="sidebar-link mb-1"
            :class="{ 'sidebar-link-active': isActive(item.path), 'sidebar-link-collapsed': sidebarCollapsed }"
            :title="sidebarCollapsed ? item.label : undefined"
            :data-tour="item.path === '/keys' ? 'sidebar-my-keys' : undefined"
            @click="handleMenuItemClick(item.path)"
          >
            <SidebarItemIcon :item="item" />
            <span
              class="sidebar-label"
              :class="{ 'sidebar-label-collapsed': sidebarCollapsed }"
              :aria-hidden="sidebarCollapsed ? 'true' : 'false'"
            >
              {{ item.label }}
            </span>
          </router-link>
        </div>
      </template>
    </nav>

    <div class="mt-auto border-t border-gray-100 p-3 dark:border-dark-800">
      <button
        @click="toggleSidebar"
        class="sidebar-link w-full"
        :class="{ 'sidebar-link-collapsed': sidebarCollapsed }"
        :title="sidebarCollapsed ? t('nav.expand') : t('nav.collapse')"
      >
        <Icon
          :name="sidebarCollapsed ? 'chevronRight' : 'chevronLeft'"
          size="md"
          class="flex-shrink-0"
        />
        <span
          class="sidebar-label"
          :class="{ 'sidebar-label-collapsed': sidebarCollapsed }"
          :aria-hidden="sidebarCollapsed ? 'true' : 'false'"
        >
          {{ t('nav.collapse') }}
        </span>
      </button>
    </div>
  </aside>

  <transition name="fade">
    <div
      v-if="mobileOpen"
      class="fixed inset-0 z-30 bg-black/50 lg:hidden"
      @click="closeMobile"
    ></div>
  </transition>
</template>

<script setup lang="ts">
import {
  computed,
  defineComponent,
  h,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch
} from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAdminSettingsStore, useAppStore, useAuthStore, useOnboardingStore } from '@/stores'
import VersionBadge from '@/components/common/VersionBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeSvg } from '@/utils/sanitize'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, makeSidebarFlag } from '@/utils/featureFlags'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'

type IconName =
  | 'badge'
  | 'bell'
  | 'chart'
  | 'chat'
  | 'creditCard'
  | 'cog'
  | 'document'
  | 'gift'
  | 'globe'
  | 'grid'
  | 'key'
  | 'server'
  | 'shield'
  | 'sync'
  | 'user'
  | 'users'

interface NavItem {
  path: string
  label: string
  icon: IconName
  iconSvg?: string
  hideInSimpleMode?: boolean
  children?: NavItem[]
  expandOnly?: boolean
  featureFlag?: () => boolean | undefined
}

const SidebarItemIcon = defineComponent({
  name: 'SidebarItemIcon',
  props: {
    item: { type: Object as () => NavItem, required: true },
    small: { type: Boolean, default: false }
  },
  setup(props) {
    return () => {
      if (props.item.iconSvg) {
        return h('span', {
          class: [
            props.small ? 'h-4 w-4' : 'h-5 w-5',
            'flex-shrink-0 sidebar-svg-icon'
          ],
          innerHTML: sanitizeSvg(props.item.iconSvg)
        })
      }

      return h(Icon, {
        name: props.item.icon,
        size: props.small ? 'sm' : 'md',
        class: 'flex-shrink-0'
      })
    }
  }
})

function applyFeatureFlags(items: NavItem[]): NavItem[] {
  const out: NavItem[] = []
  for (const item of items) {
    if (item.featureFlag && item.featureFlag() === false) continue
    if (item.children) {
      out.push({ ...item, children: applyFeatureFlags(item.children) })
    } else {
      out.push(item)
    }
  }
  return out
}

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const onboardingStore = useOnboardingStore()
const adminSettingsStore = useAdminSettingsStore()

const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const mobileOpen = computed(() => appStore.mobileOpen)
const isAdmin = computed(() => authStore.isAdmin)
const sidebarNavRef = ref<HTMLElement | null>(null)

// Track which parent nav groups are expanded
const expandedGroups = ref<Set<string>>(new Set())

const siteName = computed(() => appStore.siteName)
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteVersion = computed(() => appStore.siteVersion)
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const flagChannelMonitor = makeSidebarFlag(FeatureFlags.channelMonitor)
const flagPayment = makeSidebarFlag(FeatureFlags.payment)
const flagAvailableChannels = makeSidebarFlag(FeatureFlags.availableChannels)
const flagAffiliate = makeSidebarFlag(FeatureFlags.affiliate)
const flagRiskControl = makeSidebarFlag(FeatureFlags.riskControl)
const flagOpsMonitoring = () => adminSettingsStore.opsMonitoringEnabled
const flagAdminPayment = () => adminSettingsStore.paymentEnabled
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
const flagBatchImageAccess = () => canUseBatchImage.value

const customMenuItemsForUser = computed(() => {
  const items = appStore.cachedPublicSettings?.custom_menu_items ?? []
  return items
    .filter((item) => item.visibility === 'user')
    .sort((a, b) => a.sort_order - b.sort_order)
})

const customMenuItemsForAdmin = computed(() => {
  return adminSettingsStore.customMenuItems
    .filter((item) => item.visibility === 'admin')
    .sort((a, b) => a.sort_order - b.sort_order)
})

function buildSelfNavItems(withDashboard: boolean): NavItem[] {
  const items: NavItem[] = []
  if (withDashboard) {
    items.push({ path: '/dashboard', label: t('nav.dashboard'), icon: 'grid' })
  }
  items.push(
    { path: '/keys', label: t('nav.apiKeys'), icon: 'key' },
    { path: '/store', label: '发卡商城', icon: 'gift', hideInSimpleMode: true },
    { path: '/accounts', label: '我的账号', icon: 'globe', hideInSimpleMode: true },
    { path: '/account-share', label: '账号广场', icon: 'users', hideInSimpleMode: true },
    { path: '/conversations', label: '工单服务', icon: 'chat', hideInSimpleMode: true },
    { path: '/batch-image', label: t('nav.batchImage'), icon: 'grid', hideInSimpleMode: true, featureFlag: flagBatchImageAccess },
    { path: '/usage', label: t('nav.usage'), icon: 'chart', hideInSimpleMode: true },
    { path: '/available-channels', label: t('nav.availableChannels'), icon: 'server', hideInSimpleMode: true, featureFlag: flagAvailableChannels },
    { path: '/monitor', label: t('nav.channelStatus'), icon: 'sync', featureFlag: flagChannelMonitor },
    { path: '/subscriptions', label: t('nav.mySubscriptions'), icon: 'creditCard', hideInSimpleMode: true },
    { path: '/purchase', label: t('nav.buySubscription'), icon: 'creditCard', hideInSimpleMode: true, featureFlag: flagPayment },
    { path: '/orders', label: t('nav.myOrders'), icon: 'document', hideInSimpleMode: true, featureFlag: flagPayment },
    { path: '/redeem', label: t('nav.redeem'), icon: 'gift', hideInSimpleMode: true },
    { path: '/affiliate', label: t('nav.affiliate'), icon: 'users', hideInSimpleMode: true, featureFlag: flagAffiliate },
    { path: '/profile', label: t('nav.profile'), icon: 'user' },
    ...customMenuItemsForUser.value.map((item): NavItem => ({
      path: `/custom/${item.id}`,
      label: item.label,
      icon: 'grid',
      iconSvg: item.icon_svg
    }))
  )
  return items
}

function finalizeNav(items: NavItem[]): NavItem[] {
  const visible = applyFeatureFlags(items)
  return authStore.isSimpleMode ? visible.filter(item => !item.hideInSimpleMode) : visible
}

const userNavItems = computed((): NavItem[] => finalizeNav(buildSelfNavItems(true)))

const adminNavItems = computed((): NavItem[] => {
  const baseItems: NavItem[] = [
    { path: '/admin/dashboard', label: t('nav.dashboard'), icon: 'grid' },
    { path: '/admin/ops', label: t('nav.ops'), icon: 'chart', featureFlag: flagOpsMonitoring },
    { path: '/admin/users', label: t('nav.users'), icon: 'users', hideInSimpleMode: true },
    { path: '/admin/groups', label: t('nav.groups'), icon: 'grid' },
    { path: '/admin/accounts', label: t('nav.accounts'), icon: 'globe' },
    { path: '/admin/community', label: '共享与商城', icon: 'gift', hideInSimpleMode: true },
    {
      path: '/admin/channels',
      label: t('nav.channelManagement'),
      icon: 'server',
      hideInSimpleMode: true,
      expandOnly: true,
      children: [
        { path: '/admin/channels/pricing', label: t('nav.channelPricing'), icon: 'creditCard' },
        { path: '/admin/channels/monitor', label: t('nav.channelMonitor'), icon: 'sync', featureFlag: flagChannelMonitor },
      ],
    },
    { path: '/admin/subscriptions', label: t('nav.subscriptions'), icon: 'creditCard', hideInSimpleMode: true },
    { path: '/admin/announcements', label: t('nav.announcements'), icon: 'bell' },
    { path: '/admin/proxies', label: t('nav.proxies'), icon: 'server' },
    {
      path: '/admin/security-audit',
      label: t('nav.securityAudit'),
      icon: 'shield',
      hideInSimpleMode: true,
      expandOnly: true,
      featureFlag: flagRiskControl,
      children: [
        { path: '/admin/risk-control', label: t('nav.contentModeration'), icon: 'shield' },
        { path: '/admin/prompt-audit', label: t('nav.promptAudit'), icon: 'shield' },
      ],
    },
    { path: '/admin/redeem', label: t('nav.redeemCodes'), icon: 'badge', hideInSimpleMode: true },
    { path: '/admin/promo-codes', label: t('nav.promoCodes'), icon: 'gift', hideInSimpleMode: true },
    {
      path: '/admin/affiliates',
      label: t('nav.affiliateManagement'),
      icon: 'users',
      hideInSimpleMode: true,
      expandOnly: true,
      featureFlag: flagAffiliate,
      children: [
        { path: '/admin/affiliates/invites', label: t('nav.affiliateInviteRecords'), icon: 'users' },
        { path: '/admin/affiliates/rebates', label: t('nav.affiliateRebateRecords'), icon: 'document' },
        { path: '/admin/affiliates/transfers', label: t('nav.affiliateTransferRecords'), icon: 'creditCard' },
      ],
    },
    {
      path: '/admin/orders',
      label: t('nav.orderManagement'),
      icon: 'document',
      hideInSimpleMode: true,
      expandOnly: true,
      featureFlag: flagAdminPayment,
      children: [
        { path: '/admin/orders/dashboard', label: t('nav.paymentDashboard'), icon: 'chart' },
        { path: '/admin/orders', label: t('nav.orderManagement'), icon: 'document' },
        { path: '/admin/orders/plans', label: t('nav.paymentPlans'), icon: 'creditCard' },
      ],
    },
    { path: '/admin/usage', label: t('nav.usage'), icon: 'chart' },
    { path: '/admin/audit-logs', label: t('nav.auditLogs'), icon: 'shield', hideInSimpleMode: true }
  ]

  const visible = applyFeatureFlags(baseItems)

  if (authStore.isSimpleMode) {
    const filtered = visible.filter(item => !item.hideInSimpleMode)
    filtered.push({ path: '/dashboard', label: t('nav.userPortal'), icon: 'user' })
    filtered.push({ path: '/admin/settings', label: t('nav.settings'), icon: 'cog' })
    for (const cm of customMenuItemsForAdmin.value) {
      filtered.push({ path: `/custom/${cm.id}`, label: cm.label, icon: 'grid', iconSvg: cm.icon_svg })
    }
    return filtered
  }

  visible.push({ path: '/admin/settings', label: t('nav.settings'), icon: 'cog' })
  for (const cm of customMenuItemsForAdmin.value) {
    visible.push({ path: `/custom/${cm.id}`, label: cm.label, icon: 'grid', iconSvg: cm.icon_svg })
  }
  return visible
})

function toggleSidebar() {
  appStore.toggleSidebar()
}

function closeMobile() {
  appStore.setMobileOpen(false)
}

function handleMenuItemClick(itemPath: string) {
  if (mobileOpen.value) {
    setTimeout(() => {
      appStore.setMobileOpen(false)
    }, 150)
  }

  const pathToSelector: Record<string, string> = {
    '/admin/groups': '#sidebar-group-manage',
    '/admin/accounts': '#sidebar-channel-manage',
    '/keys': '[data-tour="sidebar-my-keys"]'
  }

  const selector = pathToSelector[itemPath]
  if (selector && onboardingStore.isCurrentStep(selector)) {
    onboardingStore.nextStep(500)
  }
}

function isActive(path: string): boolean {
  return route.path === path || route.path.startsWith(path + '/')
}

function isGroupActive(item: NavItem): boolean {
  if (!item.children) return false
  return item.children.some(child => route.path === child.path)
}

function isGroupExpanded(item: NavItem): boolean {
  return expandedGroups.value.has(item.path) || isGroupActive(item)
}

function toggleGroup(item: NavItem) {
  if (expandedGroups.value.has(item.path)) {
    expandedGroups.value.delete(item.path)
  } else {
    expandedGroups.value.add(item.path)
  }
}

function handleGroupClick(item: NavItem) {
  if (sidebarCollapsed.value) return
  if (item.expandOnly) {
    toggleGroup(item)
    return
  }
  if (route.path !== item.path) {
    router.push(item.path)
  }
  if (!expandedGroups.value.has(item.path)) {
    expandedGroups.value.add(item.path)
  }
}

watch(
  isAdmin,
  (v) => {
    if (v) {
      adminSettingsStore.fetch()
    }
  },
  { immediate: true }
)

onMounted(() => {
  if (isAdmin.value) {
    adminSettingsStore.fetch()
  }
  void refreshBatchImageAccess()
  // Restore sidebar scroll position after route change re-mounts the component
  if (appStore.sidebarScrollTop > 0 && sidebarNavRef.value) {
    void nextTick(() => {
      if (sidebarNavRef.value) {
        sidebarNavRef.value.scrollTop = appStore.sidebarScrollTop
      }
    })
  }
})

onBeforeUnmount(() => {
  if (sidebarNavRef.value) {
    appStore.sidebarScrollTop = sidebarNavRef.value.scrollTop
  }
})
</script>

<style scoped>
.sidebar-logo {
  flex: 0 0 2.25rem;
  min-width: 2.25rem;
}

.sidebar-header-collapsed {
  gap: 0;
  padding-left: 1.125rem;
  padding-right: 1.125rem;
}

.sidebar-brand {
  min-width: 0;
  flex: 1 1 auto;
  white-space: nowrap;
  transition:
    max-width 0.22s ease,
    opacity 0.14s ease,
    transform 0.14s ease;
  max-width: 12rem;
}

.sidebar-brand-collapsed {
  max-width: 0;
  overflow: hidden;
  opacity: 0;
  transform: translateX(-4px);
  pointer-events: none;
}

.sidebar-brand-title {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-link-collapsed {
  gap: 0;
  padding-left: 0.875rem;
  padding-right: 0.875rem;
}

.sidebar-section-title {
  position: relative;
  display: flex;
  align-items: center;
  min-height: 1.25rem;
  overflow: hidden;
  white-space: nowrap;
}

.sidebar-section-title-text {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition:
    opacity 0.16s ease,
    transform 0.16s ease;
}

.sidebar-section-title::after {
  content: '';
  position: absolute;
  left: 0.75rem;
  right: 0.75rem;
  top: 50%;
  height: 1px;
  background: rgb(226 232 240);
  opacity: 0;
  transform: translateY(-50%);
  transition: opacity 0.18s ease;
}

.sidebar-section-title-text-collapsed {
  opacity: 0;
  transform: translateX(-4px);
}

.sidebar-section-title-collapsed::after {
  opacity: 1;
  transition-delay: 0.08s;
}

.sidebar-label {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition:
    max-width 0.2s ease,
    opacity 0.12s ease,
    transform 0.12s ease;
  max-width: 12rem;
}

.sidebar-label-flex {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.sidebar-label-collapsed {
  max-width: 0;
  opacity: 0;
  transform: translateX(-4px);
  pointer-events: none;
}

.sidebar-svg-icon {
  color: currentColor;
}

.sidebar-svg-icon :deep(svg) {
  display: block;
  width: 1.25rem;
  height: 1.25rem;
}

.sidebar {
  background: rgba(255, 255, 255, 0.96);
  border-right: 1px solid rgb(226 232 240);
  box-shadow: 8px 0 28px rgba(15, 23, 42, 0.04);
}

.sidebar-link {
  border: 1px solid transparent;
}

.sidebar-link-active {
  border-color: rgb(251 191 36 / 0.18);
}
</style>
