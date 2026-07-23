/** 文章 + 分类 API。后台 CRUD（需鉴权）+ 官网公开读取。对齐后端 ArticleService。 */
import { request } from './client'

/** 文章分类（对齐后端 CategoryView / 前端 mock ArticleCategory） */
export interface ArticleCategory {
  id: number
  name: string
  enName: string
  cover: string
  sort: number
}

/** 文章（对齐后端 ArticleView / 前端 mock Article） */
export interface Article {
  id: number
  categoryId: number
  title: string
  summary: string
  content: string
  cover: string
  tags: string[] // 文章标签（列表 meta pill + 侧栏热门标签云聚合）
  isNew: boolean
  isTop: boolean
  status: number // 1=发布 0=草稿
  sort: number
  views: number
  addtime: string // YYYY-MM-DD
}

// ===== 官网公开读取 =====

/** 官网首页/文章页：分类 + 显示中的文章 */
export function fetchPublicArticles(): Promise<{ categories: ArticleCategory[]; articles: Article[] }> {
  return request('/site/articles')
}

/** 文章详情（浏览量 +1） */
export function fetchArticleDetail(id: number): Promise<Article> {
  return request(`/site/articles/${id}`)
}

// ===== 后台管理 =====

/** 后台文章列表（DB 分页 + 标题搜索） */
export function fetchArticles(params: { page?: number; pageSize?: number; kw?: string } = {}): Promise<{
  list: Article[]
  total: number
}> {
  return request('/admin/articles', { query: { ...params } })
}

export function createArticle(body: Partial<Article>): Promise<{ id: number }> {
  return request('/admin/articles', { method: 'POST', body })
}

export function updateArticle(id: number, body: Partial<Article>): Promise<{ id: number }> {
  return request(`/admin/articles/${id}`, { method: 'PUT', body })
}

export function setArticleActive(id: number, status: number): Promise<{ id: number }> {
  return request(`/admin/articles/${id}/status`, { method: 'PUT', body: { status } })
}

export function deleteArticle(id: number): Promise<{ id: number }> {
  return request(`/admin/articles/${id}`, { method: 'DELETE' })
}

// ===== 分类管理 =====

export function fetchArticleCategories(): Promise<{ list: ArticleCategory[] }> {
  return request('/admin/article-categories')
}

export function createArticleCategory(body: Partial<ArticleCategory>): Promise<{ id: number }> {
  return request('/admin/article-categories', { method: 'POST', body })
}

export function updateArticleCategory(id: number, body: Partial<ArticleCategory>): Promise<{ id: number }> {
  return request(`/admin/article-categories/${id}`, { method: 'PUT', body })
}

export function deleteArticleCategory(id: number): Promise<{ id: number }> {
  return request(`/admin/article-categories/${id}`, { method: 'DELETE' })
}
