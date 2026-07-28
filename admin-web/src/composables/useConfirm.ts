/**
 * 全局确认弹窗（替代原生 window.confirm，统一收单同款 Modal 视觉）。任意组件里：
 *   import { useConfirm } from '@/composables/useConfirm'
 *   const confirm = useConfirm()
 *   if (!(await confirm('确认删除？', { title: '删除', danger: true }))) return
 * 由 <ConfirmHost /> 统一渲染（挂在 App.vue）。返回 Promise<boolean>：确定 true / 取消 false。
 */
import { ref } from 'vue'

export interface ConfirmOptions {
  title?: string // 弹窗标题，默认「请确认」
  confirmText?: string // 确定按钮文案，默认「确定」
  cancelText?: string // 取消按钮文案，默认「取消」
  danger?: boolean // 确定按钮是否红色（删除/退款等不可逆操作）
}

export interface ConfirmState extends Required<Omit<ConfirmOptions, 'danger'>> {
  message: string
  danger: boolean
  resolve: (v: boolean) => void
}

// 模块级单例：全站共享同一个确认弹窗
const state = ref<ConfirmState | null>(null)

function open(message: string, opts: ConfirmOptions = {}): Promise<boolean> {
  // 若已有一个未决的确认，先当作取消收掉，避免叠加
  if (state.value) state.value.resolve(false)
  return new Promise<boolean>((resolve) => {
    state.value = {
      message,
      title: opts.title ?? '请确认',
      confirmText: opts.confirmText ?? '确定',
      cancelText: opts.cancelText ?? '取消',
      danger: opts.danger ?? false,
      resolve,
    }
  })
}

function settle(v: boolean) {
  if (!state.value) return
  state.value.resolve(v)
  state.value = null
}

export function useConfirm() {
  return Object.assign(open, { state, settle })
}
