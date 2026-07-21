/** 支付通道 API。Channel 类型复用 mock 里已定义的结构（字段一致）。 */
import { request, type PageResult } from './client'
import type { Channel } from '@/lib/mock/channels'

export interface ChannelListParams {
  page?: number
  pageSize?: number
  keyword?: string
  plugin?: string
  type?: number
  status?: number
}

/** 拉取支付通道列表（分页） */
export function fetchChannels(params: ChannelListParams = {}): Promise<PageResult<Channel>> {
  return request<PageResult<Channel>>('/admin/channels', { query: { ...params } })
}

/** 新增/编辑通道的表单入参（对齐后端 dto.ChannelSaveReq） */
export interface ChannelSaveReq {
  name: string
  type: number
  plugin: string
  mode: number
  rate: string
  costrate: string
  daytop: number
  paymin: string
  paymax: string
}

/** 新增通道，返回新建 ID */
export function createChannel(body: ChannelSaveReq): Promise<{ id: number }> {
  return request<{ id: number }>('/admin/channels', { method: 'POST', body })
}

/** 编辑通道 */
export function updateChannel(id: number, body: ChannelSaveReq): Promise<{ id: number }> {
  return request<{ id: number }>(`/admin/channels/${id}`, { method: 'PUT', body })
}

/** 删除通道 */
export function deleteChannel(id: number): Promise<{ id: number }> {
  return request<{ id: number }>(`/admin/channels/${id}`, { method: 'DELETE' })
}

/** 切换通道状态 */
export function setChannelStatus(id: number, status: number): Promise<{ id: number; status: number }> {
  return request<{ id: number; status: number }>(`/admin/channels/${id}/status`, {
    method: 'PUT',
    body: { status },
  })
}

/** 通道密钥配置回填 */
export interface ChannelConfig {
  id: number
  name: string
  plugin: string
  config: string
}

/** 读取通道密钥配置 */
export function fetchChannelConfig(id: number): Promise<ChannelConfig> {
  return request<ChannelConfig>(`/admin/channels/${id}/config`)
}

/** 保存通道密钥配置 */
export function saveChannelConfig(id: number, config: string): Promise<{ id: number }> {
  return request<{ id: number }>(`/admin/channels/${id}/config`, {
    method: 'PUT',
    body: { config },
  })
}
