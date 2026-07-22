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
  keylogin: number
  refund: number
  transfer: number
  remain_money: string
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

// ===== 自助流程：图形验证码 / 注册 / 找回密码 / 完善资料 =====

/** 图形验证码（返回 token + SVG 图，公开） */
export interface CaptchaResp {
  token: string
  svg: string
}
export function fetchCaptcha(): Promise<CaptchaResp> {
  return request<CaptchaResp>('/merchant/captcha')
}

/** 商户注册（公开）。verifytype: 0邮箱 1手机 */
export interface MerchantRegReq {
  verifytype: number
  account: string
  password: string
  invite?: string
  captcha_token: string
  captcha: string
}
export interface MerchantRegResp {
  uid: number
  need_review: boolean
  msg: string
}
export function merchantRegister(body: MerchantRegReq): Promise<MerchantRegResp> {
  return request<MerchantRegResp>('/merchant/register', { method: 'POST', body })
}

/** 找回密码（公开）。type: email/phone */
export interface MerchantFindPwdReq {
  type: string
  account: string
  password: string
  captcha_token: string
  captcha: string
}
export function merchantFindPwd(body: MerchantFindPwdReq): Promise<{ msg: string }> {
  return request<{ msg: string }>('/merchant/findpwd', { method: 'POST', body })
}

/** 完善资料（需登录） */
export interface MerchantCompleteReq {
  settle_id: number
  account: string
  username: string
  qq?: string
  url?: string
  email?: string
}
export function merchantComplete(body: MerchantCompleteReq): Promise<{ uid: number }> {
  return request<{ uid: number }>('/merchant/complete', { method: 'POST', body })
}

// ===== 快捷登录 OAuth（QQ/微信/支付宝，对齐 epay connect/wxlogin/oauth）=====
export interface OAuthResult {
  token?: string
  info?: MerchantInfo
  need_bind: boolean
  provider?: string
  openid?: string
}
/** 取第三方授权跳转 URL */
export function fetchOAuthURL(provider: string, redirect: string, state: string): Promise<{ url: string }> {
  return request(`/merchant/oauth/${provider}/url`, { query: { redirect, state } })
}
/** 回调换 openid → 登录或 need_bind */
export function oauthCallback(provider: string, code: string, redirect: string): Promise<OAuthResult> {
  return request(`/merchant/oauth/${provider}/callback`, { method: 'POST', body: { code, redirect } })
}
/** 未绑定用户输入账号密码绑定 openid 并登录 */
export function oauthBind(body: { provider: string; openid: string; account: string; password: string; type: number }): Promise<OAuthResult> {
  return request('/merchant/oauth/bind', { method: 'POST', body })
}
