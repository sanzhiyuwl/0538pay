/**
 * 图片上传 API。走 multipart/form-data（request 助手仅支持 JSON，故单列）。
 * 后端：POST /api/admin/upload/image，字段 file + 可选 dir，返回 { url }。
 * 对齐 epay admin/ajax.php article_upload（本地磁盘存储 + 静态访问）。
 */
import { ApiError, getToken } from './client'

interface ApiBody<T> {
  code: number
  msg: string
  data: T
}

/** 上传图片，返回可直接用于 <img src> 的相对 URL（如 /uploads/article/xxx.png）。 */
export async function uploadImage(file: File, dir: 'article' | 'cover' | 'category' = 'article'): Promise<string> {
  const fd = new FormData()
  fd.append('file', file)
  fd.append('dir', dir)

  const headers: Record<string, string> = {}
  const token = getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`

  const res = await fetch('/api/admin/upload/image', {
    method: 'POST',
    headers, // 不手动设 Content-Type，交给浏览器带 boundary
    body: fd,
  })
  if (res.status === 401) throw new ApiError(401, '登录已失效，请重新登录')

  let json: ApiBody<{ url: string }>
  try {
    json = await res.json()
  } catch {
    throw new ApiError(res.status, `上传失败(${res.status})`)
  }
  if (json.code !== 0) throw new ApiError(json.code, json.msg || '上传失败')
  return json.data.url
}
