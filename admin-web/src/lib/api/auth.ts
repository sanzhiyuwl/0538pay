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
