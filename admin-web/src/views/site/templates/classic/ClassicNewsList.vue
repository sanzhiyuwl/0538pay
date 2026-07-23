<script setup lang="ts">
/**
 * 官网资讯聚合页。1:1 对齐参考图 CRMEB「官方动态 news」：
 * - 蓝色 hero（官方动态 + news 描边字 + 喇叭图标）
 * - 分类 tab（· 前缀蓝点选中）
 * - 左列横向封面卡列表（封面 + 标题 + 摘要 + meta：标签 pill / 日期 / 浏览量）
 * - 右侧栏（热门资讯 + 热门标签云，NewsSidebar 共用组件）
 * - 「查看更多」逐步加载
 *
 * 两个路由复用本组件：
 * - /news              全部资讯（顶部分类 tab，支持 ?tag= 标签筛选）
 * - /news/category/:id 单分类归档（tab 锁定该分类）
 */
import { computed, ref, watch, onMounted, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Megaphone, Clock, Eye, ChevronDown, X,
  LayoutGrid, Rocket, Building2, Newspaper, TrendingUp, BookOpen, Tag,
} from 'lucide-vue-next'
import { Badge } from '@/components/ui'
import { useArticlesStore } from '@/stores/articles'
import NewsSidebar from './sections/NewsSidebar.vue'

const route = useRoute()
const router = useRouter()
const store = useArticlesStore()
onMounted(() => store.hydrate())

// 路由模式：分类归档页锁定 categoryId；全部资讯页 tab 可切
const routeCatId = computed(() => (route.name === 'site-news-category' ? Number(route.params.id) : 0))
// 当前选中分类（0=全部）；进入分类页时以路由为准
const activeCat = ref<number>(routeCatId.value)
watch(routeCatId, (v) => { activeCat.value = v })

// ?tag= 标签筛选（仅全部资讯页）
const activeTag = computed(() => String(route.query.tag ?? '').trim())

// 分类 tab 列表（分类页不显示「全部」，锁定当前分类）
const tabs = computed(() => {
  const cats = [...store.categories].sort((a, b) => a.sort - b.sort)
  return routeCatId.value ? cats.filter((c) => c.id === routeCatId.value) : cats
})

// 当前分类对象（hero 副标题用）
const currentCat = computed(() => store.categories.find((c) => c.id === routeCatId.value))

// 筛选后的文章列表
const list = computed(() => {
  let arr = store.published
  if (activeCat.value) arr = arr.filter((a) => a.categoryId === activeCat.value)
  if (activeTag.value) arr = arr.filter((a) => (a.tags ?? []).includes(activeTag.value))
  return arr
})

// 「查看更多」分页（首屏 10 篇，对齐参考图）
const PAGE = 10
const shown = ref(PAGE)
watch([activeCat, activeTag, routeCatId], () => { shown.value = PAGE })
const visible = computed(() => list.value.slice(0, shown.value))
const hasMore = computed(() => list.value.length > shown.value)

function selectCat(id: number) {
  activeCat.value = id
  // 切分类时清掉标签筛选，避免交叉空结果
  if (activeTag.value) router.replace({ path: '/news' })
}
function clearTag() {
  router.replace({ path: '/news' })
}
function goNews(id: number) {
  router.push(`/news/${id}`)
}
function catName(id: number) {
  return store.categoryName[id] ?? ''
}

// 分类 tab 图标：按分类名关键词匹配，匹配不到用默认「报刊」图标
function catIcon(name: string): Component {
  const n = name.toLowerCase()
  if (/产品|动态|发布|上线|更新|版本/.test(n)) return Rocket
  if (/公司|企业|团队|集团/.test(n)) return Building2
  if (/行业|市场|趋势|观察|经济/.test(n)) return TrendingUp
  if (/教程|指南|帮助|文档|说明|知识/.test(n)) return BookOpen
  if (/活动|专题|标签/.test(n)) return Tag
  return Newspaper
}
</script>

<template>
  <div>
    <!-- 蓝色 hero -->
    <section class="relative overflow-hidden bg-gradient-to-r from-[#1a6dff] to-[#155bd4]">
      <div class="mx-auto flex max-w-7xl items-center px-4 py-14 lg:px-8">
        <div class="flex items-baseline gap-3">
          <h1 class="text-3xl font-extrabold tracking-wide text-white sm:text-4xl">
            {{ currentCat ? currentCat.name : '官方动态' }}
          </h1>
          <span class="text-3xl font-extrabold tracking-wide text-white/25 sm:text-4xl">
            {{ currentCat ? currentCat.enName : 'news' }}
          </span>
        </div>
      </div>
      <!-- 右侧喇叭装饰（大屏显示）-->
      <Megaphone class="pointer-events-none absolute -right-2 top-1/2 hidden size-40 -translate-y-1/2 -rotate-12 text-white/10 lg:block" />
    </section>

    <!-- 正文两栏（浅灰页面底，衬出白卡对比）-->
    <section class="bg-[#f5f5f5]">
      <div class="mx-auto grid max-w-7xl gap-4 px-4 py-8 lg:grid-cols-[1fr_385px] lg:px-8">
        <!-- 左列 -->
        <div class="min-w-0 space-y-1.5">
          <!-- 分类 tab（独立白卡）：选中=加粗深色 + 底部主题色下划线条，未选=灰 -->
          <div class="relative flex flex-wrap items-center justify-center gap-x-10 gap-y-1 bg-background px-6 shadow-[0_1px_2px_rgba(0,0,0,0.04)]">
            <button
              v-if="!routeCatId"
              class="group relative flex items-center gap-1.5 py-4 text-[15px] transition-colors"
              :class="activeCat === 0 ? 'font-bold text-foreground' : 'font-medium text-muted-foreground hover:text-foreground'"
              @click="selectCat(0)"
            >
              <LayoutGrid class="size-4 shrink-0 transition-colors" :class="activeCat === 0 ? 'text-primary' : 'text-muted-foreground/60 group-hover:text-foreground'" :stroke-width="2" />
              全部
              <span
                class="absolute inset-x-0 -bottom-px mx-auto h-[3px] rounded-full bg-primary transition-all duration-300"
                :class="activeCat === 0 ? 'w-6 opacity-100' : 'w-0 opacity-0'"
              />
            </button>
            <button
              v-for="t in tabs"
              :key="t.id"
              class="group relative flex items-center gap-1.5 py-4 text-[15px] transition-colors"
              :class="activeCat === t.id ? 'font-bold text-foreground' : 'font-medium text-muted-foreground hover:text-foreground'"
              @click="selectCat(t.id)"
            >
              <component :is="catIcon(t.name)" class="size-4 shrink-0 transition-colors" :class="activeCat === t.id ? 'text-primary' : 'text-muted-foreground/60 group-hover:text-foreground'" :stroke-width="2" />
              {{ t.name }}
              <span
                class="absolute inset-x-0 -bottom-px mx-auto h-[3px] rounded-full bg-primary transition-all duration-300"
                :class="activeCat === t.id ? 'w-6 opacity-100' : 'w-0 opacity-0'"
              />
            </button>

            <!-- 标签筛选提示（绝对定位到右侧，不影响 tab 居中）-->
            <button
              v-if="activeTag"
              class="absolute right-6 top-1/2 -translate-y-1/2 inline-flex items-center gap-1 rounded-sm bg-[#ff6a00] px-2.5 py-1 text-xs text-white"
              @click="clearTag"
            >标签：{{ activeTag }}<X class="size-3" /></button>
          </div>

          <!-- 列表：每篇一张白卡（近直角，平铺弱阴影）-->
          <article
            v-for="a in visible"
            :key="a.id"
            class="group flex cursor-pointer gap-5 bg-background p-4 shadow-[0_1px_2px_rgba(0,0,0,0.04)] transition-shadow hover:shadow-[0_2px_10px_-4px_rgba(0,0,0,0.12)]"
            @click="goNews(a.id)"
          >
            <!-- 封面（直角，250×140）-->
            <div class="relative h-[140px] w-[250px] shrink-0 overflow-hidden">
              <img
                v-if="a.cover"
                :src="a.cover"
                :alt="a.title"
                class="size-full object-cover transition-transform duration-500 group-hover:scale-105"
              />
              <div v-else class="flex size-full items-center justify-center bg-gradient-to-br from-[#3a4152] to-[#232834] text-xs font-medium text-white/55">
                {{ catName(a.categoryId) }}
              </div>
              <span v-if="a.isTop" class="absolute left-0 top-0 bg-[#e94b4b] px-2.5 py-1 text-[11px] font-medium leading-none text-white">置顶</span>
            </div>
            <!-- 正文 -->
            <div class="flex min-w-0 flex-1 flex-col justify-center py-0.5">
              <h3 class="flex items-center gap-2 text-xl font-semibold leading-snug">
                <span class="truncate transition-colors group-hover:text-primary">{{ a.title }}</span>
                <Badge v-if="a.isNew" variant="destructive" class="shrink-0">new</Badge>
              </h3>
              <p class="mt-2.5 line-clamp-2 text-sm leading-relaxed text-muted-foreground">{{ a.summary }}</p>
              <!-- 标签 pill（金橙渐变底白字，对齐参考图）-->
              <div v-if="a.tags?.length" class="mt-3">
                <span class="inline-block bg-gradient-to-r from-[#f9a825] to-[#f57f17] px-3 py-1 text-[13px] text-white">{{ a.tags[0] }}</span>
              </div>
              <!-- meta 行（日期 + 浏览量，空心细线图标）-->
              <div class="mt-3 flex flex-wrap items-center gap-x-5 gap-y-1.5 text-[13px] text-muted-foreground/60">
                <span class="inline-flex items-center gap-1.5"><Clock class="size-4" :stroke-width="1.5" />{{ a.addtime }}</span>
                <span class="inline-flex items-center gap-1.5 tabular-nums"><Eye class="size-4" :stroke-width="1.5" />{{ a.views }}</span>
              </div>
            </div>
          </article>

          <!-- 空态 -->
          <div v-if="!visible.length" class="bg-background py-24 text-center shadow-[0_1px_2px_rgba(0,0,0,0.04)]">
            <p class="text-muted-foreground">该分类下暂无资讯</p>
          </div>

          <!-- 查看更多 -->
          <div v-if="hasMore" class="flex justify-center pt-3">
            <button
              class="inline-flex items-center gap-1.5 rounded-sm border border-border bg-background px-8 py-2.5 text-sm text-muted-foreground transition-colors hover:border-primary/40 hover:text-primary"
              @click="shown += PAGE"
            >查看更多<ChevronDown class="size-4" /></button>
          </div>
        </div>

        <!-- 右侧栏 -->
        <NewsSidebar :active-tag="activeTag" />
      </div>
    </section>
  </div>
</template>
