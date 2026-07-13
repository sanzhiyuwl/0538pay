export interface Notice {
  id: number
  type: 'order' | 'settle' | 'risk' | 'system'
  title: string
  desc: string
  time: string
  unread: boolean
}

export const notices: Notice[] = [
  { id: 1, type: 'risk', title: '风控预警', desc: '商户「极客数码」触发单IP限额规则，已拦截 3 笔订单', time: '2分钟前', unread: true },
  { id: 2, type: 'settle', title: '结算申请', desc: '商户「云上便利店」提交提现申请 ¥12,600.00，待审核', time: '15分钟前', unread: true },
  { id: 3, type: 'order', title: '大额订单', desc: '收到一笔 ¥25,990.00 的支付宝订单，已成功', time: '1小时前', unread: true },
  { id: 4, type: 'system', title: '系统通知', desc: '支付通道「微信服务商」今日成功率 96.2%，运行正常', time: '3小时前', unread: false },
  { id: 5, type: 'settle', title: '结算完成', desc: '批次 B20260712 共 37 笔结算已打款完成', time: '昨天 18:20', unread: false },
  { id: 6, type: 'order', title: '退款提醒', desc: '商户「快充能源」发起退款 ¥30.00，已原路退回', time: '昨天 14:05', unread: false },
]
