<script setup lang="ts">
import { computed } from 'vue'
import Splitter from 'primevue/splitter'
import SplitterPanel from 'primevue/splitterpanel'
import CollectionBrowser from '../components/CollectionBrowser.vue'
import RequestEditor from '../components/RequestEditor.vue'
import ResponseViewer from '../components/ResponseViewer.vue'
import { useAppStore } from '../stores/app'

const appStore = useAppStore()

const envLabel = computed(() => appStore.activeEnv || 'No Environment')

const lastResponseTime = computed(() => {
  const timing = appStore.response?.Timing
  if (!timing) return null
  const totalNs = timing.Total as number
  if (!totalNs) return null
  const ms = totalNs / 1_000_000
  return ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${Math.round(ms)}ms`
})
</script>

<template>
  <div class="app-shell">
    <!-- ── Top Bar ── -->
    <header class="top-bar">
      <div class="top-bar-left">
        <i class="pi pi-bolt app-icon" />
        <span class="app-title">Restless</span>
      </div>
      <div class="top-bar-right">
        <span class="env-badge">
          <i class="pi pi-globe env-icon" />
          {{ envLabel }}
        </span>
      </div>
    </header>

    <!-- ── Main Splitter Area ── -->
    <Splitter class="app-splitter">
      <SplitterPanel class="panel-sidebar" :size="25" :minSize="15">
        <CollectionBrowser />
      </SplitterPanel>
      <SplitterPanel class="panel-main" :size="75" :minSize="40">
        <Splitter layout="vertical" class="main-vertical-splitter">
          <SplitterPanel class="panel-editor" :size="50" :minSize="20">
            <RequestEditor />
          </SplitterPanel>
          <SplitterPanel class="panel-viewer" :size="50" :minSize="20">
            <ResponseViewer />
          </SplitterPanel>
        </Splitter>
      </SplitterPanel>
    </Splitter>

    <!-- ── Status Bar ── -->
    <footer class="status-bar">
      <div class="status-left">
        <i class="pi pi-server status-icon" />
        <span class="status-env">{{ envLabel }}</span>
      </div>
      <div class="status-right">
        <template v-if="lastResponseTime">
          <i class="pi pi-clock status-icon" />
          <span class="status-time">{{ lastResponseTime }}</span>
        </template>
        <template v-if="appStore.loading">
          <i class="pi pi-spin pi-spinner status-icon" />
          <span class="status-time">Sending...</span>
        </template>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.app-shell {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}

/* ── Top Bar ── */
.top-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 40px;
  min-height: 40px;
  padding: 0 0.875rem;
  background: color-mix(in srgb, var(--app-bg) 85%, black);
  border-bottom: 1px solid var(--app-border);
  --wails-draggable: drag;
  user-select: none;
}

.top-bar-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.app-icon {
  font-size: 0.95rem;
  color: #89b4fa;
  --wails-draggable: no-drag;
}

.app-title {
  font-size: 0.82rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--app-text);
  opacity: 0.85;
}

.top-bar-right {
  display: flex;
  align-items: center;
  --wails-draggable: no-drag;
}

.env-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.2rem 0.625rem;
  border-radius: 4px;
  background: var(--app-overlay);
  font-size: 0.7rem;
  font-weight: 500;
  letter-spacing: 0.02em;
  color: var(--app-muted);
}

.env-icon {
  font-size: 0.62rem;
}

/* ── Main Splitter ── */
.app-splitter {
  flex: 1;
  min-height: 0;
  border: none;
  border-radius: 0;
  background: transparent;
}

.panel-sidebar {
  background: var(--app-surface);
  border-right: 1px solid var(--app-border);
  overflow: hidden;
}

.panel-main {
  background: var(--app-bg);
  overflow: hidden;
}

.main-vertical-splitter {
  height: 100%;
  border: none;
  border-radius: 0;
  background: transparent;
}

.panel-editor {
  background: var(--app-bg);
  overflow: hidden;
}

.panel-viewer {
  background: var(--app-bg);
  border-top: 1px solid var(--app-border);
  overflow: hidden;
}

/* ── Splitter gutter overrides ── */
.app-splitter :deep(.p-splitter-gutter),
.main-vertical-splitter :deep(.p-splitter-gutter) {
  background: transparent;
  transition: background 0.15s ease;
}

.app-splitter :deep(.p-splitter-gutter:hover),
.main-vertical-splitter :deep(.p-splitter-gutter:hover) {
  background: rgba(137, 180, 250, 0.12);
}

.app-splitter :deep(.p-splitter-gutter-handle),
.main-vertical-splitter :deep(.p-splitter-gutter-handle) {
  background: var(--app-overlay);
  border-radius: 2px;
}

/* ── Status Bar ── */
.status-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 24px;
  min-height: 24px;
  padding: 0 0.75rem;
  background: color-mix(in srgb, var(--app-bg) 85%, black);
  border-top: 1px solid var(--app-border);
  user-select: none;
}

.status-left,
.status-right {
  display: flex;
  align-items: center;
  gap: 0.375rem;
}

.status-icon {
  font-size: 0.58rem;
  color: var(--app-muted);
}

.status-env {
  font-size: 0.65rem;
  font-weight: 500;
  color: var(--app-muted);
  letter-spacing: 0.02em;
}

.status-time {
  font-size: 0.65rem;
  font-weight: 600;
  color: #a6e3a1;
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace;
  letter-spacing: 0.02em;
}
</style>
