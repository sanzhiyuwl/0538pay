<script setup lang="ts">
import { ref, computed } from 'vue'
import { Download, TrendingUp, TrendingDown, ArrowUpRight, ArrowDownRight } from 'lucide-vue-next'
import { Panel, Button, Badge, Select } from '@/components/ui'
import { bills, billSummary, billStatus } from '@/lib/mock/billing'
import { formatMoney } from '@/lib/utils'

const periodOptions = bills.map((b) => ({ value: b.period, label: b.period }))
const activePeriod = ref(bills[0].period)
const currentBill = computed(() => bills.find((b) => b.period === activePeriod.value)!)
const summary = computed(() => billSummary(currentBill.value))
</script>

<template>
  <div class="space-y-2.5">
    <!-- 账期选择 + 当期概况 -->
    <Panel title="账单中心" subtitle="平台财务月度对账：归集手续费利润、结算打款、代付、分账、退款等收支">
      <template #actions>
        <Select v-model="activePeriod" :options="periodOptions" class="w-32" />
        <Button variant="outline" size="sm"><Download />导出账单</Button>
      </template>
      <div class="grid grid-cols-2 gap-x-8 gap-y-5 sm:grid-cols-3 lg:grid-cols-5">
        <div>
          <div class="text-[13px] text-muted-foreground">期初留存</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(currentBill.opening) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">本期收入</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-success">+ ¥{{ formatMoney(summary.income) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">本期支出</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-destructive">- ¥{{ formatMoney(summary.expense) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">净收入</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums" :class="summary.net >= 0 ? 'text-primary' : 'text-destructive'">
            {{ summary.net >= 0 ? '+' : '-' }} ¥{{ formatMoney(Math.abs(summary.net)) }}
          </div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">期末留存</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums font-medium"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(summary.closing) }}</div>
        </div>
      </div>
    </Panel>

    <!-- 收支明细两栏 -->
    <div class="grid grid-cols-1 gap-2.5 lg:grid-cols-2">
      <Panel title="收入明细" :subtitle="currentBill.period">
        <template #actions>
          <span class="inline-flex items-center gap-1 text-sm text-success">
            <TrendingUp class="size-4" />¥{{ formatMoney(summary.income) }}
          </span>
        </template>
        <div class="space-y-2.5">
          <div v-for="item in currentBill.incomes" :key="item.label" class="flex items-center justify-between text-sm">
            <span class="inline-flex items-center gap-1.5">
              <ArrowUpRight class="size-3.5 text-success" />{{ item.label }}
            </span>
            <span class="tabular-nums text-success">+ <span class="text-xs opacity-70">¥</span>{{ formatMoney(item.amount) }}</span>
          </div>
        </div>
        <div class="mt-3 flex items-center justify-between border-t-2 border-border pt-3 text-sm font-medium">
          <span>收入合计</span>
          <span class="tabular-nums text-success">+ <span class="text-xs opacity-70">¥</span>{{ formatMoney(summary.income) }}</span>
        </div>
      </Panel>

      <Panel title="支出明细" :subtitle="currentBill.period">
        <template #actions>
          <span class="inline-flex items-center gap-1 text-sm text-destructive">
            <TrendingDown class="size-4" />¥{{ formatMoney(summary.expense) }}
          </span>
        </template>
        <div class="space-y-2.5">
          <div v-for="item in currentBill.expenses" :key="item.label" class="flex items-center justify-between text-sm">
            <span class="inline-flex items-center gap-1.5">
              <ArrowDownRight class="size-3.5 text-destructive" />{{ item.label }}
            </span>
            <span class="tabular-nums text-destructive">- <span class="text-xs opacity-70">¥</span>{{ formatMoney(item.amount) }}</span>
          </div>
        </div>
        <div class="mt-3 flex items-center justify-between border-t-2 border-border pt-3 text-sm font-medium">
          <span>支出合计</span>
          <span class="tabular-nums text-destructive">- <span class="text-xs opacity-70">¥</span>{{ formatMoney(summary.expense) }}</span>
        </div>
      </Panel>
    </div>

    <!-- 历史账单 -->
    <Panel title="历史账单" :subtitle="`${bills.length} 期`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[14%]">账期</th>
              <th class="num w-[15%]">期初留存</th>
              <th class="num w-[15%]">收入</th>
              <th class="num w-[15%]">支出</th>
              <th class="num w-[15%]">净收入</th>
              <th class="num w-[15%]">期末留存</th>
              <th class="col-center w-[11%]">状态</th>
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
              <td class="num tabular-nums dim">{{ formatMoney(b.opening) }}</td>
              <td class="num tabular-nums text-success">{{ formatMoney(billSummary(b).income) }}</td>
              <td class="num tabular-nums text-destructive">{{ formatMoney(billSummary(b).expense) }}</td>
              <td class="num tabular-nums font-medium">{{ formatMoney(billSummary(b).net) }}</td>
              <td class="num tabular-nums font-semibold">{{ formatMoney(billSummary(b).closing) }}</td>
              <td class="col-center">
                <Badge :variant="billStatus[b.status].variant">{{ billStatus[b.status].text }}</Badge>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </Panel>
  </div>
</template>
