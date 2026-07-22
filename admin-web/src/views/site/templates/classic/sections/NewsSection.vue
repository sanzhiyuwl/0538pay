<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Badge } from '@/components/ui'
import type { SiteContent } from '@/lib/mock/site-content'
import { useArticlesStore } from '@/stores/articles'

defineProps<{ content: SiteContent; preview?: boolean }>()
const router = useRouter()

// 「最新动态」板块：按分类分列（头条 + 列表），来自文章 CMS
const articlesStore = useArticlesStore()
onMounted(() => articlesStore.hydrate())
const newsColumns = computed(() =>
  articlesStore.publishedByCategory
    .filter((col) => col.list.length > 0)
    .slice(0, 3) // 首页最多 3 列，与参考图一致
    .map((col) => ({
      category: col.category,
      headline: col.list[0], // 头条（置顶优先）
      rest: col.list.slice(1, 5), // 其余最多 4 条
    })),
)
function goNews(id: number) {
  router.push(`/news/${id}`)
}
// 发布日期取月-日（列表左侧显示）
function mmdd(date: string) {
  const parts = date.split('-')
  return parts.length === 3 ? `${parts[1]}-${parts[2]}` : date
}
</script>

<template>
  <!-- ===== 最新动态（3 列分类：深色头图 + 头条 + 列表，来自文章 CMS）===== -->
  <section v-if="newsColumns.length" id="news" class="scroll-mt-20 bg-background">
    <div class="mx-auto max-w-7xl px-4 pb-0 pt-24 lg:px-6">
      <!-- 标题居中 -->
      <div v-reveal class="mx-auto max-w-2xl text-center">
        <h2 class="text-3xl font-bold tracking-tight">{{ content.newsTitle }}</h2>
        <p class="mt-3 text-muted-foreground">{{ content.newsSubtitle }}</p>
      </div>

      <!-- 3 列 -->
      <div v-reveal class="mt-14 grid gap-6 lg:grid-cols-3">
        <div
          v-for="col in newsColumns"
          :key="col.category.id"
          class="flex flex-col bg-background shadow-[0_1px_3px_rgba(0,0,0,0.04),0_10px_30px_-14px_rgba(0,0,0,0.15)]"
        >
          <!-- 深色头图：英文小标题 + 中文大标题 + | 分类名 -->
          <div class="group relative aspect-[21/9] overflow-hidden">
            <img
              v-if="col.category.cover"
              :src="col.category.cover"
              :alt="col.category.name"
              class="size-full object-cover transition-transform duration-500 group-hover:scale-105"
            />
            <div v-else class="size-full bg-gradient-to-br from-[#3a4152] to-[#232834]" />
            <!-- 暗色遮罩保证白字可读 -->
            <div class="absolute inset-0 bg-gradient-to-t from-black/70 via-black/30 to-black/10" aria-hidden="true" />
            <!-- 文案 -->
            <div class="absolute inset-0 flex flex-col justify-end p-6">
              <div class="text-xs font-medium uppercase tracking-wider text-white/60">{{ col.category.enName }}</div>
              <div class="mt-1 flex items-baseline gap-2">
                <span class="text-2xl font-bold text-white">{{ col.category.name }}</span>
                <span class="text-sm text-white/50">|&nbsp;{{ col.category.name }}</span>
              </div>
            </div>
          </div>

          <!-- 头条：标题（带 new 标）+ 摘要 -->
          <div class="border-b border-border/60 px-6 py-6">
            <button
              class="group flex w-full items-center gap-2 text-left"
              @click="goNews(col.headline.id)"
            >
              <span class="truncate text-base font-semibold transition-colors group-hover:text-primary">{{ col.headline.title }}</span>
              <Badge v-if="col.headline.isNew" variant="destructive" class="shrink-0">new</Badge>
            </button>
            <p class="mt-3 line-clamp-3 text-sm leading-relaxed text-muted-foreground">{{ col.headline.summary }}</p>
          </div>

          <!-- 列表：日期 + 标题 -->
          <ul class="flex-1 px-6 py-4">
            <li v-for="a in col.rest" :key="a.id">
              <button
                class="group flex w-full items-center gap-3 py-2.5 text-left"
                @click="goNews(a.id)"
              >
                <span class="shrink-0 text-xs tabular-nums text-muted-foreground/70">{{ mmdd(a.addtime) }}</span>
                <span class="truncate text-sm text-foreground/80 transition-colors group-hover:text-primary">{{ a.title }}</span>
              </button>
            </li>
          </ul>
        </div>
      </div>
    </div>
  </section>
</template>
