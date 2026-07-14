/**
 * 商户中心工作台首页假数据。对齐 epay user/index.php + ajax2.php?act=getcount。
 * 全部为【当前登录商户】维度。
 */

/** 当前商户信息 */
export const merchantInfo = {
  uid: 1001,
  name: '泰安优选商贸',
  qq: '800538',
  // 商户状态：normal 正常 / banned 已封禁 / payoff 关闭支付 / settleoff 关闭结算 / auditing 待审核 / uncert 未实名
  status: 'normal' as 'normal' | 'banned' | 'payoff' | 'settleoff' | 'auditing' | 'uncert',
  groupName: '普通会员',
  money: 12860.55, // 当前余额
  settleMoney: 38200.0, // 已结算余额
  todayIncome: 3420.5, // 今日收入
  yesterdayIncome: 2980.0, // 昨日收入
  orders: 5832, // 订单总数
  ordersToday: 128, // 今日订单
}

/** 商户状态字典 */
export const merchantStatusMeta: Record<
  string,
  { text: string; variant: 'success' | 'warning' | 'destructive' | 'muted' }
> = {
  normal: { text: '正常', variant: 'success' },
  banned: { text: '已封禁', variant: 'destructive' },
  payoff: { text: '关闭支付', variant: 'warning' },
  settleoff: { text: '关闭结算', variant: 'warning' },
  auditing: { text: '待审核', variant: 'muted' },
  uncert: { text: '未实名', variant: 'warning' },
}

/** 顶部条件提醒横幅（mock 开关，真实环境由账户状态/配置决定） */
export const dashboardAlerts = {
  needCert: false, // 未实名强制提醒
  noSecurity: true, // 未绑定密保手机/邮箱
  noLoginPwd: false, // 未设置登录密码
}

/** 收入统计与通道费率（对齐 index.php 通道表 + getcount channels[]） */
export interface ChannelStat {
  typename: string // 支付方式标识（图标名）
  showname: string // 显示名
  today: number // 今日金额
  yesterday: number // 昨日金额
  successRate: number // 成功率 %
  rate: string // 费率 %
}

export const channelStats: ChannelStat[] = [
  { typename: 'alipay', showname: '支付宝', today: 1580.5, yesterday: 1320.0, successRate: 98.2, rate: '0.38' },
  { typename: 'wxpay', showname: '微信支付', today: 1240.0, yesterday: 1180.5, successRate: 97.6, rate: '0.38' },
  { typename: 'qqpay', showname: 'QQ钱包', today: 320.0, yesterday: 280.0, successRate: 95.0, rate: '0.60' },
  { typename: 'bank', showname: '云闪付', today: 280.0, yesterday: 199.5, successRate: 96.4, rate: '0.45' },
]

/** 公告通知（对齐 pre_anounce） */
export interface Announce {
  id: number
  content: string
  color: string
  time: string
}
export const announces: Announce[] = [
  { id: 3, content: '平台已支持微信、支付宝服务商模式，费率低至 0.38%，欢迎咨询。', color: '#e11d48', time: '2026-07-10 09:00' },
  { id: 2, content: '7月15日 02:00-04:00 系统例行维护，届时支付可能短暂波动。', color: '#f59e0b', time: '2026-07-08 15:30' },
  { id: 1, content: '新增云闪付通道，支持大额收款，单笔最高 5 万元。', color: '', time: '2026-07-05 11:20' },
]

/** 结算金额趋势（近 9 条已结算记录，对齐 index.php flot 图） */
export const settleTrend = {
  labels: ['06-20', '06-25', '06-28', '07-01', '07-04', '07-07', '07-09', '07-11', '07-12'],
  data: [3200, 4100, 2800, 5300, 3900, 4600, 5100, 3700, 4800],
}
