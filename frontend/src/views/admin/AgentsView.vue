<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex w-full flex-col gap-3 md:flex-row md:flex-wrap-reverse md:items-center md:justify-between md:gap-4">
          <!-- Left: Search -->
          <div class="flex min-w-[280px] flex-1 flex-wrap content-start items-center gap-3 md:order-1">
            <!-- Search Box -->
            <div class="relative w-full md:w-64">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
              />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('admin.agents.searchAgents')"
                class="input pl-10"
                @input="handleSearch"
              />
            </div>
          </div>

          <!-- Right: Actions and Settings -->
          <div class="flex w-full items-center justify-between gap-2 md:order-2 md:ml-auto md:max-w-full md:flex-wrap md:justify-end md:gap-3">
            <div class="flex items-center gap-2 md:contents">
              <!-- Refresh Button -->
              <button
                @click="() => loadAgents()"
                :disabled="loading"
                class="btn btn-secondary px-2 md:px-3"
                :title="t('common.refresh')"
              >
                <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
              </button>

              <!-- Column Settings Dropdown -->
              <div class="relative" ref="columnDropdownRef">
                <button
                  @click="showColumnDropdown = !showColumnDropdown"
                  class="btn btn-secondary px-2 md:px-3"
                  :title="t('admin.agents.columnSettings')"
                >
                  <svg class="h-4 w-4 md:mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M9 4.5v15m6-15v15m-10.875 0h15.75c.621 0 1.125-.504 1.125-1.125V5.625c0-.621-.504-1.125-1.125-1.125H4.125C3.504 4.5 3 5.004 3 5.625v12.75c0 .621.504 1.125 1.125 1.125z" />
                  </svg>
                  <span class="hidden md:inline">{{ t('admin.agents.columnSettings') }}</span>
                </button>
                <!-- Dropdown menu -->
                <div
                  v-if="showColumnDropdown"
                  class="absolute right-0 top-full z-50 mt-1 max-h-80 w-48 overflow-y-auto rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-600 dark:bg-dark-800"
                >
                  <button
                    v-for="col in toggleableColumns"
                    :key="col.key"
                    @click="toggleColumn(col.key)"
                    class="flex w-full items-center justify-between px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
                  >
                    <span>{{ col.label }}</span>
                    <Icon
                      v-if="isColumnVisible(col.key)"
                      name="check"
                      size="sm"
                      class="text-primary-500"
                      :stroke-width="2"
                    />
                  </button>
                </div>
              </div>
            </div>

            <!-- Set Agent Button -->
            <button @click="showSetAgentModal = true" class="btn btn-primary flex-1 md:flex-initial">
              <Icon name="plus" size="md" class="mr-2" />
              {{ t('admin.agents.setAgent') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="agents" :loading="loading">
          <template #cell-email="{ value, row }">
            <div class="flex flex-col">
              <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
              <span class="text-xs text-gray-500">{{ row.username }}</span>
            </div>
          </template>

          <template #cell-invite_code="{ value }">
            <div v-if="value" class="flex items-center space-x-2">
              <code class="font-mono text-sm text-gray-900 dark:text-gray-100">{{ value }}</code>
              <button
                @click="copyToClipboard(value)"
                :class="[
                  'flex items-center transition-colors',
                  copiedCode === value
                    ? 'text-green-500'
                    : 'text-gray-400 hover:text-gray-600 dark:hover:text-gray-300'
                ]"
                :title="copiedCode === value ? t('admin.agents.copied') : t('keys.copyToClipboard')"
              >
                <Icon v-if="copiedCode !== value" name="copy" size="sm" :stroke-width="2" />
                <svg v-else class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                </svg>
              </button>
            </div>
            <span v-else class="text-gray-400">-</span>
          </template>

          <template #cell-is_agent="{ value }">
            <span :class="['badge', value ? 'badge-success' : 'badge-secondary']">
              {{ value ? t('admin.agents.statusAgent') : t('admin.agents.statusUser') }}
            </span>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">
              {{ formatDateTime(value) }}
            </span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-2">
              <button
                v-if="row.is_agent"
                @click="viewDownline(row)"
                class="btn btn-sm btn-secondary"
                :title="t('admin.agents.viewDownline')"
              >
                <Icon name="users" size="sm" />
              </button>
              <button
                v-if="row.is_agent"
                @click="viewStats(row)"
                class="btn btn-sm btn-secondary"
                :title="t('admin.agents.viewStats')"
              >
                <Icon name="chartBar" size="sm" />
              </button>
              <button
                v-if="row.is_agent && row.parent_agent_id"
                @click="handleRevokeAgent(row)"
                class="btn btn-sm btn-secondary text-red-500 hover:text-red-600"
                :title="t('admin.agents.revokeAgent')"
              >
                <Icon name="x" size="sm" />
              </button>
            </div>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination && pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
        />
      </template>

      <template #empty>
        <div class="py-12 text-center">
          <Icon name="users" size="xl" class="mx-auto mb-4 text-gray-400" />
          <h3 class="mb-2 text-lg font-medium text-gray-900 dark:text-white">
            {{ t('admin.agents.noAgentsYet') }}
          </h3>
          <p class="text-gray-500 dark:text-dark-400">
            {{ t('admin.agents.createFirstAgent') }}
          </p>
        </div>
      </template>
    </TablePageLayout>

    <!-- Downline Modal -->
    <BaseDialog :show="showDownlineModal" :title="t('admin.agents.downlineTitle')" width="wide" @close="showDownlineModal = false">
      <div v-if="downlineLoading" class="flex justify-center py-8">
        <Icon name="refresh" size="lg" class="animate-spin text-gray-400" />
      </div>
      <div v-else-if="downlineUsers.length === 0" class="py-8 text-center text-gray-500">
        {{ t('common.noData') }}
      </div>
      <div v-else>
        <table class="w-full">
          <thead>
            <tr class="border-b dark:border-dark-600">
              <th class="px-4 py-2 text-left text-sm font-medium text-gray-500">{{ t('admin.agents.columns.email') }}</th>
              <th class="px-4 py-2 text-left text-sm font-medium text-gray-500">{{ t('admin.agents.columns.status') }}</th>
              <th class="px-4 py-2 text-left text-sm font-medium text-gray-500">{{ t('admin.agents.columns.createdAt') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="user in downlineUsers" :key="user.id" class="border-b dark:border-dark-700">
              <td class="px-4 py-3">
                <div class="flex flex-col">
                  <span class="font-medium">{{ user.email }}</span>
                  <span class="text-xs text-gray-500">{{ user.username }}</span>
                </div>
              </td>
              <td class="px-4 py-3">
                <span :class="['badge', user.is_agent ? 'badge-success' : 'badge-secondary']">
                  {{ user.is_agent ? t('admin.agents.statusAgent') : t('admin.agents.statusUser') }}
                </span>
              </td>
              <td class="px-4 py-3 text-sm text-gray-500">{{ formatDateTime(user.created_at) }}</td>
            </tr>
          </tbody>
        </table>
        <Pagination
          v-if="downlinePagination && downlinePagination.total > 0"
          :page="downlinePagination.page"
          :total="downlinePagination.total"
          :page-size="downlinePagination.page_size"
          @update:page="handleDownlinePageChange"
          class="mt-4"
        />
      </div>
    </BaseDialog>

    <!-- Stats Modal -->
    <BaseDialog :show="showStatsModal" :title="t('admin.agents.statsTitle')" width="normal" @close="showStatsModal = false">
      <div v-if="statsLoading" class="flex justify-center py-8">
        <Icon name="refresh" size="lg" class="animate-spin text-gray-400" />
      </div>
      <div v-else-if="inviteStats" class="space-y-4">
        <div class="grid grid-cols-2 gap-4">
          <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-700">
            <div class="text-2xl font-bold text-gray-900 dark:text-white">{{ inviteStats.total_invited }}</div>
            <div class="text-sm text-gray-500">{{ t('admin.agents.totalInvited') }}</div>
          </div>
          <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-700">
            <div class="text-2xl font-bold text-gray-900 dark:text-white">{{ inviteStats.direct_invited }}</div>
            <div class="text-sm text-gray-500">{{ t('admin.agents.directInvited') }}</div>
          </div>
          <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-700">
            <div class="text-2xl font-bold text-green-600">{{ inviteStats.invited_agents }}</div>
            <div class="text-sm text-gray-500">{{ t('admin.agents.invitedAgents') }}</div>
          </div>
          <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-700">
            <div class="text-2xl font-bold text-blue-600">{{ inviteStats.invited_users }}</div>
            <div class="text-sm text-gray-500">{{ t('admin.agents.invitedUsers') }}</div>
          </div>
        </div>
      </div>
    </BaseDialog>

    <!-- Set Agent Modal -->
    <BaseDialog :show="showSetAgentModal" :title="t('admin.agents.setAgentTitle')" width="normal" @close="closeSetAgentModal">
      <div class="space-y-4">
        <div>
          <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.agents.selectUser') }}
          </label>
          <div class="relative">
            <input
              v-model="userSearchQuery"
              type="text"
              :placeholder="t('admin.agents.searchUserPlaceholder')"
              class="input w-full"
              @input="handleUserSearch"
            />
          </div>
          <!-- User search results -->
          <div v-if="userSearchResults.length > 0" class="mt-2 max-h-48 overflow-y-auto rounded-lg border border-gray-200 dark:border-dark-600">
            <button
              v-for="user in userSearchResults"
              :key="user.id"
              @click="selectUser(user)"
              :class="[
                'flex w-full items-center justify-between px-4 py-2 text-left hover:bg-gray-100 dark:hover:bg-dark-700',
                selectedUser?.id === user.id ? 'bg-primary-50 dark:bg-primary-900/20' : ''
              ]"
            >
              <div>
                <div class="font-medium">{{ user.email }}</div>
                <div class="text-xs text-gray-500">{{ user.username }}</div>
              </div>
              <span v-if="user.is_agent" class="badge badge-success text-xs">{{ t('admin.agents.statusAgent') }}</span>
            </button>
          </div>
          <!-- Selected user display -->
          <div v-if="selectedUser" class="mt-2 rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
            <div class="flex items-center justify-between">
              <div>
                <div class="font-medium">{{ selectedUser.email }}</div>
                <div class="text-xs text-gray-500">{{ selectedUser.username }}</div>
              </div>
              <button @click="selectedUser = null" class="text-gray-400 hover:text-gray-600">
                <Icon name="x" size="sm" />
              </button>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button @click="closeSetAgentModal" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button
            @click="setAgentStatus"
            :disabled="!selectedUser || setAgentLoading"
            class="btn btn-primary"
          >
            <Icon v-if="setAgentLoading" name="refresh" size="sm" class="mr-2 animate-spin" />
            {{ selectedUser?.is_agent ? t('admin.agents.removeAgent') : t('admin.agents.confirmSetAgent') }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { User } from '@/types'
import type { InviteStats } from '@/api/admin/agents'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

// State
const agents = ref<User[]>([])
const loading = ref(false)
const searchQuery = ref('')
const copiedCode = ref<string | null>(null)
const pagination = ref<{
  total: number
  page: number
  page_size: number
  pages: number
} | null>(null)

// Downline modal state
const showDownlineModal = ref(false)
const downlineLoading = ref(false)
const downlineUsers = ref<User[]>([])
const downlinePagination = ref<{
  total: number
  page: number
  page_size: number
  pages: number
} | null>(null)
const selectedAgentId = ref<number | null>(null)

// Stats modal state
const showStatsModal = ref(false)
const statsLoading = ref(false)
const inviteStats = ref<InviteStats | null>(null)

// Column settings dropdown
const columnDropdownRef = ref<HTMLElement | null>(null)
const showColumnDropdown = ref(false)
const hiddenColumns = ref<Set<string>>(new Set())

// Set agent modal state
const showSetAgentModal = ref(false)
const setAgentLoading = ref(false)
const userSearchQuery = ref('')
const userSearchResults = ref<User[]>([])
const selectedUser = ref<User | null>(null)
let userSearchTimeout: ReturnType<typeof setTimeout> | null = null
let userSearchAbortController: AbortController | null = null

// Table columns - all available columns
const allColumns = computed(() => [
  { key: 'email', label: t('admin.agents.columns.email'), sortable: true },
  { key: 'invite_code', label: t('admin.agents.columns.inviteCode') },
  { key: 'is_agent', label: t('admin.agents.columns.status') },
  { key: 'created_at', label: t('admin.agents.columns.createdAt'), sortable: true },
  { key: 'actions', label: t('admin.agents.columns.actions'), align: 'right' as const }
])

// Visible columns (filtered by hiddenColumns)
const columns = computed(() =>
  allColumns.value.filter(col => !hiddenColumns.value.has(col.key))
)

// Toggleable columns (exclude actions)
const toggleableColumns = computed(() =>
  allColumns.value.filter(col => col.key !== 'actions')
)

// Column visibility functions
const isColumnVisible = (key: string) => !hiddenColumns.value.has(key)
const toggleColumn = (key: string) => {
  const newHidden = new Set(hiddenColumns.value)
  if (newHidden.has(key)) {
    newHidden.delete(key)
  } else {
    newHidden.add(key)
  }
  hiddenColumns.value = newHidden
}

// Debounce search
let searchTimeout: ReturnType<typeof setTimeout> | null = null
const handleSearch = () => {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    loadAgents(1)
  }, 300)
}

// Load agents
const loadAgents = async (page = 1) => {
  loading.value = true
  try {
    const response = await adminAPI.agents.list(page, 20, {
      search: searchQuery.value || undefined
    })
    agents.value = response.items
    pagination.value = {
      total: response.total,
      page: response.page,
      page_size: response.page_size,
      pages: response.pages
    }
  } catch (error: any) {
    appStore.showError(t('admin.agents.failedToLoad'))
    console.error('Failed to load agents:', error)
  } finally {
    loading.value = false
  }
}

// Handle page change
const handlePageChange = (page: number) => {
  loadAgents(page)
}

// Copy to clipboard
const copyToClipboard = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    copiedCode.value = text
    setTimeout(() => {
      copiedCode.value = null
    }, 2000)
  } catch (error) {
    console.error('Failed to copy:', error)
  }
}

// Format date time
const formatDateTime = (dateStr: string) => {
  return new Date(dateStr).toLocaleString()
}

// View downline
const viewDownline = async (agent: User) => {
  selectedAgentId.value = agent.id
  // Clear old data before opening modal to prevent stale data display on error
  downlineUsers.value = []
  downlinePagination.value = null
  showDownlineModal.value = true
  await loadDownline(1)
}

// Load downline users
const loadDownline = async (page = 1) => {
  if (!selectedAgentId.value) return
  downlineLoading.value = true
  try {
    const response = await adminAPI.agents.getDownline(selectedAgentId.value, page, 10)
    downlineUsers.value = response.items
    downlinePagination.value = {
      total: response.total,
      page: response.page,
      page_size: response.page_size,
      pages: response.pages
    }
  } catch (error: any) {
    appStore.showError(t('admin.agents.failedToLoad'))
    console.error('Failed to load downline:', error)
  } finally {
    downlineLoading.value = false
  }
}

// Handle downline page change
const handleDownlinePageChange = (page: number) => {
  loadDownline(page)
}

// View stats
const viewStats = async (agent: User) => {
  // Clear old data before opening modal to prevent stale data display on error
  inviteStats.value = null
  showStatsModal.value = true
  statsLoading.value = true
  try {
    inviteStats.value = await adminAPI.agents.getInviteStats(agent.id)
  } catch (error: any) {
    appStore.showError(t('admin.agents.failedToLoad'))
    console.error('Failed to load stats:', error)
  } finally {
    statsLoading.value = false
  }
}

// Revoke agent status
const handleRevokeAgent = async (agent: User) => {
  const confirmed = window.confirm(t('admin.agents.revokeAgentConfirm', { email: agent.email }))
  if (!confirmed) return

  try {
    await adminAPI.agents.setAgentStatus(agent.id, { is_agent: false })
    appStore.showToast('success', t('admin.agents.revokeAgentSuccess'))
    loadAgents()
  } catch (error: any) {
    console.error('Failed to revoke agent:', error)
    appStore.showToast('error', t('admin.agents.revokeAgentFailed'))
  }
}

// Set agent modal functions
const handleUserSearch = () => {
  if (userSearchTimeout) clearTimeout(userSearchTimeout)
  // Cancel any in-flight request
  if (userSearchAbortController) {
    userSearchAbortController.abort()
    userSearchAbortController = null
  }
  userSearchTimeout = setTimeout(async () => {
    if (!userSearchQuery.value.trim()) {
      userSearchResults.value = []
      return
    }
    // Create new AbortController for this request
    userSearchAbortController = new AbortController()
    try {
      const response = await adminAPI.users.list(1, 10, {
        search: userSearchQuery.value
      }, {
        signal: userSearchAbortController.signal
      })
      userSearchResults.value = response.items
    } catch (error: any) {
      // Ignore abort errors
      if (error.name !== 'AbortError' && error.code !== 'ERR_CANCELED') {
        console.error('Failed to search users:', error)
      }
    }
  }, 300)
}

const selectUser = (user: User) => {
  selectedUser.value = user
  userSearchResults.value = []
  userSearchQuery.value = ''
}

const closeSetAgentModal = () => {
  showSetAgentModal.value = false
  selectedUser.value = null
  userSearchQuery.value = ''
  userSearchResults.value = []
  // Cancel any in-flight search request
  if (userSearchAbortController) {
    userSearchAbortController.abort()
    userSearchAbortController = null
  }
}

const setAgentStatus = async () => {
  if (!selectedUser.value) return
  setAgentLoading.value = true
  try {
    await adminAPI.agents.setAgentStatus(selectedUser.value.id, {
      is_agent: !selectedUser.value.is_agent
    })
    appStore.showSuccess(
      selectedUser.value.is_agent
        ? t('admin.agents.agentRemoved')
        : t('admin.agents.agentSet')
    )
    closeSetAgentModal()
    loadAgents()
  } catch (error: any) {
    appStore.showError(error.message || t('admin.agents.failedToSetAgent'))
  } finally {
    setAgentLoading.value = false
  }
}

// Click outside handler for dropdowns
const handleClickOutside = (event: MouseEvent) => {
  if (columnDropdownRef.value && !columnDropdownRef.value.contains(event.target as Node)) {
    showColumnDropdown.value = false
  }
}

// Initialize
onMounted(() => {
  loadAgents()
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  // Clean up timers to prevent memory leaks
  if (searchTimeout) clearTimeout(searchTimeout)
  if (userSearchTimeout) clearTimeout(userSearchTimeout)
  // Cancel any in-flight search request
  if (userSearchAbortController) {
    userSearchAbortController.abort()
    userSearchAbortController = null
  }
  document.removeEventListener('click', handleClickOutside)
})
</script>
