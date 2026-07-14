<script setup lang="ts">
/**
 * 官网首页「壳」：按当前启用的模板 key 渲染对应模板组件。
 * 原型阶段用默认模板；未来由租户/分站配置决定（见 docs/官网模板库规划.txt）。
 * 支持 ?tpl=key 预览指定模板，方便模板市场里点开预览。
 */
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { getTemplate, defaultTemplateKey } from './registry'

const route = useRoute()
const activeKey = computed(() => {
  const q = route.query.tpl
  return typeof q === 'string' && q ? q : defaultTemplateKey
})
const template = computed(() => getTemplate(activeKey.value))
</script>

<template>
  <component :is="template.component" />
</template>
