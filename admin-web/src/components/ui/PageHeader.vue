<script setup lang="ts">
/**
 * 页面标题区：大标题 + 副标题 + 右侧操作插槽。
 * 放在每个页面根容器最上方（Panel 之前）。
 *
 * 用法：
 *   <PageHeader title="订单管理" subtitle="共 1,234 笔订单">
 *     <template #actions><Button>导出</Button></template>
 *   </PageHeader>
 */
defineProps<{ title: string; subtitle?: string }>()
</script>

<template>
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="text-xl font-semibold tracking-tight">{{ title }}</h1>
      <p v-if="subtitle" class="mt-1 text-sm text-muted-foreground">{{ subtitle }}</p>
    </div>
    <div class="flex items-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
