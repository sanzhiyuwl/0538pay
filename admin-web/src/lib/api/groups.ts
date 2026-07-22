/** 用户组管理 API（对齐后端 /admin/groups 系列）。 */
import { request } from './client'

/** 通道费率项（解析自 info） */
export interface GroupRateItem {
  label: string
  rate: string
}

/** 用户组对外结构（对齐后端 dto.GroupView） */
export interface GroupView {
  gid: number
  name: string
  isbuy: number
  price: string
  expire: number
  sort: number
  visible: string
  rates: GroupRateItem[]
  info: string
  config: string
  settings: string
  merchantCount: number
}

/** 新增/编辑用户组入参（对齐后端 dto.GroupSaveReq） */
export interface GroupSaveReq {
  name: string
  isbuy: number
  price: string
  expire: number
  sort: number
  visible: string
  info: string
  config: string
  settings: string
}

/** 拉取全部用户组 */
export function fetchGroups(): Promise<{ list: GroupView[] }> {
  return request<{ list: GroupView[] }>('/admin/groups')
}

/** 新增用户组，返回新建 gid */
export function createGroup(body: GroupSaveReq): Promise<{ gid: number }> {
  return request<{ gid: number }>('/admin/groups', { method: 'POST', body })
}

/** 编辑用户组 */
export function updateGroup(gid: number, body: GroupSaveReq): Promise<{ gid: number }> {
  return request<{ gid: number }>(`/admin/groups/${gid}`, { method: 'PUT', body })
}

/** 上架(1)/下架(0) */
export function setGroupBuy(gid: number, isbuy: number): Promise<{ gid: number; isbuy: number }> {
  return request<{ gid: number; isbuy: number }>(`/admin/groups/${gid}/buy`, {
    method: 'PUT',
    body: { isbuy },
  })
}

/** 删除用户组 */
export function deleteGroup(gid: number): Promise<{ gid: number }> {
  return request<{ gid: number }>(`/admin/groups/${gid}`, { method: 'DELETE' })
}

/** 用户组通道分配项（对齐后端 dto.GroupAssignItem） */
export interface GroupAssignItem {
  type: number // 支付方式ID
  kind: string // "channel"|"roll"（正整数目标是通道还是轮询组）
  channel: string // "0"关/"-1"随机/"-2"子通道/正整数
  rate: string // 组级费率覆盖（百分数字符串，空=通道默认）
}

/** 读取该组的通道分配 */
export function fetchGroupAssigns(gid: number): Promise<{ list: GroupAssignItem[] }> {
  return request<{ list: GroupAssignItem[] }>(`/admin/groups/${gid}/assigns`)
}

/** 保存该组的通道分配（整组一次性覆盖） */
export function saveGroupAssigns(gid: number, assigns: GroupAssignItem[]): Promise<{ gid: number }> {
  return request<{ gid: number }>(`/admin/groups/${gid}/assigns`, {
    method: 'PUT',
    body: { assigns },
  })
}
