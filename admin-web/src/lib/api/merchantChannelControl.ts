/**
 * 子商户管控（风控第二段）—— 商户端（/m）「业务受限」面板 API。
 * 只看/只刷新自己名下已开通子商户的管控状态 + 解脱指引，不开放批量刷新/解脱代办（平台运维能力，仅 admin）。
 * 类型复用 admin 侧定义（同一份快照数据，仅数据源维度不同）。
 */
import { request } from './client'
import type { ChannelControlListResp, ChannelControlView } from './riskControl'

export type { ChannelControlState, ChannelControlRecovery, ChannelControlView, ChannelControlOverview, ChannelControlListResp } from './riskControl'

/** 我的业务受限面板：概览 + 我名下已开通子商户管控列表（读本地快照，不现查微信）。 */
export function merchantListChannelControls(): Promise<ChannelControlListResp> {
  return request<ChannelControlListResp>('/merchant/channel-controls')
}

/** 我名下单个进件单的管控快照（渠道申请详情「业务受限」就地快照用）。 */
export function merchantGetChannelControl(enrollId: number): Promise<{ view: ChannelControlView | null }> {
  return request<{ view: ChannelControlView | null }>(`/merchant/channel-controls/${enrollId}`)
}

/** 刷新我名下单个子商户的管控状态（现查微信落快照）。 */
export function merchantRefreshChannelControl(enrollId: number): Promise<{ refreshed: number; failed: number; views: ChannelControlView[] }> {
  return request<{ refreshed: number; failed: number; views: ChannelControlView[] }>(`/merchant/channel-controls/${enrollId}/refresh`, { method: 'POST' })
}
