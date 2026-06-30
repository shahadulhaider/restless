<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import Select from 'primevue/select'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import CodeEditor from './CodeEditor.vue'
import { useAppStore } from '../stores/app'
import { Execute } from '../../bindings/github.com/shahadulhaider/restless/internal/gui/requestservice.js'

const appStore = useAppStore()

// ── Method options with Catppuccin colors ──
const methods = [
  { label: 'GET', value: 'GET', color: '#a6e3a1' },
  { label: 'POST', value: 'POST', color: '#89b4fa' },
  { label: 'PUT', value: 'PUT', color: '#fab387' },
  { label: 'DELETE', value: 'DELETE', color: '#f38ba8' },
  { label: 'PATCH', value: 'PATCH', color: '#cba6f7' },
  { label: 'HEAD', value: 'HEAD', color: '#94e2d5' },
  { label: 'OPTIONS', value: 'OPTIONS', color: '#f9e2af' },
]

// ── Form state ──
const method = ref('GET')
const url = ref('')
const headers = ref<Array<{ key: string; value: string; id: number }>>([
  { key: '', value: '', id: Date.now() },
])
const body = ref('')
const activeTab = ref<'headers' | 'body'>('headers')

let headerIdCounter = Date.now()

// ── Computed ──
const selectedMethodColor = computed(() => {
  return methods.find(m => m.value === method.value)?.color ?? 'var(--app-text)'
})

const bodyMethods = ['POST', 'PUT', 'PATCH']
const showBody = computed(() => bodyMethods.includes(method.value))

const bodyLanguage = computed(() => {
  const ct = headers.value.find(
    h => h.key.toLowerCase() === 'content-type',
  )?.value?.toLowerCase()
  if (!ct) return 'json'
  if (ct.includes('json')) return 'json'
  if (ct.includes('xml')) return 'xml'
  if (ct.includes('html')) return 'html'
  return 'plaintext'
})

// ── Headers table actions ──
function addHeader() {
  headerIdCounter++
  headers.value.push({ key: '', value: '', id: headerIdCounter })
}

function removeHeader(index: number) {
  headers.value.splice(index, 1)
  if (headers.value.length === 0) addHeader()
}

// ── Build request from form ──
function buildRequest() {
  const hdrs = headers.value
    .filter(h => h.key.trim() !== '')
    .map(h => ({ Key: h.key, Value: h.value }))

  return {
    Name: appStore.selectedRequest?.Name ?? '',
    Method: method.value,
    URL: url.value,
    HTTPVersion: '',
    Headers: hdrs,
    Body: showBody.value ? body.value : '',
    BodyFile: '',
    Metadata: appStore.selectedRequest?.Metadata ?? {
      NoRedirect: false,
      NoCookieJar: false,
      Timeout: 0,
      ConnTimeout: 0,
      Insecure: false,
      Proxy: '',
    },
    Assertions: appStore.selectedRequest?.Assertions ?? [],
    PreRequestScript: appStore.selectedRequest?.PreRequestScript ?? '',
    PostResponseScript: appStore.selectedRequest?.PostResponseScript ?? '',
    SourceFile: appStore.selectedRequest?.SourceFile ?? '',
    SourceLine: appStore.selectedRequest?.SourceLine ?? 0,
  }
}

// ── Send request ──
async function sendRequest() {
  if (!url.value.trim()) return

  appStore.loading = true
  try {
    const req = buildRequest()
    const result = await Execute(req as any, appStore.activeEnv)
    appStore.response = result
  } catch (err: any) {
    appStore.response = {
      StatusCode: 0,
      Status: 'Error',
      Headers: [],
      Body: err?.message ?? String(err),
      ContentType: 'text/plain',
    }
  } finally {
    appStore.loading = false
  }
}

// ── Keyboard shortcut: Ctrl+Enter ──
function handleKeydown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
    e.preventDefault()
    sendRequest()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})

// ── Watch store for selected request changes ──
watch(
  () => appStore.selectedRequest,
  (req) => {
    if (!req) return
    method.value = req.Method || 'GET'
    url.value = req.URL || ''
    body.value = req.Body || ''

    const reqHeaders = req.Headers ?? []
    if (reqHeaders.length > 0) {
      headers.value = reqHeaders.map((h: any, i: number) => ({
        key: h.Key ?? '',
        value: h.Value ?? '',
        id: Date.now() + i,
      }))
    } else {
      headerIdCounter++
      headers.value = [{ key: '', value: '', id: headerIdCounter }]
    }

    // Switch to body tab if method supports body and body is present
    if (bodyMethods.includes(req.Method) && req.Body) {
      activeTab.value = 'body'
    } else {
      activeTab.value = 'headers'
    }
  },
  { deep: true },
)
</script>

<template>
  <div class="request-editor">
    <!-- ── URL Bar ── -->
    <div class="url-bar">
      <Select
        v-model="method"
        :options="methods"
        optionLabel="label"
        optionValue="value"
        class="method-select"
        :pt="{
          label: { style: `color: ${selectedMethodColor}; font-weight: 700; font-family: 'JetBrains Mono', 'Fira Code', monospace; font-size: 0.8rem; letter-spacing: 0.03em` },
        }"
      >
        <template #option="{ option }">
          <span
            class="method-option"
            :style="{ color: option.color }"
          >
            {{ option.label }}
          </span>
        </template>
      </Select>

      <InputText
        v-model="url"
        placeholder="Enter request URL"
        class="url-input"
        @keydown.ctrl.enter.prevent="sendRequest"
        @keydown.meta.enter.prevent="sendRequest"
      />

      <Button
        label="Send"
        icon="pi pi-send"
        class="send-btn"
        :loading="appStore.loading"
        @click="sendRequest"
      />
    </div>

    <!-- ── Tab Bar ── -->
    <div class="tab-bar">
      <button
        class="tab-item"
        :class="{ active: activeTab === 'headers' }"
        @click="activeTab = 'headers'"
      >
        <i class="pi pi-list" />
        Headers
        <span v-if="headers.filter(h => h.key.trim()).length" class="tab-badge">
          {{ headers.filter(h => h.key.trim()).length }}
        </span>
      </button>
      <button
        v-if="showBody"
        class="tab-item"
        :class="{ active: activeTab === 'body' }"
        @click="activeTab = 'body'"
      >
        <i class="pi pi-code" />
        Body
      </button>
    </div>

    <!-- ── Tab Content ── -->
    <div class="tab-content">
      <!-- Headers Table -->
      <div v-if="activeTab === 'headers'" class="headers-panel">
        <div class="headers-table">
          <div class="headers-row headers-head">
            <span class="col-key">Key</span>
            <span class="col-value">Value</span>
            <span class="col-actions"></span>
          </div>
          <div
            v-for="(header, index) in headers"
            :key="header.id"
            class="headers-row"
          >
            <InputText
              v-model="header.key"
              placeholder="Header name"
              class="header-input"
            />
            <InputText
              v-model="header.value"
              placeholder="Header value"
              class="header-input"
            />
            <button
              class="row-action-btn remove-btn"
              title="Remove header"
              @click="removeHeader(index)"
            >
              <i class="pi pi-times" />
            </button>
          </div>
        </div>
        <button class="add-header-btn" @click="addHeader">
          <i class="pi pi-plus" />
          Add Header
        </button>
      </div>

      <!-- Body Editor -->
      <div v-if="activeTab === 'body' && showBody" class="body-panel">
        <CodeEditor
          v-model="body"
          :language="bodyLanguage"
          height="100%"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.request-editor {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

/* ── URL Bar ── */
.url-bar {
  display: flex;
  align-items: center;
  gap: 0;
  padding: 0.625rem 0.75rem;
  border-bottom: 1px solid var(--app-border);
  background: var(--app-surface);
  --wails-draggable: no-drag;
}

.method-select {
  flex-shrink: 0;
  width: 7.5rem;
  border-radius: 6px 0 0 6px;
  border-right: none;
}

/* Override PrimeVue Select styling to blend */
.method-select :deep(.p-select) {
  background: var(--app-bg);
  border-color: var(--app-border);
  border-radius: 6px 0 0 6px;
  border-right: 1px solid var(--app-overlay);
  height: 2.375rem;
}

.method-select :deep(.p-select:hover),
.method-select :deep(.p-select.p-focus) {
  border-color: var(--app-muted);
}

.method-option {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-weight: 700;
  font-size: 0.8rem;
  letter-spacing: 0.03em;
}

.url-input {
  flex: 1;
  border-radius: 0;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 0.85rem;
  letter-spacing: 0.01em;
}

.url-input:deep(.p-inputtext) {
  background: var(--app-bg);
  border-color: var(--app-border);
  color: var(--app-text);
  height: 2.375rem;
  border-radius: 0;
}

.send-btn {
  flex-shrink: 0;
  border-radius: 0 6px 6px 0;
  font-weight: 600;
  font-size: 0.82rem;
  letter-spacing: 0.02em;
  height: 2.375rem;
  padding: 0 1.125rem;
}

/* ── Tab Bar ── */
.tab-bar {
  display: flex;
  gap: 0;
  padding: 0 0.75rem;
  background: var(--app-surface);
  border-bottom: 1px solid var(--app-border);
  --wails-draggable: no-drag;
}

.tab-item {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.5rem 0.875rem;
  border: none;
  background: none;
  color: var(--app-muted);
  font-size: 0.78rem;
  font-weight: 500;
  letter-spacing: 0.02em;
  cursor: pointer;
  position: relative;
  transition: color 0.15s ease;
  --wails-draggable: no-drag;
}

.tab-item:hover {
  color: var(--app-text);
}

.tab-item.active {
  color: var(--app-text);
}

.tab-item.active::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0.5rem;
  right: 0.5rem;
  height: 2px;
  background: #89b4fa;
  border-radius: 1px 1px 0 0;
}

.tab-item i {
  font-size: 0.72rem;
}

.tab-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.15rem;
  height: 1.15rem;
  padding: 0 0.3rem;
  border-radius: 8px;
  background: var(--app-overlay);
  color: var(--app-text);
  font-size: 0.65rem;
  font-weight: 600;
}

/* ── Tab Content ── */
.tab-content {
  flex: 1;
  overflow: hidden;
  background: var(--app-bg);
}

/* ── Headers Panel ── */
.headers-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow-y: auto;
  padding: 0.5rem 0.75rem;
}

.headers-table {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.headers-row {
  display: grid;
  grid-template-columns: 1fr 1fr 2rem;
  gap: 0.375rem;
  align-items: center;
  padding: 0.25rem 0;
}

.headers-head {
  padding-bottom: 0.375rem;
  border-bottom: 1px solid var(--app-border);
  margin-bottom: 0.25rem;
}

.headers-head span {
  font-size: 0.68rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--app-muted);
}

.header-input {
  width: 100%;
  font-size: 0.82rem;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
}

.header-input:deep(.p-inputtext) {
  background: var(--app-surface);
  border-color: transparent;
  color: var(--app-text);
  height: 2rem;
  font-size: 0.82rem;
  padding: 0 0.5rem;
}

.header-input:deep(.p-inputtext:hover) {
  border-color: var(--app-border);
}

.header-input:deep(.p-inputtext:focus) {
  border-color: var(--app-muted);
  box-shadow: none;
}

.row-action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 1.75rem;
  height: 1.75rem;
  border: none;
  background: none;
  color: var(--app-muted);
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.12s ease;
  --wails-draggable: no-drag;
}

.remove-btn:hover {
  background: rgba(243, 139, 168, 0.12);
  color: #f38ba8;
}

.add-header-btn {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.625rem;
  margin-top: 0.375rem;
  border: 1px dashed var(--app-border);
  background: none;
  color: var(--app-muted);
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.15s ease;
  --wails-draggable: no-drag;
}

.add-header-btn:hover {
  border-color: var(--app-muted);
  color: var(--app-text);
  background: var(--app-surface);
}

.add-header-btn i {
  font-size: 0.65rem;
}

/* ── Body Panel ── */
.body-panel {
  height: 100%;
  overflow: hidden;
}
</style>
