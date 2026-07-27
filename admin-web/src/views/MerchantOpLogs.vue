<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Search, RotateCcw, Download, ExternalLink, Eye } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, DateRange, Pagination, Modal } from '@/components/ui'
import {
  fetchMerchantOpLogs,
  fetchOpActionOptions,
  exportMerchantOpLogs,
  opLevelMeta,
  opCategoryText,
  opResultMeta,
  opCategoryOptions,
  opLevelOptions,
  opResultOptions,
  type OpLog,
  type OpActionOption,
} from '@/lib/api/oplogs'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

// ===== 筛选 =====
const filters = reactive({
  uid: '',
  action: '',
  category: '',
  level: '',
  result: '',
  keyword: '',
  starttime: '',
  endtime: '',
})

// 动作下拉（后端映射表派生）
const actionOptions = ref<{ value: string; label: string }[]>([{ value: '', label: '全部动作' }])
async function loadActions() {
  try {
    const res = await fetchOpActionOptions()
    actionOptions.value = [
      { value: '', label: '全部动作' },
      ...res.actions.map((a: OpActionOption) => ({ value: a.value, label: a.label })),
    ]
  } catch {
    /* 动作选项拉取失败不打扰，保留默认 */
  }
}

// ===== 分页 + 数据 =====
const page = ref(1)
const pageSize = 15
const total = ref(0)
const rows = ref<OpLog[]>([])
const loading = ref(false)

function buildParams() {
  const uidNum = Number(filters.uid.trim())
  return {
    page: page.value,
    pageSize,
    uid: filters.uid.trim() && !Number.isNaN(uidNum) ? uidNum : undefined,
    action: filters.action || undefined,
    category: filters.category || undefined,
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
    const res = await fetchMerchantOpLogs(buildParams())
    rows.value = res.list
    total.value = res.total
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载操作日志失败')
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
  filters.uid = ''
  filters.action = ''
  filters.category = ''
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

const route = useRoute()
onMounted(() => {
  // 从商户页快捷跳转而来：预置商户号筛选
  const qu = route.query.uid
  if (qu != null && String(qu).trim()) filters.uid = String(qu).trim()
  loadActions()
  load()
})

// ===== 导出（按当前筛选条件，后端流式 CSV）=====
const exporting = ref(false)
async function exportList() {
  if (exporting.value) return
  exporting.value = true
  try {
    await exportMerchantOpLogs({ ...buildParams(), page: undefined, pageSize: undefined })
    toast.success('已开始下载操作日志 CSV')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '导出失败')
  } finally {
    exporting.value = false
  }
}

// ===== 详情弹窗 =====
const detailOpen = ref(false)
const detailRow = ref<OpLog | null>(null)
function openDetail(row: OpLog) {
  detailRow.value = row
  detailOpen.value = true
}

// detail JSON 解析成键值对（供详情展示）
function parseDetail(detail: string): [string, string][] {
  if (!detail || !detail.trim()) return []
  try {
    const obj = JSON.parse(detail)
    if (obj && typeof obj === 'object') {
      return Object.entries(obj).map(([k, v]) => [k, String(v)])
    }
  } catch {
    /* 非 JSON 原样忽略 */
  }
  return []
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="商户操作日志" subtitle="记录商户在商户中心的写操作（改资料 / 提现 / 退款 / 绑域名 / 改密钥等），供审计追溯">
      <div class="filter-bar">
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">商户号</label>
          <input v-model="filters.uid" placeholder="商户号" class="field-input w-32" @keyup.enter="search" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">操作动作</label>
          <Select v-model="filters.action" :options="actionOptions" class="w-36" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">分类</label>
          <Select v-model="filters.category" :options="opCategoryOptions" class="w-28" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">级别</label>
          <Select v-model="filters.level" :options="opLevelOptions" class="w-28" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">结果</label>
          <Select v-model="filters.result" :options="opResultOptions" class="w-28" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">关键字</label>
          <input v-model="filters.keyword" placeholder="操作者 / 对象" class="field-input w-40" @keyup.enter="search" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">时间</label>
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

    <!-- 列表 -->
    <Panel title="操作日志列表" :subtitle="`共 ${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full">
          <thead>
            <tr>
              <th class="w-[6%]">ID</th>
              <th class="w-[10%]">商户号</th>
              <th class="w-[12%]">操作者</th>
              <th class="w-[12%]">操作</th>
              <th class="w-[7%]">分类</th>
              <th class="w-[7%]">级别</th>
              <th class="w-[20%]">对象</th>
              <th class="w-[7%]">结果</th>
              <th class="w-[12%]">IP</th>
              <th class="w-[14%]">时间</th>
              <th class="w-[6%] col-center">详情</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="l in rows" :key="l.id">
              <td class="tabular-nums dim">{{ l.id }}</td>
              <td class="tabular-nums">{{ l.uid }}</td>
              <td class="truncate" :title="l.operator">{{ l.operator || '—' }}</td>
              <td class="font-medium">{{ l.actionCN }}</td>
              <td class="dim">{{ opCategoryText[l.category] ?? l.category }}</td>
              <td>
                <Badge :variant="opLevelMeta[l.level]?.variant ?? 'muted'">
                  {{ opLevelMeta[l.level]?.text ?? l.level }}
                </Badge>
              </td>
              <td class="truncate" :title="l.target">{{ l.target }}</td>
              <td>
                <Badge :variant="opResultMeta[l.result]?.variant ?? 'muted'">
                  {{ opResultMeta[l.result]?.text ?? l.result }}
                </Badge>
              </td>
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
              <td class="col-center">
                <button
                  class="inline-flex size-7 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                  title="查看详情"
                  @click="openDetail(l)"
                >
                  <Eye class="size-4" />
                </button>
              </td>
            </tr>
            <tr v-if="!loading && !rows.length">
              <td colspan="11" class="py-10 text-center dim">没有符合条件的操作日志</td>
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

    <!-- 详情弹窗 -->
    <Modal v-model="detailOpen" title="操作日志详情" width="max-w-lg">
      <div v-if="detailRow" class="space-y-3 text-sm">
        <div class="grid grid-cols-2 gap-x-4 gap-y-2.5">
          <div><span class="dim">商户号：</span><span class="tabular-nums">{{ detailRow.uid }}</span></div>
          <div><span class="dim">操作者：</span>{{ detailRow.operator || '—' }}</div>
          <div><span class="dim">操作：</span><span class="font-medium">{{ detailRow.actionCN }}</span></div>
          <div>
            <span class="dim">级别：</span>
            <Badge :variant="opLevelMeta[detailRow.level]?.variant ?? 'muted'">
              {{ opLevelMeta[detailRow.level]?.text ?? detailRow.level }}
            </Badge>
          </div>
          <div><span class="dim">分类：</span>{{ opCategoryText[detailRow.category] ?? detailRow.category }}</div>
          <div>
            <span class="dim">结果：</span>
            <Badge :variant="opResultMeta[detailRow.result]?.variant ?? 'muted'">
              {{ opResultMeta[detailRow.result]?.text ?? detailRow.result }}
            </Badge>
          </div>
          <div><span class="dim">IP：</span><span class="tabular-nums">{{ detailRow.ip }}</span></div>
          <div><span class="dim">时间：</span>{{ detailRow.date }}</div>
        </div>
        <div class="border-t border-border/60 pt-3">
          <div class="dim mb-1">操作对象</div>
          <div>{{ detailRow.target || '—' }}</div>
        </div>
        <div v-if="parseDetail(detailRow.detail).length" class="border-t border-border/60 pt-3">
          <div class="dim mb-1.5">明细</div>
          <div class="space-y-1 rounded-lg bg-muted/40 p-3">
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
