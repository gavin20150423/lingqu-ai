<template>
  <div class="min-h-screen bg-[#f5f7fb] text-gray-900">
    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      class="relative min-h-screen transition-all duration-300"
      :class="[sidebarCollapsed ? 'lg:ml-[72px]' : 'lg:ml-64']"
    >
      <!-- Header -->
      <AppHeader />

      <!-- Main Content -->
      <main class="p-4 md:p-6 lg:p-8">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'

const appStore = useAppStore()
const authStore = useAuthStore()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: false
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

defineExpose({ replayTour })
</script>

<style scoped>
:deep(.card) {
  background: rgba(255, 255, 255, 0.96);
  border-color: rgb(226 232 240);
  border-radius: 1rem;
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.06);
}

:deep(.card-hover:hover) {
  transform: translateY(-1px);
  border-color: rgb(203 213 225);
  box-shadow: 0 14px 34px rgba(15, 23, 42, 0.08);
}

:deep(.glass),
:deep(.glass-card),
:deep(.dropdown),
:deep(.modal-content),
:deep(.dialog-container),
:deep(.select-dropdown-portal),
:deep(.date-picker-dropdown) {
  border-color: rgb(226 232 240);
  box-shadow: 0 14px 36px rgba(15, 23, 42, 0.08);
}

:deep(.btn-primary) {
  box-shadow: 0 8px 18px rgba(240, 68, 56, 0.16);
}

:deep(.btn-secondary) {
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.05);
}

:deep(.table-container) {
  border-color: rgb(226 232 240);
}
</style>
