<script setup lang="ts">
import { computed, ref } from 'vue'
import { useAppStore } from '../stores/app'
import CodeEditor from './CodeEditor.vue'
import Tabs from 'primevue/tabs'
import TabList from 'primevue/tablist'
import Tab from 'primevue/tab'
import TabPanels from 'primevue/tabpanels'
import TabPanel from 'primevue/tabpanel'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import type {
  Response,
  ResponseTiming,
  AssertionResult,
  Header,
} from '../../bindings/github.com/shahadulhaider/restless/internal/model/models.js'

const appStore = useAppStore()
const activeTab = ref('body')
const copiedKey = ref<string | null>(null)

const response = computed<Response | null>(() => appStore.response)

// --- Status line ---
const statusCode = computed(() => response.value?.StatusCode ?? 0)
const statusText = computed(() => response.value?.Status ?? '')
const contentType = computed(() => response.value?.ContentType ?? '')

const statusColorClass = computed(() => {
  const code = statusCode.value
  if (code >= 200 && code < 300) return 'status-2xx'
  if (code >= 300 && code < 400) return 'status-3xx'
  if (code >= 400 && code < 500) return 'status-4xx'
  if (code >= 500) return 'status-5xx'
  return 'status-unknown'
})

/** Convert Go nanoseconds to display ms */
function nsToMs(ns: number): string {
  if (!ns || ns === 0) return '0'
  return (ns / 1_000_000).toFixed(1)
}

const totalTimeMs = computed(() => {
  const t = response.value?.Timing
  if (!t) return '0'
  return nsToMs(t.Total as number)
})

const bodySize = computed(() => {
  const body = response.value?.Body
  if (!body) return '0 B'
  const bytes = new Blob([body]).size
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
})

// --- Body tab ---
const bodyLanguage = computed(() => {
  const ct = contentType.value.toLowerCase()
  if (ct.includes('json')) return 'json'
  if (ct.includes('xml') || ct.includes('svg')) return 'xml'
  if (ct.includes('html')) return 'html'
  if (ct.includes('css')) return 'css'
  if (ct.includes('javascript') || ct.includes('ecmascript')) return 'javascript'
  return 'plaintext'
})

const formattedBody = computed(() => {
  const raw = response.value?.Body
  if (!raw) return ''
  if (bodyLanguage.value === 'json') {
    try {
      return JSON.stringify(JSON.parse(raw), null, 2)
    } catch {
      return raw
    }
  }
  return raw
})

// --- Headers tab ---
const headers = computed<Header[]>(() => response.value?.Headers ?? [])

async function copyValue(value: string, key: string) {
  try {
    await navigator.clipboard.writeText(value)
    copiedKey.value = key
    setTimeout(() => { copiedKey.value = null }, 1200)
  } catch { /* clipboard not available */ }
}

// --- Timing tab ---
interface TimingEntry {
  label: string
  ms: number
  color: string
}

const timingEntries = computed<TimingEntry[]>(() => {
  const t = response.value?.Timing
  if (!t) return []
  return [
    { label: 'DNS',      ms: (t.DNS as number) / 1_000_000,      color: 'var(--timing-dns)' },
    { label: 'Connect',  ms: (t.Connect as number) / 1_000_000,  color: 'var(--timing-connect)' },
    { label: 'TLS',      ms: (t.TLS as number) / 1_000_000,      color: 'var(--timing-tls)' },
    { label: 'TTFB',     ms: (t.TTFB as number) / 1_000_000,     color: 'var(--timing-ttfb)' },
    { label: 'Body',     ms: (t.BodyRead as number) / 1_000_000, color: 'var(--timing-body)' },
    { label: 'Total',    ms: (t.Total as number) / 1_000_000,    color: 'var(--timing-total)' },
  ]
})

const timingMax = computed(() => {
  const entries = timingEntries.value
  if (!entries.length) return 1
  return Math.max(...entries.map(e => e.ms), 1)
})

// --- Assertions tab ---
const assertions = computed<AssertionResult[]>(() => response.value?.AssertionResults ?? [])

const assertionSummary = computed(() => {
  const results = assertions.value
  const passed = results.filter(r => r.Passed).length
  return { passed, total: results.length }
})

// --- Error / Script error ---
const scriptError = computed(() => response.value?.ScriptError ?? '')
const hasResponse = computed(() => response.value !== null && response.value !== undefined)
</script>

<template>
  <!-- Loading spinner -->
  <div v-if="appStore.loading" class="rv-empty">
    <i class="pi pi-spin pi-spinner rv-spinner" />
    <span class="rv-empty-label">Sending request…</span>
  </div>

  <!-- Empty state -->
  <div v-else-if="!hasResponse" class="rv-empty">
    <i class="pi pi-arrow-right rv-empty-icon" />
    <span class="rv-empty-label">Send a request to see the response</span>
  </div>

  <!-- Response content -->
  <div v-else class="rv-root">
    <!-- Status line -->
    <div class="rv-status-bar">
      <span :class="['rv-status-badge', statusColorClass]">{{ statusCode }}</span>
      <span class="rv-status-text">{{ statusText }}</span>
      <span class="rv-meta-divider" />
      <span class="rv-meta">{{ contentType }}</span>
      <span class="rv-meta-divider" />
      <span class="rv-meta">{{ bodySize }}</span>
      <span class="rv-meta-divider" />
      <span class="rv-meta rv-meta-time">
        <i class="pi pi-stopwatch" />
        {{ totalTimeMs }} ms
      </span>
    </div>

    <!-- Script error banner -->
    <div v-if="scriptError" class="rv-error-banner">
      <i class="pi pi-exclamation-triangle" />
      <span>Script error: {{ scriptError }}</span>
    </div>

    <!-- Tabs -->
    <Tabs v-model:value="activeTab" class="rv-tabs">
      <TabList>
        <Tab value="body">Body</Tab>
        <Tab value="headers">
          Headers
          <span v-if="headers.length" class="rv-tab-count">{{ headers.length }}</span>
        </Tab>
        <Tab value="timing">Timing</Tab>
        <Tab value="assertions">
          Assertions
          <span v-if="assertions.length" :class="['rv-tab-count', assertionSummary.passed === assertionSummary.total ? 'count-pass' : 'count-fail']">
            {{ assertionSummary.passed }}/{{ assertionSummary.total }}
          </span>
        </Tab>
      </TabList>

      <TabPanels class="rv-tab-panels">
        <!-- Body -->
        <TabPanel value="body" class="rv-panel-body">
          <CodeEditor
            :model-value="formattedBody"
            :language="bodyLanguage"
            :read-only="true"
            height="100%"
          />
        </TabPanel>

        <!-- Headers -->
        <TabPanel value="headers" class="rv-panel-headers">
          <DataTable
            :value="headers"
            size="small"
            stripedRows
            class="rv-headers-table"
            scrollable
            scrollHeight="flex"
          >
            <Column field="Key" header="Header" :style="{ width: '35%' }">
              <template #body="{ data }">
                <span class="rv-header-key">{{ data.Key }}</span>
              </template>
            </Column>
            <Column field="Value" header="Value">
              <template #body="{ data }">
                <span
                  class="rv-header-value"
                  :title="'Click to copy'"
                  @click="copyValue(data.Value, data.Key)"
                >
                  {{ data.Value }}
                  <i
                    :class="['rv-copy-icon', 'pi', copiedKey === data.Key ? 'pi-check' : 'pi-copy']"
                  />
                </span>
              </template>
            </Column>
          </DataTable>
        </TabPanel>

        <!-- Timing -->
        <TabPanel value="timing" class="rv-panel-timing">
          <div class="rv-waterfall">
            <div v-for="entry in timingEntries" :key="entry.label" class="rv-wf-row">
              <span class="rv-wf-label">{{ entry.label }}</span>
              <div class="rv-wf-track">
                <div
                  class="rv-wf-bar"
                  :style="{
                    width: `${Math.max((entry.ms / timingMax) * 100, entry.ms > 0 ? 2 : 0)}%`,
                    backgroundColor: entry.color,
                  }"
                />
              </div>
              <span class="rv-wf-value">{{ entry.ms.toFixed(1) }} ms</span>
            </div>
          </div>
        </TabPanel>

        <!-- Assertions -->
        <TabPanel value="assertions" class="rv-panel-assertions">
          <div v-if="!assertions.length" class="rv-assertions-empty">
            <i class="pi pi-check-circle" />
            <span>No assertions defined</span>
          </div>
          <div v-else class="rv-assertion-list">
            <div
              v-for="(result, idx) in assertions"
              :key="idx"
              :class="['rv-assertion-item', result.Passed ? 'assertion-pass' : 'assertion-fail']"
            >
              <i :class="['rv-assertion-icon', 'pi', result.Passed ? 'pi-check-circle' : 'pi-times-circle']" />
              <div class="rv-assertion-body">
                <span class="rv-assertion-expr">{{ result.Assertion.Raw }}</span>
                <div v-if="!result.Passed" class="rv-assertion-detail">
                  <span v-if="result.Actual" class="rv-assertion-actual">
                    Got: <code>{{ result.Actual }}</code>
                  </span>
                  <span v-if="result.Error" class="rv-assertion-error">{{ result.Error }}</span>
                </div>
              </div>
            </div>
          </div>
        </TabPanel>
      </TabPanels>
    </Tabs>
  </div>
</template>

<style scoped>
/* ─── Catppuccin Mocha timing bar palette ─── */
.rv-root {
  --timing-dns:     #89b4fa; /* blue */
  --timing-connect: #a6e3a1; /* green */
  --timing-tls:     #f9e2af; /* yellow */
  --timing-ttfb:    #cba6f7; /* mauve */
  --timing-body:    #fab387; /* peach */
  --timing-total:   #f38ba8; /* red */
}

/* ─── Layout ─── */
.rv-root {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.rv-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 0.75rem;
  opacity: 0.4;
  user-select: none;
}

.rv-spinner {
  font-size: 1.75rem;
  color: var(--app-text);
}

.rv-empty-icon {
  font-size: 2rem;
}

.rv-empty-label {
  font-size: 0.85rem;
  font-weight: 500;
  letter-spacing: 0.03em;
  text-transform: uppercase;
}

/* ─── Status bar ─── */
.rv-status-bar {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  background: var(--app-surface);
  border-bottom: 1px solid var(--app-border);
  flex-shrink: 0;
  font-size: 0.8rem;
}

.rv-status-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.15rem 0.5rem;
  border-radius: 4px;
  font-weight: 700;
  font-size: 0.8rem;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.02em;
}

.status-2xx { background: rgba(166, 227, 161, 0.18); color: #a6e3a1; }
.status-3xx { background: rgba(137, 180, 250, 0.18); color: #89b4fa; }
.status-4xx { background: rgba(250, 179, 135, 0.18); color: #fab387; }
.status-5xx { background: rgba(243, 139, 168, 0.18); color: #f38ba8; }
.status-unknown { background: var(--app-overlay); color: var(--app-muted); }

.rv-status-text {
  color: var(--app-muted);
  font-weight: 500;
}

.rv-meta-divider {
  width: 1px;
  height: 14px;
  background: var(--app-border);
  flex-shrink: 0;
}

.rv-meta {
  color: var(--app-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.rv-meta-time {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  color: #f9e2af;
  font-variant-numeric: tabular-nums;
}

.rv-meta-time .pi {
  font-size: 0.75rem;
}

/* ─── Error banner ─── */
.rv-error-banner {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.4rem 0.75rem;
  background: rgba(243, 139, 168, 0.12);
  color: #f38ba8;
  font-size: 0.78rem;
  border-bottom: 1px solid rgba(243, 139, 168, 0.2);
  flex-shrink: 0;
}

.rv-error-banner .pi {
  font-size: 0.85rem;
}

/* ─── Tabs ─── */
.rv-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.rv-tabs :deep(.p-tablist) {
  background: var(--app-surface);
  border-bottom: 1px solid var(--app-border);
}

.rv-tabs :deep(.p-tab) {
  font-size: 0.78rem;
  font-weight: 600;
  letter-spacing: 0.03em;
  text-transform: uppercase;
  padding: 0.55rem 0.9rem;
  color: var(--app-muted);
  transition: color 0.15s ease;
  gap: 0.35rem;
}

.rv-tabs :deep(.p-tab:hover) {
  color: var(--app-text);
}

.rv-tabs :deep(.p-tab-active) {
  color: #89b4fa;
}

.rv-tabs :deep(.p-tabpanels) {
  flex: 1;
  overflow: hidden;
  background: transparent;
  padding: 0;
}

.rv-tabs :deep(.p-tabpanel) {
  height: 100%;
  padding: 0;
}

.rv-tab-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.25rem;
  padding: 0 0.3rem;
  height: 1.1rem;
  border-radius: 3px;
  font-size: 0.65rem;
  font-weight: 700;
  background: var(--app-overlay);
  color: var(--app-text);
  font-variant-numeric: tabular-nums;
}

.rv-tab-count.count-pass { background: rgba(166, 227, 161, 0.18); color: #a6e3a1; }
.rv-tab-count.count-fail { background: rgba(243, 139, 168, 0.18); color: #f38ba8; }

.rv-tab-panels {
  flex: 1;
  overflow: hidden;
}

/* ─── Body panel ─── */
.rv-panel-body {
  height: 100%;
}

/* ─── Headers panel ─── */
.rv-panel-headers {
  height: 100%;
  overflow: auto;
}

.rv-headers-table {
  font-size: 0.8rem;
}

.rv-headers-table :deep(.p-datatable-table) {
  background: transparent;
}

.rv-headers-table :deep(.p-datatable-thead > tr > th) {
  background: var(--app-surface);
  color: var(--app-muted);
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-color: var(--app-border);
  padding: 0.5rem 0.75rem;
}

.rv-headers-table :deep(.p-datatable-tbody > tr > td) {
  border-color: var(--app-border);
  padding: 0.4rem 0.75rem;
}

.rv-headers-table :deep(.p-datatable-tbody > tr:hover) {
  background: rgba(69, 71, 90, 0.3);
}

.rv-header-key {
  color: #89b4fa;
  font-weight: 600;
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.78rem;
}

.rv-header-value {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  cursor: pointer;
  color: var(--app-text);
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.78rem;
  transition: color 0.12s ease;
  word-break: break-all;
}

.rv-header-value:hover {
  color: #a6e3a1;
}

.rv-copy-icon {
  font-size: 0.65rem;
  opacity: 0;
  transition: opacity 0.12s ease;
  flex-shrink: 0;
}

.rv-header-value:hover .rv-copy-icon {
  opacity: 0.7;
}

.rv-copy-icon.pi-check {
  opacity: 1;
  color: #a6e3a1;
}

/* ─── Timing waterfall ─── */
.rv-panel-timing {
  padding: 1rem;
  overflow: auto;
}

.rv-waterfall {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
  max-width: 600px;
}

.rv-wf-row {
  display: grid;
  grid-template-columns: 4.5rem 1fr 5rem;
  align-items: center;
  gap: 0.5rem;
}

.rv-wf-label {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--app-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  text-align: right;
}

.rv-wf-track {
  height: 18px;
  background: var(--app-overlay);
  border-radius: 3px;
  overflow: hidden;
}

.rv-wf-bar {
  height: 100%;
  border-radius: 3px;
  transition: width 0.4s cubic-bezier(0.22, 1, 0.36, 1);
  min-width: 0;
}

.rv-wf-value {
  font-size: 0.73rem;
  font-weight: 600;
  color: var(--app-text);
  font-variant-numeric: tabular-nums;
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace;
}

/* ─── Assertions panel ─── */
.rv-panel-assertions {
  padding: 0.75rem;
  overflow: auto;
}

.rv-assertions-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 2rem;
  opacity: 0.4;
  user-select: none;
}

.rv-assertions-empty .pi {
  font-size: 1.5rem;
}

.rv-assertions-empty span {
  font-size: 0.8rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.rv-assertion-list {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.rv-assertion-item {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.5rem 0.65rem;
  border-radius: 4px;
  transition: background 0.12s ease;
}

.rv-assertion-item:hover {
  background: rgba(69, 71, 90, 0.2);
}

.rv-assertion-icon {
  font-size: 0.9rem;
  margin-top: 0.1rem;
  flex-shrink: 0;
}

.assertion-pass .rv-assertion-icon { color: #a6e3a1; }
.assertion-fail .rv-assertion-icon { color: #f38ba8; }

.rv-assertion-body {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
}

.rv-assertion-expr {
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.78rem;
  color: var(--app-text);
}

.rv-assertion-detail {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.rv-assertion-actual {
  font-size: 0.73rem;
  color: var(--app-muted);
}

.rv-assertion-actual code {
  color: #fab387;
  background: rgba(250, 179, 135, 0.1);
  padding: 0.1rem 0.3rem;
  border-radius: 3px;
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.73rem;
}

.rv-assertion-error {
  font-size: 0.73rem;
  color: #f38ba8;
}

.assertion-pass {
  border-left: 2px solid rgba(166, 227, 161, 0.3);
}

.assertion-fail {
  border-left: 2px solid rgba(243, 139, 168, 0.3);
  background: rgba(243, 139, 168, 0.04);
}
</style>
