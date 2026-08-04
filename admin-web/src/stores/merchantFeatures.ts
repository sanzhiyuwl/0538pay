import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchMerchantFeatures, type MerchantFeatures } from '@/lib/api/merchantCenter'

/**
 * 商户端全局功能开关（由平台后台控制）。
 * 商户布局挂载时 load 一次，用于过滤导航 / 守卫页面。
 * 目前含：deposit（保证金门槛开关 user_deposit，关闭时隐藏保证金入口与页面）。
 */
export const useMerchantFeaturesStore = defineStore('merchantFeatures', () => {
  // 默认 true，避免开关拉取完成前误隐藏入口造成闪烁；拉取失败也按开启处理（不误伤已有功能）。
  const features = ref<MerchantFeatures>({ deposit: true })
  const loaded = ref(false)

  async function load() {
    try {
      features.value = await fetchMerchantFeatures()
    } catch {
      // 拉取失败保持默认（全开），不阻断商户端使用
    } finally {
      loaded.value = true
    }
  }

  return { features, loaded, load }
})
