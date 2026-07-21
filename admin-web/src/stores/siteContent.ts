/**
 * 官网首页营销内容单一数据源（CMS）。
 * 运营后台「官网管理 / 首页内容」写入 → 官网首页(ClassicHome)读取，实时联动。
 * 持久化到 localStorage，从 mock defaultSiteContent 取默认值（后端接好后换成接口）。
 */
import { defineStore } from 'pinia'
import { reactive, watch } from 'vue'
import { defaultSiteContent, defaultSections, type SiteContent, type SectionItem, type SectionKey } from '@/lib/mock/site-content'
import { fetchSiteConfig, saveSiteConfig } from '@/lib/api/siteConfig'

const STORAGE_KEY = 'site-content'

/** 深拷贝默认内容，避免污染 mock 常量 */
function clone(src: SiteContent): SiteContent {
  return JSON.parse(JSON.stringify(src))
}

/**
 * 板块编排容错：改版后旧缓存缺 sections 或键漂移时兜底。
 * - 缺失/非数组 → 全量默认。
 * - 保留旧缓存里合法且顺序不变的板块，剔除未知 key，补齐新增 key（追加末尾）。
 * - 强制 hero 存在且置首（它压在导航下，恒为首项）。
 */
function normalizeSections(raw: unknown): SectionItem[] {
  const known = new Set<SectionKey>(defaultSections.map((s) => s.key))
  if (!Array.isArray(raw)) return defaultSections.map((s) => ({ ...s }))
  const seen = new Set<SectionKey>()
  const out: SectionItem[] = []
  for (const item of raw) {
    const it = item as { key?: unknown; visible?: unknown } | null
    if (!it || typeof it.key !== 'string') continue
    const key = it.key as SectionKey
    if (!known.has(key) || seen.has(key)) continue
    seen.add(key)
    out.push({ key, visible: it.visible !== false })
  }
  // 补齐默认里有但缓存缺的板块（按默认顺序追加）
  for (const s of defaultSections) {
    if (!seen.has(s.key)) out.push({ ...s })
  }
  // hero 恒为首项且可见
  const heroIdx = out.findIndex((s) => s.key === 'hero')
  if (heroIdx > 0) out.unshift(out.splice(heroIdx, 1)[0])
  if (out[0]) out[0].visible = true
  return out
}

function load(): SiteContent {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      // 与默认值浅合并顶层键，容忍旧缓存缺字段
      const merged = { ...clone(defaultSiteContent), ...JSON.parse(raw) }
      // 迁移：testimonials 旧结构为 {name,role,avatar,text}，新结构为 {name,desc,image}。
      // 旧缓存缺 image/desc 字段时回退默认，避免首页轮播读到 undefined。
      if (
        Array.isArray(merged.testimonials) &&
        merged.testimonials.some((t: unknown) => {
          const item = t as { image?: unknown; desc?: unknown } | null
          return item == null || typeof item.image !== 'string' || typeof item.desc !== 'string'
        })
      ) {
        merged.testimonials = clone(defaultSiteContent).testimonials
      }
      merged.sections = normalizeSections(merged.sections)
      return merged
    }
  } catch {
    // 损坏缓存忽略，回退默认
  }
  return clone(defaultSiteContent)
}

export const useSiteContentStore = defineStore('site-content', () => {
  // 先用 localStorage/默认值即时渲染，随后 hydrate 从后端拉取真实数据覆盖。
  const content = reactive<SiteContent>(load())
  let hydrated = false

  /** 从后端拉取并覆盖（官网/后台初始化时调用，一次即可）。失败静默回退本地缓存。 */
  async function hydrate() {
    if (hydrated) return
    hydrated = true
    try {
      const remote = await fetchSiteConfig<SiteContent>('content')
      if (remote) {
        const merged = { ...clone(defaultSiteContent), ...remote }
        merged.sections = normalizeSections(merged.sections)
        Object.assign(content, merged)
      }
    } catch {
      // 后端不可用时用本地缓存，不阻塞渲染
    }
  }

  /** 用整份新内容覆盖并持久化到后端（后台保存调用）。 */
  async function update(next: SiteContent) {
    Object.assign(content, clone(next))
    await saveSiteConfig('content', content)
  }

  /** 恢复默认内容（本地，不落后端；需保存才生效）。 */
  function reset() {
    Object.assign(content, clone(defaultSiteContent))
  }

  // 本地缓存：任何变化即时落 localStorage（即时渲染 + 离线兜底）。
  watch(
    content,
    () => localStorage.setItem(STORAGE_KEY, JSON.stringify(content)),
    { deep: true },
  )

  return { content, hydrate, update, reset }
})
