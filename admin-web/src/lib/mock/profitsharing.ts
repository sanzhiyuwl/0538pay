/**
 * 分账记录假数据。对齐 epay admin/ps_order.php + ajax_profitsharing.php?act=orderList。
 * 数据表 pre_ps_order（分账订单）：一笔支付订单按分账规则拆出的分账明细。
 */

/** 分账记录（pre_ps_order） */
export interface PsOrder {
  id: number
  trade_no: string // 系统订单号（关联 pre_order）
  api_trade_no: string // 接口订单号
  rid: number // 分账规则 ID
  rulename: string // 规则名称
  channelid: number // 支付通道 ID
  channelname: string // 通道名称
  receiver: string // 分账接收方（名称）
  money: string // 分账金额
  addtime: string // 时间
  status: 0 | 1 | 2 | 3 | 4 // 0=待分账 1=已提交 2=分账成功 3=分账失败 4=已取消
  result: string // 失败原因
}

/** 分账状态字典（对齐 ps_order.php formatter） */
export const psStatus: Record<
  number,
  { text: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'muted' }
> = {
  0: { text: '待分账', variant: 'default' },
  1: { text: '已提交', variant: 'warning' },
  2: { text: '分账成功', variant: 'success' },
  3: { text: '分账失败', variant: 'destructive' },
  4: { text: '已取消', variant: 'muted' },
}

/** 搜索字段下拉（对齐 ps_order.php 的 column 选项） */
export const searchColumns = [
  { value: 'trade_no', label: '系统订单号' },
  { value: 'api_trade_no', label: '接口订单号' },
  { value: 'money', label: '分账金额' },
]

/** 支付通道（对齐 pre_channel，分账走微信/支付宝服务商） */
const channels = [
  { id: 2, name: '微信服务商' },
  { id: 1, name: '支付宝直连' },
  { id: 6, name: '微信官方' },
]

/** 分账规则 + 接收方（pre_ps_rule / pre_ps_receiver） */
const rules = [
  { rid: 1, rulename: '推广分成 5%', receiver: '推广渠道商A' },
  { rid: 2, rulename: '平台服务费 2%', receiver: '平台运营方' },
  { rid: 3, rulename: '供应商结算 30%', receiver: '优选供应商' },
  { rid: 4, rulename: '合伙人分润 10%', receiver: '城市合伙人' },
]

function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}

// 生成 52 条分账记录（确定性）
export const psOrders: PsOrder[] = Array.from({ length: 52 }, (_, i) => {
  const ch = channels[i % channels.length]
  const rule = rules[i % rules.length]
  // 状态分布：多为成功，少量待分账/已提交/失败/取消
  const statusPool: PsOrder['status'][] = [2, 2, 2, 0, 2, 1, 2, 3, 2, 4, 2, 1]
  const status = statusPool[i % statusPool.length]
  const moneyNum = +(((i * 27.3) % 1600) + 5).toFixed(2)
  const mm = pad((i * 7) % 60)
  const ss = pad((i * 17) % 60)
  return {
    id: 20000 + (52 - i),
    trade_no: `202607121${pad(4000 - i, 4)}${pad(i, 2)}`,
    api_trade_no: `${ch.id === 1 ? 'alipay' : 'wx'}${pad(88800 + i, 5)}`,
    rid: rule.rid,
    rulename: rule.rulename,
    channelid: ch.id,
    channelname: ch.name,
    receiver: rule.receiver,
    money: moneyNum.toFixed(2),
    addtime: `2026-07-12 ${pad(9 + (i % 10))}:${mm}:${ss}`,
    status,
    result: status === 3 ? '分账接收方账户状态异常，分账被拒绝' : '',
  }
})

/** 分账状态可执行的操作（对齐 ps_order.php 操作列） */
export function psActions(status: number): string[] {
  if (status === 0) return ['提交分账', '取消']
  if (status === 1) return ['查询结果']
  if (status === 2) return ['分账回退']
  if (status === 3) return ['重试', '取消']
  return [] // 已取消无操作
}

/** 分账汇总统计（对齐 act=statistics） */
export function calcPsStats(list: PsOrder[]) {
  const num = (s: string) => parseFloat(s) || 0
  const totalMoney = list.reduce((a, o) => a + num(o.money), 0)
  const successList = list.filter((o) => o.status === 2)
  const failList = list.filter((o) => o.status === 3)
  const successMoney = successList.reduce((a, o) => a + num(o.money), 0)
  const failMoney = failList.reduce((a, o) => a + num(o.money), 0)
  const totalCount = list.length
  const successRate = totalCount ? +((successList.length / totalCount) * 100).toFixed(2) : 0
  return {
    totalMoney,
    successMoney,
    failMoney,
    totalCount,
    successCount: successList.length,
    failCount: failList.length,
    successRate,
  }
}
