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
