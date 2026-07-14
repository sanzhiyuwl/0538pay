/**
 * 站点配置单一数据源。
 * 运营后台「网站设置」写入 → 官网(SiteLayout/页脚/告示条/SEO)读取，实时联动。
 * 持久化到 localStorage，从 mock siteConfig 取默认值（后端接好后换成接口）。
 * 同时负责把 title / keywords / description 同步到 document 的 <title> 与 <meta>。
 */
import { defineStore } from 'pinia'
import { reactive, watch } from 'vue'
import { siteConfig } from '@/lib/mock/settings'

export type SiteConfig = typeof siteConfig

const STORAGE_KEY = 'site-config'

function load(): SiteConfig {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) return { ...siteConfig, ...JSON.parse(raw) }
  } catch {
    // 损坏的缓存忽略，回退默认
  }
  return { ...siteConfig }
}

/** 写/更新一个 <meta name="..."> 标签 */
function setMeta(name: string, content: string) {
  let el = document.querySelector<HTMLMetaElement>(`meta[name="${name}"]`)
  if (!el) {
    el = document.createElement('meta')
    el.setAttribute('name', name)
    document.head.appendChild(el)
  }
  el.setAttribute('content', content)
}

export const useSiteStore = defineStore('site', () => {
  const config = reactive<SiteConfig>(load())

  /** 同步 SEO 相关标签到 document */
  function applySeo() {
    if (config.title) document.title = config.title
    setMeta('keywords', config.keywords || '')
    setMeta('description', config.description || '')
  }

  /** 批量更新并持久化（后台保存调用） */
  function update(patch: Partial<SiteConfig>) {
    Object.assign(config, patch)
  }

  // config 任何变化都落库 + 刷新 SEO
  watch(
    config,
    () => {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(config))
      applySeo()
    },
    { immediate: true, deep: true },
  )

  return { config, update }
})
