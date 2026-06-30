<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import { useAppStore } from '../stores/app'
import {
  List,
  Diff,
} from '../../bindings/github.com/shahadulhaider/restless/internal/gui/historyservice.js'
import type { HistoryEntry } from '../../bindings/github.com/shahadulhaider/restless/internal/history/models.js'

const appStore = useAppStore()

const entries = ref<HistoryEntry[]>([])
const loading = ref(false)
const selectedIndices = ref<Set<number>>(new Set())
const diffResult = ref('')
const showDiff = ref(false)
const diffing = ref(false)

async function loadHistory() {
  if (!appStore.rootDir || !appStore.selectedRequest) return
  loading.value = true
  try {
    const result = await List(appStore.rootDir, appStore.selectedRequest)
    entries.value = result ?? []
  } catch {
    entries.value = []
  } finally {
    loading.value = false
  }
}

function toggleSelect(idx: number) {
  const s = new Set(selectedIndices.value)
  if (s.has(idx)) {
    s.delete(idx)
  } else {
    if (s.size >= 2) {
      // Replace oldest selection
      const first = s.values().next().value
      if (first !== undefined) s.delete(first)
    }
    s.add(idx)
  }
  selectedIndices.value = s
  showDiff.value = false
  diffResult.value = ''
}

function viewResponse(entry: HistoryEntry) {
  if (entry.response) {
    appStore.response = entry.response
  }
}

async function doDiff() {
  const indices = Array.from(selectedIndices.value).sort()
  if (indices.length !== 2) return
  const a = entries.value[indices[0]]
  const b = entries.value[indices[1]]
  if (!a || !b) return
  diffing.value = true
  try {
    diffResult.value = await Diff(a, b)
    showDiff.value = true
  } catch (err) {
    diffResult.value = `Error: ${err}`
    showDiff.value = true
  } finally {
    diffing.value = false
  }
}

function statusColor(code: number): string {
  if (code >= 200 && code < 300) return '#a6e3a1'
  if (code >= 300 && code < 400) return '#89b4fa'
  if (code >= 400 && code < 500) return '#fab387'
  if (code >= 500) return '#f38ba8'
  return 'var(--app-muted)'
}

function formatTimestamp(ts: any): string {
  if (!ts) return '—'
  try {
    const d = new Date(ts)
    if (isNaN(d.getTime())) return String(ts)
    return d.toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  } catch {
    return String(ts)
  }
}

const canDiff = computed(() => selectedIndices.value.size === 2)

watch(() => appStore.selectedRequest, loadHistory)
</script>

<template>
  <div class="hp-root">
    <!-- Header -->
    <div class="hp-header">
      <div class="hp-title">
        <i class="pi pi-history" />
        <span>History</span>
        <span v-if="entries.length" class="hp-count">{{ entries.length }}</span>
      </div>
      <div class="hp-actions">
        <Button
          v-if="canDiff"
          icon="pi pi-arrows-h"
          label="Diff"
          severity="secondary"
          text
          size="small"
          :loading="diffing"
          @click="doDiff"
        />
        <Button
          icon="pi pi-refresh"
          severity="secondary"
          text
          rounded
          size="small"
          :loading="loading"
          @click="loadHistory"
        />
      </div>
    </div>

    <!-- Diff view -->
    <div v-if="showDiff" class="hp-diff-panel">
      <div class="hp-diff-header">
        <span class="hp-diff-title">Response Diff</span>
        <Button
          icon="pi pi-times"
          severity="secondary"
          text
          rounded
          size="small"
          @click="showDiff = false"
        />
      </div>
      <pre class="hp-diff-content">{{ diffResult || 'No differences found' }}</pre>
    </div>

    <!-- Loading -->
    <div v-if="loading && entries.length === 0" class="hp-empty">
      <i class="pi pi-spin pi-spinner" />
      <span>Loading history…</span>
    </div>

    <!-- Empty state -->
    <div v-else-if="entries.length === 0" class="hp-empty">
      <i class="pi pi-history" />
      <span>No history entries</span>
    </div>

    <!-- Entry list -->
    <div v-else class="hp-list">
      <div
        v-for="(entry, idx) in entries"
        :key="idx"
        :class="['hp-entry', { 'hp-entry-selected': selectedIndices.has(idx) }]"
      >
        <div class="hp-entry-select" @click.stop="toggleSelect(idx)">
          <i :class="['pi', selectedIndices.has(idx) ? 'pi-check-square' : 'pi-stop']" />
        </div>
        <div class="hp-entry-body" @click="viewResponse(entry)">
          <div class="hp-entry-top">
            <Tag
              v-if="entry.response"
              :value="String(entry.response.StatusCode)"
              :style="{
                backgroundColor: `${statusColor(entry.response.StatusCode ?? 0)}18`,
                color: statusColor(entry.response.StatusCode ?? 0),
                fontSize: '0.7rem',
                fontWeight: 700,
                padding: '0.1rem 0.4rem',
                borderRadius: '3px',
              }"
            />
            <span class="hp-entry-time">{{ formatTimestamp(entry.timestamp) }}</span>
          </div>
          <div class="hp-entry-meta">
            <span v-if="entry.environment" class="hp-entry-env">
              <i class="pi pi-globe" />
              {{ entry.environment }}
            </span>
            <span v-if="entry.response?.ContentType" class="hp-entry-ct">
              {{ entry.response.ContentType }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.hp-root {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

/* ─── Header ─── */
.hp-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 0.75rem;
  border-bottom: 1px solid var(--app-border);
  background: var(--app-surface);
  flex-shrink: 0;
}

.hp-title {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.73rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--app-muted);
}

.hp-title .pi {
  font-size: 0.8rem;
}

.hp-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.25rem;
  height: 1.1rem;
  padding: 0 0.3rem;
  border-radius: 3px;
  font-size: 0.65rem;
  font-weight: 700;
  background: var(--app-overlay);
  color: var(--app-text);
  font-variant-numeric: tabular-nums;
}

.hp-actions {
  display: flex;
  gap: 0.25rem;
}

/* ─── List ─── */
.hp-list {
  flex: 1;
  overflow-y: auto;
}

.hp-entry {
  display: flex;
  align-items: stretch;
  border-bottom: 1px solid rgba(88, 91, 112, 0.3);
  transition: background 0.12s ease;
}

.hp-entry:hover {
  background: rgba(69, 71, 90, 0.2);
}

.hp-entry-selected {
  background: rgba(137, 180, 250, 0.06);
  border-left: 2px solid #89b4fa;
}

.hp-entry-select {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  cursor: pointer;
  color: var(--app-muted);
  flex-shrink: 0;
  transition: color 0.12s ease;
}

.hp-entry-select:hover {
  color: #89b4fa;
}

.hp-entry-selected .hp-entry-select {
  color: #89b4fa;
}

.hp-entry-body {
  flex: 1;
  padding: 0.5rem 0.6rem 0.5rem 0.25rem;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
}

.hp-entry-top {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.hp-entry-time {
  font-size: 0.73rem;
  color: var(--app-text);
  font-variant-numeric: tabular-nums;
}

.hp-entry-meta {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  font-size: 0.68rem;
  color: var(--app-muted);
}

.hp-entry-env {
  display: inline-flex;
  align-items: center;
  gap: 0.2rem;
}

.hp-entry-env .pi {
  font-size: 0.6rem;
}

.hp-entry-ct {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ─── Diff panel ─── */
.hp-diff-panel {
  display: flex;
  flex-direction: column;
  border-bottom: 1px solid var(--app-border);
  background: rgba(30, 30, 46, 0.8);
  max-height: 50%;
  flex-shrink: 0;
}

.hp-diff-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.4rem 0.75rem;
  border-bottom: 1px solid var(--app-border);
}

.hp-diff-title {
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: #f9e2af;
}

.hp-diff-content {
  flex: 1;
  overflow: auto;
  padding: 0.75rem;
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.73rem;
  line-height: 1.5;
  color: var(--app-text);
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}

/* ─── Empty state ─── */
.hp-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  gap: 0.5rem;
  opacity: 0.4;
  user-select: none;
}

.hp-empty .pi {
  font-size: 1.5rem;
}

.hp-empty span {
  font-size: 0.8rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}
</style>
