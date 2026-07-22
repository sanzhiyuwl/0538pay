/** 后台登录相关 API */
import { request, setToken } from './client'

export interface LoginResp {
  token: string
  nickname: string
  role: string
}

/** 后台登录，成功后自动存 token */
export async function login(username: string, password: string): Promise<LoginResp> {
  const data = await request<LoginResp>('/admin/login', {
    method: 'POST',
    body: { username, password },
  })
  setToken(data.token)
  return data
}

/** 当前管理员账号资料 */
export interface AdminProfile {
  id: number
  username: string
  nickname: string
  role: string
}

export function fetchProfile(): Promise<AdminProfile> {
  return request<AdminProfile>('/admin/profile')
}

/** 修改当前管理员昵称/用户名 */
export function updateProfile(body: { nickname: string; username: string }): Promise<{ ok: boolean }> {
  return request('/admin/profile', { method: 'PUT', body })
}

/** 修改当前管理员登录密码 */
export function changePassword(body: { oldpwd: string; newpwd: string; newpwd2: string }): Promise<{ ok: boolean }> {
  return request('/admin/password', { method: 'PUT', body })
}

/** 修改管理员支付密码（对齐 epay admin_paypwd，用于转账/结算/退款二次校验；独立于登录密码） */
export function changePayPassword(body: { oldpwd: string; newpwd: string; newpwd2: string }): Promise<{ ok: boolean }> {
  return request('/admin/paypwd', { method: 'PUT', body })
}
