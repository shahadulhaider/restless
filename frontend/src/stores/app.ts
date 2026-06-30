import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAppStore = defineStore('app', () => {
  const selectedRequest = ref<any>(null)
  const activeEnv = ref('')
  const response = ref<any>(null)
  const loading = ref(false)
  const rootDir = ref('')

  return {
    selectedRequest,
    activeEnv,
    response,
    loading,
    rootDir,
  }
})
