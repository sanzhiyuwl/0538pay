/**
 * 结算假数据。对齐 epay admin 结算业务：
 * - pre_batch（结算批次）：settle.php 顶部批次列表
 * - pre_settle（结算明细）：slist.php / ajax_settle.php?act=settleList 的行结构
 */

/** 结算批次（pre_batch） */
export interface SettleBatch {
  batch: string // 批次号
  allmoney: string // 批次总金额
  count: number // 批次总数量
  time: string // 生成时间
  status: 0 | 1 | 2 // 0=处理中 1=已完成 2=部分完成
}

/** 结算明细（pre_settle） */
export interface SettleRecord {
  id: number
  batch: string // 所属批次号
  uid: number // 商户号
  merchant: string // 商户名称
  type: 1 | 2 | 3 | 4 | 5 // 结算方式
  auto: 0 | 1 // 是否自动（0=手动）
  account: string // 结算账号
  username: string // 结算姓名
  money: string // 结算金额
  realmoney: string // 实际到账
  addtime: string // 创建时间
  endtime: string | null // 完成时间
  status: 0 | 1 | 2 | 3 // 0=待结算 1=已完成 2=正在结算 3=结算失败
  result: string // 失败原因
}

/** 结算方式字典（对齐 slist.php） */
export const settleTypes: Record<
  number,
  { name: string; showname: string }
> = {
  1: { name: 'alipay', showname: '支付宝' },
  2: { name: 'wxpay', showname: '微信' },
  3: { name: 'qqpay', showname: 'QQ钱包' },
  4: { name: 'bank', showname: '银行卡' },
  5: { name: 'org', showname: '支付机构' },
}

/** 结算状态字典 */
export const settleStatus: Record<
  number,
  { text: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'muted' }
> = {
  0: { text: '待结算', variant: 'default' },
  1: { text: '已完成', variant: 'success' },
  2: { text: '正在结算', variant: 'warning' },
  3: { text: '结算失败', variant: 'destructive' },
}

/** 批次状态字典 */
export const batchStatus: Record<
  number,
  { text: string; variant: 'default' | 'success' | 'warning' | 'muted' }
> = {
  0: { text: '处理中', variant: 'warning' },
  1: { text: '已完成', variant: 'success' },
  2: { text: '部分完成', variant: 'muted' },
}

const merchants = [
  { uid: 1001, name: '云上便利店' },
  { uid: 1002, name: '极客数码' },
  { uid: 1003, name: '花间集鲜花' },
  { uid: 1005, name: '悦读书屋' },
  { uid: 1008, name: '快充能源' },
  { uid: 1012, name: '优选生活馆' },
]
const names = ['张伟', '王芳', '李强', '刘洋', '陈静', '赵磊', '孙丽', '周涛']

function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}

function maskAccount(type: number, seed: number): string {
  if (type === 1) return `13${pad(seed % 100)}****${pad((seed * 7) % 10000, 4)}` // 支付宝手机
  if (type === 2) return `wxid_${pad(seed % 100)}****${pad(seed % 100)}` // 微信
  if (type === 3) return `${pad(10000 + seed * 3, 5)}****` // QQ
  if (type === 4) return `6222 **** **** ${pad((seed * 13) % 10000, 4)}` // 银行卡
  return `ORG${pad(seed, 4)}` // 支付机构
}

// 生成 4 个批次
export const batches: SettleBatch[] = Array.from({ length: 4 }, (_, i) => {
  const day = 12 - i
  return {
    batch: `B2026070${pad(day)}${pad(1000 + i * 7, 4)}`,
    allmoney: '0.00', // 汇总后回填
    count: 0,
    time: `2026-07-${pad(day)} 02:00:0${i}`,
    status: (i === 0 ? 0 : i === 3 ? 2 : 1) as SettleBatch['status'],
  }
})

// 生成 48 条结算明细，均匀挂到各批次
export const settleRecords: SettleRecord[] = Array.from({ length: 48 }, (_, i) => {
  const m = merchants[i % merchants.length]
  const type = (((i % 5) + 1) as SettleRecord['type'])
  const batch = batches[i % batches.length]
  // 批次0=处理中→记录全为正在结算（入批次即从"待结算"转为"正在结算"）
  // 批次3=部分完成→含结算失败；其余批次=已完成
  let status: SettleRecord['status']
  const bi = i % batches.length
  if (bi === 0) status = 2
  else if (bi === 3) status = (i % 4 === 0 ? 3 : 1)
  else status = 1
  const moneyNum = +(((i * 53.7) % 4800) + 120).toFixed(2)
  const fee = +(moneyNum * 0.001).toFixed(2) // 结算手续费
  const realNum = +(moneyNum - fee).toFixed(2)
  const auto = (i % 5 === 4 ? 0 : 1) as 0 | 1 // 支付机构那档标手动
  const done = status === 1
  const mm = pad((i * 11) % 60)
  return {
    id: 10000 + (48 - i),
    batch: batch.batch,
    uid: m.uid,
    merchant: m.name,
    type,
    auto,
    account: maskAccount(type, i + 3),
    username: names[i % names.length],
    money: moneyNum.toFixed(2),
    realmoney: realNum.toFixed(2),
    addtime: batch.time,
    endtime: done ? `2026-07-${pad(12 - bi)} 09:${mm}:0${i % 10}` : null,
    status,
    result: status === 3 ? '收款账号有误，转账被退回' : '',
  }
})

// 回填批次的总金额/总数量
for (const b of batches) {
  const rows = settleRecords.filter((r) => r.batch === b.batch)
  b.count = rows.length
  b.allmoney = rows.reduce((a, r) => a + parseFloat(r.money), 0).toFixed(2)
}

/** 明细汇总统计 */
export function calcSettleStats(list: SettleRecord[]) {
  const num = (s: string) => parseFloat(s) || 0
  const totalMoney = list.reduce((a, r) => a + num(r.money), 0)
  const realMoney = list.reduce((a, r) => a + num(r.realmoney), 0)
  const doneList = list.filter((r) => r.status === 1)
  const pendingList = list.filter((r) => r.status === 0)
  const processingList = list.filter((r) => r.status === 2)
  const failList = list.filter((r) => r.status === 3)
  return {
    totalMoney,
    realMoney,
    doneMoney: doneList.reduce((a, r) => a + num(r.money), 0),
    totalCount: list.length,
    doneCount: doneList.length,
    pendingCount: pendingList.length,
    processingCount: processingList.length,
    failCount: failList.length,
  }
}
