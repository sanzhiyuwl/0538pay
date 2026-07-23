/**
 * 官网文章资讯单一数据源。
 * 运营后台「官网管理 / 文章管理」写入 → 官网首页「最新动态」板块 + 文章详情页读取，实时联动。
 *
 * 存储：真后端行表（pay_article + pay_article_category，对齐 epay pre_article）。
 * - 后台（有 admin token）：hydrate 拉全量 + CRUD 走 /admin/articles 真接口，写后重新 hydrate。
 * - 官网（无 token 公开页）：hydrate 拉 /site/articles 公开列表（仅显示中的），只读。
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  fetchPublicArticles,
  fetchArticles,
  fetchArticleCategories,
  createArticle as apiCreateArticle,
  updateArticle as apiUpdateArticle,
  setArticleActive as apiSetActive,
  deleteArticle as apiDeleteArticle,
  createArticleCategory as apiCreateCat,
  updateArticleCategory as apiUpdateCat,
  deleteArticleCategory as apiDeleteCat,
  type Article,
  type ArticleCategory,
} from '@/lib/api/articles'
import { getToken } from '@/lib/api/client'

export const useArticlesStore = defineStore('site-articles', () => {
  const categories = ref<ArticleCategory[]>([])
  const articles = ref<Article[]>([])

  const isAdmin = () => !!getToken()

  // ===== 拉取 =====
  let hydrated = false
  async function hydrate() {
    if (hydrated) return
    hydrated = true
    await refresh()
  }

  /** 强制重新拉取（写操作后调用，保持与后端一致）。 */
  async function refresh() {
    try {
      if (isAdmin()) {
        // 后台：分类 + 全量文章（含草稿）
        const [catRes, artRes] = await Promise.all([
          fetchArticleCategories(),
          fetchArticles({ pageSize: 200 }),
        ])
        categories.value = catRes.list
        articles.value = artRes.list
      } else {
        // 官网公开：分类 + 显示中的文章
        const res = await fetchPublicArticles()
        categories.value = res.categories
        articles.value = res.articles
      }
    } catch {
      // 后端不可用时保持现有内存数据
    }
  }

  // ===== 分类增删改（后台真接口）=====
  async function addCategory(c: Omit<ArticleCategory, 'id'>) {
    await apiCreateCat(c)
    await refresh()
  }
  async function updateCategory(c: ArticleCategory) {
    await apiUpdateCat(c.id, c)
    await refresh()
  }
  async function removeCategory(id: number) {
    await apiDeleteCat(id)
    await refresh()
  }

  // ===== 文章增删改（后台真接口）=====
  async function addArticle(a: Omit<Article, 'id'>) {
    await apiCreateArticle(a)
    await refresh()
  }
  async function updateArticle(a: Article) {
    await apiUpdateArticle(a.id, a)
    await refresh()
  }
  async function removeArticle(id: number) {
    await apiDeleteArticle(id)
    await refresh()
  }
  async function setArticleStatus(id: number, status: number) {
    await apiSetActive(id, status)
    await refresh()
  }

  // ===== 官网读取用派生数据 =====
  /** 按分类分组，已发布文章按 置顶优先 → sort 升序 排列 */
  const publishedByCategory = computed(() =>
    [...categories.value]
      .sort((a, b) => a.sort - b.sort)
      .map((cat) => ({
        category: cat,
        list: articles.value
          .filter((a) => a.categoryId === cat.id && a.status === 1)
          .sort((a, b) => Number(b.isTop) - Number(a.isTop) || a.sort - b.sort),
      })),
  )

  /** 按 id 取单篇文章（详情页用；列表中没有时返回 undefined，详情页应走 fetchArticleDetail） */
  function getArticle(id: number): Article | undefined {
    return articles.value.find((a) => a.id === id)
  }

  /** 分类名映射（列表页显示用）*/
  const categoryName = computed<Record<number, string>>(() =>
    Object.fromEntries(categories.value.map((c) => [c.id, c.name])),
  )

  /** 全部已发布文章（置顶优先 → sort 升序 → id 倒序，与后端排序一致）*/
  const published = computed(() =>
    articles.value
      .filter((a) => a.status === 1)
      .sort(
        (a, b) => Number(b.isTop) - Number(a.isTop) || a.sort - b.sort || b.id - a.id,
      ),
  )

  /** 热门资讯：已发布文章按浏览量降序取前 N（侧栏「热门资讯」用）*/
  function hotArticles(limit = 5): Article[] {
    return [...published.value].sort((a, b) => b.views - a.views).slice(0, limit)
  }

  /** 热门标签云：聚合全部已发布文章的 tags，按出现频次降序（侧栏「热门标签」用）*/
  const hotTags = computed<{ name: string; count: number }[]>(() => {
    const freq = new Map<string, number>()
    for (const a of published.value) {
      for (const t of a.tags ?? []) {
        const name = t.trim()
        if (name) freq.set(name, (freq.get(name) ?? 0) + 1)
      }
    }
    return [...freq.entries()]
      .map(([name, count]) => ({ name, count }))
      .sort((a, b) => b.count - a.count || a.name.localeCompare(b.name))
  })

  return {
    categories,
    articles,
    hydrate,
    refresh,
    addCategory,
    updateCategory,
    removeCategory,
    addArticle,
    updateArticle,
    removeArticle,
    setArticleStatus,
    publishedByCategory,
    published,
    hotArticles,
    hotTags,
    getArticle,
    categoryName,
  }
})
