<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Search, RotateCcw, Wallet, AlertCircle, QrCode } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination, Modal } from '@/components/ui'
import {
  settleTypeMeta,
  settleStatus,
  statusOptions,
  type SettleRecord,
} from '@/lib/mock/merchant/settle'
import { fetchMerchantSettles } from '@/lib/api/merchantCenter'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { formatMoney } from '@/lib/utils'

const router = useRouter()
const toast = useToast()

// ===== 真接口数据（一次拉当前商户结算记录，客户端筛选/分页）=====
const settleRecords = ref<SettleRecord[]>([])
async function loadSettles() {
  try {
    const res = await fetchMerchantSettles({ page: 1, pageSize: 100 })
    settleRecords.value = res.list
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '结算记录加载失败')
    settleRecords.value = []
  }
}
onMounted(loadSettles)

// ===== 筛选 =====
const filters = ref({ status: -1 })
const filtered = computed(() =>
  settleRecords.value.filter((s) => (filters.value.status > -1 ? s.status === filters.value.status : true)),
)

// ===== 分页 =====
const page = ref(1)
const pageSize = 15
const total = computed(() => filtered.value.length)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const safePage = computed(() => Math.min(page.value, pageCount.value))
const pageRows = computed(() => filtered.value.slice((safePage.value - 1) * pageSize, safePage.value * pageSize))
function go(p: number) {
  page.value = Math.min(Math.max(1, p), pageCount.value)
}
watch(filters, () => { page.value = 1 }, { deep: true })

// ===== 失败原因弹窗 =====
const failOpen = ref(false)
const failRow = ref<SettleRecord | null>(null)
function showFail(s: SettleRecord) {
  failRow.value = s
  failOpen.value = true
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="结算记录" :subtitle="`共 ${total} 条`">
      <template #actions>
        <Button size="sm" @click="router.push('/m/apply')"><Wallet />申请提现</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">结算状态</label>
          <Select v-model="filters.status" :options="statusOptions" class="w-36" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="page = 1"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="filters.status = -1"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="结算下发记录" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[14%]">结算方式</th>
              <th class="w-[22%]">结算账号</th>
              <th class="num w-[14%]">结算金额</th>
              <th class="num w-[14%]">实际到账</th>
              <th class="w-[18%]">结算时间</th>
              <th class="col-center w-[10%]">状态</th>
              <th class="col-center w-[8%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="s in pageRows" :key="s.id">
              <td>
                {{ settleTypeMeta[s.type] }}
                <span v-if="!s.auto" class="ml-1 text-xs text-muted-foreground">[手动]</span>
              </td>
              <td class="truncate font-mono text-[13px]">{{ s.account }}</td>
              <td class="num tabular-nums"><span class="dim text-xs">¥</span>{{ formatMoney(s.money) }}</td>
              <td class="num tabular-nums font-medium"><span class="dim text-xs">¥</span>{{ formatMoney(s.realmoney) }}</td>
              <td class="text-xs">{{ s.addtime }}</td>
              <td class="col-center">
                <Badge :variant="settleStatus[s.status].variant">{{ settleStatus[s.status].text }}</Badge>
              </td>
              <td class="col-center">
                <Button v-if="s.status === 3" variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="showFail(s)">
                  <AlertCircle class="size-4" />
                </Button>
                <Button v-else-if="s.status === 1 && s.type === 2" variant="ghost" size="sm" title="确认收款">
                  <QrCode class="size-4" />
                </Button>
                <span v-else class="dim">—</span>
              </td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="7" class="py-10 text-center dim">暂无结算记录</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="safePage" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>

    <!-- 失败原因弹窗 -->
    <Modal v-model="failOpen" title="结算失败原因" width="max-w-md">
      <div v-if="failRow" class="space-y-2 text-sm">
        <div class="text-muted-foreground">结算金额 <b class="text-foreground">¥{{ formatMoney(failRow.money) }}</b> · {{ settleTypeMeta[failRow.type] }}</div>
        <div class="rounded bg-destructive/[0.08] px-3 py-2.5 text-destructive">{{ failRow.failReason }}</div>
      </div>
      <template #footer>
        <Button size="sm" @click="failOpen = false">知道了</Button>
      </template>
    </Modal>
  </div>
</template>
