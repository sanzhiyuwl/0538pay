<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Search, RotateCcw, ExternalLink } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination } from '@/components/ui'
import { fetchLogs, type LoginLog } from '@/lib/api/stats'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const columnOptions = [
  { value: 'uid', label: '商户号' },
  { value: 'ip', label: '操作IP' },
]

// ===== 筛选（column 精确等值，对齐 epay）=====
const filters = reactive({ column: 'uid', value: '' })
const page = ref(1)
const pageSize = 15
const total = ref(0)
const rows = ref<LoginLog[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await fetchLogs({
      page: page.value, pageSize,
      column: filters.value.trim() ? filters.column : undefined,
      value: filters.value.trim() || undefined,
    })
    rows.value = res.list
    total.value = res.total
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载登录日志失败')
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}
function applySearch() {
  page.value = 1
  load()
}
function resetFilters() {
  filters.column = 'uid'
  filters.value = ''
  applySearch()
}
function go(p: number) {
  page.value = p
  load()
}
onMounted(load)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

function typeVariant(type: string): 'default' | 'success' | 'destructive' | 'muted' {
  if (type === '登录失败') return 'destructive'
  if (type === '登录后台') return 'default'
  return 'success'
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="登录日志" :subtitle="`共 ${total} 条`">
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">日志搜索</label>
          <Select v-model="filters.column" :options="columnOptions" class="w-28" />
          <input v-model="filters.value" placeholder="精确匹配（0 为管理员）" class="field-input w-52" @keyup.enter="applySearch" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="applySearch"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="登录日志列表" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[10%]">ID</th>
              <th class="w-[14%]">商户号</th>
              <th class="w-[20%]">操作类型</th>
              <th class="w-[28%]">操作 IP</th>
              <th class="w-[28%]">时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="l in rows" :key="l.id">
              <td class="tabular-nums dim">{{ l.id }}</td>
              <td>
                <span v-if="l.uid > 0" class="font-medium tabular-nums text-primary">{{ l.uid }}</span>
                <span v-else class="dim">管理员</span>
              </td>
              <td><Badge :variant="typeVariant(l.type)">{{ l.type }}</Badge></td>
              <td>
                <a
                  :href="`https://www.ip138.com/iplookup.php?ip=${l.ip}`"
                  target="_blank"
                  rel="noreferrer"
                  class="inline-flex items-center gap-1 tabular-nums hover:text-primary"
                >
                  {{ l.ip }}<ExternalLink class="size-3 opacity-50" />
                </a>
              </td>
              <td class="text-xs">{{ l.date }}</td>
            </tr>
            <tr v-if="loading">
              <td colspan="5" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!rows.length">
              <td colspan="5" class="py-10 text-center dim">没有符合条件的登录日志</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="page" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>
  </div>
</template>
