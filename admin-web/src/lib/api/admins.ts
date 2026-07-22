/** 管理员账号 CRUD（RBAC 增强，对齐后端 /admin/admins）。 */
import { request } from './client'

export interface AdminAccount {
  id: number
  username: string
  nickname: string
  role: string // super=超级管理员，其余为自定义角色名
  status: number // 0 停用 1 正常
  last_login: string
  created_at: string
}

export interface AdminSaveReq {
  username: string
  nickname: string
  password?: string
  role: string
  status: number
}

export function fetchAdmins(): Promise<AdminAccount[]> {
  return request<AdminAccount[]>('/admin/admins')
}

export function createAdmin(body: AdminSaveReq): Promise<{ ok: boolean }> {
  return request('/admin/admins', { method: 'POST', body })
}

export function updateAdmin(id: number, body: AdminSaveReq): Promise<{ ok: boolean }> {
  return request(`/admin/admins/${id}`, { method: 'PUT', body })
}

export function setAdminStatus(id: number, status: number): Promise<{ ok: boolean }> {
  return request(`/admin/admins/${id}/status`, { method: 'PUT', body: { status } })
}

export function deleteAdmin(id: number): Promise<{ ok: boolean }> {
  return request(`/admin/admins/${id}`, { method: 'DELETE' })
}
