/**
 * 官网文章资讯单一数据源（CMS）。
 * 运营后台「官网管理 / 文章管理」写入 → 官网首页「最新动态」板块 + 文章详情页读取，实时联动。
 * 持久化到 localStorage，从 mock 取默认值（后端接好后换成接口）。
 */
import { defineStore } from 'pinia'
import { ref, watch, computed } from 'vue'
import {
  defaultArticles,
  defaultArticleCategories,
  type Article,
  type ArticleCategory,
} from '@/lib/mock/articles'

const STORAGE_KEY = 'site-articles'

interface Persisted {
  categories: ArticleCategory[]
  articles: Article[]
}

function clone<T>(src: T): T {
  return JSON.parse(JSON.stringify(src))
}

function load(): Persisted {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw)
      if (Array.isArray(parsed?.categories) && Array.isArray(parsed?.articles)) {
        return parsed as Persisted
      }
    }
  } catch {
    // 损坏缓存忽略，回退默认
  }
  return { categories: clone(defaultArticleCategories), articles: clone(defaultArticles) }
}

export const useArticlesStore = defineStore('site-articles', () => {
  const initial = load()
  const categories = ref<ArticleCategory[]>(initial.categories)
  const articles = ref<Article[]>(initial.articles)

  /** 恢复默认数据 */
  function reset() {
    categories.value = clone(defaultArticleCategories)
    articles.value = clone(defaultArticles)
  }

  // ===== 分类增删改 =====
  function addCategory(c: Omit<ArticleCategory, 'id'>) {
    const id = Math.max(0, ...categories.value.map((x) => x.id)) + 1
    categories.value.push({ ...c, id })
  }
  function updateCategory(c: ArticleCategory) {
    const i = categories.value.findIndex((x) => x.id === c.id)
    if (i >= 0) categories.value[i] = { ...c }
  }
  function removeCategory(id: number) {
    categories.value = categories.value.filter((x) => x.id !== id)
    // 该分类下的文章一并移除
    articles.value = articles.value.filter((a) => a.categoryId !== id)
  }

  // ===== 文章增删改 =====
  function addArticle(a: Omit<Article, 'id'>) {
    const id = Math.max(0, ...articles.value.map((x) => x.id)) + 1
    articles.value.push({ ...a, id })
  }
  function updateArticle(a: Article) {
    const i = articles.value.findIndex((x) => x.id === a.id)
    if (i >= 0) articles.value[i] = { ...a }
  }
  function removeArticle(id: number) {
    articles.value = articles.value.filter((a) => a.id !== id)
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

  /** 按 id 取单篇文章（详情页用）*/
  function getArticle(id: number): Article | undefined {
    return articles.value.find((a) => a.id === id)
  }

  /** 分类名映射（列表页显示用）*/
  const categoryName = computed<Record<number, string>>(() =>
    Object.fromEntries(categories.value.map((c) => [c.id, c.name])),
  )

  watch(
    [categories, articles],
    () => {
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({ categories: categories.value, articles: articles.value }),
      )
    },
    { deep: true },
  )

  return {
    categories,
    articles,
    reset,
    addCategory,
    updateCategory,
    removeCategory,
    addArticle,
    updateArticle,
    removeArticle,
    publishedByCategory,
    getArticle,
    categoryName,
  }
})
