/**
 * 行操作下拉菜单的弹出方向测算。
 *
 * 统一替代各列表页里「按行索引猜方向」的启发式（si >= rows.length - 3 …）：
 * 打开菜单时用触发按钮的点击事件实测其在视口中的位置，按下方剩余空间是否放得下菜单
 * 决定向上/向下弹，从而页面滚动到任意位置、菜单任意高度都不会露出屏幕外。
 *
 * 用法（列表页）：
 *   const dropUp = ref(false)
 *   function toggleMenu(id, ev?) {
 *     if (openMenu.value === id) { openMenu.value = null; return }
 *     openMenu.value = id
 *     dropUp.value = shouldDropUp(ev)
 *   }
 *   // 触发按钮： @click.stop="toggleMenu(row.id, $event)"
 *   // 菜单容器： :class="dropUp ? 'bottom-full mb-1.5' : 'top-full mt-1.5'"
 */

// 菜单高度不确定时的兜底估算（px）：够放约 8 个菜单项 + 分隔线 + 内边距。
const FALLBACK_MENU_HEIGHT = 320
// 距视口边缘留白（px）。
const EDGE_GAP = 12

/**
 * 根据触发按钮的位置判断菜单是否应向上弹。
 * 下方剩余空间放不下菜单、且上方比下方宽裕时向上弹；否则向下弹（默认）。
 */
export function shouldDropUp(ev?: MouseEvent, menuHeight = FALLBACK_MENU_HEIGHT): boolean {
  if (!ev) return false
  const btn = (ev.currentTarget as HTMLElement) ?? (ev.target as HTMLElement)
  const rect = btn?.getBoundingClientRect?.()
  if (!rect) return false
  const spaceBelow = window.innerHeight - rect.bottom - EDGE_GAP
  const spaceAbove = rect.top - EDGE_GAP
  return spaceBelow < menuHeight && spaceAbove > spaceBelow
}
