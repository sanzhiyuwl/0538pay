/**
 * 订单假数据。字段对齐 epay admin/ajax_order.php?act=orderList 的真实返回
 * （pre_order LEFT JOIN pre_channel + 派生 typename/typeshowname）。
 */

export interface Order {
  trade_no: string // 系统订单号
  out_trade_no: string // 商户订单号
  api_trade_no: string // 接口订单号
  uid: number // 商户号
  domain: string // 网站域名
  name: string // 商品名称
  money: string // 订单金额
  realmoney: string | null // 实际支付
  getmoney: string // 商户分成
  refundmoney: string // 已退款金额
  profitmoney: string // 手续费利润
  type: number // 支付方式ID
  typename: string // 支付方式英文名（图标）
  typeshowname: string // 支付方式中文名
  channel: number // 通道ID
  plugin: string // 插件标识
  ip: string // 支付IP
  buyer: string // 支付账号
  addtime: string // 创建时间
  endtime: string | null // 完成时间
  status: 0 | 1 | 2 | 3 | 4 // 状态
  settle: 0 | 1 | 2 | 3 // 结算子状态
  combine: 0 | 1 // 是否合单
}

/** 状态字典（列表/详情通用） */
export const orderStatus: Record<
  number,
  { text: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'muted' }
> = {
  0: { text: '未支付', variant: 'default' },
  1: { text: '已支付', variant: 'success' },
  2: { text: '已退款', variant: 'destructive' },
  3: { text: '已冻结', variant: 'muted' },
  4: { text: '预授权', variant: 'warning' },
}

/** 支付方式字典（对齐 pre_type） */
export const payTypes = [
  { id: 1, name: 'alipay', showname: '支付宝' },
  { id: 2, name: 'wxpay', showname: '微信支付' },
  { id: 3, name: 'qqpay', showname: 'QQ钱包' },
  { id: 4, name: 'bank', showname: '云闪付' },
]

/** 搜索字段下拉（对齐 order.php 的 column 选项） */
export const searchColumns = [
  { value: 'trade_no', label: '系统订单号' },
  { value: 'out_trade_no', label: '商户订单号' },
  { value: 'api_trade_no', label: '接口订单号' },
  { value: 'name', label: '商品名称' },
  { value: 'money', label: '订单金额' },
  { value: 'realmoney', label: '实付金额' },
  { value: 'domain', label: '网站域名' },
  { value: 'buyer', label: '支付账号' },
  { value: 'ip', label: '支付IP' },
  { value: 'mobile', label: '手机号码' },
  { value: 'param', label: '扩展参数' },
]

const merchants = [
  { uid: 1001, domain: 'shop.example.com', name: '云上便利店' },
  { uid: 1002, domain: 'geek.example.com', name: '极客数码' },
  { uid: 1003, domain: 'flower.example.com', name: '花间集鲜花' },
  { uid: 1005, domain: 'book.example.com', name: '悦读书屋' },
  { uid: 1008, domain: 'charge.example.com', name: '快充能源' },
  { uid: 1012, domain: 'life.example.com', name: '优选生活馆' },
]
const goods = ['数码配件套装', '会员充值', '鲜花配送', '图书订购', '充电服务', '生活百货', '话费充值', '游戏点卡']
const typeMeta: Record<number, { name: string; showname: string; plugin: string; channel: number }> = {
  1: { name: 'alipay', showname: '支付宝', plugin: 'alipay', channel: 1 },
  2: { name: 'wxpay', showname: '微信支付', plugin: 'wxpaynp', channel: 2 },
  3: { name: 'qqpay', showname: 'QQ钱包', plugin: 'qqpay', channel: 3 },
  4: { name: 'bank', showname: '云闪付', plugin: 'unionpay', channel: 4 },
}

// 生成 60 条订单假数据（确定性，便于分页演示）
function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}
export const orders: Order[] = Array.from({ length: 60 }, (_, i) => {
  const m = merchants[i % merchants.length]
  const typeId = ((i % 4) + 1) as 1 | 2 | 3 | 4
  const tm = typeMeta[typeId]
  const statusPool: Order['status'][] = [1, 1, 1, 1, 0, 1, 2, 1, 4, 1, 3]
  const status = statusPool[i % statusPool.length]
  const moneyNum = +(((i * 37.5) % 2600) + 30).toFixed(2)
  const paid = status === 1 || status === 2
  const getmoneyNum = +(moneyNum * (1 - 0.02)).toFixed(2)
  const profitNum = +(moneyNum * 0.018).toFixed(2)
  const refundNum = status === 2 ? (i % 3 === 0 ? +(moneyNum / 2).toFixed(2) : moneyNum) : 0
  const mm = pad(38 - (i % 40))
  const ss = pad((i * 13) % 60)
  return {
    trade_no: `202607121${pad(4000 - i, 4)}${pad(i, 2)}`,
    out_trade_no: `SH${pad(20260712000 + i, 5)}`,
    api_trade_no: paid ? `${tm.plugin}${pad(88800 + i, 5)}` : '',
    uid: m.uid,
    domain: m.domain,
    name: goods[i % goods.length],
    money: moneyNum.toFixed(2),
    realmoney: paid ? moneyNum.toFixed(2) : null,
    getmoney: paid ? getmoneyNum.toFixed(2) : '0.00',
    refundmoney: refundNum.toFixed(2),
    profitmoney: status === 1 ? profitNum.toFixed(2) : '0.00',
    type: typeId,
    typename: tm.name,
    typeshowname: tm.showname,
    channel: tm.channel,
    plugin: tm.plugin,
    ip: `${100 + (i % 100)}.${(i * 7) % 255}.${(i * 3) % 255}.${i % 255}`,
    buyer: paid ? `user_${pad(1000 + i, 4)}@pay` : '',
    addtime: `2026-07-12 14:${mm}:${ss}`,
    endtime: paid ? `2026-07-12 14:${mm}:${pad(((i * 13) % 55) + 3)}` : null,
    status,
    settle: 0,
    combine: 0,
  }
})

/** 汇总统计（对齐 statistics 接口） */
export function calcStats(list: Order[]) {
  const num = (s: string) => parseFloat(s) || 0
  const totalMoney = list.reduce((a, o) => a + num(o.money), 0)
  const successList = list.filter((o) => o.status === 1)
  const unpaidList = list.filter((o) => o.status === 0)
  const refundList = list.filter((o) => o.status === 2)
  const successMoney = successList.reduce((a, o) => a + num(o.money), 0)
  const unpaidMoney = unpaidList.reduce((a, o) => a + num(o.money), 0)
  const refundMoney = refundList.reduce((a, o) => a + num(o.refundmoney), 0)
  const platformProfit = successList.reduce((a, o) => a + num(o.profitmoney), 0)
  const totalCount = list.length
  const successRate = totalCount ? +(((totalCount - unpaidList.length) / totalCount) * 100).toFixed(2) : 0
  return {
    totalMoney,
    successMoney,
    unpaidMoney,
    refundMoney,
    platformProfit,
    totalCount,
    successCount: successList.length,
    unpaidCount: unpaidList.length,
    refundCount: refundList.length,
    successRate,
  }
}
