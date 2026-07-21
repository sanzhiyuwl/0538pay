/** 分账 API（后台）。对齐 epay profitsharing + 后端 dto.PsOrderView。 */
import { request, type PageResult } from './client'

/** 分账订单（对齐后端 dto.PsOrderView / mock PsOrder） */
export interface PsOrder {
  id: number
  trade_no: string
  api_trade_no: string
  rid: number
  rulename: string
  channelid: number
  channelname: string
  receiver: string
  money: string
  addtime: string
  status: number // 0待分账 1已提交 2成功 3失败 4取消
  result: string
}

/** 分账统计（对齐后端 dto.PsStats） */
export interface PsStats {
  totalMoney: number
  successMoney: number
  failMoney: number
  totalCount: number
  successCount: number
  failCount: number
  successRate: number
}

export interface PsListParams {
  page?: number
  pageSize?: number
  rid?: number
  status?: number
  column?: string
  value?: string
  starttime?: string
  endtime?: string
}

export function fetchPsOrders(params: PsListParams = {}): Promise<PageResult<PsOrder>> {
  return request<PageResult<PsOrder>>('/admin/ps/orders', { query: { ...params } })
}

export function fetchPsStats(params: PsListParams = {}): Promise<PsStats> {
  return request<PsStats>('/admin/ps/orders/stats', { query: { ...params } })
}

/** 分账状态操作：submit(提交)/query(查询)/return(回退)/cancel(取消)/editmoney(改额)/delete(删除) */
export function operatePsOrder(
  id: number,
  action: string,
  money = '',
): Promise<{ id: number; action: string }> {
  return request(`/admin/ps/orders/${id}/op`, { method: 'POST', body: { action, money } })
}
