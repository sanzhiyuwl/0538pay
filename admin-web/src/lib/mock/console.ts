/**
 * SaaS 控制台 P1/P2 页面假数据 —— 租户套餐定价 / 租户计费收入 / 分站总览 / 操作审计。
 * 复用 sites.ts 的租户与套餐定义，字段对齐 docs/saas开发规划.txt。
 */
import { sites, sitePlanText, planPrice, planQuota, type SitePlan } from './sites'

function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}

/* ============================================================
 * 一、租户套餐定价（/console/plans）
 * ============================================================ */

export interface PlanCard {
  key: SitePlan
  name: string
  price: number // 元/年
  monthly: number // 折算元/月
  desc: string
  highlight: boolean // 是否推荐
  modules: string[] // 含的功能模块
  quotaText: { label: string; value: string }[]
  features: string[] // 卖点差异
  siteCount: number // 当前使用该套餐的租户数
}

const planDesc: Record<SitePlan, string> = {
  basic: '适合起步的小微分站，核心收款能力开箱即用',
  pro: '成长型分站首选，全功能 + 更高配额',
  ultimate: '大型分站 / 服务商，超高配额与全通道',
}
const planModules: Record<SitePlan, string[]> = {
  basic: ['交易管理', '商户管理', '支付接口', '财务管理'],
  pro: ['交易管理', '商户管理', '支付接口', '财务管理', '风控管理'],
  ultimate: ['交易管理', '商户管理', '支付接口', '财务管理', '风控管理'],
}
const planFeatures: Record<SitePlan, string[]> = {
  basic: ['2 条支付通道', '标准工单支持', '基础数据报表'],
  pro: ['4 条支付通道', '风控管理模块', '优先工单支持', '独立域名绑定'],
  ultimate: ['全部支付通道', '白标品牌定制', '专属客户经理', '免密登录管理', '开放 API 对接'],
}

export const planCards: PlanCard[] = (Object.keys(sitePlanText) as SitePlan[]).map((key) => {
  const q = planQuota[key]
  return {
    key,
    name: sitePlanText[key],
    price: planPrice[key],
    monthly: Math.round(planPrice[key] / 12),
    desc: planDesc[key],
    highlight: key === 'pro',
    modules: planModules[key],
    quotaText: [
      { label: '最大商户数', value: q.maxMerchants.toLocaleString() },
      { label: '最大通道数', value: String(q.maxChannels) },
      { label: '月交易额上限', value: `¥${(q.monthlyAmount / 10000).toLocaleString()} 万` },
    ],
    features: planFeatures[key],
    siteCount: sites.filter((s) => s.plan === key).length,
  }
})

/* ============================================================
 * 二、租户计费收入（/console/billing）
 * ============================================================ */

export type BillType = 'new' | 'renew' | 'upgrade' // 开通 / 续费 / 升级
export const billTypeText: Record<BillType, string> = {
  new: '新开通',
  renew: '续费',
  upgrade: '套餐升级',
}
export const billTypeVariant: Record<BillType, 'success' | 'default' | 'warning'> = {
  new: 'success',
  renew: 'default',
  upgrade: 'warning',
}

export type BillStatus = 0 | 1 | 2 // 待支付 / 已支付 / 已退款
export const billStatusText: Record<BillStatus, { text: string; variant: 'warning' | 'success' | 'muted' }> = {
  0: { text: '待支付', variant: 'warning' },
  1: { text: '已支付', variant: 'success' },
  2: { text: '已退款', variant: 'muted' },
}

export interface BillOrder {
  id: string // 账单号
  siteName: string // 分站名
  domain: string
  plan: SitePlan
  type: BillType
  months: number // 购买时长（月）
  amount: number // 金额
  status: BillStatus
  payMethod: string // 支付方式
  createTime: string
}

const payMethods = ['支付宝', '微信支付', '对公转账', '余额抵扣']
const billTypes: BillType[] = ['new', 'renew', 'renew', 'upgrade', 'new', 'renew']
const billStatusPool: BillStatus[] = [1, 1, 1, 1, 0, 1, 2, 1]

export const billOrders: BillOrder[] = Array.from({ length: 28 }, (_, i) => {
  const site = sites[i % sites.length]
  const type = billTypes[i % billTypes.length]
  const months = type === 'new' ? 12 : [12, 6, 24, 12][i % 4]
  const base = planPrice[site.plan]
  const amount =
    type === 'upgrade'
      ? Math.round(base * 0.6)
      : Math.round((base / 12) * months)
  const status = billStatusPool[i % billStatusPool.length]
  const mo = 1 + ((7 - Math.floor(i / 4)) % 7)
  return {
    id: `SB${2026}${pad(mo)}${pad(28 - i, 4)}`,
    siteName: site.name,
    domain: site.domain,
    plan: site.plan,
    type,
    months,
    amount,
    status,
    payMethod: payMethods[i % payMethods.length],
    createTime: `2026-${pad(mo)}-${pad(1 + ((i * 3) % 27))} ${pad(9 + (i % 10))}:${pad((i * 7) % 60)}:00`,
  }
})

/** 计费概况统计 */
export function calcBillingStats(list: BillOrder[]) {
  const paid = list.filter((b) => b.status === 1)
  const totalRevenue = paid.reduce((a, b) => a + b.amount, 0)
  // 本月（示意取 2026-07）
  const monthRevenue = paid
    .filter((b) => b.createTime.startsWith('2026-07'))
    .reduce((a, b) => a + b.amount, 0)
  return {
    totalRevenue,
    monthRevenue,
    paidCount: paid.length,
    pendingCount: list.filter((b) => b.status === 0).length,
    refundAmount: list.filter((b) => b.status === 2).reduce((a, b) => a + b.amount, 0),
  }
}

/* ============================================================
 * 三、分站总览（/console/overview）—— 跨分站汇总看板
 * ============================================================ */

/** 近 7 日全平台 GMV 趋势（各套餐分层） */
export const overviewDates = Array.from({ length: 7 }, (_, i) => `07-${pad(8 + i)}`)

export const overviewGmvSeries = [
  { name: '基础版', color: '#94a3b8', data: [82, 90, 76, 105, 98, 88, 112].map((v) => v * 1000) },
  { name: '专业版', color: '#4b7bec', data: [210, 245, 198, 268, 255, 232, 290].map((v) => v * 1000) },
  { name: '旗舰版', color: '#7c3aed', data: [520, 560, 610, 580, 640, 690, 720].map((v) => v * 1000) },
]

/** 分站 GMV 排行（Top） */
export interface SiteRank {
  name: string
  domain: string
  plan: SitePlan
  gmv: number // 本月 GMV
  orders: number // 本月订单数
  merchants: number // 商户数
  growth: number // 环比 %
}

export const siteRanks: SiteRank[] = sites
  .filter((s) => s.status === 1)
  .map((s, i) => ({
    name: s.name,
    domain: s.domain,
    plan: s.plan,
    gmv: s.usage.monthlyAmount,
    orders: Math.round(s.usage.monthlyAmount / (120 + (i % 5) * 30)),
    merchants: s.usage.merchants,
    growth: [12.5, 8.3, -4.2, 23.1, 5.6, -1.8, 15.0][i % 7],
  }))
  .sort((a, b) => b.gmv - a.gmv)

/** 总览汇总卡 */
export function calcOverviewStats() {
  const running = sites.filter((s) => s.status === 1)
  const totalGmv = running.reduce((a, s) => a + s.usage.monthlyAmount, 0)
  const totalMerchants = running.reduce((a, s) => a + s.usage.merchants, 0)
  const totalOrders = siteRanks.reduce((a, s) => a + s.orders, 0)
  return {
    siteCount: running.length,
    totalGmv,
    totalMerchants,
    totalOrders,
    avgGmv: running.length ? Math.round(totalGmv / running.length) : 0,
  }
}

/* ============================================================
 * 四、操作审计（/console/logs）
 * ============================================================ */

export type AuditAction =
  | 'create'
  | 'edit'
  | 'renew'
  | 'stop'
  | 'enable'
  | 'delete'
  | 'login'
  | 'permission'

export const auditActionText: Record<AuditAction, { text: string; variant: 'success' | 'default' | 'warning' | 'destructive' | 'muted' }> = {
  create: { text: '创建租户', variant: 'success' },
  edit: { text: '编辑配置', variant: 'default' },
  renew: { text: '续期', variant: 'default' },
  stop: { text: '停用租户', variant: 'warning' },
  enable: { text: '启用租户', variant: 'success' },
  delete: { text: '删除租户', variant: 'destructive' },
  login: { text: '免密登录', variant: 'warning' },
  permission: { text: '修改权限', variant: 'default' },
}

export interface AuditLog {
  id: number
  operator: string // 操作者（平台管理员）
  action: AuditAction
  target: string // 目标分站
  detail: string // 详情
  ip: string
  time: string
}

const operators = ['superadmin', 'ops_wang', 'ops_li']
const auditActions: AuditAction[] = ['create', 'edit', 'renew', 'stop', 'enable', 'delete', 'login', 'permission']

export const auditLogs: AuditLog[] = Array.from({ length: 36 }, (_, i) => {
  const action = auditActions[i % auditActions.length]
  const site = sites[i % sites.length]
  const detailMap: Record<AuditAction, string> = {
    create: `套餐「${sitePlanText[site.plan]}」，期限 12 个月`,
    edit: '调整资源配额与基础信息',
    renew: `续费 ${[6, 12, 24][i % 3]} 个月`,
    stop: '因欠费停用分站',
    enable: '恢复分站运行',
    delete: '删除已过期分站',
    login: '免密进入分站后台排查问题',
    permission: '新增「风控管理」模块权限',
  }
  const day = 12 - (i % 10)
  return {
    id: 90000 + (36 - i),
    operator: operators[i % operators.length],
    action,
    target: site.name,
    detail: detailMap[action],
    ip: `${100 + (i % 100)}.${(i * 7) % 255}.${(i * 3) % 255}.${i % 255}`,
    time: `2026-07-${pad(day)} ${pad(8 + (i % 12))}:${pad((i * 11) % 60)}:${pad((i * 13) % 60)}`,
  }
})
