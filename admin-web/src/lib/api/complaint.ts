/**
 * 微信支付消费者投诉2.0（自研扩展，挂服务商进件线）API —— admin/wx-complaints 投诉工单台。
 *
 * 后端只有一套：微信回调只打服务商回调地址 → 平台收下 → 按 complainted_mchid 反查本地商户 → 落库分发。
 * 只走商户进件不走二清：处理类接口带越权拦截（被诉子商户须在本平台名下已进件成功）。
 * 敏感字段 payer_phone 存密文 + 脱敏回显，前端只见脱敏号。
 */
import { request, upload } from './client'

/** 投诉状态：PENDING 待处理 / PROCESSING 处理中 / PROCESSED 已完成 */
export type ComplaintState = 'PENDING' | 'PROCESSING' | 'PROCESSED'

/** 投诉关联订单单项 */
export interface ComplaintOrder {
  transaction_id: string
  out_trade_no: string
  amount: number // 分
  state: string
}

/** 投诉单（对应后端 model.WxComplaint，敏感字段已脱敏） */
export interface WxComplaint {
  id: number
  complaint_id: string
  complainted_mchid: string // 被诉子商户号
  merchant_id: number // 本地商户 uid（反查）
  enroll_id: number
  merchant_name: string
  complaint_state: ComplaintState
  complaint_time: string
  complaint_detail: string
  problem_type: string // REFUND/SERVICE_NOT_WORK/OTHERS
  problem_description: string
  apply_refund_amount: number // 分
  complaint_full_refunded: boolean
  complaint_order_info: string // JSON 字符串
  payer_phone: string // 脱敏后手机号（前3后4）
  payer_openid: string
  user_complaint_times: number
  incoming_user_response: boolean
  in_platform_service: boolean
  need_immediate_service: boolean
  complaint_media_list: string
  user_tag_list: string
  last_action_type: string
  last_event_type: string
  created_at: string
  updated_at: string
}

/** 回调流水一条（动作时间线，新在上） */
export interface WxComplaintNotify {
  id: number
  notify_id: string
  complaint_id: string
  event_type: string
  action_type: string
  summary: string
  created_at: string
}

export interface ComplaintListItem extends WxComplaint {
  state_text: string
}

export interface ComplaintListResult {
  list: ComplaintListItem[]
  total: number
  stats: Record<string, number> // 各状态计数
}

export interface ComplaintDetailResult {
  complaint: WxComplaint
  state_text: string
  notifies: WxComplaintNotify[]
  orders: ComplaintOrder[]
}

/** 协商历史一条 */
export interface NegotiationHistory {
  log_id: string
  operator: string // USER/MERCHANT/PLATFORM
  operate_time: string
  operate_type: string
  operate_details: string
  image_list?: string[]
}

export interface NegotiationHistoryResp {
  data: NegotiationHistory[]
  limit: number
  offset: number
  total_count: number
}

export interface NotifyURLState {
  registered: string // 微信侧已注册回调地址
  local: string // 本地期望值
}

export interface ComplaintListQuery {
  keyword?: string
  complainted_mchid?: string
  state?: string
  page?: number
  pagesize?: number
}

/** 投诉单列表（admin 全量，可按状态/子商户/关键词筛选） */
export function adminListComplaints(q: ComplaintListQuery = {}): Promise<ComplaintListResult> {
  return request<ComplaintListResult>('/admin/wx-complaints', { query: { ...q } })
}

/** 投诉单详情（读本地 + 回调时间线） */
export function adminComplaintDetail(id: number): Promise<ComplaintDetailResult> {
  return request<ComplaintDetailResult>(`/admin/wx-complaints/${id}`)
}

/** 现查微信详情覆盖本地快照 */
export function adminSyncComplaint(id: number): Promise<ComplaintDetailResult> {
  return request<ComplaintDetailResult>(`/admin/wx-complaints/${id}/sync`, { method: 'POST' })
}

/** 协商历史（现查微信） */
export function adminComplaintHistory(id: number, limit = 50, offset = 0): Promise<NegotiationHistoryResp> {
  return request<NegotiationHistoryResp>(`/admin/wx-complaints/${id}/history`, { query: { limit, offset } })
}

/** 回复用户（≤200字符，图片 media_id ≤4） */
export function adminReplyComplaint(
  id: number,
  body: { content: string; images?: string[]; jump_url?: string; jump_url_text?: string },
): Promise<{ ok: boolean }> {
  return request<{ ok: boolean }>(`/admin/wx-complaints/${id}/reply`, { method: 'POST', body })
}

/** 反馈处理完成 */
export function adminCompleteComplaint(id: number): Promise<{ ok: boolean }> {
  return request<{ ok: boolean }>(`/admin/wx-complaints/${id}/complete`, { method: 'POST' })
}

/** 更新退款审批（APPROVE/REJECT） */
export function adminUpdateComplaintRefund(
  id: number,
  body: { action: 'APPROVE' | 'REJECT'; launch_refund_day?: number; reject_reason?: string; reject_media_list?: string[]; remark?: string },
): Promise<{ ok: boolean }> {
  return request<{ ok: boolean }>(`/admin/wx-complaints/${id}/refund`, { method: 'POST', body })
}

/** 回复需即时服务的投诉单 */
export function adminReplyComplaintImmediate(
  id: number,
  body: { content: string; images?: string[] },
): Promise<{ ok: boolean }> {
  return request<{ ok: boolean }>(`/admin/wx-complaints/${id}/immediate`, { method: 'POST', body })
}

/** 上传商户反馈图片，返回 media_id */
export function adminUploadComplaintImage(file: File): Promise<{ media_id: string }> {
  const form = new FormData()
  form.append('file', file)
  return upload<{ media_id: string }>('/admin/wx-complaints/upload', form)
}

/** 查询回调地址（微信侧已注册 + 本地期望） */
export function adminGetComplaintNotifyURL(): Promise<NotifyURLState> {
  return request<NotifyURLState>('/admin/wx-complaints/notify-url')
}

/** 设置回调地址（创建或更新，幂等） */
export function adminSetComplaintNotifyURL(url: string): Promise<NotifyURLState> {
  return request<NotifyURLState>('/admin/wx-complaints/notify-url', { method: 'PUT', body: { url } })
}

/** 删除微信侧回调地址 */
export function adminDeleteComplaintNotifyURL(): Promise<{ ok: boolean }> {
  return request<{ ok: boolean }>('/admin/wx-complaints/notify-url', { method: 'DELETE' })
}

/** 手动触发轮询兜底对账（begin_date/end_date，YYYY-MM-DD） */
export function adminReconcileComplaints(begin_date?: string, end_date?: string): Promise<{ synced: number }> {
  return request<{ synced: number }>('/admin/wx-complaints/reconcile', { method: 'POST', body: { begin_date, end_date } })
}
