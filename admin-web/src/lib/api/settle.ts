/** 结算 API。SettleRecord / SettleBatch 类型复用 mock 里已定义的结构（字段一致）。 */
import { request, type PageResult } from './client'
import type { SettleRecord, SettleBatch } from '@/lib/mock/settle'

export interface SettleListParams {
  page?: number
  pageSize?: number
  keyword?: string // 结算账号/姓名
  uid?: number
  type?: number
  status?: number
  batch?: string
}

/** 拉取结算明细列表（分页） */
export function fetchSettles(params: SettleListParams = {}): Promise<PageResult<SettleRecord>> {
  return request<PageResult<SettleRecord>>('/admin/settles', { query: { ...params } })
}

/** 结算明细概况（全量聚合，与列表同筛选，不分页） */
export interface SettleStats {
  totalMoney: string
  realMoney: string
  doneMoney: string
  totalCount: number
  doneCount: number
  pendingCount: number
  processingCount: number
  failCount: number
}
export function fetchSettleStats(
  params: Omit<SettleListParams, 'page' | 'pageSize'> = {},
): Promise<SettleStats> {
  return request<SettleStats>('/admin/settle/stats', { query: { ...params } })
}

/** 结算明细服务端 CSV 导出（与列表同筛选，全量不分页；修正旧版前端仅导出前 100 条） */
export async function exportSettles(
  params: Omit<SettleListParams, 'page' | 'pageSize'> = {},
): Promise<void> {
  const token = localStorage.getItem('admin_token') || ''
  const qs = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== '') qs.set(k, String(v))
  }
  const res = await fetch(`/api/admin/settle/export?${qs.toString()}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) throw new Error(`导出失败(${res.status})`)
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `结算明细_${new Date().toISOString().slice(0, 10)}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

/** 拉取结算批次列表（分页） */
export function fetchSettleBatches(
  params: { page?: number; pageSize?: number } = {},
): Promise<PageResult<SettleBatch>> {
  return request<PageResult<SettleBatch>>('/admin/settle/batches', { query: { ...params } })
}

/** 生成结算批次（收当前所有待结算记录），返回批次号与收批条数 */
export function createSettleBatch(): Promise<{ batch: string; count: number }> {
  return request<{ batch: string; count: number }>('/admin/settle/batch', { method: 'POST' })
}

/** 批次一键完成，返回置完成的记录数 */
export function completeSettleBatch(batch: string): Promise<{ affected: number }> {
  return request<{ affected: number }>(
    `/admin/settle/batch/${encodeURIComponent(batch)}/complete`,
    { method: 'POST' },
  )
}

/** 导出批次银行专用打款文件（C-4）。tmpl: mybank/alipay/wxpay/common。带鉴权头 fetch→Blob 下载。 */
export async function exportSettleBatch(batch: string, tmpl: string): Promise<void> {
  const token = localStorage.getItem('admin_token') || ''
  const res = await fetch(
    `/api/admin/settle/batch/${encodeURIComponent(batch)}/export?tmpl=${encodeURIComponent(tmpl)}`,
    { headers: token ? { Authorization: `Bearer ${token}` } : {} },
  )
  if (!res.ok) throw new Error(`导出失败(${res.status})`)
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `pay_${tmpl}_${batch}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

/**
 * 变更单条结算记录状态。
 * status: 0待结算 1已完成 2正在结算 3结算失败 4删除(余额退回)。
 * result: status=3 时可携带失败原因。
 */
export function setSettleStatus(
  id: number,
  status: number,
  result = '',
  password = '',
): Promise<{ id: number; status: number }> {
  return request<{ id: number; status: number }>(`/admin/settles/${id}/status`, {
    method: 'PUT',
    body: { status, result, password },
  })
}
