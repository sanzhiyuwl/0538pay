<script setup lang="ts">
import { Check, Pencil, Star } from 'lucide-vue-next'
import { Panel, Button } from '@/components/ui'
import { planCards } from '@/lib/mock/console'
import { sitePlanText, type SitePlan } from '@/lib/mock/sites'
import { formatMoney } from '@/lib/utils'

// 对比表的功能行（各套餐是否包含 / 数值）
const compareRows: { label: string; get: (k: SitePlan) => string }[] = [
  { label: '年费价格', get: (k) => `¥${formatMoney(planCards.find((p) => p.key === k)!.price)}` },
  { label: '折合每月', get: (k) => `¥${formatMoney(planCards.find((p) => p.key === k)!.monthly)}` },
  { label: '最大商户数', get: (k) => planCards.find((p) => p.key === k)!.quotaText[0].value },
  { label: '最大通道数', get: (k) => planCards.find((p) => p.key === k)!.quotaText[1].value },
  { label: '月交易额上限', get: (k) => planCards.find((p) => p.key === k)!.quotaText[2].value },
  { label: '风控管理模块', get: (k) => (k === 'basic' ? '—' : '✓') },
  { label: '白标品牌定制', get: (k) => (k === 'ultimate' ? '✓' : '—') },
  { label: '专属客户经理', get: (k) => (k === 'ultimate' ? '✓' : '—') },
]
const planKeys = Object.keys(sitePlanText) as SitePlan[]
</script>

<template>
  <div class="space-y-2.5">
    <Panel
      title="租户套餐"
      subtitle="平台对外出租的套餐定价，分站按套餐 + 期限付费。区别于主后台「用户组 / 套餐」（分站内商户的套餐）"
    >
      <template #actions>
        <Button size="sm" variant="outline"><Pencil />编辑套餐</Button>
      </template>

      <!-- 定价卡 -->
      <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
        <div
          v-for="p in planCards"
          :key="p.key"
          class="relative flex flex-col bg-muted/40 p-5"
          :class="p.highlight ? 'ring-2 ring-primary' : ''"
        >
          <div
            v-if="p.highlight"
            class="absolute right-4 top-4 inline-flex items-center gap-1 rounded bg-primary px-2 py-0.5 text-[11px] font-medium text-primary-foreground"
          >
            <Star class="size-3" />推荐
          </div>

          <div class="text-base font-semibold">{{ p.name }}</div>
          <p class="mt-1 min-h-[2.5rem] text-xs text-muted-foreground">{{ p.desc }}</p>

          <div class="mt-3 flex items-baseline gap-1">
            <span class="text-sm text-muted-foreground">¥</span>
            <span class="text-3xl font-semibold tabular-nums">{{ formatMoney(p.price) }}</span>
            <span class="text-sm text-muted-foreground">/ 年</span>
          </div>
          <div class="mt-1 text-xs text-muted-foreground">折合 ¥{{ formatMoney(p.monthly) }} / 月</div>

          <div class="mt-4 space-y-2 border-t border-border/60 pt-4">
            <div v-for="q in p.quotaText" :key="q.label" class="flex items-center justify-between text-sm">
              <span class="text-muted-foreground">{{ q.label }}</span>
              <span class="font-medium tabular-nums">{{ q.value }}</span>
            </div>
          </div>

          <ul class="mt-4 space-y-2 border-t border-border/60 pt-4 text-sm">
            <li v-for="f in p.features" :key="f" class="flex items-center gap-2">
              <Check class="size-4 shrink-0 text-success" />
              <span>{{ f }}</span>
            </li>
          </ul>

          <div class="mt-5 flex items-center justify-between border-t border-border/60 pt-4">
            <span class="text-xs text-muted-foreground">当前 {{ p.siteCount }} 个分站在用</span>
            <Button :variant="p.highlight ? 'default' : 'outline'" size="sm">配置</Button>
          </div>
        </div>
      </div>
    </Panel>

    <!-- 套餐对比 -->
    <Panel title="套餐对比" subtitle="各版本功能与配额差异一览">
      <div class="overflow-x-auto">
        <table class="tbl w-full">
          <thead>
            <tr>
              <th class="w-[28%]">对比项</th>
              <th v-for="k in planKeys" :key="k" class="col-center">
                {{ sitePlanText[k] }}
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in compareRows" :key="row.label">
              <td class="font-medium text-muted-foreground">{{ row.label }}</td>
              <td v-for="k in planKeys" :key="k" class="col-center">
                <span
                  v-if="row.get(k) === '✓'"
                  class="inline-flex text-success"
                >
                  <Check class="size-4" />
                </span>
                <span v-else-if="row.get(k) === '—'" class="dim">—</span>
                <span v-else class="tabular-nums">{{ row.get(k) }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </Panel>
  </div>
</template>
