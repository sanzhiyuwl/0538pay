/**
 * 支付方式假数据。对齐 epay admin/pay_type.php（pre_type）。
 * 支付方式 = 前端可选的收款方式（支付宝/微信…），关联多个支付通道。
 */

/** 支付方式（pre_type） */
export interface PayType {
  id: number
  name: string // 调用值（英文，与支付文档一致）
  showname: string // 显示名称
  device: 0 | 1 | 2 // 0=PC+Mobile 1=PC 2=Mobile
  today: string // 今日收款
  status: 0 | 1 // 0=关闭 1=开启
}

/** 支持设备字典 */
export const deviceText: Record<number, string> = {
  0: 'PC + Mobile',
  1: 'PC',
  2: 'Mobile',
}

export const deviceOptions = [
  { value: 0, label: 'PC + Mobile' },
  { value: 1, label: 'PC' },
  { value: 2, label: 'Mobile' },
]

/** 系统自带的支付方式（id < 4 不可删），后续为自定义 */
export const payTypeList: PayType[] = [
  { id: 1, name: 'alipay', showname: '支付宝', device: 0, today: '12680.50', status: 1 },
  { id: 2, name: 'wxpay', showname: '微信支付', device: 0, today: '9842.00', status: 1 },
  { id: 3, name: 'qqpay', showname: 'QQ钱包', device: 0, today: '0.00', status: 0 },
  { id: 4, name: 'bank', showname: '云闪付', device: 0, today: '3125.80', status: 1 },
  { id: 5, name: 'jdpay', showname: '京东支付', device: 2, today: '860.00', status: 1 },
  { id: 6, name: 'douyinpay', showname: '抖音支付', device: 2, today: '540.20', status: 1 },
  { id: 7, name: 'usdt', showname: 'USDT', device: 0, today: '0.00', status: 0 },
]

/** 汇总统计 */
export function calcPayTypeStats(list: PayType[]) {
  const num = (s: string) => parseFloat(s) || 0
  return {
    total: list.length,
    open: list.filter((t) => t.status === 1).length,
    closed: list.filter((t) => t.status === 0).length,
    todayTotal: list.reduce((a, t) => a + num(t.today), 0),
  }
}
