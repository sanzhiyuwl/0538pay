/** 商户中心业务 API（工作台/订单/流水/结算/提现/退款）。类型复用 mock 里已定义的结构。 */
import { request, type PageResult } from './client'
import type { Order } from '@/lib/mock/merchant/orders'
import type { FundRecord } from '@/lib/mock/merchant/records'
import type { SettleRecord } from '@/lib/mock/merchant/settle'

// ===== 工作台聚合 =====
export interface DashboardInfo {
  uid: number
  name: string
  qq: string
  status: string // normal/banned/payoff/settleoff/auditing/uncert
  groupName: string
  cert: number // 0未认证 1已实名
  certType: number // 0个人 1企业
  money: number
  settleMoney: number
  todayIncome: number
  yesterdayIncome: number
  orders: number
  ordersToday: number
}
export interface DashboardAlerts {
  needCert: boolean
  noSecurity: boolean
  noLoginPwd: boolean
}
export interface DashboardCertGuide {
  show: boolean
}
export interface DashboardDepositGuide {
  show: boolean
  min: number
  current: number
  gap: number
}
export interface DashboardGuides {
  cert: DashboardCertGuide
  deposit: DashboardDepositGuide
}
export interface DashboardChannel {
  typename: string
  showname: string
  today: number
  yesterday: number
  successRate: number
  rate: string
}
export interface DashboardAnnounce {
  id: number
  content: string
  color: string
  time: string
}
export interface DashboardTrend {
  labels: string[]
  data: number[]
}
export interface MerchantDashboard {
  merchantInfo: DashboardInfo
  alerts: DashboardAlerts
  guides: DashboardGuides
  channels: DashboardChannel[]
  announces: DashboardAnnounce[]
  trend: DashboardTrend
}

export function fetchDashboard(): Promise<MerchantDashboard> {
  return request<MerchantDashboard>('/merchant/dashboard')
}

// ===== 订单查询 =====
export interface MerchantOrderParams {
  page?: number
  pageSize?: number
  column?: string
  keyword?: string
  status?: number
  type?: number
  channel?: number
  starttime?: string
  endtime?: string
}

/** 商户订单统计概况（对齐后端 dto.OrderStats，F-13） */
export interface MerchantOrderStats {
  totalMoney: number
  successMoney: number
  unpaidMoney: number
  refundMoney: number
  platformProfit: number
  totalCount: number
  successCount: number
  unpaidCount: number
  refundCount: number
  successRate: number
}
export function fetchMerchantOrders(
  params: MerchantOrderParams = {},
): Promise<PageResult<Order>> {
  return request<PageResult<Order>>('/merchant/orders', { query: { ...params } })
}

/** 商户订单统计概况（服务端聚合，含 platformProfit；F-13，对齐 epay statistics）。 */
export function fetchMerchantOrderStats(
  params: MerchantOrderParams = {},
): Promise<MerchantOrderStats> {
  return request<MerchantOrderStats>('/merchant/orders/stats', { query: { ...params } })
}

/** 订单退款（全额） */
/** 订单退款：支持部分退款(money 空则全额) + 登录密码二次校验（对齐 epay refund_submit）。 */
export function refundOrder(
  tradeNo: string,
  password: string,
  money?: string,
): Promise<{ trade_no: string; status: number }> {
  return request('/merchant/order/refund', {
    method: 'POST',
    body: { trade_no: tradeNo, password, money: money ?? '' },
  })
}

/** 重新通知（补单/重发回调） */
export function renotifyOrder(tradeNo: string): Promise<{ trade_no: string }> {
  return request('/merchant/order/notify', { method: 'POST', body: { trade_no: tradeNo } })
}

// ===== 资金流水 =====
export interface MerchantRecordParams {
  page?: number
  pageSize?: number
  action?: number
  keyword?: string
}
export function fetchMerchantRecords(
  params: MerchantRecordParams = {},
): Promise<PageResult<FundRecord>> {
  return request<PageResult<FundRecord>>('/merchant/records', { query: { ...params } })
}

// ===== 结算记录 =====
export function fetchMerchantSettles(
  params: { page?: number; pageSize?: number; status?: number } = {},
): Promise<PageResult<SettleRecord>> {
  return request<PageResult<SettleRecord>>('/merchant/settles', { query: { ...params } })
}

// ===== 申请提现 =====
export interface ApplyInfo {
  settleName: string
  account: string
  username: string
  money: number
  enableMoney: number
  settleMin: number
  settleMaxLimit: number
  settleRate: number
  settleFeeMin: number
  settleFeeMax: number
  settleType: number
  todayCount: number
}
export function fetchApplyInfo(): Promise<ApplyInfo> {
  return request<ApplyInfo>('/merchant/apply/info')
}
export function submitApply(amount: string): Promise<{ ok: boolean }> {
  return request('/merchant/apply', { method: 'POST', body: { amount } })
}

// ===== API 信息 / 资料 / 密码（D3）=====
export interface ApiInfo {
  uid: number
  mdkey: string
  apiurl: string
  keytype: number // 0=MD5+RSA兼容 1=仅RSA安全
  has_rsa: boolean // 是否已配 RSA 公钥
}
export function fetchApiInfo(): Promise<ApiInfo> {
  return request<ApiInfo>('/merchant/apikey')
}
export function resetApiKey(): Promise<{ mdkey: string }> {
  return request('/merchant/apikey/reset', { method: 'POST' })
}
/** 生成商户 RSA 密钥对（V2），私钥一次性返回 */
export function genRsaKeyPair(): Promise<{ private_key: string }> {
  return request('/merchant/apikey/rsa', { method: 'POST' })
}
/** 设置签名模式 0=MD5+RSA兼容 1=仅RSA安全 */
export function setKeyType(keytype: number): Promise<{ keytype: number }> {
  return request('/merchant/apikey/keytype', { method: 'PUT', body: { keytype } })
}

export interface ProfileReq {
  settle_id: number
  account: string
  username: string
  email: string
  qq: string
  url: string
  mode: number
  // 对齐 epay edit_info：可选提交（不传则不改）
  keylogin?: number
  refund?: number
  transfer?: number
  remain_money?: string
  password?: string // 改动收款账号/姓名/结算方式时登录密码二次确认
}
export function updateProfile(body: ProfileReq): Promise<{ ok: boolean }> {
  return request('/merchant/profile', { method: 'PUT', body })
}

// D-3 消息提醒配置
export function fetchMsgConfig(): Promise<{ msgconfig: string }> {
  return request('/merchant/msgconfig')
}
export function saveMsgConfig(msgconfig: string): Promise<{ ok: boolean }> {
  return request('/merchant/msgconfig', { method: 'PUT', body: { msgconfig } })
}

// D-3 换绑手机/邮箱（登录密码二次确认）
export function rebindContact(field: 'phone' | 'email', value: string, password: string): Promise<{ ok: boolean }> {
  return request('/merchant/rebind', { method: 'POST', body: { field, value, password } })
}

// F-20 自定义接口信息设置（对齐 epay editinfo edit_channel_info，按用户组 settings 模板动态渲染）
export interface ChannelInfoField {
  key: string
  label: string
  value: string
}
export interface ChannelInfoView {
  editable: boolean
  fields: ChannelInfoField[]
}
export function fetchChannelInfo(): Promise<ChannelInfoView> {
  return request('/merchant/channelinfo')
}
export function saveChannelInfo(settings: Record<string, string>): Promise<{ ok: boolean }> {
  return request('/merchant/channelinfo', { method: 'PUT', body: { settings } })
}

/** 解绑第三方账号（F-10，对齐 epay editinfo ?unbind=1 清 openid 列）。 */
export function unbindOAuth(provider: 'qq' | 'wx' | 'alipay'): Promise<{ provider: string }> {
  return request('/merchant/oauth/unbind', { method: 'POST', body: { provider } })
}

export function changePassword(oldpwd: string, newpwd: string): Promise<{ ok: boolean }> {
  return request('/merchant/password', { method: 'PUT', body: { oldpwd, newpwd } })
}

// ===== 保证金（D3）=====
export interface DepositInfo {
  enabled: boolean // 保证金门槛全局开关（user_deposit）；关闭时隐藏保证金入口/页面
  deposit: number
  depositMin: number
  money: number
}
export function fetchDepositInfo(): Promise<DepositInfo> {
  return request<DepositInfo>('/merchant/deposit')
}
export function rechargeDeposit(amount: string, payType = 'balance'): Promise<{ ok: boolean }> {
  return request('/merchant/deposit/recharge', { method: 'POST', body: { amount, pay_type: payType } })
}
export function withdrawDeposit(amount: string): Promise<{ ok: boolean }> {
  return request('/merchant/deposit/withdraw', { method: 'POST', body: { amount } })
}

// ===== 商户端全局功能开关 =====
// 由平台在后台「支付设置」等处控制，商户端据此过滤导航/守卫页面（如保证金门槛开关）。
export interface MerchantFeatures {
  deposit: boolean // 保证金门槛开关（user_deposit），关闭时隐藏保证金入口与页面
}
export function fetchMerchantFeatures(): Promise<MerchantFeatures> {
  return request<MerchantFeatures>('/merchant/features')
}

// ===== 购买会员（D3）=====
export interface GroupRateItem {
  label: string
  rate: string
}
export interface GroupPlan {
  id: number
  name: string
  price: number
  expire: number // 月数，0=永久
  rates: GroupRateItem[]
}
export interface GroupCurrent {
  gid: number
  name: string
  expire: string // 到期时间，"—"=永久/无
  rates: GroupRateItem[] // 当前组可用通道及费率（含默认组）
}
export function fetchGroups(): Promise<{ plans: GroupPlan[]; current: GroupCurrent }> {
  return request('/merchant/groups')
}
export function buyGroup(gid: number, num: number, payType = 'balance'): Promise<{ ok: boolean }> {
  return request('/merchant/groups/buy', { method: 'POST', body: { gid, num, pay_type: payType } })
}

// ===== 余额充值（D3）=====
export interface RechargeResp {
  trade_no: string
  pay_type: string
  pay_url?: string
  qrcode?: string
  money: string
}
export function rechargeBalance(amount: string, plugin = 'mock'): Promise<RechargeResp> {
  return request<RechargeResp>('/merchant/recharge', { method: 'POST', body: { amount, plugin } })
}

// ===== 实名认证（D3，第三方认证待凭证）=====
export interface CertInfo {
  cert: number // 0未认证/审核中 1已认证
  certtype: number // 0个人 1企业
  certname: string
  certno: string
  certcorp: string
  certcorpno: string // 营业执照号（企业认证，脱敏）
  certtime: string
  certmoney: number
  method: string
  corpopen: boolean
}
export function fetchCertInfo(): Promise<CertInfo> {
  return request<CertInfo>('/merchant/cert')
}
// 异步扫码方式（腾讯云人脸核身）返回 async=true + 扫码链接 qrurl；同步方式 async=false。
export interface CertSubmitResult {
  async: boolean
  qrurl?: string
}
export function submitCert(body: { certtype: number; certname: string; certno: string; certcorp: string; certcorpno?: string }): Promise<CertSubmitResult> {
  return request('/merchant/cert', { method: 'POST', body })
}

// ===== 测试支付（对齐 epay user/test.php）=====
export interface PayTypeOption {
  type: string
  showname: string
}
export interface TestPayInfo {
  open: boolean
  min_money: string
  max_money: string
  types: PayTypeOption[]
}
export interface SubmitResp {
  trade_no: string
  out_trade_no: string
  pay_type: string
  pay_url?: string
  qrcode?: string
  money: string
}
export function fetchTestPayInfo(): Promise<TestPayInfo> {
  return request<TestPayInfo>('/merchant/test')
}
export function submitTestPay(money: string, type: string): Promise<SubmitResp> {
  return request<SubmitResp>('/merchant/test', { method: 'POST', body: { money, type } })
}

// ===== 聚合收款码（对齐 epay user/onecode.php）=====
export interface OnecodeInfo {
  open: boolean
  pay_url: string
  codename: string
}
export function fetchOnecodeInfo(): Promise<OnecodeInfo> {
  return request<OnecodeInfo>('/merchant/onecode')
}
export function saveCodeName(codename: string): Promise<{ codename: string }> {
  return request('/merchant/onecode/name', { method: 'POST', body: { codename } })
}

// ===== 邀请返现（对齐 epay user/invite.php）=====
export interface InviteRewardInfo {
  open: boolean
  rate: string
  order_type: number // 0按订单金额/1按手续费/2按利润
  order_fee: boolean
  link: string
  code: string
}
export interface InviteRewardStat {
  users: number
  income_today: string
  income_yesterday: string
  income_total: string
}
export interface InvitedUser {
  uid: number
  addtime: string
  status: number
}
export interface InviteData {
  info: InviteRewardInfo
  stat: InviteRewardStat
  list: InvitedUser[]
  total: number
}
export function fetchInvite(params: { page?: number; pageSize?: number } = {}): Promise<InviteData> {
  return request<InviteData>('/merchant/invite', { query: { ...params } })
}

// ===== 授权域名自助（对齐 epay user/domain.php）=====
export interface MerchantDomain {
  id: number
  uid: number
  domain: string
  status: number // 0待审核 1正常 2拒绝
  addtime: string
  endtime: string | null
}
export function fetchMerchantDomains(): Promise<{ list: MerchantDomain[] }> {
  return request('/merchant/domains')
}
export function addMerchantDomain(domain: string): Promise<{ domain: string }> {
  return request('/merchant/domains', { method: 'POST', body: { domain } })
}
export function deleteMerchantDomain(id: number): Promise<{ id: number }> {
  return request(`/merchant/domains/${id}`, { method: 'DELETE' })
}

// ===== 使用说明（后台可编辑）=====
export function fetchHelp(): Promise<{ content: string; sitename: string }> {
  return request('/merchant/help')
}

// ===== 站内信（我方新增）=====
export interface MerchantMessage {
  id: number
  uid: number
  title: string
  content: string
  is_read: boolean
  date: string
}
export function fetchMessages(params: { page?: number; pageSize?: number } = {}): Promise<{ list: MerchantMessage[]; total: number; unread: number }> {
  return request('/merchant/messages', { query: { ...params } })
}
export function readMessage(id: number): Promise<{ id: number }> {
  return request(`/merchant/messages/${id}/read`, { method: 'POST' })
}
