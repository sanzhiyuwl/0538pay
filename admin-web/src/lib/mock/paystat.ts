/**
 * 商户支付统计假数据。对齐 epay admin/ustat.php + ajax_user.php?act=userPayStat。
 * 交叉透视表：行=商户，列=各支付方式/通道，值=按统计口径(type)聚合的金额。
 */
import { merchants } from './merchants'
import { payTypes } from './orders'

/** 查看维度（method） */
export const methodOptions = [
  { value: 'type', label: '以支付方式查看' },
  { value: 'channel', label: '以支付通道查看' },
]

/** 统计口径（type），对齐 ustat.php 的下拉 */
export const statTypeOptions = [
  { value: 0, label: '订单金额统计' },
  { value: 1, label: '支付金额统计' },
  { value: 2, label: '分成金额统计' },
  { value: 3, label: '手续费利润统计' },
  { value: 4, label: '代付金额统计' },
]

/** 支付方式列（以支付方式查看） */
export const typeColumns = payTypes.map((t) => ({ key: t.name, label: t.showname }))

/** 支付通道列（以支付通道查看，对齐 pre_channel 常见通道） */
export const channelColumns = [
  { key: 'ch_alipay', label: '支付宝直连' },
  { key: 'ch_wx_sp', label: '微信服务商' },
  { key: 'ch_wx_gf', label: '微信官方' },
  { key: 'ch_union', label: '云闪付通道' },
]

/** 统计口径对应的缩放系数（相对订单金额），模拟不同口径的金额差异 */
const typeScale: Record<number, number> = {
  0: 1, // 订单金额
  1: 0.94, // 支付金额（扣未支付）
  2: 0.92, // 分成金额
  3: 0.018, // 手续费利润
  4: 0.15, // 代付金额
}

export interface StatRow {
  uid: number
  name: string // 商户标识（结算姓名，空则域名）
  values: Record<string, number> // 各列金额
  total: number // 行合计
}

// 参与统计的商户（去重 uid，取前若干）
const uniqMerchants = Array.from(new Map(merchants.map((m) => [m.uid, m])).values()).slice(0, 12)

function seedAmount(uid: number, colIdx: number) {
  // 确定性伪随机：基于 uid 和列索引
  const base = ((uid * 31 + colIdx * 97) % 40) * 137.5
  return +base.toFixed(2)
}

/**
 * 生成统计数据。
 * @param method 'type' | 'channel'
 * @param statType 0-4 统计口径
 */
export function buildStat(method: string, statType: number): { columns: { key: string; label: string }[]; rows: StatRow[] } {
  const columns = method === 'channel' ? channelColumns : typeColumns
  const scale = typeScale[statType] ?? 1
  const rows: StatRow[] = uniqMerchants.map((m) => {
    const values: Record<string, number> = {}
    let total = 0
    columns.forEach((col, ci) => {
      // 代付金额口径下，部分商户无代付：置 0
      const raw = statType === 4 && (m.uid % 3 === 0) ? 0 : seedAmount(m.uid, ci) * scale
      const v = +raw.toFixed(2)
      values[col.key] = v
      total += v
    })
    return {
      uid: m.uid,
      name: m.username || m.url,
      values,
      total: +total.toFixed(2),
    }
  })
  // 按合计降序
  rows.sort((a, b) => b.total - a.total)
  return { columns, rows }
}

/** 列合计（表尾汇总行） */
export function columnTotals(columns: { key: string }[], rows: StatRow[]) {
  const totals: Record<string, number> = {}
  let grand = 0
  for (const col of columns) {
    const sum = rows.reduce((a, r) => a + (r.values[col.key] || 0), 0)
    totals[col.key] = +sum.toFixed(2)
    grand += sum
  }
  return { totals, grand: +grand.toFixed(2) }
}
