/**
 * 商户假数据。字段对齐 epay admin/ajax_user.php?act=userList 的真实返回（pre_user + groupname）。
 */

export interface Merchant {
  uid: number // 商户号
  gid: number // 用户组ID
  groupname: string // 用户组名
  endtime: string | null // 用户组到期时间
  money: string // 余额
  settle_id: number // 结算方式 1支付宝2微信3QQ4银行卡
  account: string // 结算账号
  username: string // 结算姓名
  qq: string
  phone: string
  email: string
  url: string // 域名
  addtime: string // 添加时间
  status: 0 | 1 | 2 // 0封禁 1正常 2未审核
  cert: 0 | 1 // 实名
  pay: 0 | 1 | 2 // 支付权限 0关1开2未审核
  settle: 0 | 1 // 结算权限
  upid: number // 邀请方
  mode: 0 | 1 // 手续费模式
  deposit: string // 保证金
}

/** 结算方式字典 */
export const settleTypes: Record<number, { label: string; prefix: string }> = {
  1: { label: '支付宝', prefix: '' },
  2: { label: '微信', prefix: 'WX:' },
  3: { label: 'QQ钱包', prefix: 'QQ:' },
  4: { label: '银行卡', prefix: '' },
}

/** 用户组 */
export const groups = [
  { gid: 0, name: '默认用户组' },
  { gid: 1, name: '普通商户' },
  { gid: 2, name: 'VIP商户' },
  { gid: 3, name: '企业商户' },
]

/** 搜索字段下拉（对齐 ulist column） */
export const searchColumns = [
  { value: 'uid', label: '商户号' },
  { value: 'account', label: '结算账号' },
  { value: 'username', label: '结算姓名' },
  { value: 'url', label: '域名' },
  { value: 'qq', label: 'QQ' },
  { value: 'phone', label: '手机号码' },
  { value: 'email', label: '邮箱' },
]

/** 状态筛选下拉（对齐 dstatus 字段_值） */
export const statusFilters = [
  { value: '0', label: '全部用户' },
  { value: 'pay_2', label: '待审核用户' },
  { value: 'status_1', label: '用户状态正常' },
  { value: 'status_0', label: '用户状态封禁' },
  { value: 'pay_1', label: '支付状态正常' },
  { value: 'pay_0', label: '支付状态关闭' },
  { value: 'settle_1', label: '结算状态正常' },
  { value: 'settle_0', label: '结算状态关闭' },
]

const names = ['云上便利店', '极客数码', '花间集鲜花', '悦读书屋', '快充能源', '优选生活馆', '星辰科技', '一点通商贸', '海角零食', '云帆网络']
const domains = ['shop.example.com', 'geek.example.com', 'flower.example.com', 'book.example.com', 'charge.example.com', 'life.example.com', 'star.example.com', 'ydt.example.com', 'snack.example.com', 'yunfan.example.com']

function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}

export const merchants: Merchant[] = Array.from({ length: 48 }, (_, i) => {
  const gid = [0, 1, 1, 2, 1, 3, 0, 2][i % 8]
  const statusPool: Merchant['status'][] = [1, 1, 1, 1, 2, 1, 0, 1]
  const status = statusPool[i % statusPool.length]
  const payPool: Merchant['pay'][] = [1, 1, 2, 1, 1, 0, 1, 1]
  const settleId = ((i % 4) + 1) as 1 | 2 | 3 | 4
  const hasAccount = i % 5 !== 0
  return {
    uid: 1001 + i,
    gid,
    groupname: groups.find((g) => g.gid === gid)?.name ?? '默认用户组',
    endtime: gid > 0 && i % 3 === 0 ? '2026-12-31' : null,
    money: (((i * 137.4) % 8000) + 12).toFixed(2),
    settle_id: settleId,
    account: hasAccount ? (settleId === 4 ? `6222${pad(i, 12)}` : `user_${pad(1000 + i, 4)}@pay`) : '',
    username: hasAccount ? names[i % names.length] : '',
    qq: i % 2 === 0 ? `${100000 + i * 137}` : '',
    phone: `13${pad(800000000 + i * 12345, 9)}`,
    email: `m${1001 + i}@example.com`,
    url: domains[i % domains.length],
    addtime: `2026-0${(i % 6) + 1}-${pad((i % 27) + 1)} 09:${pad(i % 60)}:00`,
    status,
    cert: (i % 3 === 0 ? 1 : 0) as 0 | 1,
    pay: payPool[i % payPool.length],
    settle: (i % 4 === 0 ? 0 : 1) as 0 | 1,
    upid: i % 6 === 0 ? 1001 + ((i + 3) % 40) : 0,
    mode: (i % 3 === 0 ? 1 : 0) as 0 | 1,
    deposit: i % 4 === 0 ? (500).toFixed(2) : '0.00',
  }
})
