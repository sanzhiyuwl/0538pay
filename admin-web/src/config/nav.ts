import type { Component } from 'vue'
import {
  LayoutDashboard,
  ReceiptText,
  Users,
  CreditCard,
  Wallet,
  ShieldAlert,
  Settings,
  Server,
  ShieldCheck,
  LayoutGrid,
  Package,
  Receipt,
  BarChart3,
  ScrollText,
} from 'lucide-vue-next'

export interface NavLeaf {
  title: string
  to: string
  badge?: string
}

export interface NavNode {
  title: string
  icon: Component
  to?: string // 单项（无子菜单）
  badge?: string
  children?: NavLeaf[]
}

/** 两级折叠菜单：一级图标+标题，可展开子项。基于 epay admin 真实功能重组 */
export const navMenu: NavNode[] = [
  { title: '平台概况', icon: LayoutDashboard, to: '/' },
  {
    title: '交易管理',
    icon: ReceiptText,
    children: [
      { title: '订单管理', to: '/orders' },
      { title: '结算管理', to: '/settle' },
      { title: '分账记录', to: '/profit-sharing' },
      { title: '结算设置', to: '/settle-settings' },
    ],
  },
  {
    title: '商户管理',
    icon: Users,
    children: [
      { title: '商户列表', to: '/merchants' },
      { title: '用户组 / 套餐', to: '/groups' },
      { title: '资金明细', to: '/records' },
      { title: '支付统计', to: '/merchant-stats' },
      { title: '授权域名', to: '/domains' },
    ],
  },
  {
    title: '支付接口',
    icon: CreditCard,
    children: [
      { title: '支付通道', to: '/channels' },
      { title: '支付方式', to: '/pay-types' },
      { title: '支付插件', to: '/plugins' },
      { title: '通道轮询', to: '/rolls' },
      { title: '公众号 / 小程序', to: '/wechat' },
      { title: '支付设置', to: '/pay-settings' },
    ],
  },
  {
    title: '财务管理',
    icon: Wallet,
    children: [
      { title: '转账付款', to: '/transfer' },
      { title: '付款记录', to: '/transfer-records' },
      { title: '账单中心', to: '/billing', badge: 'New' },
    ],
  },
  {
    title: '风控管理',
    icon: ShieldAlert,
    children: [
      { title: '风控记录', to: '/risk' },
      { title: '黑名单', to: '/blacklist' },
      { title: '风控设置', to: '/risk-settings' },
    ],
  },
  {
    title: '权限管理',
    icon: ShieldCheck,
    children: [
      { title: '管理员', to: '/admins' },
      { title: '角色管理', to: '/roles' },
      { title: '操作日志', to: '/oplogs' },
    ],
  },
  {
    title: '系统设置',
    icon: Settings,
    children: [
      { title: '网站设置', to: '/settings' },
      { title: '注册登录', to: '/reg-settings' },
      { title: '网站公告', to: '/announcements' },
      { title: '登录日志', to: '/logs' },
      { title: '数据清理', to: '/clean' },
    ],
  },
]

/** 控制台入口（SaaS 独立后台）——单独固定在主后台侧栏最底部，不参与菜单流式排列 */
export const consoleEntry: NavNode = { title: '控制台', icon: Server, to: '/console', badge: 'SaaS' }

/** 扁平化所有可路由的叶子（供路由/面包屑用）。含控制台入口 */
export const allLeaves: NavLeaf[] = [
  ...navMenu.flatMap((n) => (n.children ? n.children : n.to ? [{ title: n.title, to: n.to }] : [])),
  { title: consoleEntry.title, to: consoleEntry.to!, badge: consoleEntry.badge },
]

/**
 * 控制台（SaaS 独立后台）专属导航。一级平铺，独立于主后台 navMenu。
 * 租户管理已实现，其余为规划中（见 docs/saas开发规划.txt），暂用占位页。
 */
export const consoleNav: NavNode[] = [
  { title: '租户管理', icon: LayoutGrid, to: '/console' },
  { title: '租户套餐', icon: Package, to: '/console/plans' },
  { title: '租户计费', icon: Receipt, to: '/console/billing' },
  { title: '分站总览', icon: BarChart3, to: '/console/overview' },
  { title: '操作审计', icon: ScrollText, to: '/console/logs' },
]

/** 控制台可路由叶子 */
export const consoleLeaves: NavLeaf[] = consoleNav.map((n) => ({
  title: n.title,
  to: n.to!,
  badge: n.badge,
}))
