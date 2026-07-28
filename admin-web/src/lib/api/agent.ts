/**
 * 独立代理端 /agent API（代理只看/只碰自己名下；自研扩展）。
 * 走 agent_token（独立于 admin/merchant token）。接口前缀 /agent/*。
 * agent_id 由后端从 token 取，前端不传，天然数据隔离。
 */
import { request, setAgentToken, type PageResult } from './client'
import type {
  Agent,
  AgentPermission,
  QuotaWallet,
  QuotaLog,
  Enroll,
  CreateEnrollResp,
  Invite,
  Settlement,
  RefundResult,
  EnrollMaterialReq,
  EnrollMaterialView,
} from './console'

// —— 登录 / 会话 ——
export interface AgentLoginResp {
  token: string
  name: string
  account: string
  permissions: string
}
export async function agentLogin(body: { account: string; password: string }): Promise<AgentLoginResp> {
  const data = await request<AgentLoginResp>('/agent/login', { method: 'POST', body })
  setAgentToken(data.token) // 成功后存独立 agent_token
  return data
}
export function fetchAgentProfile(): Promise<Agent> {
  return request<Agent>('/agent/profile')
}
export function fetchAgentPermissionCatalog(): Promise<AgentPermission[]> {
  return request<AgentPermission[]>('/agent/permissions')
}

// —— 名额钱包 ——
export function fetchMyWallet(): Promise<QuotaWallet> {
  return request<QuotaWallet>('/agent/quota')
}
export function fetchMyQuotaLogs(params: { page?: number; pageSize?: number } = {}): Promise<PageResult<QuotaLog>> {
  return request<PageResult<QuotaLog>>('/agent/quota-logs', { query: { ...params } })
}

// —— 进件申请 ——
export function fetchMyEnrolls(params: {
  keyword?: string
  status?: string
  page?: number
  pageSize?: number
} = {}): Promise<PageResult<Enroll>> {
  return request<PageResult<Enroll>>('/agent/enrolls', { query: { ...params } })
}
export function getMyEnroll(id: number): Promise<Enroll> {
  return request<Enroll>(`/agent/enrolls/${id}`)
}
export function createMyEnroll(body: {
  merchant_name: string
  contact_phone?: string
  path: number
  plugin?: string
}): Promise<CreateEnrollResp> {
  return request<CreateEnrollResp>('/agent/enrolls', { method: 'POST', body })
}
export function submitMyEnroll(id: number): Promise<Enroll> {
  return request<Enroll>(`/agent/enrolls/${id}/submit`, { method: 'POST' })
}
export function syncMyEnroll(id: number): Promise<Enroll> {
  return request<Enroll>(`/agent/enrolls/${id}/sync`, { method: 'POST' })
}
// 手动退款：原路退全额开户费（需 refund 权限 + 只退自己名下；四道拦截在后端校验）
export function refundMyEnroll(id: number): Promise<RefundResult> {
  return request<RefundResult>(`/agent/enrolls/${id}/refund`, { method: 'POST' })
}
// 填全套资料（只碰自己名下；敏感字段后端加密落库）
export function getMyEnrollMaterial(id: number): Promise<EnrollMaterialView> {
  return request<EnrollMaterialView>(`/agent/enrolls/${id}/material`)
}
export function fillMyEnrollMaterial(id: number, body: EnrollMaterialReq): Promise<Enroll> {
  return request<Enroll>(`/agent/enrolls/${id}/material`, { method: 'POST', body })
}

// —— 邀请链接 ——
export function fetchMyInvites(params: { page?: number; pageSize?: number } = {}): Promise<PageResult<Invite>> {
  return request<PageResult<Invite>>('/agent/enroll-invites', { query: { ...params } })
}
export function createMyInvite(body: { name?: string }): Promise<Invite> {
  return request<Invite>('/agent/enroll-invites', { method: 'POST', body })
}
export function setMyInviteStatus(id: number, status: number) {
  return request(`/agent/enroll-invites/${id}/status`, { method: 'PUT', body: { status } })
}
export function deleteMyInvite(id: number) {
  return request(`/agent/enroll-invites/${id}`, { method: 'DELETE' })
}

// —— 佣金结算 ——
export function fetchMySettlements(params: { page?: number; pageSize?: number } = {}): Promise<PageResult<Settlement>> {
  return request<PageResult<Settlement>>('/agent/enroll-settlements', { query: { ...params } })
}
