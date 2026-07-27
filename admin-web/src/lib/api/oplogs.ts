/**
 * 商户操作日志 API（我方独有安全审计增强，epay 无此概念）。
 * 记录商户在 /m 端的写操作（改资料/提现/退款/绑域名/改密钥等），管理端查询/导出。
 * 对齐后端 dto.OpLogView / dto.OpLogQuery。
 */
import { request, type PageResult } from './client'

/** 单条操作日志 */
export interface OpLog {
  id: number
  scope: string
  uid: number
  operator: string
  action: string
  actionCN: string
  category: string // account/fund/auth/config
  level: string // normal/warning/danger
  target: string
  detail: string // JSON 明细字符串
  result: string // ok/fail
  ip: string
  date: string
}

/** 查询参数（多维筛选 + 分页） */
export interface OpLogParams {
  page?: number
  pageSize?: number
  uid?: number
  action?: string
  category?: string
  level?: string
  result?: string
  keyword?: string
  starttime?: string
  endtime?: string
}

/** 动作下拉选项 */
export interface OpActionOption {
  value: string
  label: string
  category: string
  level: string
}

/** 商户操作日志列表（多维筛选 + 分页）。 */
export function fetchMerchantOpLogs(params: OpLogParams = {}): Promise<PageResult<OpLog>> {
  return request<PageResult<OpLog>>('/admin/oplogs/merchant', { query: { ...params } })
}

/** 动作下拉选项（前端筛选用）。 */
export function fetchOpActionOptions(): Promise<{ actions: OpActionOption[] }> {
  return request<{ actions: OpActionOption[] }>('/admin/oplogs/merchant/options')
}

/** 管理端操作日志列表（scope=admin，复用同表；记录管理员在后台的写操作）。 */
export function fetchAdminOpLogs(params: OpLogParams = {}): Promise<PageResult<OpLog>> {
  return request<PageResult<OpLog>>('/admin/oplogs/admin', { query: { ...params } })
}

/** 管理端动作下拉选项（后端取已落库去重动作）。 */
export function fetchAdminOpActionOptions(): Promise<{ actions: OpActionOption[] }> {
  return request<{ actions: OpActionOption[] }>('/admin/oplogs/admin/options')
}

/** 导出管理端操作日志 CSV（UTF-8 BOM）。 */
export async function exportAdminOpLogs(params: OpLogParams = {}): Promise<void> {
  const token = localStorage.getItem('admin_token') || ''
  const qs = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== '') qs.append(k, String(v))
  }
  const suffix = qs.toString() ? `?${qs.toString()}` : ''
  const res = await fetch(`/api/admin/oplogs/admin/export${suffix}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) throw new Error(`导出失败(${res.status})`)
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `管理员操作日志_${new Date().toISOString().slice(0, 10)}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

/** 导出商户操作日志 CSV（UTF-8 BOM，浏览器直接下载）。对齐 exportBilling 的鉴权下载模式。 */
export async function exportMerchantOpLogs(params: OpLogParams = {}): Promise<void> {
  const token = localStorage.getItem('admin_token') || ''
  const qs = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== '') qs.append(k, String(v))
  }
  const suffix = qs.toString() ? `?${qs.toString()}` : ''
  const res = await fetch(`/api/admin/oplogs/merchant/export${suffix}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) throw new Error(`导出失败(${res.status})`)
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `商户操作日志_${new Date().toISOString().slice(0, 10)}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

/** 级别 → Badge 变体 + 中文。 */
export const opLevelMeta: Record<string, { text: string; variant: 'destructive' | 'warning' | 'muted' }> = {
  danger: { text: '高危', variant: 'destructive' },
  warning: { text: '重要', variant: 'warning' },
  normal: { text: '常规', variant: 'muted' },
}

/** 分类 → 中文。 */
export const opCategoryText: Record<string, string> = {
  account: '账户',
  fund: '资金',
  auth: '认证',
  config: '配置',
}

/** 结果 → Badge 变体 + 中文。 */
export const opResultMeta: Record<string, { text: string; variant: 'success' | 'destructive' }> = {
  ok: { text: '成功', variant: 'success' },
  fail: { text: '失败', variant: 'destructive' },
}

/** 分类筛选下拉选项。 */
export const opCategoryOptions = [
  { value: '', label: '全部分类' },
  { value: 'account', label: '账户' },
  { value: 'fund', label: '资金' },
  { value: 'auth', label: '认证' },
  { value: 'config', label: '配置' },
]

/** 级别筛选下拉选项。 */
export const opLevelOptions = [
  { value: '', label: '全部级别' },
  { value: 'danger', label: '高危' },
  { value: 'warning', label: '重要' },
  { value: 'normal', label: '常规' },
]

/** 结果筛选下拉选项。 */
export const opResultOptions = [
  { value: '', label: '全部结果' },
  { value: 'ok', label: '成功' },
  { value: 'fail', label: '失败' },
]
