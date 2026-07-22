/** 官网 CMS 内容 API（自研）。三份文档：content(首页营销)/docs(开发文档)/settings(网站设置)。
 * 官网公开读 GET /api/site/config/:key；后台鉴权写 PUT /api/admin/site/config/:key。
 * value 为整份 JSON 字符串（后端只做 KV + JSON 合法性校验，结构由前端定义）。
 */
import { request } from './client'

export type SiteConfigKey = 'content' | 'docs' | 'settings' | 'articles'

/** 读取某份 CMS 文档，返回解析后的对象；无记录或失败返回 null（调用方回退默认）。 */
export async function fetchSiteConfig<T>(key: SiteConfigKey): Promise<T | null> {
  const res = await request<{ value: string }>(`/site/config/${key}`)
  if (!res.value) return null
  try {
    return JSON.parse(res.value) as T
  } catch {
    return null
  }
}

/** 保存某份 CMS 文档（整份覆盖）。 */
export function saveSiteConfig(key: SiteConfigKey, doc: unknown): Promise<{ key: string }> {
  return request(`/admin/site/config/${key}`, {
    method: 'PUT',
    body: { value: JSON.stringify(doc) },
  })
}
