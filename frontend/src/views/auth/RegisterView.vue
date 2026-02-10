<template>
  <AuthLayout>
    <div class="space-y-6">
      <!-- Title -->
      <div class="text-center">
        <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
          {{ t('auth.createAccount') }}
        </h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          {{ t('auth.signUpToStart', { siteName }) }}
        </p>
      </div>

      <!-- Step Indicator (only when wechat enabled) -->
      <div v-if="wechatEnabled && registrationEnabled && settingsLoaded" class="flex items-center justify-center gap-2">
        <div class="h-2 w-2 rounded-full transition-colors" :class="currentStep === 1 ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600'" />
        <div class="h-px w-6 transition-colors" :class="currentStep === 2 ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600'" />
        <div class="h-2 w-2 rounded-full transition-colors" :class="currentStep === 2 ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600'" />
      </div>

      <!-- LinuxDo Connect OAuth 登录 -->
      <LinuxDoOAuthSection v-if="linuxdoOAuthEnabled" :disabled="isLoading" />

      <!-- Registration Disabled Message -->
      <div
        v-if="!registrationEnabled && settingsLoaded"
        class="rounded-xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-800/50 dark:bg-amber-900/20"
      >
        <div class="flex items-start gap-3">
          <div class="flex-shrink-0">
            <Icon name="exclamationCircle" size="md" class="text-amber-500" />
          </div>
          <p class="text-sm text-amber-700 dark:text-amber-400">
            {{ t('auth.registrationDisabled') }}
          </p>
        </div>
      </div>

      <!-- Registration Form -->
      <form v-else @submit.prevent="currentStep === 1 ? handleNext() : handleRegister()" class="space-y-5">

        <!-- ==================== Step 1: Account Info ==================== -->
        <template v-if="currentStep === 1">
          <!-- Email Input -->
          <div>
            <label for="email" class="input-label">
              {{ t('auth.emailLabel') }}
            </label>
            <div class="relative">
              <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5">
                <Icon name="mail" size="md" class="text-gray-400 dark:text-dark-500" />
              </div>
              <input
                id="email"
                v-model="formData.email"
                type="email"
                required
                autofocus
                autocomplete="email"
                :disabled="isLoading"
                class="input pl-11"
                :class="{ 'input-error': errors.email }"
                :placeholder="t('auth.emailPlaceholder')"
              />
            </div>
            <p v-if="errors.email" class="input-error-text">
              {{ errors.email }}
            </p>
          </div>

          <!-- Password Input -->
          <div>
            <label for="password" class="input-label">
              {{ t('auth.passwordLabel') }}
            </label>
            <div class="relative">
              <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5">
                <Icon name="lock" size="md" class="text-gray-400 dark:text-dark-500" />
              </div>
              <input
                id="password"
                v-model="formData.password"
                :type="showPassword ? 'text' : 'password'"
                required
                autocomplete="new-password"
                :disabled="isLoading"
                class="input pl-11 pr-11"
                :class="{ 'input-error': errors.password }"
                :placeholder="t('auth.createPasswordPlaceholder')"
              />
              <button
                type="button"
                @click="showPassword = !showPassword"
                class="absolute inset-y-0 right-0 flex items-center pr-3.5 text-gray-400 transition-colors hover:text-gray-600 dark:hover:text-dark-300"
              >
                <Icon v-if="showPassword" name="eyeOff" size="md" />
                <Icon v-else name="eye" size="md" />
              </button>
            </div>
            <p v-if="errors.password" class="input-error-text">
              {{ errors.password }}
            </p>
            <p v-else class="input-hint">
              {{ t('auth.passwordHint') }}
            </p>
          </div>

          <!-- Invite Code Input -->
          <div v-if="promoCodeEnabled">
            <label for="promo_code" class="input-label">
              {{ t('auth.promoCodeLabel') }}
            </label>
            <div class="relative">
              <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5">
                <Icon name="gift" size="md" :class="promoValidation.valid ? 'text-green-500' : 'text-gray-400 dark:text-dark-500'" />
              </div>
              <input
                id="promo_code"
                v-model="formData.promo_code"
                type="text"
                :disabled="isLoading"
                class="input pl-11 pr-10"
                :class="{
                  'border-green-500 focus:border-green-500 focus:ring-green-500': promoValidation.valid,
                  'border-red-500 focus:border-red-500 focus:ring-red-500': promoValidation.invalid
                }"
                :placeholder="t('auth.promoCodePlaceholder')"
                @input="handlePromoCodeInput"
              />
              <div v-if="promoValidating" class="absolute inset-y-0 right-0 flex items-center pr-3.5">
                <svg class="h-4 w-4 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
              </div>
              <div v-else-if="promoValidation.valid" class="absolute inset-y-0 right-0 flex items-center pr-3.5">
                <Icon name="checkCircle" size="md" class="text-green-500" />
              </div>
              <div v-else-if="promoValidation.invalid" class="absolute inset-y-0 right-0 flex items-center pr-3.5">
                <Icon name="exclamationCircle" size="md" class="text-red-500" />
              </div>
            </div>
            <transition name="fade">
              <div v-if="promoValidation.valid" class="mt-2 flex items-center gap-2 rounded-lg bg-green-50 px-3 py-2 dark:bg-green-900/20">
                <Icon name="gift" size="sm" class="text-green-600 dark:text-green-400" />
                <span class="text-sm text-green-700 dark:text-green-400">
                  {{ t('auth.promoCodeValid', { amount: promoValidation.bonusAmount?.toFixed(2) }) }}
                </span>
              </div>
              <p v-else-if="promoValidation.invalid" class="input-error-text">
                {{ promoValidation.message }}
              </p>
            </transition>
          </div>

          <!-- Turnstile Widget -->
          <div v-if="turnstileEnabled && turnstileSiteKey">
            <TurnstileWidget
              ref="turnstileRef"
              :site-key="turnstileSiteKey"
              @verify="onTurnstileVerify"
              @expire="onTurnstileExpire"
              @error="onTurnstileError"
            />
            <p v-if="errors.turnstile" class="input-error-text mt-2 text-center">
              {{ errors.turnstile }}
            </p>
          </div>
        </template>

        <!-- ==================== Step 2: WeChat Verification ==================== -->
        <template v-if="currentStep === 2">
          <!-- Instruction -->
          <div class="rounded-lg border border-primary-200 bg-primary-50 p-4 dark:border-primary-800/50 dark:bg-primary-900/20">
            <div class="space-y-2.5">
              <template v-if="wechatAccountName">
                <p class="text-center text-sm text-primary-700 dark:text-primary-300">
                  {{ t('auth.wechat.followAccount') }}
                  <span class="font-bold text-red-600 dark:text-red-400">{{ wechatAccountName }}</span>
                </p>
                <div class="flex flex-wrap items-center justify-center gap-x-1.5 gap-y-1 rounded bg-white/60 px-3 py-2 text-xs text-gray-600 dark:bg-gray-800/40 dark:text-gray-400">
                  <span>{{ t('auth.wechat.searchFollow', { account: wechatAccountName }) }}</span>
                </div>
              </template>
              <p v-else class="text-center text-sm text-primary-700 dark:text-primary-300">
                {{ t('auth.wechat.followAccountGeneric') }}
              </p>
            </div>
          </div>

          <!-- Short Code Card -->
          <div class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-700 dark:bg-gray-800/50">
            <div class="flex flex-col items-center space-y-3">
              <!-- Loading -->
              <div v-if="wechatLoading && !wechatShortCode" class="flex items-center gap-2 py-4 text-gray-500">
                <div class="h-5 w-5 animate-spin rounded-full border-b-2 border-primary-600"></div>
                <span class="text-sm">{{ t('common.loading') }}</span>
              </div>

              <!-- Short Code Display -->
              <template v-if="wechatShortCode">
                <div class="flex items-center gap-2">
                  <span
                    class="rounded-lg bg-white px-4 py-2 font-mono text-2xl font-bold tracking-[0.3em] text-gray-900 shadow-sm ring-1 ring-gray-200 dark:bg-gray-900 dark:text-white dark:ring-gray-700"
                    :class="{ 'opacity-40': wechatScanStatus === 'expired' }"
                  >
                    {{ wechatShortCode }}
                  </span>
                  <button
                    type="button"
                    @click="copyShortCode"
                    class="rounded-md p-2 text-gray-400 hover:bg-gray-200 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-300"
                    title="Copy"
                  >
                    <Icon name="clipboard" size="md" />
                  </button>
                </div>

                <!-- Status -->
                <div v-if="wechatScanStatus === 'code_sent'" class="flex items-center gap-1.5 text-green-600">
                  <Icon name="checkCircle" size="sm" />
                  <span class="text-sm font-medium">{{ t('auth.wechat.codeSentSuccess') }}</span>
                </div>
                <div v-else-if="wechatScanStatus === 'expired'" class="flex flex-col items-center gap-1.5">
                  <span class="text-sm text-yellow-600">{{ t('auth.wechat.shortCodeExpired') }}</span>
                  <button type="button" @click="refreshShortCode" class="flex items-center gap-1 text-xs font-medium text-primary-600 hover:underline">
                    <Icon name="refresh" size="sm" />
                    {{ t('auth.wechat.refreshShortCode') }}
                  </button>
                </div>
                <div v-else class="flex items-center gap-1.5 text-gray-500">
                  <div class="h-3 w-3 animate-pulse rounded-full bg-yellow-400"></div>
                  <span class="text-xs">{{ t('auth.wechat.waitingForCode') }}</span>
                </div>
              </template>
            </div>
          </div>

          <!-- Verification Code Input -->
          <div>
            <label for="wechat_code" class="input-label">
              {{ t('auth.wechat.verifyCodeLabel') }}
            </label>
            <input
              id="wechat_code"
              v-model="formData.wechat_verify_code"
              type="text"
              maxlength="6"
              inputmode="numeric"
              placeholder="000000"
              class="input text-center font-mono tracking-[0.2em]"
              :class="{ 'input-error': errors.wechat }"
              :disabled="wechatLoading"
            />
            <p v-if="errors.wechat" class="input-error-text">
              {{ errors.wechat }}
            </p>
            <p v-else class="input-hint">
              {{ t('auth.wechat.verifyCodeHint') }}
            </p>
          </div>
        </template>

        <!-- ==================== Error Message (both steps) ==================== -->
        <transition name="fade">
          <div
            v-if="errorMessage"
            class="rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-800/50 dark:bg-red-900/20"
          >
            <div class="flex items-start gap-3">
              <div class="flex-shrink-0">
                <Icon name="exclamationCircle" size="md" class="text-red-500" />
              </div>
              <p class="text-sm text-red-700 dark:text-red-400">
                {{ errorMessage }}
              </p>
            </div>
          </div>
        </transition>

        <!-- ==================== Action Buttons ==================== -->
        <!-- Step 1: Next / Submit -->
        <template v-if="currentStep === 1">
          <button
            type="submit"
            :disabled="isLoading || (turnstileEnabled && !turnstileToken)"
            class="btn btn-primary w-full"
          >
            <template v-if="wechatEnabled">
              <Icon name="arrowRight" size="md" class="mr-2" />
              {{ t('auth.wechat.stepNext') }}
            </template>
            <template v-else>
              <svg
                v-if="isLoading"
                class="-ml-1 mr-2 h-4 w-4 animate-spin text-white"
                fill="none"
                viewBox="0 0 24 24"
              >
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              <Icon v-else name="userPlus" size="md" class="mr-2" />
              {{ isLoading ? t('auth.processing') : t('auth.createAccount') }}
            </template>
          </button>
        </template>

        <!-- Step 2: Back + Create Account -->
        <template v-if="currentStep === 2">
          <div class="flex gap-3">
            <button
              type="button"
              @click="handleBack"
              class="btn flex-1 border border-gray-300 bg-white text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"
            >
              <Icon name="arrowLeft" size="md" class="mr-2" />
              {{ t('auth.wechat.stepBack') }}
            </button>
            <button
              type="submit"
              :disabled="isLoading"
              class="btn btn-primary flex-1"
            >
              <svg
                v-if="isLoading"
                class="-ml-1 mr-2 h-4 w-4 animate-spin text-white"
                fill="none"
                viewBox="0 0 24 24"
              >
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              <Icon v-else name="userPlus" size="md" class="mr-2" />
              {{ isLoading ? t('auth.processing') : t('auth.createAccount') }}
            </button>
          </div>
        </template>
      </form>
    </div>

    <!-- Footer -->
    <template #footer>
      <p class="text-gray-500 dark:text-dark-400">
        {{ t('auth.alreadyHaveAccount') }}
        <router-link
          to="/login"
          class="font-medium text-primary-600 transition-colors hover:text-primary-500 dark:text-primary-400 dark:hover:text-primary-300"
        >
          {{ t('auth.signIn') }}
        </router-link>
      </p>
    </template>
  </AuthLayout>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { AuthLayout } from '@/components/layout'
import LinuxDoOAuthSection from '@/components/auth/LinuxDoOAuthSection.vue'
import Icon from '@/components/icons/Icon.vue'
import TurnstileWidget from '@/components/TurnstileWidget.vue'
import { useAuthStore, useAppStore } from '@/stores'
import { getPublicSettings, validateInviteCode } from '@/api/auth'
import { createShortCode, checkScanStatus } from '@/api/wechat'

const { t } = useI18n()

// ==================== Router & Stores ====================

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const appStore = useAppStore()

// ==================== State ====================

const isLoading = ref<boolean>(false)
const settingsLoaded = ref<boolean>(false)
const errorMessage = ref<string>('')
const showPassword = ref<boolean>(false)
const currentStep = ref<number>(1)

// Public settings
const registrationEnabled = ref<boolean>(true)
const promoCodeEnabled = ref<boolean>(true)
const turnstileEnabled = ref<boolean>(false)
const turnstileSiteKey = ref<string>('')
const siteName = ref<string>('Sub2API')
const linuxdoOAuthEnabled = ref<boolean>(false)

// WeChat verification
const wechatEnabled = ref<boolean>(false)
const wechatAccountName = ref<string>('')
const wechatLoading = ref<boolean>(false)
const wechatShortCode = ref<string>('')
const wechatSceneID = ref<string>('')
const wechatScanStatus = ref<'idle' | 'pending' | 'code_sent' | 'expired'>('idle')
let wechatPollTimer: ReturnType<typeof setInterval> | null = null

// Turnstile
const turnstileRef = ref<InstanceType<typeof TurnstileWidget> | null>(null)
const turnstileToken = ref<string>('')

// Promo code validation
const promoValidating = ref<boolean>(false)
const promoValidation = reactive({
  valid: false,
  invalid: false,
  bonusAmount: null as number | null,
  message: ''
})
let promoValidateTimeout: ReturnType<typeof setTimeout> | null = null

const formData = reactive({
  email: '',
  password: '',
  promo_code: '',
  wechat_verify_code: ''
})

const errors = reactive({
  email: '',
  password: '',
  turnstile: '',
  wechat: ''
})

// ==================== Lifecycle ====================

onMounted(async () => {
  try {
    const settings = await getPublicSettings()
    registrationEnabled.value = settings.registration_enabled
    promoCodeEnabled.value = settings.invite_registration_enabled
    turnstileEnabled.value = settings.turnstile_enabled
    turnstileSiteKey.value = settings.turnstile_site_key || ''
    siteName.value = settings.site_name || 'Sub2API'
    linuxdoOAuthEnabled.value = settings.linuxdo_oauth_enabled
    wechatEnabled.value = settings.wechat_enabled
    wechatAccountName.value = settings.wechat_account_name || ''

    // Read promo code from URL parameter only if promo code is enabled
    // Support both ?promo_code=xxx and ?promo=xxx
    if (promoCodeEnabled.value) {
      const promoParam = (route.query.promo_code || route.query.promo) as string
      if (promoParam) {
        formData.promo_code = promoParam
        // Validate the invite code from URL
        await validateInviteCodeDebounced(promoParam)
      }
    }
  } catch (error) {
    console.error('Failed to load public settings:', error)
  } finally {
    settingsLoaded.value = true
  }
})

onUnmounted(() => {
  if (promoValidateTimeout) {
    clearTimeout(promoValidateTimeout)
  }
  if (wechatPollTimer) {
    clearInterval(wechatPollTimer)
  }
})

// ==================== Promo Code Validation ====================

function handlePromoCodeInput(): void {
  const code = formData.promo_code.trim()

  // Clear previous validation
  promoValidation.valid = false
  promoValidation.invalid = false
  promoValidation.bonusAmount = null
  promoValidation.message = ''

  if (!code) {
    promoValidating.value = false
    return
  }

  // Debounce validation
  if (promoValidateTimeout) {
    clearTimeout(promoValidateTimeout)
  }

  promoValidateTimeout = setTimeout(() => {
    validateInviteCodeDebounced(code)
  }, 500)
}

async function validateInviteCodeDebounced(code: string): Promise<void> {
  if (!code.trim()) return

  // 前端格式校验：必须是 16 位十六进制字符
  const trimmedCode = code.trim().toUpperCase()
  if (!isValidInviteCodeFormat(trimmedCode)) {
    promoValidation.valid = false
    promoValidation.invalid = true
    promoValidation.bonusAmount = null
    promoValidation.message = getPromoErrorMessage('INVITE_CODE_INVALID_FORMAT')
    return
  }

  promoValidating.value = true

  try {
    const result = await validateInviteCode(code)

    if (result.valid) {
      promoValidation.valid = true
      promoValidation.invalid = false
      promoValidation.bonusAmount = result.bonus_amount || 0
      promoValidation.message = ''
    } else {
      promoValidation.valid = false
      promoValidation.invalid = true
      promoValidation.bonusAmount = null
      // 根据错误码显示对应的翻译
      promoValidation.message = getPromoErrorMessage(result.error_code)
    }
  } catch (error) {
    console.error('Failed to validate promo code:', error)
    promoValidation.valid = false
    promoValidation.invalid = true
    promoValidation.message = t('auth.promoCodeInvalid')
  } finally {
    promoValidating.value = false
  }
}

function getPromoErrorMessage(errorCode?: string): string {
  switch (errorCode) {
    case 'INVITE_CODE_NOT_FOUND':
      return t('auth.promoCodeNotFound')
    case 'INVITE_CODE_EXPIRED':
      return t('auth.promoCodeExpired')
    case 'INVITE_CODE_DISABLED':
      return t('auth.promoCodeDisabled')
    case 'INVITER_NOT_ACTIVE':
      return t('auth.inviterNotActive')
    case 'INVITE_CODE_MAX_USED':
      return t('auth.promoCodeMaxUsed')
    case 'INVITE_CODE_ALREADY_USED':
      return t('auth.promoCodeAlreadyUsed')
    case 'INVITE_CODE_INVALID_FORMAT':
      return t('auth.promoCodeInvalidFormat')
    default:
      return t('auth.promoCodeInvalid')
  }
}

// 校验邀请码格式：必须是 16 位十六进制字符
function isValidInviteCodeFormat(code: string): boolean {
  if (code.length !== 16) return false
  return /^[0-9A-F]{16}$/.test(code)
}

// ==================== WeChat Verification ====================

async function startWeChatShortCode(): Promise<void> {
  wechatLoading.value = true
  wechatScanStatus.value = 'idle'
  try {
    const res = await createShortCode()
    // Guard: user may have navigated back during the request
    if (currentStep.value !== 2) return
    wechatSceneID.value = res.scene_id
    wechatShortCode.value = res.short_code
    wechatScanStatus.value = 'pending'
    startPolling(res.scene_id)
  } catch (err) {
    if (currentStep.value !== 2) return
    console.error('Failed to create WeChat short code:', err)
    errors.wechat = t('auth.wechat.shortCodeFailed')
  } finally {
    wechatLoading.value = false
  }
}

function startPolling(sceneID: string): void {
  if (wechatPollTimer) clearInterval(wechatPollTimer)
  wechatPollTimer = setInterval(async () => {
    try {
      const res = await checkScanStatus(sceneID)
      wechatScanStatus.value = res.status
      if (res.status === 'code_sent' || res.status === 'expired') {
        if (wechatPollTimer) {
          clearInterval(wechatPollTimer)
          wechatPollTimer = null
        }
      }
    } catch {
      // Silently ignore polling errors
    }
  }, 2000)
}

function refreshShortCode(): void {
  wechatShortCode.value = ''
  wechatSceneID.value = ''
  wechatScanStatus.value = 'idle'
  formData.wechat_verify_code = ''
  errors.wechat = ''
  startWeChatShortCode()
}

function copyShortCode(): void {
  if (wechatShortCode.value) {
    navigator.clipboard.writeText(wechatShortCode.value).catch(() => {})
  }
}

// ==================== Turnstile Handlers ====================

function onTurnstileVerify(token: string): void {
  turnstileToken.value = token
  errors.turnstile = ''
}

function onTurnstileExpire(): void {
  turnstileToken.value = ''
  errors.turnstile = t('auth.turnstileExpired')
}

function onTurnstileError(): void {
  turnstileToken.value = ''
  errors.turnstile = t('auth.turnstileFailed')
}

// ==================== Validation ====================

function validateEmail(email: string): boolean {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  return emailRegex.test(email)
}

function validateStep1(): boolean {
  errors.email = ''
  errors.password = ''
  errors.turnstile = ''
  errorMessage.value = ''

  let isValid = true

  if (!formData.email.trim()) {
    errors.email = t('auth.emailRequired')
    isValid = false
  } else if (!validateEmail(formData.email)) {
    errors.email = t('auth.invalidEmail')
    isValid = false
  }

  if (!formData.password) {
    errors.password = t('auth.passwordRequired')
    isValid = false
  } else if (formData.password.length < 6) {
    errors.password = t('auth.passwordMinLength')
    isValid = false
  }

  if (turnstileEnabled.value && !turnstileToken.value) {
    errors.turnstile = t('auth.completeVerification')
    isValid = false
  }

  return isValid
}

function validateStep2(): boolean {
  errors.wechat = ''

  if (!wechatEnabled.value) return true

  if (!formData.wechat_verify_code.trim()) {
    errors.wechat = t('auth.wechat.codeRequired')
    return false
  }
  if (!/^\d{6}$/.test(formData.wechat_verify_code.trim())) {
    errors.wechat = t('auth.wechat.codeInvalid')
    return false
  }
  return true
}


// ==================== Form Handlers ====================

// ==================== Step Navigation ====================

function handleNext(): void {
  errorMessage.value = ''

  if (!validateStep1()) return

  // Validate invite code before proceeding
  if (promoCodeEnabled.value) {
    if (!formData.promo_code.trim()) {
      errorMessage.value = t('auth.inviteCodeRequired')
      return
    }
    if (promoValidating.value) {
      errorMessage.value = t('auth.promoCodeValidating')
      return
    }
    if (promoValidation.invalid) {
      errorMessage.value = t('auth.promoCodeInvalidCannotRegister')
      return
    }
    if (!promoValidation.valid) {
      errorMessage.value = t('auth.inviteCodeRequired')
      return
    }
  }

  // If wechat not enabled, submit directly
  if (!wechatEnabled.value) {
    handleRegister()
    return
  }

  // Move to step 2 and start WeChat short code
  currentStep.value = 2
  startWeChatShortCode()
}

function handleBack(): void {
  if (wechatPollTimer) {
    clearInterval(wechatPollTimer)
    wechatPollTimer = null
  }
  wechatShortCode.value = ''
  wechatSceneID.value = ''
  wechatScanStatus.value = 'idle'
  formData.wechat_verify_code = ''
  errors.wechat = ''
  errorMessage.value = ''
  // Reset turnstile to force fresh solve on step 1
  if (turnstileRef.value) {
    turnstileRef.value.reset()
    turnstileToken.value = ''
  }
  currentStep.value = 1
}

// ==================== Form Handlers ====================

async function handleRegister(): Promise<void> {
  errorMessage.value = ''

  // Validate wechat step if enabled
  if (wechatEnabled.value) {
    if (!validateStep2()) return
    if (!wechatSceneID.value) {
      errors.wechat = t('auth.wechat.shortCodeFailed')
      return
    }
  }

  isLoading.value = true

  try {
    await authStore.register({
      email: formData.email,
      password: formData.password,
      turnstile_token: turnstileEnabled.value ? turnstileToken.value : undefined,
      promo_code: formData.promo_code || undefined,
      wechat_verify_code: wechatEnabled.value ? formData.wechat_verify_code.trim() : undefined,
      scene_id: wechatEnabled.value && wechatSceneID.value ? wechatSceneID.value : undefined
    })

    // Show success toast
    appStore.showSuccess(t('auth.accountCreatedSuccess', { siteName: siteName.value }))

    // Redirect to dashboard
    await router.push('/dashboard')
  } catch (error: unknown) {
    // Reset Turnstile on error
    if (turnstileRef.value) {
      turnstileRef.value.reset()
      turnstileToken.value = ''
    }

    // Handle registration error
    const err = error as { message?: string; response?: { data?: { detail?: string } } }

    if (err.response?.data?.detail) {
      errorMessage.value = err.response.data.detail
    } else if (err.message) {
      errorMessage.value = err.message
    } else {
      errorMessage.value = t('auth.registrationFailed')
    }

    // Also show error toast
    appStore.showError(errorMessage.value)
  } finally {
    isLoading.value = false
  }
}
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
