<template>
  <Teleport to="body">
    <Transition name="popup-fade">
      <div
        v-if="announcementStore.currentPopup"
        class="announcement-popup"
        :data-user-theme="theme"
      >
        <div
          class="announcement-popup__dialog"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="`announcement-popup-title-${announcementStore.currentPopup.id}`"
          @click.stop
        >
          <header class="announcement-popup__header">
            <div class="announcement-popup__heading">
              <span class="announcement-popup__icon" aria-hidden="true">
                <Icon name="bell" size="md" />
              </span>
              <div class="announcement-popup__heading-copy">
                <span
                  class="announcement-popup__status"
                  :class="{ 'announcement-popup__status--read': !isUnread }"
                >
                  <i v-if="isUnread" aria-hidden="true"></i>
                  {{ isUnread ? t('announcements.unread') : t('announcements.read') }}
                </span>
                <h2 :id="`announcement-popup-title-${announcementStore.currentPopup.id}`">
                  {{ announcementStore.currentPopup.title }}
                </h2>
              </div>
            </div>
            <div class="announcement-popup__time">
              <Icon name="clock" size="sm" />
              <time>{{ formatRelativeWithDateTime(announcementStore.currentPopup.created_at) }}</time>
            </div>
          </header>

          <main class="announcement-popup__body">
            <div class="announcement-popup__content">
              <div
                class="markdown-body prose prose-sm max-w-none dark:prose-invert"
                v-html="renderedContent"
              ></div>
            </div>
          </main>

          <footer class="announcement-popup__footer">
            <span class="announcement-popup__footer-note">
              <Icon name="bell" size="sm" />
              {{ isUnread ? '关闭后将标记为已读' : '可随时从顶部公告栏再次查看' }}
            </span>
            <button type="button" class="announcement-popup__action" @click="handleDismiss">
              <Icon :name="isUnread ? 'check' : 'x'" size="sm" />
              {{ isUnread ? t('announcements.markRead') : t('common.close') }}
            </button>
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import Icon from '@/components/icons/Icon.vue'
import { useAnnouncementStore } from '@/stores/announcements'
import { useUserThemeStore } from '@/stores/userTheme'
import { formatRelativeWithDateTime } from '@/utils/format'

const { t } = useI18n()
const announcementStore = useAnnouncementStore()
const userThemeStore = useUserThemeStore()
const { theme } = storeToRefs(userThemeStore)

marked.setOptions({
  breaks: true,
  gfm: true,
})

const renderedContent = computed(() => {
  const content = announcementStore.currentPopup?.content
  if (!content) return ''
  const html = marked.parse(content) as string
  return DOMPurify.sanitize(html)
})

const isUnread = computed(() => !announcementStore.currentPopup?.read_at)

function handleDismiss() {
  announcementStore.dismissPopup()
}

// Manage body overflow — only set, never unset (bell component handles restore)
watch(
  () => announcementStore.currentPopup,
  (popup) => {
    if (popup) {
      document.body.style.overflow = 'hidden'
    }
  }
)
</script>

<style scoped>
.announcement-popup {
  --popup-accent: #e9849b;
  --popup-accent-strong: #c85c78;
  --popup-accent-soft: #fff0f4;
  --popup-ink: #272421;
  --popup-muted: #746d66;
  --popup-line: rgba(39, 36, 33, 0.13);
  --popup-surface: #fffefb;
  --popup-subtle: #faf8f3;
  position: fixed;
  inset: 0;
  z-index: 120;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow-y: auto;
  background: rgba(29, 31, 34, 0.58);
  padding: 1.25rem;
  backdrop-filter: blur(8px);
}

.announcement-popup[data-user-theme='business'] {
  --popup-accent: #2456a6;
  --popup-accent-strong: #173f7b;
  --popup-accent-soft: #eef4fc;
  --popup-ink: #202a33;
  --popup-muted: #66727d;
  --popup-line: #d8e0e6;
  --popup-surface: #ffffff;
  --popup-subtle: #f7f9fb;
  background: rgba(28, 36, 43, 0.48);
  backdrop-filter: blur(6px);
}

.announcement-popup__dialog {
  width: min(100%, 40rem);
  overflow: hidden;
  border: 1px solid var(--popup-line);
  border-radius: 18px;
  background: var(--popup-surface);
  box-shadow: 0 28px 70px rgba(24, 28, 31, 0.24);
  color: var(--popup-ink);
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.announcement-popup[data-user-theme='business'] .announcement-popup__dialog {
  border-radius: 8px;
  box-shadow: 0 24px 60px rgba(24, 34, 42, 0.22);
}

.announcement-popup__header {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: end;
  gap: 1rem;
  border-bottom: 1px solid var(--popup-line);
  background: var(--popup-subtle);
  padding: 1.3rem 1.4rem;
}

.announcement-popup__heading {
  min-width: 0;
  display: grid;
  grid-template-columns: 2.65rem minmax(0, 1fr);
  align-items: start;
  gap: 0.85rem;
}

.announcement-popup__icon {
  width: 2.65rem;
  height: 2.65rem;
  display: grid;
  place-items: center;
  border: 1px solid color-mix(in srgb, var(--popup-accent) 28%, transparent);
  border-radius: 10px;
  background: var(--popup-accent-soft);
  color: var(--popup-accent);
}

.announcement-popup[data-user-theme='business'] .announcement-popup__icon {
  border-radius: 6px;
}

.announcement-popup__heading-copy {
  min-width: 0;
}

.announcement-popup__status {
  display: inline-flex;
  min-height: 1.45rem;
  align-items: center;
  gap: 0.38rem;
  border-radius: 999px;
  background: var(--popup-accent-soft);
  color: var(--popup-accent-strong);
  padding: 0 0.52rem;
  font-size: 0.68rem;
  font-weight: 700;
}

.announcement-popup[data-user-theme='business'] .announcement-popup__status {
  border-radius: 4px;
}

.announcement-popup__status--read {
  background: color-mix(in srgb, var(--popup-muted) 10%, transparent);
  color: var(--popup-muted);
}

.announcement-popup__status i {
  width: 0.4rem;
  height: 0.4rem;
  border-radius: 999px;
  background: var(--popup-accent);
}

.announcement-popup__header h2 {
  margin-top: 0.48rem;
  overflow-wrap: anywhere;
  color: var(--popup-ink);
  font-size: 1.24rem;
  font-weight: 750;
  line-height: 1.3;
}

.announcement-popup[data-user-theme='business'] .announcement-popup__header h2 {
  font-family: 'Avenir Next', 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', sans-serif;
  font-size: 1.12rem;
  font-weight: 700;
}

.announcement-popup__time {
  display: inline-flex;
  align-items: center;
  gap: 0.38rem;
  color: var(--popup-muted);
  font-size: 0.74rem;
  white-space: nowrap;
}

.announcement-popup__body {
  max-height: min(50vh, 25rem);
  overflow-y: auto;
  background: var(--popup-surface);
  padding: 1.35rem 1.4rem;
}

.announcement-popup__content {
  min-height: 5rem;
  border-left: 3px solid var(--popup-accent);
  padding: 0.15rem 0 0.15rem 1rem;
  color: var(--popup-ink);
  line-height: 1.7;
}

.announcement-popup__content :deep(h1),
.announcement-popup__content :deep(h2),
.announcement-popup__content :deep(h3),
.announcement-popup__content :deep(h4) {
  margin: 0 0 0.65rem;
  color: var(--popup-ink);
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.45;
}

.announcement-popup__content :deep(p) {
  margin: 0 0 0.75rem;
}

.announcement-popup__content :deep(p:last-child),
.announcement-popup__content :deep(ul:last-child),
.announcement-popup__content :deep(ol:last-child) {
  margin-bottom: 0;
}

.announcement-popup__content :deep(ul),
.announcement-popup__content :deep(ol) {
  margin: 0.25rem 0 0.75rem;
  padding-left: 1.2rem;
}

.announcement-popup__content :deep(ul) {
  list-style: disc;
}

.announcement-popup__content :deep(ol) {
  list-style: decimal;
}

.announcement-popup__content :deep(li) {
  margin: 0.2rem 0;
  padding-left: 0.15rem;
}

.announcement-popup__content :deep(a) {
  color: var(--popup-accent);
  font-weight: 650;
  text-decoration: underline;
  text-underline-offset: 2px;
}

.announcement-popup__content :deep(strong) {
  color: var(--popup-ink);
  font-weight: 750;
}

.announcement-popup[data-user-theme='business'] .announcement-popup__content {
  border-left-width: 2px;
}

.announcement-popup__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-top: 1px solid var(--popup-line);
  background: var(--popup-subtle);
  padding: 0.85rem 1.4rem;
}

.announcement-popup__footer-note {
  display: inline-flex;
  align-items: center;
  gap: 0.38rem;
  color: var(--popup-muted);
  font-size: 0.72rem;
}

.announcement-popup__action {
  min-height: 2.35rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.42rem;
  border-radius: 9px;
  background: var(--popup-accent);
  color: #ffffff;
  padding: 0 0.9rem;
  font-size: 0.78rem;
  font-weight: 700;
  box-shadow: 0 8px 18px color-mix(in srgb, var(--popup-accent) 24%, transparent);
  transition: background 150ms ease, box-shadow 150ms ease, transform 150ms ease;
}

.announcement-popup[data-user-theme='business'] .announcement-popup__action {
  border-radius: 6px;
  box-shadow: none;
}

.announcement-popup__action:hover {
  background: var(--popup-accent-strong);
  box-shadow: 0 10px 22px color-mix(in srgb, var(--popup-accent) 28%, transparent);
  transform: translateY(-1px);
}

.announcement-popup__action:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--popup-accent) 24%, transparent);
  outline-offset: 2px;
}

.announcement-popup[data-user-theme='business'] .announcement-popup__action:hover {
  box-shadow: 0 4px 12px rgba(36, 86, 166, 0.2);
}

.popup-fade-enter-active {
  transition: opacity 0.22s ease;
}

.popup-fade-leave-active {
  transition: opacity 0.16s ease;
}

.popup-fade-enter-from,
.popup-fade-leave-to {
  opacity: 0;
}

.popup-fade-enter-from > div {
  transform: translateY(-8px);
  opacity: 0;
}

.popup-fade-leave-to > div {
  transform: translateY(-5px);
  opacity: 0;
}

.announcement-popup__body::-webkit-scrollbar {
  width: 6px;
}

.announcement-popup__body::-webkit-scrollbar-track {
  background: transparent;
}

.announcement-popup__body::-webkit-scrollbar-thumb {
  border-radius: 3px;
  background: color-mix(in srgb, var(--popup-muted) 45%, transparent);
}

@media (max-width: 640px) {
  .announcement-popup {
    align-items: flex-start;
    padding: 1rem;
  }

  .announcement-popup__header {
    grid-template-columns: 1fr;
    align-items: start;
    gap: 0.7rem;
    padding: 1rem;
  }

  .announcement-popup__time {
    padding-left: 3.5rem;
    white-space: normal;
  }

  .announcement-popup__body {
    max-height: 55vh;
    padding: 1rem;
  }

  .announcement-popup__footer {
    align-items: stretch;
    flex-direction: column;
    padding: 0.85rem 1rem 1rem;
  }

  .announcement-popup__action {
    width: 100%;
  }
}
</style>
