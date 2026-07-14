/**
 * 企业微信 + H5 微信客服支付假数据。
 * 对齐 epay admin/pay_wework.php（pre_wework 企业微信账号）与
 * admin/set_wxkf.php（H5 跳转微信客服支付配置，pre_config + pre_wxkfaccount）。
 */

/** 企业微信账号（pre_wework） */
export interface WeworkAccount {
  id: number
  name: string // 显示名称
  appid: string // 企业ID
  appsecret: string // Secret（脱敏）
  kfnum: number // 该企业下客服账号数量
  status: 0 | 1 // 1开启 0关闭
}

export const weworkAccounts: WeworkAccount[] = [
  { id: 1, name: '主收款企业微信', appid: 'ww8a1b2c3d4e5f6a', appsecret: 'Xy9k****QpL2', kfnum: 3, status: 1 },
  { id: 2, name: '备用企业微信A', appid: 'ww7f6e5d4c3b2a1', appsecret: 'Mn3p****Rt8w', kfnum: 2, status: 1 },
  { id: 3, name: '备用企业微信B', appid: 'ww1a2b3c4d5e6f7', appsecret: 'Qz5v****Ab6c', kfnum: 0, status: 0 },
]

export function calcWeworkStats(list: WeworkAccount[]) {
  return {
    total: list.length,
    active: list.filter((w) => w.status === 1).length,
    kfTotal: list.reduce((s, w) => s + w.kfnum, 0),
  }
}

/** H5 微信客服支付配置（set_wxkf → pre_config） */
export const wxkfConfig = {
  callbackUrl: 'https://0538pay.com/wework.php', // 只读回调地址
  wework_token: '',
  wework_aeskey: '',
  wework_payopen: '0', // 0关 1开
  wework_paymsgmode: '0', // 0确认消息后发链接 1直接发链接
  wework_paykfid: '0', // 客服账号ID，0=多客服轮询
  wework_contact: '', // 人工客服链接
  wework_remark: '', // 支付消息尾部附加内容
}

export const paymsgModeOptions = [
  { value: '0', label: '发送确认消息，用户回复后发链接（默认）' },
  { value: '1', label: '直接发送支付链接（用户无法支付第二单）' },
]

/** 客服账号选项（对齐 wxkfaccount：0=多客服轮询 + 各账号） */
export const kfAccountOptions = [
  { value: '0', label: '多客服账号轮询' },
  { value: '1', label: 'wkf_001 - 客服小艾' },
  { value: '2', label: 'wkf_002 - 客服小美' },
  { value: '3', label: 'wkf_003 - 客服小易' },
]
