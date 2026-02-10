<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-6">
      <div class="grid grid-cols-1 gap-6 sm:grid-cols-3">
        <div class="stat-card">
          <div class="stat-icon stat-icon-success">
            <WalletIcon class="h-6 w-6" aria-hidden="true" />
          </div>
          <div class="min-w-0 flex-1">
            <p class="stat-label truncate">{{ t('profile.accountBalance') }}</p>
            <div class="mt-2 grid gap-2">
              <div class="flex items-baseline justify-between gap-3">
                <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('profile.balance') }}</span>
                <span class="stat-value">{{ formatCurrency(user?.balance || 0) }}</span>
              </div>
              <div class="h-px bg-gray-200/70 dark:bg-white/10"></div>
              <div class="flex items-baseline justify-between gap-3">
                <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('profile.accountPoints') }}</span>
                <span class="text-lg font-semibold text-gray-700 dark:text-gray-200">{{ formatCurrency(user?.points || 0) }}</span>
              </div>
            </div>
          </div>
        </div>
        <StatCard :title="t('profile.concurrencyLimit')" :value="user?.concurrency || 0" :icon="BoltIcon" icon-variant="warning" />
        <StatCard :title="t('profile.memberSince')" :value="formatDate(user?.created_at || '', { year: 'numeric', month: 'long' })" :icon="CalendarIcon" icon-variant="primary" />
      </div>
      <ProfileInfoCard :user="user" />
      <ProfileEditForm :initial-username="user?.username || ''" />
      <ProfilePasswordForm />
      <ProfileTotpCard />
      <ProfileWeChatBindCard />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, h } from 'vue'; import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'; import { formatDate } from '@/utils/format'
import AppLayout from '@/components/layout/AppLayout.vue'
import StatCard from '@/components/common/StatCard.vue'
import ProfileInfoCard from '@/components/user/profile/ProfileInfoCard.vue'
import ProfileEditForm from '@/components/user/profile/ProfileEditForm.vue'
import ProfilePasswordForm from '@/components/user/profile/ProfilePasswordForm.vue'
import ProfileTotpCard from '@/components/user/profile/ProfileTotpCard.vue'
import ProfileWeChatBindCard from '@/components/user/profile/ProfileWeChatBindCard.vue'

const { t } = useI18n(); const authStore = useAuthStore(); const user = computed(() => authStore.user)

const WalletIcon = { render: () => h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [h('path', { d: 'M21 12a2.25 2.25 0 00-2.25-2.25H15a3 3 0 11-6 0H5.25A2.25 2.25 0 003 12' })]) }
const BoltIcon = { render: () => h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [h('path', { d: 'm3.75 13.5 10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z' })]) }
const CalendarIcon = { render: () => h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [h('path', { d: 'M6.75 3v2.25M17.25 3v2.25' })]) }

const formatCurrency = (v: number) => `$${v.toFixed(2)}`
</script>