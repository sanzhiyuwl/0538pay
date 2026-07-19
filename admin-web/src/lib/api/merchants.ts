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
