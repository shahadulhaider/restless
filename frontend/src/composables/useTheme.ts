import { ref, watch, onMounted, onUnmounted } from 'vue'
import { loader } from '@guolao/vue-monaco-editor'

type Theme = 'dark' | 'light'

const STORAGE_KEY = 'restless-theme'

const isDark = ref(true)

function applyTheme(dark: boolean) {
  const html = document.documentElement
  if (dark) {
    html.classList.add('p-dark')
  } else {
    html.classList.remove('p-dark')
  }

  loader.config({}).then(() => {
    import('monaco-editor').then((monaco) => {
      monaco.editor.setTheme(dark ? 'vs-dark' : 'vs')
    })
  })
}

function getSystemPreference(): boolean {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

function loadSavedTheme(): boolean {
  const saved = localStorage.getItem(STORAGE_KEY) as Theme | null
  if (saved) return saved === 'dark'
  return getSystemPreference()
}

function toggleTheme() {
  isDark.value = !isDark.value
}

function setTheme(theme: Theme) {
  isDark.value = theme === 'dark'
}

export function useTheme() {
  let mediaQuery: MediaQueryList | null = null
  let systemListener: ((e: MediaQueryListEvent) => void) | null = null

  onMounted(() => {
    isDark.value = loadSavedTheme()
    applyTheme(isDark.value)

    mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    systemListener = (e: MediaQueryListEvent) => {
      if (!localStorage.getItem(STORAGE_KEY)) {
        isDark.value = e.matches
      }
    }
    mediaQuery.addEventListener('change', systemListener)
  })

  onUnmounted(() => {
    if (mediaQuery && systemListener) {
      mediaQuery.removeEventListener('change', systemListener)
    }
  })

  watch(isDark, (dark) => {
    localStorage.setItem(STORAGE_KEY, dark ? 'dark' : 'light')
    applyTheme(dark)
  })

  return { isDark, toggleTheme, setTheme }
}
