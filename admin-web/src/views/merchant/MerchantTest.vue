<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { FlaskConical } from 'lucide-vue-next'
import { Panel, Button } from '@/components/ui'
import { fetchTestPayInfo, submitTestPay, type TestPayInfo } from '@/lib/api/merchantCenter'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const router = useRouter()
const toast = useToast()

// 测试支付（对齐 epay user/test.php）：用测试收款商户下一笔小额真实订单走收单链
const info = ref<TestPayInfo>({ open: false, min_money: '0.01', max_money: '50000', types: [] })
const money = ref('1')
const busy = ref(false)

async function load() {
  try {
    info.value = await fetchTestPayInfo()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载测试支付信息失败')
  }
}
onMounted(load)

async function pay(type: string) {
  if (busy.value) return
  const m = Number(money.value)
  if (!(m > 0)) {
    toast.error('请输入有效金额')
    return
  }
  busy.value = true
  try {
    const res = await submitTestPay(money.value, type)
    // 下单成功 → 跳收银台（mock 渠道走模拟支付页）
    router.push({ path: `/pay/mock/cashier/${res.trade_no}` })
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '测试支付下单失败')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="测试支付" subtitle="发起一笔小额真实订单，验证支付通道是否正常收款">
      <div v-if="!info.open" class="rounded bg-muted/40 px-3 py-2.5 text-sm text-muted-foreground">
        平台当前未开启测试支付功能，如需测试请联系平台管理员开启。
      </div>
      <div v-else class="max-w-lg space-y-4">
        <div class="row-field">
          <label class="lbl">支付金额</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="money" type="number" step="0.01" class="field-input flex-1" placeholder="默认 1 元" />
            <span class="text-sm text-muted-foreground">元</span>
          </div>
        </div>
        <p class="text-xs text-muted-foreground">
          金额范围 {{ info.min_money }} ~ {{ info.max_money }} 元。该笔为真实订单，将实际扣款用于验证通道到账。
        </p>

        <div>
          <div class="mb-2 text-sm text-muted-foreground">选择支付方式发起测试</div>
          <div v-if="info.types.length" class="flex flex-wrap gap-2">
            <Button v-for="t in info.types" :key="t.type" variant="outline" :disabled="busy" @click="pay(t.type)">
              <FlaskConical class="size-4" />{{ t.showname }}
            </Button>
          </div>
          <div v-else class="rounded bg-muted/40 px-3 py-2.5 text-sm text-muted-foreground">
            暂无可用的支付方式（需平台配置并开启至少一个已实现的支付通道）。
          </div>
        </div>
      </div>
    </Panel>
  </div>
</template>
