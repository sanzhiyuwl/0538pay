/** 风控 / 黑名单 / 授权域名 API（C4）。对齐后端 dto。 */
import { request, type PageResult } from './client'

// ===== 风控（只读）=====
export interface RiskRecord {
  id: number
  uid: number
  type: number // 0关键词 1成功率 2通知失败 3投诉率
  content: string
  url: string
  date: string
}
export interface RiskListParams {
  page?: number
  pageSize?: number
  column?: string
  value?: string
  type?: number
}
export function fetchRisks(params: RiskListParams = {}): Promise<PageResult<RiskRecord>> {
  return request<PageResult<RiskRecord>>('/admin/risks', { query: { ...params } })
}

// ===== 黑名单 =====
export interface BlackItem {
  id: number
  type: number // 0支付账号 1IP
  content: string
  addtime: string
  endtime: string | null
  remark: string
}
export interface BlackStats {
  total: number
  account: number
  ip: number
  permanent: number
}
export interface BlackListParams {
  page?: number
  pageSize?: number
  kw?: string
  type?: number
}
export function fetchBlacklist(params: BlackListParams = {}): Promise<PageResult<BlackItem>> {
  return request<PageResult<BlackItem>>('/admin/blacklist', { query: { ...params } })
}
export function fetchBlackStats(): Promise<BlackStats> {
  return request<BlackStats>('/admin/blacklist/stats')
}
export function addBlacklist(body: { type: number; content: string; days: number; remark: string }): Promise<{ ok: boolean }> {
  return request('/admin/blacklist', { method: 'POST', body })
}
export function deleteBlacklist(id: number): Promise<{ id: number }> {
  return request(`/admin/blacklist/${id}`, { method: 'DELETE' })
}
export function batchDeleteBlacklist(ids: number[]): Promise<{ deleted: number }> {
  return request('/admin/blacklist/batch-delete', { method: 'POST', body: { ids } })
}

// ===== 授权域名 =====
export interface DomainItem {
  id: number
  uid: number
  domain: string
  status: number // 0待审核 1正常 2拒绝
  addtime: string
  endtime: string | null
}
export interface DomainStats {
  total: number
  pending: number
  normal: number
  rejected: number
}
export interface DomainListParams {
  page?: number
  pageSize?: number
  uid?: number
  kw?: string
  dstatus?: number
}
export function fetchDomains(params: DomainListParams = {}): Promise<PageResult<DomainItem>> {
  return request<PageResult<DomainItem>>('/admin/domains', { query: { ...params } })
}
export function fetchDomainStats(): Promise<DomainStats> {
  return request<DomainStats>('/admin/domains/stats')
}
export function addDomain(body: { uid: number; domain: string }): Promise<{ ok: boolean }> {
  return request('/admin/domains', { method: 'POST', body })
}
export function setDomainStatus(id: number, status: number): Promise<{ id: number; status: number }> {
  return request(`/admin/domains/${id}/status`, { method: 'PUT', body: { status } })
}
export function deleteDomain(id: number): Promise<{ id: number }> {
  return request(`/admin/domains/${id}`, { method: 'DELETE' })
}
export function batchOpDomain(ids: number[], status: number): Promise<{ affected: number }> {
  return request('/admin/domains/batch', { method: 'POST', body: { ids, status } })
}
