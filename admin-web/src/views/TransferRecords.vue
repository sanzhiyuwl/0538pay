<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { RouterLink } from 'vue-router'
import {
  Search,
  RotateCcw,
  Plus,
  MoreHorizontal,
  CheckCircle2,
  XCircle,
  RefreshCw,
  Undo2,
  Copy,
  FileText,
  Trash2,
} from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination } from '@/components/ui'
import {
  transferRecords,
  transferTypeText,
  transferStatus,
  transferActions,
  calcTransferStats,
} from '@/lib/mock/transfer'
import { formatMoney } from '@/lib/utils'

const typeOptions = [
  { value: '', label: '所有付款方式' },
  { value: 'alipay', label: '支付宝' },
  { value: 'wxpay', label: '微信' },
  { value: 'qqpay', label: 'QQ钱包' },
  { value: 'bank', label: '银行卡' },
]
const statusOptions = [
  { value: -1, label: '全部状态' },
  { value: 0, label: '正在处理' },
  { value: 1, label: '转账成功' },
  { value: 2, label: '转账失败' },
]

// ===== 筛选 =====
const filters = ref({ value: '', uid: '', type: '', dstatus: -1 })

const filtered = computed(() => {
  return transferRecords.filter((r) => {
    if (filters.value.uid && String(r.uid) !== filters.value.uid.trim()) return false
    if (filters.value.type && r.type !== filters.value.type) return false
    if (filters.value.dstatus > -1 && r.status !== filters.value.dstatus) return false
    if (filters.value.value.trim()) {
      const v = filters.value.value.trim()
      if (!`${r.biz_no}${r.account}${r.username}`.includes(v)) return false
    }
    return true
  })
})

function resetFilters() {
  filters.value = { value: '', uid: '', type: '', dstatus: -1 }
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
const stats = computed(() => calcTransferStats(filtered.value))

// ===== 多选 =====
const selected = ref<Set<string>>(new Set())
const pageAllChecked = computed(
  () => pageRows.value.length > 0 && pageRows.value.every((r) => selected.value.has(r.biz_no)),
)
function toggleAll() {
  if (pageAllChecked.value) pageRows.value.forEach((r) => selected.value.delete(r.biz_no))
  else pageRows.value.forEach((r) => selected.value.add(r.biz_no))
}
function toggleOne(biz: string) {
  if (selected.value.has(biz)) selected.value.delete(biz)
  else selected.value.add(biz)
}

// 筛选即时变化时回到第1页并清空选中，避免批量操作命中不可见行
watch(filters, applySearch, { deep: true })

// ===== 行操作菜单 =====
const openMenu = ref<string | null>(null)
function toggleMenu(biz: string) {
  openMenu.value = openMenu.value === biz ? null : biz
}
function closeMenu() {
  openMenu.value = null
}
onMounted(() => window.addEventListener('click', closeMenu))
onUnmounted(() => window.removeEventListener('click', closeMenu))

const actionIcons: Record<string, any> = {
  改为成功: CheckCircle2,
  改为失败: XCircle,
  查询状态: RefreshCw,
  退回: Undo2,
  复制: Copy,
  获取凭证: FileText,
  删除: Trash2,
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="付款记录" :subtitle="`共 ${total} 笔付款`">
      <template #actions>
        <RouterLink to="/transfer">
          <Button size="sm"><Plus />新增付款</Button>
        </RouterLink>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">付款搜索</label>
          <input v-model="filters.value" placeholder="交易号 / 收款账号 / 姓名" class="field-input w-52" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">商户号</label>
          <input v-model="filters.uid" placeholder="请输入商户号" class="field-input w-32" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">付款方式</label>
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

    <!-- 概况 -->
    <Panel title="付款概况" subtitle="按当前筛选条件">
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">付款总额</div>
          <div class="mt-1 text-xl font-normal tabular-nums"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.totalMoney) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">成功金额</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.successMoney) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">成功笔数</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ stats.successCount }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">处理中 / 失败</div>
          <div class="mt-1 text-xl font-normal tabular-nums">
            <span class="text-warning">{{ stats.processingCount }}</span>
            <span class="text-sm text-muted-foreground">/</span>
            <span class="text-destructive">{{ stats.failCount }}</span>
          </div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="付款记录列表" :subtitle="selected.size ? `已选 ${selected.size} 笔` : `${total} 条`">
      <template v-if="selected.size" #actions>
        <span class="text-sm text-muted-foreground">批量：</span>
        <Button variant="outline" size="sm" @click="selected.clear()"><CheckCircle2 />改为成功</Button>
        <Button variant="outline" size="sm" @click="selected.clear()"><XCircle />改为失败</Button>
        <Button variant="outline" size="sm" @click="selected.clear()"><Trash2 />删除</Button>
      </template>
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="col-center w-[4%]">
                <input type="checkbox" :checked="pageAllChecked" @change="toggleAll" />
              </th>
              <th class="w-[16%]">交易号 / 第三方号</th>
              <th class="w-[9%]">商户号</th>
              <th class="w-[14%]">付款方式 / 备注</th>
              <th class="w-[15%]">收款账号 / 姓名</th>
              <th class="w-[12%]">付款 / 花费</th>
              <th class="w-[14%]">提交 / 付款时间</th>
              <th class="col-center w-[8%]">状态</th>
              <th class="col-center w-[8%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(r, si) in pageRows" :key="r.biz_no">
              <td class="col-center">
                <input type="checkbox" :checked="selected.has(r.biz_no)" @change="toggleOne(r.biz_no)" />
              </td>
              <td>
                <div class="truncate font-medium tabular-nums">{{ r.biz_no }}</div>
                <div class="truncate text-xs dim">{{ r.pay_order_no || '—' }}</div>
              </td>
              <td>
                <span v-if="r.uid > 0" class="tabular-nums text-primary">{{ r.uid }}</span>
                <span v-else class="dim">管理员</span>
              </td>
              <td>
                <div>{{ transferTypeText[r.type] }} <span class="dim">({{ r.channel }})</span></div>
                <div v-if="r.desc" class="truncate text-xs dim">{{ r.desc }}</div>
              </td>
              <td>
                <div class="truncate tabular-nums">{{ r.account }}</div>
                <div class="text-xs dim">{{ r.username }}</div>
              </td>
              <td>
                <div class="tabular-nums"><span class="dim text-xs">¥</span><b>{{ r.money }}</b></div>
                <div class="text-xs dim tabular-nums">花费 ¥{{ r.costmoney }}</div>
              </td>
              <td>
                <div class="text-xs">{{ r.addtime }}</div>
                <div class="text-xs dim">{{ r.paytime ?? '—' }}</div>
              </td>
              <td class="col-center">
                <Badge :variant="transferStatus[r.status].variant">{{ transferStatus[r.status].text }}</Badge>
                <div v-if="r.status === 2" class="mt-1 truncate text-xs text-destructive" :title="r.result">
                  {{ r.result }}
                </div>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(r.biz_no)">
                    <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === r.biz_no"
                    class="menu-panel absolute right-0 z-20 w-32"
                    :class="si >= pageRows.length - 3 && pageRows.length > 3
                      ? 'bottom-full mb-1.5'
                      : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <template v-for="(a, ai) in transferActions(r.status, r.uid)" :key="ai">
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
              <td colspan="9" class="py-10 text-center dim">没有符合条件的付款记录</td>
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
