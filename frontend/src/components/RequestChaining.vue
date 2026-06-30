<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Button from 'primevue/button'
import { GetChainVariables } from '../../bindings/github.com/shahadulhaider/restless/internal/gui/requestservice.js'

const chainVars = ref<string[]>([])
const loading = ref(false)
const expanded = ref(true)

async function loadChainVars() {
  loading.value = true
  try {
    const vars = await GetChainVariables()
    chainVars.value = vars ?? []
  } catch {
    chainVars.value = []
  } finally {
    loading.value = false
  }
}

function chainSyntax(name: string): string {
  return '{{' + name + '.response.body.$}}'
}

async function copySyntax(name: string) {
  try {
    await navigator.clipboard.writeText(chainSyntax(name))
  } catch { /* clipboard unavailable */ }
}

onMounted(loadChainVars)
</script>

<template>
  <div class="rc-root">
    <!-- Header -->
    <div class="rc-header" @click="expanded = !expanded">
      <div class="rc-title">
        <i class="pi pi-link" />
        <span>Chain Variables</span>
        <span v-if="chainVars.length" class="rc-count">{{ chainVars.length }}</span>
      </div>
      <div class="rc-header-actions">
        <Button
          icon="pi pi-refresh"
          severity="secondary"
          text
          rounded
          size="small"
          :loading="loading"
          @click.stop="loadChainVars"
        />
        <i :class="['pi', expanded ? 'pi-chevron-up' : 'pi-chevron-down', 'rc-chevron']" />
      </div>
    </div>

    <div v-if="expanded" class="rc-body">
      <!-- Empty -->
      <div v-if="chainVars.length === 0 && !loading" class="rc-empty">
        <i class="pi pi-info-circle" />
        <span>No chain variables available</span>
        <span class="rc-empty-hint">Send requests to populate the chain context</span>
      </div>

      <!-- Loading -->
      <div v-else-if="loading && chainVars.length === 0" class="rc-empty">
        <i class="pi pi-spin pi-spinner" />
        <span>Loading…</span>
      </div>

      <!-- Variable list -->
      <div v-else class="rc-var-list">
        <div class="rc-var-hint">
          <i class="pi pi-lightbulb" />
          Click a variable to copy its chaining syntax
        </div>
        <div
          v-for="name in chainVars"
          :key="name"
          class="rc-var-item"
          @click="copySyntax(name)"
        >
          <div class="rc-var-name">
            <i class="pi pi-bolt rc-var-icon" />
            <span>{{ name }}</span>
          </div>
          <code class="rc-var-syntax">{{ chainSyntax(name) }}</code>
        </div>
      </div>

      <!-- Run chain placeholder -->
      <div class="rc-footer">
        <Button
          icon="pi pi-play"
          label="Run Chain"
          severity="secondary"
          outlined
          size="small"
          disabled
          class="rc-run-btn"
          v-tooltip.top="'Chain execution — coming soon'"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.rc-root {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--app-border);
  border-radius: 6px;
  background: var(--app-surface);
  overflow: hidden;
}

/* ─── Header ─── */
.rc-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 0.75rem;
  cursor: pointer;
  user-select: none;
  transition: background 0.12s ease;
}

.rc-header:hover {
  background: rgba(69, 71, 90, 0.3);
}

.rc-title {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.73rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--app-muted);
}

.rc-title .pi {
  font-size: 0.8rem;
  color: #cba6f7;
}

.rc-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.25rem;
  height: 1.1rem;
  padding: 0 0.3rem;
  border-radius: 3px;
  font-size: 0.65rem;
  font-weight: 700;
  background: rgba(203, 166, 247, 0.15);
  color: #cba6f7;
  font-variant-numeric: tabular-nums;
}

.rc-header-actions {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.rc-chevron {
  font-size: 0.65rem;
  color: var(--app-muted);
  transition: transform 0.2s ease;
}

/* ─── Body ─── */
.rc-body {
  display: flex;
  flex-direction: column;
  border-top: 1px solid var(--app-border);
}

/* ─── Variable list ─── */
.rc-var-list {
  display: flex;
  flex-direction: column;
  max-height: 300px;
  overflow-y: auto;
}

.rc-var-hint {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.4rem 0.75rem;
  font-size: 0.68rem;
  color: var(--app-muted);
  border-bottom: 1px solid rgba(88, 91, 112, 0.3);
}

.rc-var-hint .pi {
  font-size: 0.7rem;
  color: #f9e2af;
}

.rc-var-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.45rem 0.75rem;
  cursor: pointer;
  border-bottom: 1px solid rgba(88, 91, 112, 0.2);
  transition: background 0.12s ease;
}

.rc-var-item:last-child {
  border-bottom: none;
}

.rc-var-item:hover {
  background: rgba(69, 71, 90, 0.25);
}

.rc-var-name {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--app-text);
}

.rc-var-icon {
  font-size: 0.65rem;
  color: #a6e3a1;
}

.rc-var-syntax {
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.68rem;
  color: var(--app-muted);
  background: rgba(69, 71, 90, 0.3);
  padding: 0.15rem 0.4rem;
  border-radius: 3px;
  max-width: 55%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ─── Footer ─── */
.rc-footer {
  display: flex;
  justify-content: center;
  padding: 0.6rem 0.75rem;
  border-top: 1px solid var(--app-border);
}

.rc-run-btn {
  font-size: 0.75rem;
  width: 100%;
}

/* ─── Empty ─── */
.rc-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
  gap: 0.35rem;
  opacity: 0.5;
  user-select: none;
}

.rc-empty .pi {
  font-size: 1.25rem;
}

.rc-empty span {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.rc-empty-hint {
  font-size: 0.68rem !important;
  text-transform: none !important;
  color: var(--app-muted);
  opacity: 0.7;
}
</style>
