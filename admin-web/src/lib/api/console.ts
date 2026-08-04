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
  wx_state: string
  sign_url: string
  audit_detail: string
  settle_application_no: string
  settle_meta: string
  status: string
  reject_reason: string
  source: number
  invite_code: string
}
// 驳回详情单项（audit_detail JSON 数组元素）。
export interface EnrollAuditDetail {
  field: string
  field_name: string
  reject_reason: string
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

// —— 经营场景子对象（sales_info）——
export interface EnrollBizStore {
  biz_store_name: string
  biz_address_code: string
  biz_store_address: string
  store_entrance_pic: string[]
  indoor_pic: string[]
  biz_sub_appid: string
}
export interface EnrollMpInfo {
  mp_appid: string
  mp_sub_appid: string
  mp_pics: string[]
}
export interface EnrollMiniProgram {
  mini_program_appid: string
  mini_program_sub_appid: string
  mini_program_pics: string[]
}
export interface EnrollWebInfo {
  domain: string
  web_authorisation: string
  web_appid: string
}
export interface EnrollAppInfo {
  app_appid: string
  app_sub_appid: string
  app_pics: string[]
}
export interface EnrollWeworkInfo {
  sub_corp_id: string
  wework_pics: string[]
}
// UBO 最终受益人（提交入参，敏感字段明文提交后端加密）
export interface EnrollUBO {
  ubo_id_doc_type: string
  ubo_id_doc_copy: string
  ubo_id_doc_copy_back: string
  ubo_id_doc_name: string
  ubo_id_doc_number: string
  ubo_id_doc_address: string
  ubo_period_begin: string
  ubo_period_end: string
}
// UBO 回显（敏感脱敏，仅 has_*）
export interface EnrollUBOView {
  ubo_id_doc_type: string
  ubo_id_doc_copy: string
  ubo_id_doc_copy_back: string
  ubo_period_begin: string
  ubo_period_end: string
  has_name: boolean
  has_number: boolean
  has_address: boolean
}

// —— 填全套资料（applyment4sub 核心字段；敏感字段后端 RSA-OAEP 加密，不明文落库）——
// 提交入参：敏感字段前端明文提交，后端加密。图片类填 media_id 占位（一期不做上传）。
export interface EnrollMaterialReq {
  subject_type: string
  merchant_shortname: string
  service_phone: string
  license_number: string
  license_copy: string
  business_merchant_name: string
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
  bank_branch_id: string
  bank_name: string
  account_number: string
  contact_type: string
  contact_name: string
  contact_id_number: string
  mobile_phone: string
  contact_email: string
  contact_id_doc_type: string
  contact_id_doc_copy: string
  contact_id_doc_copy_back: string
  contact_period_begin: string
  contact_period_end: string
  // 结算规则 settlement_info（非敏感；settlement_id/qualification_type 微信必填）
  settlement_id: string
  qualification_type: string
  qualifications: string[]
  activities_id: string
  debit_activities_rate: string
  credit_activities_rate: string
  activities_additions: string[]
  // 登记证书（政府/事业/社会组织）
  cert_type: string
  cert_copy: string
  cert_number: string
  cert_merchant_name: string
  cert_company_address: string
  cert_legal_person: string
  cert_period_begin: string
  cert_period_end: string
  cert_letter_copy: string
  // 身份证件：证件类型 + 企业身份证居住地址 + 非身份证证件（护照/通行证等）
  id_doc_type: string
  id_card_address: string
  id_doc_copy: string
  id_doc_copy_back: string
  id_doc_name: string
  id_doc_number: string
  id_doc_address: string
  doc_period_begin: string
  doc_period_end: string
  // 经营场景 sales_info
  sales_scenes_type: string[]
  biz_store: EnrollBizStore
  mp_info: EnrollMpInfo
  mini_program: EnrollMiniProgram
  web_info: EnrollWebInfo
  app_info: EnrollAppInfo
  wework_info: EnrollWeworkInfo
  // 最终受益人
  ubo_list: EnrollUBO[]
  // 补充材料
  legal_person_commitment: string
  legal_person_video: string
  business_addition_pics: string[]
  business_addition_msg: string
}
// 回显：敏感字段一律不回原文，只回 has_* 是否已填。
export interface EnrollMaterialView {
  filled: boolean
  subject_type: string
  merchant_shortname: string
  service_phone: string
  license_number: string
  license_copy: string
  business_merchant_name: string
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
  bank_branch_id: string
  bank_name: string
  has_id_card_name: boolean
  has_id_card_number: boolean
  has_account_name: boolean
  has_account_number: boolean
  has_contact_name: boolean
  contact_name_masked: string // 超管姓名脱敏（张*/徐*坤），提示谁来扫码签约
  has_contact_id_number: boolean
  has_mobile_phone: boolean
  has_contact_email: boolean
  contact_type: string
  contact_id_doc_type: string
  contact_id_doc_copy: string
  contact_id_doc_copy_back: string
  contact_period_begin: string
  contact_period_end: string
  // 结算规则（非敏感，原样回显）
  settlement_id: string
  qualification_type: string
  qualifications: string[]
  activities_id: string
  debit_activities_rate: string
  credit_activities_rate: string
  activities_additions: string[]
  // 登记证书（非敏感回显）
  cert_type: string
  cert_copy: string
  cert_number: string
  cert_merchant_name: string
  cert_company_address: string
  cert_legal_person: string
  cert_period_begin: string
  cert_period_end: string
  cert_letter_copy: string
  // 身份证件（非敏感回显 + 敏感 has_*）
  id_doc_type: string
  id_doc_copy: string
  id_doc_copy_back: string
  doc_period_begin: string
  doc_period_end: string
  has_id_card_address: boolean
  has_id_doc_name: boolean
  has_id_doc_number: boolean
  has_id_doc_address: boolean
  // 经营场景（非敏感回显）
  sales_scenes_type: string[]
  biz_store: EnrollBizStore
  mp_info: EnrollMpInfo
  mini_program: EnrollMiniProgram
  web_info: EnrollWebInfo
  app_info: EnrollAppInfo
  wework_info: EnrollWeworkInfo
  // UBO 脱敏回显
  ubo_list: EnrollUBOView[]
  // 补充材料（非敏感回显）
  legal_person_commitment: string
  legal_person_video: string
  business_addition_pics: string[]
  business_addition_msg: string
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
/** 上传一段进件资料视频（部分指定行业进件时微信要求补充），返回微信 media_id。 */
export function uploadEnrollVideo(id: number, file: File): Promise<{ media_id: string }> {
  const form = new FormData()
  form.append('file', file)
  return upload<{ media_id: string }>(`/console/enrolls/${id}/video`, form)
}

// —— 结算账户管理（进件成功后售后，接口 6/7/8）——
// 修改结算账户入参：敏感字段（银行账号/开户名）前端明文提交，后端加密。
export interface SettleModifyReq {
  account_type: string   // ACCOUNT_TYPE_BUSINESS 对公 / ACCOUNT_TYPE_PRIVATE 对私
  account_bank: string   // 开户银行
  bank_name?: string     // 开户银行全称（含支行），按需
  bank_branch_id?: string // 联行号，按需
  account_number: string // ★银行账号
  account_name?: string  // ★开户名称，按需
}
// 查询结算账户结果（掩码账号+验证结果）。
export interface SettlementView {
  account_type: string
  account_bank: string
  bank_name: string
  bank_branch_id: string
  account_number: string     // 掩码
  verify_result: string      // VERIFY_SUCCESS / VERIFY_FAIL / VERIFYING
  verify_fail_reason: string
}
// 查询改单审核状态结果。
export interface SettleApplicationView {
  account_name: string
  account_type: string
  account_bank: string
  bank_name: string
  bank_branch_id: string
  account_number: string
  verify_result: string      // AUDIT_SUCCESS / AUDITING / AUDIT_FAIL
  verify_fail_reason: string
  verify_finish_time: string
}
/** 查当前生效的结算账户（掩码+验证结果，落库留痕）。 */
export function getEnrollSettlement(id: number): Promise<SettlementView> {
  return request<SettlementView>(`/console/enrolls/${id}/settlement`)
}
/** 修改结算银行账户，返回修改申请单号 application_no。 */
export function modifyEnrollSettlement(id: number, body: SettleModifyReq): Promise<{ application_no: string }> {
  return request<{ application_no: string }>(`/console/enrolls/${id}/settlement`, { method: 'POST', body })
}
/** 查改单审核状态。 */
export function getEnrollSettleApplication(id: number): Promise<SettleApplicationView> {
  return request<SettleApplicationView>(`/console/enrolls/${id}/settlement/application`)
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
/** 通用：按筛选拉操作日志 CSV（走 admin token，UTF-8 BOM）并触发浏览器下载。三类审计日志共用。 */
async function downloadOpLogCsv(path: string, params: AgentOpLogParams, filePrefix: string): Promise<void> {
  const token = localStorage.getItem('admin_token') || ''
  const qs = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== '') qs.append(k, String(v))
  }
  const suffix = qs.toString() ? `?${qs.toString()}` : ''
  const res = await fetch(`${path}${suffix}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) throw new Error(`导出失败(${res.status})`)
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${filePrefix}_${new Date().toISOString().slice(0, 10)}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

/** 导出代理操作日志 CSV（UTF-8 BOM，浏览器直接下载）。走 admin token。 */
export async function exportAgentOpLogs(params: AgentOpLogParams = {}): Promise<void> {
  await downloadOpLogCsv('/api/console/agent-oplogs/export', params, '代理操作日志')
}

// —— 控制台管理日志（scope=console；平台运营在 /console 的写操作：管代理/改权限/发名额/建单/退款/改结算）——
// 日志结构与代理日志同构，复用 AgentOpLog 类型 + level/result meta。
export function fetchConsoleOpLogs(params: AgentOpLogParams = {}): Promise<PageResult<AgentOpLog>> {
  return request<PageResult<AgentOpLog>>('/console/oplogs', { query: { ...params } })
}
export function fetchConsoleOpActionOptions(): Promise<{ actions: AgentOpActionOption[] }> {
  return request<{ actions: AgentOpActionOption[] }>('/console/oplogs/options')
}
export async function exportConsoleOpLogs(params: AgentOpLogParams = {}): Promise<void> {
  await downloadOpLogCsv('/api/console/oplogs/export', params, '控制台管理日志')
}

// —— 系统运维日志（scope=system；系统事件：提交微信/开通/驳回/名额三态/超时关单等，非人工触发）——
export function fetchSystemOpLogs(params: AgentOpLogParams = {}): Promise<PageResult<AgentOpLog>> {
  return request<PageResult<AgentOpLog>>('/console/system-oplogs', { query: { ...params } })
}
export function fetchSystemOpActionOptions(): Promise<{ actions: AgentOpActionOption[] }> {
  return request<{ actions: AgentOpActionOption[] }>('/console/system-oplogs/options')
}
export async function exportSystemOpLogs(params: AgentOpLogParams = {}): Promise<void> {
  await downloadOpLogCsv('/api/console/system-oplogs/export', params, '系统运维日志')
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
