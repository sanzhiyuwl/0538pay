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

// ===== 分账规则管理（ps_receiver，C-1）=====

/** 分账规则（对齐后端 dto.PsReceiverView） */
export interface PsReceiver {
  id: number
  channel: number
  channel_name: string
  subchannel: number
  uid: number // 0=通道级全局
  account: string
  name: string
  rate: string
  minmoney: string
  status: number // 0关 1开
  addtime: string
}

/** 新增/编辑分账规则入参（对齐后端 dto.PsReceiverReq） */
export interface PsReceiverReq {
  channel: number
  subchannel: number
  uid: number
  account: string
  name: string
  rate: string
  minmoney: string
}

export function fetchPsReceivers(): Promise<{ list: PsReceiver[] }> {
  return request<{ list: PsReceiver[] }>('/admin/ps/receivers')
}

export function createPsReceiver(body: PsReceiverReq): Promise<{ msg: string }> {
  return request('/admin/ps/receivers', { method: 'POST', body })
}

export function updatePsReceiver(id: number, body: PsReceiverReq): Promise<{ id: number }> {
  return request(`/admin/ps/receivers/${id}`, { method: 'PUT', body })
}

export function setPsReceiverStatus(id: number, status: number): Promise<{ id: number; status: number }> {
  return request(`/admin/ps/receivers/${id}/status`, { method: 'PUT', body: { status } })
}

export function deletePsReceiver(id: number): Promise<{ id: number }> {
  return request(`/admin/ps/receivers/${id}`, { method: 'DELETE' })
}
