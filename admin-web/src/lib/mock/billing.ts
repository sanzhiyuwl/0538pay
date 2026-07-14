/**
 * 账单中心假数据（SaaS 增强项，epay 无直接对应）。
 * 平台财务对账视图：按月归集收入（手续费利润）与支出（结算打款 / 代付 / 分账 / 退款），
 * 生成可对账、可导出的月度账单。数据口径与订单/结算/分账/付款各模块一致。
 */

/** 账单收支分项 */
export interface BillItem {
  label: string
  amount: number
  kind: 'income' | 'expense'
}

/** 月度账单 */
export interface MonthlyBill {
  period: string // 账期 YYYY-MM
  opening: number // 期初平台留存
  incomes: BillItem[] // 收入分项
  expenses: BillItem[] // 支出分项
  status: 0 | 1 // 0=待对账 1=已对账
}

function round(n: number) {
  return +n.toFixed(2)
}

// 生成近 6 个月账单（确定性）
const months = ['2026-07', '2026-06', '2026-05', '2026-04', '2026-03', '2026-02']

export const bills: MonthlyBill[] = months.map((period, i) => {
  const base = 100000 - i * 8000
  const feeIncome = round(base * 0.018) // 手续费利润
  const channelCost = round(base * 0.006) // 通道成本支出
  const settleOut = round(base * 0.9) // 结算打款
  const transferOut = round(base * 0.12) // 代付打款
  const psOut = round(base * 0.05) // 分账支出
  const refundOut = round(base * 0.02) // 退款支出
  return {
    period,
    opening: round(50000 + i * 1200),
    incomes: [
      { label: '订单手续费利润', amount: feeIncome, kind: 'income' as const },
      { label: '会员套餐售卖', amount: round(base * 0.004), kind: 'income' as const },
      { label: '其他收入', amount: round(base * 0.001), kind: 'income' as const },
    ],
    expenses: [
      { label: '商户结算打款', amount: settleOut, kind: 'expense' as const },
      { label: '代付转账', amount: transferOut, kind: 'expense' as const },
      { label: '分账支出', amount: psOut, kind: 'expense' as const },
      { label: '订单退款', amount: refundOut, kind: 'expense' as const },
      { label: '通道成本', amount: channelCost, kind: 'expense' as const },
    ],
    status: (i === 0 ? 0 : 1) as 0 | 1,
  }
})

/** 账单派生汇总 */
export function billSummary(b: MonthlyBill) {
  const income = b.incomes.reduce((a, x) => a + x.amount, 0)
  const expense = b.expenses.reduce((a, x) => a + x.amount, 0)
  const net = income - expense
  return {
    income: round(income),
    expense: round(expense),
    net: round(net),
    closing: round(b.opening + net),
  }
}

/** 账单状态字典 */
export const billStatus: Record<number, { text: string; variant: 'warning' | 'success' }> = {
  0: { text: '待对账', variant: 'warning' },
  1: { text: '已对账', variant: 'success' },
}
