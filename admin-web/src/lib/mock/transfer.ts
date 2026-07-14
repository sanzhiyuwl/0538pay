/**
 * 转账付款 / 付款记录假数据。对齐 epay admin/transfer_add.php（发起）+ transfer.php（记录）。
 * 数据表 pre_transfer：后台主动发起的对外转账（代付/提现打款）。
 */

/** 付款方式（对应 transfer_add.php 的 Tab） */
export interface TransferApp {
  key: 'alipay' | 'wxpay' | 'qqpay' | 'bank'
  label: string
  accountLabel: string // 收款账号字段名（随类型变）
  accountPlaceholder: string
  nameLabel: string
}

export const transferApps: TransferApp[] = [
  { key: 'alipay', label: '支付宝', accountLabel: '支付宝账号', accountPlaceholder: '支付宝登录账号 / UID / Openid', nameLabel: '支付宝姓名' },
  { key: 'wxpay', label: '微信', accountLabel: 'Openid', accountPlaceholder: '只能填写微信 Openid', nameLabel: '真实姓名' },
  { key: 'qqpay', label: 'QQ钱包', accountLabel: '收款方QQ', accountPlaceholder: '收款方 QQ 号码', nameLabel: '真实姓名' },
  { key: 'bank', label: '银行卡', accountLabel: '银行卡号', accountPlaceholder: '收款方银行卡号', nameLabel: '姓名' },
]

/** 各付款方式可选的转账通道（对齐 pre_channel where transtypes like app） */
export const transferChannels: Record<string, { id: number; name: string }[]> = {
  alipay: [
    { id: 1, name: '支付宝官方直连（默认）' },
    { id: 5, name: '支付宝个人码' },
  ],
  wxpay: [
    { id: 2, name: '微信服务商A（默认）' },
    { id: 9, name: '微信原生备用' },
  ],
  qqpay: [{ id: 3, name: 'QQ钱包官方（默认）' }],
  bank: [
    { id: 4, name: '云闪付通道（默认）' },
    { id: 8, name: '银联二维码' },
  ],
}

/** 付款记录（pre_transfer） */
export interface TransferRecord {
  biz_no: string // 交易号（19位）
  pay_order_no: string // 第三方交易号
  uid: number // 商户号（0=管理员）
  type: 'alipay' | 'wxpay' | 'qqpay' | 'bank'
  channel: number // 通道 ID
  account: string // 收款账号
  username: string // 收款姓名
  money: string // 付款金额
  costmoney: string // 花费金额
  desc: string // 备注
  addtime: string // 提交时间
  paytime: string | null // 付款时间
  status: 0 | 1 | 2 // 0=处理中 1=成功 2=失败
  result: string // 失败原因
}

/** 付款方式字典 */
export const transferTypeText: Record<string, string> = {
  alipay: '支付宝',
  wxpay: '微信',
  qqpay: 'QQ钱包',
  bank: '银行卡',
}

/** 状态字典 */
export const transferStatus: Record<
  number,
  { text: string; variant: 'warning' | 'success' | 'destructive' }
> = {
  0: { text: '正在处理', variant: 'warning' },
  1: { text: '转账成功', variant: 'success' },
  2: { text: '转账失败', variant: 'destructive' },
}

const names = ['张伟', '王芳', '李强', '刘洋', '陈静', '赵磊', '孙丽', '周涛']
const types: TransferRecord['type'][] = ['alipay', 'wxpay', 'qqpay', 'bank']
const descs = ['商户提现', '佣金结算', '推广返现', '保证金退款', '']

function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}
function maskAccount(type: string, seed: number): string {
  if (type === 'alipay') return `13${pad(seed % 100)}****${pad((seed * 7) % 10000, 4)}`
  if (type === 'wxpay') return `oXyZ_${pad(seed % 100)}****${pad(seed % 100)}`
  if (type === 'qqpay') return `${pad(10000 + seed * 3, 6)}`
  return `6222 **** **** ${pad((seed * 13) % 10000, 4)}`
}

// 生成 36 条付款记录
export const transferRecords: TransferRecord[] = Array.from({ length: 36 }, (_, i) => {
  const type = types[i % types.length]
  const uid = i % 4 === 0 ? 0 : 1000 + (i % 12) + 1
  const statusPool: TransferRecord['status'][] = [1, 1, 1, 0, 1, 2, 1, 1, 0]
  const status = statusPool[i % statusPool.length]
  const moneyNum = +(((i * 63.7) % 3000) + 10).toFixed(2)
  const cost = +(moneyNum + (type === 'bank' ? 1 : 0)).toFixed(2)
  const day = 12 - (i % 6)
  const submit = `2026-07-${pad(day)} ${pad(9 + (i % 10))}:${pad((i * 7) % 60)}:${pad((i * 13) % 60)}`
  const paid = status === 1
  return {
    biz_no: `${2026070000000000000 + i * 137}`.slice(0, 19),
    pay_order_no: status !== 0 ? `${type}${pad(200000 + i * 7, 6)}` : '',
    uid,
    type,
    channel: transferChannels[type][0].id,
    account: maskAccount(type, i + 5),
    username: names[i % names.length],
    money: moneyNum.toFixed(2),
    costmoney: cost.toFixed(2),
    desc: descs[i % descs.length],
    addtime: submit,
    paytime: paid ? `2026-07-${pad(day)} ${pad(9 + (i % 10))}:${pad(((i * 7) % 55) + 3)}:${pad((i * 13) % 60)}` : null,
    status,
    result: status === 2 ? '收款账号不存在或姓名不匹配' : '',
  }
})

/** 状态可执行操作（对齐 transfer.php 操作列） */
export function transferActions(status: number, uid: number): string[] {
  if (status === 1) return ['改为失败', '获取凭证', '复制', '删除']
  if (status === 2) return ['改为成功', '复制', '删除']
  // 处理中
  const base = ['查询状态', '复制']
  return uid > 0 ? ['查询状态', '退回', '复制'] : [...base, '删除']
}

/** 汇总统计 */
export function calcTransferStats(list: TransferRecord[]) {
  const num = (s: string) => parseFloat(s) || 0
  const successList = list.filter((r) => r.status === 1)
  return {
    total: list.length,
    totalMoney: list.reduce((a, r) => a + num(r.money), 0),
    successMoney: successList.reduce((a, r) => a + num(r.money), 0),
    successCount: successList.length,
    processingCount: list.filter((r) => r.status === 0).length,
    failCount: list.filter((r) => r.status === 2).length,
  }
}
