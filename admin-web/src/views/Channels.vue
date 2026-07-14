<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import {
  Search,
  RotateCcw,
  Plus,
  MoreHorizontal,
  KeyRound,
  Pencil,
  Trash2,
  ReceiptText,
  Copy,
  FlaskConical,
} from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Switch, Pagination } from '@/components/ui'
import {
  channels as allChannels,
  channelMode,
  typeOptions,
  calcChannelStats,
} from '@/lib/mock/channels'
import { formatMoney } from '@/lib/utils'

const statusOptions = [
  { value: -1, label: '全部状态' },
  { value: 1, label: '状态已开启' },
  { value: 0, label: '状态已关闭' },
]

// ===== 筛选 =====
const filters = ref({ kw: '', plugin: '', type: 0, dstatus: -1 })

const filtered = computed(() => {
  return allChannels.filter((c) => {
    if (filters.value.type && c.type !== filters.value.type) return false
    if (filters.value.dstatus > -1 && c.status !== filters.value.dstatus) return false
    if (filters.value.plugin.trim() && !c.plugin.includes(filters.value.plugin.trim())) return false
    if (filters.value.kw.trim()) {
      const v = filters.value.kw.trim()
      if (!`${c.id}${c.name}`.includes(v)) return false
    }
    return true
  })
})

function resetFilters() {
  filters.value = { kw: '', plugin: '', type: 0, dstatus: -1 }
  page.value = 1
}

// ===== 分页 =====
const page = ref(1)
const pageSize = 15
const total = computed(() => filtered.value.length)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const safePage = computed(() => Math.min(page.value, pageCount.value))
const pageRows = computed(() =>
  filtered.value.slice((safePage.value - 1) * pageSize, safePage.value * pageSize),
)
function go(p: number) {
  page.value = Math.min(Math.max(1, p), pageCount.value)
}

// ===== 概况 =====
const stats = computed(() => calcChannelStats(filtered.value))

// ===== 状态切换（本地 mock，直接改对象）=====
function toggleStatus(id: number) {
  const c = allChannels.find((x) => x.id === id)
  if (c) c.status = c.status === 1 ? 0 : 1
}

// ===== 行操作菜单 =====
const openMenu = ref<number | null>(null)
function toggleMenu(id: number) {
  openMenu.value = openMenu.value === id ? null : id
}
function closeMenu() {
  openMenu.value = null
}
onMounted(() => window.addEventListener('click', closeMenu))
onUnmounted(() => window.removeEventListener('click', closeMenu))
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="支付通道" :subtitle="`共 ${total} 个通道`">
      <template #actions>
        <Button size="sm"><Plus />新增通道</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">通道搜索</label>
          <input v-model="filters.kw" placeholder="通道 ID / 名称" class="field-input w-44" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">支付插件</label>
          <input v-model="filters.plugin" placeholder="插件标识" class="field-input w-32" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">支付方式</label>
          <Select v-model="filters.type" :options="typeOptions" class="w-32" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">状态</label>
          <Select v-model="filters.dstatus" :options="statusOptions" class="w-32" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="page = 1"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 概况 -->
    <Panel title="通道概况" subtitle="按当前筛选条件">
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">通道总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.total }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已开启</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ stats.open }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已关闭</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-muted-foreground">{{ stats.closed }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">今日收款合计</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-primary"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.todayTotal) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">昨日收款合计</div>
          <div class="mt-1 text-xl font-normal tabular-nums"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.yesterdayTotal) }}</div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="通道列表" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[6%]">ID</th>
              <th class="w-[16%]">显示名称</th>
              <th class="w-[10%]">通道模式</th>
              <th class="w-[10%]">支付方式</th>
              <th class="w-[12%]">支付插件</th>
              <th class="num w-[9%]">分成比例</th>
              <th class="num w-[12%]">今日收款</th>
              <th class="num w-[12%]">昨日收款</th>
              <th class="col-center w-[7%]">状态</th>
              <th class="col-center w-[6%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(c, si) in pageRows" :key="c.id">
              <td class="font-medium tabular-nums">{{ c.id }}</td>
              <td class="truncate">{{ c.name }}</td>
              <td>
                <Badge :variant="c.mode === 1 ? 'warning' : 'muted'">{{ channelMode[c.mode] }}</Badge>
              </td>
              <td>{{ c.typeshowname }}</td>
              <td>
                <span class="truncate text-primary" :title="c.plugin">{{ c.plugin }}</span>
              </td>
              <td class="num tabular-nums">{{ c.rate }}%</td>
              <td class="num tabular-nums">
                <span v-if="c.status === 1"><span class="dim text-xs">¥</span>{{ c.today }}</span>
                <span v-else class="dim">—</span>
              </td>
              <td class="num tabular-nums">
                <span v-if="+c.yesterday > 0"><span class="dim text-xs">¥</span>{{ c.yesterday }}</span>
                <span v-else class="dim">—</span>
              </td>
              <td class="col-center">
                <div class="flex justify-center">
                  <Switch :model-value="c.status === 1" size="sm" @update:model-value="toggleStatus(c.id)" />
                </div>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(c.id)">
                    <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === c.id"
                    class="menu-panel absolute right-0 z-20 w-32"
                    :class="si >= pageRows.length - 3 && pageRows.length > 3
                      ? 'bottom-full mb-1.5'
                      : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <button class="menu-item" @click="openMenu = null">
                      <KeyRound class="size-4 shrink-0 opacity-70" /><span class="flex-1">配置密钥</span>
                    </button>
                    <button class="menu-item" @click="openMenu = null">
                      <Pencil class="size-4 shrink-0 opacity-70" /><span class="flex-1">编辑</span>
                    </button>
                    <button class="menu-item" @click="openMenu = null">
                      <Copy class="size-4 shrink-0 opacity-70" /><span class="flex-1">复制</span>
                    </button>
                    <button class="menu-item" @click="openMenu = null">
                      <ReceiptText class="size-4 shrink-0 opacity-70" /><span class="flex-1">订单</span>
                    </button>
                    <button class="menu-item" @click="openMenu = null">
                      <FlaskConical class="size-4 shrink-0 opacity-70" /><span class="flex-1">测试支付</span>
                    </button>
                    <div class="menu-sep" />
                    <button class="menu-item menu-item-danger" @click="openMenu = null">
                      <Trash2 class="size-4 shrink-0 opacity-70" /><span class="flex-1">删除</span>
                    </button>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="10" class="py-10 text-center dim">没有符合条件的支付通道</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="safePage" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>
  </div>
</template>
