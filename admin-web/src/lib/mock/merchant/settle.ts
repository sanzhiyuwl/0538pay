/**
 * 商户中心「结算记录」+「申请提现」假数据。
 * 对齐 epay user/settle.php（ajax2.php?act=settleList，pre_settle）与 user/apply.php。
 */

/** 结算方式：1支付宝 2微信 3QQ钱包 4银行卡 5支付机构 */
export const settleTypeMeta: Record<number, string> = {
  1: '支付宝',
  2: '微信',
  3: 'QQ钱包',
  4: '银行卡',
  5: '支付机构',
}

/** 结算状态：0待结算 1已完成 2正在结算 3结算失败 */
export const settleStatus: Record<number, { text: string; variant: 'warning' | 'success' | 'muted' | 'destructive' }> = {
  0: { text: '待结算', variant: 'warning' },
  1: { text: '已完成', variant: 'success' },
  2: { text: '正在结算', variant: 'muted' },
  3: { text: '结算失败', variant: 'destructive' },
}

export const statusOptions = [
  { value: -1, label: '全部状态' },
  { value: 0, label: '待结算' },
  { value: 1, label: '已完成' },
  { value: 2, label: '正在结算' },
  { value: 3, label: '结算失败' },
]

export interface SettleRecord {
  id: number
  type: number // 结算方式
  auto: boolean // 是否自动（否则显示 [手动]）
  account: string // 结算账号
  money: number // 结算金额
  realmoney: number // 实际到账
  addtime: string // 结算时间
  status: number
  failReason?: string // 失败原因
}

function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}

const fails = ['收款账号有误，请核对后重试', '银行系统维护，稍后自动重试', '单笔超限，请分批提现']

// 生成 30 条结算记录
export const settleRecords: SettleRecord[] = Array.from({ length: 30 }, (_, i) => {
  const type = (i % 4) + 1
  const r = i % 10
  const status = r < 6 ? 1 : r < 8 ? 0 : r === 8 ? 2 : 3
  const money = Math.round((100 + ((i * 137) % 4900)) * 100) / 100
  const rate = 0.005
  const fee = Math.min(Math.max(money * rate, 1), 25)
  const realmoney = Math.round((money - fee) * 100) / 100
  const day = 12 - (i % 9)
  const account =
    type === 1 ? `ali***${pad(i, 3)}@qq.com` : type === 2 ? `wx_openid_***${pad(i, 3)}` : type === 3 ? `QQ***${pad(i * 7, 4)}` : `6222***${pad(i * 13, 4)}`
  return {
    id: 50000 + (30 - i),
    type,
    auto: i % 3 !== 0,
    account,
    money,
    realmoney,
    addtime: `2026-07-${pad(day)} ${pad(2 + (i % 6))}:${pad((i * 11) % 60)}:${pad((i * 7) % 60)}`,
    status,
    failReason: status === 3 ? fails[i % fails.length] : undefined,
  }
})

/** 提现设置与账户（对齐 apply.php 展示/规则字段） */
export const applyInfo = {
  settleName: '支付宝', // 提现方式（只读）
  account: 'ali***538@qq.com', // 提现账号
  username: '张伟', // 真实姓名
  money: 12860.55, // 当前余额
  enableMoney: 9800.0, // 可提现余额（D+1 扣当日已收）
  settleMin: 100, // 最低提现额
  settleMaxLimit: 3, // 每日次数上限
  settleRate: 0.5, // 手续费率 %
  settleFeeMin: 1, // 最低手续费
  settleFeeMax: 25, // 最高手续费
  settleType: 2, // 1=D+0 2=D+1
  todayCount: 1, // 今日已提现次数
}
