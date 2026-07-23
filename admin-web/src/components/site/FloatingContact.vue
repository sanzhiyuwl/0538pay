<script setup lang="ts">
/**
 * 官网首页右侧悬浮联系栏。竖排图标：在线客服(QQ) / 公众号(悬浮出二维码) / 邮箱 / 返回顶部。
 * 联系信息来自网站设置(useSiteStore)，实时联动。返回顶部按钮在页面滚动一定距离后才出现。
 */
import { computed, onMounted, onBeforeUnmount, ref } from 'vue'
import { Headset, QrCode, Mail, ChevronUp } from 'lucide-vue-next'
import { useSiteStore } from '@/stores/site'

const siteStore = useSiteStore()
const site = siteStore.config
// 悬浮栏配置（网站设置「悬浮栏」tab），旧缓存无该字段时默认开启
const cfg = site as Record<string, unknown>
const barOn = computed(() => cfg.floatBarOn !== false)
const showKf = computed(() => cfg.floatKf !== false)
const showQr = computed(() => cfg.floatQr !== false)
const showMail = computed(() => cfg.floatMail !== false)
const showTopItem = computed(() => cfg.floatTop !== false)
// 公众号二维码图（网站设置 mpQrcode，未配置则显示占位框）
const qrImg = computed(() => (cfg.mpQrcode as string) || '')

// 悬浮展开的项（hover 时右侧弹出面板）
const active = ref<'kf' | 'qr' | 'mail' | null>(null)

// 返回顶部：滚动超过一屏才显示
const showTop = ref(false)
function onScroll() {
  showTop.value = window.scrollY > 400
}
function backTop() {
  window.scrollTo({ top: 0, behavior: 'smooth' })
}
onMounted(() => {
  window.addEventListener('scroll', onScroll, { passive: true })
  onScroll()
})
onBeforeUnmount(() => window.removeEventListener('scroll', onScroll))

// QQ 临时会话链接
const qqLink = computed(() => `https://wpa.qq.com/msgrd?v=3&uin=${site.qq}&site=qq&menu=yes`)
</script>

<template>
  <div v-if="barOn" class="fixed right-4 top-1/2 z-40 hidden -translate-y-1/2 lg:block">
    <div class="flex flex-col divide-y divide-border/60 overflow-hidden rounded-xl border border-border/70 bg-white shadow-[0_8px_30px_rgba(0,0,0,0.08)]">
      <!-- 在线客服 QQ -->
      <a
        v-if="showKf"
        :href="qqLink"
        target="_blank"
        rel="noopener"
        class="group relative flex size-14 flex-col items-center justify-center gap-0.5 text-muted-foreground transition-colors hover:bg-primary/[0.06] hover:text-primary"
        @mouseenter="active = 'kf'"
        @mouseleave="active = null"
      >
        <Headset class="size-5" />
        <span class="text-[11px] leading-none">客服</span>
        <!-- 悬浮面板：QQ 号 -->
        <div
          v-show="active === 'kf'"
          class="absolute right-full top-1/2 mr-3 -translate-y-1/2 whitespace-nowrap rounded-lg border border-border/70 bg-white px-4 py-3 text-center shadow-lg"
        >
          <div class="text-xs text-muted-foreground">在线客服 QQ</div>
          <div class="mt-1 text-sm font-semibold text-foreground">{{ site.qq }}</div>
        </div>
      </a>

      <!-- 公众号二维码 -->
      <button
        v-if="showQr"
        type="button"
        class="group relative flex size-14 flex-col items-center justify-center gap-0.5 text-muted-foreground transition-colors hover:bg-primary/[0.06] hover:text-primary"
        @mouseenter="active = 'qr'"
        @mouseleave="active = null"
      >
        <QrCode class="size-5" />
        <span class="text-[11px] leading-none">公众号</span>
        <!-- 悬浮面板：二维码 -->
        <div
          v-show="active === 'qr'"
          class="absolute right-full top-1/2 mr-3 -translate-y-1/2 whitespace-nowrap rounded-lg border border-border/70 bg-white p-3 text-center shadow-lg"
        >
          <img v-if="qrImg" :src="qrImg" alt="官方公众号" class="size-32 rounded" />
          <div
            v-else
            class="flex size-32 flex-col items-center justify-center gap-1.5 rounded border border-dashed border-border text-muted-foreground/60"
          >
            <QrCode class="size-7" />
            <span class="text-[11px]">二维码未配置</span>
          </div>
          <div class="mt-2 text-xs text-muted-foreground">扫码关注官方公众号</div>
        </div>
      </button>

      <!-- 邮箱 -->
      <a
        v-if="showMail"
        :href="`mailto:${site.email}`"
        class="group relative flex size-14 flex-col items-center justify-center gap-0.5 text-muted-foreground transition-colors hover:bg-primary/[0.06] hover:text-primary"
        @mouseenter="active = 'mail'"
        @mouseleave="active = null"
      >
        <Mail class="size-5" />
        <span class="text-[11px] leading-none">邮箱</span>
        <!-- 悬浮面板：邮箱地址 -->
        <div
          v-show="active === 'mail'"
          class="absolute right-full top-1/2 mr-3 -translate-y-1/2 whitespace-nowrap rounded-lg border border-border/70 bg-white px-4 py-3 text-center shadow-lg"
        >
          <div class="text-xs text-muted-foreground">商务合作邮箱</div>
          <div class="mt-1 text-sm font-semibold text-foreground">{{ site.email }}</div>
        </div>
      </a>

      <!-- 返回顶部：配置开启且滚动后出现 -->
      <button
        v-if="showTopItem && showTop"
        type="button"
        class="flex size-14 flex-col items-center justify-center gap-0.5 text-muted-foreground transition-colors hover:bg-primary/[0.06] hover:text-primary"
        @click="backTop"
      >
        <ChevronUp class="size-5" />
        <span class="text-[11px] leading-none">顶部</span>
      </button>
    </div>
  </div>
</template>
