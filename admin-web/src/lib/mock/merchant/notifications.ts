/**
 * 商户中心站内信假数据。商户视角：自己的订单/结算/账户/系统通知。
 */
export interface MerchantNotice {
  id: number
  type: 'order' | 'settle' | 'account' | 'system'
  title: string
  desc: string
  time: string
  unread: boolean
}

export const merchantNotices: MerchantNotice[] = [
  { id: 1, type: 'order', title: '新订单到账', desc: '您收到一笔 ¥3,420.50 的支付宝订单，已成功入账', time: '3分钟前', unread: true },
  { id: 2, type: 'settle', title: '提现处理中', desc: '您的提现申请 ¥9,800.00 正在处理，预计 1 个工作日内到账', time: '32分钟前', unread: true },
  { id: 3, type: 'account', title: '安全提醒', desc: '您还未绑定密保手机/邮箱，建议尽快前往账户设置绑定', time: '2小时前', unread: true },
  { id: 4, type: 'settle', title: '结算完成', desc: '结算 ¥4,800.00 已成功打款至您的支付宝账户', time: '昨天 18:20', unread: false },
  { id: 5, type: 'system', title: '平台公告', desc: '7月15日 02:00-04:00 系统例行维护，届时支付可能短暂波动', time: '昨天 09:00', unread: false },
  { id: 6, type: 'order', title: '退款完成', desc: '订单 20260711xxxx 退款 ¥30.00 已原路退回买家', time: '2天前 14:05', unread: false },
]
