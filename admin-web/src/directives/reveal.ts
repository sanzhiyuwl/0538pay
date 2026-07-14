import type { Directive } from 'vue'

/**
 * v-reveal：滚动进场淡入上移。轻量，基于 IntersectionObserver，不引第三方库。
 * 用法：<div v-reveal>...</div> 或 <div v-reveal="120">（120=延迟 ms）
 * 元素初始透明下移，进入视口后过渡到位；尊重 prefers-reduced-motion。
 */
const reduceMotion =
  typeof window !== 'undefined' &&
  window.matchMedia &&
  window.matchMedia('(prefers-reduced-motion: reduce)').matches

let observer: IntersectionObserver | null = null
function getObserver(): IntersectionObserver {
  if (!observer) {
    observer = new IntersectionObserver(
      (entries) => {
        for (const e of entries) {
          if (e.isIntersecting) {
            const el = e.target as HTMLElement
            const delay = Number(el.dataset.revealDelay || 0)
            window.setTimeout(() => el.classList.add('reveal-in'), delay)
            observer!.unobserve(el)
          }
        }
      },
      { threshold: 0.12, rootMargin: '0px 0px -8% 0px' },
    )
  }
  return observer
}

export const reveal: Directive<HTMLElement, number | undefined> = {
  mounted(el, binding) {
    if (reduceMotion) return
    el.classList.add('reveal')
    if (binding.value) el.dataset.revealDelay = String(binding.value)
    getObserver().observe(el)
  },
  unmounted(el) {
    observer?.unobserve(el)
  },
}
