/**
 * 风控记录假数据。对齐 epay admin/risk.php + ajax_order.php?act=riskList（pre_risk）。
 * 系统自动触发的风控命中记录：关键词屏蔽 / 订单成功率 / 连续通知失败 / 订单投诉率。
 */

/** 风控记录（pre_risk） */
export interface RiskRecord {
  id: number
  uid: number // 商户号
  type: 0 | 1 | 2 | 3 // 0=关键词屏蔽 1=订单成功率 2=连续通知失败 3=订单投诉率
  content: string // 风控内容
  url: string // 风控网址
  date: string // 时间
}

/** 风控类型字典（对齐 riskList formatter） */
export const riskType: Record<
  number,
  { text: string; variant: 'destructive' | 'warning' | 'default' | 'muted' }
> = {
  0: { text: '关键词屏蔽', variant: 'destructive' },
  1: { text: '订单成功率', variant: 'warning' },
  2: { text: '连续通知失败', variant: 'default' },
  3: { text: '订单投诉率', variant: 'muted' },
}

/** 搜索字段下拉（对齐 risk.php column 选项） */
export const searchColumns = [
  { value: 'uid', label: '商户号' },
  { value: 'url', label: '风控网址' },
  { value: 'content', label: '风控内容' },
]

/** 类型筛选 */
export const typeOptions = [
  { value: -1, label: '风控类型' },
  ...Object.entries(riskType).map(([k, t]) => ({ value: Number(k), label: t.text })),
]

const domains = ['shop.example.com', 'geek.example.com', 'flower.example.com', 'book.example.com', 'charge.example.com']
// 各类型对应的风控内容样例
const contentByType: Record<number, string[]> = {
  0: ['商品名含违禁词「', '订单备注含敏感词「', '商品标题命中黑词「'],
  1: ['24小时订单成功率 31%，低于阈值 60%', '近1小时成功率 42%，触发预警', '当日成功率跌至 55%'],
  2: ['异步通知连续失败 18 次', '回调地址连续超时 12 次', '通知失败累计 25 次，已暂停'],
  3: ['当日投诉率 3.2%，超过阈值 2%', '近7日投诉 8 单，触发风控', '投诉率 4.1%，已限制收款'],
}
const keywords = ['充值', '博彩', '代付', '虚拟货币', 'VPN']

function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}

// 生成 40 条风控记录
export const riskRecords: RiskRecord[] = Array.from({ length: 40 }, (_, i) => {
  const type = (i % 4) as 0 | 1 | 2 | 3
  const uid = 1000 + (i % 12) + 1
  const pool = contentByType[type]
  let content = pool[i % pool.length]
  if (type === 0) content += keywords[i % keywords.length] + '」'
  const day = 12 - (i % 7)
  return {
    id: 50000 + (40 - i),
    uid,
    type,
    content,
    url: domains[i % domains.length],
    date: `2026-07-${pad(day)} ${pad(8 + (i % 12))}:${pad((i * 7) % 60)}:${pad((i * 13) % 60)}`,
  }
})

/** 汇总统计 */
export function calcRiskStats(list: RiskRecord[]) {
  return {
    total: list.length,
    keyword: list.filter((r) => r.type === 0).length,
    successRate: list.filter((r) => r.type === 1).length,
    notifyFail: list.filter((r) => r.type === 2).length,
    complaint: list.filter((r) => r.type === 3).length,
  }
}
