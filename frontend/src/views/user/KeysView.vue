<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="flex justify-end gap-3">
        <button
          @click="loadApiKeys"
          :disabled="loading"
          class="btn btn-secondary"
          :title="t('common.refresh')"
        >
          <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
        </button>
        <button @click="showCreateModal = true" class="btn btn-primary" data-tour="keys-create-btn">
          <Icon name="plus" size="md" class="mr-2" />
          {{ t('keys.createKey') }}
        </button>
      </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="apiKeys" :loading="loading" :expanded-row-keys="expandedKeyIds" row-key="id">
          <template #cell-key="{ value, row }">
            <div class="flex items-center gap-2">
              <code class="code text-xs">
                {{ maskKey(value) }}
              </code>
              <button
                @click="copyToClipboard(value, row.id)"
                class="rounded-lg p-1 transition-colors hover:bg-gray-100 dark:hover:bg-dark-700"
                :class="
                  copiedKeyId === row.id
                    ? 'text-green-500'
                    : 'text-gray-400 hover:text-gray-600 dark:hover:text-gray-300'
                "
                :title="copiedKeyId === row.id ? t('keys.copied') : t('keys.copyToClipboard')"
              >
                <Icon
                  v-if="copiedKeyId === row.id"
                  name="check"
                  size="sm"
                  :stroke-width="2"
                />
                <Icon v-else name="clipboard" size="sm" />
              </button>
            </div>
          </template>

          <template #cell-name="{ value, row }">
            <div class="flex items-center gap-1.5">
              <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
              <Icon
                v-if="row.ip_whitelist?.length > 0 || row.ip_blacklist?.length > 0"
                name="shield"
                size="sm"
                class="text-blue-500"
                :title="t('keys.ipRestrictionEnabled')"
              />
            </div>
          </template>

          <template #cell-group="{ row }">
            <div class="group/dropdown relative">
              <button
                :ref="(el) => setGroupButtonRef(row.id, el)"
                @click="openGroupSelector(row)"
                class="-mx-2 -my-1 flex cursor-pointer items-center gap-2 rounded-lg px-2 py-1 transition-all duration-200 hover:bg-gray-100 dark:hover:bg-dark-700"
                :title="t('keys.clickToChangeGroup')"
              >
                <GroupBadge
                  v-if="row.group"
                  :name="row.group.name"
                  :platform="row.group.platform"
                  :subscription-type="row.group.subscription_type"
                  :rate-multiplier="row.group.rate_multiplier"
                />
                <span v-else class="text-sm text-gray-400 dark:text-dark-500">{{
                  t('keys.noGroup')
                }}</span>
                <svg
                  class="h-3.5 w-3.5 text-gray-400 opacity-0 transition-opacity group-hover/dropdown:opacity-100"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  stroke-width="2"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M8.25 15L12 18.75 15.75 15m-7.5-6L12 5.25 15.75 9"
                  />
                </svg>
              </button>
            </div>
          </template>

          <template #cell-usage="{ row }">
            <div class="text-sm">
              <div class="flex items-center gap-1.5">
                <span class="text-gray-500 dark:text-gray-400">{{ t('keys.today') }}:</span>
                <span class="font-medium text-gray-900 dark:text-white">
                  ${{ (usageStats[row.id]?.today_actual_cost ?? 0).toFixed(4) }}
                </span>
              </div>
              <div class="mt-0.5 flex items-center gap-1.5">
                <span class="text-gray-500 dark:text-gray-400">{{ t('keys.total') }}:</span>
                <span class="font-medium text-gray-900 dark:text-white">
                  ${{ (usageStats[row.id]?.total_actual_cost ?? 0).toFixed(4) }}
                </span>
              </div>
            </div>
          </template>

          <template #cell-status="{ value }">
            <span :class="['badge', value === 'active' ? 'badge-success' : 'badge-gray']">
              {{ t('admin.accounts.status.' + value) }}
            </span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <!-- Use Key Button -->
              <button
                @click="openUseKeyModal(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-green-50 hover:text-green-600 dark:hover:bg-green-900/20 dark:hover:text-green-400"
              >
                <Icon name="terminal" size="sm" />
                <span class="text-xs">{{ t('keys.useKey') }}</span>
              </button>
              <!-- Import to CC Switch Button -->
              <button
                v-if="!publicSettings?.hide_ccs_import_button"
                @click="importToCcswitch(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400"
              >
                <Icon name="upload" size="sm" />
                <span class="text-xs">{{ t('keys.importToCcSwitch') }}</span>
              </button>
              <!-- Toggle Status Button -->
              <button
                @click="toggleKeyStatus(row)"
                :class="[
                  'flex flex-col items-center gap-0.5 rounded-lg p-1.5 transition-colors',
                  row.status === 'active'
                    ? 'text-gray-500 hover:bg-yellow-50 hover:text-yellow-600 dark:hover:bg-yellow-900/20 dark:hover:text-yellow-400'
                    : 'text-gray-500 hover:bg-green-50 hover:text-green-600 dark:hover:bg-green-900/20 dark:hover:text-green-400'
                ]"
              >
                <Icon v-if="row.status === 'active'" name="ban" size="sm" />
                <Icon v-else name="checkCircle" size="sm" />
                <span class="text-xs">{{ row.status === 'active' ? t('keys.disable') : t('keys.enable') }}</span>
              </button>
              <!-- Edit Button -->
              <button
                @click="editKey(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
              >
                <Icon name="edit" size="sm" />
                <span class="text-xs">{{ t('common.edit') }}</span>
              </button>
              <!-- Delete Button -->
              <button
                @click="confirmDelete(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
              >
                <Icon name="trash" size="sm" />
                <span class="text-xs">{{ t('common.delete') }}</span>
              </button>
            </div>
          </template>

          <!-- Row Expand Trigger: Click to expand models -->
          <template #row-expand-trigger="{ row, expanded }">
            <button
              v-if="row.group"
              @click="toggleModelsExpand(row)"
              class="px-3 py-1 flex items-center gap-4 text-xs text-gray-400 dark:text-dark-500 hover:bg-gray-100 dark:hover:bg-dark-700 transition-colors cursor-pointer"
            >
              <Icon
                :name="expanded ? 'chevronDown' : 'chevronRight'"
                size="xs"
                :class="expanded ? 'text-primary-500' : 'text-gray-400'"
              />
              <span>{{ expanded ? t('keys.hideModels') : t('keys.expandModels') }}</span>
              <span class="text-gray-300 dark:text-dark-600">{{ getModelsCountText(row.id) }}</span>
              <span v-if="getEstimatedQuota(row)" class="text-gray-500 dark:text-dark-400">
                {{ t('keys.estimatedQuota') }}: {{ getEstimatedQuota(row) }}
              </span>
            </button>
          </template>

          <!-- Row Expand: Models List -->
          <template #row-expand="{ row }">
            <div class="px-4 py-3 bg-gray-50 dark:bg-dark-800">
              <!-- Loading State -->
              <div v-if="isExpandedLoading(row.id)" class="flex items-center gap-2 text-sm text-gray-500">
                <Icon name="refresh" size="sm" class="animate-spin" />
                <span>{{ t('keys.modelsLoading') }}</span>
              </div>

              <!-- Error State -->
              <div v-else-if="getExpandedError(row.id)" class="flex items-center justify-between">
                <span class="text-sm text-red-500">{{ getExpandedError(row.id) }}</span>
                <button
                  @click="retryFetchModelsForExpand(row)"
                  class="text-sm text-primary-600 hover:underline"
                >
                  {{ t('keys.modelsRetry') }}
                </button>
              </div>

              <!-- Models Content -->
              <div v-else-if="getExpandedModels(row.id)" class="w-full space-y-2">
                <!-- Header with source info -->
                <div class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                  <span class="font-medium">{{ t('keys.availableModels') }}</span>
                  <span
                    class="px-1.5 py-0.5 rounded text-[10px] font-medium"
                    :class="getExpandedModels(row.id)?.source === 'mapping'
                      ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
                      : getExpandedModels(row.id)?.source === 'unlimited'
                        ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                        : 'bg-gray-100 text-gray-600 dark:bg-dark-600 dark:text-gray-400'"
                  >
                    {{
                      getExpandedModels(row.id)?.source === 'mapping'
                        ? t('keys.modelsSourceMapping')
                        : getExpandedModels(row.id)?.source === 'unlimited'
                          ? t('keys.modelsSourceUnlimited')
                          : t('keys.modelsSourceDefault')
                    }}
                  </span>
                </div>

                <!-- Unlimited State -->
                <div v-if="getExpandedModels(row.id)?.source === 'unlimited'" class="flex items-center gap-2 py-2">
                  <Icon name="checkCircle" size="sm" class="text-green-500" />
                  <span class="text-sm text-green-600 dark:text-green-400">{{ t('keys.modelsUnlimited') }}</span>
                </div>

                <!-- No Accounts State -->
                <div v-else-if="getExpandedModels(row.id)?.source === 'no_accounts'" class="flex items-center gap-2 py-2">
                  <Icon name="xCircle" size="sm" class="text-red-500" />
                  <span class="text-sm text-red-600 dark:text-red-400">{{ t('keys.modelsNoAccounts') }}</span>
                </div>

                <!-- All Paused State -->
                <div v-else-if="getExpandedModels(row.id)?.source === 'all_paused'" class="flex items-center gap-2 py-2">
                  <Icon name="ban" size="sm" class="text-yellow-500" />
                  <span class="text-sm text-yellow-600 dark:text-yellow-400">{{ t('keys.modelsAllPaused') }}</span>
                </div>

                <!-- Models List -->
                <div v-else-if="getExpandedModels(row.id)?.models?.length" class="w-full">
                  <!-- Mobile: Compact scrollable list -->
                  <div class="md:hidden w-full max-h-32 overflow-y-auto rounded-lg bg-white dark:bg-dark-700 border border-gray-200 dark:border-dark-600 p-2">
                    <div class="flex flex-wrap gap-1">
                      <span
                        v-for="(model, idx) in getExpandedModels(row.id)?.models"
                        :key="model"
                        @click="copyModelName(model)"
                        class="text-xs text-gray-600 dark:text-gray-300 cursor-pointer hover:text-primary-600 dark:hover:text-primary-400"
                        :class="{ 'text-green-500 dark:text-green-400': copiedModel === model }"
                      >{{ model }}{{ idx < (getExpandedModels(row.id)?.models?.length || 0) - 1 ? ',' : '' }}</span>
                    </div>
                  </div>
                  <!-- Desktop: Compact tag style with scroll -->
                  <div class="hidden md:block w-full max-h-28 overflow-y-auto rounded-lg border border-gray-200 dark:border-dark-600 bg-white/50 dark:bg-dark-700/50 p-2">
                    <div class="flex flex-wrap gap-1">
                      <span
                        v-for="model in getExpandedModels(row.id)?.models"
                        :key="model"
                        @click="copyModelName(model)"
                        class="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-xs font-mono cursor-pointer transition-colors bg-gray-100 dark:bg-dark-600 text-gray-700 dark:text-gray-300 hover:bg-primary-100 hover:text-primary-700 dark:hover:bg-primary-900/30 dark:hover:text-primary-400"
                        :class="{ '!bg-green-100 !text-green-700 dark:!bg-green-900/30 dark:!text-green-400': copiedModel === model }"
                        :title="t('keys.clickToCopyModel')"
                      >
                        {{ model }}
                        <Icon v-if="copiedModel === model" name="check" size="xs" />
                      </span>
                    </div>
                  </div>
                </div>

                <!-- Empty State -->
                <div v-else class="text-sm text-gray-500 dark:text-gray-400 py-2">
                  {{ t('keys.modelsEmpty') }}
                </div>
              </div>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('keys.noKeysYet')"
              :description="t('keys.createFirstKey')"
              :action-text="t('keys.createKey')"
              @action="showCreateModal = true"
            />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <!-- Create/Edit Modal -->
    <BaseDialog
      :show="showCreateModal || showEditModal"
      :title="showEditModal ? t('keys.editKey') : t('keys.createKey')"
      width="normal"
      @close="closeModals"
    >
      <form id="key-form" @submit.prevent="handleSubmit" class="space-y-5">
        <div>
          <label class="input-label">{{ t('keys.nameLabel') }}</label>
          <input
            v-model="formData.name"
            type="text"
            required
            class="input"
            :placeholder="t('keys.namePlaceholder')"
            data-tour="key-form-name"
          />
        </div>

        <div>
          <label class="input-label">{{ t('keys.groupLabel') }}</label>
          <Select
            v-model="formData.group_id"
            :options="groupOptions"
            :placeholder="t('keys.selectGroup')"
            data-tour="key-form-group"
          >
            <template #selected="{ option }">
              <GroupBadge
                v-if="option"
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
              />
              <span v-else class="text-gray-400">{{ t('keys.selectGroup') }}</span>
            </template>
            <template #option="{ option, selected }">
              <GroupOptionItem
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
                :description="(option as unknown as GroupOption).description"
                :selected="selected"
              />
            </template>
          </Select>
        </div>

        <!-- Available Models Panel -->
        <div
          v-if="formData.group_id !== null"
          class="rounded-lg border border-gray-200 bg-gray-50/60 p-3 dark:border-dark-600 dark:bg-dark-800/40"
        >
          <div class="flex items-center justify-between">
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">
              {{ t('keys.availableModels') }}
            </span>
            <span v-if="selectedGroupModelSource" class="text-xs text-gray-500 dark:text-dark-400">
              {{
                selectedGroupModelSource === 'mapping'
                  ? t('keys.modelsSourceMapping')
                  : t('keys.modelsSourceDefault')
              }}
            </span>
          </div>

          <div
            v-if="groupModelsLoading"
            class="mt-2 flex items-center gap-2 text-xs text-gray-500 dark:text-dark-400"
          >
            <Icon name="refresh" size="sm" class="animate-spin" />
            <span>{{ t('keys.modelsLoading') }}</span>
          </div>

          <div
            v-else-if="groupModelsError"
            class="mt-2 flex items-center justify-between text-xs text-red-600 dark:text-red-400"
          >
            <span>{{ groupModelsError }}</span>
            <button
              type="button"
              class="text-xs font-medium text-primary-600 hover:text-primary-700"
              @click="refreshGroupModels"
            >
              {{ t('keys.modelsRetry') }}
            </button>
          </div>

          <div v-else-if="selectedGroupModels.length === 0" class="mt-2 text-xs text-gray-500 dark:text-dark-400">
            {{ t('keys.modelsEmpty') }}
          </div>

          <div v-else class="mt-2 flex max-h-28 flex-wrap gap-1.5 overflow-y-auto">
            <span v-for="model in selectedGroupModels" :key="model" class="badge badge-gray text-xs">
              {{ model }}
            </span>
          </div>
        </div>

        <!-- Custom Key Section (only for create) -->
        <div v-if="!showEditModal" class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.customKeyLabel') }}</label>
            <button
              type="button"
              @click="formData.use_custom_key = !formData.use_custom_key"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.use_custom_key ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  formData.use_custom_key ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>
          <div v-if="formData.use_custom_key">
            <input
              v-model="formData.custom_key"
              type="text"
              class="input font-mono"
              :placeholder="t('keys.customKeyPlaceholder')"
              :class="{ 'border-red-500 dark:border-red-500': customKeyError }"
            />
            <p v-if="customKeyError" class="mt-1 text-sm text-red-500">{{ customKeyError }}</p>
            <p v-else class="input-hint">{{ t('keys.customKeyHint') }}</p>
          </div>
        </div>

        <div v-if="showEditModal">
          <label class="input-label">{{ t('keys.statusLabel') }}</label>
          <Select
            v-model="formData.status"
            :options="statusOptions"
            :placeholder="t('keys.selectStatus')"
          />
        </div>

        <!-- IP Restriction Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.ipRestriction') }}</label>
            <button
              type="button"
              @click="formData.enable_ip_restriction = !formData.enable_ip_restriction"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_ip_restriction ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_ip_restriction ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="formData.enable_ip_restriction" class="space-y-4 pt-2">
            <div>
              <label class="input-label">{{ t('keys.ipWhitelist') }}</label>
              <textarea
                v-model="formData.ip_whitelist"
                rows="3"
                class="input font-mono text-sm"
                :placeholder="t('keys.ipWhitelistPlaceholder')"
              />
              <p class="input-hint">{{ t('keys.ipWhitelistHint') }}</p>
            </div>

            <div>
              <label class="input-label">{{ t('keys.ipBlacklist') }}</label>
              <textarea
                v-model="formData.ip_blacklist"
                rows="3"
                class="input font-mono text-sm"
                :placeholder="t('keys.ipBlacklistPlaceholder')"
              />
              <p class="input-hint">{{ t('keys.ipBlacklistHint') }}</p>
            </div>
          </div>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button @click="closeModals" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button
            form="key-form"
            type="submit"
            :disabled="submitting"
            class="btn btn-primary"
            data-tour="key-form-submit"
          >
            <svg
              v-if="submitting"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{
              submitting
                ? t('keys.saving')
                : showEditModal
                  ? t('common.update')
                  : t('common.create')
            }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Delete Confirmation Dialog -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('keys.deleteKey')"
      :message="t('keys.deleteConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="handleDelete"
      @cancel="showDeleteDialog = false"
    />

    <!-- Use Key Modal -->
    <UseKeyModal
      :show="showUseKeyModal"
      :api-key="selectedKey?.key || ''"
      :base-url="publicSettings?.api_base_url || ''"
      :platform="selectedKey?.group?.platform || null"
      @close="closeUseKeyModal"
    />

    <!-- View Models Modal -->
    <BaseDialog
      :show="showModelsModal"
      :title="t('keys.availableModels')"
      width="narrow"
      @close="closeModelsModal"
    >
      <div class="space-y-4">
        <!-- Group Info -->
        <div v-if="modelsModalKey?.group" class="flex items-center gap-2">
          <GroupBadge
            :name="modelsModalKey.group.name"
            :platform="modelsModalKey.group.platform"
            :subscription-type="modelsModalKey.group.subscription_type"
            :rate-multiplier="modelsModalKey.group.rate_multiplier"
          />
          <span
            v-if="modelsModalData"
            class="text-xs px-2 py-0.5 rounded-full"
            :class="modelsModalData.source === 'mapping'
              ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
              : modelsModalData.source === 'unlimited'
                ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                : 'bg-gray-100 text-gray-600 dark:bg-dark-600 dark:text-gray-400'"
          >
            {{ modelsModalData.source === 'mapping' ? t('keys.modelsSourceMapping') : modelsModalData.source === 'unlimited' ? t('keys.modelsSourceUnlimited') : t('keys.modelsSourceDefault') }}
          </span>
        </div>

        <!-- Loading State -->
        <div v-if="modelsModalLoading" class="flex items-center justify-center py-8">
          <Icon name="refresh" size="lg" class="animate-spin text-gray-400" />
        </div>

        <!-- Error State -->
        <div v-else-if="modelsModalError" class="text-center py-6">
          <p class="text-sm text-red-500 dark:text-red-400">{{ modelsModalError }}</p>
          <button @click="fetchModelsForModal" class="mt-2 text-sm text-primary-600 hover:underline">
            {{ t('keys.modelsRetry') }}
          </button>
        </div>

        <!-- Models List -->
        <div v-else-if="modelsModalData?.models?.length" class="max-h-80 overflow-y-auto">
          <div class="grid gap-1.5">
            <button
              v-for="model in modelsModalData.models"
              :key="model"
              @click="copyModelName(model)"
              class="group flex items-center justify-between px-3 py-2 rounded-lg bg-gray-50 dark:bg-dark-700 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-colors cursor-pointer text-left w-full"
              :title="t('keys.clickToCopyModel')"
            >
              <div class="flex items-center gap-2 min-w-0">
                <Icon name="cube" size="sm" class="text-gray-400 group-hover:text-primary-500 flex-shrink-0" />
                <code class="text-sm text-gray-700 dark:text-gray-300 group-hover:text-primary-600 dark:group-hover:text-primary-400 truncate">{{ model }}</code>
              </div>
              <Icon
                v-if="copiedModel === model"
                name="check"
                size="sm"
                class="text-green-500 flex-shrink-0"
              />
              <Icon
                v-else
                name="clipboard"
                size="sm"
                class="text-gray-400 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0"
              />
            </button>
          </div>
        </div>

        <!-- Unlimited State -->
        <div v-else-if="modelsModalData?.source === 'unlimited'" class="text-center py-6">
          <Icon name="checkCircle" size="xl" class="text-green-500 mx-auto mb-2" />
          <p class="text-sm text-green-600 dark:text-green-400 font-medium">{{ t('keys.modelsUnlimited') }}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">{{ t('keys.modelsUnlimitedDesc') }}</p>
        </div>

        <!-- No Accounts State -->
        <div v-else-if="modelsModalData?.source === 'no_accounts'" class="text-center py-6">
          <Icon name="xCircle" size="xl" class="text-red-500 mx-auto mb-2" />
          <p class="text-sm text-red-600 dark:text-red-400 font-medium">{{ t('keys.modelsNoAccounts') }}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">{{ t('keys.modelsNoAccountsDesc') }}</p>
        </div>

        <!-- All Paused State -->
        <div v-else-if="modelsModalData?.source === 'all_paused'" class="text-center py-6">
          <Icon name="ban" size="xl" class="text-yellow-500 mx-auto mb-2" />
          <p class="text-sm text-yellow-600 dark:text-yellow-400 font-medium">{{ t('keys.modelsAllPaused') }}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">{{ t('keys.modelsAllPausedDesc') }}</p>
        </div>

        <!-- Empty State -->
        <div v-else class="text-center py-6">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('keys.modelsEmpty') }}</p>
        </div>
      </div>

      <template #footer>
        <button @click="closeModelsModal" class="btn btn-secondary">
          {{ t('common.close') }}
        </button>
      </template>
    </BaseDialog>

    <!-- CCS Client Selection Dialog for Antigravity -->
    <BaseDialog
      :show="showCcsClientSelect"
      :title="t('keys.ccsClientSelect.title')"
      width="narrow"
      @close="closeCcsClientSelect"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-400">
          {{ t('keys.ccsClientSelect.description') }}
	        </p>
	        <div class="grid grid-cols-2 gap-3">
	          <button
	            @click="handleCcsClientSelect('claude')"
	            class="flex flex-col items-center gap-2 p-4 rounded-xl border-2 border-gray-200 dark:border-dark-600 hover:border-primary-500 dark:hover:border-primary-500 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-all"
	          >
	            <Icon name="terminal" size="xl" class="text-gray-600 dark:text-gray-400" />
	            <span class="font-medium text-gray-900 dark:text-white">{{
	              t('keys.ccsClientSelect.claudeCode')
	            }}</span>
	            <span class="text-xs text-gray-500 dark:text-gray-400">{{
	              t('keys.ccsClientSelect.claudeCodeDesc')
	            }}</span>
	          </button>
	          <button
	            @click="handleCcsClientSelect('gemini')"
	            class="flex flex-col items-center gap-2 p-4 rounded-xl border-2 border-gray-200 dark:border-dark-600 hover:border-primary-500 dark:hover:border-primary-500 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-all"
	          >
	            <Icon name="sparkles" size="xl" class="text-gray-600 dark:text-gray-400" />
	            <span class="font-medium text-gray-900 dark:text-white">{{
	              t('keys.ccsClientSelect.geminiCli')
	            }}</span>
	            <span class="text-xs text-gray-500 dark:text-gray-400">{{
	              t('keys.ccsClientSelect.geminiCliDesc')
	            }}</span>
	          </button>
	        </div>
	      </div>
      <template #footer>
        <div class="flex justify-end">
          <button @click="closeCcsClientSelect" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Group Selector Dropdown (Teleported to body to avoid overflow clipping) -->
    <Teleport to="body">
      <div
        v-if="groupSelectorKeyId !== null && dropdownPosition"
        ref="dropdownRef"
        class="animate-in fade-in slide-in-from-top-2 fixed z-[100000020] w-64 overflow-hidden rounded-xl bg-white shadow-lg ring-1 ring-black/5 duration-200 dark:bg-dark-800 dark:ring-white/10"
        style="pointer-events: auto !important;"
        :style="{ top: dropdownPosition.top + 'px', left: dropdownPosition.left + 'px' }"
      >
        <div class="max-h-64 overflow-y-auto p-1.5">
          <button
            v-for="option in groupOptions"
            :key="option.value ?? 'null'"
            @click="changeGroup(selectedKeyForGroup!, option.value)"
            :class="[
              'flex w-full items-center justify-between rounded-lg px-3 py-2 text-sm transition-colors',
              selectedKeyForGroup?.group_id === option.value ||
              (!selectedKeyForGroup?.group_id && option.value === null)
                ? 'bg-primary-50 dark:bg-primary-900/20'
                : 'hover:bg-gray-100 dark:hover:bg-dark-700'
            ]"
            :title="option.description || undefined"
          >
            <GroupOptionItem
              :name="option.label"
              :platform="option.platform"
              :subscription-type="option.subscriptionType"
              :rate-multiplier="option.rate"
              :description="option.description"
              :selected="
                selectedKeyForGroup?.group_id === option.value ||
                (!selectedKeyForGroup?.group_id && option.value === null)
              "
            />
          </button>
        </div>
      </div>
    </Teleport>
  </AppLayout>
</template>

<script setup lang="ts">
	import { ref, computed, onMounted, onUnmounted, watch, nextTick, type ComponentPublicInstance } from 'vue'
	import { useI18n } from 'vue-i18n'
	import { useAppStore } from '@/stores/app'
	import { useAuthStore } from '@/stores/auth'
	import { useOnboardingStore } from '@/stores/onboarding'
	import { useClipboard } from '@/composables/useClipboard'

const { t } = useI18n()
import { keysAPI, authAPI, usageAPI, userGroupsAPI } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
	import DataTable from '@/components/common/DataTable.vue'
	import Pagination from '@/components/common/Pagination.vue'
	import BaseDialog from '@/components/common/BaseDialog.vue'
	import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
	import EmptyState from '@/components/common/EmptyState.vue'
	import Select from '@/components/common/Select.vue'
	import Icon from '@/components/icons/Icon.vue'
	import UseKeyModal from '@/components/keys/UseKeyModal.vue'
	import GroupBadge from '@/components/common/GroupBadge.vue'
	import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
	import type { ApiKey, Group, PublicSettings, SubscriptionType, GroupPlatform, GroupModelsResponse } from '@/types'
import type { Column } from '@/components/common/types'
import type { BatchApiKeyUsageStats } from '@/api/usage'

interface GroupOption {
  value: number
  label: string
  description: string | null
  rate: number
  subscriptionType: SubscriptionType
  platform: GroupPlatform
}

const appStore = useAppStore()
const authStore = useAuthStore()
const onboardingStore = useOnboardingStore()
const { copyToClipboard: clipboardCopy } = useClipboard()

const columns = computed<Column[]>(() => [
  { key: 'name', label: t('common.name'), sortable: true },
  { key: 'key', label: t('keys.apiKey'), sortable: false },
  { key: 'group', label: t('keys.group'), sortable: false },
  { key: 'usage', label: t('keys.usage'), sortable: false },
  { key: 'status', label: t('common.status'), sortable: true },
  { key: 'actions', label: t('common.actions'), sortable: false }
])

const apiKeys = ref<ApiKey[]>([])
const groups = ref<Group[]>([])
const loading = ref(false)
const submitting = ref(false)
const usageStats = ref<Record<string, BatchApiKeyUsageStats>>({})

const pagination = ref({
  page: 1,
  page_size: 10,
  total: 0,
  pages: 0
})

const showCreateModal = ref(false)
const showEditModal = ref(false)
const showDeleteDialog = ref(false)
const showUseKeyModal = ref(false)
const showCcsClientSelect = ref(false)
const showModelsModal = ref(false)
const pendingCcsRow = ref<ApiKey | null>(null)
const selectedKey = ref<ApiKey | null>(null)
const modelsModalKey = ref<ApiKey | null>(null)
const modelsModalData = ref<GroupModelsResponse | null>(null)
const modelsModalLoading = ref(false)
const modelsModalError = ref('')
const copiedKeyId = ref<number | null>(null)
const copiedModel = ref<string | null>(null)
const groupSelectorKeyId = ref<number | null>(null)
const publicSettings = ref<PublicSettings | null>(null)
// 展开行状态 - 用于显示模型列表
const expandedKeyIds = ref<number[]>([])
// 展开行的模型数据缓存
const expandedModelsData = ref<Map<number, GroupModelsResponse>>(new Map())
const expandedModelsLoading = ref<Set<number>>(new Set())
const expandedModelsError = ref<Map<number, string>>(new Map())
// 分组可用模型相关状态
const groupModelsCache = ref<Map<number, GroupModelsResponse>>(new Map())
const groupModelsLoading = ref(false)
const groupModelsError = ref('')
const groupModelsRequestId = ref(0)
const dropdownRef = ref<HTMLElement | null>(null)
const dropdownPosition = ref<{ top: number; left: number } | null>(null)
const groupButtonRefs = ref<Map<number, HTMLElement>>(new Map())
let abortController: AbortController | null = null

// Get the currently selected key for group change
const selectedKeyForGroup = computed(() => {
  if (groupSelectorKeyId.value === null) return null
  return apiKeys.value.find((k) => k.id === groupSelectorKeyId.value) || null
})

const setGroupButtonRef = (keyId: number, el: Element | ComponentPublicInstance | null) => {
  if (el instanceof HTMLElement) {
    groupButtonRefs.value.set(keyId, el)
  } else {
    groupButtonRefs.value.delete(keyId)
  }
}

const formData = ref({
  name: '',
  group_id: null as number | null,
  status: 'active' as 'active' | 'inactive',
  use_custom_key: false,
  custom_key: '',
  enable_ip_restriction: false,
  ip_whitelist: '',
  ip_blacklist: ''
})

// 自定义Key验证
const customKeyError = computed(() => {
  if (!formData.value.use_custom_key || !formData.value.custom_key) {
    return ''
  }
  const key = formData.value.custom_key
  if (key.length < 16) {
    return t('keys.customKeyTooShort')
  }
  // 检查字符：只允许字母、数字、下划线、连字符
  if (!/^[a-zA-Z0-9_-]+$/.test(key)) {
    return t('keys.customKeyInvalidChars')
  }
  return ''
})

const statusOptions = computed(() => [
  { value: 'active', label: t('common.active') },
  { value: 'inactive', label: t('common.inactive') }
])

// Convert groups to Select options format with rate multiplier and subscription type
const groupOptions = computed(() =>
  groups.value.map((group) => ({
    value: group.id,
    label: group.name,
    description: group.description,
    rate: group.rate_multiplier,
    subscriptionType: group.subscription_type,
    platform: group.platform
  }))
)

// 分组可用模型计算属性
const selectedGroupModelInfo = computed(() => {
  const groupId = formData.value.group_id
  if (groupId === null) return null
  return groupModelsCache.value.get(groupId) || null
})

const selectedGroupModels = computed(() => selectedGroupModelInfo.value?.models ?? [])
const selectedGroupModelSource = computed(() => selectedGroupModelInfo.value?.source ?? null)

const maskKey = (key: string): string => {
  if (key.length <= 12) return key
  return `${key.slice(0, 8)}...${key.slice(-4)}`
}

const copyToClipboard = async (text: string, keyId: number) => {
  const success = await clipboardCopy(text, t('keys.copied'))
  if (success) {
    copiedKeyId.value = keyId
    setTimeout(() => {
      copiedKeyId.value = null
    }, 800)
  }
}

const copyModelName = async (model: string) => {
  const success = await clipboardCopy(model, t('keys.modelCopied'))
  if (success) {
    copiedModel.value = model
    setTimeout(() => {
      copiedModel.value = null
    }, 800)
  }
}

const isAbortError = (error: unknown) => {
  if (!error || typeof error !== 'object') return false
  const { name, code } = error as { name?: string; code?: string }
  return name === 'AbortError' || code === 'ERR_CANCELED'
}

const loadApiKeys = async () => {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  const { signal } = controller
  loading.value = true
  try {
    const response = await keysAPI.list(pagination.value.page, pagination.value.page_size, {
      signal
    })
    if (signal.aborted) return
    apiKeys.value = response.items
    pagination.value.total = response.total
    pagination.value.pages = response.pages

    // Load usage stats for all API keys in the list
    if (response.items.length > 0) {
      const keyIds = response.items.map((k) => k.id)
      try {
        const usageResponse = await usageAPI.getDashboardApiKeysUsage(keyIds, { signal })
        if (signal.aborted) return
        usageStats.value = usageResponse.stats
      } catch (e) {
        if (!isAbortError(e)) {
          console.error('Failed to load usage stats:', e)
        }
      }
    }
  } catch (error) {
    if (isAbortError(error)) {
      return
    }
    appStore.showError(t('keys.failedToLoad'))
  } finally {
    if (abortController === controller) {
      loading.value = false
    }
  }
}

const loadGroups = async () => {
  try {
    groups.value = await userGroupsAPI.getAvailable()
  } catch (error) {
    console.error('Failed to load groups:', error)
    appStore.showError(t('keys.failedToLoadGroups'))
  }
}

const loadPublicSettings = async () => {
  try {
    publicSettings.value = await authAPI.getPublicSettings()
  } catch (error) {
    console.error('Failed to load public settings:', error)
  }
}

// 获取分组可用模型
const fetchGroupModels = async (groupId: number, requestId: number) => {
  if (groupModelsCache.value.has(groupId)) return
  groupModelsLoading.value = true
  groupModelsError.value = ''
  try {
    const data = await userGroupsAPI.getAvailableModels(groupId)
    if (requestId !== groupModelsRequestId.value) return
    groupModelsCache.value.set(groupId, data)
  } catch (error) {
    if (requestId !== groupModelsRequestId.value) return
    groupModelsError.value = t('keys.modelsLoadFailed')
  } finally {
    if (requestId === groupModelsRequestId.value) {
      groupModelsLoading.value = false
    }
  }
}

const refreshGroupModels = async () => {
  if (formData.value.group_id === null) return
  const groupId = formData.value.group_id
  groupModelsCache.value.delete(groupId)
  groupModelsError.value = ''
  groupModelsRequestId.value += 1
  await fetchGroupModels(groupId, groupModelsRequestId.value)
}

// 监听分组变化，自动获取模型
watch(
  () => formData.value.group_id,
  (groupId) => {
    groupModelsRequestId.value += 1
    groupModelsError.value = ''
    groupModelsLoading.value = false
    if (groupId === null) return
    fetchGroupModels(groupId, groupModelsRequestId.value)
  }
)

const openUseKeyModal = (key: ApiKey) => {
  selectedKey.value = key
  showUseKeyModal.value = true
}

const closeUseKeyModal = () => {
  showUseKeyModal.value = false
  selectedKey.value = null
}

// Models Modal methods (closeModelsModal still used by modal template)

const closeModelsModal = () => {
  showModelsModal.value = false
  modelsModalKey.value = null
  modelsModalData.value = null
  modelsModalError.value = ''
}

const fetchModelsForModal = async () => {
  if (!modelsModalKey.value?.group?.id) return

  const groupId = modelsModalKey.value.group.id

  // Always fetch fresh data for modal (no cache)
  modelsModalLoading.value = true
  modelsModalError.value = ''

  try {
    const data = await userGroupsAPI.getAvailableModels(groupId)
    groupModelsCache.value.set(groupId, data)
    modelsModalData.value = data
  } catch (err: any) {
    modelsModalError.value = err.message || t('keys.modelsLoadFailed')
  } finally {
    modelsModalLoading.value = false
  }
}

// 展开行方法 - 切换模型列表展开/收起
const toggleModelsExpand = (key: ApiKey) => {
  const keyId = key.id
  const index = expandedKeyIds.value.indexOf(keyId)

  // 保存当前滚动位置，防止展开/收起时页面自动滚动
  const tableWrapper = document.querySelector('.table-wrapper') as HTMLElement | null
  const scrollLeft = tableWrapper?.scrollLeft ?? 0
  const scrollTop = window.scrollY // 保存垂直滚动位置（移动端）

  // 恢复滚动位置的辅助函数
  const restoreScrollPosition = () => {
    if (tableWrapper) {
      tableWrapper.scrollLeft = scrollLeft
    }
    window.scrollTo({ top: scrollTop, behavior: 'instant' })
  }

  if (index > -1) {
    // 收起
    expandedKeyIds.value.splice(index, 1)
  } else {
    // 展开并加载数据
    expandedKeyIds.value.push(keyId)
    if (key.group?.id && !expandedModelsData.value.has(keyId)) {
      fetchModelsForExpand(key)
    }
    // 刷新用户数据以获取最新的订阅使用量（用于计算预计可用额度）
    authStore.refreshUser().catch(() => {
      // 静默处理刷新失败，不影响主流程
    })
  }

  // 多次恢复滚动位置，确保在各种时机都能正确恢复
  // 1. DOM 更新后立即恢复
  nextTick(restoreScrollPosition)
  // 2. 渲染帧后恢复
  requestAnimationFrame(restoreScrollPosition)
  // 3. 延迟恢复，处理异步渲染
  setTimeout(restoreScrollPosition, 50)
  setTimeout(restoreScrollPosition, 100)
}

// 获取展开行的模型数据
const fetchModelsForExpand = async (key: ApiKey) => {
  if (!key.group?.id) return

  const keyId = key.id
  const groupId = key.group.id

  expandedModelsLoading.value.add(keyId)
  expandedModelsError.value.delete(keyId)

  try {
    const data = await userGroupsAPI.getAvailableModels(groupId)
    expandedModelsData.value.set(keyId, data)
  } catch (err: any) {
    expandedModelsError.value.set(keyId, err.message || t('keys.modelsLoadFailed'))
  } finally {
    expandedModelsLoading.value.delete(keyId)
  }
}

// 重试加载展开行的模型
const retryFetchModelsForExpand = (key: ApiKey) => {
  fetchModelsForExpand(key)
}

// 获取展开行的模型数据
const getExpandedModels = (keyId: number) => expandedModelsData.value.get(keyId)

// 检查展开行是否正在加载
const isExpandedLoading = (keyId: number) => expandedModelsLoading.value.has(keyId)

// 获取展开行的错误信息
const getExpandedError = (keyId: number) => expandedModelsError.value.get(keyId)

// 获取模型数量文本
const getModelsCountText = (keyId: number) => {
  const data = expandedModelsData.value.get(keyId)
  if (!data) return ''
  if (data.source === 'unlimited') return t('keys.modelsUnlimitedShort')
  if (data.source === 'no_accounts' || data.source === 'all_paused') return ''
  const count = data.models?.length ?? 0
  return t('keys.modelsCount', { count })
}

// 计算预计可用额度
const getEstimatedQuota = (row: ApiKey): string | null => {
  const group = row.group
  if (!group) return null

  const rateMultiplier = group.rate_multiplier || 1
  const user = authStore.user

  // 订阅类型分组：从用户订阅中获取剩余额度
  if (group.subscription_type === 'subscription') {
    const subscription = user?.subscriptions?.find(s => s.group_id === group.id && s.status === 'active')
    if (!subscription) return null

    // 计算最小剩余额度（日/周/月限制中取最小值）
    let minRemaining: number | null = null

    if (group.daily_limit_usd != null) {
      const remaining = group.daily_limit_usd - (subscription.daily_usage_usd || 0)
      if (minRemaining === null || remaining < minRemaining) {
        minRemaining = remaining
      }
    }
    if (group.weekly_limit_usd != null) {
      const remaining = group.weekly_limit_usd - (subscription.weekly_usage_usd || 0)
      if (minRemaining === null || remaining < minRemaining) {
        minRemaining = remaining
      }
    }
    if (group.monthly_limit_usd != null) {
      const remaining = group.monthly_limit_usd - (subscription.monthly_usage_usd || 0)
      if (minRemaining === null || remaining < minRemaining) {
        minRemaining = remaining
      }
    }

    if (minRemaining === null) return null
    const quota = Math.max(0, minRemaining) / rateMultiplier
    return `$${quota.toFixed(2)}`
  }

  // 标准类型分组：从用户余额计算（balance 为 null 时视为 0）
  const balance = user?.balance ?? 0
  const quota = balance / rateMultiplier
  return `$${quota.toFixed(2)}`
}

const handlePageChange = (page: number) => {
  pagination.value.page = page
  loadApiKeys()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.page_size = pageSize
  pagination.value.page = 1
  loadApiKeys()
}

const editKey = (key: ApiKey) => {
  selectedKey.value = key
  const hasIPRestriction = (key.ip_whitelist?.length > 0) || (key.ip_blacklist?.length > 0)
  formData.value = {
    name: key.name,
    group_id: key.group_id,
    status: key.status,
    use_custom_key: false,
    custom_key: '',
    enable_ip_restriction: hasIPRestriction,
    ip_whitelist: (key.ip_whitelist || []).join('\n'),
    ip_blacklist: (key.ip_blacklist || []).join('\n')
  }
  showEditModal.value = true
}

const toggleKeyStatus = async (key: ApiKey) => {
  const newStatus = key.status === 'active' ? 'inactive' : 'active'
  try {
    await keysAPI.toggleStatus(key.id, newStatus)
    appStore.showSuccess(
      newStatus === 'active' ? t('keys.keyEnabledSuccess') : t('keys.keyDisabledSuccess')
    )
    loadApiKeys()
  } catch (error) {
    appStore.showError(t('keys.failedToUpdateStatus'))
  }
}

const openGroupSelector = (key: ApiKey) => {
  if (groupSelectorKeyId.value === key.id) {
    groupSelectorKeyId.value = null
    dropdownPosition.value = null
  } else {
    const buttonEl = groupButtonRefs.value.get(key.id)
    if (buttonEl) {
      const rect = buttonEl.getBoundingClientRect()
      dropdownPosition.value = {
        top: rect.bottom + 4,
        left: rect.left
      }
    }
    groupSelectorKeyId.value = key.id
  }
}

const changeGroup = async (key: ApiKey, newGroupId: number | null) => {
  groupSelectorKeyId.value = null
  dropdownPosition.value = null
  if (key.group_id === newGroupId) return

  try {
    await keysAPI.update(key.id, { group_id: newGroupId })
    appStore.showSuccess(t('keys.groupChangedSuccess'))
    loadApiKeys()
  } catch (error) {
    appStore.showError(t('keys.failedToChangeGroup'))
  }
}

const closeGroupSelector = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  // Check if click is inside the dropdown or the trigger button
  if (!target.closest('.group\\/dropdown') && !dropdownRef.value?.contains(target)) {
    groupSelectorKeyId.value = null
    dropdownPosition.value = null
  }
}

const confirmDelete = (key: ApiKey) => {
  selectedKey.value = key
  showDeleteDialog.value = true
}

const handleSubmit = async () => {
  // Validate group_id is required
  if (formData.value.group_id === null) {
    appStore.showError(t('keys.groupRequired'))
    return
  }

  // Validate custom key if enabled
  if (!showEditModal.value && formData.value.use_custom_key) {
    if (!formData.value.custom_key) {
      appStore.showError(t('keys.customKeyRequired'))
      return
    }
    if (customKeyError.value) {
      appStore.showError(customKeyError.value)
      return
    }
  }

  // Parse IP lists only if IP restriction is enabled
  const parseIPList = (text: string): string[] =>
    text.split('\n').map(ip => ip.trim()).filter(ip => ip.length > 0)
  const ipWhitelist = formData.value.enable_ip_restriction ? parseIPList(formData.value.ip_whitelist) : []
  const ipBlacklist = formData.value.enable_ip_restriction ? parseIPList(formData.value.ip_blacklist) : []

  submitting.value = true
  try {
    if (showEditModal.value && selectedKey.value) {
      await keysAPI.update(selectedKey.value.id, {
        name: formData.value.name,
        group_id: formData.value.group_id,
        status: formData.value.status,
        ip_whitelist: ipWhitelist,
        ip_blacklist: ipBlacklist
      })
      appStore.showSuccess(t('keys.keyUpdatedSuccess'))
    } else {
      const customKey = formData.value.use_custom_key ? formData.value.custom_key : undefined
      await keysAPI.create(formData.value.name, formData.value.group_id, customKey, ipWhitelist, ipBlacklist)
      appStore.showSuccess(t('keys.keyCreatedSuccess'))
      // Only advance tour if active, on submit step, and creation succeeded
      if (onboardingStore.isCurrentStep('[data-tour="key-form-submit"]')) {
        onboardingStore.nextStep(500)
      }
    }
    closeModals()
    loadApiKeys()
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToSave')
    appStore.showError(errorMsg)
    // Don't advance tour on error
  } finally {
    submitting.value = false
  }
}

/**
 * 处理删除 API Key 的操作
 * 优化：错误处理改进，优先显示后端返回的具体错误消息（如权限不足等），
 * 若后端未返回消息则显示默认的国际化文本
 */
const handleDelete = async () => {
  if (!selectedKey.value) return

  try {
    await keysAPI.delete(selectedKey.value.id)
    appStore.showSuccess(t('keys.keyDeletedSuccess'))
    showDeleteDialog.value = false
    loadApiKeys()
  } catch (error: any) {
    // 优先使用后端返回的错误消息，提供更具体的错误信息给用户
    const errorMsg = error?.message || t('keys.failedToDelete')
    appStore.showError(errorMsg)
  }
}

const closeModals = () => {
  showCreateModal.value = false
  showEditModal.value = false
  selectedKey.value = null
  formData.value = {
    name: '',
    group_id: null,
    status: 'active',
    use_custom_key: false,
    custom_key: '',
    enable_ip_restriction: false,
    ip_whitelist: '',
    ip_blacklist: ''
  }
}

const importToCcswitch = (row: ApiKey) => {
  const platform = row.group?.platform || 'anthropic'

  // For antigravity platform, show client selection dialog
  if (platform === 'antigravity') {
    pendingCcsRow.value = row
    showCcsClientSelect.value = true
    return
  }

  // For other platforms, execute directly
  executeCcsImport(row, platform === 'gemini' ? 'gemini' : 'claude')
}

const executeCcsImport = (row: ApiKey, clientType: 'claude' | 'gemini') => {
  const baseUrl = publicSettings.value?.api_base_url || window.location.origin
  const platform = row.group?.platform || 'anthropic'

  // Determine app name and endpoint based on platform and client type
  let app: string
  let endpoint: string

  if (platform === 'antigravity') {
    // Antigravity always uses /antigravity suffix
    app = clientType === 'gemini' ? 'gemini' : 'claude'
    endpoint = `${baseUrl}/antigravity`
  } else {
    switch (platform) {
      case 'openai':
        app = 'codex'
        endpoint = baseUrl
        break
      case 'gemini':
        app = 'gemini'
        endpoint = baseUrl
        break
      default: // anthropic
        app = 'claude'
        endpoint = baseUrl
    }
  }

  const usageScript = `({
    request: {
      url: "{{baseUrl}}/v1/usage",
      method: "GET",
      headers: { "Authorization": "Bearer {{apiKey}}" }
    },
    extractor: function(response) {
      return {
        isValid: response.is_active || true,
        remaining: response.balance,
        unit: "USD"
      };
    }
  })`
  const params = new URLSearchParams({
    resource: 'provider',
    app: app,
    name: 'sub2api',
    homepage: baseUrl,
    endpoint: endpoint,
    apiKey: row.key,
    configFormat: 'json',
    usageEnabled: 'true',
    usageScript: btoa(usageScript),
    usageAutoInterval: '30'
  })
  const deeplink = `ccswitch://v1/import?${params.toString()}`

  try {
    window.open(deeplink, '_self')

    // Check if the protocol handler worked by detecting if we're still focused
    setTimeout(() => {
      if (document.hasFocus()) {
        // Still focused means the protocol handler likely failed
        appStore.showError(t('keys.ccSwitchNotInstalled'))
      }
    }, 100)
  } catch (error) {
    appStore.showError(t('keys.ccSwitchNotInstalled'))
  }
}

const handleCcsClientSelect = (clientType: 'claude' | 'gemini') => {
  if (pendingCcsRow.value) {
    executeCcsImport(pendingCcsRow.value, clientType)
  }
  showCcsClientSelect.value = false
  pendingCcsRow.value = null
}

const closeCcsClientSelect = () => {
  showCcsClientSelect.value = false
  pendingCcsRow.value = null
}

onMounted(() => {
  loadApiKeys()
  loadGroups()
  loadPublicSettings()
  document.addEventListener('click', closeGroupSelector)
})

onUnmounted(() => {
  document.removeEventListener('click', closeGroupSelector)
})
</script>
