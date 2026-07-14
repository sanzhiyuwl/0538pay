<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import {
  Search,
  RotateCcw,
  Download,
  Plus,
  Send,
  CheckCircle2,
  Layers,
  MoreHorizontal,
  Trash2,
  Clock,
  Undo2,
} from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination } from '@/components/ui'
import {
  batches,
  settleRecords,
  settleTypes,
  settleStatus,
  batchStatus,
  calcSettleStats,
} from '@/lib/mock/settle'
import { formatMoney } from '@/lib/utils'

// ===== 结算方式 / 状态下拉 =====
const typeOptions = [
  { value: 0, label: '所有结算方式' },
  ...Object.entries(settleTypes).map(([k, t]) => ({ value: Number(k), label: t.showname })),
]
const statusOptions = [
  { value: -1, label: '全部状态' },
  ...Object.entries(settleStatus).map(([k, s]) => ({ value: Number(k), label: s.text })),
]
const batchOptions = [
  { value: '', label: '全部批次' },
  ...batches.map((b) => ({ value: b.batch, label: b.batch })),
]

// ===== 筛选 =====
const filters = ref({
  value: '', // 结算账号/姓名
  uid: '',
  batch: '',
  type: 0,
  dstatus: -1,
})

const filtered = computed(() => {
  return settleRecords.filter((r) => {
    if (filters.value.uid && String(r.uid) !== filters.value.uid.trim()) return false
    if (filters.value.batch && r.batch !== filters.value.batch) return false
    if (filters.value.type && r.type !== filters.value.type) return false
    if (filters.value.dstatus > -1 && r.status !== filters.value.dstatus) return false
    if (filters.value.value.trim()) {
      const v = filters.value.value.trim()
      if (!`${r.account}${r.username}`.includes(v)) return false
    }
    return true
  })
})

function resetFilters() {
  filters.value = { value: '', uid: '', batch: '', type: 0, dstatus: -1 }
  page.value = 1
  selected.value.clear()
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

// 搜索 / 切换筛选：回到第一页并清空跨筛选的选中，避免误批量操作不可见记录
function applySearch() {
  page.value = 1
  selected.value.clear()
}
function viewBatch(batch: string) {
  filters.value.batch = batch
  applySearch()
}

// ===== 明细汇总 =====
const stats = computed(() => calcSettleStats(filtered.value))

// ===== 多选（批量修改状态）=====
const selected = ref<Set<number>>(new Set())
const pageAllChecked = computed(
  () => pageRows.value.length > 0 && pageRows.value.every((r) => selected.value.has(r.id)),
)
function toggleAll() {
  if (pageAllChecked.value) {
    pageRows.value.forEach((r) => selected.value.delete(r.id))
  } else {
    pageRows.value.forEach((r) => selected.value.add(r.id))
  }
}
function toggleOne(id: number) {
  if (selected.value.has(id)) selected.value.delete(id)
  else selected.value.add(id)
}

// 筛选即时变化时回到第1页并清空选中，避免批量操作命中不可见行
watch(filters, () => {
  page.value = 1
  selected.value.clear()
}, { deep: true })

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

// 结算方式对应图标底色（用文字缩写块，无需外部图标资源）
function typeInitial(type: number) {
  return settleTypes[type]?.showname?.[0] ?? '?'
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 结算批次 -->
    <Panel title="结算管理" subtitle="按批次归集待结算记录，通过转账接口或手动打款完成结算">
      <template #actions>
        <Button size="sm"><Plus />生成结算批次</Button>
      </template>
      <div class="grid grid-cols-1 gap-2.5 sm:grid-cols-2 lg:grid-cols-4">
        <div
          v-for="b in batches"
          :key="b.batch"
          class="group bg-muted/40 p-3.5 transition-colors hover:bg-muted/70"
        >
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-1.5 text-sm font-medium">
              <Layers class="size-4 text-primary" />
              <span class="tabular-nums">{{ b.batch }}</span>
            </div>
            <Badge :variant="batchStatus[b.status].variant">{{ batchStatus[b.status].text }}</Badge>
          </div>
          <div class="mt-3 flex items-end justify-between">
            <div>
              <div class="text-[13px] text-muted-foreground">批次金额</div>
              <div class="mt-0.5 text-lg font-semibold tabular-nums">
                <span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(+b.allmoney) }}
              </div>
            </div>
            <div class="text-right">
              <div class="text-[13px] text-muted-foreground">笔数</div>
              <div class="mt-0.5 text-lg font-semibold tabular-nums">{{ b.count }}</div>
            </div>
          </div>
          <div class="mt-3 flex items-center gap-2 border-t border-border/60 pt-2.5 text-xs text-muted-foreground">
            <Clock class="size-3.5" />{{ b.time }}
          </div>
          <div class="mt-2.5 flex flex-wrap gap-1.5">
            <Button variant="outline" size="sm" @click="viewBatch(b.batch)">
              结算列表
            </Button>
            <Button variant="outline" size="sm"><Send />批量转账</Button>
            <Button variant="ghost" size="sm"><Download /></Button>
          </div>
        </div>
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        结算标准：可结算余额大于 <b class="text-foreground">100</b> 元，或商户主动申请。
        生成批次后所有「待结算」记录转为「正在结算」，转账成功后自动置为「已完成」；手动打款需手动改为完成。
      </p>
    </Panel>

    <!-- 明细汇总 -->
    <Panel title="结算概况" subtitle="按当前筛选条件">
      <div class="grid grid-cols-2 gap-x-8 gap-y-5 sm:grid-cols-3 lg:grid-cols-6">
        <div>
          <div class="text-[13px] text-muted-foreground">结算总金额</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.totalMoney) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">实际到账</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-primary"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.realMoney) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已完成金额</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-success"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.doneMoney) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">记录总数</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums">{{ stats.totalCount }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">正在结算 / 失败</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums">
            <span class="text-warning">{{ stats.processingCount }}</span>
            <span class="text-sm text-muted-foreground">/</span>
            <span class="text-destructive">{{ stats.failCount }}</span>
          </div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已完成笔数</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-success">{{ stats.doneCount }}</div>
        </div>
      </div>
    </Panel>

    <!-- 筛选 -->
    <Panel title="结算明细" :subtitle="`共 ${total} 条记录`">
      <template #actions>
        <Button variant="outline" size="sm"><Download />导出列表</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">账号 / 姓名</label>
          <input v-model="filters.value" placeholder="结算账号或姓名" class="field-input w-48" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">商户号</label>
          <input v-model="filters.uid" placeholder="请输入商户号" class="field-input w-36" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">批次</label>
          <Select v-model="filters.batch" :options="batchOptions" class="w-44" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">结算方式</label>
          <Select v-model="filters.type" :options="typeOptions" class="w-32" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">状态</label>
          <Select v-model="filters.dstatus" :options="statusOptions" class="w-28" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="applySearch"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 明细列表 -->
    <Panel title="结算记录" :subtitle="selected.size ? `已选 ${selected.size} 条` : `${total} 条`">
      <template v-if="selected.size" #actions>
        <span class="text-sm text-muted-foreground">批量修改为：</span>
        <Button variant="outline" size="sm" @click="selected.clear()"><CheckCircle2 />已完成</Button>
        <Button variant="outline" size="sm" @click="selected.clear()"><Clock />正在结算</Button>
        <Button variant="outline" size="sm" @click="selected.clear()"><Trash2 />删除退回</Button>
      </template>
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[4%] col-center">
                <input type="checkbox" :checked="pageAllChecked" @change="toggleAll" />
              </th>
              <th class="w-[8%]">ID</th>
              <th class="w-[13%]">商户</th>
              <th class="w-[14%]">结算方式</th>
              <th class="w-[18%]">结算账号 / 姓名</th>
              <th class="w-[14%]">结算金额 / 到账</th>
              <th class="w-[15%]">创建 / 完成时间</th>
              <th class="col-center w-[8%]">状态</th>
              <th class="col-center w-[6%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(r, si) in pageRows" :key="r.id">
              <td class="col-center">
                <input type="checkbox" :checked="selected.has(r.id)" @change="toggleOne(r.id)" />
              </td>
              <td class="font-medium tabular-nums">{{ r.id }}</td>
              <td>
                <div class="truncate">{{ r.merchant }}</div>
                <div class="text-xs dim tabular-nums">{{ r.uid }}</div>
              </td>
              <td>
                <div class="flex items-center gap-1.5">
                  <span class="grid size-5 place-items-center rounded bg-primary/10 text-[11px] font-medium text-primary">
                    {{ typeInitial(r.type) }}
                  </span>
                  <span>{{ settleTypes[r.type].showname }}</span>
                  <span v-if="!r.auto" class="text-xs dim">[手动]</span>
                </div>
              </td>
              <td>
                <div class="truncate tabular-nums">{{ r.account }}</div>
                <div class="text-xs dim">{{ r.username }}</div>
              </td>
              <td>
                <div class="tabular-nums"><span class="dim text-xs">¥</span><b>{{ r.money }}</b></div>
                <div class="text-xs dim tabular-nums">到账 ¥{{ r.realmoney }}</div>
              </td>
              <td>
                <div class="text-xs">{{ r.addtime }}</div>
                <div class="text-xs dim">{{ r.endtime ?? '—' }}</div>
              </td>
              <td class="col-center">
                <Badge :variant="settleStatus[r.status].variant">{{ settleStatus[r.status].text }}</Badge>
                <div v-if="r.status === 3" class="mt-1 truncate text-xs text-destructive" :title="r.result">
                  {{ r.result }}
                </div>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(r.id)">
                    <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === r.id"
                    class="menu-panel absolute right-0 z-20 w-36"
                    :class="si >= pageRows.length - 3 && pageRows.length > 3
                      ? 'bottom-full mb-1.5'
                      : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <button class="menu-item" @click="openMenu = null">
                      <CheckCircle2 class="size-4 shrink-0 opacity-70" /><span class="flex-1">改为已完成</span>
                    </button>
                    <button class="menu-item" @click="openMenu = null">
                      <Clock class="size-4 shrink-0 opacity-70" /><span class="flex-1">改为正在结算</span>
                    </button>
                    <button class="menu-item" @click="openMenu = null">
                      <RotateCcw class="size-4 shrink-0 opacity-70" /><span class="flex-1">改为待结算</span>
                    </button>
                    <button class="menu-item" @click="openMenu = null">
                      <Send class="size-4 shrink-0 opacity-70" /><span class="flex-1">重新转账</span>
                    </button>
                    <div class="menu-sep" />
                    <button class="menu-item menu-item-danger" @click="openMenu = null">
                      <Undo2 class="size-4 shrink-0 opacity-70" /><span class="flex-1">删除并退回</span>
                    </button>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="9" class="py-10 text-center dim">没有符合条件的结算记录</td>
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
