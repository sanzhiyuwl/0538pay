<script setup lang="ts">
/**
 * 官网资讯侧栏（列表页 ClassicNewsList + 详情页 ClassicNews 共用）。
 * 1:1 对齐 CRMEB 参考图右侧栏：
 * - 热门资讯：顶部大封面轮播（自动播放 + 圆点指示器），下方蓝点标题列表。
 * - 热门标签云：浅灰底胶囊，hover 变主题蓝，聚合文章 tags。
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { QrCode } from 'lucide-vue-next'
import { useArticlesStore } from '@/stores/articles'
import { useSiteStore } from '@/stores/site'

const props = withDefaults(
  defineProps<{
    activeTag?: string
    /** list=列表页(顶部轮播)；article=详情页(顶部换成官方公众号二维码卡) */
    variant?: 'list' | 'article'
  }>(),
  { activeTag: '', variant: 'list' },
)
const router = useRouter()
const store = useArticlesStore()
const siteStore = useSiteStore()

// 官方公众号二维码图（来自网站设置 mpQrcode，未配置则显示占位框）
const qrImg = computed(() => (siteStore.config as Record<string, unknown>).mpQrcode as string || '')

// 热门资讯（列表页）：浏览量 Top8
const hot = computed(() => store.hotArticles(8))
const slides = computed(() => hot.value.slice(0, 3)) // 列表页轮播

// 近期文章（详情页）：已发布文章按发布日期降序取前 6
const recent = computed(() =>
  [...store.published]
    .sort((a, b) => (a.addtime < b.addtime ? 1 : a.addtime > b.addtime ? -1 : b.id - a.id))
    .slice(0, 6),
)

// 圆点列表：列表页取热门 Top4-8；详情页取近期文章
const listRest = computed(() =>
  props.variant === 'article' ? recent.value : hot.value.slice(3, 8),
)

// 热门标签云
const tags = computed(() => store.hotTags)

// ===== 轮播状态 =====
const current = ref(0)
let timer: ReturnType<typeof setInterval> | null = null

function clampCurrent() {
  if (current.value > slides.value.length - 1) current.value = 0
}
watch(slides, clampCurrent)

function start() {
  stop()
  if (props.variant === 'article') return // 详情页无轮播
  if (slides.value.length > 1) {
    timer = setInterval(() => {
      current.value = (current.value + 1) % slides.value.length
    }, 4500)
  }
}
function stop() {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}
function goSlide(i: number) {
  current.value = i
  start() // 手动切换后重置计时，避免立刻跳走
}
onMounted(start)
onBeforeUnmount(stop)

function goNews(id: number) {
  router.push(`/news/${id}`)
}
function goTag(name: string) {
  router.push({ path: '/news', query: { tag: name } })
}
</script>

<template>
  <aside class="space-y-3">
    <!-- ===== 官方公众号二维码（仅详情页 article 模式）===== -->
    <div v-if="variant === 'article'" class="bg-background p-5 shadow-[0_1px_2px_rgba(0,0,0,0.04)]">
      <div class="mb-5 flex items-center justify-center gap-3">
        <span class="h-px w-8 bg-gradient-to-r from-transparent to-primary" />
        <span class="text-base font-bold tracking-wide">官方公众号</span>
        <span class="h-px w-8 bg-gradient-to-l from-transparent to-primary" />
      </div>
      <div class="flex justify-center">
        <img
          v-if="qrImg"
          :src="qrImg"
          alt="官方公众号二维码"
          class="size-44 object-contain"
        />
        <!-- 无配置图时的占位（虚线框 + 提示）-->
        <div v-else class="flex size-44 flex-col items-center justify-center gap-2 border border-dashed border-border text-muted-foreground/60">
          <QrCode class="size-10" :stroke-width="1.25" />
          <span class="text-xs">公众号二维码</span>
        </div>
      </div>
      <p class="mt-4 text-center text-xs text-muted-foreground">扫码关注，获取最新资讯与活动</p>
    </div>

    <!-- ===== 热门资讯（列表页）/ 近期文章（详情页）===== -->
    <div v-if="listRest.length || slides.length" class="bg-background p-5 shadow-[0_1px_2px_rgba(0,0,0,0.04)]">
      <!-- 标题：居中 + 双侧蓝渐变短线 -->
      <div class="mb-4 flex items-center justify-center gap-3">
        <span class="h-px w-8 bg-gradient-to-r from-transparent to-primary" />
        <span class="text-base font-bold tracking-wide">{{ variant === 'article' ? '近期文章' : '热门资讯' }}</span>
        <span class="h-px w-8 bg-gradient-to-l from-transparent to-primary" />
      </div>

      <!-- 轮播（大封面 + 标题 + 摘要）：图片全直角，无任何圆弧（仅列表页）-->
      <div
        v-if="variant === 'list' && slides.length"
        class="select-none"
        @mouseenter="stop"
        @mouseleave="start"
      >
        <div class="overflow-hidden">
          <div
            class="flex transition-transform duration-500 ease-out"
            :style="{ transform: `translateX(-${current * 100}%)` }"
          >
            <button
              v-for="s in slides"
              :key="s.id"
              class="group w-full shrink-0 text-left"
              @click="goNews(s.id)"
            >
              <div class="relative h-[200px] w-full overflow-hidden">
                <img
                  v-if="s.cover"
                  :src="s.cover"
                  :alt="s.title"
                  class="size-full object-cover transition-transform duration-500 group-hover:scale-105"
                />
                <div v-else class="flex size-full items-center justify-center bg-gradient-to-br from-[#c0392b] to-[#e74c3c] text-lg font-bold text-white">
                  热门
                </div>
              </div>
              <div class="pt-3">
                <div class="truncate text-base font-semibold transition-colors group-hover:text-primary">{{ s.title }}</div>
                <p class="mt-2 line-clamp-2 text-sm leading-relaxed text-muted-foreground">{{ s.summary }}</p>
              </div>
            </button>
          </div>
        </div>

        <!-- 圆点指示器（选中项拉长为蓝色胶囊）-->
        <div v-if="slides.length > 1" class="mt-3 flex items-center justify-center gap-1.5">
          <button
            v-for="(s, i) in slides"
            :key="s.id"
            class="h-1.5 rounded-full transition-all duration-300"
            :class="current === i ? 'w-4 bg-primary' : 'w-1.5 bg-muted-foreground/25 hover:bg-muted-foreground/40'"
            :aria-label="`第 ${i + 1} 条`"
            @click="goSlide(i)"
          />
        </div>
      </div>

      <!-- 圆点列表（列表页在轮播下方带分隔；详情页作为唯一内容无分隔）-->
      <ul v-if="listRest.length" class="space-y-0.5" :class="variant === 'list' ? 'mt-4 border-t border-border/50 pt-3' : ''">
        <li v-for="a in listRest" :key="a.id">
          <button
            class="group flex w-full items-start gap-2.5 py-2 text-left"
            @click="goNews(a.id)"
          >
            <span class="mt-1.5 size-1.5 shrink-0 rounded-full bg-primary/70 transition-colors group-hover:bg-primary" />
            <span class="line-clamp-1 text-[15px] leading-relaxed text-foreground/80 transition-colors group-hover:text-primary">{{ a.title }}</span>
          </button>
        </li>
      </ul>
    </div>

    <!-- ===== 热门标签 ===== -->
    <div v-if="tags.length" class="bg-background p-5 shadow-[0_1px_2px_rgba(0,0,0,0.04)]">
      <div class="mb-4 flex items-center justify-center gap-3">
        <span class="h-px w-8 bg-gradient-to-r from-transparent to-primary" />
        <span class="text-base font-bold tracking-wide">热门标签</span>
        <span class="h-px w-8 bg-gradient-to-l from-transparent to-primary" />
      </div>
      <div class="flex flex-wrap gap-2">
        <button
          v-for="t in tags"
          :key="t.name"
          class="px-3 py-1.5 text-[13px] transition-colors"
          :class="props.activeTag === t.name
            ? 'bg-primary/[0.1] text-primary'
            : 'bg-muted text-foreground/70 hover:bg-primary/[0.08] hover:text-primary'"
          @click="goTag(t.name)"
        >{{ t.name }}</button>
      </div>
    </div>
  </aside>
</template>
