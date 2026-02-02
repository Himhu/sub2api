<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-6">
      <!-- 非代理用户：显示如何成为代理 -->
      <template v-if="!isAgent">
        <div class="card overflow-hidden">
          <div class="bg-gradient-to-br from-primary-500 to-primary-600 px-6 py-8 text-center">
            <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-white/20">
              <svg class="h-8 w-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z" />
              </svg>
            </div>
            <h2 class="text-2xl font-bold text-white">{{ t('agentCenter.becomeAgent') }}</h2>
            <p class="mt-2 text-primary-100">{{ t('agentCenter.becomeAgentDesc') }}</p>
          </div>
          <div class="p-6">
            <div class="space-y-4">
              <div class="flex items-start gap-3">
                <div class="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-primary-100 text-primary-600 dark:bg-primary-900 dark:text-primary-400">
                  <span class="text-sm font-medium">1</span>
                </div>
                <div>
                  <h3 class="font-medium text-gray-900 dark:text-white">{{ t('agentCenter.step1Title') }}</h3>
                  <p class="text-sm text-gray-500">{{ t('agentCenter.step1Desc') }}</p>
                </div>
              </div>
              <div class="flex items-start gap-3">
                <div class="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-primary-100 text-primary-600 dark:bg-primary-900 dark:text-primary-400">
                  <span class="text-sm font-medium">2</span>
                </div>
                <div>
                  <h3 class="font-medium text-gray-900 dark:text-white">{{ t('agentCenter.step2Title') }}</h3>
                  <p class="text-sm text-gray-500">{{ t('agentCenter.step2Desc') }}</p>
                </div>
              </div>
              <div class="flex items-start gap-3">
                <div class="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-primary-100 text-primary-600 dark:bg-primary-900 dark:text-primary-400">
                  <span class="text-sm font-medium">3</span>
                </div>
                <div>
                  <h3 class="font-medium text-gray-900 dark:text-white">{{ t('agentCenter.step3Title') }}</h3>
                  <p class="text-sm text-gray-500">{{ t('agentCenter.step3Desc') }}</p>
                </div>
              </div>
            </div>
            <div class="mt-6 rounded-lg bg-gray-50 p-4 dark:bg-dark-700">
              <p class="text-sm text-gray-600 dark:text-gray-400">
                {{ t('agentCenter.contactAdmin') }}
              </p>
            </div>
          </div>
        </div>
      </template>

      <!-- 代理用户：显示统计数据和下线用户 -->
      <template v-else>
        <!-- Stats Cards -->
        <div class="grid grid-cols-2 gap-4 md:grid-cols-4">
          <div class="card p-4">
            <div class="text-2xl font-bold text-gray-900 dark:text-white">
              {{ stats?.total_invited ?? 0 }}
            </div>
            <div class="text-sm text-gray-500">{{ t('agentCenter.totalInvited') }}</div>
          </div>
          <div class="card p-4">
            <div class="text-2xl font-bold text-gray-900 dark:text-white">
              {{ stats?.direct_invited ?? 0 }}
            </div>
            <div class="text-sm text-gray-500">{{ t('agentCenter.directInvited') }}</div>
          </div>
          <div class="card p-4">
            <div class="text-2xl font-bold text-green-600">
              {{ stats?.invited_agents ?? 0 }}
            </div>
            <div class="text-sm text-gray-500">{{ t('agentCenter.invitedAgents') }}</div>
          </div>
          <div class="card p-4">
            <div class="text-2xl font-bold text-blue-600">
              {{ stats?.invited_users ?? 0 }}
            </div>
            <div class="text-sm text-gray-500">{{ t('agentCenter.invitedUsers') }}</div>
          </div>
        </div>

        <!-- Invite Code Card -->
        <div class="card overflow-hidden">
          <div class="bg-gradient-to-br from-primary-500 to-primary-600 px-6 py-6 text-center">
            <p class="text-sm font-medium text-primary-100">{{ t('promotion.inviteCode') }}</p>
            <p class="mt-2 text-2xl font-bold text-white font-mono tracking-wider">
              {{ user?.invite_code || t('promotion.noInviteCode') }}
            </p>
          </div>
          <div class="p-4 flex gap-2">
            <button
              type="button"
              class="btn btn-secondary flex-1"
              :disabled="!user?.invite_code"
              @click="copyInviteCode"
            >
              {{ t('promotion.copyCode') }}
            </button>
            <button
              type="button"
              class="btn btn-primary flex-1"
              :disabled="!user?.invite_code"
              @click="copyInviteLink"
            >
              {{ t('promotion.copyLink') }}
            </button>
          </div>
        </div>

        <!-- Downline Users Table -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-medium text-gray-900 dark:text-white">
              {{ t('agentCenter.downlineUsers') }}
            </h2>
          </div>
          <div class="p-6">
            <div v-if="loading" class="flex justify-center py-8">
              <svg class="h-8 w-8 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
            </div>
            <div v-else-if="downlineUsers.length === 0" class="py-8 text-center text-gray-500">
              {{ t('agentCenter.noDownline') }}
            </div>
            <div v-else>
              <table class="w-full">
                <thead>
                  <tr class="border-b dark:border-dark-600">
                    <th class="px-4 py-2 text-left text-sm font-medium text-gray-500">
                      {{ t('agentCenter.columns.email') }}
                    </th>
                    <th class="px-4 py-2 text-left text-sm font-medium text-gray-500">
                      {{ t('agentCenter.columns.status') }}
                    </th>
                    <th class="px-4 py-2 text-left text-sm font-medium text-gray-500">
                      {{ t('agentCenter.columns.createdAt') }}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="u in downlineUsers" :key="u.id" class="border-b dark:border-dark-700">
                    <td class="px-4 py-3">
                      <div class="flex flex-col">
                        <span class="font-medium">{{ u.email }}</span>
                        <span class="text-xs text-gray-500">{{ u.username }}</span>
                      </div>
                    </td>
                    <td class="px-4 py-3">
                      <span :class="['badge', u.is_agent ? 'badge-success' : 'badge-secondary']">
                        {{ u.is_agent ? t('agentCenter.statusAgent') : t('agentCenter.statusUser') }}
                      </span>
                    </td>
                    <td class="px-4 py-3 text-sm text-gray-500">
                      {{ formatDateTime(u.created_at) }}
                    </td>
                  </tr>
                </tbody>
              </table>
              <Pagination
                v-if="pagination && pagination.total > 0"
                :page="pagination.page"
                :total="pagination.total"
                :page-size="pagination.page_size"
                @update:page="handlePageChange"
                class="mt-4"
              />
            </div>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { agentAPI, type InviteStats } from '@/api/agent'
import type { User } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const user = computed(() => authStore.user)
const isAgent = computed(() => authStore.user?.is_agent ?? false)
const loading = ref(false)
const stats = ref<InviteStats | null>(null)
const downlineUsers = ref<User[]>([])
const pagination = ref<{
  total: number
  page: number
  page_size: number
  pages: number
} | null>(null)

// Generate invite link
const inviteLink = computed(() => {
  if (!user.value?.invite_code) return ''
  const baseUrl = window.location.origin
  return `${baseUrl}/register?promo_code=${user.value.invite_code}`
})

// Load stats (only for agents)
const loadStats = async () => {
  if (!isAgent.value) return
  try {
    stats.value = await agentAPI.getMyInviteStats()
  } catch (error) {
    console.error('Failed to load stats:', error)
  }
}

// Load downline users (only for agents)
const loadDownline = async (page = 1) => {
  if (!isAgent.value) return
  loading.value = true
  try {
    const response = await agentAPI.getMyDownline(page, 10)
    downlineUsers.value = response.items
    pagination.value = {
      total: response.total,
      page: response.page,
      page_size: response.page_size,
      pages: response.pages
    }
  } catch (error) {
    console.error('Failed to load downline:', error)
  } finally {
    loading.value = false
  }
}

// Handle page change
const handlePageChange = (page: number) => {
  loadDownline(page)
}

// Format date time
const formatDateTime = (dateStr: string) => {
  return new Date(dateStr).toLocaleString()
}

// Copy invite code
const copyInviteCode = async () => {
  if (!user.value?.invite_code) return
  try {
    await navigator.clipboard.writeText(user.value.invite_code)
    appStore.showSuccess(t('common.copiedToClipboard'))
  } catch {
    appStore.showError(t('common.copyFailed'))
  }
}

// Copy invite link
const copyInviteLink = async () => {
  if (!inviteLink.value) return
  try {
    await navigator.clipboard.writeText(inviteLink.value)
    appStore.showSuccess(t('common.copiedToClipboard'))
  } catch {
    appStore.showError(t('common.copyFailed'))
  }
}

onMounted(() => {
  if (isAgent.value) {
    loadStats()
    loadDownline()
  }
})
</script>
