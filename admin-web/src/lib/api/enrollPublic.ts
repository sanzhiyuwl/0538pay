/** 客户自助进件公开页 API（/enroll/:code，免登录，靠邀请 code）。自研扩展，epay 无。 */
import { request } from './client'

/** 公开页落地信息：归属代理名 + 开户价说明（不含敏感字段） */
export interface EnrollPublicInfo {
  code: string
  agent_name: string
  retail_note: string
}

/** 图形验证码（token + SVG，公开） */
export interface EnrollCaptchaResp {
  token: string
  svg: string
}

/** 自助建单收银台信息（对齐后端 SubmitResp，pay 可能为 null=免付直放行） */
export interface EnrollSubmitPay {
  trade_no: string
  out_trade_no: string
  pay_type: string
  money: string
}

/** 自助建单返回：进件单 + 收银台（pay 为 null 表示免开户费，已直接放行） */
export interface EnrollSubmitResp {
  enroll: { enroll_no: string; merchant_name: string; status: string }
  pay: EnrollSubmitPay | null
}

/** 落地校验邀请可用 + 打点打开数，返回展示信息 */
export function fetchEnrollInfo(code: string): Promise<EnrollPublicInfo> {
  return request<EnrollPublicInfo>(`/enroll/${encodeURIComponent(code)}`)
}

/** 下发图形验证码 */
export function fetchEnrollCaptcha(): Promise<EnrollCaptchaResp> {
  return request<EnrollCaptchaResp>('/enroll/captcha')
}

/** 自助提交建单（校验验证码 → source=3 建单 → 收银台） */
export interface EnrollSubmitReq {
  merchant_name: string
  contact_phone: string
  plugin: string
  captcha_token: string
  captcha_code: string
}
export function submitEnroll(code: string, body: EnrollSubmitReq): Promise<EnrollSubmitResp> {
  return request<EnrollSubmitResp>(`/enroll/${encodeURIComponent(code)}`, { method: 'POST', body })
}
