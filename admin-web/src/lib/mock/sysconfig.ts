/**
 * 系统设置补充配置假数据。对齐 epay admin/set.php 剩余 mod：
 * oauth 快捷登录 / notice 消息提醒 / certificate 实名认证 / template 首页模板 /
 * mail 邮箱与短信 / cron 计划任务 / proxy 代理 / account 账号 / iptype IP 获取。
 * 字段名与 pre_config 保持一致，值为原型示例。
 */

// ============ oauth 快捷登录 ============
export const oauthConfig = {
  login_qq: '0', // 0关 1QQ互联 2手机QQ扫码 3彩虹聚合
  login_qq_appid: '',
  login_qq_appkey: '',
  login_alipay: '0', // 0关 / 通道ID / -1彩虹聚合
  login_wx: '0', // 0关 / 公众号ID / -1彩虹聚合
  login_apiurl: '', // 聚合登录 API 地址
  login_appid: '',
  login_appkey: '',
}
export const qqLoginOptions = [
  { value: '0', label: '关闭' },
  { value: '1', label: 'QQ互联官方快捷登录' },
  { value: '2', label: '手机QQ扫码登录' },
  { value: '3', label: '彩虹聚合登录' },
]
export const alipayLoginOptions = [
  { value: '0', label: '关闭' },
  { value: '1', label: '支付宝主通道（alipay #1）' },
  { value: '-1', label: '彩虹聚合登录' },
]
export const wxLoginOptions = [
  { value: '0', label: '关闭' },
  { value: '1', label: '默认服务号（wxpay #1）' },
  { value: '-1', label: '彩虹聚合登录' },
]

// ============ notice 消息提醒 ============
export const noticeConfig = {
  // 微信公众号消息
  wxnotice: '0', // 公众号消息开关
  wxnotice_tpl_order: '', // 新订单通知模板ID
  wxnotice_tpl_settle: '', // 结算通知模板ID
  wxnotice_tpl_login: '', // 登录通知模板ID
  wxnotice_tpl_balance: '', // 余额不足提醒模板ID
  // 管理员邮件开关
  msgconfig_regaudit: '0', // 新注册商户待审核提醒
  msgconfig_apply: '0', // 商户手动提现提醒
  msgconfig_domain: '0', // 授权域名待审核提醒
  // 用户邮件开关
  mailnotice: '0', // 邮件消息总开关
  msgconfig_order: '0', // 新订单通知
  msgconfig_settle: '0', // 结算通知
  msgconfig_balance: '0', // 余额不足提醒
}

// ============ certificate 实名认证 ============
export const certConfig = {
  cert_open: '0', // 0关 1支付宝身份验证 3支付宝实名信息 5阿里云金融级 4微信扫码 2手机三要素
  cert_channel: '0', // 支付宝通道
  cert_appcode: '', // 手机三要素 APPCODE
  cert_qcloudid: '', // 腾讯云 SecretId
  cert_qcloudkey: '', // 腾讯云 SecretKey
  cert_aliyunid: '', // 阿里云 AccessKeyId
  cert_aliyunkey: '', // 阿里云 AccessKeySecret
  cert_aliyunsceneid: '', // 阿里云认证场景ID
  cert_corpopen: '0', // 企业认证开关
  cert_appcode2: '', // 企业信息校验 APPCODE
  cert_force: '0', // 商户强制认证
}
export const certOpenOptions = [
  { value: '0', label: '关闭' },
  { value: '1', label: '支付宝身份验证' },
  { value: '3', label: '支付宝实名信息验证' },
  { value: '5', label: '阿里云金融级实人认证' },
  { value: '4', label: '微信扫码实名认证' },
  { value: '2', label: '手机号三要素实名认证' },
]
export const certChannelOptions = [
  { value: '0', label: '关闭' },
  { value: '1', label: '支付宝主通道（alipay #1）' },
]
export const switchOptions = [
  { value: '0', label: '关闭' },
  { value: '1', label: '开启' },
]

// ============ template 首页模板 ============
export interface TemplateItem {
  name: string
  label: string
}
// epay template 目录：default + index1~10（\lib\Template::getList()）
export const templates: TemplateItem[] = [
  { name: 'default', label: 'default（SaaS 官网风）' },
  ...Array.from({ length: 10 }, (_, i) => ({
    name: `index${i + 1}`,
    label: `index${i + 1}`,
  })),
]
export const currentTemplate = 'default'

// ============ mail 邮箱与短信 ============
export const mailConfig = {
  mail_cloud: '0', // 0 SMTP / 1 SendCloud / 2 阿里云邮件推送
  mail_smtp: 'smtp.qq.com',
  mail_port: '465',
  mail_name: '',
  mail_pwd: '',
  mail_apiuser: '',
  mail_apikey: '',
  mail_name2: '',
  mail_recv: '',
}
export const mailCloudOptions = [
  { value: '0', label: 'SMTP 发信' },
  { value: '1', label: '搜狐 SendCloud' },
  { value: '2', label: '阿里云邮件推送' },
]
export const smsConfig = {
  sms_api: '0', // 0企信通 1腾讯云 2阿里云 3ThinkAPI 4短信宝
  sms_appid: '',
  sms_appkey: '',
  sms_sign: '',
  sms_tpl_reg: '', // 商户注册模板ID
  sms_tpl_find: '', // 找回密码模板ID
  sms_tpl_edit: '', // 修改结算账号模板ID
  sms_tpl_balance: '', // 余额不足通知模板ID
}
export const smsApiOptions = [
  { value: '0', label: '企信通短信接口' },
  { value: '1', label: '腾讯云短信接口' },
  { value: '2', label: '阿里云短信接口' },
  { value: '3', label: 'ThinkAPI 短信接口' },
  { value: '4', label: '短信宝短信接口' },
]

// ============ cron 计划任务 ============
export const cronConfig = {
  cronkey: 'cron_0538pay_8f3a2c', // 计划任务访问密钥
}
export interface CronTask {
  title: string
  desc: string
  path: string // cron.php?do=xxx
}
export const cronTasks: CronTask[] = [
  { title: '订单统计任务', desc: '0 点后访问一次即可', path: 'cron.php?do=order' },
  { title: '自动生成结算任务', desc: '0 点后访问一次即可', path: 'cron.php?do=settle' },
  { title: '订单异步通知重试任务', desc: '通知失败时自动重试（1分钟/3分钟/20分钟/1小时/2小时）', path: 'cron.php?do=notify' },
  { title: '订单分账 & 延迟结算任务', desc: '存在启用的分账规则或直付通延迟结算时需配置', path: 'cron.php?do=profitsharing' },
  { title: '商户订单成功率检查任务', desc: '开启自动风控检查时需配置', path: 'cron.php?do=check' },
]

// ============ proxy 中转代理 ============
export const proxyConfig = {
  proxy: '0', // 0关 1开
  proxy_server: '', // 代理IP
  proxy_port: '', // 代理端口
  proxy_user: '', // 代理账号
  proxy_pwd: '', // 代理密码
  proxy_type: 'http', // 代理协议 http/https/sock4/sock5
}
export const proxyTypeOptions = [
  { value: 'http', label: 'HTTP' },
  { value: 'https', label: 'HTTPS' },
  { value: 'sock4', label: 'SOCK4' },
  { value: 'sock5', label: 'SOCK5' },
]

// ============ iptype IP 获取方式 ============
export const ipConfig = {
  ip_type: '0', // 0 X_FORWARDED_FOR / 1 X_REAL_IP / 2 REMOTE_ADDR
}
export const ipTypeOptions = [
  { value: '0', label: 'X_FORWARDED_FOR（易被伪造）' },
  { value: '1', label: 'X_REAL_IP（使用 CDN 时选）' },
  { value: '2', label: 'REMOTE_ADDR（真实请求 IP，无法伪造）' },
]

// 「其余设置」页 Tab（proxy / iptype；管理员账号与支付密码已移至右上角用户菜单）
export const otherSettingTabs = [
  { key: 'proxy', label: '中转代理' },
  { key: 'iptype', label: 'IP 获取方式' },
]
