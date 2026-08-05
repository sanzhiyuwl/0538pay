/**
 * 子商户管控（风控第二段）API —— admin/risk-controls 风控总览页。
 * 数据源：已开通子商户（channel_enroll status=approved）+ 微信「查询子商户管控情况」(4012803072) 落地快照。
 * 只走商户进件不走二清：被管控（关闭收单等）的商户，通道对他下单硬锁不回退平台号（收单硬锁为下一批）。
 */
import { request, upload } from './client'

/** 本地管控态：normal 正常 / controlled 被管控 / delayed 延迟管控（未到点） */
export type ChannelControlState = 'normal' | 'controlled' | 'delayed'

/** 被管控原因及解脱路径单项（一商户可多条，各自关联不同能力） */
export interface ChannelControlRecovery {
  limitation_case_id: string // 被管控单据号（↔第三段订阅关联主键/幂等键）
  limitation_reason_type: string // 原因类型枚举（RISK_ABNORMAL 等）
  limitation_reason_type_text: string // 原因类型中文
  limitation_reason: string // 被管控原因文本
  limitation_reason_describe: string // 原因描述（给商户看的人话）
  relate_limitations?: string[] // 本条关联的被管控能力枚举
  other_relate_limitations?: string // 其他关联能力（自由文本）
  recover_way: string // 解脱路径枚举
  recover_way_text: string // 解脱路径中文
  recover_way_param?: string // 解脱路径参数
  recover_help_url?: string // 解脱帮助链接
  limitation_action_type?: string // 处置方式（立即/延迟管控）
  limitation_start_date?: string // 预计管控开始时间（延迟管控）
  limitation_date?: string // 实际被管控时间
}

/** 某已开通子商户的管控状态视图（风控总览一行） */
export interface ChannelControlView {
  enroll_id: number
  uid: number
  merchant_name: string
  merchant_phone: string
  channel_id: number
  channel_name: string
  sub_mchid: string
  state: ChannelControlState
  state_text: string
  limited_functions?: string[] // 顶层聚合被管控能力枚举
  limited_function_texts?: string[] // 对应中文
  other_limited_functions?: string // 枚举外能力（自由文本）
  recovery?: ChannelControlRecovery[] // 原因+解脱路径列表
  queried: boolean // 是否已刷新过（有快照）
  last_query_at: string // 最近刷新时间（未刷新为空）
  last_error?: string // 最近刷新失败原因
  // 解脱路径主动代办留痕（服务商直接调 API 代办，非引导商户去小程序自助）
  last_settle_apply_no?: string // 最近一次改结算账户申请单号
  last_settle_apply_at?: string
  last_subject_apply_no?: string // 最近一次改主体资料申请单号
  last_subject_apply_at?: string
}

/** 概览卡数据 */
export interface ChannelControlOverview {
  approved_total: number // 已开通子商户总数
  controlled: number // 被管控家数
  delayed: number // 延迟管控家数
  normal: number // 正常家数（含未刷新）
  never_queried: number // 尚未刷新过的家数
}

export interface ChannelControlListResp {
  overview: ChannelControlOverview
  list: ChannelControlView[]
}

export interface ChannelControlRefreshResp {
  refreshed: number // 成功刷新家数
  failed: number // 失败家数
  views: ChannelControlView[] // 刷新后的最新视图（单个刷新返回一条）
}

/**
 * 管控流水一条（风控第三段：处置/管控订阅回调时间线）。
 * 两套机制落同一表：violation=商户平台处置通知 / merchant_notify=合作伙伴订阅·管控流水。
 */
export interface ChannelControlFlowItem {
  id: number
  mechanism: 'violation' | 'merchant_notify'
  event_type: string
  summary?: string
  sub_mchid: string
  // (A) 商户平台处置通知
  record_id?: string
  company_name?: string
  punish_plan?: string
  punish_time?: string
  punish_description?: string
  risk_type?: string
  risk_description?: string
  // (B) 合作伙伴订阅·管控流水
  business_code?: string // ↔ 第二段 limitation_case_id
  business_state?: string
  business_state_text?: string
  business_time?: string
  created_at: string // 落库时间（时间线排序）
}

export interface ChannelControlFlowResp {
  list: ChannelControlFlowItem[]
}

/** 风控总览：概览 + 已开通子商户管控列表（读本地快照，不现查微信）。 */
export function adminListChannelControls(): Promise<ChannelControlListResp> {
  return request<ChannelControlListResp>('/admin/channel-controls')
}

/** 单个商户刷新管控状态（现查微信落快照）。 */
export function adminRefreshChannelControl(enrollId: number): Promise<ChannelControlRefreshResp> {
  return request<ChannelControlRefreshResp>(`/admin/channel-controls/${enrollId}/refresh`, { method: 'POST' })
}

/** 批量刷新全部已开通商户（后端串行限速，前端等待即可）。 */
export function adminRefreshAllChannelControls(): Promise<ChannelControlRefreshResp> {
  return request<ChannelControlRefreshResp>('/admin/channel-controls/refresh-all', { method: 'POST' })
}

/** 某进件单的管控流水时间线（风控第三段处置/订阅回调）。 */
export function adminListChannelControlFlows(enrollId: number): Promise<ChannelControlFlowResp> {
  return request<ChannelControlFlowResp>(`/admin/channel-controls/${enrollId}/flows`)
}

/** 单个进件单的管控快照（进件详情抽屉「业务受限」就地快照用，两处不重复造轮子；未开通返回 view=null）。 */
export function adminGetChannelControl(enrollId: number): Promise<{ view: ChannelControlView | null }> {
  return request<{ view: ChannelControlView | null }>(`/admin/channel-controls/${enrollId}`)
}

// —— 解脱路径主动代办（0804 方案补遗：recover_way=修改主体资料/修改结算账户时，服务商直接调 API 代办）——

/** 代办修改结算账户入参（明文，敏感字段后端加密）。 */
export interface ModifySettlementReq {
  account_type: string // ACCOUNT_TYPE_BUSINESS 对公 / ACCOUNT_TYPE_PRIVATE 对私
  account_bank: string
  bank_name?: string
  bank_branch_id?: string
  account_number: string // ★银行账号
  account_name?: string // ★开户名称
}

/** 代办修改结算账户。 */
export function adminModifySettlement(enrollId: number, req: ModifySettlementReq): Promise<{ application_no: string }> {
  return request<{ application_no: string }>(`/admin/channel-controls/${enrollId}/modify-settlement`, { method: 'POST', body: req })
}

/** 主体资料变更最终受益人（字段名 card_*，与进件申请单的 ubo_id_doc_* 不同产品各自命名）。 */
export interface SubjectAlterUBO {
  id_doc_type: string
  card_front: string
  card_back?: string
  card_name: string // ★
  card_number: string // ★
  card_address: string // ★
  period_begin: string
  period_end: string
}

/** 代办修改主体资料入参（明文，敏感字段后端加密）。 */
export interface SubjectAlterReq {
  alter_scope?: string // ALTER_SCOPE_FULL / _BUSINESS_CERT / _UBO（空=全部）
  organization_type?: string
  finance_institution?: boolean
  // 营业执照（个体户/企业）
  license_number?: string
  license_copy?: string
  business_merchant_name?: string
  legal_person?: string
  company_address?: string
  license_period_begin?: string
  license_period_end?: string
  // 登记证书（政府机关/事业单位/社会组织）
  cert_type?: string
  cert_number?: string
  cert_copy?: string
  cert_merchant_name?: string
  cert_company_address?: string
  cert_legal_person?: string
  cert_period_begin?: string
  cert_period_end?: string
  // 金融机构许可证
  finance_type?: string
  finance_license_pics?: string[]
  // 法人身份信息
  id_holder_type?: string // LEGAL 经营者/法人 / SUPER 经办人
  id_doc_type?: string
  authorize_letter_copy?: string
  card_front?: string
  card_back?: string
  card_name?: string // ★
  card_number?: string // ★
  card_address?: string // ★
  card_period_begin?: string
  card_period_end?: string
  as_ubo?: boolean
  // 最终受益人列表
  ubo_list?: SubjectAlterUBO[]
  // 补充材料
  bank_openaccount_license?: string[]
  openaccount_approval?: string[]
  legal_other_prove?: string[]
  agency_prove?: string[]
  ubo_prove?: string[]
}

/** 代办提交主体资料变更申请。 */
export function adminModifySubjectInfo(enrollId: number, req: SubjectAlterReq): Promise<{ apply_id: string; out_request_no: string }> {
  return request<{ apply_id: string; out_request_no: string }>(`/admin/channel-controls/${enrollId}/modify-subject`, { method: 'POST', body: req })
}

/** 上传解脱代办表单资料图片，返回微信 media_id 供表单回填。 */
export function adminUploadChannelControlMedia(file: File): Promise<{ media_id: string }> {
  const form = new FormData()
  form.append('file', file)
  return upload<{ media_id: string }>('/admin/channel-controls/upload', form)
}
