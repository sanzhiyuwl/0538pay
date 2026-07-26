<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ShieldCheck, Loader2, AlertCircle } from 'lucide-vue-next'
import { Button } from '@/components/ui'
import { submitPayRaw } from '@/lib/api/pay'
import { ApiError } from '@/lib/api/client'
import { useSiteStore } from '@/stores/site'
import { splitBrand } from '@/lib/utils'

// 支付安全验证页（对齐 epay showPayVerifyPage / verify_jump.php）。
// 后端 Submit 命中验证策略且无合法 __defend 时，返回 pay_type=verify + pay_url 指向本页，
// URL 携带 dk(defendKey) / vt(验证方式) / q(base64 原始下单参数)。
// 本页据 vt 渲染：0=跳转确认页（无需第三方，可闭环）；1/2=极验（待凭证，降级为确认页并提示）。
// 通过后生成合法 __defend（{10位时间}+defendKey+{6位随机}，中段32位=defendKey），
// 连同原始参数复发起 POST /api/pay/submit 放行，再按返回 pay_type 跳转。
const route = useRoute()
const router = useRouter()

const siteStore = useSiteStore()
siteStore.hydrate()
const brand = computed(() => splitBrand(siteStore.config.sitename))

const defendKey = String(route.query.dk || '')
const verifyType = String(route.query.vt || '0')
const rawQuery = String(route.query.q || '')

const submitting = ref(false)
const errMsg = ref('')

// 解码原始下单参数
let bizParams: Record<string, string> = {}
onMounted(() => {
  if (!defendKey || !rawQuery) {
    errMsg.value = '验证参数缺失，请返回重新发起支付'
    return
  }
  try {
    // base64url 解码 → JSON
    const b64 = rawQuery.replace(/-/g, '+').replace(/_/g, '/')
    const json = decodeURIComponent(
      atob(b64)
        .split('')
        .map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
        .join(''),
    )
    bizParams = JSON.parse(json)
  } catch {
    errMsg.value = '验证参数解析失败，请返回重新发起支付'
  }
})

// 生成合法 __defend：10位秒级时间戳 + 32位 defendKey + 6位随机（对齐 epay verify_jump 的 time().key.rand）
function buildDefend(): string {
  const ts = String(Math.floor(Date.now() / 1000)).padStart(10, '0').slice(0, 10)
  const rnd = String(Math.floor(100000 + Math.random() * 900000))
  return ts + defendKey + rnd
}

async function confirm() {
  if (submitting.value || errMsg.value) return
  submitting.value = true
  errMsg.value = ''
  try {
    const resp = await submitPayRaw({ ...bizParams, __defend: buildDefend() })
    // 通过后按返回 pay_type 跳转
    if (resp.pay_type === 'verify') {
      errMsg.value = '验证未通过，请重试'
      return
    }
    if (resp.pay_type === 'jump' && resp.pay_url) {
      // 收银台聚合选方式等跳转
      if (resp.pay_url.startsWith('http')) {
        window.location.href = resp.pay_url
      } else {
        router.replace(resp.pay_url)
      }
      return
    }
    // 已定通道：进收银台展示二维码/支付
    router.replace(`/pay/mock/cashier/${resp.trade_no}`)
  } catch (e) {
    errMsg.value = e instanceof ApiError ? e.message : '验证提交失败，请重试'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="mx-auto flex min-h-[70vh] max-w-md flex-col items-center justify-center px-4">
    <div class="w-full rounded-lg bg-muted/40 p-8 text-center">
      <div class="mx-auto mb-4 flex size-14 items-center justify-center rounded-full bg-primary/10">
        <ShieldCheck class="size-7 text-primary" />
      </div>
      <h1 class="mb-1 text-lg font-medium">
        <span>{{ brand.lead }}</span><span class="text-primary">{{ brand.accent }}</span> 支付安全验证
      </h1>
      <p class="mb-6 text-sm text-muted-foreground">
        为保障交易安全，本次支付需完成验证后继续。
      </p>

      <div v-if="errMsg" class="mb-4 flex items-center justify-center gap-2 rounded bg-destructive/10 px-3 py-2 text-sm text-destructive">
        <AlertCircle class="size-4" />{{ errMsg }}
      </div>

      <!-- 极验方式待凭证：如实提示后仍走确认页闭环 -->
      <p v-if="verifyType !== '0'" class="mb-4 rounded bg-muted px-3 py-2 text-xs text-muted-foreground">
        当前配置为极验验证，需管理员配置极验凭证后生效；未配置时降级为点击确认。
      </p>

      <Button class="w-full" :disabled="submitting || !!errMsg" @click="confirm">
        <Loader2 v-if="submitting" class="size-4 animate-spin" />
        <ShieldCheck v-else class="size-4" />
        {{ submitting ? '验证中…' : '点击完成验证并继续支付' }}
      </Button>
    </div>
  </div>
</template>
