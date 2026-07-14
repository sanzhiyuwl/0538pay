<script setup lang="ts">
import { useRoute } from 'vue-router'
import { computed } from 'vue'
import { Construction } from 'lucide-vue-next'
import { merchantLeaves } from '@/config/nav'

const route = useRoute()
const title = computed(
  () => merchantLeaves.find((i) => i.to === route.path)?.title ?? '页面',
)
</script>

<template>
  <div class="flex min-h-[60vh] flex-col items-center justify-center text-center">
    <div class="flex size-16 items-center justify-center rounded-2xl bg-muted/50 text-muted-foreground">
      <Construction class="size-8" />
    </div>
    <h2 class="mt-4 text-lg font-semibold">{{ title }}</h2>
    <p class="mt-1 text-sm text-muted-foreground">该商户中心页面正在建设中</p>
  </div>
</template>
