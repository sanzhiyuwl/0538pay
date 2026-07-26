/**
 * 平台月度对账账单 API（我方独有细化项，epay 无）。
 * 从后端订单/结算/代付/分账/退款五表归集的平台资金月度收支，对齐后端 dto.BillingResult。
 */
import { request } from './client'

/** 账单收支分项 */
export interface BillItem {
  label: string
  amount: string
  kind: 'income' | 'expense'
}

/** 月度账单 */
export interface MonthlyBill {
  period: string // 账期 YYYY-MM
  orders: number // 当期已支付订单数
  incomes: BillItem[]
  expenses: BillItem[]
  income: string // 收入合计
  expense: string // 支出合计
  net: string // 净收入
  status: 0 | 1 // 0=进行中(当月) 1=已归档
}

export interface BillingResult {
  bills: MonthlyBill[]
}

/** 拉取近 months 期（含当月）平台月度账单，倒序（最新在前）。 */
export function fetchBilling(months?: number): Promise<BillingResult> {
  return request<BillingResult>('/admin/billing', {
    query: months ? { months } : {},
  })
}

/** 导出账单 CSV（UTF-8 BOM，浏览器直接下载）。对齐 exportOrders 的鉴权下载模式。 */
export async function exportBilling(months?: number): Promise<void> {
  const token = localStorage.getItem('admin_token') || ''
  const qs = months ? `?months=${months}` : ''
  const res = await fetch(`/api/admin/billing/export${qs}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) throw new Error(`导出失败(${res.status})`)
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `平台账单_${new Date().toISOString().slice(0, 10)}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

/** 账单状态字典（0 进行中 / 1 已归档）。 */
export const billStatus: Record<number, { text: string; variant: 'warning' | 'success' }> = {
  0: { text: '进行中', variant: 'warning' },
  1: { text: '已归档', variant: 'success' },
}
