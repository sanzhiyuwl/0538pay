/**
 * 首页板块注册表（后台 CMS 与首页渲染共享，避免两处枚举漂移）。
 * - CMS：左侧板块管理器按此渲染名称/图标/是否可拖可隐/条目数。
 * - 渲染：ClassicSections 按 SectionKey 映射到板块组件。
 */
import type { Component } from 'vue'
import {
  LayoutTemplate, BarChart3, Sparkles, Tag, Grid3x3,
  Newspaper, MessageSquareQuote, HelpCircle, Megaphone,
} from 'lucide-vue-next'
import type { SectionKey, SiteContent } from '@/lib/mock/site-content'

export interface SectionMeta {
  key: SectionKey
  label: string
  icon: Component
  /** 锁定：不可拖动、不可隐藏（hero 压在导航下，恒为首项）*/
  locked?: boolean
  /** 列表型板块的条目数取值（用于 CMS 徽标）；无则不显示徽标 */
  count?: (c: SiteContent) => number
}

export const sectionMetas: SectionMeta[] = [
  { key: 'hero', label: '首屏 / Hero', icon: LayoutTemplate, locked: true },
  { key: 'metrics', label: '数据背书', icon: BarChart3, count: (c) => c.metrics.length },
  { key: 'features', label: '核心特性', icon: Sparkles, count: (c) => c.features.length },
  { key: 'pricing', label: '费率方案', icon: Tag, count: (c) => c.plans.length },
  { key: 'products', label: '产品矩阵', icon: Grid3x3, count: (c) => c.products.length },
  { key: 'news', label: '最新动态', icon: Newspaper },
  { key: 'testimonials', label: '客户评价', icon: MessageSquareQuote, count: (c) => c.testimonials.length },
  { key: 'faqs', label: '常见问题', icon: HelpCircle, count: (c) => c.faqs.length },
  { key: 'cta', label: '底部 CTA', icon: Megaphone },
]

export const sectionMetaMap: Record<SectionKey, SectionMeta> = Object.fromEntries(
  sectionMetas.map((m) => [m.key, m]),
) as Record<SectionKey, SectionMeta>
