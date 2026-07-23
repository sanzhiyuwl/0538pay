<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { RefreshCw, ExternalLink, Puzzle, Search, CheckCircle2 } from 'lucide-vue-next'
import { Panel, Button, Badge } from '@/components/ui'
import { plugins, splitTypes, calcPluginStats } from '@/lib/mock/plugins'
import { fetchPluginMeta, type PluginMeta } from '@/lib/api/channels'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

// ===== 上栏：本站真实已注册插件（真接口 /channels/plugins，Go 编译期注册的能用渠道）=====
// 与下栏 epay 全集参考区分——这里才是本站真正实现、能收单的插件。
const installed = ref<PluginMeta[]>([])
const installedLoading = ref(false)
// 从 mock 全集按 key 取显示名/作者/链接（后端 meta 只有能力位，展示名复用已有名录，不重复维护）
const metaByKey = computed(() => {
  const m: Record<string, (typeof plugins)[number]> = {}
  for (const p of plugins) m[p.name] = p
  return m
})
function showname(key: string) {
  return metaByKey.value[key]?.showname || key
}
// 已注册插件支持的支付方式：优先后端 products，回退 mock 名录 types
function installedTypes(p: PluginMeta): string[] {
  if (p.products?.length) return p.products.map((x) => x.name)
  return splitTypes(metaByKey.value[p.key]?.types || '')
}
async function loadInstalled() {
  installedLoading.value = true
  try {
    const list = await fetchPluginMeta()
    // mock 是纯测试桩，不作为"已实现支付插件"展示
    installed.value = list.filter((p) => p.key !== 'mock')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '已注册插件加载失败')
    installed.value = []
  } finally {
    installedLoading.value = false
  }
}
onMounted(loadInstalled)

// ===== 下栏：epay 全集参考（51 插件名录 mock，作规划参考，非本站已实现）=====
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
// 某 epay 插件本站是否已实现（下栏打"已实现"标）
const installedKeys = computed(() => new Set(installed.value.map((p) => p.key)))
</script>

<template>
  <div class="space-y-2.5">
    <!-- 上栏：本站已实现插件（真接口）-->
    <Panel
      title="已实现支付插件"
      subtitle="本站真正实现并注册、可用于收单的支付插件（真实接口 /channels/plugins）"
    >
      <template #actions>
        <Button variant="outline" size="sm" :disabled="installedLoading" @click="loadInstalled">
          <RefreshCw :class="installedLoading ? 'animate-spin' : ''" />刷新
        </Button>
      </template>
      <div class="overflow-x-auto">
        <table class="tbl w-full">
          <thead>
            <tr>
              <th>插件标识</th>
              <th>插件描述</th>
              <th>支持支付方式</th>
              <th>退款</th>
              <th>代付</th>
              <th>密钥表单</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in installed" :key="p.key">
              <td>
                <div class="flex items-center gap-1.5">
                  <CheckCircle2 class="size-4 text-success" />
                  <span class="font-mono text-[13px] font-medium">{{ p.key }}</span>
                </div>
              </td>
              <td>{{ showname(p.key) }}</td>
              <td>
                <div class="flex flex-wrap gap-1">
                  <Badge v-for="t in installedTypes(p)" :key="t" variant="outline">{{ t }}</Badge>
                </div>
              </td>
              <td><Badge :variant="p.can_refund ? 'success' : 'muted'">{{ p.can_refund ? '支持' : '—' }}</Badge></td>
              <td><Badge :variant="p.can_transfer ? 'success' : 'muted'">{{ p.can_transfer ? '支持' : '—' }}</Badge></td>
              <td><Badge :variant="p.configurable ? 'success' : 'muted'">{{ p.configurable ? '动态' : '通用' }}</Badge></td>
            </tr>
            <tr v-if="!installed.length && !installedLoading">
              <td colspan="6" class="py-10 text-center dim">暂无已注册插件</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        本站采用 Go 编译期注册，插件随程序内置，无需上传源码。下方为 epay 全集参考清单。
      </p>
    </Panel>

    <!-- 下栏：epay 全集参考 -->
    <Panel
      title="epay 插件全集参考"
      :subtitle="`${filtered.length} / ${stats.total} 个（来自 epay 目录名录，非本站已实现，仅作规划参考）`"
    >
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
                  <Badge v-if="installedKeys.has(p.name)" variant="success">已实现</Badge>
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
        此清单来自 epay 的 <span class="font-mono">/plugins/</span> 目录名录，供对接规划参考。标「已实现」者为本站已内置可用，其余需按资质分级逐步实现。
      </p>
    </Panel>
  </div>
</template>
