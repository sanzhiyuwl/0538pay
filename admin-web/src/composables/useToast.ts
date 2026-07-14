/**
 * 全局轻提示（Toast）。任意组件里：
 *   import { useToast } from '@/composables/useToast'
 *   const toast = useToast()
 *   toast.success('已保存')
 * 由 <ToastHost /> 统一渲染（挂在 App.vue）。
 */
import { ref } from 'vue'

export type ToastType = 'success' | 'error' | 'info'

export interface ToastItem {
  id: number
  type: ToastType
  message: string
}

// 模块级单例：全站共享同一队列
const toasts = ref<ToastItem[]>([])
let seq = 0

function push(type: ToastType, message: string, duration = 2600) {
  const id = ++seq
  toasts.value.push({ id, type, message })
  window.setTimeout(() => dismiss(id), duration)
}

function dismiss(id: number) {
  const i = toasts.value.findIndex((t) => t.id === id)
  if (i !== -1) toasts.value.splice(i, 1)
}

export function useToast() {
  return {
    toasts,
    dismiss,
    success: (msg: string, duration?: number) => push('success', msg, duration),
    error: (msg: string, duration?: number) => push('error', msg, duration),
    info: (msg: string, duration?: number) => push('info', msg, duration),
  }
}
