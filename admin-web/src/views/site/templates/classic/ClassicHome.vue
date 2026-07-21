<script setup lang="ts">
/**
 * classic 模板官网首页。
 * 板块顺序/显隐由后台「官网管理 / 首页内容」CMS 编排（content.sections），
 * 此处仅把 store.content 交给 ClassicSections 按序渲染，实时联动。
 */
import { onMounted } from 'vue'
import { useSiteContentStore } from '@/stores/siteContent'
import ClassicSections from './ClassicSections.vue'

const store = useSiteContentStore()
const content = store.content
// 官网首页加载时从后端拉取最新 CMS 内容（本地缓存先渲染，后端到达后覆盖）
onMounted(() => { store.hydrate() })
</script>

<template>
  <ClassicSections :content="content" />
</template>
