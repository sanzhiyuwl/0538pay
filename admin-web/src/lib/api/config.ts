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
