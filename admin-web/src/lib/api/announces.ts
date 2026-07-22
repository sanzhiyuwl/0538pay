/** 网站公告 API（对齐 epay gonggao.php + pre_anounce）。 */
import { request } from './client'

export interface Announce {
  id: number
  content: string
  color: string
  sort: number
  status: number // 1显示 0隐藏
  addtime: string
}

export interface AnnounceSaveReq {
  content: string
  color?: string
  sort?: number
  status?: number
}

export function fetchAnnounces(): Promise<{ list: Announce[] }> {
  return request('/admin/announces')
}
export function createAnnounce(body: AnnounceSaveReq): Promise<{ ok: boolean }> {
  return request('/admin/announces', { method: 'POST', body })
}
export function updateAnnounce(id: number, body: AnnounceSaveReq): Promise<{ id: number }> {
  return request(`/admin/announces/${id}`, { method: 'PUT', body })
}
export function setAnnounceStatus(id: number, status: number): Promise<{ id: number }> {
  return request(`/admin/announces/${id}/status`, { method: 'PUT', body: { status } })
}
export function deleteAnnounce(id: number): Promise<{ id: number }> {
  return request(`/admin/announces/${id}`, { method: 'DELETE' })
}

/** 公开读取展示中的公告（官网/商户端） */
export function fetchPublicAnnounces(): Promise<{ list: Announce[] }> {
  return request('/site/announces')
}
