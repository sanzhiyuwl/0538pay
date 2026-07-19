<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
import { Zap } from 'lucide-vue-next'
import { useSiteStore } from '@/stores/site'
import SiteHeader from '@/components/site/SiteHeader.vue'

// 站点配置来自后台「网站设置」，实时联动（config 为 reactive，模板直接读即响应）
const site = useSiteStore().config

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
    <footer class="relative bg-content">
      <!-- 顶部柔和渐变分隔线（替代生硬 border） -->
      <div class="h-px w-full bg-gradient-to-r from-transparent via-border to-transparent" />
      <div class="mx-auto max-w-7xl px-4 py-14 lg:px-6">
        <div class="grid grid-cols-2 gap-8 md:grid-cols-4 md:gap-12">
          <!-- 品牌列（右侧竖直渐变分隔线，仅桌面显示）-->
          <div class="relative col-span-2 md:col-span-1 md:pr-12 md:after:absolute md:after:right-0 md:after:top-1 md:after:h-[85%] md:after:w-px md:after:bg-gradient-to-b md:after:from-transparent md:after:via-border md:after:to-transparent">
            <div class="flex items-center gap-2">
              <div class="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                <Zap class="size-[18px]" />
              </div>
              <span class="text-lg font-bold tracking-tight">0538<span class="text-primary">Pay</span></span>
            </div>
            <p class="mt-3 text-sm leading-relaxed text-muted-foreground">
              专业的聚合支付服务平台，支持多渠道收款、实时到账、开放 API 对接。
            </p>
            <div class="mt-4 text-xs text-muted-foreground">
              客服 QQ：{{ site.qq }} · {{ site.email }}
            </div>
          </div>
          <!-- 链接列 -->
          <div v-for="col in footerCols" :key="col.title">
            <div class="text-sm font-semibold">{{ col.title }}</div>
            <ul class="mt-4 space-y-2.5">
              <li v-for="l in col.links" :key="l.label">
                <RouterLink :to="l.to" class="text-sm text-muted-foreground transition-colors hover:text-primary">{{ l.label }}</RouterLink>
              </li>
            </ul>
          </div>
        </div>

        <!-- 合规声明（纯文字）-->
        <p v-if="site.disclaimer" class="mt-10 border-t border-border/50 pt-6 text-xs leading-relaxed text-muted-foreground">
          {{ site.disclaimer }}
        </p>

        <!-- 底部版权 + 备案（读取后台站点配置）-->
        <div class="mt-6 flex flex-col items-center justify-between gap-3 text-xs text-muted-foreground sm:flex-row">
          <span>{{ site.copyright }} · {{ site.company }}</span>
          <div class="flex flex-wrap items-center gap-x-4 gap-y-1">
            <a :href="site.copyrightLink" target="_blank" rel="noopener" class="transition-colors hover:text-primary">{{ site.icp }}</a>
            <a :href="site.policeLink" target="_blank" rel="noopener" class="flex items-center gap-1 transition-colors hover:text-primary">{{ site.police }}</a>
          </div>
        </div>
      </div>
    </footer>
  </div>
</template>
