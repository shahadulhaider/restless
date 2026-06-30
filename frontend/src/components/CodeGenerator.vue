<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import Dialog from 'primevue/dialog'
import Select from 'primevue/select'
import Button from 'primevue/button'
import { useAppStore } from '../stores/app'
import CodeEditor from './CodeEditor.vue'
import {
  AvailableLanguages,
  GenerateCode,
  ToCurl,
} from '../../bindings/github.com/shahadulhaider/restless/internal/gui/exporterservice.js'

const appStore = useAppStore()

const visible = ref(false)
const languages = ref<string[]>([])
const selectedLang = ref('')
const generatedCode = ref('')
const generating = ref(false)
const copied = ref(false)

const editorLang = ref('plaintext')

function langToMonaco(lang: string): string {
  const l = lang.toLowerCase()
  if (l === 'python') return 'python'
  if (l === 'javascript') return 'javascript'
  if (l === 'go') return 'go'
  if (l === 'java') return 'java'
  if (l === 'ruby') return 'ruby'
  if (l === 'curl' || l === 'httpie' || l === 'powershell') return 'shell'
  return 'plaintext'
}

async function loadLanguages() {
  try {
    const langs = await AvailableLanguages()
    languages.value = langs ?? []
    if (languages.value.length > 0 && !selectedLang.value) {
      selectedLang.value = languages.value.find(l => l.toLowerCase() === 'curl') || languages.value[0]
    }
  } catch {
    languages.value = []
  }
}

async function generate() {
  const req = appStore.selectedRequest
  if (!req || !selectedLang.value) return
  generating.value = true
  try {
    if (selectedLang.value.toLowerCase() === 'curl') {
      generatedCode.value = await ToCurl(req)
    } else {
      generatedCode.value = await GenerateCode(req, selectedLang.value)
    }
    editorLang.value = langToMonaco(selectedLang.value)
  } catch (err) {
    generatedCode.value = `// Error: ${err}`
  } finally {
    generating.value = false
  }
}

async function copyToClipboard() {
  if (!generatedCode.value) return
  try {
    await navigator.clipboard.writeText(generatedCode.value)
    copied.value = true
    setTimeout(() => { copied.value = false }, 1500)
  } catch { /* clipboard unavailable */ }
}

function open() {
  visible.value = true
  loadLanguages()
  if (appStore.selectedRequest && selectedLang.value) {
    generate()
  }
}

watch(selectedLang, () => {
  if (visible.value && appStore.selectedRequest) generate()
})

defineExpose({ open })
</script>

<template>
  <Button
    icon="pi pi-code"
    label="Generate Code"
    severity="secondary"
    outlined
    size="small"
    class="cg-trigger"
    @click="open"
    :disabled="!appStore.selectedRequest"
  />

  <Dialog
    v-model:visible="visible"
    header="Generate Code"
    modal
    :style="{ width: '640px', maxHeight: '80vh' }"
    :pt="{
      root: { class: 'cg-dialog' },
      header: { class: 'cg-dialog-header' },
      content: { class: 'cg-dialog-content' },
    }"
  >
    <div class="cg-controls">
      <div class="cg-lang-group">
        <label class="cg-label">Language</label>
        <Select
          v-model="selectedLang"
          :options="languages"
          placeholder="Select language"
          class="cg-lang-select"
        />
      </div>
      <div class="cg-actions">
        <Button
          :icon="copied ? 'pi pi-check' : 'pi pi-copy'"
          :label="copied ? 'Copied!' : 'Copy'"
          severity="secondary"
          size="small"
          :disabled="!generatedCode"
          @click="copyToClipboard"
        />
      </div>
    </div>

    <div class="cg-editor-wrap">
      <div v-if="generating" class="cg-loading">
        <i class="pi pi-spin pi-spinner" />
        <span>Generating…</span>
      </div>
      <div v-else-if="!generatedCode" class="cg-empty">
        <i class="pi pi-code" />
        <span>Select a language to generate code</span>
      </div>
      <CodeEditor
        v-else
        :model-value="generatedCode"
        :language="editorLang"
        :read-only="true"
        height="360px"
      />
    </div>
  </Dialog>
</template>

<style scoped>
.cg-trigger {
  font-size: 0.78rem;
  --wails-draggable: no-drag;
}

/* ─── Dialog overrides ─── */
:deep(.cg-dialog) {
  background: var(--app-bg);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

:deep(.cg-dialog-header) {
  background: var(--app-surface);
  border-bottom: 1px solid var(--app-border);
  padding: 0.75rem 1rem;
  font-size: 0.85rem;
  font-weight: 700;
  letter-spacing: 0.02em;
  color: var(--app-text);
}

:deep(.cg-dialog-content) {
  padding: 0;
  background: var(--app-bg);
}

/* ─── Controls ─── */
.cg-controls {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--app-border);
  background: var(--app-surface);
}

.cg-lang-group {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}

.cg-label {
  font-size: 0.68rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--app-muted);
}

.cg-lang-select {
  min-width: 12rem;
}

.cg-actions {
  display: flex;
  gap: 0.5rem;
}

/* ─── Editor area ─── */
.cg-editor-wrap {
  min-height: 360px;
}

.cg-loading,
.cg-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 360px;
  gap: 0.75rem;
  opacity: 0.4;
  user-select: none;
}

.cg-loading .pi,
.cg-empty .pi {
  font-size: 1.5rem;
}

.cg-loading span,
.cg-empty span {
  font-size: 0.8rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}
</style>
