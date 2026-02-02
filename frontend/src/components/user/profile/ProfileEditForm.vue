<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-medium text-gray-900 dark:text-white">
        {{ t('profile.editProfile') }}
      </h2>
    </div>
    <div class="px-6 py-6">
      <form @submit.prevent="handleUpdateProfile" class="space-y-4">
        <div>
          <label for="username" class="input-label">
            {{ t('profile.username') }}
          </label>
          <input
            id="username"
            v-model="username"
            type="text"
            class="input"
            :placeholder="t('profile.enterUsername')"
          />
        </div>

        <!-- Dynamic user attributes -->
        <div v-for="attr in attributeDefinitions" :key="attr.id">
          <label :for="`attr-${attr.id}`" class="input-label">
            {{ attr.name }}
            <span v-if="attr.required" class="text-red-500">*</span>
          </label>
          <input
            :id="`attr-${attr.id}`"
            v-model="attributeValues[attr.id]"
            type="text"
            class="input"
            :placeholder="attr.placeholder || attr.description"
          />
          <p v-if="attr.description" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ attr.description }}
          </p>
        </div>

        <div class="flex justify-end pt-4">
          <button type="submit" :disabled="loading" class="btn btn-primary">
            {{ loading ? t('profile.updating') : t('profile.updateProfile') }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { userAPI, type UserAttributeDefinition } from '@/api'

const props = defineProps<{
  initialUsername: string
}>()

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const username = ref(props.initialUsername)
const loading = ref(false)

// User attributes
const attributeDefinitions = ref<UserAttributeDefinition[]>([])
const attributeValues = reactive<Record<number, string>>({})

watch(() => props.initialUsername, (val) => {
  username.value = val
})

// Load attribute definitions and values on mount
onMounted(async () => {
  try {
    const [defs, values] = await Promise.all([
      userAPI.getAttributeDefinitions(),
      userAPI.getMyAttributes()
    ])
    attributeDefinitions.value = defs
    // Initialize attribute values
    for (const def of defs) {
      attributeValues[def.id] = values[def.id] || ''
    }
  } catch (error) {
    console.error('Failed to load attributes:', error)
  }
})

const handleUpdateProfile = async () => {
  if (!username.value.trim()) {
    appStore.showError(t('profile.usernameRequired'))
    return
  }

  loading.value = true
  try {
    // Update profile and attributes in parallel
    const [updatedUser] = await Promise.all([
      userAPI.updateProfile({ username: username.value }),
      userAPI.updateMyAttributes(attributeValues)
    ])
    authStore.user = updatedUser
    appStore.showSuccess(t('profile.updateSuccess'))
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('profile.updateFailed'))
  } finally {
    loading.value = false
  }
}
</script>
