/** 后台仪表盘聚合 API（对齐 epay admin/index.php + ajax getcount）。 */
import { request } from './client'

export interface DashOverviewCard {
  label: string
  today: string
  yesterday: string
  total_label: string
  total: string
}
export interface DashTodo {
  pending_settle: number
  pending_domain: number
  pending_profit: number
  unpaid_orders: number
}
export interface DashTrend {
  labels: string[]
  orders: number[]
  amounts: string[]
}
export interface DashRecentOrder {
  trade_no: string
  uid: number
  typeshowname: string
  money: string
  status: number
  time: string
}
export interface AdminDashboard {
  overview: DashOverviewCard[]
  todo: DashTodo
  total_money: string
  settled_sum: string
  merchants: number
  orders_total: number
  success_rate: string
  trend: DashTrend
  recent: DashRecentOrder[]
}

export function fetchAdminDashboard(): Promise<AdminDashboard> {
  return request<AdminDashboard>('/admin/dashboard')
}
