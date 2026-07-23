/** 收单相关公开 API（无需 JWT）：收银台查单 + 模拟支付回调触发。 */
import { request } from './client'

/** 收银台中间页所需的公开订单信息（对齐后端 dto.CashierView） */
/** 收银台可选支付方式（B1-04 裸单聚合选方式） */
export interface CashierPayType {
  type: string // 渠道插件 key（复发起下单时作 type 传入）
  showname: string // 支付方式友好名
}
export interface CashierOrder {
  trade_no: string
  out_trade_no: string
  name: string
  money: string // 需支付金额（含加费/随机微调）
  order_money: string // 原始订单金额（B1-65：money≠order_money 时显示'含X元手续费'）
  plugin: string
  pay_type: string // qrcode/redirect/html（真实渠道）；mock 为空
  qrcode: string // 二维码内容/支付链接（真实渠道）；mock 为空
  status: number // 0未付 1已付…
  addtime: string
  return_url: string
  paytypes?: CashierPayType[] // B1-04：裸单(空 type 未定通道)时的可选支付方式；已定通道单为空
}

/** 按系统订单号查收银台订单信息 */
export function fetchCashierOrder(tradeNo: string): Promise<CashierOrder> {
  return request<CashierOrder>(`/pay/order/${encodeURIComponent(tradeNo)}`)
}

/** 收银台下单返回（对齐后端 dto.SubmitResp） */
export interface CashierSubmitResp {
  trade_no: string
  out_trade_no: string
  pay_type: string
  pay_url?: string
  qrcode?: string
  html?: string
  money: string
}

/**
 * B1-04 收银台选定支付方式：对既有裸单(空 type 单)按 trade_no 补选通道下单，无需商户签名
 * （订单在空 type 下单时已验签建号）。返回选定通道后的下单信息（二维码/跳转等）。
 */
export function cashierChoosePay(tradeNo: string, type: string): Promise<CashierSubmitResp> {
  return request<CashierSubmitResp>('/pay/choose', {
    method: 'POST',
    body: { trade_no: tradeNo, type },
  })
}

/**
 * 轮询查单：真实渠道扫码后前端轮询。走 /pay/query 让后端未付时主动问渠道，
 * 渠道确认已付则后端就地改单入账。返回 status（1=已支付）。
 */
export async function fetchOrderStatus(tradeNo: string): Promise<number> {
  const r = await request<{ status: number }>(`/pay/query/${encodeURIComponent(tradeNo)}`)
  return r.status
}

/**
 * 触发 mock 渠道的模拟支付回调（仅 mock 渠道联调用）。
 * 直接命中后端回调路由，走完 验签→改单→入账→通知 全链路。
 */
export async function triggerMockPay(order: CashierOrder): Promise<string> {
  const body = new URLSearchParams({
    trade_no: order.trade_no,
    trade_status: 'TRADE_SUCCESS',
    money: order.money,
    channel_no: 'MOCK' + Date.now(),
    buyer: 'mock_buyer',
  })
  const res = await fetch(`/api/pay/notify/${encodeURIComponent(order.trade_no)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body,
  })
  return res.text() // 后端回纯文本 success/fail
}
