import { ref } from 'vue'

type Theme = 'light' | 'dark'
const STORAGE_KEY = 'petro-price-theme'

const theme = ref<Theme>((localStorage.getItem(STORAGE_KEY) as Theme | null) ?? 'light')

function apply(value: Theme): void {
  document.documentElement.setAttribute('data-theme', value)
}

apply(theme.value)

export function useTheme() {
  function toggle(): void {
    theme.value = theme.value === 'light' ? 'dark' : 'light'
    localStorage.setItem(STORAGE_KEY, theme.value)
    apply(theme.value)
  }
  return { theme, toggle }
}
