<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { SiteContent } from '@/lib/mock/site-content'
import { fmtMetric } from '../shared'

const props = defineProps<{ content: SiteContent; preview?: boolean }>()

// 数据背书（数字滚动）：预览态直接显示目标值（缩放容器里 IO 观察不到进场）
const counts = ref(props.content.metrics.map((m) => (props.preview ? m.target : 0)))
const metricsEl = ref<HTMLElement | null>(null)
onMounted(() => {
  if (props.preview || !metricsEl.value) return
  const io = new IntersectionObserver((es) => {
    for (const e of es) {
      if (e.isIntersecting) {
        const dur = 1200
        const start = performance.now()
        const targets = props.content.metrics.map((m) => m.target)
        const step = (now: number) => {
          const p = Math.min(1, (now - start) / dur)
          const eased = 1 - Math.pow(1 - p, 3)
          counts.value = targets.map((t) => t * eased)
          if (p < 1) requestAnimationFrame(step)
          else counts.value = targets
        }
        requestAnimationFrame(step)
        io.disconnect()
      }
    }
  }, { threshold: 0.4 })
  io.observe(metricsEl.value)
})
</script>

<template>
  <!-- ===== 数据背书（深色背景大图 + 暗色遮罩 + 白色数字滚动）===== -->
  <section ref="metricsEl" class="relative overflow-hidden border-b border-border">
    <!-- 背景图 + 暗色遮罩 -->
    <div class="pointer-events-none absolute inset-0 z-0" aria-hidden="true">
      <img src="/assets/metrics-bg2.jpg" alt="" class="size-full object-cover" />
      <div class="absolute inset-0 bg-black/15" />
    </div>
    <div class="relative z-10 mx-auto grid max-w-7xl grid-cols-2 divide-x divide-white/15 px-4 py-16 lg:grid-cols-4 lg:px-6">
      <div v-for="(m, i) in content.metrics" :key="m.label" class="px-4 text-center">
        <div class="text-3xl font-bold tabular-nums text-white sm:text-4xl">
          <span v-if="m.prefix" class="text-2xl text-white/70">{{ m.prefix }}</span>{{ fmtMetric(counts[i], m.decimals) }}<span class="text-xl text-white/70">{{ m.suffix }}</span>
        </div>
        <div class="mt-1.5 text-sm text-white/60">{{ m.label }}</div>
      </div>
    </div>
  </section>
</template>
