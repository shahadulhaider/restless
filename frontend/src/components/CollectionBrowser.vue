<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import Tree from 'primevue/tree'
import InputText from 'primevue/inputtext'
import ContextMenu from 'primevue/contextmenu'
import { CollectionService } from '../../bindings/github.com/shahadulhaider/restless/internal/gui'
import { GetRootDir } from '../../bindings/github.com/shahadulhaider/restless/cmd/restless-app/greetservice'
import type { Collection } from '../../bindings/github.com/shahadulhaider/restless/internal/model/models'
import { useAppStore } from '../stores/app'

const appStore = useAppStore()

const treeNodes = ref<any[]>([])
const selectedKeys = ref<Record<string, boolean>>({})
const expandedKeys = ref<Record<string, boolean>>({})
const searchQuery = ref('')
const contextNode = ref<any>(null)
const menu = ref()

// Catppuccin Mocha method badge palette
const METHOD_COLORS: Record<string, string> = {
  GET: '#a6e3a1',
  POST: '#89b4fa',
  PUT: '#fab387',
  DELETE: '#f38ba8',
  PATCH: '#f9e2af',
  HEAD: '#cba6f7',
  OPTIONS: '#94e2d5',
}

function methodColor(method: string): string {
  return METHOD_COLORS[method.toUpperCase()] || '#cdd6f4'
}

// --- Tree building ---

function buildTree(collection: Collection): any[] {
  const rootDir = collection.RootDir || ''
  const files = collection.Files || []
  const dirMap = new Map<string, any>()
  const rootChildren: any[] = []

  for (const file of files) {
    let relPath = file.Path
    if (relPath.startsWith(rootDir)) {
      relPath = relPath.slice(rootDir.length)
    }
    if (relPath.startsWith('/') || relPath.startsWith('\\')) {
      relPath = relPath.slice(1)
    }

    const parts = relPath.split(/[/\\]/)
    const fileName = parts[parts.length - 1]
    const dirParts = parts.slice(0, -1)

    let currentChildren = rootChildren
    let currentPath = ''

    for (const dir of dirParts) {
      currentPath += (currentPath ? '/' : '') + dir
      if (!dirMap.has(currentPath)) {
        const dirNode = {
          key: 'dir:' + currentPath,
          label: dir,
          type: 'directory',
          children: [] as any[],
          data: { path: currentPath },
        }
        dirMap.set(currentPath, dirNode)
        currentChildren.push(dirNode)
      }
      currentChildren = dirMap.get(currentPath)!.children
    }

    const requests = (file.Requests || []).map((req: any, idx: number) => ({
      key: 'req:' + relPath + ':' + idx,
      label: req.Name || req.URL || 'Untitled',
      type: 'request',
      leaf: true,
      data: { request: req, file },
    }))

    currentChildren.push({
      key: 'file:' + relPath,
      label: fileName,
      type: 'file',
      children: requests,
      data: { file, relPath },
    })
  }

  sortNodes(rootChildren)
  return rootChildren
}

function sortNodes(nodes: any[]) {
  const order: Record<string, number> = { directory: 0, file: 1, request: 2 }
  nodes.sort((a: any, b: any) => {
    const ao = order[a.type] ?? 3
    const bo = order[b.type] ?? 3
    return ao !== bo ? ao - bo : a.label.localeCompare(b.label)
  })
  for (const n of nodes) {
    if (n.children) sortNodes(n.children)
  }
}

// --- Filtering ---

function filterNodes(nodes: any[], q: string): any[] {
  if (!q) return nodes
  const lower = q.toLowerCase()
  return nodes.reduce<any[]>((acc, node) => {
    const match = node.label.toLowerCase().includes(lower)
    const kids = node.children ? filterNodes(node.children, q) : []
    if (match || kids.length > 0) {
      acc.push({ ...node, children: match ? node.children : kids.length ? kids : undefined })
    }
    return acc
  }, [])
}

const filteredNodes = computed(() => filterNodes(treeNodes.value, searchQuery.value))

watch(searchQuery, (q) => {
  if (q) expandAll(filteredNodes.value)
})

function expandAll(nodes: any[]) {
  for (const n of nodes) {
    if (n.children?.length) {
      expandedKeys.value[n.key] = true
      expandAll(n.children)
    }
  }
}

// --- Collection loading ---

async function loadCollection() {
  if (!appStore.rootDir) return
  try {
    const collection = await CollectionService.LoadCollection(appStore.rootDir)
    if (collection) {
      treeNodes.value = buildTree(collection)
      expandAll(treeNodes.value)
    }
  } catch (e) {
    console.error('Failed to load collection:', e)
  }
}

// --- Selection ---

function onNodeSelect(node: any) {
  if (node.type === 'request' && node.data?.request) {
    appStore.selectedRequest = node.data.request
  }
}

// --- Context menu ---

const menuItems = computed(() => {
  const node = contextNode.value
  if (!node) return []

  if (node.type === 'directory') {
    return [
      { label: 'New File', icon: 'pi pi-file-plus', command: () => createFile(node) },
      { label: 'New Folder', icon: 'pi pi-folder-plus', command: () => createFolder(node) },
      { separator: true },
      { label: 'Rename', icon: 'pi pi-pencil', command: () => renameEntry(node) },
      { label: 'Delete', icon: 'pi pi-trash', command: () => deleteEntry(node) },
    ]
  }

  if (node.type === 'file') {
    return [
      { label: 'Rename', icon: 'pi pi-pencil', command: () => renameEntry(node) },
      { label: 'Delete', icon: 'pi pi-trash', command: () => deleteEntry(node) },
    ]
  }

  if (node.type === 'request') {
    return [
      { label: 'Duplicate', icon: 'pi pi-copy', command: () => duplicateRequest(node) },
    ]
  }

  return []
})

function onContextMenu(event: MouseEvent, node: any) {
  contextNode.value = node
  menu.value.show(event)
}

// --- Context menu actions ---

async function createFile(node: any) {
  const name = prompt('File name (.http):')
  if (!name) return
  const fileName = name.endsWith('.http') ? name : name + '.http'
  const relPath = node.data.path ? node.data.path + '/' + fileName : fileName
  try {
    await CollectionService.CreateFile(appStore.rootDir, relPath)
    await loadCollection()
  } catch (e) {
    console.error('Create file failed:', e)
  }
}

async function createFolder(node: any) {
  const name = prompt('Folder name:')
  if (!name) return
  const path = node.data.path ? node.data.path + '/' + name : name
  try {
    await CollectionService.CreateDirectory(appStore.rootDir, path)
    await loadCollection()
  } catch (e) {
    console.error('Create folder failed:', e)
  }
}

async function renameEntry(node: any) {
  const newName = prompt('New name:', node.label)
  if (!newName || newName === node.label) return
  const oldRel = node.data.relPath || node.data.path
  const parts = oldRel.split('/')
  parts[parts.length - 1] = newName
  try {
    await CollectionService.RenameEntry(appStore.rootDir, oldRel, parts.join('/'))
    await loadCollection()
  } catch (e) {
    console.error('Rename failed:', e)
  }
}

async function deleteEntry(node: any) {
  if (!confirm(`Delete "${node.label}"?`)) return
  try {
    await CollectionService.DeleteEntry(appStore.rootDir, node.data.relPath || node.data.path)
    await loadCollection()
  } catch (e) {
    console.error('Delete failed:', e)
  }
}

function duplicateRequest(_node: any) {
  // TODO: placeholder for future implementation
}

// --- Lifecycle ---

onMounted(async () => {
  try {
    const dir = await GetRootDir()
    if (dir) {
      appStore.rootDir = dir
      await loadCollection()
    }
  } catch (e) {
    console.error('Failed to init collection browser:', e)
  }
})
</script>

<template>
  <div class="collection-browser">
    <div class="browser-header">
      <div class="search-wrap">
        <i class="pi pi-search search-icon" />
        <InputText
          v-model="searchQuery"
          placeholder="Filter..."
          class="search-input"
          size="small"
        />
      </div>
    </div>

    <div class="browser-tree">
      <Tree
        v-if="filteredNodes.length > 0"
        :value="filteredNodes"
        v-model:selectionKeys="selectedKeys"
        v-model:expandedKeys="expandedKeys"
        selectionMode="single"
        :metaKeySelection="false"
        class="collection-tree"
        @nodeSelect="onNodeSelect"
      >
        <template #directory="{ node }">
          <span class="node-content" @contextmenu.prevent="onContextMenu($event, node)">
            <i class="pi pi-folder node-icon node-icon-dir" />
            <span class="node-label">{{ node.label }}</span>
          </span>
        </template>

        <template #file="{ node }">
          <span class="node-content" @contextmenu.prevent="onContextMenu($event, node)">
            <i class="pi pi-file node-icon node-icon-file" />
            <span class="node-label">{{ node.label }}</span>
          </span>
        </template>

        <template #request="{ node }">
          <span class="node-content" @contextmenu.prevent="onContextMenu($event, node)">
            <span
              class="method-badge"
              :style="{ backgroundColor: methodColor(node.data.request.Method) }"
            >
              {{ node.data.request.Method }}
            </span>
            <span class="node-label">{{ node.label }}</span>
          </span>
        </template>

        <template #default="{ node }">
          <span class="node-content">
            <span class="node-label">{{ node.label }}</span>
          </span>
        </template>
      </Tree>

      <div v-else class="empty-state">
        <i class="pi pi-inbox empty-icon" />
        <span class="empty-text">
          {{ searchQuery ? 'No matches' : 'No collection loaded' }}
        </span>
      </div>
    </div>

    <ContextMenu ref="menu" :model="menuItems" class="browser-context-menu" />
  </div>
</template>

<style scoped>
.collection-browser {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.browser-header {
  padding: 0.625rem 0.75rem;
  border-bottom: 1px solid var(--app-border);
  flex-shrink: 0;
}

.search-wrap {
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 0.625rem;
  color: var(--app-muted);
  font-size: 0.75rem;
  pointer-events: none;
  z-index: 1;
}

.search-input {
  width: 100%;
  padding-left: 2rem !important;
  font-size: 0.8125rem !important;
  background: var(--app-surface) !important;
  border-color: transparent !important;
  color: var(--app-text) !important;
  border-radius: 6px !important;
  height: 1.875rem !important;
}

.search-input:focus {
  border-color: var(--app-overlay) !important;
  box-shadow: none !important;
}

.browser-tree {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 0.25rem 0;
}

/* PrimeVue Tree overrides */
.collection-tree {
  background: transparent !important;
  border: none !important;
  padding: 0 !important;
  color: var(--app-text) !important;
  font-size: 0.8125rem !important;
}

:deep(.p-tree-root) {
  padding: 0 0.25rem;
}

:deep(.p-tree-node) {
  padding: 0;
}

:deep(.p-tree-node-content) {
  padding: 0.1875rem 0.5rem;
  border-radius: 4px;
  gap: 0.25rem;
  transition: background 0.12s ease;
  cursor: pointer;
}

:deep(.p-tree-node-content:hover) {
  background: var(--app-surface) !important;
}

:deep(.p-tree-node-content.p-tree-node-selected) {
  background: var(--app-overlay) !important;
  color: var(--app-text) !important;
}

:deep(.p-tree-node-content:focus) {
  box-shadow: none !important;
  outline: none !important;
}

:deep(.p-tree-node-toggle-button) {
  color: var(--app-muted) !important;
  width: 1.125rem !important;
  height: 1.125rem !important;
  margin-right: 0.125rem;
}

:deep(.p-tree-node-toggle-button:hover) {
  color: var(--app-text) !important;
  background: transparent !important;
}

:deep(.p-tree-node-label) {
  display: flex;
  flex: 1;
  min-width: 0;
}

:deep(.p-tree-node-children) {
  padding-left: 0.75rem;
}

/* Node content */
.node-content {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  flex: 1;
  min-width: 0;
  user-select: none;
  --wails-draggable: no-drag;
}

.node-icon {
  font-size: 0.8125rem;
  flex-shrink: 0;
  opacity: 0.7;
}

.node-icon-dir {
  color: #89b4fa;
}

.node-icon-file {
  color: #a6adc8;
}

.node-label {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.4;
}

/* Method badges */
.method-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 0.5625rem;
  font-weight: 700;
  font-family: ui-monospace, 'SF Mono', 'Cascadia Code', 'Fira Code', Menlo, monospace;
  letter-spacing: 0.02em;
  padding: 0.0625rem 0.3125rem;
  border-radius: 3px;
  color: #1e1e2e;
  min-width: 2rem;
  text-align: center;
  flex-shrink: 0;
  line-height: 1.3;
}

/* Empty state */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 0.625rem;
  opacity: 0.35;
  user-select: none;
}

.empty-icon {
  font-size: 1.75rem;
}

.empty-text {
  font-size: 0.8125rem;
  letter-spacing: 0.02em;
}
</style>
