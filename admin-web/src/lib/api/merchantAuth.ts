/** 商户中心登录相关 API */
import { request, setMerchantToken } from './client'

/** 当前登录商户信息（对齐后端 dto.MerchantInfo） */
export interface MerchantInfo {
  uid: number
  name: string
  money: string
  status: number // 0封禁1正常2未审核
  pay: number
  settle: number
  cert: number
  email: string
  phone: string
  qq: string
  gid: number
  settle_id: number
  account: string
  username: string
  url: string
  mode: number
}

export interface MerchantLoginResp {
  token: string
  info: MerchantInfo
}

/**
 * 商户登录，成功后自动存 merchant token。
 * type=1 密码登录(account=邮箱/手机)；type=0 密钥登录(account=商户ID, password=通信密钥)。
 */
export async function merchantLogin(
  type: 0 | 1,
  account: string,
  password: string,
): Promise<MerchantLoginResp> {
  const data = await request<MerchantLoginResp>('/merchant/login', {
    method: 'POST',
    body: { type, account, password },
  })
  setMerchantToken(data.token)
  return data
}

/** 拉取当前登录商户信息 */
export function fetchMerchantInfo(): Promise<MerchantInfo> {
  return request<MerchantInfo>('/merchant/info')
}
