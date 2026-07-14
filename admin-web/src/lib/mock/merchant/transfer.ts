/**
 * 商户中心「代付管理」假数据。对齐 epay user/transfer.php（transferList，pre_transfer）
 * 与 user/transfer_add.php（发起代付）。需平台开启 user_transfer。
 */

/** 付款方式 */
export const transferApps = [
  { key: 'alipay', label: '支付宝', accountLabel: '支付宝账号' },
  { key: 'wxpay', label: '微信', accountLabel: '微信 OpenId' },
  { key: 'qqpay', label: 'QQ钱包', accountLabel: 'QQ 号码' },
  { key: 'bank', label: '银行卡', accountLabel: '银行卡号' },
]

/** 付款状态：0处理中 1成功 2失败 */
export const transferStatus: Record<number, { text: string; variant: 'muted' | 'success' | 'destructive' }> = {
  0: { text: '处理中', variant: 'muted' },
  1: { text: '转账成功', variant: 'success' },
  2: { text: '转账失败', variant: 'destructive' },
}
export const statusOptions = [
  { value: -1, label: '全部状态' },
  { value: 0, label: '处理中' },
  { value: 1, label: '转账成功' },
  { value: 2, label: '转账失败' },
]
export const typeOptions = [
  { value: 'all', label: '全部方式' },
  { value: 'alipay', label: '支付宝' },
  { value: 'wxpay', label: '微信' },
  { value: 'qqpay', label: 'QQ钱包' },
  { value: 'bank', label: '银行卡' },
]

export interface TransferRecord {
  id: number
  biz_no: string // 交易号
  pay_order_no: string // 第三方交易号
  type: string // 付款方式 key
  typeLabel: string
  desc: string // 备注
  account: string // 付款账号
  username: string // 收款姓名
  money: number // 付款金额
  costmoney: number // 花费金额（含手续费）
  addtime: string // 提交时间
  paytime: string | null // 付款时间
  status: number
  failReason?: string
}

function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}
const names = ['王芳', '李强', '赵敏', '陈刚', '刘洋', '孙丽']
const descs = ['货款结算', '推广佣金', '退款补偿', '服务费', '', '分润']
const fails = ['收款账号不存在', '姓名与账号不匹配', '对方账户异常']

// 生成 26 条代付记录
export const transferRecords: TransferRecord[] = Array.from({ length: 26 }, (_, i) => {
  const app = transferApps[i % 4]
  const r = i % 8
  const status = r < 5 ? 1 : r < 7 ? 0 : 2
  const money = Math.round((50 + ((i * 91) % 3000)) * 100) / 100
  const cost = Math.round((money + Math.min(Math.max(money * 0.001, 0.1), 5)) * 100) / 100
  const day = 12 - (i % 8)
  const account =
    app.key === 'alipay' ? `ali***${pad(i, 3)}@qq.com` : app.key === 'wxpay' ? `wx_openid_***${pad(i, 3)}` : app.key === 'qqpay' ? `QQ***${pad(i * 7, 4)}` : `6222***${pad(i * 13, 4)}`
  return {
    id: 40000 + (26 - i),
    biz_no: `T${pad(20260700000 + i * 137, 13)}`,
    pay_order_no: status === 1 ? `PAY${pad(i * 991, 10)}` : '',
    type: app.key,
    typeLabel: app.label,
    desc: descs[i % descs.length],
    account,
    username: names[i % names.length],
    money,
    costmoney: cost,
    addtime: `2026-07-${pad(day)} ${pad(9 + (i % 10))}:${pad((i * 11) % 60)}:${pad((i * 7) % 60)}`,
    paytime: status === 1 ? `2026-07-${pad(day)} ${pad(9 + (i % 10))}:${pad(((i * 11) % 58) + 1)}:${pad((i * 13) % 60)}` : null,
    status,
    failReason: status === 2 ? fails[i % fails.length] : undefined,
  }
})

/** 发起代付：账户与规则（对齐 transfer_add.php） */
export const transferSetup = {
  balance: 12860.55, // 可转账余额
  rate: 0.1, // 手续费率 %
  minMoney: 1, // 单笔最小
  maxMoney: 20000, // 单笔最大
  maxLimit: 10, // 单账号每日次数上限
}
