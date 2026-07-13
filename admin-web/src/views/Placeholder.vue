<script setup lang="ts">
import { useRoute } from 'vue-router'
import { computed } from 'vue'
import { Construction } from 'lucide-vue-next'
import { allLeaves } from '@/config/nav'

const route = useRoute()
const title = computed(
  () => allLeaves.find((i) => i.to === route.path)?.title ?? '页面',
)
</script>

<template>
  <div class="flex min-h-[60vh] flex-col items-center justify-center gap-4 text-center">
    <div class="flex size-16 items-center justify-center rounded-2xl bg-muted">
      <Construction class="size-8 text-muted-foreground" />
    </div>
    <div>
      <h2 class="text-xl font-semibold">{{ title }}</h2>
      <p class="mt-1 text-sm text-muted-foreground">
        该模块原型即将上线，本阶段先呈现「仪表盘」的现代 SaaS 风格。
      </p>
    </div>
  </div>
</template>
