<template>
  <UserWorkspaceLayout>
    <div data-testid="profile-shell" class="lingqu-profile">
      <section class="lingqu-profile__grid" aria-label="账户概览">
        <article class="lingqu-profile__identity">
          <div class="lingqu-profile__avatar" aria-hidden="true">
            <span>{{ userInitials }}</span>
            <i></i>
          </div>
          <div class="lingqu-profile__identity-text">
            <small>邮箱名</small>
            <strong>{{ user?.email || '-' }}</strong>
          </div>
        </article>

        <article v-for="item in accountStats" :key="item.label" class="lingqu-profile__stat">
          <span :class="item.className">
            <Icon :name="item.icon" size="md" />
          </span>
          <small>{{ item.label }}</small>
          <strong>{{ item.value }}</strong>
        </article>
      </section>

      <section class="lingqu-profile__content">
        <article class="lingqu-profile__panel lingqu-profile__contact">
          <div class="lingqu-profile__panel-head">
            <span>
              <Icon name="chat" size="md" />
            </span>
            <div>
              <h2>客户联系方式</h2>
              <p>遇到充值、Key、模型调用问题时，用这里联系支持。</p>
            </div>
          </div>

          <div v-if="contactInfo" class="lingqu-profile__contact-body">
            <div class="lingqu-profile__qr">
              <canvas ref="contactQrCanvas" aria-label="客户联系方式二维码"></canvas>
            </div>
            <div class="lingqu-profile__contact-text">
              <small>扫码或复制联系方式</small>
              <strong>{{ contactInfo }}</strong>
              <button type="button" @click="copyContact">
                <Icon name="copy" size="sm" />
                复制
              </button>
            </div>
          </div>

          <div v-else class="lingqu-profile__empty-contact">
            <Icon name="chat" size="lg" />
            <p>管理员暂未配置客服联系方式。</p>
          </div>
        </article>

        <article class="lingqu-profile__panel">
          <div class="lingqu-profile__panel-head">
            <span>
              <Icon name="bell" size="md" />
            </span>
            <div>
              <h2>余额不足提醒</h2>
              <p>余额低于阈值时提醒你，避免调用突然中断。</p>
            </div>
          </div>

          <form class="lingqu-profile__notify" @submit.prevent="saveBalanceNotify">
            <label class="lingqu-profile__switch">
              <input v-model="notifyForm.enabled" type="checkbox" />
              <span></span>
              <strong>{{ notifyForm.enabled ? '已开启提醒' : '未开启提醒' }}</strong>
            </label>

            <label class="lingqu-profile__field">
              <span>提醒阈值</span>
              <div>
                <i>$</i>
                <input
                  v-model.number="notifyForm.threshold"
                  type="number"
                  min="0"
                  step="0.01"
                  :placeholder="systemDefaultThreshold > 0 ? `${systemDefaultThreshold}` : '0.00'"
                />
              </div>
            </label>

            <button type="submit" :disabled="savingNotify">
              <Icon :name="savingNotify ? 'refresh' : 'check'" size="sm" :class="{ 'animate-spin': savingNotify }" />
              {{ savingNotify ? '保存中' : '保存提醒' }}
            </button>
          </form>
        </article>

        <article class="lingqu-profile__panel lingqu-profile__password">
          <div class="lingqu-profile__panel-head">
            <span>
              <Icon name="lock" size="md" />
            </span>
            <div>
              <h2>修改密码</h2>
              <p>定期更新密码，保护你的 Key 和余额安全。</p>
            </div>
          </div>
          <ProfilePasswordForm embedded />
        </article>
        <ProfileWalletPanel />
      </section>
    </div>
  </UserWorkspaceLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import QRCode from 'qrcode'
import UserWorkspaceLayout from '@/components/layout/UserWorkspaceLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import ProfilePasswordForm from '@/components/user/profile/ProfilePasswordForm.vue'
import ProfileWalletPanel from '@/components/user/profile/ProfileWalletPanel.vue'
import { userAPI } from '@/api'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { extractApiErrorMessage } from '@/utils/apiError'

const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

const user = computed(() => authStore.user)
const contactQrCanvas = ref<HTMLCanvasElement | null>(null)
const contactInfo = ref('')
const systemDefaultThreshold = ref(0)
const savingNotify = ref(false)
const notifyForm = reactive({
  enabled: true,
  threshold: 0
})

const userInitials = computed(() => {
  const source = user.value?.username || user.value?.email || 'AI'
  return source.slice(0, 2).toUpperCase()
})

const accountStats = computed(() => [
  {
    label: '账户余额',
    value: `$${Number(user.value?.balance || 0).toFixed(2)}`,
    icon: 'dollar' as const,
    className: 'lingqu-profile__stat-icon lingqu-profile__stat-icon--yellow'
  },
  {
    label: '并发数',
    value: `${user.value?.concurrency ?? 0}`,
    icon: 'server' as const,
    className: 'lingqu-profile__stat-icon lingqu-profile__stat-icon--cyan'
  },
  {
    label: '注册时间',
    value: formatDate(user.value?.created_at),
    icon: 'calendar' as const,
    className: 'lingqu-profile__stat-icon lingqu-profile__stat-icon--pink'
  }
])

function formatDate(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  })
}

function syncNotifyForm() {
  notifyForm.enabled = user.value?.balance_notify_enabled ?? true
  notifyForm.threshold = user.value?.balance_notify_threshold ?? systemDefaultThreshold.value ?? 0
}

async function renderContactQr() {
  await nextTick()
  if (!contactQrCanvas.value || !contactInfo.value) return
  await QRCode.toCanvas(contactQrCanvas.value, contactInfo.value, {
    width: 148,
    margin: 1,
    color: {
      dark: '#211f1c',
      light: '#fffdf5'
    }
  })
}

function copyContact() {
  if (!contactInfo.value) return
  copyToClipboard(contactInfo.value, '联系方式已复制')
}

async function saveBalanceNotify() {
  savingNotify.value = true
  try {
    const updated = await userAPI.updateProfile({
      balance_notify_enabled: notifyForm.enabled,
      balance_notify_threshold: notifyForm.threshold && notifyForm.threshold > 0 ? notifyForm.threshold : 0
    })
    authStore.user = updated
    syncNotifyForm()
    appStore.showSuccess('余额提醒已保存')
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, '保存余额提醒失败'))
  } finally {
    savingNotify.value = false
  }
}

watch(contactInfo, () => {
  renderContactQr().catch((error) => {
    console.warn('Failed to render contact QR:', error)
  })
})

watch(user, syncNotifyForm, { immediate: true })

onMounted(async () => {
  const profileLoad = authStore.refreshUser()
    .then(syncNotifyForm)
    .catch((error) => {
      console.error('Failed to refresh profile:', error)
    })

  const settingsLoad = appStore.fetchPublicSettings()
    .then((settings) => {
      contactInfo.value = settings?.contact_info || ''
      systemDefaultThreshold.value = settings?.balance_low_notify_threshold ?? 0
      syncNotifyForm()
    })
    .catch((error) => {
      console.error('Failed to load profile settings:', error)
    })

  await Promise.all([profileLoad, settingsLoad])
  await renderContactQr()
})
</script>

<style scoped>
.lingqu-profile {
  display: grid;
  gap: 1rem;
}

.lingqu-profile__hero,
.lingqu-profile__identity,
.lingqu-profile__stat,
.lingqu-profile__panel {
  position: relative;
  overflow: hidden;
  border: 3px solid #211f1c;
  background: rgba(255, 255, 255, 0.84);
  box-shadow: 7px 7px 0 rgba(33, 31, 28, 0.88);
}

.lingqu-profile__hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-radius: 28px;
  background:
    radial-gradient(circle at 92% 14%, rgba(8, 169, 214, 0.22), transparent 30%),
    linear-gradient(135deg, rgba(255, 247, 208, 0.94), rgba(255, 255, 255, 0.88));
  padding: clamp(1.2rem, 3vw, 1.8rem);
  animation: profileRise 480ms ease both;
}

.lingqu-profile__eyebrow {
  display: inline-flex;
  border: 2px solid #211f1c;
  border-radius: 999px;
  background: #fff7d0;
  box-shadow: 3px 3px 0 rgba(33, 31, 28, 0.78);
  padding: 0.28rem 0.65rem;
  font-size: 0.74rem;
  font-weight: 950;
}

.lingqu-profile__hero h1 {
  margin-top: 0.65rem;
  font-family: theme('fontFamily.display');
  font-size: clamp(2.35rem, 5vw, 4rem);
  font-weight: 950;
  letter-spacing: 0;
  line-height: 0.95;
  color: #ff5f8f;
  text-shadow: 3px 3px 0 #211f1c;
}

.lingqu-profile__hero p {
  margin-top: 0.65rem;
  max-width: 43rem;
  color: rgba(33, 31, 28, 0.64);
  font-weight: 800;
  line-height: 1.7;
}

.lingqu-profile__recharge {
  display: inline-flex;
  min-height: 3.1rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  border: 3px solid #211f1c;
  border-radius: 18px;
  background: linear-gradient(135deg, #ff7aa5, #ffd95a);
  box-shadow: 5px 5px 0 rgba(33, 31, 28, 0.86);
  color: #211f1c;
  padding: 0 1rem;
  font-weight: 950;
  transition: transform 150ms ease, box-shadow 150ms ease;
}

.lingqu-profile__recharge:hover {
  transform: translate(-2px, -3px) rotate(-1deg);
  box-shadow: 7px 7px 0 rgba(33, 31, 28, 0.9);
}

.lingqu-profile__grid {
  display: grid;
  grid-template-columns: minmax(18rem, 1.4fr) repeat(3, minmax(0, 1fr));
  gap: 0.85rem;
}

.lingqu-profile__identity,
.lingqu-profile__stat {
  min-height: 8.2rem;
  border-radius: 24px;
  padding: 1rem;
  transition: transform 160ms ease, box-shadow 160ms ease;
  animation: profileRise 480ms ease 80ms both;
}

.lingqu-profile__identity:hover,
.lingqu-profile__stat:hover,
.lingqu-profile__panel:hover {
  transform: translateY(-3px);
  box-shadow: 9px 9px 0 rgba(33, 31, 28, 0.9);
}

.lingqu-profile__identity {
  display: flex;
  align-items: center;
  gap: 1rem;
  background:
    radial-gradient(circle at 86% 16%, rgba(255, 95, 143, 0.18), transparent 28%),
    rgba(255, 255, 255, 0.84);
}

.lingqu-profile__avatar {
  position: relative;
  display: grid;
  width: 5rem;
  height: 5rem;
  flex: 0 0 auto;
  place-items: center;
  border: 3px solid #211f1c;
  border-radius: 42%;
  background: #fff0bd;
  box-shadow: 5px 5px 0 rgba(33, 31, 28, 0.84);
}

.lingqu-profile__avatar span {
  font-family: theme('fontFamily.display');
  font-size: 1.5rem;
  font-weight: 950;
}

.lingqu-profile__avatar i {
  position: absolute;
  top: -0.58rem;
  width: 3.6rem;
  height: 1.22rem;
  border: 3px solid #211f1c;
  border-radius: 999px 999px 10px 10px;
  background: #ff5f8f;
}

.lingqu-profile__identity-text,
.lingqu-profile__stat {
  display: grid;
  gap: 0.35rem;
  min-width: 0;
}

.lingqu-profile__identity-text small,
.lingqu-profile__stat small {
  color: rgba(33, 31, 28, 0.52);
  font-size: 0.78rem;
  font-weight: 950;
}

.lingqu-profile__identity-text strong,
.lingqu-profile__stat strong {
  min-width: 0;
  overflow-wrap: anywhere;
  font-family: theme('fontFamily.display');
  font-size: clamp(1.25rem, 2vw, 1.65rem);
  font-weight: 950;
  line-height: 1.05;
}

.lingqu-profile__stat-icon {
  display: grid;
  width: 2.75rem;
  height: 2.75rem;
  place-items: center;
  border: 3px solid #211f1c;
  border-radius: 16px;
  box-shadow: 3px 3px 0 rgba(33, 31, 28, 0.82);
}

.lingqu-profile__stat-icon--yellow {
  background: #ffd447;
}

.lingqu-profile__stat-icon--cyan {
  background: #4ee9ff;
}

.lingqu-profile__stat-icon--pink {
  background: #ff9fbc;
}

.lingqu-profile__content {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 1rem;
}

.lingqu-profile__panel {
  border-radius: 24px;
  padding: 1.1rem;
  animation: profileRise 480ms ease 160ms both;
}

.lingqu-profile__password {
  grid-column: 1 / -1;
}

.lingqu-profile__panel-head {
  display: flex;
  align-items: flex-start;
  gap: 0.8rem;
}

.lingqu-profile__panel-head > span {
  display: grid;
  width: 2.8rem;
  height: 2.8rem;
  flex: 0 0 auto;
  place-items: center;
  border: 3px solid #211f1c;
  border-radius: 16px;
  background: #fff7d0;
  box-shadow: 3px 3px 0 rgba(33, 31, 28, 0.82);
}

.lingqu-profile__panel-head h2 {
  font-family: theme('fontFamily.display');
  font-size: 1.55rem;
  font-weight: 950;
  line-height: 1.05;
}

.lingqu-profile__panel-head p {
  margin-top: 0.28rem;
  color: rgba(33, 31, 28, 0.58);
  font-size: 0.9rem;
  font-weight: 750;
  line-height: 1.55;
}

.lingqu-profile__contact-body {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-top: 1.1rem;
}

.lingqu-profile__qr {
  display: grid;
  width: 10.2rem;
  height: 10.2rem;
  flex: 0 0 auto;
  place-items: center;
  border: 3px solid #211f1c;
  border-radius: 22px;
  background: #fffdf5;
  box-shadow: 5px 5px 0 rgba(33, 31, 28, 0.82);
}

.lingqu-profile__qr canvas {
  width: 9.25rem !important;
  height: 9.25rem !important;
}

.lingqu-profile__contact-text {
  display: grid;
  gap: 0.45rem;
  min-width: 0;
}

.lingqu-profile__contact-text small {
  color: rgba(33, 31, 28, 0.5);
  font-size: 0.78rem;
  font-weight: 950;
}

.lingqu-profile__contact-text strong {
  overflow-wrap: anywhere;
  font-size: 1rem;
  font-weight: 900;
  line-height: 1.5;
}

.lingqu-profile__contact-text button,
.lingqu-profile__notify button {
  display: inline-flex;
  min-height: 2.55rem;
  width: fit-content;
  align-items: center;
  justify-content: center;
  gap: 0.38rem;
  border: 3px solid #211f1c;
  border-radius: 15px;
  background: #ffd447;
  box-shadow: 3px 3px 0 rgba(33, 31, 28, 0.82);
  color: #211f1c;
  padding: 0 0.8rem;
  font-weight: 950;
  transition: transform 140ms ease, box-shadow 140ms ease;
}

.lingqu-profile__contact-text button:hover,
.lingqu-profile__notify button:hover:not(:disabled) {
  transform: translate(-1px, -2px);
  box-shadow: 5px 5px 0 rgba(33, 31, 28, 0.86);
}

.lingqu-profile__notify button:disabled {
  cursor: not-allowed;
  opacity: 0.66;
}

.lingqu-profile__empty-contact {
  display: grid;
  min-height: 10rem;
  place-items: center;
  gap: 0.55rem;
  margin-top: 1rem;
  border: 2px dashed rgba(33, 31, 28, 0.25);
  border-radius: 18px;
  color: rgba(33, 31, 28, 0.54);
  font-weight: 800;
}

.lingqu-profile__notify {
  display: grid;
  gap: 0.85rem;
  margin-top: 1.1rem;
}

.lingqu-profile__switch {
  display: flex;
  align-items: center;
  gap: 0.7rem;
  width: fit-content;
  cursor: pointer;
}

.lingqu-profile__switch input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.lingqu-profile__switch span {
  position: relative;
  width: 3.25rem;
  height: 1.78rem;
  border: 3px solid #211f1c;
  border-radius: 999px;
  background: #fff;
  box-shadow: 3px 3px 0 rgba(33, 31, 28, 0.78);
  transition: background 150ms ease;
}

.lingqu-profile__switch span::after {
  content: '';
  position: absolute;
  left: 0.16rem;
  top: 0.16rem;
  width: 1.02rem;
  height: 1.02rem;
  border: 2px solid #211f1c;
  border-radius: 999px;
  background: #ff9fbc;
  transition: transform 150ms ease, background 150ms ease;
}

.lingqu-profile__switch input:checked + span {
  background: #d7f8ff;
}

.lingqu-profile__switch input:checked + span::after {
  transform: translateX(1.44rem);
  background: #2ecf9f;
}

.lingqu-profile__switch strong {
  font-weight: 950;
}

.lingqu-profile__field {
  display: grid;
  gap: 0.38rem;
}

.lingqu-profile__field > span {
  color: rgba(33, 31, 28, 0.58);
  font-size: 0.82rem;
  font-weight: 950;
}

.lingqu-profile__field div {
  display: flex;
  align-items: center;
  border: 3px solid #211f1c;
  border-radius: 16px;
  background: #fffdf5;
  box-shadow: 3px 3px 0 rgba(33, 31, 28, 0.82);
  overflow: hidden;
}

.lingqu-profile__field i {
  display: grid;
  width: 2.6rem;
  align-self: stretch;
  place-items: center;
  border-right: 2px solid rgba(33, 31, 28, 0.14);
  background: #fff7d0;
  font-style: normal;
  font-weight: 950;
}

.lingqu-profile__field input {
  min-width: 0;
  width: 100%;
  border: 0;
  background: transparent;
  padding: 0.78rem 0.85rem;
  font-weight: 900;
  outline: none;
}

.lingqu-profile__password :deep(.input-label) {
  color: rgba(33, 31, 28, 0.62);
  font-weight: 950;
}

.lingqu-profile__password :deep(.input),
.lingqu-profile__password :deep(.btn) {
  border-color: #211f1c;
}

.lingqu-profile__password :deep(.btn-primary) {
  background: linear-gradient(135deg, #ff7aa5, #ffd95a);
  color: #211f1c;
  font-weight: 950;
}

@keyframes profileRise {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 1020px) {
  .lingqu-profile__grid,
  .lingqu-profile__content {
    grid-template-columns: 1fr 1fr;
  }

  .lingqu-profile__identity {
    grid-column: 1 / -1;
  }
}

@media (max-width: 720px) {
  .lingqu-profile__hero,
  .lingqu-profile__contact-body {
    align-items: flex-start;
    flex-direction: column;
  }

  .lingqu-profile__grid,
  .lingqu-profile__content {
    grid-template-columns: 1fr;
  }

  .lingqu-profile__recharge {
    width: 100%;
  }

  .lingqu-profile__qr {
    width: 100%;
  }
}

/* Calm profile skin: friendly mascot details, quieter product surfaces. */
.lingqu-profile__hero,
.lingqu-profile__identity,
.lingqu-profile__stat,
.lingqu-profile__panel {
  border: 1px solid rgba(33, 31, 28, 0.1);
  background: rgba(255, 255, 255, 0.84);
  box-shadow: 0 10px 26px rgba(29, 42, 42, 0.07);
}

.lingqu-profile__hero {
  border-radius: 18px;
  background:
    radial-gradient(circle at 92% 14%, rgba(72, 185, 200, 0.1), transparent 30%),
    rgba(255, 255, 255, 0.88);
  padding: clamp(0.78rem, 1.7vw, 1rem);
}

.lingqu-profile__eyebrow,
.lingqu-profile__recharge,
.lingqu-profile__avatar,
.lingqu-profile__avatar i,
.lingqu-profile__stat-icon,
.lingqu-profile__panel-head > span,
.lingqu-profile__qr,
.lingqu-profile__contact-text button,
.lingqu-profile__notify button,
.lingqu-profile__switch span,
.lingqu-profile__switch span::after,
.lingqu-profile__field div,
.lingqu-profile__password :deep(.input),
.lingqu-profile__password :deep(.btn) {
  border-color: rgba(33, 31, 28, 0.12);
  box-shadow: none;
}

.lingqu-profile__eyebrow {
  border-width: 1px;
  background: #fff8df;
}

.lingqu-profile__hero h1 {
  color: #211f1c;
  margin-top: 0.42rem;
  font-size: clamp(1.45rem, 2.4vw, 1.92rem);
  line-height: 1.02;
  text-shadow: none;
}

.lingqu-profile__hero p {
  margin-top: 0.38rem;
  line-height: 1.45;
}

.lingqu-profile__recharge {
  min-height: 2.55rem;
  border-width: 1px;
  border-radius: 14px;
}

.lingqu-profile__recharge,
.lingqu-profile__password :deep(.btn-primary) {
  background: linear-gradient(135deg, #f8e08a, #f4b4bd);
}

.lingqu-profile__recharge:hover,
.lingqu-profile__identity:hover,
.lingqu-profile__stat:hover,
.lingqu-profile__panel:hover,
.lingqu-profile__contact-text button:hover,
.lingqu-profile__notify button:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 14px 30px rgba(29, 42, 42, 0.1);
}

.lingqu-profile__identity,
.lingqu-profile__stat,
.lingqu-profile__panel {
  border-radius: 18px;
}

.lingqu-profile__identity {
  background:
    radial-gradient(circle at 86% 16%, rgba(244, 180, 189, 0.12), transparent 28%),
    rgba(255, 255, 255, 0.86);
}

.lingqu-profile__avatar {
  border-width: 1px;
  background: #fff8df;
}

.lingqu-profile__avatar i {
  border-width: 1px;
  background: #f4b4bd;
}

.lingqu-profile__stat-icon,
.lingqu-profile__panel-head > span,
.lingqu-profile__qr {
  border-width: 1px;
}

.lingqu-profile__stat-icon--yellow,
.lingqu-profile__panel-head > span,
.lingqu-profile__field i {
  background: #fff8df;
}

.lingqu-profile__stat-icon--cyan,
.lingqu-profile__switch input:checked + span {
  background: #edfafa;
}

.lingqu-profile__stat-icon--pink,
.lingqu-profile__switch span::after {
  background: #fff0f4;
}

.lingqu-profile__switch span {
  border-width: 1px;
  background: #fff;
}

.lingqu-profile__switch span::after {
  border-width: 1px;
}

.lingqu-profile__switch input:checked + span::after {
  background: #54bea0;
}

.lingqu-profile__field div {
  border-width: 1px;
  background: rgba(255, 255, 255, 0.9);
}

.lingqu-profile__password :deep(.btn:hover:not(:disabled)) {
  transform: translateY(-2px);
  box-shadow: 0 14px 30px rgba(29, 42, 42, 0.1);
}
</style>
