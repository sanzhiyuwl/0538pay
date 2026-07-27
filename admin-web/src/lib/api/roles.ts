/**
 * 后台角色管理 API（RBAC 增强，我方独有；epay 原版单管理员无角色体系）。
 * 角色 → 可访问的功能模块权限；管理员账号的 role 字符串与角色 code 对应。
 * 对齐后端 dto.RoleView / dto.RoleSaveReq。
 */
import { request } from './client'

/** 单个角色 */
export interface Role {
  id: number
  code: string
  name: string
  desc: string
  permissions: string[] // permModules 的 key，['*'] 表示全部
  builtin: boolean // 内置角色不可删除
}

/** 新增 / 编辑角色入参 */
export interface RoleSaveReq {
  code?: string
  name: string
  desc?: string
  permissions: string[]
}

/** 可分配的权限模块（对齐主后台一级菜单组）。 */
export const permModules = [
  { key: 'dashboard', label: '平台概况' },
  { key: 'trade', label: '交易管理' },
  { key: 'merchant', label: '商户管理' },
  { key: 'channel', label: '支付接口' },
  { key: 'finance', label: '财务管理' },
  { key: 'risk', label: '风控管理' },
  { key: 'system', label: '系统设置' },
  { key: 'console', label: 'SaaS 控制台' },
]

/** 角色列表（首次访问后端播种内置角色）。 */
export function fetchRoles(): Promise<{ list: Role[] }> {
  return request<{ list: Role[] }>('/admin/roles')
}

/** 新增角色。 */
export function createRole(body: RoleSaveReq): Promise<{ ok: boolean }> {
  return request<{ ok: boolean }>('/admin/roles', { method: 'POST', body })
}

/** 编辑角色。 */
export function updateRole(id: number, body: RoleSaveReq): Promise<{ id: number }> {
  return request<{ id: number }>(`/admin/roles/${id}`, { method: 'PUT', body })
}

/** 删除角色。 */
export function deleteRole(id: number): Promise<{ id: number }> {
  return request<{ id: number }>(`/admin/roles/${id}`, { method: 'DELETE' })
}

/** 角色权限展示文案。 */
export function rolePermText(role: Role): string {
  if (role.permissions.includes('*')) return '全部权限'
  return role.permissions
    .map((k) => permModules.find((m) => m.key === k)?.label)
    .filter(Boolean)
    .join('、')
}
