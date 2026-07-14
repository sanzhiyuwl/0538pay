/**
 * 支付通道假数据。对齐 epay admin/pay_channel.php + ajax_pay.php?act=channelList（pre_channel）。
 * 一个通道 = 一个支付方式 + 一个支付插件的对接实例，含费率/限额/模式。
 */
import { payTypes } from './orders'

/** 支付通道（pre_channel） */
export interface Channel {
  id: number
  name: string // 显示名称
  type: number // 支付方式 ID
  typename: string // 支付方式英文名（图标）
  typeshowname: string // 支付方式中文名
  plugin: string // 支付插件标识
  mode: 0 | 1 // 0=平台代收 1=商户直清
  rate: string // 分成比例 %
  costrate: string // 通道成本 %
  daytop: number // 单日限额（0=无）
  paymin: string // 单笔最小
  paymax: string // 单笔最大
  today: string // 今日收款
  yesterday: string // 昨日收款
  status: 0 | 1 // 0=关闭 1=开启
}

/** 通道模式字典 */
export const channelMode: Record<number, string> = {
  0: '平台代收',
  1: '商户直清',
}

/** 支付方式筛选下拉 */
export const typeOptions = [
  { value: 0, label: '所有支付方式' },
  ...payTypes.map((t) => ({ value: t.id, label: t.showname })),
]

/** 各支付方式对应的插件候选（对齐 pre_plugin，用于新增/编辑联动） */
export const pluginsByType: Record<number, { name: string; showname: string }[]> = {
  1: [
    { name: 'alipay', showname: '支付宝当面付' },
    { name: 'alipayf2f', showname: '支付宝服务商' },
    { name: 'epusdt', showname: '支付宝个人码' },
  ],
  2: [
    { name: 'wxpaynp', showname: '微信原生支付' },
    { name: 'wxpayf2f', showname: '微信服务商' },
    { name: 'wxmini', showname: '微信小程序' },
  ],
  3: [{ name: 'qqpay', showname: 'QQ钱包官方' }],
  4: [
    { name: 'unionpay', showname: '云闪付官方' },
    { name: 'unionqr', showname: '银联二维码' },
  ],
}

const typeMeta: Record<number, { name: string; showname: string }> = {
  1: { name: 'alipay', showname: '支付宝' },
  2: { name: 'wxpay', showname: '微信支付' },
  3: { name: 'qqpay', showname: 'QQ钱包' },
  4: { name: 'bank', showname: '云闪付' },
}

// 通道名称样例
const channelNames = [
  '支付宝官方直连',
  '微信服务商A',
  'QQ钱包官方',
  '云闪付通道',
  '支付宝个人码',
  '微信小程序通道',
  '支付宝服务商B',
  '银联二维码',
  '微信原生备用',
  '云闪付备用线路',
]

function money(n: number) {
  return n.toFixed(2)
}

// 生成 10 个支付通道（ID 1-4 与 orders mock 对齐）
export const channels: Channel[] = channelNames.map((name, i) => {
  const typeId = ((i % 4) + 1) as 1 | 2 | 3 | 4
  const tm = typeMeta[typeId]
  const pluginList = pluginsByType[typeId]
  const plugin = pluginList[i % pluginList.length].name
  const mode = (i % 5 === 0 ? 1 : 0) as 0 | 1
  const rateNum = 0.38 + (i % 5) * 0.3
  const status = (i % 7 === 6 ? 0 : 1) as 0 | 1
  const todayNum = status === 1 ? +((i * 733.5) % 26000 + 500).toFixed(2) : 0
  const yestNum = status === 1 ? +((i * 941.2) % 31000 + 800).toFixed(2) : 0
  return {
    id: i + 1,
    name,
    type: typeId,
    typename: tm.name,
    typeshowname: tm.showname,
    plugin,
    mode,
    rate: rateNum.toFixed(2),
    costrate: (rateNum - 0.15).toFixed(2),
    daytop: mode === 1 ? 0 : [0, 0, 50000, 100000][i % 4],
    paymin: i % 3 === 0 ? '0.01' : '',
    paymax: i % 3 === 0 ? '5000.00' : '',
    today: money(todayNum),
    yesterday: money(yestNum),
    status,
  }
})

/** 汇总统计 */
export function calcChannelStats(list: Channel[]) {
  const num = (s: string) => parseFloat(s) || 0
  return {
    total: list.length,
    open: list.filter((c) => c.status === 1).length,
    closed: list.filter((c) => c.status === 0).length,
    todayTotal: list.reduce((a, c) => a + num(c.today), 0),
    yesterdayTotal: list.reduce((a, c) => a + num(c.yesterday), 0),
  }
}
