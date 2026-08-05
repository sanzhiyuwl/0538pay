/**
 * 微信支付消费者投诉2.0 —— 商户端（/m）API。商户自助看/处理【自己名下】的投诉。
 *
 * 与 admin 共用后端同一 WxComplaintService，但走 /merchant/* 路由（自动用 merchant_token）。
 * 数据隔离由后端强制：列表按登录商户 uid 过滤，每个动作前置 merchant_id 归属校验。
 * 不含回调地址自管理 / 兜底对账（那是平台服务商级运维，仅 admin）。类型复用 admin 侧定义。
 */
import { request, upload } from './client'
import type {
  ComplaintListResult,
  ComplaintDetailResult,
  NegotiationHistoryResp,
  ComplaintListQuery,
} from './complaint'

export type {
  ComplaintState,
  ComplaintOrder,
  WxComplaint,
  WxComplaintNotify,
  ComplaintListItem,
  ComplaintListResult,
  ComplaintDetailResult,
  NegotiationHistory,
  NegotiationHistoryResp,
  ComplaintListQuery,
} from './complaint'

/** 我的投诉单列表（后端强制按登录商户隔离，不能查别家） */
export function merchantListComplaints(q: ComplaintListQuery = {}): Promise<ComplaintListResult> {
  const { keyword, state, page, pagesize } = q
  return request<ComplaintListResult>('/merchant/complaints', { query: { keyword, state, page, pagesize } })
}

/** 我的投诉单详情（读本地 + 回调时间线） */
export function merchantComplaintDetail(id: number): Promise<ComplaintDetailResult> {
  return request<ComplaintDetailResult>(`/merchant/complaints/${id}`)
}

/** 现查微信详情覆盖本地快照 */
export function merchantSyncComplaint(id: number): Promise<ComplaintDetailResult> {
  return request<ComplaintDetailResult>(`/merchant/complaints/${id}/sync`, { method: 'POST' })
}

/** 协商历史（现查微信） */
export function merchantComplaintHistory(id: number, limit = 50, offset = 0): Promise<NegotiationHistoryResp> {
  return request<NegotiationHistoryResp>(`/merchant/complaints/${id}/history`, { query: { limit, offset } })
}

/** 回复用户（≤200字符，图片 media_id ≤4） */
export function merchantReplyComplaint(
  id: number,
  body: { content: string; images?: string[]; jump_url?: string; jump_url_text?: string },
): Promise<{ ok: boolean }> {
  return request<{ ok: boolean }>(`/merchant/complaints/${id}/reply`, { method: 'POST', body })
}

/** 反馈处理完成 */
export function merchantCompleteComplaint(id: number): Promise<{ ok: boolean }> {
  return request<{ ok: boolean }>(`/merchant/complaints/${id}/complete`, { method: 'POST' })
}

/** 更新退款审批（APPROVE/REJECT） */
export function merchantUpdateComplaintRefund(
  id: number,
  body: { action: 'APPROVE' | 'REJECT'; launch_refund_day?: number; reject_reason?: string; reject_media_list?: string[]; remark?: string },
): Promise<{ ok: boolean }> {
  return request<{ ok: boolean }>(`/merchant/complaints/${id}/refund`, { method: 'POST', body })
}

/** 回复需即时服务的投诉单 */
export function merchantReplyComplaintImmediate(
  id: number,
  body: { content: string; images?: string[] },
): Promise<{ ok: boolean }> {
  return request<{ ok: boolean }>(`/merchant/complaints/${id}/immediate`, { method: 'POST', body })
}

/** 上传反馈图片，返回 media_id */
export function merchantUploadComplaintImage(file: File): Promise<{ media_id: string }> {
  const form = new FormData()
  form.append('file', file)
  return upload<{ media_id: string }>('/merchant/complaints/upload', form)
}
