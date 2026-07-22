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
  UserCog,
  ListOrdered,
  QrCode,
  Gift,
  HelpCircle,
  Globe,
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
  { title: '平台概况', icon: LayoutDashboard, to: '/admin' },
  {
    title: '交易管理',
    icon: ReceiptText,
    children: [
      { title: '订单管理', to: '/admin/orders' },
      { title: '结算管理', to: '/admin/settle' },
      { title: '分账记录', to: '/admin/profit-sharing' },
      { title: '结算设置', to: '/admin/settle-settings' },
    ],
  },
  {
    title: '商户管理',
    icon: Users,
    children: [
      { title: '商户列表', to: '/admin/merchants' },
      { title: '用户组 / 套餐', to: '/admin/groups' },
      { title: '邀请码管理', to: '/admin/invite-codes' },
      { title: '资金明细', to: '/admin/records' },
      { title: '支付统计', to: '/admin/merchant-stats' },
      { title: '支付用户统计', to: '/admin/buyer-stats' },
      { title: '授权域名', to: '/admin/domains' },
    ],
  },
  {
    title: '支付接口',
    icon: CreditCard,
    children: [
      { title: '支付通道', to: '/admin/channels' },
      { title: '支付方式', to: '/admin/pay-types' },
      { title: '支付插件', to: '/admin/plugins' },
      { title: '通道轮询', to: '/admin/rolls' },
      { title: '公众号 / 小程序', to: '/admin/wechat' },
      { title: '企业微信', to: '/admin/wework' },
      { title: '微信客服支付', to: '/admin/wxkf-settings' },
      { title: '获取用户标识', to: '/admin/gettoken' },
      { title: '支付设置', to: '/admin/pay-settings' },
    ],
  },
  {
    title: '财务管理',
    icon: Wallet,
    children: [
      { title: '转账付款', to: '/admin/transfer' },
      { title: '付款记录', to: '/admin/transfer-records' },
      { title: '账单中心', to: '/admin/billing', badge: 'New' },
    ],
  },
  {
    title: '风控管理',
    icon: ShieldAlert,
    children: [
      { title: '风控记录', to: '/admin/risk' },
      { title: '黑名单', to: '/admin/blacklist' },
      { title: '风控设置', to: '/admin/risk-settings' },
    ],
  },
  {
    title: '权限管理',
    icon: ShieldCheck,
    children: [
      { title: '管理员', to: '/admin/admins' },
      { title: '角色管理', to: '/admin/roles' },
      { title: '操作日志', to: '/admin/oplogs' },
      { title: '登录日志', to: '/admin/logs' },
    ],
  },
  {
    title: '官网管理',
    icon: Globe,
    children: [
      { title: '首页内容', to: '/admin/site-content' },
      { title: '文章管理', to: '/admin/articles' },
      { title: '文档管理', to: '/admin/docs-content' },
      { title: '首页模板', to: '/admin/template-settings' },
      { title: '网站公告', to: '/admin/announcements' },
    ],
  },
  {
    title: '系统设置',
    icon: Settings,
    children: [
      // 站点基础
      { title: '网站设置', to: '/admin/settings' },
      // 注册与登录鉴权
      { title: '注册登录', to: '/admin/reg-settings' },
      { title: '快捷登录', to: '/admin/oauth-settings' },
      { title: '实名认证', to: '/admin/cert-settings' },
      // 消息通知
      { title: '消息提醒', to: '/admin/notice-settings' },
      { title: '邮箱短信', to: '/admin/mail-settings' },
      // 运维
      { title: '计划任务', to: '/admin/cron-settings' },
      { title: '数据清理', to: '/admin/clean' },
      { title: '其余设置', to: '/admin/other-settings' },
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

/**
 * 商户中心（/m）专属导航。商户自助端，两级折叠分组，独立于主后台 navMenu。
 * 见 docs/商户中心开发规划.txt。条件功能（提现/充值/代付/购买会员/邀请）在真实环境由平台开关控制，
 * 原型阶段全部展示。
 */
export const merchantNav: NavNode[] = [
  { title: '工作台', icon: LayoutDashboard, to: '/m' },
  {
    title: '账户中心',
    icon: UserCog,
    children: [
      { title: '账户设置', to: '/m/profile' },
      { title: 'API 信息', to: '/m/api' },
      { title: '实名认证', to: '/m/certificate' },
      { title: '保证金', to: '/m/deposit' },
    ],
  },
  {
    title: '交易查询',
    icon: ListOrdered,
    children: [
      { title: '订单记录', to: '/m/orders' },
      { title: '资金明细', to: '/m/records' },
      { title: '结算记录', to: '/m/settle' },
      { title: '申请提现', to: '/m/apply' },
      { title: '余额充值', to: '/m/recharge' },
    ],
  },
  {
    title: '收款工具',
    icon: QrCode,
    children: [
      { title: '授权域名', to: '/m/domains' },
      { title: '聚合收款码', to: '/m/onecode' },
      { title: '代付管理', to: '/m/transfer' },
    ],
  },
  {
    title: '推广增值',
    icon: Gift,
    children: [
      { title: '购买会员', to: '/m/groupbuy' },
      { title: '邀请返现', to: '/m/invite' },
    ],
  },
  { title: '使用说明', icon: HelpCircle, to: '/m/help' },
]

/** 商户中心可路由叶子（供路由/面包屑用） */
export const merchantLeaves: NavLeaf[] = merchantNav.flatMap((n) =>
  n.children ? n.children : n.to ? [{ title: n.title, to: n.to }] : [],
)
