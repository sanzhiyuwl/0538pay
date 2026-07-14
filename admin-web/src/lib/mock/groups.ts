/**
 * 用户组 / 套餐假数据。对齐 epay admin/glist.php + gedit.php + group.php（pre_group）。
 * 一个用户组 = 一个 SaaS 套餐：包含通道费率(info)、能力配置(config)、购买设置(isbuy/price/expire)。
 */
import { merchants } from './merchants'

/** 单个支付方式的通道费率（pre_group.info 的一项） */
export interface ChannelRate {
  typename: string // 支付方式名（支付宝/微信…）
  channel: string // 通道名 / 随机可用 / 关闭
  rate: string // 分成比例 %
}

/** 套餐能力配置（pre_group.config 精简版，SaaS 差异化的核心开关） */
export interface GroupConfig {
  settleOpen: string // 结算开关
  settleType: string // 结算周期 D+0 / D+1
  settleRate: string // 结算手续费 %
  recharge: boolean // 余额充值
  userTransfer: boolean // 代付功能
  inviteOpen: boolean // 邀请返现
  userDeposit: boolean // 商户保证金
}

/** 用户组 / 套餐（pre_group） */
export interface Group {
  gid: number
  name: string // 套餐名称
  isbuy: 0 | 1 // 是否上架可购买
  price: string // 售价
  expire: number // 有效期(月) 0=永久
  sort: number // 排序
  visible: string // 可见范围（GID，空=全部）
  rates: ChannelRate[] // 通道与费率
  config: GroupConfig // 能力配置
  merchantCount: number // 该组下商户数（回填）
}

/** 结算开关文案 */
export const settleOpenText: Record<string, string> = {
  '1': '仅自动结算',
  '2': '仅手动申请',
  '3': '自动+手动',
}

/** 全局购买开关（对应 group.php 的 group_buy 设置） */
export const groupBuyEnabled = true

// 4 个套餐，GID 与 merchants.ts 的 groups 对齐（0默认/1普通/2VIP/3企业）
export const groups: Group[] = [
  {
    gid: 0,
    name: '默认用户组',
    isbuy: 0,
    price: '0.00',
    expire: 0,
    sort: 99,
    visible: '',
    rates: [
      { typename: '支付宝', channel: '默认通道', rate: '2.0' },
      { typename: '微信支付', channel: '默认通道', rate: '2.0' },
    ],
    config: {
      settleOpen: '1',
      settleType: '2',
      settleRate: '0.1',
      recharge: false,
      userTransfer: false,
      inviteOpen: false,
      userDeposit: false,
    },
    merchantCount: 0,
  },
  {
    gid: 1,
    name: '普通商户',
    isbuy: 1,
    price: '99.00',
    expire: 12,
    sort: 3,
    visible: '',
    rates: [
      { typename: '支付宝', channel: '随机可用通道', rate: '1.8' },
      { typename: '微信支付', channel: '随机可用通道', rate: '1.8' },
      { typename: 'QQ钱包', channel: '关闭', rate: '0' },
    ],
    config: {
      settleOpen: '3',
      settleType: '2',
      settleRate: '0.1',
      recharge: true,
      userTransfer: false,
      inviteOpen: true,
      userDeposit: false,
    },
    merchantCount: 0,
  },
  {
    gid: 2,
    name: 'VIP商户',
    isbuy: 1,
    price: '499.00',
    expire: 12,
    sort: 2,
    visible: '',
    rates: [
      { typename: '支付宝', channel: '随机可用通道', rate: '1.2' },
      { typename: '微信支付', channel: '随机可用通道', rate: '1.2' },
      { typename: 'QQ钱包', channel: '随机可用通道', rate: '1.5' },
      { typename: '云闪付', channel: '随机可用通道', rate: '0.8' },
    ],
    config: {
      settleOpen: '3',
      settleType: '1',
      settleRate: '0.05',
      recharge: true,
      userTransfer: true,
      inviteOpen: true,
      userDeposit: false,
    },
    merchantCount: 0,
  },
  {
    gid: 3,
    name: '企业商户',
    isbuy: 1,
    price: '1999.00',
    expire: 0,
    sort: 1,
    visible: '2,3',
    rates: [
      { typename: '支付宝', channel: '企业专属通道', rate: '0.6' },
      { typename: '微信支付', channel: '企业专属通道', rate: '0.6' },
      { typename: 'QQ钱包', channel: '随机可用通道', rate: '1.0' },
      { typename: '云闪付', channel: '随机可用通道', rate: '0.55' },
    ],
    config: {
      settleOpen: '3',
      settleType: '1',
      settleRate: '0.00',
      recharge: true,
      userTransfer: true,
      inviteOpen: true,
      userDeposit: true,
    },
    merchantCount: 0,
  },
]

// 回填每个套餐的商户数
for (const g of groups) {
  g.merchantCount = merchants.filter((m) => m.gid === g.gid).length
}

/** 套餐能力开关清单（用于卡片统一渲染 label + 是否开启） */
export function capabilityList(c: GroupConfig) {
  return [
    { label: '余额充值', on: c.recharge },
    { label: '代付功能', on: c.userTransfer },
    { label: '邀请返现', on: c.inviteOpen },
    { label: '商户保证金', on: c.userDeposit },
  ]
}
