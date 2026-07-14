/**
 * 商户中心「资金明细」假数据。对齐 epay user/record.php + ajax2.php?act=recordList（pre_record）。
 * 余额变动流水。action==2 为支出（红），否则收入（绿）。
 */

/** 资金明细记录 */
export interface FundRecord {
  id: number
  type: string // 操作类型
  action: 1 | 2 // 1收入 2支出
  money: number // 变更金额（正数，展示时按 action 加 +/-）
  oldmoney: number // 变更前余额
  newmoney: number // 变更后余额
  date: string // 时间
  trade_no: string // 关联订单号（可空）
}

/** 搜索类型（对齐 record.php type 下拉） */
export const searchColumns = [
  { value: 'type', label: '操作类型' },
  { value: 'money', label: '变更金额' },
  { value: 'trade_no', label: '关联订单号' },
]

// 操作类型池：收入类 / 支出类
const incomeTypes = ['订单收入', '邀请返现', '余额充值', '退款回退']
const expenseTypes = ['订单退款', '申请提现', '发起代付', '购买会员', '保证金充值', '结算扣除']

function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}

// 生成 50 条流水，余额连续变化
let balance = 8000
export const fundRecords: FundRecord[] = Array.from({ length: 50 }, (_, i) => {
  const isIncome = i % 3 !== 0 // 约 2/3 收入
  const action = (isIncome ? 1 : 2) as 1 | 2
  const type = isIncome ? incomeTypes[i % incomeTypes.length] : expenseTypes[i % expenseTypes.length]
  const money = Math.round((10 + ((i * 53) % 1500) + (i % 100) / 100) * 100) / 100
  const oldmoney = Math.round(balance * 100) / 100
  balance = isIncome ? balance + money : balance - money
  const newmoney = Math.round(balance * 100) / 100
  const day = 12 - (i % 10)
  const hasOrder = ['订单收入', '订单退款', '邀请返现'].includes(type)
  return {
    id: 70000 + (50 - i),
    type,
    action,
    money,
    oldmoney,
    newmoney,
    date: `2026-07-${pad(day)} ${pad(8 + (i % 12))}:${pad((i * 7) % 60)}:${pad((i * 13) % 60)}`,
    trade_no: hasOrder ? `2026071${pad(day)}${pad(100000 + i * 137, 6)}`.slice(0, 19) : '',
  }
})

/** 汇总统计 */
export function calcRecordStats(list: FundRecord[]) {
  const income = list.filter((r) => r.action === 1).reduce((s, r) => s + r.money, 0)
  const expense = list.filter((r) => r.action === 2).reduce((s, r) => s + r.money, 0)
  return {
    count: list.length,
    income,
    expense,
    net: income - expense,
  }
}
