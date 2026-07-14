<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Search, RotateCcw, Plus, Trash2 } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination } from '@/components/ui'
import { blackItems, blackType, typeOptions, calcBlackStats } from '@/lib/mock/blacklist'

// ===== 筛选 =====
const filters = ref({ kw: '', type: -1 })

const filtered = computed(() => {
  return blackItems.filter((b) => {
    if (filters.value.type > -1 && b.type !== filters.value.type) return false
    if (filters.value.kw.trim() && !b.content.includes(filters.value.kw.trim())) return false
    return true
  })
})

function resetFilters() {
  filters.value = { kw: '', type: -1 }
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

const stats = computed(() => calcBlackStats(filtered.value))

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

// 筛选即时变化（改下拉/输入搜索词）时回到第1页并清空选中，避免批量操作命中不可见行
watch(filters, () => {
  page.value = 1
  selected.value.clear()
}, { deep: true })
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="支付黑名单" :subtitle="`共 ${total} 条`">
      <template #actions>
        <Button size="sm"><Plus />添加黑名单</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">内容搜索</label>
          <input v-model="filters.kw" placeholder="黑名单内容" class="field-input w-52" />
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
    <Panel title="黑名单概况" subtitle="按当前筛选条件">
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
        <Button variant="outline" size="sm" @click="selected.clear()"><Trash2 />批量删除</Button>
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
            <tr v-for="b in pageRows" :key="b.id">
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
                <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive">
                  <Trash2 class="size-4" />
                </Button>
              </td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="8" class="py-10 text-center dim">没有符合条件的黑名单</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="safePage" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        支付账号黑名单仅支持微信公众号支付和支付宝 JS 支付；IP 黑名单对全部支付方式生效。有效期 0 天为永久拉黑。
      </p>
    </Panel>
  </div>
</template>
