<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import Select from 'primevue/select'
import Button from 'primevue/button'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import { useAppStore } from '../stores/app'
import {
  ListEnvironmentNames,
  ResolveVars,
  SetCurrentEnv,
} from '../../bindings/github.com/shahadulhaider/restless/internal/gui/environmentservice.js'

const appStore = useAppStore()

const environments = ref<string[]>([])
const resolvedVars = ref<{ key: string; value: string }[]>([])
const showVars = ref(false)
const loadingVars = ref(false)

const NO_ENV = '(no environment)'

const envOptions = computed(() => [NO_ENV, ...environments.value])

const selectedEnv = computed({
  get: () => appStore.activeEnv || NO_ENV,
  set: (val: string) => {
    const env = val === NO_ENV ? '' : val
    appStore.activeEnv = env
    SetCurrentEnv(env).catch(() => {})
  },
})

async function loadEnvironments() {
  if (!appStore.rootDir) return
  try {
    const names = await ListEnvironmentNames(appStore.rootDir)
    environments.value = names ?? []
  } catch {
    environments.value = []
  }
}

async function loadVars() {
  if (!appStore.rootDir) return
  loadingVars.value = true
  try {
    const vars = await ResolveVars(appStore.rootDir, appStore.activeEnv)
    if (vars) {
      resolvedVars.value = Object.entries(vars).map(([key, value]) => ({
        key,
        value: value ?? '',
      }))
    } else {
      resolvedVars.value = []
    }
  } catch {
    resolvedVars.value = []
  } finally {
    loadingVars.value = false
  }
}

function toggleVars() {
  showVars.value = !showVars.value
  if (showVars.value) loadVars()
}

watch(() => appStore.rootDir, loadEnvironments)
watch(() => appStore.activeEnv, () => {
  if (showVars.value) loadVars()
})

onMounted(loadEnvironments)
</script>

<template>
  <div class="es-root">
    <!-- Environment dropdown -->
    <div class="es-selector">
      <i class="pi pi-globe es-icon" />
      <Select
        v-model="selectedEnv"
        :options="envOptions"
        class="es-select"
        :pt="{
          root: { style: 'border: none; background: transparent; box-shadow: none;' },
          label: { style: `color: ${selectedEnv === NO_ENV ? 'var(--app-muted)' : '#a6e3a1'}; font-size: 0.78rem; font-weight: 600; padding: 0.3rem 0.5rem;` },
        }"
      />
      <Button
        :icon="showVars ? 'pi pi-eye-slash' : 'pi pi-eye'"
        severity="secondary"
        text
        rounded
        size="small"
        class="es-vars-toggle"
        @click="toggleVars"
        v-tooltip.bottom="showVars ? 'Hide variables' : 'Show variables'"
      />
    </div>

    <!-- Variable preview panel -->
    <div v-if="showVars" class="es-vars-panel">
      <div class="es-vars-header">
        <span class="es-vars-title">
          <i class="pi pi-code" />
          Resolved Variables
        </span>
        <Button
          icon="pi pi-refresh"
          severity="secondary"
          text
          rounded
          size="small"
          :loading="loadingVars"
          @click="loadVars"
        />
      </div>

      <div v-if="resolvedVars.length === 0 && !loadingVars" class="es-vars-empty">
        <i class="pi pi-info-circle" />
        <span>No variables resolved</span>
      </div>

      <DataTable
        v-else
        :value="resolvedVars"
        size="small"
        scrollable
        scrollHeight="240px"
        class="es-vars-table"
      >
        <Column field="key" header="Variable" :style="{ width: '40%' }">
          <template #body="{ data }">
            <span class="es-var-key">{{ data.key }}</span>
          </template>
        </Column>
        <Column field="value" header="Value">
          <template #body="{ data }">
            <span class="es-var-value">{{ data.value }}</span>
          </template>
        </Column>
      </DataTable>
    </div>
  </div>
</template>

<style scoped>
.es-root {
  display: flex;
  flex-direction: column;
  --wails-draggable: no-drag;
}

.es-selector {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.es-icon {
  font-size: 0.85rem;
  color: var(--app-muted);
}

.es-select {
  min-width: 10rem;
}

.es-select :deep(.p-select-dropdown) {
  color: var(--app-muted);
  width: 1.5rem;
}

.es-vars-toggle {
  width: 1.75rem;
  height: 1.75rem;
  color: var(--app-muted);
}

.es-vars-toggle:hover {
  color: #89b4fa;
}

/* ─── Variables panel ─── */
.es-vars-panel {
  position: absolute;
  top: 100%;
  right: 0;
  width: 420px;
  max-height: 320px;
  background: var(--app-surface);
  border: 1px solid var(--app-border);
  border-radius: 6px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  z-index: 100;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  margin-top: 0.35rem;
}

.es-vars-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 0.75rem;
  border-bottom: 1px solid var(--app-border);
  flex-shrink: 0;
}

.es-vars-title {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.73rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--app-muted);
}

.es-vars-title .pi {
  font-size: 0.75rem;
}

.es-vars-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 1.5rem;
  color: var(--app-muted);
  font-size: 0.8rem;
  opacity: 0.6;
}

.es-vars-table {
  font-size: 0.78rem;
}

.es-vars-table :deep(.p-datatable-thead > tr > th) {
  background: rgba(69, 71, 90, 0.3);
  color: var(--app-muted);
  font-size: 0.68rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-color: var(--app-border);
  padding: 0.4rem 0.75rem;
}

.es-vars-table :deep(.p-datatable-tbody > tr > td) {
  border-color: var(--app-border);
  padding: 0.3rem 0.75rem;
}

.es-vars-table :deep(.p-datatable-tbody > tr:hover) {
  background: rgba(69, 71, 90, 0.3);
}

.es-var-key {
  color: #89b4fa;
  font-weight: 600;
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.75rem;
}

.es-var-value {
  color: var(--app-text);
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.75rem;
  word-break: break-all;
}
</style>
