<script setup lang="ts">
import Splitter from 'primevue/splitter'
import SplitterPanel from 'primevue/splitterpanel'
</script>

<template>
  <div class="app-shell">
    <Splitter class="app-splitter">
      <SplitterPanel class="panel-sidebar" :size="25" :minSize="15">
        <div class="panel-placeholder">
          <i class="pi pi-folder-open panel-icon" />
          <span class="panel-label">Collection Browser</span>
        </div>
      </SplitterPanel>
      <SplitterPanel class="panel-main" :size="75" :minSize="40">
        <div class="panel-placeholder">
          <i class="pi pi-send panel-icon" />
          <span class="panel-label">Request / Response</span>
        </div>
      </SplitterPanel>
    </Splitter>
  </div>
</template>

<style scoped>
.app-shell {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}

.app-splitter {
  flex: 1;
  border: none;
  border-radius: 0;
  background: transparent;
}

.panel-sidebar {
  background: var(--surface-ground);
  border-right: 1px solid var(--surface-border);
}

.panel-main {
  background: var(--surface-section);
}

.panel-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 0.75rem;
  opacity: 0.4;
  user-select: none;
}

.panel-icon {
  font-size: 2rem;
}

.panel-label {
  font-size: 0.85rem;
  font-weight: 500;
  letter-spacing: 0.03em;
  text-transform: uppercase;
}
</style>
