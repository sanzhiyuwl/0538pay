/** 通道轮询组 API（对齐后端 /admin/rolls 系列）。 */
import { request } from './client'

/** 轮询组内的通道项 */
export interface RollChannelItem {
  channel: number // 通道ID
  channelname: string // 通道名（展示用，保存时后端忽略）
  weight: number // 权重（1-99，仅 kind=1 有效）
}

/** 轮询组对外结构（对齐后端 dto.RollView） */
export interface RollView {
  id: number
  name: string
  type: number
  typeshowname: string
  kind: number // 0=顺序 1=权重随机 2=首个启用
  channels: RollChannelItem[]
  status: number
}

/** 新增/编辑轮询组入参（对齐后端 dto.RollSaveReq） */
export interface RollSaveReq {
  name: string
  type: number
  kind: number
  channels: RollChannelItem[]
}

/** 拉取全部轮询组 */
export function fetchRolls(): Promise<{ list: RollView[] }> {
  return request<{ list: RollView[] }>('/admin/rolls')
}

/** 新增轮询组，返回新建 ID */
export function createRoll(body: RollSaveReq): Promise<{ id: number }> {
  return request<{ id: number }>('/admin/rolls', { method: 'POST', body })
}

/** 编辑轮询组 */
export function updateRoll(id: number, body: RollSaveReq): Promise<{ id: number }> {
  return request<{ id: number }>(`/admin/rolls/${id}`, { method: 'PUT', body })
}

/** 切换轮询组开关 */
export function setRollStatus(id: number, status: number): Promise<{ id: number; status: number }> {
  return request<{ id: number; status: number }>(`/admin/rolls/${id}/status`, {
    method: 'PUT',
    body: { status },
  })
}

/** 删除轮询组 */
export function deleteRoll(id: number): Promise<{ id: number }> {
  return request<{ id: number }>(`/admin/rolls/${id}`, { method: 'DELETE' })
}
