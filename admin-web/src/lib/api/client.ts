/**
 * 统一 API 客户端。约定后端响应 { code, msg, data }，code=0 成功。
 * 负责：拼 baseURL、带 JWT 头、解包 data、错误抛出、401 处理。
 */

const BASE = '/api' // 经 vite dev 代理转发到后端 :8080

// 多端 token 隔离：admin/console 用 admin_token，商户中心用 merchant_token，互不覆盖。
const ADMIN_TOKEN_KEY = 'admin_token'
const MERCHANT_TOKEN_KEY = 'merchant_token'

// 按请求路径前缀判定该用哪个端的 token。
function tokenKeyForPath(path: string): string {
  return path.startsWith('/merchant') ? MERCHANT_TOKEN_KEY : ADMIN_TOKEN_KEY
}

// —— admin/console 端 token（保持原有 API 名不变，兼容既有调用）——
export function getToken(): string {
  return localStorage.getItem(ADMIN_TOKEN_KEY) || ''
}
export function setToken(t: string) {
  localStorage.setItem(ADMIN_TOKEN_KEY, t)
}
export function clearToken() {
  localStorage.removeItem(ADMIN_TOKEN_KEY)
}

// —— 商户端 token ——
export function getMerchantToken(): string {
  return localStorage.getItem(MERCHANT_TOKEN_KEY) || ''
}
export function setMerchantToken(t: string) {
  localStorage.setItem(MERCHANT_TOKEN_KEY, t)
}
export function clearMerchantToken() {
  localStorage.removeItem(MERCHANT_TOKEN_KEY)
}

/** 后端统一响应体 */
interface ApiBody<T> {
  code: number
  msg: string
  data: T
}

/** 401 回调：由 app 注入（跳登录页），避免 lib 层直接依赖 router。分端各一个。 */
let unauthorizedHandler: (() => void) | null = null
export function onUnauthorized(fn: () => void) {
  unauthorizedHandler = fn
}
let merchantUnauthorizedHandler: (() => void) | null = null
export function onMerchantUnauthorized(fn: () => void) {
  merchantUnauthorizedHandler = fn
}

/** 业务错误：code 非 0 时抛出，携带 code 与 msg */
export class ApiError extends Error {
  code: number
  constructor(code: number, msg: string) {
    super(msg)
    this.code = code
    this.name = 'ApiError'
  }
}

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
  query?: Record<string, string | number | boolean | undefined | null>
  body?: unknown
}

function buildQuery(query?: RequestOptions['query']): string {
  if (!query) return ''
  const parts: string[] = []
  for (const [k, v] of Object.entries(query)) {
    if (v === undefined || v === null || v === '') continue
    parts.push(`${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`)
  }
  return parts.length ? `?${parts.join('&')}` : ''
}

/** 核心请求方法，返回解包后的 data */
export async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const { method = 'GET', query, body } = opts
  const headers: Record<string, string> = {}
  const isMerchant = path.startsWith('/merchant')
  const token = localStorage.getItem(tokenKeyForPath(path)) || ''
  if (token) headers['Authorization'] = `Bearer ${token}`

  const init: RequestInit = { method, headers }
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    init.body = JSON.stringify(body)
  }

  const res = await fetch(`${BASE}${path}${buildQuery(query)}`, init)

  // 网络/HTTP 层错误
  if (res.status === 401) {
    if (isMerchant) {
      clearMerchantToken()
      if (merchantUnauthorizedHandler) merchantUnauthorizedHandler()
    } else {
      clearToken()
      if (unauthorizedHandler) unauthorizedHandler()
    }
    throw new ApiError(401, '登录已失效，请重新登录')
  }
  let json: ApiBody<T>
  try {
    json = await res.json()
  } catch {
    throw new ApiError(res.status, `服务异常(${res.status})`)
  }

  if (json.code !== 0) {
    throw new ApiError(json.code, json.msg || '请求失败')
  }
  return json.data
}

/** 分页响应形状 */
export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}
