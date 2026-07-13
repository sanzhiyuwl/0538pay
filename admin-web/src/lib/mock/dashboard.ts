/**
 * 仪表盘假数据。字段结构对齐 epay admin/ajax.php?act=getcount 的真实返回：
 * { count1, count2, usermoney, settlemoney, success_rate,
 *   paytype:{name:showname}, channel:{id:name},
 *   order:{ 日期:{ all, profit_all, paytype:{}, channel:{}, profit_paytype:{} } },
 *   order_today:{ 同上 } }
 */

export interface DailyRow {
  all: number
  profit_all: number
  paytype: Record<string, number>
  channel: Record<string, number>
  profit_paytype: Record<string, number>
}

export interface GetCount {
  count1: number
  count2: number
  usermoney: string
  settlemoney: string
  success_rate: string
  paytype: Record<string, string>
  channel: Record<string, string>
  order_today: DailyRow
  order: Record<string, DailyRow>
}

// 支付方式字典（对齐 pre_type）
const paytype = {
  alipay: '支付宝',
  wxpay: '微信支付',
  qqpay: 'QQ钱包',
  bank: '云闪付',
}

// 通道字典（对齐 pre_channel）
const channel = {
  '1': '支付宝当面付',
  '2': '微信服务商',
  '3': 'QQ钱包直连',
  '4': '云闪付银联',
}

function row(base: number): DailyRow {
  const ali = +(base * 0.42).toFixed(2)
  const wx = +(base * 0.38).toFixed(2)
  const qq = +(base * 0.11).toFixed(2)
  const bank = +(base * 0.09).toFixed(2)
  const all = +(ali + wx + qq + bank).toFixed(2)
  return {
    all,
    profit_all: +(all * 0.018).toFixed(2),
    paytype: { alipay: ali, wxpay: wx, qqpay: qq, bank },
    channel: { '1': ali, '2': wx, '3': qq, '4': bank },
    profit_paytype: {
      alipay: +(ali * 0.018).toFixed(2),
      wxpay: +(wx * 0.016).toFixed(2),
      qqpay: +(qq * 0.02).toFixed(2),
      bank: +(bank * 0.012).toFixed(2),
    },
  }
}

// 近 7 日（含波动）
const dailyBases = [86420, 92310, 78650, 105230, 98760, 121540, 134280]
const dates = [
  '07-06',
  '07-07',
  '07-08',
  '07-09',
  '07-10',
  '07-11',
  '07-12',
]

const order: Record<string, DailyRow> = {}
dates.forEach((d, i) => {
  order[d] = row(dailyBases[i])
})

export const dashboardData: GetCount = {
  count1: 284736,
  count2: 1842,
  usermoney: '386452.18',
  settlemoney: '12847560.42',
  success_rate: '98.7',
  paytype,
  channel,
  order_today: row(142380),
  order,
}

// 最近订单（用于仪表盘底部列表）
export interface RecentOrder {
  trade_no: string
  merchant: string
  type: keyof typeof paytype
  money: string
  status: 0 | 1 | 2
  time: string
}

export const recentOrders: RecentOrder[] = [
  { trade_no: '2026071214380012', merchant: '云上便利店', type: 'wxpay', money: '128.00', status: 1, time: '14:38:02' },
  { trade_no: '2026071214372988', merchant: '极客数码', type: 'alipay', money: '2599.00', status: 1, time: '14:37:29' },
  { trade_no: '2026071214361145', merchant: '花间集鲜花', type: 'alipay', money: '199.90', status: 0, time: '14:36:11' },
  { trade_no: '2026071214352077', merchant: '悦读书屋', type: 'wxpay', money: '58.00', status: 1, time: '14:35:20' },
  { trade_no: '2026071214341056', merchant: '快充能源', type: 'qqpay', money: '30.00', status: 2, time: '14:34:10' },
  { trade_no: '2026071214332901', merchant: '云上便利店', type: 'bank', money: '860.00', status: 1, time: '14:33:29' },
  { trade_no: '2026071214320088', merchant: '优选生活馆', type: 'wxpay', money: '426.50', status: 1, time: '14:32:00' },
]

/** 实时概况：今日数值 + 昨日对比 + 累计（对齐 NIUSHOP 概况卡片布局，内容为支付业务） */
export interface OverviewStat {
  label: string
  today: string
  yesterday: string
  totalLabel: string
  total: string
  hint?: string
}

export const overviewStats: OverviewStat[] = [
  {
    label: '今日订单数',
    today: '3,286',
    yesterday: '2,974',
    totalLabel: '订单总数',
    total: '284,736',
    hint: '今日成功支付的订单笔数',
  },
  {
    label: '今日交易额',
    today: '142,380.00',
    yesterday: '121,540.00',
    totalLabel: '交易总额(元)',
    total: '12,847,560',
    hint: '今日成功支付的订单总金额',
  },
  {
    label: '今日退款金额',
    today: '1,860.00',
    yesterday: '2,340.00',
    totalLabel: '退款总额(元)',
    total: '86,452',
    hint: '今日发起并成功的退款金额',
  },
  {
    label: '今日手续费收入',
    today: '2,562.84',
    yesterday: '2,187.72',
    totalLabel: '手续费总收入(元)',
    total: '231,256',
    hint: '已扣除通道成本后的平台利润',
  },
]

/** 待办事项（对齐 NIUSHOP 待办区，内容为支付运营待处理项） */
export interface TodoStat {
  label: string
  value: number
  hint?: string
  urgent?: boolean
}

export const todoStats: TodoStat[] = [
  { label: '待审核商户', value: 12, hint: '提交入驻资料待审核', urgent: true },
  { label: '待处理结算', value: 37, hint: '商户提现申请待打款' },
  { label: '待处理代付', value: 8 },
  { label: '待处理分账', value: 3 },
  { label: '风控预警', value: 5, hint: '触发风控规则的订单', urgent: true },
  { label: '未处理投诉', value: 2, urgent: true },
]
