<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Search, RotateCcw, Download, Eye } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, DateRange, Pagination, Modal } from '@/components/ui'
import {
  fetchSystemOpLogs,
  fetchSystemOpActionOptions,
  exportSystemOpLogs,
  agentOpLevelMeta,
  agentOpResultMeta,
  agentOpLevelOptions,
  agentOpResultOptions,
  type AgentOpLog,
} from '@/lib/api/console'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

// 动作下拉（系统事件登记表去重）
const actionOptions = ref<{ value: string; label: string }[]>([{ value: '', label: '全部事件' }])

const filters = reactive({
  action: '',
  level: '',
  result: '',
  keyword: '',
  starttime: '',
  endtime: '',
})

const page = ref(1)
const pageSize = 15
const total = ref(0)
const rows = ref<AgentOpLog[]>([])
const loading = ref(false)

function buildParams() {
  return {
    page: page.value,
    pageSize,
    action: filters.action || undefined,
    level: filters.level || undefined,
    result: filters.result || undefined,
    keyword: filters.keyword.trim() || undefined,
    starttime: filters.starttime || undefined,
    endtime: filters.endtime || undefined,
  }
}

async function load() {
  loading.value = true
  try {
    const res = await fetchSystemOpLogs(buildParams())
    rows.value = res.list
    total.value = res.total
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载运维日志失败')
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function search() {
  page.value = 1
  load()
}
function resetFilters() {
  filters.action = ''
  filters.level = ''
  filters.result = ''
  filters.keyword = ''
  filters.starttime = ''
  filters.endtime = ''
  page.value = 1
  load()
}
function go(p: number) {
  page.value = p
  load()
}

onMounted(async () => {
  try {
    const res = await fetchSystemOpActionOptions()
    actionOptions.value = [{ value: '', label: '全部事件' }, ...res.actions.map((a) => ({ value: a.value, label: a.label }))]
  } catch {
    // 保留默认
  }
  await load()
})

const exporting = ref(false)
async function exportList() {
  if (exporting.value) return
  exporting.value = true
  try {
    await exportSystemOpLogs({ ...buildParams(), page: undefined, pageSize: undefined })
    toast.success('已开始下载运维日志 CSV')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '导出失败')
  } finally {
    exporting.value = false
  }
}

const detailOpen = ref(false)
const detailRow = ref<AgentOpLog | null>(null)
function openDetail(row: AgentOpLog) {
  detailRow.value = row
  detailOpen.value = true
}
function parseDetail(detail: string): [string, string][] {
  if (!detail || !detail.trim()) return []
  try {
    const obj = JSON.parse(detail)
    if (obj && typeof obj === 'object') return Object.entries(obj).map(([k, v]) => [k, String(v)])
  } catch {
    // 非 JSON 忽略
  }
  return []
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="运维日志" subtitle="记录系统自动事件（提交微信 / 开通成功 / 微信驳回 / 驳回自动退款 / 名额冻结·消耗·释放 / 超时关单等），非人工触发">
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">事件类型</label>
          <Select v-model="filters.action" :options="actionOptions" class="w-40" />
        </div>
        <div class="filter-item">
          <label class="filter-label">级别</label>
          <Select v-model="filters.level" :options="agentOpLevelOptions" class="w-28" />
        </div>
        <div class="filter-item">
          <label class="filter-label">结果</label>
          <Select v-model="filters.result" :options="agentOpResultOptions" class="w-28" />
        </div>
        <div class="filter-item">
          <label class="filter-label">关键字</label>
          <input v-model="filters.keyword" placeholder="进件单号 / 对象" class="field-input w-40" @keyup.enter="search" />
        </div>
        <div class="filter-item">
          <label class="filter-label">时间</label>
          <DateRange v-model:start="filters.starttime" v-model:end="filters.endtime" class="w-[328px]" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="search"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
          <Button variant="outline" size="sm" :disabled="exporting" @click="exportList">
            <Download />{{ exporting ? '导出中…' : '导出' }}
          </Button>
        </div>
      </div>
    </Panel>

    <Panel title="运维日志列表" :subtitle="`共 ${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[6%]">ID</th>
              <th class="w-[12%]">来源</th>
              <th class="w-[16%]">事件</th>
              <th class="w-[9%]">级别</th>
              <th class="w-[25%]">对象</th>
              <th class="w-[8%]">结果</th>
              <th class="w-[16%]">时间</th>
              <th class="col-center w-[8%]">详情</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="l in rows" :key="l.id">
              <td class="tabular-nums dim">{{ l.id }}</td>
              <td class="truncate" :title="l.operator">{{ l.operator || '系统' }}</td>
              <td class="truncate">{{ l.actionCN }}</td>
              <td>
                <Badge :variant="agentOpLevelMeta[l.level]?.variant ?? 'muted'">
                  {{ agentOpLevelMeta[l.level]?.text ?? l.level }}
                </Badge>
              </td>
              <td class="truncate" :title="l.target">{{ l.target || '—' }}</td>
              <td>
                <Badge :variant="agentOpResultMeta[l.result]?.variant ?? 'muted'">
                  {{ agentOpResultMeta[l.result]?.text ?? l.result }}
                </Badge>
              </td>
              <td class="truncate text-xs">{{ l.date }}</td>
              <td class="col-center">
                <Button variant="ghost" size="sm" @click="openDetail(l)"><Eye class="size-4" /></Button>
              </td>
            </tr>
            <tr v-if="loading">
              <td colspan="8" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!rows.length">
              <td colspan="8" class="py-10 text-center dim">没有符合条件的运维日志</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination
          :page="page"
          :page-count="Math.max(1, Math.ceil(total / pageSize))"
          :total="total"
          :page-size="pageSize"
          @change="go"
        />
      </div>
    </Panel>

    <Modal v-model="detailOpen" title="运维日志详情" width="max-w-lg">
      <div v-if="detailRow" class="space-y-3 text-sm">
        <div class="grid grid-cols-2 gap-x-4 gap-y-2.5">
          <div><span class="dim">来源：</span>{{ detailRow.operator || '系统' }}</div>
          <div><span class="dim">事件：</span><span class="font-medium">{{ detailRow.actionCN }}</span></div>
          <div>
            <span class="dim">级别：</span>
            <Badge :variant="agentOpLevelMeta[detailRow.level]?.variant ?? 'muted'">
              {{ agentOpLevelMeta[detailRow.level]?.text ?? detailRow.level }}
            </Badge>
          </div>
          <div>
            <span class="dim">结果：</span>
            <Badge :variant="agentOpResultMeta[detailRow.result]?.variant ?? 'muted'">
              {{ agentOpResultMeta[detailRow.result]?.text ?? detailRow.result }}
            </Badge>
          </div>
          <div><span class="dim">时间：</span>{{ detailRow.date }}</div>
        </div>
        <div class="border-t border-border/60 pt-3">
          <div class="dim mb-1">操作对象</div>
          <div>{{ detailRow.target || '—' }}</div>
        </div>
        <div v-if="parseDetail(detailRow.detail).length" class="border-t border-border/60 pt-3">
          <div class="dim mb-1.5">明细</div>
          <div class="space-y-1 bg-muted/40 p-3">
            <div v-for="[k, v] in parseDetail(detailRow.detail)" :key="k" class="flex gap-2">
              <span class="dim shrink-0">{{ k }}：</span><span class="break-all">{{ v }}</span>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="detailOpen = false">关闭</Button>
      </template>
    </Modal>
  </div>
</template>
