<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { RefreshCw, HelpCircle } from 'lucide-vue-next'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import TrendChart from '@/components/TrendChart.vue'
import {
  dashboardData as d,
  overviewStats,
  todoStats,
  recentOrders,
} from '@/lib/mock/dashboard'
import { formatMoney } from '@/lib/utils'

const now = '2026-07-12 17:23:33'

// 趋势图数据
const dates = Object.keys(d.order)
const palette: Record<string, string> = {
  alipay: '#1677ff',
  wxpay: '#07c160',
  qqpay: '#faad14',
  bank: '#f5222d',
}
// 订单量趋势（单条：总额折算笔数示意）
const orderTrend = computed(() => [
  {
    name: '订单量',
    color: '#4b7bec',
    data: dates.map((dt) => Math.round(d.order[dt].all / 46)),
  },
])
// 交易额趋势（各支付方式）
const amountTrend = computed(() =>
  Object.entries(d.paytype).map(([key, name]) => ({
    name,
    color: palette[key] ?? '#7266ba',
    data: dates.map((dt) => d.order[dt].paytype[key] ?? 0),
  })),
)

// 明细统计表
const paytypeKeys = Object.keys(d.paytype)
const channelKeys = Object.keys(d.channel)
function paytypeRows(field: 'paytype' | 'profit_paytype') {
  const rows = [
    { label: '今日', row: d.order_today },
    ...dates.slice().reverse().map((dt) => ({ label: dt, row: d.order[dt] })),
  ]
  return rows.map(({ label, row }) => ({
    label,
    cells: paytypeKeys.map((k) => row[field][k] ?? 0),
    total: field === 'paytype' ? row.all : row.profit_all,
  }))
}
function channelRows() {
  const rows = [
    { label: '今日', row: d.order_today },
    ...dates.slice().reverse().map((dt) => ({ label: dt, row: d.order[dt] })),
  ]
  return rows.map(({ label, row }) => ({
    label,
    cells: channelKeys.map((k) => row.channel[k] ?? 0),
    total: row.all,
  }))
}

const statusMap: Record<number, { text: string; variant: 'success' | 'warning' | 'destructive' }> = {
  0: { text: '待支付', variant: 'warning' },
  1: { text: '已支付', variant: 'success' },
  2: { text: '已退款', variant: 'destructive' },
}
const typeNames: Record<string, string> = { alipay: '支付宝', wxpay: '微信', qqpay: 'QQ钱包', bank: '云闪付' }
</script>

<template>
  <div class="space-y-2.5">
    <!-- ===== 实时概况 ===== -->
    <Card>
      <!-- 标题行 + 通栏分隔线 -->
      <div class="flex items-center gap-2 px-6 py-4">
        <h3 class="text-[15px] font-semibold tracking-tight">实时概况</h3>
        <span class="text-xs text-muted-foreground/70">更新时间:{{ now }}</span>
        <div class="flex-1" />
        <button
          class="flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-primary"
        >
          <RefreshCw class="size-3.5" /> 刷新
        </button>
      </div>
      <div class="border-t border-border/70" />

      <div class="px-6 py-6">
        <div class="grid grid-cols-2 gap-x-8 gap-y-7 lg:grid-cols-4">
          <div v-for="s in overviewStats" :key="s.label">
            <div class="flex items-center gap-1 text-[13px] text-muted-foreground">
              {{ s.label }}
              <HelpCircle v-if="s.hint" class="size-3.5 opacity-40" :title="s.hint" />
            </div>
            <div class="mt-2.5 text-[28px] font-normal leading-none tabular-nums text-foreground">
              {{ s.today }}
            </div>
            <div class="mt-2 text-xs text-muted-foreground/60">昨日:{{ s.yesterday }}</div>
            <div class="mt-5">
              <div class="text-[13px] text-muted-foreground">{{ s.totalLabel }}</div>
              <div class="mt-2 text-[22px] font-normal leading-none tabular-nums text-foreground">{{ s.total }}</div>
            </div>
          </div>
        </div>
      </div>
    </Card>

    <!-- ===== 待办事项 ===== -->
    <Card>
      <div class="px-6 py-4">
        <h3 class="text-[15px] font-semibold tracking-tight">待办事项</h3>
      </div>
      <div class="border-t border-border/70" />
      <div class="px-6 py-6">
        <div class="grid grid-cols-2 gap-x-8 gap-y-6 sm:grid-cols-3 lg:grid-cols-6">
          <div v-for="t in todoStats" :key="t.label" class="group cursor-pointer">
            <div class="flex items-center gap-1 text-[13px] text-muted-foreground">
              {{ t.label }}
              <span v-if="t.urgent && t.value > 0" class="size-1.5 rounded-full bg-destructive" />
            </div>
            <div
              :class="[
                'mt-2.5 text-[26px] font-normal leading-none tabular-nums transition-colors group-hover:text-primary',
                t.urgent && t.value > 0 ? 'text-destructive group-hover:text-destructive' : 'text-foreground',
              ]"
            >
              {{ t.value }}
            </div>
          </div>
        </div>
      </div>
    </Card>

    <!-- ===== 两张趋势图 ===== -->
    <div class="grid grid-cols-1 gap-2.5 xl:grid-cols-2">
      <Card>
        <div class="px-6 py-4">
          <h3 class="text-[15px] font-semibold tracking-tight">订单量趋势</h3>
        </div>
        <div class="border-t border-border/70" />
        <div class="px-6 py-5"><TrendChart :labels="dates" :series="orderTrend" /></div>
      </Card>
      <Card>
        <div class="px-6 py-4">
          <h3 class="text-[15px] font-semibold tracking-tight">交易额（元）</h3>
        </div>
        <div class="border-t border-border/70" />
        <div class="px-6 py-5"><TrendChart :labels="dates" :series="amountTrend" /></div>
      </Card>
    </div>

    <!-- ===== 实时订单 ===== -->
    <Card>
      <div class="flex items-center justify-between px-6 py-4">
        <h3 class="text-[15px] font-semibold tracking-tight">实时订单</h3>
        <RouterLink to="/orders" class="text-xs text-muted-foreground transition-colors hover:text-primary">查看全部</RouterLink>
      </div>
      <div class="border-t border-border/70" />
      <div class="px-6 py-4">
        <div class="overflow-x-auto">
          <table class="w-full table-fixed text-sm">
            <thead>
              <tr class="border-b border-border text-left text-muted-foreground">
                <th class="pb-2.5 font-medium">商户</th>
                <th class="pb-2.5 font-medium">订单号</th>
                <th class="pb-2.5 font-medium">支付方式</th>
                <th class="pb-2.5 font-medium">金额</th>
                <th class="pb-2.5 font-medium">状态</th>
                <th class="pb-2.5 font-medium">时间</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="o in recentOrders"
                :key="o.trade_no"
                class="border-b border-border/50 text-left last:border-0 hover:bg-accent/40"
              >
                <td class="py-3 font-medium">{{ o.merchant }}</td>
                <td class="py-3 text-xs tabular-nums tracking-wide text-muted-foreground">{{ o.trade_no }}</td>
                <td class="py-3 text-muted-foreground">{{ typeNames[o.type] }}</td>
                <td class="py-3 tabular-nums font-medium whitespace-nowrap">
                  <span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ o.money }}
                </td>
                <td class="py-3">
                  <Badge :variant="statusMap[o.status].variant">{{ statusMap[o.status].text }}</Badge>
                </td>
                <td class="py-3 text-muted-foreground tabular-nums whitespace-nowrap">{{ o.time }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </Card>

    <!-- ===== 明细统计表 ===== -->
    <div class="grid grid-cols-1 gap-2.5 xl:grid-cols-2">
      <Card>
        <div class="px-6 py-4">
          <h3 class="text-[15px] font-semibold tracking-tight">支付方式收入统计</h3>
        </div>
        <div class="border-t border-border/70" />
        <div class="px-6 py-4">
          <div class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b border-border text-left text-muted-foreground">
                  <th class="pb-2 pr-4 font-medium">日期</th>
                  <th v-for="k in paytypeKeys" :key="k" class="pb-2 pr-4 font-medium">{{ d.paytype[k] }}</th>
                  <th class="pb-2 font-medium">总计</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="(r, i) in paytypeRows('paytype')"
                  :key="i"
                  class="border-b border-border/50 last:border-0"
                  :class="i === 0 && 'border-b-2 border-border'"
                >
                  <td class="py-2.5 pr-4" :class="i === 0 ? 'font-medium text-foreground' : 'text-muted-foreground'">{{ r.label }}</td>
                  <td v-for="(c, ci) in r.cells" :key="ci" class="py-2.5 pr-4 tabular-nums" :class="i === 0 ? 'font-medium text-foreground' : 'text-foreground/70'">{{ formatMoney(c) }}</td>
                  <td class="py-2.5 font-semibold tabular-nums text-foreground">{{ formatMoney(r.total) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </Card>

      <Card>
        <div class="px-6 py-4">
          <h3 class="text-[15px] font-semibold tracking-tight">支付通道收入统计</h3>
        </div>
        <div class="border-t border-border/70" />
        <div class="px-6 py-4">
          <div class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b border-border text-left text-muted-foreground">
                  <th class="pb-2 pr-4 font-medium">日期</th>
                  <th v-for="k in channelKeys" :key="k" class="pb-2 pr-4 font-medium">{{ d.channel[k] }}</th>
                  <th class="pb-2 font-medium">总计</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="(r, i) in channelRows()"
                  :key="i"
                  class="border-b border-border/50 last:border-0"
                  :class="i === 0 && 'border-b-2 border-border'"
                >
                  <td class="py-2.5 pr-4" :class="i === 0 ? 'font-medium text-foreground' : 'text-muted-foreground'">{{ r.label }}</td>
                  <td v-for="(c, ci) in r.cells" :key="ci" class="py-2.5 pr-4 tabular-nums" :class="i === 0 ? 'font-medium text-foreground' : 'text-foreground/70'">{{ formatMoney(c) }}</td>
                  <td class="py-2.5 font-semibold tabular-nums text-foreground">{{ formatMoney(r.total) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </Card>

      <Card class="xl:col-span-2">
        <div class="flex items-baseline gap-2 px-6 py-4">
          <h3 class="text-[15px] font-semibold tracking-tight">支付方式手续费利润</h3>
          <span class="text-xs text-muted-foreground">已扣除通道成本</span>
        </div>
        <div class="border-t border-border/70" />
        <div class="px-6 py-4">
          <div class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b border-border text-left text-muted-foreground">
                  <th class="pb-2 pr-4 font-medium">日期</th>
                  <th v-for="k in paytypeKeys" :key="k" class="pb-2 pr-4 font-medium">{{ d.paytype[k] }}</th>
                  <th class="pb-2 font-medium">利润合计</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="(r, i) in paytypeRows('profit_paytype')"
                  :key="i"
                  class="border-b border-border/50 last:border-0"
                  :class="i === 0 && 'border-b-2 border-border'"
                >
                  <td class="py-2.5 pr-4" :class="i === 0 ? 'font-medium text-foreground' : 'text-muted-foreground'">{{ r.label }}</td>
                  <td v-for="(c, ci) in r.cells" :key="ci" class="py-2.5 pr-4 tabular-nums" :class="i === 0 ? 'font-medium text-foreground' : 'text-foreground/70'">{{ formatMoney(c) }}</td>
                  <td class="py-2.5 font-semibold tabular-nums text-success">{{ formatMoney(r.total) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </Card>
    </div>
  </div>
</template>
