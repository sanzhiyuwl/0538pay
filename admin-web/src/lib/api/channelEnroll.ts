/**
 * 服务商通道商户进件 API（epay 精仿线，pay_channel_enroll）。
 * 只走商户进件不走二清：商户在服务商通道下进件拿自己的子商户号，钱直清到商户号。
 * 商户端 /m 自助（my* 系列，按登录 uid 隔离）；后台 /admin 审核（看全部）。
 */
import { request, upload } from './client'
import type { EnrollMaterialReq, EnrollMaterialView } from './console'

/** 进件单状态机（全自动化：submitted=审核中；approved=已开通；pending 为历史存量兼容态） */
export type ChannelEnrollStatus = 'draft' | 'pending' | 'submitted' | 'approved' | 'rejected'

/** 微信 applyment4sub 驳回逐字段详情（REJECTED 时按字段返回，供精准补料后用相同 business_code 重提） */
export interface ChannelEnrollAuditDetailItem {
  field: string // 字段名（微信 applyment 字段路径）
  field_name: string // 字段中文名称
  reject_reason: string // 该字段的驳回原因
}

/** 进件单视图（不含敏感原文） */
export interface ChannelEnrollView {
  id: number
  enroll_no: string
  uid: number
  merchant_name: string
  subject_type: string
  contact_phone: string
  merchant_phone: string // 商户账户注册手机（后台列表展示归属商户手机）
  channel_id: number
  channel_name: string
  plugin: string
  status: ChannelEnrollStatus
  status_text: string
  sub_mchid: string
  subchannel_id: number
  reject_reason: string
  audit_detail?: ChannelEnrollAuditDetailItem[] // 微信驳回逐字段详情（非驳回单为空）
  audit_admin: string
  // 微信 applyment4sub 直提交状态
  business_code: string
  wx_applyment_id: string
  wx_state: string
  wx_state_text: string
  sign_url: string
  add_time: string
  submit_time: string
  audit_time: string
  update_time: string
  subchannel_status: number // 已开通单的子通道启停：1启用/0停用；-1=未开通不适用
}

/**
 * 填料回显（敏感只回 has_*）。★复用代理线 EnrollMaterialView（完整微信五大块）+ 商户线专属字段。
 * 与共享组件 EnrollMaterialDrawer 的 fetchFn 返回类型一致，可直接注入。
 */
export interface ChannelEnrollMaterialView extends EnrollMaterialView {
  merchant_name: string
  contact_phone: string
  remark: string
}

/** 进件单详情 */
export interface ChannelEnrollDetail extends ChannelEnrollView {
  material: ChannelEnrollMaterialView
}

/**
 * 填料入参。★复用代理线 EnrollMaterialReq（完整微信 applyment4sub 五大块）+ 商户线专属字段。
 * 与共享组件 EnrollMaterialDrawer 的 submitFn 入参类型一致，可直接注入。
 */
export interface ChannelEnrollMaterialReq extends EnrollMaterialReq {
  contact_phone?: string
  remark?: string
}

export interface ChannelEnrollListResult {
  list: ChannelEnrollView[]
  total: number
}

/** 可进件的服务商通道选项 */
export interface EnrollableChannel {
  id: number
  name: string
  plugin: string
}

// ===== 商户端 /m（自助，按登录 uid 隔离）=====

/** 可进件的服务商通道列表（选通道下拉） */
export function myEnrollableChannels(): Promise<{ list: EnrollableChannel[] }> {
  return request<{ list: EnrollableChannel[] }>('/merchant/channel-enrolls/channels')
}

export function myListChannelEnrolls(params: {
  page?: number
  pagesize?: number
  status?: string
  channel_id?: number
  keyword?: string
} = {}): Promise<ChannelEnrollListResult> {
  return request<ChannelEnrollListResult>('/merchant/channel-enrolls', { query: { ...params } })
}

export function myGetChannelEnroll(id: number): Promise<ChannelEnrollDetail> {
  return request<ChannelEnrollDetail>(`/merchant/channel-enrolls/${id}`)
}

export function myCreateChannelEnroll(body: {
  channel_id: number
  merchant_name?: string
  contact_phone?: string
}): Promise<{ id: number; enroll_no: string; status: ChannelEnrollStatus }> {
  return request('/merchant/channel-enrolls', { method: 'POST', body })
}

export function myFillChannelEnrollMaterial(
  id: number,
  body: ChannelEnrollMaterialReq,
): Promise<{ id: number; status: ChannelEnrollStatus }> {
  return request(`/merchant/channel-enrolls/${id}/material`, { method: 'POST', body })
}

/** 提交微信进件（全自动 applyment4sub）。 */
export function mySubmitChannelEnroll(
  id: number,
): Promise<{ id: number; status: ChannelEnrollStatus; wx_state: string; wx_applyment_id: string }> {
  return request(`/merchant/channel-enrolls/${id}/submit`, { method: 'POST' })
}

/** 主动拉取微信进件状态（审核中→开通/驳回）。 */
export function mySyncChannelEnroll(id: number): Promise<{
  id: number
  status: ChannelEnrollStatus
  wx_state: string
  wx_state_text: string
  sub_mchid: string
  subchannel_id: number
  sign_url: string
  reject_reason: string
}> {
  return request(`/merchant/channel-enrolls/${id}/sync`, { method: 'POST' })
}

/** 支付开关：启用/停用自己已开通渠道（开关对应子通道）。 */
export function myToggleChannelEnroll(id: number, enable: boolean): Promise<{ id: number; subchannel_status: number }> {
  return request(`/merchant/channel-enrolls/${id}/toggle`, { method: 'POST', body: { enable } })
}

/** 删除进件单（提交前放弃，仅草稿/被驳回单可删）。 */
export function myDeleteChannelEnroll(id: number): Promise<{ deleted: boolean }> {
  return request(`/merchant/channel-enrolls/${id}`, { method: 'DELETE' })
}

/** 上传一张进件资料图片（营业执照/身份证/门头照等），返回微信 media_id。 */
export function myUploadChannelEnrollMedia(id: number, file: File): Promise<{ media_id: string }> {
  const form = new FormData()
  form.append('file', file)
  return upload<{ media_id: string }>(`/merchant/channel-enrolls/${id}/media`, form)
}

/** 上传一段进件资料视频，返回微信 media_id。 */
export function myUploadChannelEnrollVideo(id: number, file: File): Promise<{ media_id: string }> {
  const form = new FormData()
  form.append('file', file)
  return upload<{ media_id: string }>(`/merchant/channel-enrolls/${id}/video`, form)
}

/**
 * 填料抽屉适配器：拉回显（EnrollMaterialDrawer.fetchFn 直接注入用）。
 * 返回展平的 material（EnrollMaterialView 形状），与共享组件契约一致。
 */
export async function myGetChannelEnrollMaterial(id: number): Promise<ChannelEnrollMaterialView> {
  const d = await myGetChannelEnroll(id)
  return d.material
}

// ===== 后台 /admin（审核，看全部）=====

export function adminListChannelEnrolls(params: {
  page?: number
  pagesize?: number
  status?: string
  channel_id?: number
  uid?: number
  keyword?: string
  wx_state?: string // 微信侧细状态（APPLYMENT_STATE_TO_BE_SIGNED / SIGNING / ...）
  sort?: string // id_asc / id_desc（默认 id_desc）
} = {}): Promise<ChannelEnrollListResult> {
  return request<ChannelEnrollListResult>('/admin/channel-enrolls', { query: { ...params } })
}

export function adminGetChannelEnroll(id: number): Promise<ChannelEnrollDetail> {
  return request<ChannelEnrollDetail>(`/admin/channel-enrolls/${id}`)
}

export function adminApproveChannelEnroll(
  id: number,
  sub_mchid: string,
): Promise<{ id: number; status: ChannelEnrollStatus; sub_mchid: string; subchannel_id: number }> {
  return request(`/admin/channel-enrolls/${id}/approve`, { method: 'POST', body: { sub_mchid } })
}

export function adminRejectChannelEnroll(
  id: number,
  reason: string,
): Promise<{ id: number; status: ChannelEnrollStatus }> {
  return request(`/admin/channel-enrolls/${id}/reject`, { method: 'POST', body: { reason } })
}

/** 后台主动拉取微信进件状态（同商户端逻辑，看全部）。 */
export function adminSyncChannelEnroll(id: number): Promise<{
  id: number
  status: ChannelEnrollStatus
  wx_state: string
  wx_state_text: string
  sub_mchid: string
  subchannel_id: number
  sign_url: string
  reject_reason: string
}> {
  return request(`/admin/channel-enrolls/${id}/sync`, { method: 'POST' })
}
