/** 代付/转账 API（后台 + 商户端）。对齐 epay transfer + 后端 dto.TransferView。 */
import { request, type PageResult } from './client'

/** 代付记录（对齐后端 dto.TransferView / mock/transfer.ts TransferRecord） */
export interface TransferRecord {
  biz_no: string
  pay_order_no: string
  uid: number // 0=管理员发起
  type: string // alipay/wxpay/qqpay/bank
  channel: number
  account: string
  username: string
  money: string
  costmoney: string
  desc: string
  addtime: string
  paytime: string | null
  status: number // 0处理中 1成功 2失败
  result: string
}

/** 代付概况统计（对齐后端 dto.TransferStats） */
export interface TransferStats {
  total: number
  totalMoney: number
  successMoney: number
  successCount: number
  processingCount: number
  failCount: number
}

export interface TransferListParams {
  page?: number
  pageSize?: number
  keyword?: string // 交易号/收款账号/姓名
  uid?: number
  type?: string
  status?: number
}

/** 发起代付入参（后台/商户共用） */
export interface TransferCreateReq {
  biz_no?: string
  type: string
  channel?: number
  account: string
  username?: string
  money: string
  desc?: string
  password: string // 后台=管理员密码 / 商户=登录密码
}

// ===== 后台 =====
export function fetchTransfers(params: TransferListParams = {}): Promise<PageResult<TransferRecord>> {
  return request<PageResult<TransferRecord>>('/admin/transfers', { query: { ...params } })
}

export function fetchTransferStats(params: TransferListParams = {}): Promise<TransferStats> {
  return request<TransferStats>('/admin/transfers/stats', { query: { ...params } })
}

export function createTransfer(body: TransferCreateReq): Promise<{ biz_no: string }> {
  return request('/admin/transfers', { method: 'POST', body })
}

export function setTransferStatus(bizNo: string, status: number, result = ''): Promise<{ biz_no: string; status: number }> {
  return request(`/admin/transfers/${encodeURIComponent(bizNo)}/status`, {
    method: 'PUT',
    body: { status, result },
  })
}

export function refundTransfer(bizNo: string): Promise<{ biz_no: string }> {
  return request(`/admin/transfers/${encodeURIComponent(bizNo)}/refund`, { method: 'POST' })
}

export function deleteTransfer(bizNo: string): Promise<{ biz_no: string }> {
  return request(`/admin/transfers/${encodeURIComponent(bizNo)}`, { method: 'DELETE' })
}

// ===== 商户端 =====
export function fetchMerchantTransfers(params: TransferListParams = {}): Promise<PageResult<TransferRecord>> {
  return request<PageResult<TransferRecord>>('/merchant/transfers', { query: { ...params } })
}

export function createMerchantTransfer(body: TransferCreateReq): Promise<{ biz_no: string }> {
  return request('/merchant/transfer', { method: 'POST', body })
}
