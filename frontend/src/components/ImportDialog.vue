<script setup lang="ts">
import { ref, computed } from 'vue'
import Dialog from 'primevue/dialog'
import Select from 'primevue/select'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import { useToast } from 'primevue/usetoast'
import { useAppStore } from '../stores/app'
import {
  ImportPostman,
  ImportInsomnia,
  ImportBruno,
  ImportCurl,
  ImportOpenAPI,
} from '../../bindings/github.com/shahadulhaider/restless/internal/gui/importerservice.js'

const appStore = useAppStore()
const toast = useToast()

const visible = ref(false)
const importing = ref(false)

interface SourceType {
  label: string
  value: string
  inputLabel: string
  placeholder: string
}

const sourceTypes: SourceType[] = [
  { label: 'Postman', value: 'postman', inputLabel: 'Collection file path', placeholder: '/path/to/postman_collection.json' },
  { label: 'Insomnia', value: 'insomnia', inputLabel: 'Export file path', placeholder: '/path/to/insomnia_export.json' },
  { label: 'Bruno', value: 'bruno', inputLabel: 'Collection directory', placeholder: '/path/to/bruno/collection' },
  { label: 'curl', value: 'curl', inputLabel: 'curl command', placeholder: 'curl -X GET https://api.example.com/users' },
  { label: 'OpenAPI / Swagger', value: 'openapi', inputLabel: 'Spec file path', placeholder: '/path/to/openapi.yaml' },
]

const selectedSource = ref<SourceType>(sourceTypes[0])
const inputValue = ref('')
const outputDir = ref('')

const currentSource = computed(() => selectedSource.value)
const isCurl = computed(() => currentSource.value.value === 'curl')

async function doImport() {
  if (!inputValue.value.trim()) {
    toast.add({ severity: 'warn', summary: 'Missing input', detail: `Please provide ${currentSource.value.inputLabel.toLowerCase()}`, life: 3000 })
    return
  }
  const output = outputDir.value.trim() || appStore.rootDir || '.'
  importing.value = true
  try {
    switch (currentSource.value.value) {
      case 'postman':
        await ImportPostman(inputValue.value.trim(), output)
        break
      case 'insomnia':
        await ImportInsomnia(inputValue.value.trim(), output)
        break
      case 'bruno':
        await ImportBruno(inputValue.value.trim(), output)
        break
      case 'curl':
        await ImportCurl(inputValue.value.trim(), output)
        break
      case 'openapi':
        await ImportOpenAPI(inputValue.value.trim(), output)
        break
    }
    toast.add({ severity: 'success', summary: 'Import complete', detail: `Successfully imported from ${currentSource.value.label}`, life: 4000 })
    visible.value = false
    inputValue.value = ''
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Import failed', detail: String(err), life: 6000 })
  } finally {
    importing.value = false
  }
}

function open() {
  visible.value = true
  outputDir.value = appStore.rootDir || ''
}

defineExpose({ open })
</script>

<template>
  <Button
    icon="pi pi-download"
    label="Import"
    severity="secondary"
    outlined
    size="small"
    class="id-trigger"
    @click="open"
  />

  <Dialog
    v-model:visible="visible"
    header="Import Collection"
    modal
    :style="{ width: '520px' }"
    :pt="{
      root: { class: 'id-dialog' },
      header: { class: 'id-dialog-header' },
      content: { class: 'id-dialog-content' },
    }"
  >
    <div class="id-form">
      <!-- Source type selector -->
      <div class="id-field">
        <label class="id-label">Source Type</label>
        <Select
          v-model="selectedSource"
          :options="sourceTypes"
          optionLabel="label"
          class="id-select"
        />
      </div>

      <!-- Source icon hint -->
      <div class="id-source-hint">
        <i :class="['pi', isCurl ? 'pi-terminal' : 'pi pi-file']" />
        <span>{{ isCurl ? 'Paste a curl command below' : 'Enter the file or directory path' }}</span>
      </div>

      <!-- Input path / curl command -->
      <div class="id-field">
        <label class="id-label">{{ currentSource.inputLabel }}</label>
        <InputText
          v-model="inputValue"
          :placeholder="currentSource.placeholder"
          class="id-input"
          :class="{ 'id-input-mono': isCurl }"
        />
      </div>

      <!-- Output directory -->
      <div class="id-field">
        <label class="id-label">Output Directory</label>
        <InputText
          v-model="outputDir"
          placeholder="Output directory (defaults to project root)"
          class="id-input"
        />
      </div>

      <!-- Import button -->
      <div class="id-actions">
        <Button
          label="Cancel"
          severity="secondary"
          text
          size="small"
          @click="visible = false"
        />
        <Button
          icon="pi pi-download"
          label="Import"
          size="small"
          :loading="importing"
          @click="doImport"
          class="id-import-btn"
        />
      </div>
    </div>
  </Dialog>
</template>

<style scoped>
.id-trigger {
  font-size: 0.78rem;
  --wails-draggable: no-drag;
}

/* ─── Dialog overrides ─── */
:deep(.id-dialog) {
  background: var(--app-bg);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

:deep(.id-dialog-header) {
  background: var(--app-surface);
  border-bottom: 1px solid var(--app-border);
  padding: 0.75rem 1rem;
  font-size: 0.85rem;
  font-weight: 700;
  letter-spacing: 0.02em;
  color: var(--app-text);
}

:deep(.id-dialog-content) {
  padding: 0;
  background: var(--app-bg);
}

/* ─── Form ─── */
.id-form {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
  padding: 1rem;
}

.id-field {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}

.id-label {
  font-size: 0.68rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--app-muted);
}

.id-select {
  width: 100%;
}

.id-input {
  width: 100%;
  font-size: 0.82rem;
}

.id-input-mono {
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.78rem;
}

.id-source-hint {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  background: rgba(137, 180, 250, 0.06);
  border: 1px solid rgba(137, 180, 250, 0.12);
  border-radius: 4px;
  font-size: 0.75rem;
  color: #89b4fa;
}

.id-source-hint .pi {
  font-size: 0.8rem;
}

.id-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  padding-top: 0.5rem;
  border-top: 1px solid var(--app-border);
}

.id-import-btn {
  background: rgba(166, 227, 161, 0.15);
  border-color: rgba(166, 227, 161, 0.3);
  color: #a6e3a1;
}

.id-import-btn:hover {
  background: rgba(166, 227, 161, 0.25);
}
</style>
