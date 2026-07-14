/**
 * 通道轮询假数据。对齐 epay admin/pay_roll.php + ajax_pay.php?act=getRoll（pre_roll）。
 * 轮询组 = 把多个同支付方式的通道编成一组，按策略（顺序/随机权重/首个启用）分流流量。
 */
import { channels } from './channels'
import { payTypes } from './orders'

/** 轮询组内的通道项（info JSON 的一项） */
export interface RollChannel {
  channel: number // 通道 ID
  channelname: string // 通道名（展示用）
  weight: number // 权重（1-99，仅随机轮询有效）
}

/** 轮询组（pre_roll） */
export interface Roll {
  id: number
  name: string // 显示名称
  type: number // 支付方式 ID
  typeshowname: string // 支付方式中文名
  kind: 0 | 1 | 2 // 0=顺序轮询 1=随机轮询 2=首个启用
  channels: RollChannel[] // 组内通道配置
  status: 0 | 1 // 0=关闭 1=开启
}

/** 轮询方式字典（对齐 pay_roll.php 的 $rolltype） */
export const rollKind: Record<number, string> = {
  0: '顺序轮询',
  1: '随机轮询',
  2: '首个启用',
}

/** 支付方式筛选下拉 */
export const typeOptions = [
  { value: 0, label: '所有支付方式' },
  ...payTypes.map((t) => ({ value: t.id, label: t.showname })),
]

// 按支付方式归组通道，供轮询组挑选
function channelsOfType(typeId: number) {
  return channels.filter((c) => c.type === typeId)
}

const typeMeta: Record<number, string> = {
  1: '支付宝',
  2: '微信支付',
  3: 'QQ钱包',
  4: '云闪付',
}

// 3 个轮询组样例
export const rolls: Roll[] = [
  {
    id: 1,
    name: '支付宝主力轮询',
    type: 1,
    typeshowname: typeMeta[1],
    kind: 1, // 随机权重
    channels: channelsOfType(1)
      .slice(0, 2)
      .map((c, i) => ({ channel: c.id, channelname: c.name, weight: [70, 30][i] ?? 10 })),
    status: 1,
  },
  {
    id: 2,
    name: '微信顺序备用',
    type: 2,
    typeshowname: typeMeta[2],
    kind: 0, // 顺序
    channels: channelsOfType(2)
      .slice(0, 2)
      .map((c) => ({ channel: c.id, channelname: c.name, weight: 1 })),
    status: 1,
  },
  {
    id: 3,
    name: '云闪付容灾组',
    type: 4,
    typeshowname: typeMeta[4],
    kind: 2, // 首个启用
    channels: channelsOfType(4)
      .slice(0, 2)
      .map((c) => ({ channel: c.id, channelname: c.name, weight: 1 })),
    status: 0,
  },
]

/** 轮询规则摘要（列表"轮询规则"列展示，对齐 info 字段） */
export function rollRuleText(roll: Roll): string {
  if (!roll.channels.length) return '未配置通道'
  if (roll.kind === 1) {
    return roll.channels.map((c) => `${c.channelname}(${c.weight})`).join(' / ')
  }
  return roll.channels.map((c) => c.channelname).join(' → ')
}
