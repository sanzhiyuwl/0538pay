/** 商户中心业务 API（工作台/订单/流水/结算/提现/退款）。类型复用 mock 里已定义的结构。 */
import { request, type PageResult } from './client'
import type { Order } from '@/lib/mock/merchant/orders'
import type { FundRecord } from '@/lib/mock/merchant/records'
import type { SettleRecord } from '@/lib/mock/merchant/settle'

// ===== 工作台聚合 =====
export interface DashboardInfo {
  uid: number
  name: string
  qq: string
  status: string // normal/banned/payoff/settleoff/auditing/uncert
  groupName: string
  money: number
  settleMoney: number
  todayIncome: number
  yesterdayIncome: number
  orders: number
  ordersToday: number
}
export interface DashboardAlerts {
  needCert: boolean
  noSecurity: boolean
  noLoginPwd: boolean
}
export interface DashboardChannel {
  typename: string
  showname: string
  today: number
  yesterday: number
  successRate: number
  rate: string
}
export interface DashboardAnnounce {
  id: number
  content: string
  color: string
  time: string
}
export interface DashboardTrend {
  labels: string[]
  data: number[]
}
export interface MerchantDashboard {
  merchantInfo: DashboardInfo
  alerts: DashboardAlerts
  channels: DashboardChannel[]
  announces: DashboardAnnounce[]
  trend: DashboardTrend
}

export function fetchDashboard(): Promise<MerchantDashboard> {
  return request<MerchantDashboard>('/merchant/dashboard')
}

// ===== 订单查询 =====
export interface MerchantOrderParams {
  page?: number
  pageSize?: number
  column?: string
  keyword?: string
  status?: number
}
export function fetchMerchantOrders(
  params: MerchantOrderParams = {},
): Promise<PageResult<Order>> {
  return request<PageResult<Order>>('/merchant/orders', { query: { ...params } })
}

/** 订单退款（全额） */
export function refundOrder(tradeNo: string): Promise<{ trade_no: string; status: number }> {
  return request('/merchant/order/refund', { method: 'POST', body: { trade_no: tradeNo } })
}

/** 重新通知（补单/重发回调） */
export function renotifyOrder(tradeNo: string): Promise<{ trade_no: string }> {
  return request('/merchant/order/notify', { method: 'POST', body: { trade_no: tradeNo } })
}

// ===== 资金流水 =====
export interface MerchantRecordParams {
  page?: number
  pageSize?: number
  action?: number
  keyword?: string
}
export function fetchMerchantRecords(
  params: MerchantRecordParams = {},
): Promise<PageResult<FundRecord>> {
  return request<PageResult<FundRecord>>('/merchant/records', { query: { ...params } })
}

// ===== 结算记录 =====
export function fetchMerchantSettles(
  params: { page?: number; pageSize?: number; status?: number } = {},
): Promise<PageResult<SettleRecord>> {
  return request<PageResult<SettleRecord>>('/merchant/settles', { query: { ...params } })
}

// ===== 申请提现 =====
export interface ApplyInfo {
  settleName: string
  account: string
  username: string
  money: number
  enableMoney: number
  settleMin: number
  settleMaxLimit: number
  settleRate: number
  settleFeeMin: number
  settleFeeMax: number
  settleType: number
  todayCount: number
}
export function fetchApplyInfo(): Promise<ApplyInfo> {
  return request<ApplyInfo>('/merchant/apply/info')
}
export function submitApply(amount: string): Promise<{ ok: boolean }> {
  return request('/merchant/apply', { method: 'POST', body: { amount } })
}

// ===== API 信息 / 资料 / 密码（D3）=====
export interface ApiInfo {
  uid: number
  mdkey: string
  apiurl: string
}
export function fetchApiInfo(): Promise<ApiInfo> {
  return request<ApiInfo>('/merchant/apikey')
}
export function resetApiKey(): Promise<{ mdkey: string }> {
  return request('/merchant/apikey/reset', { method: 'POST' })
}

export interface ProfileReq {
  settle_id: number
  account: string
  username: string
  email: string
  qq: string
  url: string
  mode: number
}
export function updateProfile(body: ProfileReq): Promise<{ ok: boolean }> {
  return request('/merchant/profile', { method: 'PUT', body })
}

export function changePassword(oldpwd: string, newpwd: string): Promise<{ ok: boolean }> {
  return request('/merchant/password', { method: 'PUT', body: { oldpwd, newpwd } })
}
