/** 公开聚合收款页 API（对齐 epay paypage/index.php，无需登录）。 */
import { request } from './client'
import type { PayTypeOption, SubmitResp } from './merchantCenter'

export interface PaypageInfo {
  codename: string
  sitename: string
  types: PayTypeOption[]
}

/** 收款页信息：收款方名称 + 可选支付方式 */
export function fetchPaypageInfo(merchant: string): Promise<PaypageInfo> {
  return request<PaypageInfo>('/paypage/info', { query: { merchant } })
}

/** 收款页下单：输入金额 + 选支付方式 → 走收单链 */
export function submitPaypage(merchant: string, money: string, type: string): Promise<SubmitResp> {
  return request<SubmitResp>('/paypage/submit', { method: 'POST', body: { merchant, money, type } })
}
