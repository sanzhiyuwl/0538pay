<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Loader2, AlertCircle, Wallet } from 'lucide-vue-next'
import { fetchPaypageInfo, submitPaypage, type PaypageInfo } from '@/lib/api/paypage'

// 公开聚合收款页（对齐 epay paypage/index.php）：扫码进入 → 输入金额 → 选支付方式 → 走收单链。
const route = useRoute()
const router = useRouter()
const merchant = (route.query.merchant as string) || ''

const info = ref<PaypageInfo | null>(null)
const loading = ref(true)
const errMsg = ref('')
const money = ref('')
const selected = ref('')
const submitting = ref(false)

onMounted(async () => {
  if (!merchant) {
    errMsg.value = '收款码无效'
    loading.value = false
    return
  }
  try {
    info.value = await fetchPaypageInfo(merchant)
    if (info.value.types.length) selected.value = info.value.types[0].type
  } catch (e) {
    errMsg.value = e instanceof Error ? e.message : '收款码无效或已停用'
  } finally {
    loading.value = false
  }
})

async function pay() {
  const m = Number(money.value)
  if (!(m > 0)) {
    errMsg.value = '请输入付款金额'
    return
  }
  if (!selected.value) {
    errMsg.value = '请选择支付方式'
    return
  }
  errMsg.value = ''
  submitting.value = true
  try {
    const res = await submitPaypage(merchant, money.value, selected.value)
    router.push({ path: `/pay/mock/cashier/${res.trade_no}` })
  } catch (e) {
    errMsg.value = e instanceof Error ? e.message : '下单失败'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-muted/30 px-4 py-10">
    <div class="w-full max-w-sm">
      <!-- 加载 -->
      <div v-if="loading" class="flex flex-col items-center gap-3 py-20 text-muted-foreground">
        <Loader2 class="size-6 animate-spin" />
        <span class="text-sm">加载中…</span>
      </div>

      <!-- 错误（无有效收款码） -->
      <div v-else-if="!info" class="flex flex-col items-center gap-3 rounded-lg bg-background p-8 text-center">
        <AlertCircle class="size-8 text-destructive" />
        <div class="text-sm text-muted-foreground">{{ errMsg || '收款码无效' }}</div>
      </div>

      <!-- 收银台 -->
      <div v-else class="overflow-hidden rounded-lg bg-background shadow-sm">
        <!-- 收款方 -->
        <div class="flex flex-col items-center gap-2 bg-primary/[0.06] px-6 py-6">
          <div class="flex size-12 items-center justify-center rounded-full bg-primary/10">
            <Wallet class="size-6 text-primary" />
          </div>
          <div class="text-base font-medium">{{ info.codename }}</div>
          <div class="text-xs text-muted-foreground">{{ info.sitename }} 提供收款服务</div>
        </div>

        <div class="space-y-5 px-6 py-6">
          <!-- 金额 -->
          <div>
            <label class="mb-1.5 block text-sm text-muted-foreground">付款金额</label>
            <div class="flex items-center gap-2 border-b-2 border-border pb-2 focus-within:border-primary">
              <span class="text-2xl font-medium">¥</span>
              <input
                v-model="money"
                type="number"
                step="0.01"
                placeholder="0.00"
                class="w-full bg-transparent text-2xl font-medium outline-none tabular-nums"
              />
            </div>
          </div>

          <!-- 支付方式 -->
          <div v-if="info.types.length">
            <label class="mb-1.5 block text-sm text-muted-foreground">选择支付方式</label>
            <div class="space-y-2">
              <button
                v-for="t in info.types"
                :key="t.type"
                class="flex w-full items-center justify-between border px-3 py-2.5 text-sm transition-colors"
                :class="selected === t.type ? 'border-primary text-primary ring-1 ring-primary' : 'border-border hover:border-primary/50'"
                @click="selected = t.type"
              >
                <span>{{ t.showname }}</span>
                <span v-if="selected === t.type" class="text-xs">✓</span>
              </button>
            </div>
          </div>
          <div v-else class="rounded bg-muted/40 px-3 py-2.5 text-center text-sm text-muted-foreground">
            暂无可用支付方式
          </div>

          <p v-if="errMsg" class="text-sm text-destructive">{{ errMsg }}</p>

          <button
            class="w-full bg-primary py-3 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-90 disabled:opacity-50"
            :disabled="submitting || !info.types.length"
            @click="pay"
          >
            <span v-if="submitting">提交中…</span>
            <span v-else>确认付款</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
