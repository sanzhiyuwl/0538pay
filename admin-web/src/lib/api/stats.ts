/** 统计 / 登录日志 / 邀请码 API（C5）。对齐后端 dto。 */
import { request, type PageResult } from './client'

// ===== 商户支付统计（交叉透视表）=====
export interface StatColumn {
  key: string
  label: string
}
export interface StatRow {
  uid: number
  name: string
  values: Record<string, number>
  total: number
}
export interface StatResult {
  columns: StatColumn[]
  rows: StatRow[]
  totals: Record<string, number>
  grand: number
}
export interface StatParams {
  method?: string // type=按支付方式 / channel=按支付通道
  type?: number // 0订单金额 1支付金额 2分成金额 3手续费利润 4代付金额
  startday?: string
  endday?: string
}
export function fetchPayStat(params: StatParams = {}): Promise<StatResult> {
  return request<StatResult>('/admin/stat/pay', { query: { ...params } })
}

// ===== 登录日志（只读）=====
export interface LoginLog {
  id: number
  uid: number
  type: string
  ip: string
  city: string
  date: string
}
export interface LogParams {
  page?: number
  pageSize?: number
  column?: string
  value?: string
}
export function fetchLogs(params: LogParams = {}): Promise<PageResult<LoginLog>> {
  return request<PageResult<LoginLog>>('/admin/logs', { query: { ...params } })
}

// ===== 邀请码 =====
export interface InviteCode {
  id: number
  code: string
  status: number // 0未使用 1已使用
  addtime: string
  usetime: string | null
  uid: number | null
}
export interface InviteParams {
  page?: number
  pageSize?: number
  kw?: string
  status?: number
}
export function fetchInviteCodes(params: InviteParams = {}): Promise<PageResult<InviteCode>> {
  return request<PageResult<InviteCode>>('/admin/invitecodes', { query: { ...params } })
}
export function generateInviteCodes(num: number): Promise<{ generated: number }> {
  return request('/admin/invitecodes/generate', { method: 'POST', body: { num } })
}
export function deleteInviteCode(id: number): Promise<{ id: number }> {
  return request(`/admin/invitecodes/${id}`, { method: 'DELETE' })
}
export function clearInviteCodes(mode: 'all' | 'used'): Promise<{ deleted: number }> {
  return request('/admin/invitecodes/clear', { method: 'POST', body: { mode } })
}
