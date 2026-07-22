<script setup lang="ts">
/**
 * 官网文章详情页。/news/:id 读取文章正文（tiptap HTML）渲染。
 * 侧栏展示同分类的其它文章，便于跳转。
 */
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Eye, CalendarDays, ChevronRight } from 'lucide-vue-next'
import { useArticlesStore } from '@/stores/articles'

const route = useRoute()
const router = useRouter()
const store = useArticlesStore()

onMounted(() => store.hydrate())

const id = computed(() => Number(route.params.id))
const article = computed(() => store.getArticle(id.value))
const catName = computed(() => (article.value ? store.categoryName[article.value.categoryId] ?? '' : ''))

// 同分类的其它已发布文章（最多 6 篇）
const related = computed(() => {
  if (!article.value) return []
  return store.articles
    .filter((a) => a.categoryId === article.value!.categoryId && a.status === 1 && a.id !== article.value!.id)
    .sort((a, b) => a.sort - b.sort)
    .slice(0, 6)
})

function goDetail(nid: number) {
  router.push(`/news/${nid}`)
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
      <!-- 页头 -->
      <section class="site-surface border-b border-border">
        <div class="mx-auto max-w-4xl px-4 py-16 lg:px-8">
          <!-- 面包屑 -->
          <div class="flex items-center gap-1.5 text-sm text-muted-foreground">
            <button class="transition-colors hover:text-primary" @click="router.push('/')">首页</button>
            <ChevronRight class="size-3.5" />
            <span>最新动态</span>
            <ChevronRight class="size-3.5" />
            <span class="text-foreground">{{ catName }}</span>
          </div>
          <h1 v-reveal class="mt-5 text-3xl font-bold leading-snug tracking-tight sm:text-4xl">{{ article.title }}</h1>
          <div v-reveal class="mt-5 flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-muted-foreground">
            <span class="inline-flex items-center gap-1.5"><CalendarDays class="size-4" />{{ article.addtime }}</span>
            <span class="inline-flex items-center gap-1.5"><Eye class="size-4" />{{ article.views }} 次浏览</span>
            <span class="inline-flex items-center rounded-full border border-border bg-background px-2.5 py-0.5 text-xs">{{ catName }}</span>
          </div>
        </div>
      </section>

      <!-- 正文 + 侧栏 -->
      <section class="mx-auto max-w-4xl px-4 py-14 lg:px-8">
        <div class="grid gap-10 lg:grid-cols-[1fr_260px]">
          <!-- 正文 -->
          <article v-reveal class="news-content min-w-0" v-html="article.content" />

          <!-- 侧栏：同分类其它文章 -->
          <aside v-if="related.length" class="lg:border-l lg:border-border lg:pl-8">
            <div class="text-sm font-semibold">{{ catName }} · 更多</div>
            <ul class="mt-4 space-y-3">
              <li v-for="r in related" :key="r.id">
                <button
                  class="group flex w-full items-start gap-2 text-left text-sm leading-relaxed text-muted-foreground transition-colors hover:text-primary"
                  @click="goDetail(r.id)"
                >
                  <span class="mt-1.5 size-1 shrink-0 rounded-full bg-border transition-colors group-hover:bg-primary" />
                  <span class="line-clamp-2">{{ r.title }}</span>
                </button>
              </li>
            </ul>
          </aside>
        </div>

        <!-- 返回 -->
        <div class="mt-12 border-t border-border pt-8">
          <button
            class="inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-primary"
            @click="router.push('/')"
          ><ArrowLeft class="size-4" />返回首页</button>
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
.news-content :deep(h2) {
  margin: 1.4em 0 0.6em;
  font-size: 1.4em;
  font-weight: 700;
}
.news-content :deep(h3) {
  margin: 1.2em 0 0.5em;
  font-size: 1.2em;
  font-weight: 600;
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
</style>
