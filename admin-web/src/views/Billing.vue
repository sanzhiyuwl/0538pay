<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Download, TrendingUp, TrendingDown, ArrowUpRight, ArrowDownRight } from 'lucide-vue-next'
import { Panel, Button, Badge, Select } from '@/components/ui'
import { fetchBilling, exportBilling, type MonthlyBill } from '@/lib/api/billing'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { formatMoney } from '@/lib/utils'

const toast = useToast()

const bills = ref<MonthlyBill[]>([])
const activePeriod = ref('')
const loading = ref(false)
const exporting = ref(false)

const periodOptions = computed(() => bills.value.map((b) => ({ value: b.period, label: b.period })))
const currentBill = computed(() => bills.value.find((b) => b.period === activePeriod.value) ?? null)

const billStatus: Record<number, { text: string; variant: 'warning' | 'success' }> = {
  0: { text: '进行中', variant: 'warning' },
  1: { text: '已归档', variant: 'success' },
}

async function load() {
  loading.value = true
  try {
    const { bills: list } = await fetchBilling()
    bills.value = list
    if (list.length) activePeriod.value = list[0].period
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载账单失败')
    bills.value = []
  } finally {
    loading.value = false
  }
}

async function doExport() {
  if (exporting.value) return
  exporting.value = true
  try {
    await exportBilling()
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '导出失败')
  } finally {
    exporting.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="space-y-2.5">
    <!-- 账期选择 + 当期概况 -->
    <Panel title="账单中心" subtitle="平台财务月度对账：手续费利润收入 vs 结算打款 / 代付 / 分账 / 退款支出">
      <template #actions>
        <Select v-if="periodOptions.length" v-model="activePeriod" :options="periodOptions" class="w-32" />
        <Button variant="outline" size="sm" :disabled="exporting || !bills.length" @click="doExport">
          <Download />导出账单
        </Button>
      </template>
      <div v-if="currentBill" class="grid grid-cols-2 gap-x-8 gap-y-5 sm:grid-cols-4">
        <div>
          <div class="text-[13px] text-muted-foreground">本期收入</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-success">+ ¥{{ formatMoney(Number(currentBill.income)) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">本期支出</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-destructive">- ¥{{ formatMoney(Number(currentBill.expense)) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">净收入</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums" :class="Number(currentBill.net) >= 0 ? 'text-primary' : 'text-destructive'">
            {{ Number(currentBill.net) >= 0 ? '+' : '-' }} ¥{{ formatMoney(Math.abs(Number(currentBill.net))) }}
          </div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已支付订单</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums">{{ currentBill.orders }}</div>
        </div>
      </div>
      <div v-else class="py-8 text-center dim">{{ loading ? '加载中…' : '暂无账单数据' }}</div>
    </Panel>

    <!-- 收支明细两栏 -->
    <div v-if="currentBill" class="grid grid-cols-1 gap-2.5 lg:grid-cols-2">
      <Panel title="收入明细" :subtitle="currentBill.period">
        <template #actions>
          <span class="inline-flex items-center gap-1 text-sm text-success">
            <TrendingUp class="size-4" />¥{{ formatMoney(Number(currentBill.income)) }}
          </span>
        </template>
        <div class="space-y-2.5">
          <div v-for="item in currentBill.incomes" :key="item.label" class="flex items-center justify-between text-sm">
            <span class="inline-flex items-center gap-1.5">
              <ArrowUpRight class="size-3.5 text-success" />{{ item.label }}
            </span>
            <span class="tabular-nums text-success">+ <span class="text-xs opacity-70">¥</span>{{ formatMoney(Number(item.amount)) }}</span>
          </div>
        </div>
        <div class="mt-3 flex items-center justify-between border-t-2 border-border pt-3 text-sm font-medium">
          <span>收入合计</span>
          <span class="tabular-nums text-success">+ <span class="text-xs opacity-70">¥</span>{{ formatMoney(Number(currentBill.income)) }}</span>
        </div>
      </Panel>

      <Panel title="支出明细" :subtitle="currentBill.period">
        <template #actions>
          <span class="inline-flex items-center gap-1 text-sm text-destructive">
            <TrendingDown class="size-4" />¥{{ formatMoney(Number(currentBill.expense)) }}
          </span>
        </template>
        <div class="space-y-2.5">
          <div v-for="item in currentBill.expenses" :key="item.label" class="flex items-center justify-between text-sm">
            <span class="inline-flex items-center gap-1.5">
              <ArrowDownRight class="size-3.5 text-destructive" />{{ item.label }}
            </span>
            <span class="tabular-nums text-destructive">- <span class="text-xs opacity-70">¥</span>{{ formatMoney(Number(item.amount)) }}</span>
          </div>
        </div>
        <div class="mt-3 flex items-center justify-between border-t-2 border-border pt-3 text-sm font-medium">
          <span>支出合计</span>
          <span class="tabular-nums text-destructive">- <span class="text-xs opacity-70">¥</span>{{ formatMoney(Number(currentBill.expense)) }}</span>
        </div>
      </Panel>
    </div>

    <!-- 历史账单 -->
    <Panel title="历史账单" :subtitle="`${bills.length} 期`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[18%]">账期</th>
              <th class="num w-[16%]">已支付订单</th>
              <th class="num w-[20%]">收入</th>
              <th class="num w-[20%]">支出</th>
              <th class="num w-[16%]">净收入</th>
              <th class="col-center w-[10%]">状态</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="b in bills"
              :key="b.period"
              class="cursor-pointer"
              :class="b.period === activePeriod && 'bg-accent/40'"
              @click="activePeriod = b.period"
            >
              <td class="font-medium tabular-nums">{{ b.period }}</td>
              <td class="num tabular-nums dim">{{ b.orders }}</td>
              <td class="num tabular-nums text-success">{{ formatMoney(Number(b.income)) }}</td>
              <td class="num tabular-nums text-destructive">{{ formatMoney(Number(b.expense)) }}</td>
              <td class="num tabular-nums font-medium">{{ formatMoney(Number(b.net)) }}</td>
              <td class="col-center">
                <Badge :variant="billStatus[b.status]?.variant ?? 'muted'">{{ billStatus[b.status]?.text ?? '未知' }}</Badge>
              </td>
            </tr>
            <tr v-if="!bills.length">
              <td colspan="6" class="py-10 text-center dim">{{ loading ? '加载中…' : '暂无账单数据' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        收入 = 已支付订单手续费利润；支出 = 商户结算打款 + 代付转账 + 分账划拨 + 订单退款（均按完成时间归入对应账期）。
        当月账期标「进行中」，历史账期标「已归档」。
      </p>
    </Panel>
  </div>
</template>
