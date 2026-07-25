<script setup lang="ts">
/**
 * 品牌 logo 渲染器。支付品牌族 → 对应图标：
 *  - 微信支付 / 支付宝 / QQ钱包：simple-icons 官方矢量品牌 logo（单 path，24×24 viewBox）；
 *  - 彩虹易支付 / 富友支付 / V免签 等无官方矢量的：回退到贴合语义的 lucide 图标。
 * 统一暴露 brand（品牌名）即可，颜色由外层 class（品牌主色）控制，fill/stroke 用 currentColor。
 */
import { computed, type Component } from 'vue'
import { siAlipay, siWechat, siQq } from 'simple-icons'
import { Landmark, QrCode, Boxes, ShieldCheck } from 'lucide-vue-next'

const props = defineProps<{ brand: string }>()

// 官方矢量品牌（simple-icons 单 path）。key 用品牌族中文名。
const siPaths: Record<string, string> = {
  微信支付: siWechat.path,
  支付宝: siAlipay.path,
  QQ钱包: siQq.path,
}
// 无官方矢量 → 贴合语义的 lucide 回退图标。
const lucideFallback: Record<string, Component> = {
  彩虹易支付: Boxes, // 聚合/多渠道
  富友支付: Landmark, // 银行系收单
  V免签: QrCode, // 监控回调收款码
}

const siPath = computed(() => siPaths[props.brand])
const fallbackComp = computed(() => lucideFallback[props.brand] ?? ShieldCheck)
</script>

<template>
  <!-- 官方矢量品牌：直接画 simple-icons path（fill 用 currentColor 继承品牌主色）-->
  <svg
    v-if="siPath"
    viewBox="0 0 24 24"
    fill="currentColor"
    class="size-full"
    aria-hidden="true"
  >
    <path :d="siPath" />
  </svg>
  <!-- 回退：lucide 组件（stroke 用 currentColor）-->
  <component :is="fallbackComp" v-else class="size-full" />
</template>
