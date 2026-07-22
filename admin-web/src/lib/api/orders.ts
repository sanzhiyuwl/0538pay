/** 订单相关 API。Order 类型复用 mock 里已定义的结构（字段一致）。 */
import { request, type PageResult } from './client'
import type { Order } from '@/lib/mock/orders'

export interface OrderListParams {
  page?: number
  pageSize?: number
  column?: string
  keyword?: string
  status?: number
}

/** 拉取订单列表（分页） */
export function fetchOrders(params: OrderListParams = {}): Promise<PageResult<Order>> {
  return request<PageResult<Order>>('/admin/orders', { query: { ...params } })
}

/**
 * 服务端全量导出订单为 CSV（不受列表 ≤100 限制）。按当前列表筛选(column/keyword/status)。
 * 直接以带鉴权头 fetch 后触发浏览器下载 Blob。
 */
export async function exportOrders(params: OrderListParams = {}): Promise<void> {
  const token = localStorage.getItem('admin_token') || ''
  const qs = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== '') qs.set(k, String(v))
  }
  const res = await fetch(`/api/admin/orders/export?${qs.toString()}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) throw new Error(`导出失败(${res.status})`)
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `订单导出_${new Date().toISOString().slice(0, 10)}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

/** 裸改订单状态（0改未完成 / 1改已完成） */
export function setOrderStatus(tradeNo: string, status: number): Promise<{ trade_no: string; status: number }> {
  return request(`/admin/orders/${tradeNo}/status`, { method: 'PUT', body: { status } })
}

/** 冻结订单（扣商户分成） */
export function freezeOrder(tradeNo: string): Promise<{ trade_no: string }> {
  return request(`/admin/orders/${tradeNo}/freeze`, { method: 'POST' })
}

/** 解冻订单（加回商户分成） */
export function unfreezeOrder(tradeNo: string): Promise<{ trade_no: string }> {
  return request(`/admin/orders/${tradeNo}/unfreeze`, { method: 'POST' })
}

/** 退款前查询可退金额 */
export interface RefundInfo {
  trade_no: string
  realmoney: number
  refunded: number
  refundable: number
  can_api: boolean
}
export function fetchRefundInfo(tradeNo: string): Promise<RefundInfo> {
  return request(`/admin/orders/${tradeNo}/refund-info`)
}

/** 退款（api=false 手动退款仅扣余额 / api=true 原路退款需管理员密码） */
export function refundOrder(body: {
  trade_no: string
  money: string
  api: boolean
  password?: string
}): Promise<{ trade_no: string }> {
  return request('/admin/orders/refund', { method: 'POST', body })
}

/** 手动补单（未支付→已支付并入账+通知） */
export function fillOrder(tradeNo: string): Promise<{ trade_no: string }> {
  return request(`/admin/orders/${tradeNo}/fill`, { method: 'POST' })
}

/** 重新通知商户 */
export function renotifyOrder(tradeNo: string): Promise<{ trade_no: string }> {
  return request(`/admin/orders/${tradeNo}/notify`, { method: 'POST' })
}

/** 删除订单（物理删除） */
export function deleteOrder(tradeNo: string): Promise<{ trade_no: string }> {
  return request(`/admin/orders/${tradeNo}`, { method: 'DELETE' })
}

/** 批量操作（0改未完成 1改已完成 2冻结 3解冻 4删除） */
export function batchOrders(action: number, tradeNos: string[]): Promise<{ affected: number }> {
  return request('/admin/orders/batch', { method: 'POST', body: { action, trade_nos: tradeNos } })
}
