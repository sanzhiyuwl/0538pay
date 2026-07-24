/**
 * 商户中心「购买会员」+「邀请返现」假数据。
 * 对齐 epay user/groupbuy.php（pre_group isbuy=1）与 user/invite.php。
 */

/** 会员套餐（用户组） */
export interface GroupPlan {
  id: number
  name: string
  rates: { label: string; rate: string }[] // 各通道费率
  price: number // 售价（按月或永久）
  expire: number // 有效期月数，0=永久
  recommended?: boolean
}

export const groupPlans: GroupPlan[] = [
  {
    id: 2,
    name: '白银会员',
    rates: [
      { label: '支付宝', rate: '0.35' },
      { label: '微信', rate: '0.35' },
      { label: 'QQ钱包', rate: '0.55' },
    ],
    price: 30,
    expire: 1,
  },
  {
    id: 3,
    name: '黄金会员',
    rates: [
      { label: '支付宝', rate: '0.30' },
      { label: '微信', rate: '0.30' },
      { label: 'QQ钱包', rate: '0.50' },
    ],
    price: 88,
    expire: 1,
    recommended: true,
  },
  {
    id: 4,
    name: '钻石会员（永久）',
    rates: [
      { label: '支付宝', rate: '0.28' },
      { label: '微信', rate: '0.28' },
      { label: 'QQ钱包', rate: '0.45' },
    ],
    price: 1288,
    expire: 0,
  },
]

/** 当前会员状态 */
export const currentGroup = {
  name: '普通会员',
  expire: '—', // 到期时间
}

/** 支付方式（购买时） */
export const buyPayOptions = [
  { value: '0', label: '余额支付' },
  { value: 'alipay', label: '支付宝' },
  { value: 'wxpay', label: '微信支付' },
]

// ============ 邀请返现 ============
export const inviteInfo = {
  link: 'https://epvia.com/?invite=A1B2C3',
  rate: 20, // 返现比例 %
  orderType: 1, // 1=按手续费 0=按订单额
}
export const inviteStat = {
  users: 12, // 已邀请数
  incomeToday: 8.6, // 今日返现
  incomeYesterday: 15.2, // 昨日返现
  incomeTotal: 386.5, // 累计返现
}

/** 已邀请用户 */
export interface InvitedUser {
  uid: number
  addtime: string
}
function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}
export const invitedUsers: InvitedUser[] = Array.from({ length: 12 }, (_, i) => ({
  uid: 1200 + i * 3 + 1,
  addtime: `2026-07-${pad(12 - (i % 10))} ${pad(9 + (i % 10))}:${pad((i * 11) % 60)}:${pad((i * 7) % 60)}`,
}))
