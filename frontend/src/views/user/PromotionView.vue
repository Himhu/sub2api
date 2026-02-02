<template>
  <AppLayout>
    <div class="mx-auto max-w-2xl space-y-6">
      <!-- Invite Code Card -->
      <div class="card overflow-hidden">
        <div class="bg-gradient-to-br from-primary-500 to-primary-600 px-6 py-8 text-center">
          <div
            class="mb-4 inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-white/20 backdrop-blur-sm"
          >
            <svg class="h-8 w-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M7.217 10.907a2.25 2.25 0 100 2.186m0-2.186c.18.324.283.696.283 1.093s-.103.77-.283 1.093m0-2.186l9.566-5.314m-9.566 7.5l9.566 5.314m0 0a2.25 2.25 0 103.935 2.186 2.25 2.25 0 00-3.935-2.186zm0-12.814a2.25 2.25 0 103.933-2.185 2.25 2.25 0 00-3.933 2.185z" />
            </svg>
          </div>
          <p class="text-sm font-medium text-primary-100">{{ t('promotion.inviteCode') }}</p>
          <p class="mt-2 text-2xl font-bold text-white font-mono tracking-wider">
            {{ user?.invite_code || t('promotion.noInviteCode') }}
          </p>
        </div>
      </div>

      <!-- Invite Stats Card -->
      <div class="card p-6">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="flex h-12 w-12 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900/40">
              <svg class="h-6 w-6 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M15 19.128a9.38 9.38 0 002.625.372 9.337 9.337 0 004.121-.952 4.125 4.125 0 00-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 018.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0111.964-3.07M12 6.375a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zm8.25 2.25a2.625 2.625 0 11-5.25 0 2.625 2.625 0 015.25 0z" />
              </svg>
            </div>
            <div>
              <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('promotion.totalInvited') }}</p>
              <p class="text-2xl font-bold text-gray-900 dark:text-white">{{ inviteCount }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Copy Buttons -->
      <div class="card">
        <div class="p-6 space-y-4">
          <!-- Copy Invite Code -->
          <div>
            <label class="input-label">{{ t('promotion.inviteCode') }}</label>
            <div class="mt-1 flex gap-2">
              <input
                type="text"
                readonly
                :value="user?.invite_code || ''"
                class="input flex-1 font-mono"
                :placeholder="t('promotion.noInviteCode')"
              />
              <button
                type="button"
                class="btn btn-primary"
                :disabled="!user?.invite_code"
                @click="copyInviteCode"
              >
                <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184" />
                </svg>
                <span class="ml-2">{{ t('promotion.copyCode') }}</span>
              </button>
            </div>
          </div>

          <!-- Copy Invite Link -->
          <div>
            <label class="input-label">{{ t('promotion.inviteLink') }}</label>
            <div class="mt-1 flex gap-2">
              <input
                type="text"
                readonly
                :value="inviteLink"
                class="input flex-1 text-sm"
                :placeholder="t('promotion.noInviteCode')"
              />
              <button
                type="button"
                class="btn btn-primary"
                :disabled="!user?.invite_code"
                @click="copyInviteLink"
              >
                <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M13.19 8.688a4.5 4.5 0 011.242 7.244l-4.5 4.5a4.5 4.5 0 01-6.364-6.364l1.757-1.757m13.35-.622l1.757-1.757a4.5 4.5 0 00-6.364-6.364l-4.5 4.5a4.5 4.5 0 001.242 7.244" />
                </svg>
                <span class="ml-2">{{ t('promotion.copyLink') }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Reward Rules -->
      <div class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-medium text-gray-900 dark:text-white">
            {{ t('promotion.rewardRules') }}
          </h2>
        </div>
        <div class="p-6">
          <div class="space-y-4">
            <!-- Inviter Reward -->
            <div class="flex items-center justify-between rounded-lg bg-green-50 p-4 dark:bg-green-900/20">
              <div class="flex items-center gap-3">
                <div class="flex h-10 w-10 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/40">
                  <svg class="h-5 w-5 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v12m-3-2.818l.879.659c1.171.879 3.07.879 4.242 0 1.172-.879 1.172-2.303 0-3.182C13.536 12.219 12.768 12 12 12c-.725 0-1.45-.22-2.003-.659-1.106-.879-1.106-2.303 0-3.182s2.9-.879 4.006 0l.415.33M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                </div>
                <div>
                  <p class="font-medium text-green-800 dark:text-green-300">{{ t('promotion.inviterReward') }}</p>
                  <p class="text-sm text-green-600 dark:text-green-400">{{ isAgent ? t('promotion.agentNoInviterReward') : t('promotion.perInvite') }}</p>
                </div>
              </div>
              <div class="text-right">
                <p class="text-2xl font-bold text-green-600 dark:text-green-400">
                  +{{ isAgent ? 0 : inviterBonus }}
                </p>
                <p class="text-sm text-green-600 dark:text-green-400">{{ t('promotion.credits') }}</p>
              </div>
            </div>

            <!-- Invitee Reward -->
            <div class="flex items-center justify-between rounded-lg bg-blue-50 p-4 dark:bg-blue-900/20">
              <div class="flex items-center gap-3">
                <div class="flex h-10 w-10 items-center justify-center rounded-full bg-blue-100 dark:bg-blue-900/40">
                  <svg class="h-5 w-5 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M21 11.25v8.25a1.5 1.5 0 01-1.5 1.5H5.25a1.5 1.5 0 01-1.5-1.5v-8.25M12 4.875A2.625 2.625 0 109.375 7.5H12m0-2.625V7.5m0-2.625A2.625 2.625 0 1114.625 7.5H12m0 0V21m-8.625-9.75h18c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125h-18c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" />
                  </svg>
                </div>
                <div>
                  <p class="font-medium text-blue-800 dark:text-blue-300">{{ t('promotion.inviteeReward') }}</p>
                  <p class="text-sm text-blue-600 dark:text-blue-400">{{ t('promotion.forNewUser') }}</p>
                </div>
              </div>
              <div class="text-right">
                <p class="text-2xl font-bold text-blue-600 dark:text-blue-400">
                  +{{ inviteeBonus }}
                </p>
                <p class="text-sm text-blue-600 dark:text-blue-400">{{ t('promotion.credits') }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>

    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { userAPI } from '@/api/user'
import AppLayout from '@/components/layout/AppLayout.vue'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const user = computed(() => authStore.user)
const isAgent = computed(() => user.value?.is_agent ?? false)
const inviteCount = ref(0)

// Load invite count
const loadInviteCount = async () => {
  try {
    const result = await userAPI.getInviteCount()
    inviteCount.value = result.invite_count
  } catch (error) {
    console.error('Failed to load invite count:', error)
  }
}

onMounted(() => {
  loadInviteCount()
})

// Get invite bonus from public settings
const inviterBonus = computed(() => {
  return appStore.cachedPublicSettings?.inviter_bonus ?? 0
})

const inviteeBonus = computed(() => {
  return appStore.cachedPublicSettings?.invitee_bonus ?? 0
})

// Generate invite link
const inviteLink = computed(() => {
  if (!user.value?.invite_code) return ''
  const baseUrl = window.location.origin
  return `${baseUrl}/register?promo_code=${user.value.invite_code}`
})

// Copy invite code to clipboard
const copyInviteCode = async () => {
  if (!user.value?.invite_code) return
  try {
    await navigator.clipboard.writeText(user.value.invite_code)
    appStore.showSuccess(t('common.copiedToClipboard'))
  } catch {
    appStore.showError(t('common.copyFailed'))
  }
}

// Copy invite link to clipboard
const copyInviteLink = async () => {
  if (!inviteLink.value) return
  try {
    await navigator.clipboard.writeText(inviteLink.value)
    appStore.showSuccess(t('common.copiedToClipboard'))
  } catch {
    appStore.showError(t('common.copyFailed'))
  }
}
</script>
