<script setup lang="ts">
import { computed } from 'vue'
import { TrendingUp, TrendingDown, Server, Users, Receipt, Wallet } from 'lucide-vue-next'
import { Panel, Badge } from '@/components/ui'
import TrendChart from '@/components/TrendChart.vue'
import {
  overviewDates,
  overviewGmvSeries,
  siteRanks,
  calcOverviewStats,
} from '@/lib/mock/console'
import { sitePlanText, type SitePlan } from '@/lib/mock/sites'
import { formatMoney } from '@/lib/utils'

const stats = computed(() => calcOverviewStats())

const cards = computed(() => [
  { label: '运行中分站', value: String(stats.value.siteCount), icon: Server, tone: 'text-primary' },
  { label: '本月总 GMV', value: `¥${formatMoney(stats.value.totalGmv)}`, icon: Wallet, tone: 'text-success' },
  { label: '本月总订单', value: stats.value.totalOrders.toLocaleString(), icon: Receipt, tone: 'text-foreground' },
  { label: '入驻商户总数', value: stats.value.totalMerchants.toLocaleString(), icon: Users, tone: 'text-foreground' },
  { label: '分站均 GMV', value: `¥${formatMoney(stats.value.avgGmv)}`, icon: TrendingUp, tone: 'text-foreground' },
])

// 排行榜最大 GMV，用于进度条
const maxGmv = computed(() => Math.max(...siteRanks.map((s) => s.gmv), 1))
</script>

<template>
  <div class="space-y-2.5">
    <!-- 汇总卡 -->
    <Panel title="分站总览" subtitle="平台方视角：全部分站的经营数据汇总。主站只做管理，各分站数据相互独立">
      <div class="grid grid-cols-2 gap-x-8 gap-y-5 sm:grid-cols-3 lg:grid-cols-5">
        <div v-for="c in cards" :key="c.label">
          <div class="flex items-center gap-1.5 text-[13px] text-muted-foreground">
            <component :is="c.icon" class="size-3.5 opacity-60" />
            {{ c.label }}
          </div>
          <div class="mt-1.5 text-xl font-normal tabular-nums" :class="c.tone">{{ c.value }}</div>
        </div>
      </div>
    </Panel>

    <!-- GMV 趋势 -->
    <Panel title="近 7 日全平台 GMV 趋势" subtitle="按套餐分层，反映各档分站的贡献结构">
      <TrendChart :labels="overviewDates" :series="overviewGmvSeries" />
    </Panel>

    <!-- 分站排行 -->
    <Panel title="分站 GMV 排行" :subtitle="`${siteRanks.length} 个运行中分站`">
      <div class="overflow-x-auto">
        <table class="tbl w-full">
          <thead>
            <tr>
              <th class="w-12 col-center">#</th>
              <th>分站</th>
              <th class="col-center">套餐</th>
              <th class="w-[26%]">本月 GMV</th>
              <th class="num">订单数</th>
              <th class="num">商户数</th>
              <th class="num">环比</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(s, i) in siteRanks" :key="s.domain">
              <td class="col-center">
                <span
                  class="inline-flex size-6 items-center justify-center rounded-full text-xs font-semibold tabular-nums"
                  :class="i < 3 ? 'bg-primary/12 text-primary' : 'text-muted-foreground'"
                >{{ i + 1 }}</span>
              </td>
              <td>
                <div class="flex items-center gap-2">
                  <Server class="size-4 shrink-0 text-primary" />
                  <div class="min-w-0">
                    <div class="truncate font-medium">{{ s.name }}</div>
                    <div class="truncate text-xs dim">{{ s.domain }}</div>
                  </div>
                </div>
              </td>
              <td class="col-center"><Badge variant="outline">{{ sitePlanText[s.plan as SitePlan] }}</Badge></td>
              <td>
                <div class="mb-1 text-xs tabular-nums font-medium">¥{{ formatMoney(s.gmv) }}</div>
                <div class="h-1.5 w-full overflow-hidden rounded-full bg-muted">
                  <div class="h-full rounded-full bg-primary" :style="{ width: Math.round((s.gmv / maxGmv) * 100) + '%' }" />
                </div>
              </td>
              <td class="num tabular-nums">{{ s.orders.toLocaleString() }}</td>
              <td class="num tabular-nums">{{ s.merchants }}</td>
              <td class="num">
                <span
                  class="inline-flex items-center gap-0.5 tabular-nums"
                  :class="s.growth >= 0 ? 'text-success' : 'text-destructive'"
                >
                  <TrendingUp v-if="s.growth >= 0" class="size-3.5" />
                  <TrendingDown v-else class="size-3.5" />
                  {{ Math.abs(s.growth).toFixed(1) }}%
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </Panel>
  </div>
</template>
