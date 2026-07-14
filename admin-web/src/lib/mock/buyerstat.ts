/**
 * 支付用户统计假数据。对齐 epay admin/buyerstat.php（菜单「支付用户统计」）。
 * 区别于「支付统计」的商户维度——本页按【付款人】维度聚合：
 * 同一付款用户（微信 openid / 支付宝 uid）的付款次数、累计金额、首末付款时间、关联商户数。
 */

import { formatMoney } from '@/lib/utils'

/** 付款渠道类型 */
export type BuyerChannel = 'wxpay' | 'alipay' | 'qqpay'

export const channelMeta: Record<BuyerChannel, { text: string; variant: 'success' | 'default' | 'warning' }> = {
  wxpay: { text: '微信', variant: 'success' },
  alipay: { text: '支付宝', variant: 'default' },
  qqpay: { text: 'QQ钱包', variant: 'warning' },
}

export const channelOptions = [
  { value: 'all', label: '全部渠道' },
  { value: 'wxpay', label: '微信' },
  { value: 'alipay', label: '支付宝' },
  { value: 'qqpay', label: 'QQ钱包' },
]

/** 排序口径 */
export const sortOptions = [
  { value: 'amount', label: '按累计金额' },
  { value: 'count', label: '按付款次数' },
  { value: 'last', label: '按最近付款' },
]

/** 付款人统计行 */
export interface BuyerStat {
  buyerId: string // 付款人标识（openid / 支付宝 uid，脱敏展示）
  channel: BuyerChannel
  count: number // 付款次数
  amount: number // 累计金额（元）
  merchants: number // 关联商户数
  firstTime: string // 首次付款
  lastTime: string // 最近付款
}

function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}

const channels: BuyerChannel[] = ['wxpay', 'alipay', 'qqpay']

function maskId(channel: BuyerChannel, seed: number): string {
  if (channel === 'wxpay') return `oXyZ${pad(seed % 1000, 3)}****${pad((seed * 7) % 10000, 4)}`
  if (channel === 'alipay') return `2088${pad((seed * 13) % 100000000, 8)}`
  return `QQ_${pad((seed * 17) % 1000000000, 9)}`
}

// 生成 40 个付款人
export const buyerStats: BuyerStat[] = Array.from({ length: 40 }, (_, i) => {
  const channel = channels[i % 3]
  const count = 1 + ((i * 7) % 48)
  const avg = 500 + ((i * 137) % 9500) // 分
  const firstDay = 1 + (i % 10)
  const lastDay = 12 - (i % 6)
  return {
    buyerId: maskId(channel, i + 1),
    channel,
    count,
    amount: (count * avg) / 100,
    merchants: 1 + ((i * 3) % 5),
    firstTime: `2026-07-${pad(firstDay)} ${pad(8 + (i % 12))}:${pad((i * 11) % 60)}:${pad((i * 7) % 60)}`,
    lastTime: `2026-07-${pad(lastDay)} ${pad(9 + (i % 10))}:${pad((i * 13) % 60)}:${pad((i * 17) % 60)}`,
  }
})

/** 排序（返回新数组，不改原始） */
export function sortBuyers(list: BuyerStat[], by: string): BuyerStat[] {
  const arr = [...list]
  if (by === 'count') arr.sort((a, b) => b.count - a.count)
  else if (by === 'last') arr.sort((a, b) => b.lastTime.localeCompare(a.lastTime))
  else arr.sort((a, b) => b.amount - a.amount)
  return arr
}

/** 汇总统计 */
export function calcBuyerStats(list: BuyerStat[]) {
  const amount = list.reduce((s, b) => s + b.amount, 0)
  const count = list.reduce((s, b) => s + b.count, 0)
  return {
    buyers: list.length,
    count,
    amount,
    avg: count ? amount / count : 0,
  }
}

export { formatMoney }
