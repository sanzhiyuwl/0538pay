import type { Component } from 'vue'
import {
  LayoutDashboard,
  ReceiptText,
  Users,
  CreditCard,
  Wallet,
  ShieldAlert,
  Crown,
  Settings,
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
    ],
  },
  {
    title: 'SaaS 运营',
    icon: Crown,
    children: [
      { title: '套餐订阅', to: '/subscriptions', badge: 'New' },
      { title: '收入分析', to: '/analytics', badge: 'New' },
      { title: '定价配置', to: '/pricing', badge: 'New' },
    ],
  },
  {
    title: '系统设置',
    icon: Settings,
    children: [
      { title: '系统设置', to: '/settings' },
      { title: '网站公告', to: '/announcements' },
      { title: '登录日志', to: '/logs' },
      { title: '数据清理', to: '/clean' },
    ],
  },
]

/** 扁平化所有可路由的叶子（供路由/面包屑用） */
export const allLeaves: NavLeaf[] = navMenu.flatMap((n) =>
  n.children ? n.children : n.to ? [{ title: n.title, to: n.to }] : [],
)
