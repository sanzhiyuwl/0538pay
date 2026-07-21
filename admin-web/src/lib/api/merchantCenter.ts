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
  keytype: number // 0=MD5+RSA兼容 1=仅RSA安全
  has_rsa: boolean // 是否已配 RSA 公钥
}
export function fetchApiInfo(): Promise<ApiInfo> {
  return request<ApiInfo>('/merchant/apikey')
}
export function resetApiKey(): Promise<{ mdkey: string }> {
  return request('/merchant/apikey/reset', { method: 'POST' })
}
/** 生成商户 RSA 密钥对（V2），私钥一次性返回 */
export function genRsaKeyPair(): Promise<{ private_key: string }> {
  return request('/merchant/apikey/rsa', { method: 'POST' })
}
/** 设置签名模式 0=MD5+RSA兼容 1=仅RSA安全 */
export function setKeyType(keytype: number): Promise<{ keytype: number }> {
  return request('/merchant/apikey/keytype', { method: 'PUT', body: { keytype } })
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

// ===== 保证金（D3）=====
export interface DepositInfo {
  deposit: number
  depositMin: number
  money: number
}
export function fetchDepositInfo(): Promise<DepositInfo> {
  return request<DepositInfo>('/merchant/deposit')
}
export function rechargeDeposit(amount: string, payType = 'balance'): Promise<{ ok: boolean }> {
  return request('/merchant/deposit/recharge', { method: 'POST', body: { amount, pay_type: payType } })
}
export function withdrawDeposit(amount: string): Promise<{ ok: boolean }> {
  return request('/merchant/deposit/withdraw', { method: 'POST', body: { amount } })
}

// ===== 购买会员（D3）=====
export interface GroupRateItem {
  label: string
  rate: string
}
export interface GroupPlan {
  id: number
  name: string
  price: number
  expire: number // 月数，0=永久
  rates: GroupRateItem[]
}
export interface GroupCurrent {
  gid: number
  name: string
  expire: string // 到期时间，"—"=永久/无
}
export function fetchGroups(): Promise<{ plans: GroupPlan[]; current: GroupCurrent }> {
  return request('/merchant/groups')
}
export function buyGroup(gid: number, num: number, payType = 'balance'): Promise<{ ok: boolean }> {
  return request('/merchant/groups/buy', { method: 'POST', body: { gid, num, pay_type: payType } })
}

// ===== 余额充值（D3）=====
export interface RechargeResp {
  trade_no: string
  pay_type: string
  pay_url?: string
  qrcode?: string
  money: string
}
export function rechargeBalance(amount: string, plugin = 'mock'): Promise<RechargeResp> {
  return request<RechargeResp>('/merchant/recharge', { method: 'POST', body: { amount, plugin } })
}

// ===== 实名认证（D3，第三方认证待凭证）=====
export interface CertInfo {
  cert: number // 0未认证/审核中 1已认证
  certtype: number // 0个人 1企业
  certname: string
  certno: string
  certcorp: string
  certtime: string
  certmoney: number
  method: string
  corpopen: boolean
}
export function fetchCertInfo(): Promise<CertInfo> {
  return request<CertInfo>('/merchant/cert')
}
export function submitCert(body: { certtype: number; certname: string; certno: string; certcorp: string }): Promise<{ ok: boolean }> {
  return request('/merchant/cert', { method: 'POST', body })
}
