/** 站内信下发管理 API（后台，我方新增；epay 无此实体）。 */
import { request, type PageResult } from './client'

export interface AdminMessage {
  id: number
  uid: number // 0=全体广播
  title: string
  content: string
  is_read: boolean
  date: string
}

export function fetchAdminMessages(params: { page?: number; pageSize?: number } = {}): Promise<PageResult<AdminMessage>> {
  return request<PageResult<AdminMessage>>('/admin/messages', { query: { ...params } })
}

export function sendMessage(body: { uid: number; title: string; content: string }): Promise<{ ok: boolean }> {
  return request('/admin/messages', { method: 'POST', body })
}

export function deleteMessage(id: number): Promise<{ id: number }> {
  return request(`/admin/messages/${id}`, { method: 'DELETE' })
}
