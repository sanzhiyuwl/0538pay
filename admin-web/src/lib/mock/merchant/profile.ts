/**
 * 商户中心「账户设置」假数据。对齐 epay user/editinfo.php（多板块，各自独立提交）。
 */

/** 收款账号设置（editSettle） */
export const settleConfig = {
  stype: '1', // 1支付宝 2微信 3QQ钱包 4银行卡
  account: 'ali***538@qq.com',
  username: '张伟',
}
export const stypeOptions = [
  { value: '1', label: '支付宝' },
  { value: '2', label: '微信（OpenId 或微信号）' },
  { value: '3', label: 'QQ钱包' },
  { value: '4', label: '银行卡' },
]

/** 联系方式设置（editInfo） */
export const contactConfig = {
  phone: '138****8888', // 脱敏
  email: 'merchant@example.com',
  qq: '800538',
  url: 'shop.abc.com',
  keylogin: true, // 开启密钥登录
  refund: false, // 订单退款 API 接口
  transfer: false, // 代付 API 接口
  remain_money: '0.00', // 预留余额（自动结算不参与）
}

/** 消息提醒接收设置（editMsgConfig，存 msgconfig） */
export const msgConfig = {
  notice_order: true, // 新订单通知
  notice_order_money: '0.00', // 通知订单金额大于
  notice_settle: true, // 结算通知
  notice_login: false, // 登录通知
  notice_balance: true, // 余额不足提醒
  notice_balance_money: '10.00', // 余额小于
  // 各通知渠道：1微信公众号 2邮件 3短信 4企业微信
  channel: '2',
}
export const noticeChannelOptions = [
  { value: '1', label: '微信公众号' },
  { value: '2', label: '邮件' },
  { value: '3', label: '短信' },
  { value: '4', label: '企业微信' },
]

/** 支付手续费扣除模式（editMode） */
export const modeConfig = {
  mode: '0', // 0=余额扣费（经典） 1=订单加费（买家承担）
}
export const modeOptions = [
  { value: '0', label: '余额扣费（从商户余额扣手续费，经典模式）' },
  { value: '1', label: '订单加费（手续费加到订单金额，买家承担）' },
]

/** 第三方账号绑定状态 */
export const bindConfig = {
  qq: { bound: true, nick: '张伟' },
  wx: { bound: false, nick: '' },
  alipay: { bound: true, nick: '张*伟' },
}
