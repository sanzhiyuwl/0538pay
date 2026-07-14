<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import {
  Search,
  RotateCcw,
  Plus,
  MoreHorizontal,
  Check,
  X,
  Trash2,
  ExternalLink,
} from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination } from '@/components/ui'
import {
  domains as allDomains,
  domainStatus,
  domainActions,
  calcDomainStats,
} from '@/lib/mock/domains'

const statusOptions = [
  { value: -1, label: '全部状态' },
  ...Object.entries(domainStatus).map(([k, s]) => ({ value: Number(k), label: s.text })),
]

// ===== 筛选 =====
const filters = ref({ kw: '', uid: '', dstatus: -1 })

const filtered = computed(() => {
  return allDomains.filter((d) => {
    if (filters.value.uid && String(d.uid) !== filters.value.uid.trim()) return false
    if (filters.value.dstatus > -1 && d.status !== filters.value.dstatus) return false
    if (filters.value.kw.trim() && !d.domain.includes(filters.value.kw.trim())) return false
    return true
  })
})

function resetFilters() {
  filters.value = { kw: '', uid: '', dstatus: -1 }
  applySearch()
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

function applySearch() {
  page.value = 1
  selected.value.clear()
}

// ===== 统计 =====
const stats = computed(() => calcDomainStats(filtered.value))

// ===== 多选 =====
const selected = ref<Set<number>>(new Set())
const pageAllChecked = computed(
  () => pageRows.value.length > 0 && pageRows.value.every((r) => selected.value.has(r.id)),
)
function toggleAll() {
  if (pageAllChecked.value) pageRows.value.forEach((r) => selected.value.delete(r.id))
  else pageRows.value.forEach((r) => selected.value.add(r.id))
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

const actionIcons: Record<string, any> = {
  通过: Check,
  改为通过: Check,
  拒绝: X,
  改为拒绝: X,
  删除: Trash2,
}

// 通配符域名转可访问链接
function domainHref(domain: string) {
  return 'http://' + domain.replace('*.', 'www.') + '/'
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="授权支付域名" :subtitle="`共 ${total} 个域名`">
      <template #actions>
        <Button size="sm"><Plus />添加域名</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">域名搜索</label>
          <input v-model="filters.kw" placeholder="要搜索的域名" class="field-input w-56" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">商户号</label>
          <input v-model="filters.uid" placeholder="请输入商户号" class="field-input w-36" />
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

    <!-- 概况 -->
    <Panel title="域名审核概况" subtitle="按当前筛选条件">
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">域名总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.total }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">待审核</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-primary">{{ stats.pending }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">正常</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ stats.normal }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已拒绝</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-destructive">{{ stats.rejected }}</div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="域名列表" :subtitle="selected.size ? `已选 ${selected.size} 个` : `${total} 条`">
      <template v-if="selected.size" #actions>
        <span class="text-sm text-muted-foreground">批量：</span>
        <Button variant="outline" size="sm" @click="selected.clear()"><Check />通过</Button>
        <Button variant="outline" size="sm" @click="selected.clear()"><X />拒绝</Button>
        <Button variant="outline" size="sm" @click="selected.clear()"><Trash2 />删除</Button>
      </template>
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="col-center w-[4%]">
                <input type="checkbox" :checked="pageAllChecked" @change="toggleAll" />
              </th>
              <th class="w-[8%]">ID</th>
              <th class="w-[10%]">商户号</th>
              <th class="w-[28%]">域名</th>
              <th class="w-[16%]">添加时间</th>
              <th class="w-[16%]">审核时间</th>
              <th class="col-center w-[9%]">状态</th>
              <th class="col-center w-[9%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(d, si) in pageRows" :key="d.id">
              <td class="col-center">
                <input type="checkbox" :checked="selected.has(d.id)" @change="toggleOne(d.id)" />
              </td>
              <td class="tabular-nums dim">{{ d.id }}</td>
              <td class="font-medium tabular-nums text-primary">{{ d.uid }}</td>
              <td>
                <a
                  :href="domainHref(d.domain)"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="inline-flex items-center gap-1 hover:text-primary"
                >
                  {{ d.domain }}
                  <ExternalLink class="size-3 opacity-50" />
                </a>
              </td>
              <td class="text-xs">{{ d.addtime }}</td>
              <td class="text-xs">{{ d.endtime ?? '—' }}</td>
              <td class="col-center">
                <Badge :variant="domainStatus[d.status].variant">{{ domainStatus[d.status].text }}</Badge>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(d.id)">
                    <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === d.id"
                    class="menu-panel absolute right-0 z-20 w-32"
                    :class="si >= pageRows.length - 3 && pageRows.length > 3
                      ? 'bottom-full mb-1.5'
                      : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <template v-for="(a, ai) in domainActions(d.status)" :key="ai">
                      <button
                        class="menu-item"
                        :class="a === '删除' && 'menu-item-danger'"
                        @click="openMenu = null"
                      >
                        <component :is="actionIcons[a]" class="size-4 shrink-0 opacity-70" />
                        <span class="flex-1">{{ a }}</span>
                      </button>
                    </template>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="8" class="py-10 text-center dim">没有符合条件的域名</td>
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
