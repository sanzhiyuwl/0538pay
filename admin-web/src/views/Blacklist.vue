<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Search, RotateCcw, Plus, Trash2 } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination, Drawer, Modal } from '@/components/ui'
import {
  fetchBlacklist, fetchBlackStats, addBlacklist, deleteBlacklist, batchDeleteBlacklist,
  type BlackItem, type BlackStats,
} from '@/lib/api/risk'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const blackType: Record<number, { text: string; variant: 'default' | 'warning' }> = {
  0: { text: '支付账号', variant: 'default' },
  1: { text: 'IP地址', variant: 'warning' },
}
const typeOptions = [
  { value: -1, label: '黑名单类型' },
  { value: 0, label: '支付账号' },
  { value: 1, label: 'IP地址' },
]
const addTypeOptions = [
  { value: 0, label: '支付账号' },
  { value: 1, label: 'IP地址' },
]

// ===== 筛选（kw 精确等值，对齐 epay）=====
const filters = reactive({ kw: '', type: -1 })
const page = ref(1)
const pageSize = 15
const total = ref(0)
const rows = ref<BlackItem[]>([])
const loading = ref(false)
const stats = ref<BlackStats | null>(null)

async function load() {
  loading.value = true
  try {
    const res = await fetchBlacklist({
      page: page.value, pageSize,
      kw: filters.kw.trim() || undefined,
      type: filters.type > -1 ? filters.type : undefined,
    })
    rows.value = res.list
    total.value = res.total
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载黑名单失败')
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}
async function loadStats() {
  try {
    stats.value = await fetchBlackStats()
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
  filters.type = -1
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

// ===== 添加抽屉 =====
const busy = ref(false)
const addOpen = ref(false)
const form = reactive({ type: 0, content: '', days: 0, remark: '' })
function openAdd() {
  form.type = 0
  form.content = ''
  form.days = 0
  form.remark = ''
  addOpen.value = true
}
async function submitAdd() {
  if (!form.content.trim() || busy.value) return
  busy.value = true
  try {
    await addBlacklist({ type: form.type, content: form.content.trim(), days: form.days, remark: form.remark.trim() })
    toast.success('已添加黑名单')
    addOpen.value = false
    await reload()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '添加失败')
  } finally {
    busy.value = false
  }
}

// ===== 删除 =====
const delOpen = ref(false)
const delRow = ref<BlackItem | null>(null)
const batchMode = ref(false)
function askDelete(b: BlackItem) {
  delRow.value = b
  batchMode.value = false
  delOpen.value = true
}
function askBatchDelete() {
  batchMode.value = true
  delOpen.value = true
}
async function doDelete() {
  if (busy.value) return
  busy.value = true
  try {
    if (batchMode.value) {
      const res = await batchDeleteBlacklist([...selected.value])
      toast.success(`已删除 ${res.deleted} 条`)
    } else if (delRow.value) {
      await deleteBlacklist(delRow.value.id)
      toast.success('已删除')
    }
    delOpen.value = false
    await reload()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '删除失败')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="支付黑名单" :subtitle="`共 ${total} 条`">
      <template #actions>
        <Button size="sm" @click="openAdd"><Plus />添加黑名单</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">内容搜索</label>
          <input v-model="filters.kw" placeholder="精确匹配账号/IP" class="field-input w-52" @keyup.enter="applySearch" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">类型</label>
          <Select v-model="filters.type" :options="typeOptions" class="w-32" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="applySearch"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 概况 -->
    <Panel v-if="stats" title="黑名单概况">
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">黑名单总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.total }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">支付账号</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.account }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">IP 地址</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-warning">{{ stats.ip }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">永久拉黑</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-destructive">{{ stats.permanent }}</div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="黑名单列表" :subtitle="selected.size ? `已选 ${selected.size} 条` : `${total} 条`">
      <template v-if="selected.size" #actions>
        <Button variant="outline" size="sm" @click="askBatchDelete"><Trash2 />批量删除</Button>
      </template>
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="col-center w-[4%]">
                <input type="checkbox" :checked="pageAllChecked" @change="toggleAll" />
              </th>
              <th class="w-[8%]">ID</th>
              <th class="w-[12%]">类型</th>
              <th class="w-[22%]">黑名单内容</th>
              <th class="w-[16%]">添加时间</th>
              <th class="w-[16%]">过期时间</th>
              <th class="w-[14%]">备注</th>
              <th class="col-center w-[8%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="b in rows" :key="b.id">
              <td class="col-center">
                <input type="checkbox" :checked="selected.has(b.id)" @change="toggleOne(b.id)" />
              </td>
              <td class="tabular-nums dim">{{ b.id }}</td>
              <td>
                <Badge :variant="blackType[b.type].variant">{{ blackType[b.type].text }}</Badge>
              </td>
              <td class="truncate font-mono text-[13px]">{{ b.content }}</td>
              <td class="text-xs">{{ b.addtime }}</td>
              <td class="text-xs">
                <span v-if="b.endtime">{{ b.endtime }}</span>
                <span v-else class="text-destructive">永久</span>
              </td>
              <td class="truncate dim">{{ b.remark || '—' }}</td>
              <td class="col-center">
                <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="askDelete(b)">
                  <Trash2 class="size-4" />
                </Button>
              </td>
            </tr>
            <tr v-if="loading">
              <td colspan="8" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!rows.length">
              <td colspan="8" class="py-10 text-center dim">没有符合条件的黑名单</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="page" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        支付账号黑名单仅支持微信公众号支付和支付宝 JS 支付；IP 黑名单对全部支付方式生效。有效期 0 天为永久拉黑。
      </p>
    </Panel>

    <!-- 添加抽屉 -->
    <Drawer v-model="addOpen" title="添加黑名单" subtitle="拉黑支付账号或 IP，命中则拦截支付" width="max-w-md">
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">拉黑类型</label>
          <Select v-model="form.type" :options="addTypeOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">拉黑内容</label>
          <input v-model="form.content" class="field-input flex-1" :placeholder="form.type === 1 ? 'IP 地址' : '支付账号 openid/手机号'" />
        </div>
        <div class="row-field">
          <label class="lbl">有效期</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model.number="form.days" type="number" min="0" class="field-input w-24" />
            <span class="text-sm text-muted-foreground">天（0=永久）</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">备注</label>
          <input v-model="form.remark" maxlength="80" class="field-input flex-1" placeholder="选填" />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="addOpen = false">取消</Button>
        <Button size="sm" :disabled="!form.content.trim() || busy" @click="submitAdd">确认添加</Button>
      </template>
    </Drawer>

    <!-- 删除确认 -->
    <Modal v-model="delOpen" title="删除确认" width="max-w-md">
      <p class="text-sm text-muted-foreground">
        {{ batchMode ? `确认删除选中的 ${selected.size} 条黑名单？` : `确认删除黑名单「${delRow?.content}」？` }}删除后该账号/IP 将不再被拦截。
      </p>
      <template #footer>
        <Button variant="outline" size="sm" @click="delOpen = false">取消</Button>
        <Button size="sm" :disabled="busy" @click="doDelete">确认删除</Button>
      </template>
    </Modal>
  </div>
</template>
