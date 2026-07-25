<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  RefreshCw,
  ExternalLink,
  Puzzle,
  Search,
  Settings2,
  SlidersHorizontal,
  ChevronDown,
} from 'lucide-vue-next'
import { Panel, Button, Badge, Drawer, Switch } from '@/components/ui'
import BrandLogo from '@/components/BrandLogo.vue'
import { plugins, splitTypes, calcPluginStats } from '@/lib/mock/plugins'
import { fetchPluginMeta, setPluginStatus, setPluginsStatus, type PluginMeta } from '@/lib/api/channels'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const router = useRouter()

// ===== 上栏：本站真实已注册插件（真接口 /channels/plugins，Go 编译期注册的能用渠道）=====
// 展示名/品牌族/协议/形态全部来自后端 meta（Go 侧 Describe 单一数据源），前端不反查 mock 名录。
const installed = ref<PluginMeta[]>([])
const installedLoading = ref(false)

// 品牌视觉：主色 text 类 + 淡底 tint 类（图标由 BrandLogo 组件按品牌名渲染官方矢量/回退）。未登记回退通用灰。
const brandStyle: Record<string, { accent: string; tint: string }> = {
  微信支付: { accent: 'text-[#07c160]', tint: 'bg-[#07c160]/10' },
  支付宝: { accent: 'text-[#1677ff]', tint: 'bg-[#1677ff]/10' },
  彩虹易支付: { accent: 'text-primary', tint: 'bg-primary/10' },
  富友支付: { accent: 'text-[#e6a23c]', tint: 'bg-[#e6a23c]/10' },
  V免签: { accent: 'text-[#8957e5]', tint: 'bg-[#8957e5]/10' },
}
function brandAccent(brand: string): string {
  return brandStyle[brand]?.accent || 'text-muted-foreground'
}
function brandTint(brand: string): string {
  return brandStyle[brand]?.tint || 'bg-muted/60'
}

// 按品牌族聚合：16 个形态级渠道其实只对应少数几个上游品牌。
// 卡内再按协议分区（同一品牌可能有 APIv3/APIv2 两套），把微信 9 形态压成 2 小组，缩小卡间高度差。
interface ProtocolGroup {
  protocol: string
  items: PluginMeta[]
}
interface BrandCard {
  brand: string
  items: PluginMeta[]
  groups: ProtocolGroup[] // 卡内按协议分区
  refund: boolean
  transfer: boolean
  configurable: boolean
}
// 卡片墙只展示「已启用」的插件；禁用的从卡片墙隐藏，仅在「功能设置」抽屉里可开回来。
const enabledPlugins = computed(() => installed.value.filter((p) => p.enabled))
const brandCards = computed<BrandCard[]>(() => {
  const map = new Map<string, PluginMeta[]>()
  for (const p of enabledPlugins.value) {
    const b = p.brand || p.key
    if (!map.has(b)) map.set(b, [])
    map.get(b)!.push(p)
  }
  const cards: BrandCard[] = []
  for (const [brand, items] of map) {
    // 卡内按协议分区，区内按形态名排序
    const gmap = new Map<string, PluginMeta[]>()
    for (const it of items) {
      const pr = it.protocol || '其它'
      if (!gmap.has(pr)) gmap.set(pr, [])
      gmap.get(pr)!.push(it)
    }
    const groups: ProtocolGroup[] = []
    for (const [protocol, gi] of gmap) {
      gi.sort((a, b) => (a.form || a.key).localeCompare(b.form || b.key))
      groups.push({ protocol, items: gi })
    }
    // 协议组按新→旧大致排序（APIv3 前于 APIv2），退化用字典序倒序把 v3 排前
    groups.sort((a, b) => b.protocol.localeCompare(a.protocol))
    cards.push({
      brand,
      items,
      groups,
      refund: items.some((x) => x.can_refund),
      transfer: items.some((x) => x.can_transfer),
      configurable: items.some((x) => x.configurable),
    })
  }
  // 品牌按形态数降序（覆盖越全越靠前），同数按名称
  cards.sort((a, b) => b.items.length - a.items.length || a.brand.localeCompare(b.brand))
  return cards
})
const summary = computed(() => ({
  channels: enabledPlugins.value.length,
  brands: brandCards.value.length,
  refund: brandCards.value.filter((c) => c.refund).length,
  transfer: brandCards.value.filter((c) => c.transfer).length,
}))

// 「去配置」：跳通道管理页，按品牌下渠道 key 的共同前缀预填插件筛选（如 wxnative→wx / alipayf2f→alipay）。
function goConfigure(card: BrandCard) {
  // 取该品牌所有 key 的最长公共前缀作为筛选词（wxnative/wxjsapi/wxv2… → "wx"），
  // 单渠道品牌直接用其 key（fuiou2/vmq/epay）。
  const keys = card.items.map((x) => x.key)
  let prefix = keys[0] || ''
  for (const k of keys) {
    while (prefix && !k.startsWith(prefix)) prefix = prefix.slice(0, -1)
  }
  router.push({ path: '/admin/channels', query: { plugin: prefix || card.items[0]?.key || '' } })
}

async function loadInstalled() {
  installedLoading.value = true
  try {
    const list = await fetchPluginMeta()
    installed.value = list.filter((p) => p.key !== 'mock') // mock 是测试桩，不展示
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '已注册插件加载失败')
    installed.value = []
  } finally {
    installedLoading.value = false
  }
}
onMounted(loadInstalled)

// ===== 功能设置：按插件启用/禁用（后端持久化 plugin_disabled，禁用后收单选通道跳过）=====
const settingsOpen = ref(false)
const toggling = ref<string | null>(null) // 正在切换的插件 key（禁重复点）
const expanded = ref<Set<string>>(new Set()) // 抽屉内展开的品牌名集合（默认空=全折叠，按需手动展开）
function toggleCollapse(brand: string) {
  const s = new Set(expanded.value)
  s.has(brand) ? s.delete(brand) : s.add(brand)
  expanded.value = s
}
// 抽屉内按品牌分组列插件开关，复用卡片墙同款品牌聚合。
const settingBrands = computed(() => {
  const map = new Map<string, PluginMeta[]>()
  for (const p of installed.value) {
    const b = p.brand || p.key
    if (!map.has(b)) map.set(b, [])
    map.get(b)!.push(p)
  }
  const groups = Array.from(map, ([brand, items]) => ({
    brand,
    items: [...items].sort((a, b) => (a.form || a.key).localeCompare(b.form || b.key)),
  }))
  groups.sort((a, b) => b.items.length - a.items.length || a.brand.localeCompare(b.brand))
  return groups
})
const disabledCount = computed(() => installed.value.filter((p) => !p.enabled).length)

async function togglePlugin(p: PluginMeta) {
  if (toggling.value) return
  const next = !p.enabled
  toggling.value = p.key
  try {
    await setPluginStatus(p.key, next)
    p.enabled = next // 就地更新，避免整表重拉抖动
    toast.success(`${p.form || p.showname} 已${next ? '启用' : '禁用'}`)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '切换插件状态失败')
  } finally {
    toggling.value = null
  }
}

// 品牌整体是否启用：只要该品牌下还有一个形态启用即视为「开」，全关才算「关」。
function brandEnabled(grp: { items: PluginMeta[] }): boolean {
  return grp.items.some((p) => p.enabled)
}
const brandToggling = ref<string | null>(null) // 正在整品牌切换的品牌名

// 抽屉内品牌级一键：关停/开启整个品牌下所有支付形态（批量接口，一次落库）。
async function toggleBrand(grp: { brand: string; items: PluginMeta[] }) {
  if (brandToggling.value) return
  const next = !brandEnabled(grp) // 当前有启用→整品牌关；全关→整品牌开
  const keys = grp.items.map((p) => p.key)
  brandToggling.value = grp.brand
  try {
    await setPluginsStatus(keys, next)
    for (const p of grp.items) p.enabled = next // 就地更新整品牌
    toast.success(`${grp.brand} 全部形态已${next ? '启用' : '停用'}`)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '切换品牌状态失败')
  } finally {
    brandToggling.value = null
  }
}

// ===== 下栏：epay 全集参考（51 插件名录 mock，规划参考，默认折叠）=====
const refOpen = ref(false)
const stats = computed(() => calcPluginStats(plugins))
const kw = ref('')
const filtered = computed(() => {
  const v = kw.value.trim().toLowerCase()
  if (!v) return plugins
  return plugins.filter(
    (p) => p.name.toLowerCase().includes(v) || p.showname.toLowerCase().includes(v),
  )
})
const installedKeys = computed(() => new Set(installed.value.map((p) => p.key)))
</script>

<template>
  <div class="space-y-2.5">
    <!-- 上栏：本站已实现插件 —— 品牌能力卡片墙 -->
    <Panel
      title="已实现支付插件"
      :subtitle="`本站编译期内置、可用于收单的支付渠道 —— 共 ${summary.channels} 个形态，归属 ${summary.brands} 个支付品牌`"
    >
      <template #actions>
        <Button
          variant="outline"
          size="sm"
          :disabled="installedLoading || !installed.length"
          @click="settingsOpen = true"
        >
          <SlidersHorizontal />功能设置
          <Badge v-if="disabledCount" variant="muted">已停用 {{ disabledCount }}</Badge>
        </Button>
        <Button variant="outline" size="sm" :disabled="installedLoading" @click="loadInstalled">
          <RefreshCw :class="installedLoading ? 'animate-spin' : ''" />刷新
        </Button>
      </template>

      <div v-if="installedLoading" class="py-10 text-center dim">加载中…</div>
      <div v-else-if="!installed.length" class="py-10 text-center dim">暂无已注册插件</div>
      <template v-else>
        <!-- 概览统计栏：浅灰底一条，项间细分隔线（软风格，不用硬边框宫格）-->
        <div class="mb-3 flex flex-wrap items-center gap-x-8 gap-y-3 bg-muted/40 px-5 py-3.5">
          <div class="flex items-baseline gap-2">
            <span class="text-2xl font-semibold tabular-nums leading-none">{{ summary.channels }}</span>
            <span class="text-xs text-muted-foreground">支付形态</span>
          </div>
          <span class="h-6 w-px bg-border/60"></span>
          <div class="flex items-baseline gap-2">
            <span class="text-2xl font-semibold tabular-nums leading-none">{{ summary.brands }}</span>
            <span class="text-xs text-muted-foreground">支付品牌</span>
          </div>
          <span class="h-6 w-px bg-border/60"></span>
          <div class="flex items-baseline gap-2">
            <span class="text-2xl font-semibold tabular-nums leading-none text-success">{{ summary.refund }}</span>
            <span class="text-xs text-muted-foreground">支持退款</span>
          </div>
          <span class="h-6 w-px bg-border/60"></span>
          <div class="flex items-baseline gap-2">
            <span class="text-2xl font-semibold tabular-nums leading-none text-success">{{ summary.transfer }}</span>
            <span class="text-xs text-muted-foreground">支持代付</span>
          </div>
        </div>

        <!-- 全部插件被停用时的空态（仍可从「功能设置」开回来）-->
        <div v-if="!brandCards.length" class="border border-dashed border-border/60 bg-muted/20 py-10 text-center dim">
          全部支付插件已停用，点右上「功能设置」可重新启用
        </div>

        <!-- 卡片按 320~420px 自适应排布 + items-start 按内容自然高度（不强制同行等高，避免内容少的卡撑空白）-->
        <div
          v-else
          class="grid items-start gap-4"
          style="grid-template-columns: repeat(auto-fill, minmax(320px, 420px))"
        >
          <!-- 每个支付品牌一张能力卡（柔边圆角 + hover 微投影）-->
          <div
            v-for="card in brandCards"
            :key="card.brand"
            class="group flex flex-col overflow-hidden rounded-lg border border-border/60 bg-card transition-shadow hover:shadow-md hover:shadow-black/[0.04]"
          >
            <!-- 卡头：品牌淡底图标 + 名 + 能力标签 -->
            <div class="flex items-center gap-3 px-4 pt-4">
              <span
                class="flex size-10 shrink-0 items-center justify-center rounded-lg p-2"
                :class="[brandTint(card.brand), brandAccent(card.brand)]"
              >
                <BrandLogo :brand="card.brand" />
              </span>
              <div class="min-w-0 flex-1">
                <div class="truncate text-sm font-semibold leading-tight">{{ card.brand }}</div>
                <div class="mt-0.5 text-[11px] text-muted-foreground">{{ card.items.length }} 个支付形态</div>
              </div>
              <div class="flex shrink-0 flex-col items-end gap-1">
                <Badge v-if="card.refund" variant="success">可退款</Badge>
                <Badge v-if="card.transfer" variant="success">可代付</Badge>
              </div>
            </div>

            <!-- 按协议分区：协议名做小胶囊标签 + 该协议下形态 chip（无边框柔胶囊）-->
            <div class="space-y-2.5 px-4 py-3.5">
              <div v-for="g in card.groups" :key="g.protocol" class="flex flex-wrap items-center gap-1.5">
                <span class="rounded bg-muted/70 px-1.5 py-0.5 text-[10px] font-medium tracking-wide text-muted-foreground">
                  {{ g.protocol }}
                </span>
                <span
                  v-for="p in g.items"
                  :key="p.key"
                  class="inline-flex items-center rounded-md bg-muted/40 px-2 py-1 text-[12.5px] text-foreground/80 transition-colors group-hover:bg-muted/60"
                  :title="p.key"
                >
                  {{ p.form || p.showname }}
                </span>
              </div>
            </div>

            <!-- 卡脚：动态表单标记 + 去配置（浅灰底与卡身区分，不用硬分割线）-->
            <div class="mt-auto flex items-center gap-2 bg-muted/30 px-4 py-2.5">
              <span
                v-if="card.configurable"
                class="inline-flex items-center gap-1 text-[11px] text-muted-foreground"
              >
                <Settings2 class="size-3" />动态密钥表单
              </span>
              <Button variant="outline" size="sm" class="ml-auto shrink-0" @click="goConfigure(card)">
                去配置
              </Button>
            </div>
          </div>
        </div>
      </template>

      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        本站采用 Go 编译期注册，插件随程序内置，无需上传源码。同一品牌的多种协议/形态是独立渠道，「去配置」将跳转通道管理页并按该品牌预筛。
      </p>
    </Panel>

    <!-- 下栏：epay 全集参考（默认折叠）-->
    <Panel flush>
      <button
        type="button"
        class="flex w-full items-center gap-2 px-6 py-4 text-left transition-colors hover:bg-muted/40"
        @click="refOpen = !refOpen"
      >
        <Puzzle class="size-4 text-muted-foreground" />
        <span class="text-sm font-semibold">epay 插件全集参考</span>
        <Badge variant="muted">{{ stats.total }} 个</Badge>
        <span class="text-xs text-muted-foreground">来自 epay 目录名录，非本站已实现，仅作对接规划参考</span>
        <ChevronDown
          class="ml-auto size-4 text-muted-foreground transition-transform"
          :class="refOpen ? 'rotate-180' : ''"
        />
      </button>

      <div v-if="refOpen" class="border-t border-border/60">
        <div class="flex items-center justify-between px-6 py-3">
          <span class="text-xs text-muted-foreground">{{ filtered.length }} / {{ stats.total }} 个</span>
          <div class="relative">
            <Search class="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <input v-model="kw" placeholder="搜索插件名称 / 描述" class="field-input w-56 !pl-9" />
          </div>
        </div>
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
        <p class="px-6 py-3 text-xs text-muted-foreground">
          标「已实现」者为本站已内置可用，其余需按资质分级逐步实现。
        </p>
      </div>
    </Panel>

    <!-- 功能设置抽屉：按插件启用/禁用（后端持久化，禁用后收单选通道跳过该插件）-->
    <Drawer v-model="settingsOpen" title="插件功能设置" subtitle="停用后收单选通道将跳过该支付形态，收银台也不再列出" width="max-w-xl">
      <div v-if="!installed.length" class="py-10 text-center dim">暂无已注册插件</div>
      <div v-else class="space-y-4">
        <div v-for="grp in settingBrands" :key="grp.brand">
          <!-- 品牌头：点整行折叠/展开；右侧「全部」开关 @click.stop 不触发折叠 -->
          <div
            class="mb-1.5 flex cursor-pointer select-none items-center gap-2 rounded px-1 py-0.5 hover:bg-muted/40"
            @click="toggleCollapse(grp.brand)"
          >
            <ChevronDown
              class="size-3.5 shrink-0 text-muted-foreground transition-transform"
              :class="expanded.has(grp.brand) ? '' : '-rotate-90'"
            />
            <span
              class="flex size-6 shrink-0 items-center justify-center rounded p-1"
              :class="[brandTint(grp.brand), brandAccent(grp.brand)]"
            >
              <BrandLogo :brand="grp.brand" />
            </span>
            <span class="text-[13px] font-semibold">{{ grp.brand }}</span>
            <span class="text-[11px] text-muted-foreground">
              {{ grp.items.filter((p) => p.enabled).length }}/{{ grp.items.length }} 启用
            </span>
            <!-- 整品牌一键开关（放右侧，与逐形态开关区分：这条控制该品牌全部）-->
            <div class="ml-auto flex items-center gap-1.5" @click.stop>
              <span class="text-[11px] text-muted-foreground">全部</span>
              <Switch
                size="sm"
                :model-value="brandEnabled(grp)"
                :disabled="brandToggling === grp.brand"
                @update:model-value="toggleBrand(grp)"
              />
            </div>
          </div>
          <div v-show="expanded.has(grp.brand)" class="divide-y divide-border/50 border border-border/60">
            <div
              v-for="p in grp.items"
              :key="p.key"
              class="flex items-center gap-3 bg-muted/20 px-3 py-2.5"
            >
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-1.5 text-[13px] font-medium">
                  {{ p.form || p.showname }}
                  <Badge variant="outline">{{ p.protocol }}</Badge>
                </div>
                <div class="mt-0.5 font-mono text-[11px] text-muted-foreground">{{ p.key }}</div>
              </div>
              <span class="shrink-0 text-xs" :class="p.enabled ? 'text-success' : 'text-muted-foreground'">
                {{ p.enabled ? '已启用' : '已停用' }}
              </span>
              <Switch
                :model-value="p.enabled"
                :disabled="toggling === p.key"
                @update:model-value="togglePlugin(p)"
              />
            </div>
          </div>
        </div>
      </div>
    </Drawer>
  </div>
</template>
