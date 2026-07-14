<script setup lang="ts">
import { ref, computed } from 'vue'
import { RefreshCw, ExternalLink, Puzzle, Search } from 'lucide-vue-next'
import { Panel, Button, Badge } from '@/components/ui'
import { plugins, splitTypes, calcPluginStats } from '@/lib/mock/plugins'

const stats = computed(() => calcPluginStats(plugins))

// 名称 / 描述搜索
const kw = ref('')
const filtered = computed(() => {
  const v = kw.value.trim().toLowerCase()
  if (!v) return plugins
  return plugins.filter(
    (p) => p.name.toLowerCase().includes(v) || p.showname.toLowerCase().includes(v),
  )
})
</script>

<template>
  <div class="space-y-2.5">
    <!-- 概况 -->
    <Panel title="支付插件" subtitle="对接上游支付接口的插件，放置于 /plugins/ 目录，上传源码后刷新识别">
      <template #actions>
        <Button variant="outline" size="sm"><RefreshCw />刷新插件列表</Button>
      </template>
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">插件总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.total }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">支持支付</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-primary">{{ stats.total }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">支持转账</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ stats.withTransfer }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">仅支付</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.onlyPay }}</div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="插件列表" :subtitle="`${filtered.length} 个`">
      <template #actions>
        <div class="relative">
          <Search class="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <input v-model="kw" placeholder="搜索插件名称 / 描述" class="field-input w-56 !pl-9" />
        </div>
      </template>
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[16%]">插件名称</th>
              <th class="w-[24%]">插件描述</th>
              <th class="w-[14%]">作者</th>
              <th class="w-[23%]">包含的支付方式</th>
              <th class="w-[23%]">包含的转账方式</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in filtered" :key="p.name">
              <td>
                <div class="flex items-center gap-1.5">
                  <Puzzle class="size-4 text-primary" />
                  <span class="font-mono text-[13px] font-medium">{{ p.name }}</span>
                </div>
              </td>
              <td>{{ p.showname }}</td>
              <td>
                <a
                  v-if="p.link"
                  :href="p.link"
                  target="_blank"
                  rel="noreferrer"
                  class="inline-flex items-center gap-1 text-primary hover:underline"
                >
                  {{ p.author }}<ExternalLink class="size-3 opacity-60" />
                </a>
                <span v-else class="text-muted-foreground">{{ p.author }}</span>
              </td>
              <td>
                <div class="flex flex-wrap gap-1">
                  <Badge v-for="t in splitTypes(p.types)" :key="t" variant="outline">{{ t }}</Badge>
                </div>
              </td>
              <td>
                <div v-if="splitTypes(p.transtypes).length" class="flex flex-wrap gap-1">
                  <Badge v-for="t in splitTypes(p.transtypes)" :key="t" variant="muted">{{ t }}</Badge>
                </div>
                <span v-else class="dim">—</span>
              </td>
            </tr>
            <tr v-if="!filtered.length">
              <td colspan="5" class="py-10 text-center dim">没有匹配的插件</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        支付插件目录：<span class="font-mono">/plugins/</span>。将符合要求的插件源码解压到该目录，然后点击「刷新插件列表」即可识别。
      </p>
    </Panel>
  </div>
</template>
