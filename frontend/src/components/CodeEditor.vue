<script setup lang="ts">
import { computed } from 'vue'
import { VueMonacoEditor } from '@guolao/vue-monaco-editor'

const props = withDefaults(
  defineProps<{
    modelValue: string
    language?: string
    readOnly?: boolean
    height?: string
  }>(),
  {
    language: 'json',
    readOnly: false,
    height: '100%',
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const editorOptions = computed(() => ({
  minimap: { enabled: false },
  wordWrap: 'on' as const,
  fontSize: 13,
  lineNumbers: 'on' as const,
  readOnly: props.readOnly,
  scrollBeyondLastLine: false,
  automaticLayout: true,
  tabSize: 2,
  renderLineHighlight: 'line' as const,
  padding: { top: 8, bottom: 8 },
  scrollbar: {
    verticalScrollbarSize: 6,
    horizontalScrollbarSize: 6,
  },
}))

function handleChange(value: string | undefined) {
  emit('update:modelValue', value ?? '')
}
</script>

<template>
  <VueMonacoEditor
    :value="modelValue"
    :language="language"
    theme="vs-dark"
    :height="height"
    :options="editorOptions"
    @change="handleChange"
  />
</template>
