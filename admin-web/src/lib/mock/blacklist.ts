/**
 * 支付黑名单假数据。对齐 epay admin/blacklist.php + ajax_user.php?act=blackList（pre_black）。
 * 拉黑支付账号 / IP，命中则拦截支付（仅支持微信公众号支付、支付宝 JS 支付）。
 */

/** 黑名单（pre_black） */
export interface BlackItem {
  id: number
  type: 0 | 1 // 0=支付账号 1=IP地址
  content: string // 黑名单内容
  addtime: string // 添加时间
  endtime: string | null // 过期时间（null=永久）
  remark: string // 备注
}

/** 类型字典（对齐 blackList formatter） */
export const blackType: Record<number, { text: string; variant: 'default' | 'warning' }> = {
  0: { text: '支付账号', variant: 'default' },
  1: { text: 'IP地址', variant: 'warning' },
}

export const typeOptions = [
  { value: -1, label: '黑名单类型' },
  { value: 0, label: '支付账号' },
  { value: 1, label: 'IP地址' },
]

const remarks = ['恶意刷单', '欺诈投诉', '盗刷风险', '异常高频', '', '批量注册']

function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}

// 生成 22 条黑名单
export const blackItems: BlackItem[] = Array.from({ length: 22 }, (_, i) => {
  const type = (i % 3 === 0 ? 1 : 0) as 0 | 1
  const content =
    type === 1
      ? `${100 + (i % 100)}.${(i * 7) % 255}.${(i * 3) % 255}.${i % 255}`
      : type === 0 && i % 2 === 0
        ? `13${pad(i % 100)}****${pad((i * 7) % 10000, 4)}`
        : `oXyZ_user_${pad(i, 3)}`
  const day = 12 - (i % 8)
  // 部分永久（endtime null），部分有过期
  const permanent = i % 3 === 0
  return {
    id: 60000 + (22 - i),
    type,
    content,
    addtime: `2026-07-${pad(day)} ${pad(9 + (i % 10))}:${pad((i * 11) % 60)}:${pad((i * 17) % 60)}`,
    endtime: permanent ? null : `2026-${pad(8 + (i % 4))}-${pad(day)} 00:00:00`,
    remark: remarks[i % remarks.length],
  }
})

/** 汇总统计 */
export function calcBlackStats(list: BlackItem[]) {
  return {
    total: list.length,
    account: list.filter((b) => b.type === 0).length,
    ip: list.filter((b) => b.type === 1).length,
    permanent: list.filter((b) => b.endtime === null).length,
  }
}
