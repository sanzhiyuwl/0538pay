/**
 * 代理控制台 API（平台运营视角，管所有代理进件；自研扩展）。
 * 走 admin token（与后台共用）。接口前缀 /console/*。
 */
import { request, upload, type PageResult } from './client'

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
// 只更新权限点（权限分配独立页用，不动名称/账号/备注）
export function setAgentPermissions(id: number, permissions: string[]) {
  return request(`/console/agents/${id}/permissions`, { method: 'PUT', body: { permissions } })
}
export function deleteAgent(id: number) {
  return request(`/console/agents/${id}`, { method: 'DELETE' })
}

// —— 名额 ——
export interface QuotaWallet {
  agent_id: number
  balance: number
  frozen: number
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
// 退款结果回显
export interface RefundResult {
  enroll_no: string
  merchant_name: string
  status: string
  executed: boolean
  msg: string
}
// 平台兜底退款：原路退全额开户费（四道拦截 + sub_mchid 硬锁在后端校验）
export function refundEnroll(id: number): Promise<RefundResult> {
  return request<RefundResult>(`/console/enrolls/${id}/refund`, { method: 'POST' })
}

// —— 填全套资料（applyment4sub 核心字段；敏感字段后端 RSA-OAEP 加密，不明文落库）——
// 提交入参：敏感字段前端明文提交，后端加密。图片类填 media_id 占位（一期不做上传）。
export interface EnrollMaterialReq {
  subject_type: string
  merchant_shortname: string
  service_phone: string
  license_number: string
  license_copy: string
  legal_person: string
  license_address: string
  period_begin: string
  period_end: string
  id_card_name: string
  id_card_number: string
  id_card_copy: string
  id_card_national: string
  card_period_begin: string
  card_period_end: string
  bank_account_type: string
  account_name: string
  account_bank: string
  bank_address_code: string
  account_number: string
  contact_name: string
  contact_id_number: string
  mobile_phone: string
  contact_email: string
}
// 回显：敏感字段一律不回原文，只回 has_* 是否已填。
export interface EnrollMaterialView {
  filled: boolean
  subject_type: string
  merchant_shortname: string
  service_phone: string
  license_number: string
  license_copy: string
  legal_person: string
  license_address: string
  period_begin: string
  period_end: string
  id_card_copy: string
  id_card_national: string
  card_period_begin: string
  card_period_end: string
  bank_account_type: string
  account_bank: string
  bank_address_code: string
  has_id_card_name: boolean
  has_id_card_number: boolean
  has_account_name: boolean
  has_account_number: boolean
  has_contact_name: boolean
  has_contact_id_number: boolean
  has_mobile_phone: boolean
  has_contact_email: boolean
}
export function getEnrollMaterial(id: number): Promise<EnrollMaterialView> {
  return request<EnrollMaterialView>(`/console/enrolls/${id}/material`)
}
export function fillEnrollMaterial(id: number, body: EnrollMaterialReq): Promise<Enroll> {
  return request<Enroll>(`/console/enrolls/${id}/material`, { method: 'POST', body })
}
/** 上传一张进件资料图片（营业执照/身份证），返回微信 media_id。 */
export function uploadEnrollMedia(id: number, file: File): Promise<{ media_id: string }> {
  const form = new FormData()
  form.append('file', file)
  return upload<{ media_id: string }>(`/console/enrolls/${id}/media`, form)
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

// —— 微信服务商凭证（脱敏读 / 保存；私钥·公钥落后端 secrets/ 文件，不入库不对外）——
export interface WxPartnerView {
  sp_mchid: string
  sp_appid: string
  serial_no: string
  public_key_id: string
  has_private_key: boolean
  has_public_key: boolean
  has_apiv3_key: boolean
  private_key_fp: string // 私钥内容指纹(SHA256前12位)，空=未配
  public_key_fp: string
  apiv3_key_fp: string
  configured: boolean
}
export interface WxPartnerSaveReq {
  sp_mchid: string
  sp_appid: string
  serial_no: string
  public_key_id: string
  private_key?: string // 留空=不改
  public_key?: string // 留空=不改
  apiv3_key?: string // 留空=不改
}
export function getWxPartner(): Promise<WxPartnerView> {
  return request<WxPartnerView>('/console/wx-partner')
}
export function saveWxPartner(body: WxPartnerSaveReq): Promise<WxPartnerView> {
  return request<WxPartnerView>('/console/wx-partner', { method: 'PUT', body })
}

// —— 代理操作日志（scope=agent，复用 pay_oplog 表；平台查看代理在 /agent 端的写操作审计）——
export interface AgentOpLog {
  id: number
  scope: string
  uid: number // 代理 ID
  operator: string // 代理名（冗余存）
  action: string
  actionCN: string
  category: string // enroll/fund
  level: string // normal/warning/danger
  target: string
  detail: string
  result: string // ok/fail
  ip: string
  date: string
}
export interface AgentOpLogParams {
  page?: number
  pageSize?: number
  uid?: number
  action?: string
  level?: string
  result?: string
  keyword?: string
  starttime?: string
  endtime?: string
}
export interface AgentOpActionOption {
  value: string
  label: string
  category: string
  level: string
}
export function fetchAgentOpLogs(params: AgentOpLogParams = {}): Promise<PageResult<AgentOpLog>> {
  return request<PageResult<AgentOpLog>>('/console/agent-oplogs', { query: { ...params } })
}
export function fetchAgentOpActionOptions(): Promise<{ actions: AgentOpActionOption[] }> {
  return request<{ actions: AgentOpActionOption[] }>('/console/agent-oplogs/options')
}
/** 导出代理操作日志 CSV（UTF-8 BOM，浏览器直接下载）。走 admin token。 */
export async function exportAgentOpLogs(params: AgentOpLogParams = {}): Promise<void> {
  const token = localStorage.getItem('admin_token') || ''
  const qs = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== '') qs.append(k, String(v))
  }
  const suffix = qs.toString() ? `?${qs.toString()}` : ''
  const res = await fetch(`/api/console/agent-oplogs/export${suffix}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) throw new Error(`导出失败(${res.status})`)
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `代理操作日志_${new Date().toISOString().slice(0, 10)}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

/** 级别 → Badge 变体 + 中文（代理日志复用管理端语义）。 */
export const agentOpLevelMeta: Record<string, { text: string; variant: 'destructive' | 'warning' | 'muted' }> = {
  danger: { text: '高危', variant: 'destructive' },
  warning: { text: '重要', variant: 'warning' },
  normal: { text: '常规', variant: 'muted' },
}
/** 结果 → Badge 变体 + 中文。 */
export const agentOpResultMeta: Record<string, { text: string; variant: 'success' | 'destructive' }> = {
  ok: { text: '成功', variant: 'success' },
  fail: { text: '失败', variant: 'destructive' },
}
export const agentOpLevelOptions = [
  { value: '', label: '全部级别' },
  { value: 'danger', label: '高危' },
  { value: 'warning', label: '重要' },
  { value: 'normal', label: '常规' },
]
export const agentOpResultOptions = [
  { value: '', label: '全部结果' },
  { value: 'ok', label: '成功' },
  { value: 'fail', label: '失败' },
]
