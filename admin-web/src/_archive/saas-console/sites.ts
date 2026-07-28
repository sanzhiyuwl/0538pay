/**
 * 子站点租户（分站）假数据 —— SaaS 控制台核心。
 * 主站运营方创建、售卖、管理一个个独立支付站实例（tenant），按套餐 + 期限付费。
 * 字段对齐未来后端 pre_site 表设想（见 docs/saas开发规划.txt）。
 */

/** 租户套餐 */
export type SitePlan = 'basic' | 'pro' | 'ultimate'
export const sitePlanText: Record<SitePlan, string> = {
  basic: '基础版',
  pro: '专业版',
  ultimate: '旗舰版',
}

/** 可分配的功能模块（对应主后台的一级功能组） */
export const allModules = [
  { key: 'trade', label: '交易管理' },
  { key: 'merchant', label: '商户管理' },
  { key: 'channel', label: '支付接口' },
  { key: 'finance', label: '财务管理' },
  { key: 'risk', label: '风控管理' },
]

/** 可分配的支付通道（引用主站通道池，简化版） */
export const allChannels = [
  { id: 1, name: '支付宝官方直连' },
  { id: 2, name: '微信服务商A' },
  { id: 3, name: 'QQ钱包官方' },
  { id: 4, name: '云闪付通道' },
]

export interface SiteQuota {
  maxMerchants: number // 最大商户数
  maxChannels: number // 最大通道数
  monthlyAmount: number // 月交易额上限
}
export interface SiteUsage {
  merchants: number
  channels: number
  monthlyAmount: number
}

/** 子站点租户（pre_site） */
export interface Site {
  id: number
  name: string // 站点名称
  domain: string // 绑定域名
  adminUser: string // 分站管理员账号
  plan: SitePlan // 租户套餐
  status: 0 | 1 | 2 | 3 // 0待激活 1运行中 2已停用 3已过期
  createTime: string // 开通时间
  expireTime: string // 到期时间
  permissions: string[] // 功能模块权限（allModules 的 key）
  channelScope: number[] // 可用通道 ID
  quota: SiteQuota
  usage: SiteUsage
}

/** 租户状态字典 */
export const siteStatus: Record<
  number,
  { text: string; variant: 'default' | 'success' | 'muted' | 'destructive' }
> = {
  0: { text: '待激活', variant: 'default' },
  1: { text: '运行中', variant: 'success' },
  2: { text: '已停用', variant: 'muted' },
  3: { text: '已过期', variant: 'destructive' },
}

/** 各套餐的默认配额 */
export const planQuota: Record<SitePlan, SiteQuota> = {
  basic: { maxMerchants: 50, maxChannels: 2, monthlyAmount: 500000 },
  pro: { maxMerchants: 300, maxChannels: 4, monthlyAmount: 3000000 },
  ultimate: { maxMerchants: 2000, maxChannels: 4, monthlyAmount: 20000000 },
}

/** 各套餐价格（元/年） */
export const planPrice: Record<SitePlan, number> = {
  basic: 1980,
  pro: 5980,
  ultimate: 19800,
}

const siteNames = [
  { name: '云支付分站', domain: 'pay.yunfu.com', admin: 'yunfu_admin' },
  { name: '极速收银台', domain: 'cashier.jisu.cn', admin: 'jisu_op' },
  { name: '优商支付', domain: 'pay.youshang.net', admin: 'ys_master' },
  { name: '闪电聚合', domain: 'sd.shandian.shop', admin: 'sd_admin' },
  { name: '汇通支付平台', domain: 'pay.huitong.vip', admin: 'ht_boss' },
  { name: '小微收款', domain: 'shou.xiaowei.com', admin: 'xw_admin' },
  { name: '易付宝', domain: 'pay.yifubao.cn', admin: 'yfb_op' },
  { name: '聚宝盆支付', domain: 'pay.jubaopen.net', admin: 'jbp_admin' },
]
const plans: SitePlan[] = ['basic', 'pro', 'ultimate']

function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}

export const sites: Site[] = siteNames.map((s, i) => {
  const plan = plans[i % plans.length]
  const quota = planQuota[plan]
  const statusPool: Site['status'][] = [1, 1, 1, 0, 1, 2, 1, 3]
  const status = statusPool[i % statusPool.length]
  // 全模块 key
  const allKeys = allModules.map((m) => m.key)
  // 基础版少给风控，旗舰版全给
  const perms =
    plan === 'basic' ? allKeys.filter((k) => k !== 'risk') : allKeys
  const chScope = plan === 'basic' ? [1, 2] : [1, 2, 3, 4]
  const usageRatio = [0.3, 0.55, 0.82, 0.1, 0.68, 0, 0.45, 0.95][i % 8]
  return {
    id: 70000 + (siteNames.length - i),
    name: s.name,
    domain: s.domain,
    adminUser: s.admin,
    plan,
    status,
    createTime: `2026-0${1 + (i % 6)}-${pad(1 + (i % 27))} 10:00:00`,
    expireTime: status === 3 ? '2026-06-30 23:59:59' : `2027-0${1 + (i % 6)}-${pad(1 + (i % 27))} 23:59:59`,
    permissions: perms,
    channelScope: chScope,
    quota,
    usage: {
      merchants: Math.round(quota.maxMerchants * usageRatio),
      channels: Math.min(chScope.length, plan === 'basic' ? 2 : 3),
      monthlyAmount: Math.round(quota.monthlyAmount * usageRatio),
    },
  }
})

/** 控制台概况统计 */
export function calcConsoleStats(list: Site[]) {
  const now = list.filter((s) => s.status === 1).length
  const revenue = list.reduce((a, s) => a + planPrice[s.plan], 0)
  // 30 天内到期（简化：状态运行中且到期年份 2027 前几个月，示意用固定 2 个）
  const expiringSoon = list.filter((s) => s.status === 1).slice(0, 2).length
  return {
    total: list.length,
    running: now,
    stopped: list.filter((s) => s.status === 2 || s.status === 3).length,
    expiringSoon,
    revenue,
  }
}
