<script setup lang="ts">
import { cn } from '@/lib/utils'
import Card from './Card.vue'

/**
 * 标准内容面板：白卡片 + 标题行 + 通栏分隔线 + 内容区。
 * 全站页面统一使用，保证卡片风格一致（直角、无边框、浅灰底浮白面板）。
 *
 * 用法：
 *   <Panel title="订单管理" subtitle="共 1,234 笔">
 *     <template #actions><Button>新增</Button></template>
 *     ...内容...
 *   </Panel>
 *
 * 无标题时（纯内容容器）：<Panel :no-header="true">...</Panel>
 * 内容区无内边距（如整块表格自控）：<Panel title="x" flush>...</Panel>
 */
const props = defineProps<{
  title?: string
  subtitle?: string
  noHeader?: boolean
  flush?: boolean
  class?: string
  bodyClass?: string
}>()
</script>

<template>
  <Card :class="props.class">
    <template v-if="!noHeader">
      <div class="flex items-center gap-2 px-6 py-4">
        <h3 class="text-[15px] font-semibold tracking-tight">{{ title }}</h3>
        <span v-if="subtitle" class="text-xs text-muted-foreground">{{ subtitle }}</span>
        <div class="flex-1" />
        <slot name="actions" />
      </div>
      <div class="border-t border-border/70" />
    </template>
    <div :class="cn(flush ? '' : 'px-6 py-4', props.bodyClass)">
      <slot />
    </div>
  </Card>
</template>
