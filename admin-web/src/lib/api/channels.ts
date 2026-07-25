/** 支付通道 API。Channel 类型复用 mock 里已定义的结构（字段一致）。 */
import { request, type PageResult } from './client'
import type { Channel } from '@/lib/mock/channels'

export interface ChannelListParams {
  page?: number
  pageSize?: number
  keyword?: string
  plugin?: string
  type?: number
  status?: number
}

/** 拉取支付通道列表（分页） */
export function fetchChannels(params: ChannelListParams = {}): Promise<PageResult<Channel>> {
  return request<PageResult<Channel>>('/admin/channels', { query: { ...params } })
}

/** 新增/编辑通道的表单入参（对齐后端 dto.ChannelSaveReq） */
export interface ChannelSaveReq {
  name: string
  type: number
  plugin: string
  mode: number
  rate: string
  costrate: string
  daytop: number
  paymin: string
  paymax: string
}

/** 新增通道，返回新建 ID */
export function createChannel(body: ChannelSaveReq): Promise<{ id: number }> {
  return request<{ id: number }>('/admin/channels', { method: 'POST', body })
}

/** 编辑通道 */
export function updateChannel(id: number, body: ChannelSaveReq): Promise<{ id: number }> {
  return request<{ id: number }>(`/admin/channels/${id}`, { method: 'PUT', body })
}

/** 删除通道 */
export function deleteChannel(id: number): Promise<{ id: number }> {
  return request<{ id: number }>(`/admin/channels/${id}`, { method: 'DELETE' })
}

/** 切换通道状态 */
export function setChannelStatus(id: number, status: number): Promise<{ id: number; status: number }> {
  return request<{ id: number; status: number }>(`/admin/channels/${id}/status`, {
    method: 'PUT',
    body: { status },
  })
}

/** 通道密钥配置回填 */
export interface ChannelConfig {
  id: number
  name: string
  plugin: string
  config: string
}

/** 读取通道密钥配置 */
export function fetchChannelConfig(id: number): Promise<ChannelConfig> {
  return request<ChannelConfig>(`/admin/channels/${id}/config`)
}

/** 保存通道密钥配置 */
export function saveChannelConfig(id: number, config: string): Promise<{ id: number }> {
  return request<{ id: number }>(`/admin/channels/${id}/config`, {
    method: 'PUT',
    body: { config },
  })
}

/** 插件配置字段元数据（对齐后端 channel.FieldInput，驱动密钥表单动态渲染） */
export interface PluginFieldInput {
  name: string
  label: string
  type: string // text/password/textarea/select
  options: string[] | null
  require: boolean
  tip: string
}

/** 插件支持的支付产品形态（对齐后端 channel.ProductType） */
export interface PluginProduct {
  code: string
  name: string
  group?: string // 所属支付方式分组（alipay/wxpay/qqpay/bank；空=不分组），对齐 ltzf select_xxx
  need_sign?: boolean // 是否需渠道侧签约才可选，对齐 alipay「只能选已签约产品」
}

/** 插件能力与配置元数据（对齐后端 channel.PluginMeta） */
export interface PluginMeta {
  key: string
  showname: string // 完整中文名（后端 Describe 单一数据源）
  brand: string // 品牌族（前端分组折叠用）
  protocol: string // 协议/版本（APIv3/APIv2/V1(MD5)…）
  form: string // 支付形态（Native 扫码/JSAPI/H5…）
  methods: string[] | null // 支持的支付方式（alipay/wxpay/qqpay/bank），驱动新建通道时按支付方式过滤插件候选
  inputs: PluginFieldInput[] | null
  products: PluginProduct[] | null
  can_refund: boolean
  can_transfer: boolean
  configurable: boolean
  enabled: boolean // 是否启用（后台开关，禁用后收单选通道跳过该插件）
}

/** 拉取所有已注册渠道插件的能力/配置元数据（后台按插件动态渲染密钥表单） */
export function fetchPluginMeta(): Promise<PluginMeta[]> {
  return request<PluginMeta[]>('/admin/channels/plugins')
}

/** 启用/禁用某已注册插件（后台开关，禁用后收单选通道跳过该插件） */
export function setPluginStatus(
  key: string,
  enabled: boolean,
): Promise<{ key: string; enabled: boolean }> {
  return request<{ key: string; enabled: boolean }>(
    `/admin/channels/plugins/${encodeURIComponent(key)}/status`,
    { method: 'PUT', body: { enabled } },
  )
}

/** 批量启用/禁用多个已注册插件（品牌卡「一键关停/开启整个品牌」） */
export function setPluginsStatus(
  keys: string[],
  enabled: boolean,
): Promise<{ keys: string[]; enabled: boolean }> {
  return request<{ keys: string[]; enabled: boolean }>('/admin/channels/plugins/status', {
    method: 'PUT',
    body: { keys, enabled },
  })
}

/** 通道测试支付返回（对齐后端 dto.SubmitResp，收银台据 trade_no 渲染） */
export interface ChannelTestPayResp {
  trade_no: string
  out_trade_no: string
  pay_type: string
  pay_url: string
  qrcode: string
  html: string
  money: string
}

/** 后台通道测试支付：定向指定通道(+可选子通道)下一笔真实测试单，返回收银台可用下单信息 */
export function channelTestPay(body: {
  channel: number
  subchannel?: number
  name?: string
  money: string
}): Promise<ChannelTestPayResp> {
  return request<ChannelTestPayResp>('/admin/channels/testpay', { method: 'POST', body })
}
