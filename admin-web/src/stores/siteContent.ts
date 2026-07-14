/**
 * 官网首页营销内容单一数据源（CMS）。
 * 运营后台「官网管理 / 首页内容」写入 → 官网首页(ClassicHome)读取，实时联动。
 * 持久化到 localStorage，从 mock defaultSiteContent 取默认值（后端接好后换成接口）。
 */
import { defineStore } from 'pinia'
import { reactive, watch } from 'vue'
import { defaultSiteContent, type SiteContent } from '@/lib/mock/site-content'

const STORAGE_KEY = 'site-content'

/** 深拷贝默认内容，避免污染 mock 常量 */
function clone(src: SiteContent): SiteContent {
  return JSON.parse(JSON.stringify(src))
}

function load(): SiteContent {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      // 与默认值浅合并顶层键，容忍旧缓存缺字段
      return { ...clone(defaultSiteContent), ...JSON.parse(raw) }
    }
  } catch {
    // 损坏缓存忽略，回退默认
  }
  return clone(defaultSiteContent)
}

export const useSiteContentStore = defineStore('site-content', () => {
  const content = reactive<SiteContent>(load())

  /** 用整份新内容覆盖并持久化（后台保存调用） */
  function update(next: SiteContent) {
    Object.assign(content, clone(next))
  }

  /** 恢复默认内容 */
  function reset() {
    Object.assign(content, clone(defaultSiteContent))
  }

  watch(
    content,
    () => localStorage.setItem(STORAGE_KEY, JSON.stringify(content)),
    { deep: true },
  )

  return { content, update, reset }
})
