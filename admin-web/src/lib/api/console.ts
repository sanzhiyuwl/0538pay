/**
 * 代理控制台 API（平台运营视角，管所有代理进件；自研扩展）。
 * 走 admin token（与后台共用）。接口前缀 /console/*。
 */
import { request, type PageResult } from './client'

// —— 权限点清单 ——
export interface AgentPermission {
  key: string
  name: string
  group: string
  desc: string
}
export function fetchAgentPermissions(): Promise<AgentPermission[]> {
  return request<AgentPermission[]>('/console/agent-permissions')
}

// —— 代理 ——
export interface Agent {
  id: number
  name: string
  account: string
  contact: string
  status: number
  permissions: string
  remark: string
}
export interface AgentSaveReq {
  name: string
  account: string
  password?: string
  contact?: string
  remark?: string
  status?: number
  permissions: string[]
}
export function fetchAgents(params: {
  keyword?: string
  status?: number
  page?: number
  pageSize?: number
} = {}): Promise<PageResult<Agent>> {
  return request<PageResult<Agent>>('/console/agents', { query: { ...params } })
}
export function getAgent(id: number): Promise<Agent> {
  return request<Agent>(`/console/agents/${id}`)
}
export function createAgent(body: AgentSaveReq): Promise<{ id: number }> {
  return request<{ id: number }>('/console/agents', { method: 'POST', body })
}
export function updateAgent(id: number, body: AgentSaveReq) {
  return request(`/console/agents/${id}`, { method: 'PUT', body })
}
export function setAgentStatus(id: number, status: number) {
  return request(`/console/agents/${id}/status`, { method: 'PUT', body: { status } })
}
export function deleteAgent(id: number) {
  return request(`/console/agents/${id}`, { method: 'DELETE' })
}

// —— 名额 ——
export interface QuotaWallet {
  agent_id: number
  balance: number
  total_buy: number
  total_used: number
}
export interface QuotaLog {
  id: number
  agent_id: number
  type: string
  change: number
  before: number
  after: number
  rel_no: string
}
export function getAgentWallet(id: number): Promise<QuotaWallet> {
  return request<QuotaWallet>(`/console/agents/${id}/quota`)
}
export function adjustQuota(id: number, body: { change: number; amount?: string; remark?: string }) {
  return request(`/console/agents/${id}/quota`, { method: 'POST', body })
}
export function fetchQuotaLogs(params: {
  agent_id?: number
  page?: number
  pageSize?: number
} = {}): Promise<PageResult<QuotaLog>> {
  return request<PageResult<QuotaLog>>('/console/quota-logs', { query: { ...params } })
}

// —— 进件申请 ——
export interface Enroll {
  id: number
  enroll_no: string
  agent_id: number
  merchant_name: string
  subject_type: string
  contact_phone: string
  path: number
  pay_order_no: string
  business_code: string
  wx_applyment_id: string
  wx_sub_mchid: string
  status: string
  reject_reason: string
  source: number
  invite_code: string
}
export function fetchEnrolls(params: {
  keyword?: string
  status?: string
  agent_id?: number
  source?: number
  page?: number
  pageSize?: number
} = {}): Promise<PageResult<Enroll>> {
  return request<PageResult<Enroll>>('/console/enrolls', { query: { ...params } })
}
export function getEnroll(id: number): Promise<Enroll> {
  return request<Enroll>(`/console/enrolls/${id}`)
}
// 建进件单（付费前置：建单即下开户费收款，返回收银台信息）
export interface CreateEnrollResp {
  enroll: Enroll
  pay: { trade_no: string; pay_url?: string; qrcode?: string; money: string } | null
}
export function createEnroll(body: {
  merchant_name: string
  contact_phone?: string
  path: number
  agent_id?: number
  plugin?: string
}): Promise<CreateEnrollResp> {
  return request<CreateEnrollResp>('/console/enrolls', { method: 'POST', body })
}
// 提交微信审核（需已支付待完善 / 被驳回后重提）
export function submitEnroll(id: number): Promise<Enroll> {
  return request<Enroll>(`/console/enrolls/${id}/submit`, { method: 'POST' })
}
// 主动拉取微信申请单最新状态，推进本地状态机
export function syncEnroll(id: number): Promise<Enroll> {
  return request<Enroll>(`/console/enrolls/${id}/sync`, { method: 'POST' })
}

// —— 邀请链接 ——
export interface Invite {
  id: number
  code: string
  agent_id: number
  name: string
  status: number
  open_count: number
  submit_count: number
}
export function fetchInvites(params: {
  agent_id?: number
  page?: number
  pageSize?: number
} = {}): Promise<PageResult<Invite>> {
  return request<PageResult<Invite>>('/console/enroll-invites', { query: { ...params } })
}
export function createInvite(body: { agent_id?: number; name?: string }): Promise<Invite> {
  return request<Invite>('/console/enroll-invites', { method: 'POST', body })
}
export function setInviteStatus(id: number, status: number) {
  return request(`/console/enroll-invites/${id}/status`, { method: 'PUT', body: { status } })
}
export function deleteInvite(id: number) {
  return request(`/console/enroll-invites/${id}`, { method: 'DELETE' })
}

// —— 佣金结算 ——
export interface Settlement {
  id: number
  enroll_id: number
  agent_id: number
  path: number
  pay_order_no: string
}
export function fetchSettlements(params: {
  agent_id?: number
  page?: number
  pageSize?: number
} = {}): Promise<PageResult<Settlement>> {
  return request<PageResult<Settlement>>('/console/enroll-settlements', { query: { ...params } })
}
