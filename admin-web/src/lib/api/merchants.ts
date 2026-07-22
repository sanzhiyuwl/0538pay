/** 商户相关 API。Merchant 类型复用 mock 里已定义的结构（字段一致）。 */
import { request, type PageResult } from './client'
import type { Merchant } from '@/lib/mock/merchants'

export interface MerchantListParams {
  page?: number
  pageSize?: number
  column?: string
  keyword?: string
  gid?: number
  status?: number
  pay?: number
  settle?: number
}

/** 拉取商户列表（分页） */
export function fetchMerchants(params: MerchantListParams = {}): Promise<PageResult<Merchant>> {
  return request<PageResult<Merchant>>('/admin/merchants', { query: { ...params } })
}

/** 添加商户入参（对齐后端 dto.MerchantCreateReq） */
export interface MerchantCreateReq {
  gid: number
  settle_id: number
  account: string
  username: string
  url: string
  email: string
  qq: string
  phone: string
  mode: number
  pay: number
  settle: number
  status: number
  password?: string
}

/** 编辑商户入参（对齐后端 dto.MerchantEditReq） */
export interface MerchantEditReq {
  gid: number
  upid: number
  settle_id: number
  account: string
  username: string
  money: string
  url: string
  email: string
  qq: string
  phone: string
  mode: number
  pay: number
  settle: number
  status: number
  password?: string
}

/** 添加商户，返回新建 uid 与通信密钥 */
export function createMerchant(body: MerchantCreateReq): Promise<{ uid: number; key: string }> {
  return request<{ uid: number; key: string }>('/admin/merchants', { method: 'POST', body })
}

/** 编辑商户 */
export function updateMerchant(uid: number, body: MerchantEditReq): Promise<{ uid: number }> {
  return request<{ uid: number }>(`/admin/merchants/${uid}`, { method: 'PUT', body })
}

/** 余额充值(action=0)/扣除(action=1) */
export function rechargeMerchant(uid: number, action: number, amount: string): Promise<{ uid: number }> {
  return request<{ uid: number }>(`/admin/merchants/${uid}/recharge`, {
    method: 'POST',
    body: { action, amount },
  })
}

/** 修改用户组 + 有效期（endtime 空=永久） */
export function setMerchantGroup(uid: number, gid: number, endtime: string): Promise<{ uid: number; gid: number }> {
  return request<{ uid: number; gid: number }>(`/admin/merchants/${uid}/group`, {
    method: 'PUT',
    body: { gid, endtime },
  })
}

/** 权限/状态切换：field=user(状态)/pay(支付)/settle(结算) */
export function setMerchantStatus(uid: number, field: string, status: number): Promise<{ uid: number }> {
  return request<{ uid: number }>(`/admin/merchants/${uid}/status`, {
    method: 'PUT',
    body: { field, status },
  })
}

/** 重置通信密钥，返回新密钥 */
export function resetMerchantKey(uid: number): Promise<{ uid: number; key: string }> {
  return request<{ uid: number; key: string }>(`/admin/merchants/${uid}/resetkey`, { method: 'POST' })
}

/** 删除商户 */
export function deleteMerchant(uid: number): Promise<{ uid: number }> {
  return request<{ uid: number }>(`/admin/merchants/${uid}`, { method: 'DELETE' })
}
