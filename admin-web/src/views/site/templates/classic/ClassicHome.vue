<script setup lang="ts">
/**
 * classic 模板官网首页。
 * 板块顺序/显隐由后台「官网管理 / 首页内容」CMS 编排（content.sections），
 * 此处仅把 store.content 交给 ClassicSections 按序渲染，实时联动。
 * 首屏 Hero 是否用幻灯片，由 HeroSection 内部按「网站设置 / 幻灯片设置」自行判断。
 */
import { onMounted } from 'vue'
import { useSiteContentStore } from '@/stores/siteContent'
import { useSiteStore } from '@/stores/site'
import ClassicSections from './ClassicSections.vue'
import FloatingContact from '@/components/site/FloatingContact.vue'

const store = useSiteContentStore()
const content = store.content
const siteStore = useSiteStore()

// 官网首页加载时从后端拉取最新 CMS 内容 + 网站设置（本地缓存先渲染，后端到达后覆盖）
onMounted(() => {
  store.hydrate()
  siteStore.hydrate()
})
</script>

<template>
  <ClassicSections :content="content" />
  <!-- 右侧悬浮联系栏（客服/公众号/邮箱/返回顶部）-->
  <FloatingContact />
</template>
