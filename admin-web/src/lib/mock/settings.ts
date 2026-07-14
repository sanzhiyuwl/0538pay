/**
 * 系统设置假数据。对齐 epay admin/set.php（pre_config，17 个设置模块）。
 * 取最核心的几组做分组 Tab 表单：网站信息 / 支付相关 / 风控 / 结算 / 管理员账号。
 */

/** 设置分组 */
export const settingTabs = [
  { key: 'site', label: '网站信息' },
  { key: 'logo', label: 'LOGO 设置' },
  { key: 'pay', label: '支付相关' },
  { key: 'risk', label: '风控设置' },
  { key: 'settle', label: '结算设置' },
  { key: 'account', label: '管理员账号' },
]

/** LOGO 设置项（对齐参考图，各处 Logo / ICO） */
export const logoItems = [
  { key: 'loginLogo', label: '后台登录页 LOGO', desc: '建议尺寸 270×75', size: '270×75' },
  { key: 'menuLogoSm', label: '后台小 LOGO', desc: '后台菜单缩进小 LOGO，尺寸 180×180', size: '180×180' },
  { key: 'menuLogoLg', label: '后台大 LOGO', desc: '菜单展开左上角 LOGO，建议尺寸 170×50', size: '170×50' },
  { key: 'mobileLogo', label: '移动端登录 LOGO', desc: '移动端登录 LOGO，建议尺寸 86×86，建议 png 格式', size: '86×86' },
  { key: 'favicon', label: '系统 ICO 图标', desc: '程序 ICO 图标，更换后需要清除浏览器缓存', size: '' },
]

/** 网站信息设置（set.php mod=site 精简） */
export const siteConfig = {
  sitename: '0538Pay 聚合支付',
  title: '0538Pay - 专业的聚合支付平台',
  keywords: '聚合支付,支付宝,微信支付,第三方支付',
  description: '0538Pay 提供支付宝、微信、QQ钱包、云闪付等多渠道聚合支付服务',
  company: '泰安市数智支付科技有限公司',
  email: 'service@0538pay.com',
  qq: '800820538',
  copyrightLink: 'https://beian.miit.gov.cn/',
  copyright: 'Copyright © 2026 0538Pay 版权所有',
  icp: '鲁ICP备2026000538号-1',
  police: '鲁公网安备 37098202000538号',
  policeLink: 'https://beian.mps.gov.cn/',
  marketLink: 'https://www.gsxt.gov.cn/',
}

/** 注册登录设置（set.php mod=reg 相关项精简，对齐 epay 真实字段） */
export const regConfig = {
  regOpen: '1', // 开放注册：1开启 0关闭 2仅邀请
  regAudit: false, // 开启注册审核
  regInputSettle: false, // 注册后可不填结算账户
  regPay: false, // 注册付费
  regPayPrice: '10.00', // 注册付费金额
  loginCaptcha: true, // 登录验证码
  captchaVersion: '1', // 极验版本：0 = V3.0，1 = V4.0
  captchaId: '', // 极验验证码 ID
  captchaKey: '', // 极验验证码密钥
}

/** 开放注册选项 */
export const regOpenOptions = [
  { value: '1', label: '开启注册' },
  { value: '0', label: '关闭注册' },
  { value: '2', label: '仅邀请注册' },
]

/** 极验验证码版本选项 */
export const captchaVersionOptions = [
  { value: '0', label: 'V3.0' },
  { value: '1', label: 'V4.0' },
]

/** 支付相关设置（set.php mod=pay 精简） */
export const payConfig = {
  maxMoney: '50000.00', // 最大支付金额
  minMoney: '0.01', // 最小支付金额
  blockKeywords: '博彩,菠菜,赌博,私彩', // 商品屏蔽关键词
  blockTip: '该商品名称含违规词，禁止支付', // 屏蔽显示内容
  customName: false, // 商品名称自定义
  hideName: false, // 扫码页隐藏商品名
  qqRequired: false, // 未填QQ禁止支付
}

/** 风控设置（set.php mod=risk 精简） */
export const riskConfig = {
  successRateOn: true, // 订单成功率风控
  successRateMin: '60', // 成功率阈值 %
  notifyFailOn: true, // 连续通知失败风控
  notifyFailMax: '20', // 连续失败次数上限
  complaintOn: true, // 投诉率风控
  complaintMax: '2', // 投诉率阈值 %
  autoBlock: true, // 触发自动限制收款
}

/** 结算设置（set.php mod=settle 精简） */
export const settleConfig = {
  settleOpen: '3', // 结算开关（对齐用户组）
  settleType: '2', // 结算周期 D+0/D+1
  settleRate: '0.10', // 结算手续费 %
  settleMin: '100', // 最低结算金额
  autoSettleTime: '02:00', // 每日自动结算时间
}

/** 管理员账号（set.php mod=account） */
export const accountConfig = {
  username: 'admin',
}

/** 结算开关选项 */
export const settleOpenOptions = [
  { value: '1', label: '仅每日自动结算' },
  { value: '2', label: '仅手动申请结算' },
  { value: '3', label: '自动 + 手动结算' },
]
export const settleTypeOptions = [
  { value: '1', label: 'D+0（可结算全部余额）' },
  { value: '2', label: 'D+1（可结算前1天余额）' },
]

/** 网站公告（pre_anounce） */
export interface Announce {
  id: number
  content: string
  color: string // 文字颜色
  sort: number
  addtime: string
  status: 0 | 1 // 1显示 0隐藏
}

export const announces: Announce[] = [
  { id: 5, content: '平台已支持微信、支付宝服务商模式，费率低至 0.38%，欢迎咨询接入。', color: '#e11d48', sort: 1, addtime: '2026-07-10 09:00:00', status: 1 },
  { id: 4, content: '7月15日 02:00-04:00 系统例行维护，届时支付可能短暂波动。', color: '#f59e0b', sort: 2, addtime: '2026-07-08 15:30:00', status: 1 },
  { id: 3, content: '新增云闪付通道，支持大额收款，单笔最高 5 万元。', color: '', sort: 3, addtime: '2026-07-05 11:20:00', status: 1 },
  { id: 2, content: '关于加强商户实名认证的通知，请未认证商户尽快完成认证。', color: '', sort: 4, addtime: '2026-06-28 10:00:00', status: 0 },
  { id: 1, content: '平台正式上线，感谢各位商户的支持！', color: '#2563eb', sort: 5, addtime: '2026-06-20 08:00:00', status: 1 },
]

/** 登录日志（pre_log） */
export interface LoginLog {
  id: number
  uid: number // 0=管理员
  type: string // 操作类型
  ip: string
  date: string
}

export const searchColumns = [
  { value: 'uid', label: '商户号' },
  { value: 'ip', label: '操作IP' },
]

const logTypes = ['管理员登录', '商户登录', '登录失败', '修改密码', '退出登录']
function pad(n: number, len = 2) {
  return String(n).padStart(len, '0')
}
export const loginLogs: LoginLog[] = Array.from({ length: 40 }, (_, i) => {
  const isAdmin = i % 5 === 0
  const type = isAdmin ? logTypes[0] : logTypes[(i % 4) + 1]
  const day = 12 - (i % 8)
  return {
    id: 80000 + (40 - i),
    uid: isAdmin ? 0 : 1000 + (i % 12) + 1,
    type,
    ip: `${100 + (i % 100)}.${(i * 7) % 255}.${(i * 3) % 255}.${i % 255}`,
    date: `2026-07-${pad(day)} ${pad(8 + (i % 12))}:${pad((i * 7) % 60)}:${pad((i * 13) % 60)}`,
  }
})

/** 数据清理项（clean.php） */
export const cleanTables = [
  { key: 'order', label: '订单记录', table: 'pre_order' },
  { key: 'settle', label: '结算记录', table: 'pre_settle' },
  { key: 'record', label: '资金明细', table: 'pre_record' },
  { key: 'transfer', label: '付款记录', table: 'pre_transfer' },
  { key: 'psorder', label: '分账记录', table: 'pre_psorder' },
]
