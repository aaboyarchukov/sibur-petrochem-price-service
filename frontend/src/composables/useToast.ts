import { ref } from 'vue'

const message = ref('')
let timer: ReturnType<typeof setTimeout> | undefined

export function useToast() {
  function show(text: string): void {
    message.value = text
    clearTimeout(timer)
    timer = setTimeout(() => (message.value = ''), 2600)
  }
  return { message, show }
}
