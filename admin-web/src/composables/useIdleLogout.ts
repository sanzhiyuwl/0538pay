/**
 * 管理后台闲置超时自动退出（安全增强，epay 无此机制）。
 *
 * 语义：一段时间内无任何用户操作（鼠标/键盘/滚动/触摸）即视为闲置，
 * 到达闲置阈值后强制登出、跳登录页。到期前 WARN_BEFORE_MS 弹出倒计时警告，
 * 期间任意操作即撤销警告并重新计时。
 *
 * 挂在 AdminLayout 这一层，覆盖所有能进入管理后台的角色，无需逐角色处理。
 * 后端 JWT 的绝对过期（默认 72h）作为服务端兜底，二者互补。
 *
 * 用法（在 AdminLayout setup 内）：
 *   const { warning, remaining, stay } = useIdleLogout()
 */
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

// 闲置阈值：15 分钟无操作自动退出。
const IDLE_LIMIT_MS = 15 * 60 * 1000
// 到期前多久弹出倒计时警告。
const WARN_BEFORE_MS = 60 * 1000
// 倒计时刷新间隔。
const TICK_MS = 1000

// 触发「有操作」的事件：鼠标移动/点击、键盘、滚动、触摸。
const ACTIVITY_EVENTS: (keyof WindowEventMap)[] = [
  'mousemove',
  'mousedown',
  'keydown',
  'scroll',
  'touchstart',
  'wheel',
]

/** 各端接入配置：登出动作 + 到期跳转的登录路由名（两端各不相同）。 */
interface IdleLogoutOptions {
  logout: () => void
  loginRouteName: string
}

export function useIdleLogout(opts: IdleLogoutOptions) {
  const router = useRouter()

  // 是否处于「即将超时」警告态。
  const warning = ref(false)
  // 警告态下剩余秒数（用于弹窗倒计时展示）。
  const remaining = ref(Math.ceil(WARN_BEFORE_MS / 1000))

  let idleTimer: number | undefined
  let warnTimer: number | undefined
  let tickTimer: number | undefined
  // 节流：activity 高频事件仅在超过间隔后才重排定时器。
  let lastActivity = 0

  function clearTimers() {
    if (idleTimer) window.clearTimeout(idleTimer)
    if (warnTimer) window.clearTimeout(warnTimer)
    if (tickTimer) window.clearInterval(tickTimer)
    idleTimer = warnTimer = tickTimer = undefined
  }

  // 到期执行：登出并跳登录页，带 reason=timeout 供登录页提示。
  function doLogout() {
    clearTimers()
    stopListening()
    warning.value = false
    opts.logout()
    router.replace({ name: opts.loginRouteName, query: { reason: 'timeout' } })
  }

  // 进入警告态：开始倒计时，WARN_BEFORE_MS 后真正登出。
  function enterWarning() {
    warning.value = true
    remaining.value = Math.ceil(WARN_BEFORE_MS / 1000)
    tickTimer = window.setInterval(() => {
      remaining.value -= 1
      if (remaining.value <= 0) remaining.value = 0
    }, TICK_MS)
    warnTimer = window.setTimeout(doLogout, WARN_BEFORE_MS)
  }

  // 重排计时：清掉旧定时器，重新从满额闲置阈值开始。
  function resetTimers() {
    clearTimers()
    warning.value = false
    // 距警告弹出还有 (IDLE_LIMIT - WARN_BEFORE)；再过 WARN_BEFORE 登出。
    idleTimer = window.setTimeout(enterWarning, IDLE_LIMIT_MS - WARN_BEFORE_MS)
  }

  // 用户操作回调：警告态下任意操作按节流跳过，避免误撤销弹窗（需显式点「继续」）。
  function onActivity() {
    if (warning.value) return // 警告态只认弹窗按钮，不被背景滚动等误撤销
    const now = Date.now()
    if (now - lastActivity < TICK_MS) return // 节流
    lastActivity = now
    resetTimers()
  }

  // 弹窗「继续操作」按钮：撤销警告，重新计时。
  function stay() {
    resetTimers()
  }

  // 弹窗「立即退出」按钮。
  function logoutNow() {
    doLogout()
  }

  function startListening() {
    ACTIVITY_EVENTS.forEach((ev) =>
      window.addEventListener(ev, onActivity, { passive: true }),
    )
  }
  function stopListening() {
    ACTIVITY_EVENTS.forEach((ev) => window.removeEventListener(ev, onActivity))
  }

  onMounted(() => {
    startListening()
    resetTimers()
  })
  onBeforeUnmount(() => {
    clearTimers()
    stopListening()
  })

  return { warning, remaining, stay, logoutNow, idleLimitMinutes: IDLE_LIMIT_MS / 60000 }
}
