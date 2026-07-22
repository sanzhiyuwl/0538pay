<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { Search, RotateCcw, Plus, MoreHorizontal, Check, X, Trash2, ExternalLink } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination, Drawer } from '@/components/ui'
import {
  fetchDomains, fetchDomainStats, addDomain, setDomainStatus, deleteDomain, batchOpDomain,
  type DomainItem, type DomainStats,
} from '@/lib/api/risk'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { shouldDropUp } from '@/composables/useRowMenu'

const toast = useToast()

const domainStatus: Record<number, { text: string; variant: 'default' | 'success' | 'destructive' }> = {
  0: { text: '审核中', variant: 'default' },
  1: { text: '正常', variant: 'success' },
  2: { text: '拒绝', variant: 'destructive' },
}
const statusOptions = [
  { value: -1, label: '全部状态' },
  ...Object.entries(domainStatus).map(([k, s]) => ({ value: Number(k), label: s.text })),
]

// ===== 筛选 =====
const filters = reactive({ kw: '', uid: '', dstatus: -1 })
const page = ref(1)
const pageSize = 15
const total = ref(0)
const rows = ref<DomainItem[]>([])
const loading = ref(false)
const stats = ref<DomainStats | null>(null)

async function load() {
  loading.value = true
  try {
    const uidNum = Number(filters.uid.trim())
    const res = await fetchDomains({
      page: page.value, pageSize,
      kw: filters.kw.trim() || undefined,
      uid: filters.uid.trim() && !Number.isNaN(uidNum) ? uidNum : undefined,
      dstatus: filters.dstatus > -1 ? filters.dstatus : undefined,
    })
    rows.value = res.list
    total.value = res.total
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载域名失败')
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}
async function loadStats() {
  try {
    stats.value = await fetchDomainStats()
  } catch {
    stats.value = null
  }
}
async function reload() {
  selected.value.clear()
  await Promise.all([load(), loadStats()])
}
function applySearch() {
  page.value = 1
  reload()
}
function resetFilters() {
  filters.kw = ''
  filters.uid = ''
  filters.dstatus = -1
  applySearch()
}
function go(p: number) {
  page.value = p
  load()
}
onMounted(reload)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

// ===== 多选 =====
const selected = ref<Set<number>>(new Set())
const pageAllChecked = computed(() => rows.value.length > 0 && rows.value.every((r) => selected.value.has(r.id)))
function toggleAll() {
  if (pageAllChecked.value) rows.value.forEach((r) => selected.value.delete(r.id))
  else rows.value.forEach((r) => selected.value.add(r.id))
}
function toggleOne(id: number) {
  if (selected.value.has(id)) selected.value.delete(id)
  else selected.value.add(id)
}

// ===== 行操作菜单 =====
const openMenu = ref<number | null>(null)
const dropUp = ref(false)
function toggleMenu(id: number, ev?: MouseEvent) {
  if (openMenu.value === id) { openMenu.value = null; return }
  openMenu.value = id
  dropUp.value = shouldDropUp(ev)
}
function closeMenu() {
  openMenu.value = null
}
onMounted(() => window.addEventListener('click', closeMenu))
onUnmounted(() => window.removeEventListener('click', closeMenu))

function domainHref(domain: string) {
  return 'http://' + domain.replace('*.', 'www.') + '/'
}

// 状态可执行操作
function rowActions(status: number): { key: string; label: string }[] {
  if (status === 1) return [{ key: 'reject', label: '改为拒绝' }, { key: 'delete', label: '删除' }]
  if (status === 2) return [{ key: 'pass', label: '改为通过' }, { key: 'delete', label: '删除' }]
  return [{ key: 'pass', label: '通过' }, { key: 'reject', label: '拒绝' }, { key: 'delete', label: '删除' }]
}
const actionIcons: Record<string, any> = { pass: Check, reject: X, delete: Trash2 }

// ===== 写操作 =====
const busy = ref(false)
async function doRowAction(d: DomainItem, key: string) {
  openMenu.value = null
  if (busy.value) return
  busy.value = true
  try {
    if (key === 'pass') await setDomainStatus(d.id, 1)
    else if (key === 'reject') await setDomainStatus(d.id, 2)
    else if (key === 'delete') await deleteDomain(d.id)
    toast.success('操作成功')
    await reload()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  } finally {
    busy.value = false
  }
}
async function doBatch(status: number) {
  if (busy.value || !selected.value.size) return
  busy.value = true
  try {
    const res = await batchOpDomain([...selected.value], status)
    toast.success(`已处理 ${res.affected} 个域名`)
    await reload()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '批量操作失败')
  } finally {
    busy.value = false
  }
}

// ===== 添加抽屉 =====
const addOpen = ref(false)
const form = reactive({ uid: '', domain: '' })
function openAdd() {
  form.uid = ''
  form.domain = ''
  addOpen.value = true
}
async function submitAdd() {
  const uidNum = Number(form.uid.trim())
  if (!uidNum || Number.isNaN(uidNum)) return toast.error('请输入有效商户号')
  if (!form.domain.trim()) return toast.error('请填写域名')
  if (busy.value) return
  busy.value = true
  try {
    await addDomain({ uid: uidNum, domain: form.domain.trim() })
    toast.success('已添加域名（后台添加免审核，直接生效）')
    addOpen.value = false
    await reload()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '添加失败')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="授权支付域名" :subtitle="`共 ${total} 个域名`">
      <template #actions>
        <Button size="sm" @click="openAdd"><Plus />添加域名</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">域名搜索</label>
          <input v-model="filters.kw" placeholder="精确匹配域名" class="field-input w-56" @keyup.enter="applySearch" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">商户号</label>
          <input v-model="filters.uid" placeholder="请输入商户号" class="field-input w-36" @keyup.enter="applySearch" />
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
    <Panel v-if="stats" title="域名审核概况">
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
        <Button variant="outline" size="sm" @click="doBatch(1)"><Check />通过</Button>
        <Button variant="outline" size="sm" @click="doBatch(2)"><X />拒绝</Button>
        <Button variant="outline" size="sm" @click="doBatch(3)"><Trash2 />删除</Button>
      </template>
      <div>
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
            <tr v-for="d in rows" :key="d.id">
              <td class="col-center">
                <input type="checkbox" :checked="selected.has(d.id)" @change="toggleOne(d.id)" />
              </td>
              <td class="tabular-nums dim">{{ d.id }}</td>
              <td class="font-medium tabular-nums text-primary">{{ d.uid }}</td>
              <td>
                <a :href="domainHref(d.domain)" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1 hover:text-primary">
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
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(d.id, $event)">
                    <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === d.id"
                    class="menu-panel absolute right-0 z-20 w-32"
                    :class="dropUp ? 'bottom-full mb-1.5' : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <button
                      v-for="(a, ai) in rowActions(d.status)"
                      :key="ai"
                      class="menu-item"
                      :class="a.key === 'delete' && 'menu-item-danger'"
                      @click="doRowAction(d, a.key)"
                    >
                      <component :is="actionIcons[a.key]" class="size-4 shrink-0 opacity-70" />
                      <span class="flex-1">{{ a.label }}</span>
                    </button>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="loading">
              <td colspan="8" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!rows.length">
              <td colspan="8" class="py-10 text-center dim">没有符合条件的域名</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="page" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>

    <!-- 添加抽屉 -->
    <Drawer v-model="addOpen" title="添加授权域名" subtitle="后台添加免审核，直接生效（status=正常）" width="max-w-md">
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">商户号</label>
          <input v-model="form.uid" class="field-input flex-1" placeholder="域名归属商户 UID" />
        </div>
        <div class="row-field">
          <label class="lbl">域名</label>
          <input v-model="form.domain" class="field-input flex-1" placeholder="如 pay.example.com 或 *.example.com" />
        </div>
        <p class="text-xs text-muted-foreground">支持 *. 通配符匹配主域名下所有子域名。域名白名单开启后，仅通过审核的域名可发起支付。</p>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="addOpen = false">取消</Button>
        <Button size="sm" :disabled="busy" @click="submitAdd">确认添加</Button>
      </template>
    </Drawer>
  </div>
</template>
