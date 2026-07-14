/**
 * 官网首页模板注册表。
 * 每套模板 = 一个独立 Vue 组件（组件式模板，整体切换）。
 * 新增模板：在 templates/ 下加组件 → 这里 登记一行。
 * 售卖：tier + price 供模板市场/控制台衔接（免费/会员/付费）。见 docs/官网模板库规划.txt。
 */
import type { Component } from 'vue'
import { defineAsyncComponent } from 'vue'

/** 模板售卖层级 */
export type TemplateTier = 'free' | 'member' | 'paid'
export const tierMeta: Record<TemplateTier, { text: string; variant: 'muted' | 'default' | 'warning' }> = {
  free: { text: '免费', variant: 'muted' },
  member: { text: '会员', variant: 'default' },
  paid: { text: '付费', variant: 'warning' },
}

export interface SiteTemplate {
  key: string // 唯一标识
  name: string // 展示名
  desc: string // 风格一句话
  preview: string // 预览图 /site/previews/xxx.png
  tier: TemplateTier // 售卖层级
  price: number // 付费模板价格（元），free/member 为 0
  component: Component // 懒加载的模板组件
}

/** 所有官网模板。数组顺序 = 展示顺序，第一项作为默认。 */
export const siteTemplates: SiteTemplate[] = [
  {
    key: 'classic',
    name: '经典蓝',
    desc: '简洁现代的 SaaS 风格，宽屏左右分栏，蓝白配色，通用稳重。',
    preview: '/site/previews/classic.png',
    tier: 'free',
    price: 0,
    component: defineAsyncComponent(() => import('./templates/classic/ClassicHome.vue')),
  },
]

/** 默认启用模板 key（原型阶段固定；未来由租户分站配置决定）。 */
export const defaultTemplateKey = siteTemplates[0].key

/** 按 key 取模板；找不到回退默认。 */
export function getTemplate(key?: string): SiteTemplate {
  return siteTemplates.find((t) => t.key === key) ?? siteTemplates[0]
}
