/**
 * 资金明细假数据。对齐 epay admin/record.php + ajax_user.php?act=recordList（pre_record）。
 * 每条 = 商户余额的一次变更流水：action 区分增(1)/减(2)，记录变更前后余额。
 */
import { merchants } from './merchants'

/** 资金明细（pre_record） */
export interface FundRecord {
  id: number
  uid: number // 商户号
  type: string // 操作类型（文案）
  action: 1 | 2 // 1=增加(绿) 2=减少(红)
  money: string // 变更金额
  oldmoney: string // 变更前余额
  newmoney: string // 变更后余额
  date: string // 时间
  trade_no: string // 关联订单号（可空）
}

/** 搜索字段下拉（对齐 record.php 的 column 选项） */
export const searchColumns = [
  { value: 'type', label: '操作类型' },
  { value: 'money', label: '变更金额' },
  { value: 'trade_no', label: '关联订单号' },
]

/** 资金变更类型（action=1 入账 / action=2 出账），对齐易支付资金流水场景 */
const incTypes = ['订单收入', '余额充值', '邀请返现', '保证金解冻', '手动增加']
const decTypes = ['结算扣款', '提现转账', '订单退款', '保证金冻结', '手动扣除']

/** 类型下拉（供筛选，收入在前、支出在后） */
export const typeOptions = [
  { value: '', label: '全部类型' },
  ...incTypes.map((t) => ({ value: t, label: t })),
  ...decTypes.map((t) => ({ value: t, label: t })),
]

function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}

// 各商户维护一个"当前余额"游标，保证 oldmoney/newmoney 链条自洽
const balanceCursor: Record<number, number> = {}
for (const m of merchants) balanceCursor[m.uid] = +(2000 + (m.uid % 100) * 37).toFixed(2)

// 生成 64 条资金明细（按时间倒序展示，但先正序累积余额再反转）
const uids = merchants.map((m) => m.uid)
const rawRecords: FundRecord[] = []
for (let i = 0; i < 64; i++) {
  const uid = uids[i % uids.length]
  // 偶数入账、每第 3 条出账，制造增减混合
  const isDec = i % 3 === 0
  const action: 1 | 2 = isDec ? 2 : 1
  const typeArr = isDec ? decTypes : incTypes
  const type = typeArr[i % typeArr.length]
  const moneyNum = +(((i * 41.3) % 1500) + 8).toFixed(2)
  const old = balanceCursor[uid]
  // 出账不能把余额扣成负数：不足则改为入账等额，保持链条真实
  let newBal: number
  let realAction = action
  if (action === 2 && old < moneyNum) {
    realAction = 1
    newBal = +(old + moneyNum).toFixed(2)
  } else {
    newBal = +(realAction === 2 ? old - moneyNum : old + moneyNum).toFixed(2)
  }
  balanceCursor[uid] = newBal
  const hasTrade = type === '订单收入' || type === '订单退款' || type === '结算扣款'
  const mm = pad((i * 7) % 60)
  const ss = pad((i * 19) % 60)
  const hh = pad(8 + (i % 12))
  rawRecords.push({
    id: 30000 + i,
    uid,
    type: realAction === 1 && isDec ? incTypes[i % incTypes.length] : type,
    action: realAction,
    money: moneyNum.toFixed(2),
    oldmoney: old.toFixed(2),
    newmoney: newBal.toFixed(2),
    date: `2026-07-${pad(12 - (i % 6))} ${hh}:${mm}:${ss}`,
    trade_no: hasTrade ? `202607121${pad(4000 - i, 4)}${pad(i, 2)}` : '',
  })
}

// 展示按 id 倒序（最新在前）
export const fundRecords: FundRecord[] = rawRecords.slice().reverse()

/** 资金明细汇总（对齐 act=record_stats） */
export function calcRecordStats(list: FundRecord[]) {
  const num = (s: string) => parseFloat(s) || 0
  const incMoney = list.filter((r) => r.action === 1).reduce((a, r) => a + num(r.money), 0)
  const decMoney = list.filter((r) => r.action === 2).reduce((a, r) => a + num(r.money), 0)
  return {
    incMoney,
    decMoney,
    totalMoney: incMoney - decMoney,
    incCount: list.filter((r) => r.action === 1).length,
    decCount: list.filter((r) => r.action === 2).length,
    totalCount: list.length,
  }
}
