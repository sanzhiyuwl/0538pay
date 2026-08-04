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
  Receipt,
  BarChart3,
  UserCog,
  ListOrdered,
  QrCode,
  Gift,
  HelpCircle,
  Globe,
  Plug,
  MessageCircle,
  Handshake,
  KeyRound,
  ScrollText,
  Activity,
} from 'lucide-vue-next'

export interface NavLeaf {
  title: string
  to: string
  badge?: string
  icon?: Component // 子项图标（可选；仅部分布局如控制台侧栏渲染）
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
    title: '交易中心',
    icon: ReceiptText,
    children: [
      { title: '订单管理', to: '/admin/orders' },
      { title: '结算管理', to: '/admin/settle' },
      { title: '分账记录', to: '/admin/profit-sharing' },
    ],
  },
  {
    title: '商户管理',
    icon: Users,
    children: [
      { title: '商户列表', to: '/admin/merchants' },
      { title: '通道进件审核', to: '/admin/channel-enrolls' },
      { title: '商户风控管控', to: '/admin/risk-controls' },
      { title: '用户组 / 套餐', to: '/admin/groups' },
      { title: '邀请码管理', to: '/admin/invite-codes' },
      { title: '授权域名', to: '/admin/domains' },
      { title: '商户日志', to: '/admin/merchant-oplogs' },
    ],
  },
  {
    title: '数据统计',
    icon: BarChart3,
    children: [
      { title: '支付统计', to: '/admin/merchant-stats' },
      { title: '支付用户统计', to: '/admin/buyer-stats' },
      { title: '资金明细', to: '/admin/records' },
    ],
  },
  {
    title: '代付管理',
    icon: Wallet,
    children: [
      { title: '转账付款', to: '/admin/transfer' },
      { title: '付款记录', to: '/admin/transfer-records' },
      { title: '账单中心', to: '/admin/billing', badge: 'New' },
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
    ],
  },
  {
    title: '微信配置',
    icon: MessageCircle,
    children: [
      { title: '公众号 / 小程序', to: '/admin/wechat' },
      { title: '企业微信', to: '/admin/wework' },
      { title: '微信客服支付', to: '/admin/wxkf-settings' },
      { title: '获取用户标识', to: '/admin/gettoken' },
      { title: '消息提醒', to: '/admin/notice-settings' },
    ],
  },
  {
    title: '风控管理',
    icon: ShieldAlert,
    children: [
      { title: '风控记录', to: '/admin/risk' },
      { title: '黑名单', to: '/admin/blacklist' },
    ],
  },
  {
    title: '内容运营',
    icon: Globe,
    children: [
      { title: '首页内容', to: '/admin/site-content' },
      { title: '文章管理', to: '/admin/articles' },
      { title: '文档管理', to: '/admin/docs-content' },
      { title: '首页模板', to: '/admin/template-settings' },
      { title: '网站公告', to: '/admin/announcements' },
      { title: '使用说明', to: '/admin/help-settings' },
      { title: '站内信下发', to: '/admin/messages' },
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
    title: '三方配置',
    icon: Plug,
    children: [
      { title: '快捷登录', to: '/admin/oauth-settings' },
      { title: '实名认证', to: '/admin/cert-settings' },
      { title: 'OCR 识别', to: '/admin/ocr-settings' },
      { title: '邮箱短信', to: '/admin/mail-settings' },
    ],
  },
  {
    title: '系统设置',
    icon: Settings,
    children: [
      // 站点基础
      { title: '网站设置', to: '/admin/settings' },
      // 交易/资金配置
      { title: '支付设置', to: '/admin/pay-settings' },
      { title: '结算设置', to: '/admin/settle-settings' },
      { title: '转账设置', to: '/admin/transfer-settings' },
      { title: '风控设置', to: '/admin/risk-settings' },
      // 注册与登录鉴权
      { title: '注册登录', to: '/admin/reg-settings' },
      // 运维
      { title: '计划任务', to: '/admin/cron-settings' },
      { title: '数据清理', to: '/admin/clean' },
      { title: '其余设置', to: '/admin/other-settings' },
    ],
  },
]

/** 代理控制台入口——单独固定在主后台侧栏最底部，不参与菜单流式排列 */
export const consoleEntry: NavNode = { title: '控制台', icon: Server, to: '/console', badge: '代理' }

/** 扁平化所有可路由的叶子（供路由/面包屑用）。含控制台入口 */
export const allLeaves: NavLeaf[] = [
  ...navMenu.flatMap((n) => (n.children ? n.children : n.to ? [{ title: n.title, to: n.to }] : [])),
  { title: consoleEntry.title, to: consoleEntry.to!, badge: consoleEntry.badge },
]

/**
 * 代理控制台专属导航（平台运营视角，管所有代理进件）。两级折叠，独立于主后台 navMenu。
 * 顶端「概况」为单项落地页（/console）；现有六项收进「微信特约商户」分组。
 * 后续往控制台加的新模块各自成组追加在此，不与特约商户混放。
 * 自研扩展（epay 无），见 docs-代理进件/。SaaS 出租方向已于 2026-07-28 停做，旧 5 页归档到 _archive。
 */
export const consoleNav: NavNode[] = [
  { title: '概况', icon: LayoutDashboard, to: '/console' },
  {
    title: '微信特约商户',
    icon: Handshake,
    children: [
      { title: '进件申请', to: '/console/enroll', icon: ListOrdered },
      { title: '邀请链接', to: '/console/invites', icon: QrCode },
      { title: '进件设置', to: '/console/settings', icon: Settings },
      { title: '服务商配置', to: '/console/wx-partner', icon: KeyRound },
    ],
  },
  // 代理管理 / 名额管理 / 佣金结算 / 权限分配 独立成顶级项——各自后续还会挂更多子功能，先各占一席。
  { title: '代理管理', icon: Users, to: '/console/agents' },
  { title: '权限分配', icon: ShieldCheck, to: '/console/permissions' },
  { title: '名额管理', icon: Wallet, to: '/console/quota' },
  { title: '佣金结算', icon: Receipt, to: '/console/settlement' },
  {
    title: '审计日志',
    icon: ScrollText,
    children: [
      { title: '代理操作日志', to: '/console/agent-logs', icon: ListOrdered },
      { title: '管理日志', to: '/console/manage-logs', icon: UserCog },
      { title: '运维日志', to: '/console/system-logs', icon: Activity },
    ],
  },
]

/** 控制台可路由叶子（供路由/面包屑用；含单项与分组子项） */
export const consoleLeaves: NavLeaf[] = consoleNav.flatMap((n) =>
  n.children ? n.children : n.to ? [{ title: n.title, to: n.to, badge: n.badge }] : [],
)

/**
 * 商户中心（/m）专属导航。商户自助端，两级折叠分组，独立于主后台 navMenu。
 * 见 docs/商户中心开发规划.txt。条件功能（提现/充值/代付/通道套餐/邀请）在真实环境由平台开关控制，
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
  { title: '渠道申请', icon: Plug, to: '/m/channel-enroll' },
  {
    title: '交易查询',
    icon: ListOrdered,
    children: [
      { title: '订单记录', to: '/m/orders' },
      { title: '资金明细', to: '/m/records' },
      { title: '结算记录', to: '/m/settle' },
      // { title: '申请提现', to: '/m/apply' }, // 隐藏（路由保留）
      { title: '余额充值', to: '/m/recharge' },
    ],
  },
  {
    title: '收款工具',
    icon: QrCode,
    children: [
      { title: '授权域名', to: '/m/domains' },
      { title: '聚合收款', to: '/m/onecode' },
      { title: '测试支付', to: '/m/test' },
      // { title: '代付管理', to: '/m/transfer' }, // 隐藏（路由保留）
    ],
  },
  // { title: '通道套餐', icon: Layers, to: '/m/groupbuy' }, // 隐藏（用户组套餐，路由保留）
  { title: '邀请返现', icon: Gift, to: '/m/invite' },
  { title: '使用说明', icon: HelpCircle, to: '/m/help' },
]

/** 商户中心可路由叶子（供路由/面包屑用） */
export const merchantLeaves: NavLeaf[] = merchantNav.flatMap((n) =>
  n.children ? n.children : n.to ? [{ title: n.title, to: n.to }] : [],
)

/**
 * 独立代理端（/agent）专属导航。代理自助端，只看/只碰自己名下，一级平铺。
 * 自研扩展（epay 无），见 docs-代理进件/。每项带 perm 权限点 key（概览无 perm 恒可见），
 * 前端按代理已开通的 permissions 动态过滤菜单——权限开通啥代理就看到啥。
 * perm key 与后端 service.AgentPermissionCatalog 一致（enroll/quota/invite/settlement）。
 */
export interface AgentNavLeaf extends NavLeaf {
  perm?: string // 需要的权限点 key；空=恒可见
}
export const agentNav: (NavNode & { perm?: string })[] = [
  { title: '工作台', icon: LayoutDashboard, to: '/agent' },
  { title: '进件申请', icon: ListOrdered, to: '/agent/enroll', perm: 'enroll' },
  { title: '名额钱包', icon: Wallet, to: '/agent/quota', perm: 'quota' },
  { title: '邀请链接', icon: QrCode, to: '/agent/invites', perm: 'invite' },
  { title: '佣金结算', icon: Receipt, to: '/agent/settlement', perm: 'settlement' },
]

/** 代理端可路由叶子（供路由/面包屑用；路由不做权限过滤，无权时页面自身回退，菜单才隐藏） */
export const agentLeaves: AgentNavLeaf[] = agentNav.map((n) => ({
  title: n.title,
  to: n.to!,
  perm: n.perm,
}))
