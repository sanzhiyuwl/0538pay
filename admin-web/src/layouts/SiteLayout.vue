<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterLink, RouterView } from 'vue-router'
import { Zap, MessageCircle, Mail, ShieldCheck } from 'lucide-vue-next'
import { useSiteStore } from '@/stores/site'
import SiteHeader from '@/components/site/SiteHeader.vue'

// 站点配置来自后台「网站设置」，实时联动（config 为 reactive，模板直接读即响应）
const store = useSiteStore()
const site = store.config
// 官网加载时从后端拉取最新网站设置（本地缓存先渲染，后端到达后覆盖 + 刷新 SEO）
onMounted(() => { store.hydrate() })

// 页脚链接列
const footerCols = [
  { title: '产品', links: [{ label: '聚合支付', to: '/#features' }, { label: '费率方案', to: '/#pricing' }, { label: '商户中心', to: '/m/login' }, { label: '控制台', to: '/console' }] },
  { title: '开发者', links: [{ label: '接入文档', to: '/docs' }, { label: '错误码', to: '/docs?p=errcode' }, { label: '常见问题', to: '/docs?p=faq' }] },
  { title: '关于', links: [{ label: '关于我们', to: '/about' }, { label: '服务协议', to: '/agreement' }, { label: '联系客服', to: '/about' }] },
]
</script>

<template>
  <div class="relative flex min-h-screen flex-col bg-background">
    <!-- 官网顶部导航（告示条 + 吸顶导航 + 移动菜单，抽为共享组件）-->
    <SiteHeader />

    <!-- 内容 -->
    <main class="flex-1 bg-content">
      <RouterView />
    </main>

    <!-- 大页脚 -->
    <footer class="relative bg-white">
      <!-- 顶部柔和渐变分隔线 -->
      <div class="h-px w-full bg-gradient-to-r from-transparent via-border to-transparent" />
      <div class="mx-auto max-w-7xl px-4 py-16 lg:px-6">
        <div class="grid grid-cols-2 gap-x-8 gap-y-10 md:grid-cols-[1.5fr_1fr_1fr_1fr] md:gap-12">
          <!-- 品牌列（右侧竖直渐变分隔线，仅桌面显示）-->
          <div class="relative col-span-2 md:col-span-1 md:pr-12 md:after:absolute md:after:right-0 md:after:top-1 md:after:h-[85%] md:after:w-px md:after:bg-gradient-to-b md:after:from-transparent md:after:via-border md:after:to-transparent">
            <div class="flex items-center gap-2">
              <div class="flex size-9 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-lg shadow-primary/25">
                <Zap class="size-5" />
              </div>
              <span class="text-lg font-bold tracking-tight text-foreground">0538<span class="text-primary">Pay</span></span>
            </div>
            <p class="mt-4 max-w-xs text-sm leading-relaxed text-muted-foreground">
              专业的聚合支付服务平台，支持多渠道收款、实时到账、开放 API 对接。
            </p>
            <!-- 联系方式：图标 + 文字 -->
            <div class="mt-5 space-y-2.5">
              <div class="flex items-center gap-2.5 text-sm text-muted-foreground">
                <MessageCircle class="size-4 shrink-0 text-muted-foreground/60" />
                <span>客服 QQ：{{ site.qq }}</span>
              </div>
              <a :href="`mailto:${site.email}`" class="flex items-center gap-2.5 text-sm text-muted-foreground transition-colors hover:text-primary">
                <Mail class="size-4 shrink-0 text-muted-foreground/60" />
                <span>{{ site.email }}</span>
              </a>
            </div>
          </div>
          <!-- 链接列 -->
          <div v-for="col in footerCols" :key="col.title">
            <div class="text-sm font-semibold text-foreground">{{ col.title }}</div>
            <ul class="mt-4 space-y-3">
              <li v-for="l in col.links" :key="l.label">
                <RouterLink :to="l.to" class="text-sm text-muted-foreground transition-colors hover:text-primary">{{ l.label }}</RouterLink>
              </li>
            </ul>
          </div>
        </div>

        <!-- 合规声明：一行纯文字，盾牌图标引导 -->
        <p v-if="site.disclaimer" class="mt-12 flex items-start gap-2 text-xs leading-relaxed text-muted-foreground">
          <ShieldCheck class="mt-0.5 size-3.5 shrink-0 text-muted-foreground/60" />
          <span>{{ site.disclaimer }}</span>
        </p>

        <!-- 底部：备案徽标 + 版权同一行（徽标靠左、版权靠右），仅一条分隔线 -->
        <div class="mt-6 flex flex-col items-start justify-between gap-y-4 border-t border-border/60 pt-6 lg:flex-row lg:items-center">
          <div class="flex flex-wrap items-center gap-x-6 gap-y-3">
            <a :href="site.policeLink" target="_blank" rel="noopener" class="flex items-center gap-1.5 text-xs text-muted-foreground transition-colors hover:text-primary">
              <img src="/home/gongan.png" alt="公安备案" class="size-4" />
              <span>{{ site.police }}</span>
            </a>
            <a :href="site.copyrightLink" target="_blank" rel="noopener" class="flex items-center gap-1.5 text-xs text-muted-foreground transition-colors hover:text-primary">
              <img src="/home/icp.png" alt="ICP备案" class="h-4 w-auto" />
              <span>{{ site.icp }}</span>
            </a>
            <a v-if="site.qingsuan" :href="site.qingsuanLink" target="_blank" rel="noopener" class="flex items-center gap-1.5 text-xs text-muted-foreground transition-colors hover:text-primary">
              <img src="/home/qingsuan.png" alt="清算协会备案" class="size-4 object-contain" />
              <span>{{ site.qingsuan }}</span>
            </a>
            <a :href="site.marketLink" target="_blank" rel="noopener" class="flex items-center gap-1.5 text-xs text-muted-foreground transition-colors hover:text-primary">
              <img src="/home/gongshang.png" alt="工商备案" class="size-4" />
              <span>工商信息公示</span>
            </a>
          </div>
          <div class="shrink-0 text-xs text-muted-foreground/70">{{ site.copyright }} · {{ site.company }}</div>
        </div>
      </div>
    </footer>
  </div>
</template>
