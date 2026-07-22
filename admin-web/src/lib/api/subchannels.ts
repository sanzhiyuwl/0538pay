/** 子通道 API（商户维度，对齐后端 /admin/subchannels 系列）。 */
import { request } from './client'

/** 子通道对外结构（对齐后端 dto.SubChannelView） */
export interface SubChannelView {
  id: number
  channel: number // 归属主通道ID
  channelname: string // 主通道名（展示用）
  uid: number
  name: string
  status: number
  info: string // 自定义参数 JSON 原文
  usetime: string // 上次使用时间（"—"=从未使用）
}

/** 新增/编辑子通道入参（对齐后端 dto.SubChannelSaveReq） */
export interface SubChannelSaveReq {
  channel: number
  name: string
  info: string
}

/** 拉取某商户的子通道列表 */
export function fetchSubChannels(uid: number): Promise<{ list: SubChannelView[] }> {
  return request<{ list: SubChannelView[] }>('/admin/subchannels', { query: { uid } })
}

/** 新增子通道，返回新建 ID */
export function createSubChannel(uid: number, body: SubChannelSaveReq): Promise<{ id: number }> {
  return request<{ id: number }>('/admin/subchannels', {
    method: 'POST',
    query: { uid },
    body,
  })
}

/** 编辑子通道 */
export function updateSubChannel(id: number, body: SubChannelSaveReq): Promise<{ id: number }> {
  return request<{ id: number }>(`/admin/subchannels/${id}`, { method: 'PUT', body })
}

/** 切换子通道开关 */
export function setSubChannelStatus(id: number, status: number): Promise<{ id: number; status: number }> {
  return request<{ id: number; status: number }>(`/admin/subchannels/${id}/status`, {
    method: 'PUT',
    body: { status },
  })
}

/** 删除子通道 */
export function deleteSubChannel(id: number): Promise<{ id: number }> {
  return request<{ id: number }>(`/admin/subchannels/${id}`, { method: 'DELETE' })
}
