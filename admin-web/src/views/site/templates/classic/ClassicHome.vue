<script setup lang="ts">
/**
 * classic 模板官网首页。
 * 板块顺序/显隐由后台「官网管理 / 首页内容」CMS 编排（content.sections），
 * 此处仅把 store.content 交给 ClassicSections 按序渲染，实时联动。
 */
import { useSiteContentStore } from '@/stores/siteContent'
import ClassicSections from './ClassicSections.vue'

const content = useSiteContentStore().content
</script>

<template>
  <ClassicSections :content="content" />
</template>
