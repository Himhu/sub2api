<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-medium text-gray-900 dark:text-white">
        {{ t('profile.wechat.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('profile.wechat.description') }}
      </p>
    </div>
    <div class="px-6 py-6">
      <!-- Loading -->
      <div v-if="loading" class="flex items-center justify-center py-8">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500"></div>
      </div>

      <!-- Bound -->
      <div v-else-if="status?.bound" class="flex items-center justify-between">
        <div class="flex items-center gap-4">
          <div class="flex-shrink-0 rounded-full bg-[#07C160]/10 p-3">
            <svg class="h-6 w-6 text-[#07C160]" viewBox="0 0 1024 1024" fill="currentColor">
              <path d="M690.1 377.4c5.9 0 11.8.2 17.6.5-24.4-128.7-158.3-227.1-313.4-227.1C209 150.8 57.7 272.6 57.7 423.6c0 87.3 47.6 159.2 127.3 214.6l-31.8 95.8 111.1-55.5c39.8 8 71.9 16 111.9 16 5.6 0 11.1-.2 16.6-.5-3.5-11.9-5.5-24.3-5.5-37.1 0-156.6 135.3-279.5 302.8-279.5zM485.2 310.5c24 0 39.7 15.8 39.7 39.7 0 24-15.7 39.8-39.7 39.8-23.9 0-47.7-15.8-47.7-39.8 0-23.9 23.8-39.7 47.7-39.7zM281.3 390c-23.9 0-47.8-15.8-47.8-39.8 0-23.9 23.9-39.7 47.8-39.7 23.9 0 39.7 15.8 39.7 39.7 0 24-15.8 39.8-39.7 39.8z"/>
              <path d="M967.1 656.9c0-127.3-127.3-231.5-270.5-231.5-151.2 0-270.6 104.2-270.6 231.5S545.4 888.4 696.6 888.4c31.8 0 63.7-8 95.5-16l87.4 47.7-23.9-79.5c63.7-47.8 111.5-111.5 111.5-183.7zM616.8 624.9c-16 0-31.8-15.8-31.8-31.8s15.8-31.8 31.8-31.8 39.7 15.8 39.7 31.8-23.7 31.8-39.7 31.8zm159.3 0c-15.8 0-31.8-15.8-31.8-31.8s16-31.8 31.8-31.8c24 0 39.8 15.8 39.8 31.8s-15.8 31.8-39.8 31.8z"/>
            </svg>
          </div>
          <div>
            <p class="font-medium text-gray-900 dark:text-white">{{ t('profile.wechat.bound') }}</p>
            <p v-if="status.openid_masked" class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('profile.wechat.boundOpenId') }}: {{ status.openid_masked }}
            </p>
          </div>
        </div>
        <button class="btn btn-outline-danger" @click="showUnbindDialog = true">
          {{ t('profile.wechat.unbind') }}
        </button>
      </div>

      <!-- Not bound -->
      <div v-else class="flex items-center justify-between">
        <div class="flex items-center gap-4">
          <div class="flex-shrink-0 rounded-full bg-gray-100 p-3 dark:bg-dark-700">
            <svg class="h-6 w-6 text-gray-400" viewBox="0 0 1024 1024" fill="currentColor">
              <path d="M690.1 377.4c5.9 0 11.8.2 17.6.5-24.4-128.7-158.3-227.1-313.4-227.1C209 150.8 57.7 272.6 57.7 423.6c0 87.3 47.6 159.2 127.3 214.6l-31.8 95.8 111.1-55.5c39.8 8 71.9 16 111.9 16 5.6 0 11.1-.2 16.6-.5-3.5-11.9-5.5-24.3-5.5-37.1 0-156.6 135.3-279.5 302.8-279.5zM485.2 310.5c24 0 39.7 15.8 39.7 39.7 0 24-15.7 39.8-39.7 39.8-23.9 0-47.7-15.8-47.7-39.8 0-23.9 23.8-39.7 47.7-39.7zM281.3 390c-23.9 0-47.8-15.8-47.8-39.8 0-23.9 23.9-39.7 47.8-39.7 23.9 0 39.7 15.8 39.7 39.7 0 24-15.8 39.8-39.7 39.8z"/>
              <path d="M967.1 656.9c0-127.3-127.3-231.5-270.5-231.5-151.2 0-270.6 104.2-270.6 231.5S545.4 888.4 696.6 888.4c31.8 0 63.7-8 95.5-16l87.4 47.7-23.9-79.5c63.7-47.8 111.5-111.5 111.5-183.7zM616.8 624.9c-16 0-31.8-15.8-31.8-31.8s15.8-31.8 31.8-31.8 39.7 15.8 39.7 31.8-23.7 31.8-39.7 31.8zm159.3 0c-15.8 0-31.8-15.8-31.8-31.8s16-31.8 31.8-31.8c24 0 39.8 15.8 39.8 31.8s-15.8 31.8-39.8 31.8z"/>
            </svg>
          </div>
          <div>
            <p class="font-medium text-gray-700 dark:text-gray-300">
              {{ t('profile.wechat.notBound') }}
            </p>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('profile.wechat.notBoundHint') }}
            </p>
          </div>
        </div>
        <button class="btn btn-primary" @click="startBindFlow">
          {{ t('profile.wechat.bind') }}
        </button>
      </div>
    </div>

    <!-- Bind Modal -->
    <BaseDialog :show="showBindModal" :title="t('profile.wechat.bindTitle')" @close="closeBindModal">
      <div class="space-y-6">
        <!-- Step 1: Password verification -->
        <div v-if="bindStep === 'password'">
          <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
            {{ t('profile.wechat.bindPasswordHint') }}
          </p>
          <input
            v-model="bindPassword"
            type="password"
            class="input"
            :placeholder="t('profile.wechat.unbindPasswordLabel')"
            @keyup.enter="submitBindPassword"
          />
          <p v-if="bindError" class="mt-2 text-sm text-red-600">{{ bindError }}</p>
        </div>

        <!-- Step 2: Show short code -->
        <div v-else-if="bindStep === 'code_display'">
          <div class="mb-4 rounded-lg border border-primary-200 bg-primary-50 p-3 dark:border-primary-800/50 dark:bg-primary-900/20">
            <div class="space-y-2">
              <template v-if="wechatAccountName">
                <p class="text-center text-sm text-primary-700 dark:text-primary-300">
                  {{ t('auth.wechat.followAccount') }}
                  <span class="font-bold text-red-600 dark:text-red-400">{{ wechatAccountName }}</span>
                </p>
                <div class="flex flex-wrap items-center justify-center gap-x-1.5 gap-y-1 rounded bg-white/60 px-3 py-1.5 text-xs text-gray-600 dark:bg-gray-800/40 dark:text-gray-400">
                  <span>{{ t('auth.wechat.searchFollow', { account: wechatAccountName }) }}</span>
                </div>
              </template>
              <p v-else class="text-center text-sm text-primary-700 dark:text-primary-300">
                {{ t('auth.wechat.followAccountGeneric') }}
              </p>
            </div>
          </div>
          <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
            {{ t('profile.wechat.bindStep1') }}
          </p>
          <div class="flex justify-center my-4">
            <div class="bg-gray-100 dark:bg-dark-700 rounded-lg px-6 py-4 text-center">
              <p class="text-3xl font-mono font-bold tracking-wider text-primary-600 dark:text-primary-400">
                {{ shortCode }}
              </p>
            </div>
          </div>
          <p class="text-sm text-gray-600 dark:text-gray-400 text-center">
            {{ t('profile.wechat.bindStep2') }}
          </p>
        </div>

        <!-- Step 3: Enter verification code -->
        <div v-else-if="bindStep === 'code_input'">
          <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
            {{ t('profile.wechat.bindEnterCode') }}
          </p>
          <input
            v-model="verificationCode"
            type="text"
            class="input text-center text-lg tracking-widest"
            maxlength="6"
            placeholder="000000"
            @keyup.enter="confirmBind"
          />
          <p v-if="bindError" class="mt-2 text-sm text-red-600">{{ bindError }}</p>
        </div>
      </div>

      <template #footer>
        <div class="flex justify-end space-x-3">
          <button class="btn btn-outline" @click="closeBindModal">{{ t('common.cancel') }}</button>
          <button v-if="bindStep === 'password'" class="btn btn-primary" :disabled="!bindPassword || binding" @click="submitBindPassword">
            <span v-if="binding" class="mr-2 inline-block h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
            {{ t('common.next') }}
          </button>
          <button v-else-if="bindStep === 'code_display'" class="btn btn-primary" @click="bindStep = 'code_input'">
            {{ t('common.next') }}
          </button>
          <button v-else class="btn btn-primary" :disabled="!verificationCode || binding" @click="confirmBind">
            <span v-if="binding" class="mr-2 inline-block h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
            {{ t('profile.wechat.bindConfirm') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Unbind Dialog -->
    <BaseDialog :show="showUnbindDialog" :title="t('profile.wechat.unbindTitle')" width="narrow" @close="showUnbindDialog = false">
      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-400">
          {{ t('profile.wechat.unbindConfirm') }}
        </p>
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {{ t('profile.wechat.unbindPasswordLabel') }}
          </label>
          <input
            v-model="unbindPassword"
            type="password"
            class="input"
            @keyup.enter="handleUnbind"
          />
        </div>
        <p v-if="unbindError" class="text-sm text-red-600">{{ unbindError }}</p>
      </div>

      <template #footer>
        <div class="flex justify-end space-x-3">
          <button class="btn btn-outline" @click="showUnbindDialog = false">{{ t('common.cancel') }}</button>
          <button class="btn btn-danger" :disabled="!unbindPassword || unbinding" @click="handleUnbind">
            <span v-if="unbinding" class="mr-2 inline-block h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
            {{ t('profile.wechat.unbind') }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { wechatAPI } from '@/api/wechat'
import type { WeChatBindStatus } from '@/api/wechat'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useAppStore } from '@/stores'
import { getPublicSettings } from '@/api/auth'

const { t } = useI18n()
const appStore = useAppStore()
const wechatAccountName = ref('')

const loading = ref(true)
const status = ref<WeChatBindStatus | null>(null)

// Bind state
const showBindModal = ref(false)
const bindStep = ref<'password' | 'code_display' | 'code_input'>('password')
const bindPassword = ref('')
const shortCode = ref('')
const sceneId = ref('')
const verificationCode = ref('')
const binding = ref(false)
const bindError = ref('')

// Unbind state
const showUnbindDialog = ref(false)
const unbindPassword = ref('')
const unbinding = ref(false)
const unbindError = ref('')

const loadStatus = async () => {
  loading.value = true
  try {
    status.value = await wechatAPI.getBindStatus()
  } catch (error) {
    console.error('Failed to load WeChat status:', error)
  } finally {
    loading.value = false
  }
}

const startBindFlow = async () => {
  try {
    const settings = await getPublicSettings()
    wechatAccountName.value = settings.wechat_account_name || ''
  } catch { /* use fallback */ }
  showBindModal.value = true
  bindStep.value = 'password'
  bindPassword.value = ''
  shortCode.value = ''
  sceneId.value = ''
  verificationCode.value = ''
  bindError.value = ''
}

const submitBindPassword = async () => {
  if (!bindPassword.value) return
  binding.value = true
  bindError.value = ''
  try {
    const res = await wechatAPI.bindWeChat(bindPassword.value)
    shortCode.value = res.short_code
    sceneId.value = res.scene_id
    bindStep.value = 'code_display'
  } catch (error: unknown) {
    const err = error as { response?: { data?: { message?: string } }; message?: string }
    bindError.value = err.response?.data?.message || err.message || t('common.error')
  } finally {
    binding.value = false
  }
}

const closeBindModal = () => {
  showBindModal.value = false
  bindStep.value = 'password'
  bindPassword.value = ''
  verificationCode.value = ''
  bindError.value = ''
}

const confirmBind = async () => {
  if (!verificationCode.value) return
  binding.value = true
  bindError.value = ''
  try {
    await wechatAPI.confirmBindWeChat(sceneId.value, verificationCode.value)
    appStore.showSuccess(t('profile.wechat.bindSuccess'))
    closeBindModal()
    loadStatus()
  } catch (error: unknown) {
    const err = error as { response?: { data?: { message?: string } }; message?: string }
    bindError.value = err.response?.data?.message || err.message || t('common.error')
  } finally {
    binding.value = false
  }
}

const handleUnbind = async () => {
  if (!unbindPassword.value) return
  unbinding.value = true
  unbindError.value = ''
  try {
    await wechatAPI.unbindWeChat(unbindPassword.value)
    appStore.showSuccess(t('profile.wechat.unbindSuccess'))
    showUnbindDialog.value = false
    unbindPassword.value = ''
    loadStatus()
  } catch (error: unknown) {
    const err = error as { response?: { data?: { message?: string } }; message?: string }
    unbindError.value = err.response?.data?.message || err.message || t('common.error')
  } finally {
    unbinding.value = false
  }
}

onMounted(() => {
  loadStatus()
})
</script>
