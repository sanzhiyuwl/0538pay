/** 系统设置 config 域 API（对齐后端 /admin/config/:group，键名对齐 epay set.php）。 */
import { request } from './client'

/** 配置分组：键→值（字符串）。前端按分组读写。 */
export type ConfigKV = Record<string, string>

/** 读取某分组配置（回填设置页），返回 key→当前值(含默认)。 */
export function fetchConfig(group: string): Promise<ConfigKV> {
  return request<ConfigKV>(`/admin/config/${group}`)
}

/** 保存某分组配置（仅该分组白名单键会被后端接受）。 */
export function saveConfig(group: string, kv: ConfigKV): Promise<{ group: string }> {
  return request<{ group: string }>(`/admin/config/${group}`, { method: 'PUT', body: kv })
}

/** 用当前邮件配置发送一封测试邮件（K-3，对齐 epay set.php 测试邮件）。to 留空则发到管理员收信邮箱。 */
export function testMail(to?: string): Promise<{ to: string }> {
  return request<{ to: string }>('/admin/config/mail/test', { method: 'POST', body: { to: to || '' } })
}

/** 真实 IP 探测结果行（对齐 epay ajax.php iptype） */
export interface IPTypeProbe {
  name: string
  ip: string
  city: string
}

/** 探测三种取值方式各自解析到的真实 IP 与归属地（F-9，对齐 epay set.php mod=iptype）。 */
export function detectIPType(): Promise<IPTypeProbe[]> {
  return request<IPTypeProbe[]>('/admin/config/iptype')
}

/** 企微客服账号（H5 微信客服支付设置页下拉，对齐 epay set_wxkf.php 账号选择）。 */
export interface WxkfAccount {
  id: number
  openkfid: string
  name: string
}

/** 列出所有已启用企微下的客服账号（供「指定客服账号」下拉）。 */
export function fetchWxkfAccounts(): Promise<{ list: WxkfAccount[] }> {
  return request<{ list: WxkfAccount[] }>('/admin/wxkf/accounts')
}
