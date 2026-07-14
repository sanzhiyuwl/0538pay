/**
 * 商户中心「订单记录」假数据。对齐 epay user/order.php + ajax2.php?act=orderList。
 * 仅【当前登录商户】的订单。区别于后台 Orders（看全站、可改状态/删除）。
 */

/** 订单状态：0未支付 1已支付 2已退款 3已冻结 4预授权 */
export const orderStatus: Record<number, { text: string; variant: 'warning' | 'success' | 'destructive' | 'muted' }> = {
  0: { text: '未支付', variant: 'warning' },
  1: { text: '已支付', variant: 'success' },
  2: { text: '已退款', variant: 'destructive' },
  3: { text: '已冻结', variant: 'muted' },
  4: { text: '预授权', variant: 'muted' },
}

/** 支付方式 */
export const payTypes = [
  { id: 1, name: 'alipay', showname: '支付宝' },
  { id: 2, name: 'wxpay', showname: '微信支付' },
  { id: 3, name: 'qqpay', showname: 'QQ钱包' },
  { id: 4, name: 'bank', showname: '云闪付' },
]

/** 搜索类型（对齐 order.php type 下拉） */
export const searchColumns = [
  { value: 'trade_no', label: '系统订单号' },
  { value: 'out_trade_no', label: '商户订单号' },
  { value: 'api_trade_no', label: '接口订单号' },
  { value: 'buyer', label: '用户交易号' },
  { value: 'name', label: '商品名称' },
  { value: 'money', label: '商品金额' },
  { value: 'realmoney', label: '实付金额' },
  { value: 'domain', label: '网站域名' },
  { value: 'ip', label: '支付IP' },
  { value: 'account', label: '支付账号' },
]

export interface Order {
  trade_no: string // 系统订单号
  out_trade_no: string // 商户订单号
  api_trade_no: string // 接口订单号
  buyer: string // 用户交易号
  name: string // 商品名称
  money: string // 商品金额
  realmoney: string | null // 实际支付
  refundmoney: string // 已退款金额
  type: number // 支付方式 id
  typename: string // 支付方式标识
  typeshowname: string // 支付方式显示名
  channel: number // 通道 id
  domain: string // 网站域名
  ip: string // 支付 IP
  account: string // 支付账号
  addtime: string // 创建时间
  endtime: string | null // 完成时间
  status: number
}

const goods = ['充值套餐A', 'VIP会员月卡', '游戏点券100', '知识付费专栏', '虚拟商品-礼包', '话费充值50元', '流量包1GB', '课程订阅']
const domains = ['shop.abc.com', 'pay.demo.cn', 'vip.test.net', 'www.mysite.com']

function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}

// 生成 48 条当前商户订单
export const orders: Order[] = Array.from({ length: 48 }, (_, i) => {
  const t = payTypes[i % 4]
  // 状态分布：多数已支付，少量未付/退款/冻结
  const r = i % 10
  const status = r < 6 ? 1 : r < 8 ? 0 : r === 8 ? 2 : 3
  const money = (10 + ((i * 37) % 990) + (i % 100) / 100).toFixed(2)
  const paid = status === 1 || status === 2
  const day = 12 - (i % 9)
  const hh = pad(8 + (i % 12))
  const mm = pad((i * 7) % 60)
  const addtime = `2026-07-${pad(day)} ${hh}:${mm}:${pad((i * 13) % 60)}`
  const refund = status === 2 ? (Number(money) * (i % 2 === 0 ? 1 : 0.5)).toFixed(2) : '0.00'
  return {
    trade_no: `2026071${pad(day)}${pad(hh as unknown as number)}${pad(100000 + i * 137, 6)}`.slice(0, 19),
    out_trade_no: `OUT${pad(20260700 + i, 8)}`,
    api_trade_no: i % 3 === 0 ? `API${pad(i * 991, 8)}` : '',
    buyer: paid ? `buyer_${pad((i * 17) % 100000, 5)}` : '',
    name: goods[i % goods.length],
    money,
    realmoney: paid ? money : null,
    refundmoney: refund,
    type: t.id,
    typename: t.name,
    typeshowname: t.showname,
    channel: (i % 4) + 1,
    domain: domains[i % domains.length],
    ip: `${100 + (i % 100)}.${(i * 7) % 255}.${(i * 3) % 255}.${i % 255}`,
    account: paid ? `${t.name}_${pad((i * 23) % 100000, 5)}` : '',
    addtime,
    endtime: paid ? `2026-07-${pad(day)} ${hh}:${pad(((i * 7) % 58) + 1)}:${pad((i * 19) % 60)}` : null,
    status,
  }
})

/** 汇总统计（对齐 order.php statistics 弹窗） */
export function calcStats(list: Order[]) {
  const paidList = list.filter((o) => o.status === 1)
  const unpaidList = list.filter((o) => o.status === 0)
  const refundList = list.filter((o) => o.status === 2)
  const sum = (arr: Order[], f: (o: Order) => number) => arr.reduce((s, o) => s + f(o), 0)
  const totalMoney = sum(list, (o) => Number(o.money))
  const successMoney = sum(paidList, (o) => Number(o.realmoney ?? 0))
  const unpaidMoney = sum(unpaidList, (o) => Number(o.money))
  const refundMoney = sum(refundList, (o) => Number(o.refundmoney))
  // 商户视角"总收入" = 已支付实收 - 已退款（不含平台利润口径）
  const income = successMoney - refundMoney
  const totalCount = list.length
  const successCount = paidList.length
  return {
    totalMoney,
    successMoney,
    unpaidMoney,
    refundMoney,
    income,
    totalCount,
    successCount,
    unpaidCount: unpaidList.length,
    refundCount: refundList.length,
    successRate: totalCount ? Math.round((successCount / totalCount) * 1000) / 10 : 0,
  }
}
