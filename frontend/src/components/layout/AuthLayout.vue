<template>
  <div class="auth-stage">
    <section class="auth-stage__visual" :aria-label="t('authLayout.visualLabel')">
      <div class="auth-stage__glow auth-stage__glow--blue" aria-hidden="true" />
      <div class="auth-stage__glow auth-stage__glow--violet" aria-hidden="true" />
      <div class="auth-stage__field" aria-hidden="true" />
      <router-link to="/" class="auth-stage__brand" :aria-label="`${siteName} ${t('authLayout.homeLabel')}`">
        <span class="auth-stage__brand-mark">
          <img :src="siteLogo || '/brand/lingqu-ai-logo.svg'" alt="" />
        </span>
        <span class="auth-stage__brand-copy">
          <strong>{{ siteName }}</strong>
          <small>{{ t('authLayout.brandTagline') }}</small>
        </span>
      </router-link>

      <div class="auth-stage__visual-content">
        <div class="auth-stage__artwork" aria-hidden="true">
          <img class="auth-stage__generated-art" src="/brand/auth-login-panel-generated.png" alt="" />
        </div>
      </div>
    </section>

    <main class="auth-stage__form-column">
      <div class="auth-stage__locale">
        <LocaleSwitcher />
      </div>

      <router-link to="/" class="auth-stage__mobile-brand" :aria-label="`${siteName} ${t('authLayout.homeLabel')}`">
        <img :src="siteLogo || '/brand/lingqu-ai-logo.svg'" alt="" />
        <span>
          <strong>{{ siteName }}</strong>
          <small>{{ t('authLayout.brandTagline') }}</small>
        </span>
      </router-link>

      <section class="auth-panel">
        <slot />

        <div class="auth-panel__footer">
          <slot name="footer" />
        </div>

        <div class="auth-panel__legal">
          &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
        </div>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import { useAppStore } from '@/stores'
import { resolveBrandLogo, resolveBrandName } from '@/constants/brand'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()
const { t } = useI18n()

const siteName = computed(() => resolveBrandName(appStore.siteName))
const siteLogo = computed(() => sanitizeUrl(resolveBrandLogo(appStore.siteLogo), { allowRelative: true, allowDataUrl: true }))
const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.auth-stage {
  --auth-ink: #172238;
  --auth-muted: #718096;
  --auth-primary: #2f7cf0;
  --auth-primary-dark: #1f65cf;
  display: grid;
  min-height: 100vh;
  min-height: 100dvh;
  grid-template-columns: minmax(0, 1.08fr) minmax(31rem, 0.92fr);
  overflow-x: hidden;
  background: #ffffff;
  color: var(--auth-ink);
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Noto Sans SC", sans-serif;
  font-synthesis: none;
}

.auth-stage__visual {
  position: relative;
  display: flex;
  min-width: 0;
  min-height: 100vh;
  min-height: 100dvh;
  flex-direction: column;
  overflow: hidden;
  isolation: isolate;
  padding: clamp(1.5rem, 3.2vw, 3.2rem) clamp(1.75rem, 5vw, 6.5rem) clamp(1.3rem, 2.8vw, 2.8rem);
  background:
    radial-gradient(circle at 11% 5%, rgba(255, 255, 255, 0.55), transparent 22%),
    linear-gradient(145deg, #9bdafa 0%, #d9f2fb 43%, #e9efff 76%, #f0edff 100%);
}

.auth-stage__visual::before,
.auth-stage__visual::after {
  display: none;
  position: absolute;
  content: '';
  pointer-events: none;
}

.auth-stage__visual::before {
  top: 17%;
  right: -16%;
  width: 62%;
  height: 38%;
  border: 1px solid rgba(255, 255, 255, 0.46);
  border-radius: 50%;
  transform: rotate(-19deg);
}

.auth-stage__visual::after {
  right: -9%;
  bottom: -24%;
  width: 72%;
  height: 48%;
  border-radius: 50% 0 0 0;
  background: linear-gradient(135deg, rgba(57, 168, 153, 0.8), rgba(185, 236, 210, 0.72) 58%, rgba(237, 249, 230, 0.35));
  transform: rotate(-8deg);
}

.auth-stage__glow,
.auth-stage__field {
  display: none;
  position: absolute;
  pointer-events: none;
}

.auth-stage__glow--blue {
  top: 7%;
  left: 10%;
  width: min(30rem, 52%);
  aspect-ratio: 1;
  border-radius: 50%;
  background: rgba(79, 190, 255, 0.24);
  filter: blur(34px);
}

.auth-stage__glow--violet {
  right: 8%;
  bottom: 8%;
  width: min(24rem, 42%);
  aspect-ratio: 1;
  border-radius: 50%;
  background: rgba(141, 132, 238, 0.17);
  filter: blur(38px);
}

.auth-stage__field {
  z-index: -1;
  bottom: -28%;
  left: -22%;
  width: 66%;
  height: 42%;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.48);
  filter: blur(2px);
  transform: rotate(12deg);
}

.auth-stage__brand {
  position: relative;
  z-index: 1;
}

.auth-stage__brand {
  display: inline-flex;
  width: fit-content;
  align-items: center;
  gap: 0.75rem;
  color: inherit;
  text-decoration: none;
  position: relative;
  z-index: 3;
}

.auth-stage__brand-mark {
  display: grid;
  width: 3rem;
  height: 3rem;
  flex: none;
  place-items: center;
  padding: 0.18rem;
  border: 1px solid rgba(255, 255, 255, 0.88);
  border-radius: 0.9rem;
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 0.65rem 1.2rem rgba(39, 93, 124, 0.16);
  box-sizing: border-box;
}

.auth-stage__brand-mark img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.auth-stage__brand-copy,
.auth-stage__mobile-brand span {
  display: grid;
  gap: 0.25rem;
}

.auth-stage__brand-copy strong,
.auth-stage__mobile-brand strong {
  color: #16243b;
  font-size: 1.05rem;
  font-weight: 760;
  line-height: 1;
}

.auth-stage__brand-copy small,
.auth-stage__mobile-brand small {
  color: #8492a7;
  font-size: 0.58rem;
  font-weight: 700;
  letter-spacing: 0.11em;
  line-height: 1;
}

.auth-stage__visual-content {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  justify-content: space-between;
  width: 100%;
  max-width: none;
  margin: 0;
  z-index: 1;
}

.auth-stage__copy {
  max-width: 35rem;
  margin-top: clamp(3.5rem, 11vh, 8.5rem);
}

.auth-stage__kicker {
  margin: 0 0 0.9rem;
  color: #367baa;
  font-size: 0.72rem;
  font-weight: 760;
  letter-spacing: 0.1em;
  line-height: 1.4;
}

.auth-stage__copy h1 {
  margin: 0;
  color: #15243d;
  font-size: clamp(2.5rem, 4.05vw, 4.5rem);
  font-weight: 780;
  letter-spacing: -0.055em;
  line-height: 1.03;
}

.auth-stage__copy h1 span {
  display: block;
  color: #3f79a4;
}

.auth-stage__description {
  max-width: 30rem;
  margin: 1.2rem 0 0;
  color: rgba(34, 74, 104, 0.72);
  font-size: clamp(0.86rem, 1.1vw, 0.98rem);
  line-height: 1.8;
}

.auth-stage__artwork {
  position: absolute;
  inset: 0;
  z-index: 0;
  min-height: 0;
  margin: 0;
  pointer-events: none;
}

.auth-stage__scene,
.auth-stage__art,
.auth-stage__generated-art {
  position: absolute;
  max-width: none;
  pointer-events: none;
}

.auth-stage__generated-art {
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: center 62%;
  opacity: 1;
}

.auth-stage__scene {
  right: -2%;
  bottom: 0;
  width: min(94%, 43rem);
  height: 100%;
  object-fit: contain;
  object-position: right bottom;
  opacity: 0.96;
}

.auth-stage__art {
  right: 17%;
  bottom: 2%;
  width: min(42%, 18rem);
  height: 78%;
  object-fit: contain;
  object-position: center bottom;
  filter: drop-shadow(0 1.4rem 1.7rem rgba(44, 102, 137, 0.14));
}

.auth-stage__visual-note {
  display: flex;
  align-items: center;
  gap: 0.48rem;
  margin: 0.4rem 0 0;
  color: rgba(43, 86, 111, 0.72);
  font-size: 0.68rem;
  font-weight: 650;
  letter-spacing: 0.02em;
}

.auth-stage__status-dot {
  width: 0.42rem;
  height: 0.42rem;
  border-radius: 50%;
  background: #2da981;
  box-shadow: 0 0 0 0.22rem rgba(45, 169, 129, 0.14);
}

.auth-stage__form-column {
  position: relative;
  display: flex;
  min-width: 0;
  min-height: 100vh;
  min-height: 100dvh;
  align-items: center;
  justify-content: center;
  padding: clamp(2rem, 6vh, 5rem) clamp(2rem, 6vw, 7rem);
  background: #ffffff;
}

.auth-stage__locale {
  position: absolute;
  top: clamp(1.25rem, 3vw, 2.5rem);
  right: clamp(1.25rem, 4vw, 4rem);
  z-index: 2;
}

.auth-stage__mobile-brand {
  display: none;
}

.auth-panel {
  width: min(100%, 28rem);
  padding: 0;
}

.auth-panel :deep(.auth-form__heading) {
  text-align: left;
}

.auth-panel :deep(.auth-form__eyebrow) {
  display: inline-flex;
  align-items: center;
  min-height: 1.7rem;
  padding: 0 0.7rem;
  border: 1px solid #dbe8fb;
  border-radius: 999px;
  background: #f4f8ff;
  color: #3977ca;
  font-size: 0.63rem;
  font-weight: 760;
  letter-spacing: 0.1em;
  line-height: 1;
  text-transform: uppercase;
}

.auth-panel :deep(.auth-form__title) {
  margin: 0.95rem 0 0;
  color: var(--auth-ink);
  font-size: clamp(1.8rem, 2.5vw, 2.3rem);
  font-weight: 750;
  letter-spacing: -0.025em;
  line-height: 1.18;
}

.auth-panel :deep(.auth-form__description) {
  max-width: 28rem;
  margin: 0.7rem 0 0;
  color: var(--auth-muted);
  font-size: 0.88rem;
  line-height: 1.7;
}

.auth-panel :deep(.auth-form__fields) {
  display: grid;
  gap: 1rem;
  margin-top: 2rem;
}

.auth-panel :deep(.auth-form--register .auth-form__fields) {
  gap: 0.82rem;
  margin-top: 1.45rem;
}

.auth-panel :deep(.auth-field) {
  min-width: 0;
}

.auth-panel :deep(.auth-field__icon),
.auth-panel :deep(.auth-field__action) {
  color: #91a0b5;
}

.auth-panel :deep(.auth-field__action:hover) {
  color: var(--auth-primary);
}

.auth-panel :deep(.input) {
  min-height: 3.25rem;
  border: 1px solid #dce5f0;
  border-radius: 0.55rem;
  background: #ffffff;
  color: var(--auth-ink);
  font-size: 0.94rem;
  box-shadow: none;
}

.auth-panel :deep(.input.pl-11) {
  padding-left: 3.25rem;
}

.auth-panel :deep(.input::placeholder) {
  color: #a4afbf;
}

.auth-panel :deep(.input:focus) {
  border-color: #70a7ef;
  box-shadow: 0 0 0 3px rgba(54, 128, 229, 0.12);
}

.auth-panel :deep(.auth-form__alternatives) {
  display: grid;
  gap: 0.75rem;
  margin-top: 0.35rem;
}

.auth-panel :deep(.auth-form__rule) {
  background: #e7edf5;
}

.auth-panel :deep(.auth-form__or) {
  color: #95a1b1;
}

.auth-panel :deep(.auth-link) {
  color: #2776d5;
  font-weight: 650;
  text-decoration: none;
}

.auth-panel :deep(.auth-link:hover) {
  color: var(--auth-primary-dark);
}

.auth-panel :deep(.btn-primary) {
  min-height: 3.25rem;
  border: 0;
  border-radius: 0.55rem;
  background: linear-gradient(95deg, #1e80f4 0%, #3d76e8 56%, #8d68e5 100%);
  box-shadow: 0 0.8rem 1.5rem rgba(47, 108, 221, 0.18);
  color: #ffffff;
  font-size: 0.98rem;
  font-weight: 700;
}

.auth-panel :deep(.btn-primary:hover:not(:disabled)) {
  background: linear-gradient(95deg, #146fdf 0%, #2f66d8 56%, #7957d5 100%);
  box-shadow: 0 1rem 1.8rem rgba(47, 108, 221, 0.22);
  transform: translateY(-1px);
}

.auth-panel :deep(.btn-secondary) {
  min-height: 3rem;
  border-color: #dce5f0;
  border-radius: 0.55rem;
  box-shadow: none;
}

.auth-panel__footer {
  margin-top: 1.35rem;
  color: var(--auth-muted);
  font-size: 0.85rem;
  text-align: center;
}

.auth-panel__footer:empty {
  display: none;
}

.auth-panel__legal {
  margin-top: 1.7rem;
  color: #a2acb9;
  font-size: 0.7rem;
  line-height: 1.5;
  text-align: center;
}

:global(.dark) .auth-stage {
  --auth-ink: #edf3fc;
  --auth-muted: #9aa9bd;
  background: #111a2a;
}

:global(.dark) .auth-stage__visual {
  background:
    radial-gradient(circle at 76% 72%, rgba(80, 87, 168, 0.27), transparent 34%),
    linear-gradient(135deg, #111d30 0%, #14243b 52%, #211c3b 100%);
}

:global(.dark) .auth-stage__brand-copy strong,
:global(.dark) .auth-stage__mobile-brand strong,
:global(.dark) .auth-stage__copy h1 {
  color: #f2f6fd;
}

:global(.dark) .auth-stage__brand-copy small,
:global(.dark) .auth-stage__mobile-brand small,
:global(.dark) .auth-stage__description,
:global(.dark) .auth-stage__visual-note {
  color: #9cacbf;
}

:global(.dark) .auth-stage__form-column {
  background: #121c2d;
}

:global(.dark) .auth-panel :deep(.auth-form__title) {
  color: #f2f6fd;
}

:global(.dark) .auth-panel :deep(.auth-form__eyebrow) {
  border-color: rgba(86, 143, 225, 0.3);
  background: rgba(45, 102, 183, 0.2);
  color: #9fc9ff;
}

:global(.dark) .auth-panel :deep(.input) {
  border-color: #34455e;
  background: #18263a;
  color: #f2f6fd;
}

:global(.dark) .auth-panel :deep(.auth-form__rule) {
  background: #34455e;
}

@media (max-width: 1180px) {
  .auth-stage {
    grid-template-columns: minmax(0, 1fr) minmax(29rem, 0.9fr);
  }

  .auth-stage__visual {
    padding-inline: clamp(1.5rem, 4vw, 3rem);
  }

  .auth-stage__copy h1 {
    font-size: clamp(2.35rem, 4vw, 3.5rem);
  }
}

@media (max-width: 900px) {
  .auth-stage {
    display: block;
    min-height: 100vh;
    min-height: 100dvh;
  }

  .auth-stage__visual {
    display: none;
  }

  .auth-stage__form-column {
    min-height: 100vh;
    min-height: 100dvh;
    align-items: flex-start;
    justify-content: flex-start;
    padding: clamp(1.25rem, 5vh, 3rem) clamp(1.15rem, 6vw, 3.5rem) 2rem;
  }

  .auth-stage__locale {
    top: 1.25rem;
    right: 1rem;
  }

  .auth-stage__mobile-brand {
    display: inline-flex;
    width: fit-content;
    align-items: center;
    gap: 0.7rem;
    margin: 0 auto clamp(2rem, 5vh, 3rem);
    color: #17233a;
    text-decoration: none;
  }

  .auth-stage__mobile-brand img {
    width: 2.7rem;
    height: 2.7rem;
    object-fit: contain;
  }

  .auth-panel {
    width: min(100%, 30rem);
    margin: 0 auto;
  }
}

@media (max-width: 520px) {
  .auth-stage__form-column {
    padding: 1.25rem 1rem 1.65rem;
  }

  .auth-stage__mobile-brand {
    margin-bottom: 1.75rem;
  }

  .auth-panel :deep(.auth-form__title) {
    font-size: 1.75rem;
  }

  .auth-panel :deep(.auth-form__description) {
    font-size: 0.82rem;
  }

  .auth-panel :deep(.auth-field .relative),
  .auth-panel :deep(.auth-field .input) {
    width: 100%;
    max-width: 100%;
    box-sizing: border-box;
  }
}

@media (prefers-reduced-motion: reduce) {
  .auth-panel :deep(*) {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
</style>
