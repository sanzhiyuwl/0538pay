<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { QrCode, Loader2, ShieldCheck, AlertCircle } from 'lucide-vue-next'
import QRCodeLib from 'qrcode'
import { Button } from '@/components/ui'
import {
  fetchCashierOrder,
  triggerMockPay,
  fetchOrderStatus,
  cashierChoosePay,
  fetchWxAuthURL,
  fetchWxOpenID,
  type CashierOrder,
  type CashierSubmitResp,
} from '@/lib/api/pay'
import { useSiteStore } from '@/stores/site'
import { splitBrand } from '@/lib/utils'

// 收银台中间页。真实渠道(有 qrcode)渲染真二维码 + 轮询查单；mock 渠道无二维码，
// 用"模拟支付成功"按钮直接触发后端回调走完整链路。对齐 epay cashier.php 语义。
const route = useRoute()
const router = useRouter()
const tradeNo = route.params.trade_no as string

// 收银台品牌名取网站名称，末尾一个词高亮
const siteStore = useSiteStore()
siteStore.hydrate()
const brand = computed(() => splitBrand(siteStore.config.sitename))

const order = ref<CashierOrder | null>(null)
const loading = ref(true)
const paying = ref(false)
const errMsg = ref('')
const qrDataURL = ref('') // 真实渠道二维码图片(DataURL)
let pollTimer: ReturnType<typeof setInterval> | null = null

// B1-04：裸单(空 type 未定通道)带 paytypes → 先渲染聚合选方式，选定后补选通道再走扫码/模拟支付。
const needChoose = computed(() => !!order.value && !!order.value.paytypes && order.value.paytypes.length > 0 && !order.value.plugin)
const choosing = ref(false)

// 是否真实渠道：mock 渠道的 qrcode 是收银台自身链接(占位)，走模拟支付按钮；
// 其余渠道有真实 qrcode 内容，渲染真二维码 + 轮询查单。
const isRealChannel = computed(() => !!order.value && order.value.plugin !== 'mock' && !!order.value.qrcode)

// 是否在微信内置浏览器（JSAPI 支付只在微信内有意义）。
const inWechat = /micromessenger/i.test(navigator.userAgent)
// 微信内置 JS 支付桥（wx.requestPayment / WeixinJSBridge）。
interface WxPayParams {
  appId: string
  timeStamp: string
  nonceStr: string
  package: string
  signType: string
  paySign: string
}
interface WeixinBridge {
  invoke(
    method: string,
    params: Record<string, string>,
    cb: (res: { err_msg: string }) => void,
  ): void
}
declare global {
  interface Window {
    WeixinJSBridge?: WeixinBridge
  }
}

// jsapiPending：记录当前正在走 JSAPI 授权的支付方式 type（回跳后据此复发起下单）。
const JSAPI_TYPE_KEY = 'cashier_jsapi_type'

// 选定支付方式：对既有裸单补选通道下单，成功后重载订单信息渲染二维码/模拟支付。
// 微信内选微信支付时，若无 openid 先走网页授权（跳转微信授权页，回跳后自动续下单）。
async function choose(type: string) {
  if (choosing.value) return
  // 微信内 + 选微信支付 + 尚无 code：先跳授权拿 openid（JSAPI 前置）。
  if (inWechat && isWxType(type) && !getCodeFromURL()) {
    choosing.value = true
    errMsg.value = ''
    try {
      sessionStorage.setItem(JSAPI_TYPE_KEY, type)
      const redirect = cashierURLForRedirect()
      const authURL = await fetchWxAuthURL(tradeNo, redirect)
      window.location.href = authURL // 跳微信授权，回跳后 onMounted 带 code 续流程
      return
    } catch (e: unknown) {
      errMsg.value = e instanceof Error ? e.message : '发起微信授权失败'
      choosing.value = false
      return
    }
  }
  choosing.value = true
  errMsg.value = ''
  try {
    const resp = await cashierChoosePay(tradeNo, type)
    if (await handleJsapiResp(resp)) return // JSAPI：已拉起微信支付，无需渲染二维码
    // 重载订单（此时已定通道，plugin/qrcode 就绪）。
    order.value = await fetchCashierOrder(tradeNo)
    if (isRealChannel.value && order.value.qrcode) {
      qrDataURL.value = await QRCodeLib.toDataURL(order.value.qrcode, { width: 220, margin: 1 })
      startPolling()
    }
  } catch (e: unknown) {
    errMsg.value = e instanceof Error ? e.message : '选择支付方式失败'
  } finally {
    choosing.value = false
  }
}

// isWxType 判断支付方式是否为微信类（JSAPI 需公众号授权）。
function isWxType(type: string): boolean {
  return type.startsWith('wx') || type === 'wxpay'
}

// getCodeFromURL 取微信授权回跳带回的 code（?code=..&state=..）。
function getCodeFromURL(): string {
  return new URLSearchParams(window.location.search).get('code') ?? ''
}

// cashierURLForRedirect 当前收银台地址（去掉已有 code/state，供微信授权 redirect_uri）。
function cashierURLForRedirect(): string {
  const u = new URL(window.location.href)
  u.searchParams.delete('code')
  u.searchParams.delete('state')
  return u.toString()
}

// handleJsapiResp 若下单返回 JSAPI 拉起参数（pay_type=jsapi/wap + html 为参数 JSON），
// 调微信内置 requestPayment 拉起收银；返回 true 表示已处理。
async function handleJsapiResp(resp: CashierSubmitResp): Promise<boolean> {
  if (!resp.html || (resp.pay_type !== 'jsapi' && resp.pay_type !== 'wap')) return false
  let params: WxPayParams
  try {
    params = JSON.parse(resp.html)
  } catch {
    return false
  }
  if (!params.paySign) return false
  await invokeWxPay(params)
  return true
}

// invokeWxPay 调微信内置 JSSDK 拉起支付（WeixinJSBridge.invoke getBrandWCPayRequest）。
function invokeWxPay(params: WxPayParams): Promise<void> {
  return new Promise((resolve) => {
    const doPay = () => {
      window.WeixinJSBridge?.invoke(
        'getBrandWCPayRequest',
        {
          appId: params.appId,
          timeStamp: params.timeStamp,
          nonceStr: params.nonceStr,
          package: params.package,
          signType: params.signType,
          paySign: params.paySign,
        },
        (res) => {
          if (res.err_msg === 'get_brand_wcpay_request:ok') {
            goPayok()
          } else if (res.err_msg === 'get_brand_wcpay_request:cancel') {
            errMsg.value = '您已取消支付'
          } else {
            errMsg.value = '微信支付未完成，请重试'
            startPolling() // 兜底：轮询查单，防拉起异常但实际已付
          }
          resolve()
        },
      )
    }
    if (window.WeixinJSBridge) {
      doPay()
    } else {
      // 桥未就绪时等待其加载完成（微信内置浏览器异步注入）。
      document.addEventListener('WeixinJSBridgeReady', doPay, { once: true })
    }
  })
}

// resumeJsapiAfterAuth 微信授权回跳后（URL 带 code）续 JSAPI 下单：换 openid → 带 openid 下单 → 拉起。
async function resumeJsapiAfterAuth(code: string): Promise<void> {
  const type = sessionStorage.getItem(JSAPI_TYPE_KEY) || 'wxpay'
  sessionStorage.removeItem(JSAPI_TYPE_KEY)
  choosing.value = true
  try {
    const openid = await fetchWxOpenID(tradeNo, code)
    const resp = await cashierChoosePay(tradeNo, type, openid)
    if (await handleJsapiResp(resp)) return
    // 未返回 JSAPI 参数（异常）：退回重载订单渲染。
    order.value = await fetchCashierOrder(tradeNo)
  } catch (e: unknown) {
    errMsg.value = e instanceof Error ? e.message : '微信支付发起失败'
  } finally {
    choosing.value = false
  }
}
// B1-65：需支付(money)与订单额(order_money)差额即手续费，>0 才展示明细。
const feeAmount = computed(() => {
  const o = order.value
  if (!o || !o.order_money) return ''
  const fee = Number(o.money) - Number(o.order_money)
  return fee > 0 ? fee.toFixed(2) : ''
})

onMounted(async () => {
  try {
    order.value = await fetchCashierOrder(tradeNo)
    if (order.value.status === 1) {
      // 已支付：直接跳成功页，避免重复支付（对齐 epay status==1 拦截）
      goPayok()
      return
    }
    // 微信授权回跳（URL 带 code）：续 JSAPI 下单 → 拉起微信支付。
    const wxCode = getCodeFromURL()
    if (inWechat && wxCode) {
      loading.value = false
      await resumeJsapiAfterAuth(wxCode)
      return
    }
    if (isRealChannel.value) {
      // 真实渠道：把 code_url/qr_code 渲染成二维码图片，并开始轮询查单
      qrDataURL.value = await QRCodeLib.toDataURL(order.value.qrcode, { width: 220, margin: 1 })
      startPolling()
    }
  } catch (e: unknown) {
    errMsg.value = e instanceof Error ? e.message : '订单加载失败'
  } finally {
    loading.value = false
  }
})

onUnmounted(stopPolling)

// 轮询查单：每 3 秒查一次订单状态，已支付则停轮询并跳成功页。
function startPolling() {
  stopPolling()
  pollTimer = setInterval(async () => {
    try {
      const status = await fetchOrderStatus(tradeNo)
      if (status === 1) {
        stopPolling()
        goPayok()
      }
    } catch {
      // 轮询失败静默重试，不打断用户
    }
  }, 3000)
}
function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function goPayok() {
  const o = order.value
  if (!o) return
  router.push({
    name: 'payok',
    query: {
      trade_no: o.trade_no,
      out_trade_no: o.out_trade_no,
      money: o.money,
      name: o.name,
      return_url: o.return_url,
    },
  })
}

async function pay() {
  if (!order.value || paying.value) return
  paying.value = true
  errMsg.value = ''
  try {
    const res = await triggerMockPay(order.value)
    if (res.toLowerCase().includes('success')) {
      goPayok()
    } else {
      errMsg.value = '支付回调未成功，请重试'
    }
  } catch (e: unknown) {
    errMsg.value = e instanceof Error ? e.message : '支付失败'
  } finally {
    paying.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen flex-col items-center justify-center bg-content px-4 py-10">
    <div class="mb-6 flex items-center gap-2 text-lg font-bold tracking-tight">
      <span>{{ brand.lead }}<span v-if="brand.accent" class="text-primary">{{ brand.accent }}</span></span> 收银台
    </div>

    <!-- 加载态 -->
    <div v-if="loading" class="flex items-center gap-2 text-sm text-muted-foreground">
      <Loader2 class="size-4 animate-spin" /> 正在加载订单…
    </div>

    <!-- 错误态（订单不存在等） -->
    <div
      v-else-if="!order"
      class="w-full max-w-md rounded-2xl border border-border bg-background p-8 text-center shadow-sm"
    >
      <div class="mx-auto flex size-14 items-center justify-center rounded-full bg-destructive/10 text-destructive">
        <AlertCircle class="size-8" />
      </div>
      <h1 class="mt-4 text-xl font-bold">无法加载订单</h1>
      <p class="mt-2 text-sm text-muted-foreground">{{ errMsg || '该订单不存在或已失效' }}</p>
      <Button variant="outline" class="mt-6" @click="router.push('/')">返回首页</Button>
    </div>

    <!-- 收银台主体 -->
    <div
      v-else
      class="grid w-full max-w-3xl gap-5 md:grid-cols-[1.1fr_1fr]"
    >
      <!-- 左：订单信息 -->
      <div class="rounded-2xl border border-border bg-background p-6 shadow-sm">
        <h2 class="text-sm font-semibold text-muted-foreground">订单信息</h2>
        <div class="mt-4 space-y-3 text-sm">
          <div class="flex justify-between gap-4">
            <span class="text-muted-foreground">商品名称</span>
            <span class="text-right">{{ order.name }}</span>
          </div>
          <div class="flex justify-between gap-4">
            <span class="text-muted-foreground">系统订单号</span>
            <span class="font-mono text-xs">{{ order.trade_no }}</span>
          </div>
          <div class="flex justify-between gap-4">
            <span class="text-muted-foreground">商户订单号</span>
            <span class="font-mono text-xs">{{ order.out_trade_no }}</span>
          </div>
          <div class="flex justify-between gap-4">
            <span class="text-muted-foreground">创建时间</span>
            <span class="tabular-nums">{{ order.addtime }}</span>
          </div>
        </div>
        <div class="mt-5 border-t border-border pt-4">
          <div class="flex items-end justify-between">
            <span class="text-sm text-muted-foreground">应付金额</span>
            <span class="text-3xl font-bold tabular-nums text-primary">
              <span class="text-lg font-normal text-muted-foreground">¥</span>{{ order.money }}
            </span>
          </div>
          <!-- B1-65：加费时展示订单额与手续费明细（对齐 epay cashier.php:98 '含X元手续费'） -->
          <div v-if="feeAmount" class="mt-2 flex items-center justify-between text-xs text-muted-foreground">
            <span>订单金额 ¥{{ order.order_money }}</span>
            <span>含 ¥{{ feeAmount }} 元手续费</span>
          </div>
        </div>
      </div>

      <!-- 右：扫码支付 -->
      <div class="flex flex-col items-center rounded-2xl border border-border bg-background p-6 text-center shadow-sm">
        <div class="text-sm font-semibold text-muted-foreground">{{ needChoose ? '选择支付方式' : '扫码支付' }}</div>

        <!-- B1-04：裸单聚合选方式（对齐 epay cashier.php 选方式） -->
        <template v-if="needChoose">
          <div class="mt-4 flex w-full flex-col gap-2">
            <button
              v-for="pt in order.paytypes"
              :key="pt.type"
              type="button"
              class="flex items-center justify-between rounded-xl border border-border px-4 py-3 text-sm transition hover:border-primary hover:bg-primary/5 disabled:opacity-60"
              :disabled="choosing"
              @click="choose(pt.type)"
            >
              <span class="font-medium">{{ pt.showname }}</span>
              <Loader2 v-if="choosing" class="size-4 animate-spin text-muted-foreground" />
            </button>
          </div>
          <p class="mt-4 text-xs text-muted-foreground">请选择一种支付方式完成付款</p>
        </template>

        <!-- 真实渠道：渲染真二维码 + 轮询查单 -->
        <template v-else-if="isRealChannel">
          <div class="mt-4 flex size-44 items-center justify-center rounded-xl border border-border bg-white p-2">
            <img v-if="qrDataURL" :src="qrDataURL" alt="支付二维码" class="size-full" />
            <QrCode v-else class="size-20 text-muted-foreground/50" />
          </div>
          <p class="mt-3 text-xs text-muted-foreground">
            请使用{{ order.plugin?.startsWith('wx') ? '微信' : order.plugin?.startsWith('ali') ? '支付宝' : '对应 App' }}扫码完成支付
          </p>
          <div class="mt-4 flex items-center gap-1.5 text-xs text-muted-foreground">
            <Loader2 class="size-3.5 animate-spin" /> 等待支付中，支付后自动跳转…
          </div>
        </template>

        <!-- mock 渠道：无真实二维码，模拟支付按钮 -->
        <template v-else>
          <div class="mt-4 flex size-44 items-center justify-center rounded-xl border-2 border-dashed border-border bg-muted/40">
            <QrCode class="size-20 text-muted-foreground/50" />
          </div>
          <p class="mt-3 text-xs text-muted-foreground">
            模拟渠道（mock）无真实二维码，<br />点击下方按钮模拟用户完成支付
          </p>
          <Button class="mt-5 w-full" :disabled="paying" @click="pay">
            <Loader2 v-if="paying" class="mr-1 size-4 animate-spin" />
            {{ paying ? '支付处理中…' : '模拟支付成功' }}
          </Button>
        </template>

        <p v-if="errMsg" class="mt-3 text-xs text-destructive">{{ errMsg }}</p>

        <div class="mt-4 flex items-center gap-1 text-xs text-muted-foreground">
          <ShieldCheck class="size-3.5" /> 支付信息由平台加密处理
        </div>
      </div>
    </div>
  </div>
</template>
