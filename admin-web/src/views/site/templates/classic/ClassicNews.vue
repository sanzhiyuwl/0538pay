<script setup lang="ts">
/**
 * 官网文章详情页。/news/:id 读取文章正文（tiptap HTML）渲染。
 * 侧栏展示同分类的其它文章，便于跳转。
 */
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Eye, CalendarDays, ChevronRight, ChevronLeft, Megaphone } from 'lucide-vue-next'
import { useArticlesStore } from '@/stores/articles'
import { fetchArticleDetail, type Article } from '@/lib/api/articles'
import NewsSidebar from './sections/NewsSidebar.vue'

const route = useRoute()
const router = useRouter()
const store = useArticlesStore()

const id = computed(() => Number(route.params.id))
// 详情走公开接口（浏览量 +1），侧栏热门/标签从 store 列表取，故也 hydrate。
const article = ref<Article | null>(null)
async function loadDetail() {
  article.value = null
  try {
    article.value = await fetchArticleDetail(id.value)
  } catch {
    article.value = null // 不存在/已下架 → 显示未找到
  }
}
onMounted(() => {
  store.hydrate()
  loadDetail()
})
// 侧栏跳转同页不同 id 时重新拉取，并回顶
watch(id, () => {
  loadDetail()
  window.scrollTo({ top: 0 })
})

const catName = computed(() => (article.value ? store.categoryName[article.value.categoryId] ?? '' : ''))

// 上一篇 / 下一篇：同分类已发布文章按发布顺序（id 升序）排列，取当前文章的相邻项。
// 「上一篇」= 更早发布（id 更小），「下一篇」= 更新发布（id 更大）。
const siblings = computed(() => {
  if (!article.value) return [] as Article[]
  const catId = article.value.categoryId
  return store.published
    .filter((a) => a.categoryId === catId)
    .slice()
    .sort((a, b) => a.id - b.id)
})
const curIndex = computed(() =>
  article.value ? siblings.value.findIndex((a) => a.id === article.value!.id) : -1,
)
const prevArticle = computed(() =>
  curIndex.value > 0 ? siblings.value[curIndex.value - 1] : null,
)
const nextArticle = computed(() =>
  curIndex.value >= 0 && curIndex.value < siblings.value.length - 1
    ? siblings.value[curIndex.value + 1]
    : null,
)

function goCategory() {
  if (article.value) router.push(`/news/category/${article.value.categoryId}`)
}
function goTag(name: string) {
  router.push({ path: '/news', query: { tag: name } })
}
</script>

<template>
  <div>
    <!-- 未找到 -->
    <section v-if="!article" class="mx-auto max-w-3xl px-4 py-32 text-center lg:px-8">
      <h1 class="text-2xl font-bold">文章不存在或已下架</h1>
      <p class="mt-3 text-muted-foreground">该文章可能已被删除，返回首页查看其它资讯。</p>
      <button
        class="mt-8 inline-flex items-center gap-2 rounded-lg bg-primary px-5 py-2.5 text-sm font-medium text-white transition-colors hover:bg-[#0052cc]"
        @click="router.push('/')"
      ><ArrowLeft class="size-4" />返回首页</button>
    </section>

    <template v-else>
      <!-- 蓝色 hero（与资讯列表页统一）-->
      <section class="relative overflow-hidden bg-gradient-to-r from-[#1a6dff] to-[#155bd4]">
        <div class="mx-auto flex max-w-7xl items-center px-4 py-12 lg:px-8">
          <div class="flex items-baseline gap-3">
            <span class="text-3xl font-extrabold tracking-wide text-white sm:text-4xl">官方动态</span>
            <span class="text-3xl font-extrabold tracking-wide text-white/25 sm:text-4xl">news</span>
          </div>
        </div>
        <Megaphone class="pointer-events-none absolute -right-2 top-1/2 hidden size-40 -translate-y-1/2 -rotate-12 text-white/10 lg:block" />
      </section>

      <!-- 正文 + 侧栏（浅灰页面底，衬出白卡对比，与列表页统一）-->
      <section class="bg-[#f5f5f5]">
        <div class="mx-auto max-w-7xl px-4 py-8 lg:px-8">
        <!-- 面包屑 -->
        <div class="flex items-center gap-1.5 text-sm text-muted-foreground">
          <button class="transition-colors hover:text-primary" @click="router.push('/')">首页</button>
          <ChevronRight class="size-3.5" />
          <button class="transition-colors hover:text-primary" @click="router.push('/news')">官方动态</button>
          <ChevronRight class="size-3.5" />
          <button class="text-foreground transition-colors hover:text-primary" @click="goCategory">{{ catName }}</button>
        </div>

        <div class="mt-4 grid gap-4 lg:grid-cols-[1fr_385px]">
          <!-- 正文主体（白色容器）-->
          <div class="min-w-0 bg-background p-6 shadow-[0_1px_2px_rgba(0,0,0,0.04)] lg:p-8">
            <h1 v-reveal class="text-2xl font-bold leading-snug tracking-tight sm:text-3xl">{{ article.title }}</h1>
            <div v-reveal class="mt-4 flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-muted-foreground">
              <span class="inline-flex items-center gap-1.5"><CalendarDays class="size-4" />{{ article.addtime }}</span>
              <span class="inline-flex items-center gap-1.5 tabular-nums"><Eye class="size-4" />{{ article.views }} 次浏览</span>
              <button class="inline-flex items-center rounded-full border border-border bg-background px-2.5 py-0.5 text-xs transition-colors hover:border-primary/40 hover:text-primary" @click="goCategory">{{ catName }}</button>
            </div>

            <!-- 正文 -->
            <article v-reveal class="news-content mt-6" v-html="article.content" />

            <!-- 文章标签 -->
            <div v-if="article.tags?.length" class="mt-10 flex flex-wrap items-center gap-2">
              <span class="text-sm text-muted-foreground">标签：</span>
              <button
                v-for="t in article.tags"
                :key="t"
                class="rounded-sm bg-muted/50 px-2.5 py-1 text-xs text-muted-foreground transition-colors hover:bg-primary/[0.08] hover:text-primary"
                @click="goTag(t)"
              >{{ t }}</button>
            </div>

            <!-- 上一篇 / 下一篇（同分类相邻文章）-->
            <div v-if="prevArticle || nextArticle" class="mt-8 flex flex-col gap-2 bg-muted/40 p-4 sm:flex-row sm:items-center">
              <button
                v-if="prevArticle"
                class="group flex min-w-0 flex-1 items-center gap-2 text-left"
                @click="router.push(`/news/${prevArticle.id}`)"
              >
                <ChevronLeft class="size-4 shrink-0 text-muted-foreground/60 transition-colors group-hover:text-primary" />
                <span class="shrink-0 text-sm text-muted-foreground">上一篇</span>
                <span class="truncate text-sm text-foreground/80 transition-colors group-hover:text-primary">{{ prevArticle.title }}</span>
              </button>
              <div v-else class="flex-1" />

              <div class="hidden h-6 w-px bg-border/60 sm:block" />

              <button
                v-if="nextArticle"
                class="group flex min-w-0 flex-1 items-center justify-end gap-2 text-right"
                @click="router.push(`/news/${nextArticle.id}`)"
              >
                <span class="truncate text-sm text-foreground/80 transition-colors group-hover:text-primary">{{ nextArticle.title }}</span>
                <span class="shrink-0 text-sm text-muted-foreground">下一篇</span>
                <ChevronRight class="size-4 shrink-0 text-muted-foreground/60 transition-colors group-hover:text-primary" />
              </button>
              <div v-else class="flex-1" />
            </div>

            <!-- 返回 -->
            <div class="mt-8">
              <button
                class="inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-primary"
                @click="router.push('/news')"
              ><ArrowLeft class="size-4" />返回资讯列表</button>
            </div>
          </div>

          <!-- 右侧栏：官方公众号二维码 + 热门资讯 + 热门标签云 -->
          <NewsSidebar variant="article" />
        </div>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
/* 正文渲染样式（与后台 RichEditor 编辑区保持一致的排版）*/
.news-content {
  font-size: 15px;
  line-height: 1.85;
  color: var(--foreground);
}
.news-content :deep(p) {
  margin: 0.85em 0;
}
/* 多级标题：H1-H4 四级各不同样式（蓝竖块无阴影），与后台编辑器一致 */
.news-content :deep(h1) {
  margin: 1.3em 0 0.75em;
  padding: 0.4em 0 0.4em 0.8em;
  border-left: 4px solid var(--primary);
  background: var(--muted);
  font-size: 1.7em;
  font-weight: 700;
  line-height: 1.4;
}
.news-content :deep(h2) {
  margin: 1.2em 0 0.65em;
  padding: 0.3em 0 0.3em 0.75em;
  border-left: 4px solid var(--primary);
  background: color-mix(in oklch, var(--muted) 55%, transparent);
  font-size: 1.4em;
  font-weight: 700;
  line-height: 1.4;
}
.news-content :deep(h3) {
  margin: 1.1em 0 0.55em;
  padding-left: 0.7em;
  border-left: 3px solid var(--primary);
  font-size: 1.2em;
  font-weight: 600;
  line-height: 1.4;
}
.news-content :deep(h4) {
  position: relative;
  margin: 1em 0 0.5em;
  padding-left: 0.95em;
  font-size: 1.08em;
  font-weight: 600;
}
.news-content :deep(h4)::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0.6em;
  width: 6px;
  height: 6px;
  border-radius: 9999px;
  background: var(--primary);
}
.news-content :deep(ul),
.news-content :deep(ol) {
  margin: 0.85em 0;
  padding-left: 1.6em;
}
.news-content :deep(ul) {
  list-style: disc;
}
.news-content :deep(ol) {
  list-style: decimal;
}
.news-content :deep(li) {
  margin: 0.3em 0;
}
.news-content :deep(blockquote) {
  margin: 1em 0;
  border-left: 3px solid var(--primary);
  padding-left: 1em;
  color: var(--muted-foreground);
}
.news-content :deep(code) {
  border-radius: 3px;
  background: var(--muted);
  padding: 0.1em 0.35em;
  font-size: 0.9em;
}
.news-content :deep(pre) {
  margin: 1em 0;
  border-radius: 6px;
  background: #1e293b;
  padding: 0.85em 1em;
  color: #e2e8f0;
  overflow-x: auto;
}
.news-content :deep(pre code) {
  background: transparent;
  padding: 0;
  color: inherit;
}
.news-content :deep(a) {
  color: var(--primary);
  text-decoration: underline;
}
.news-content :deep(img) {
  max-width: 100%;
  border-radius: 8px;
}
.news-content :deep(hr) {
  margin: 1.5em 0;
  border: none;
  border-top: 1px solid var(--border);
}
/* 编辑器文字颜色（tiptap Color 输出内联 style 的 <span>），确保正文如实呈现 */
.news-content :deep(span[style*='color']) {
  color: inherit; /* 占位：内联 style 优先级更高，实际取 span 自身 color；此规则仅防被祖先规则误伤 */
}
</style>
