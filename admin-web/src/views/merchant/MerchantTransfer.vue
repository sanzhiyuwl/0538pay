<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Search, RotateCcw, Plus, AlertCircle } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination, Drawer, Modal } from '@/components/ui'
import {
  fetchMerchantTransfers,
  createMerchantTransfer,
  type TransferRecord,
  type TransferCreateReq,
} from '@/lib/api/transfer'
import { ApiError } from '@/lib/api/client'
import { useMerchantAuthStore } from '@/stores/merchantAuth'
import { useToast } from '@/composables/useToast'
import { formatMoney } from '@/lib/utils'

const toast = useToast()
const auth = useMerchantAuthStore()

// 付款方式 / 状态字典（对齐后端 type/status）
const transferApps = [
  { key: 'alipay', label: '支付宝', accountLabel: '支付宝账号' },
  { key: 'wxpay', label: '微信', accountLabel: '微信 OpenId' },
  { key: 'qqpay', label: 'QQ钱包', accountLabel: 'QQ 号码' },
  { key: 'bank', label: '银行卡', accountLabel: '银行卡号' },
]
const typeLabel: Record<string, string> = { alipay: '支付宝', wxpay: '微信', qqpay: 'QQ钱包', bank: '银行卡' }
const transferStatus: Record<number, { text: string; variant: 'warning' | 'success' | 'destructive' }> = {
  0: { text: '处理中', variant: 'warning' },
  1: { text: '转账成功', variant: 'success' },
  2: { text: '转账失败', variant: 'destructive' },
}
const statusOptions = [
  { value: -1, label: '全部状态' },
  { value: 0, label: '处理中' },
  { value: 1, label: '转账成功' },
  { value: 2, label: '转账失败' },
]
const typeOptions = [
  { value: '', label: '全部方式' },
  { value: 'alipay', label: '支付宝' },
  { value: 'wxpay', label: '微信' },
  { value: 'qqpay', label: 'QQ钱包' },
  { value: 'bank', label: '银行卡' },
]

// 代付规则（与后端 transfer.go 常量对齐；余额取登录商户信息）
const setup = { rate: 0.5, minMoney: 1, maxMoney: 20000 }
const balance = computed(() => Number(auth.info?.money ?? 0))

// ===== 筛选 + 分页 =====
const filters = reactive({ value: '', type: '', status: -1 })
const page = ref(1)
const pageSize = 15
const total = ref(0)
const rows = ref<TransferRecord[]>([])
const loading = ref(false)

function buildParams() {
  return {
    page: page.value,
    pageSize,
    keyword: filters.value.trim() || undefined,
    type: filters.type || undefined,
    status: filters.status > -1 ? filters.status : undefined,
  }
}
async function load() {
  loading.value = true
  try {
    const res = await fetchMerchantTransfers(buildParams())
    rows.value = res.list
    total.value = res.total
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载代付记录失败')
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
  filters.value = ''
  filters.type = ''
  filters.status = -1
  applySearch()
}
function go(p: number) {
  page.value = p
  load()
}
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
onMounted(load)

// ===== 失败原因弹窗 =====
const failOpen = ref(false)
const failRow = ref<TransferRecord | null>(null)
function showFail(t: TransferRecord) {
  failRow.value = t
  failOpen.value = true
}

// ===== 发起代付抽屉 =====
const addOpen = ref(false)
const app = ref('alipay')
const form = reactive({ biz_no: '', account: '', username: '', money: '', desc: '', paypwd: '' })
const busy = ref(false)
const currentApp = computed(() => transferApps.find((a) => a.key === app.value)!)
const fee = computed(() => {
  const m = Number(form.money) || 0
  return m > 0 ? Math.round(m * (setup.rate / 100) * 100) / 100 : 0
})
const need = computed(() => (Number(form.money) || 0) + fee.value)
const canSubmit = computed(() => {
  const m = Number(form.money)
  return !!form.biz_no && !!form.account && m >= setup.minMoney && m <= setup.maxMoney && !!form.paypwd
})

function genBizNo() {
  const now = new Date()
  const p = (n: number, l = 2) => String(n).padStart(l, '0')
  const ymdhis =
    now.getFullYear() + p(now.getMonth() + 1) + p(now.getDate()) +
    p(now.getHours()) + p(now.getMinutes()) + p(now.getSeconds())
  return (ymdhis + String(Math.floor(11111 + Math.random() * 88888))).slice(0, 19)
}
function openAdd() {
  form.biz_no = genBizNo()
  form.account = ''
  form.username = ''
  form.money = ''
  form.desc = ''
  form.paypwd = ''
  app.value = 'alipay'
  addOpen.value = true
}
async function submitAdd() {
  if (!canSubmit.value || busy.value) return
  busy.value = true
  try {
    const body: TransferCreateReq = {
      biz_no: form.biz_no,
      type: app.value,
      account: form.account.trim(),
      username: form.username.trim(),
      money: String(form.money),
      desc: form.desc.trim(),
      password: form.paypwd,
    }
    const res = await createMerchantTransfer(body)
    toast.success(`已提交代付，交易号 ${res.biz_no}`)
    addOpen.value = false
    await auth.refreshInfo() // 余额已扣，刷新
    applySearch()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '发起代付失败')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="代付管理" :subtitle="`共 ${total} 笔代付`">
      <template #actions>
        <Button size="sm" @click="openAdd"><Plus />发起代付</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">收款方</label>
          <input v-model="filters.value" placeholder="交易号 / 收款账号 / 姓名" class="field-input w-52" @keyup.enter="applySearch" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">付款方式</label>
          <Select v-model="filters.type" :options="typeOptions" class="w-32" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">状态</label>
          <Select v-model="filters.status" :options="statusOptions" class="w-32" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="applySearch"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="代付记录" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[18%]">交易号 / 三方号</th>
              <th class="w-[13%]">方式 / 备注</th>
              <th class="w-[18%]">收款账号 / 姓名</th>
              <th class="num w-[14%]">金额 / 花费</th>
              <th class="w-[16%]">提交 / 付款时间</th>
              <th class="col-center w-[10%]">状态</th>
              <th class="col-center w-[9%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="t in rows" :key="t.biz_no">
              <td>
                <div class="truncate font-mono text-[13px] text-primary">{{ t.biz_no }}</div>
                <div class="truncate text-xs dim">{{ t.pay_order_no || '—' }}</div>
              </td>
              <td>
                <div>{{ typeLabel[t.type] }}</div>
                <div class="truncate text-xs dim">{{ t.desc || '—' }}</div>
              </td>
              <td>
                <div class="truncate font-mono text-[13px]">{{ t.account }}</div>
                <div class="truncate text-xs dim">{{ t.username || '—' }}</div>
              </td>
              <td class="num">
                <div class="tabular-nums font-medium"><span class="dim text-xs">¥</span>{{ formatMoney(t.money) }}</div>
                <div class="text-xs dim tabular-nums">花 ¥{{ formatMoney(t.costmoney) }}</div>
              </td>
              <td>
                <div class="text-xs">{{ t.addtime }}</div>
                <div class="text-xs dim">{{ t.paytime ?? '—' }}</div>
              </td>
              <td class="col-center"><Badge :variant="transferStatus[t.status].variant">{{ transferStatus[t.status].text }}</Badge></td>
              <td class="col-center">
                <Button v-if="t.status === 2 && t.result" variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="showFail(t)">
                  <AlertCircle class="size-4" />
                </Button>
                <span v-else class="dim text-xs">—</span>
              </td>
            </tr>
            <tr v-if="loading">
              <td colspan="7" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!rows.length">
              <td colspan="7" class="py-10 text-center dim">暂无代付记录</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="page" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>

    <!-- 失败原因弹窗 -->
    <Modal v-model="failOpen" title="转账失败原因" width="max-w-md">
      <div v-if="failRow" class="space-y-2 text-sm">
        <div class="text-muted-foreground">{{ typeLabel[failRow.type] }} · <b class="text-foreground">¥{{ formatMoney(failRow.money) }}</b> → {{ failRow.username || failRow.account }}</div>
        <div class="rounded bg-destructive/[0.08] px-3 py-2.5 text-destructive">{{ failRow.result }}</div>
      </div>
      <template #footer>
        <Button size="sm" @click="failOpen = false">知道了</Button>
      </template>
    </Modal>

    <!-- 发起代付抽屉 -->
    <Drawer v-model="addOpen" title="发起代付" subtitle="向支付宝/微信/QQ钱包/银行卡单笔转账" width="max-w-md">
      <div class="space-y-3.5">
        <!-- 付款方式 Tab -->
        <div class="flex gap-1 border-b border-border">
          <button
            v-for="a in transferApps"
            :key="a.key"
            class="-mb-px border-b-2 px-3 py-2 text-sm transition-colors"
            :class="app === a.key ? 'border-primary font-medium text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'"
            @click="app = a.key"
          >
            {{ a.label }}
          </button>
        </div>

        <div class="row-field">
          <label class="lbl">交易号</label>
          <input v-model="form.biz_no" class="field-input flex-1 font-mono text-[13px]" placeholder="19 位数字，防重复" />
        </div>
        <div class="row-field">
          <label class="lbl">{{ currentApp.accountLabel }}</label>
          <input v-model="form.account" class="field-input flex-1" :placeholder="`收款方${currentApp.accountLabel}`" />
        </div>
        <div class="row-field">
          <label class="lbl">收款姓名</label>
          <input v-model="form.username" class="field-input flex-1" placeholder="选填，校验更安全" />
        </div>
        <div class="row-field">
          <label class="lbl">转账金额</label>
          <input v-model="form.money" class="field-input flex-1" :placeholder="`单笔 ¥${setup.minMoney}~¥${setup.maxMoney}`" />
        </div>
        <div class="row-field">
          <label class="lbl">备注</label>
          <input v-model="form.desc" maxlength="32" class="field-input flex-1" placeholder="选填，≤32 字" />
        </div>
        <div class="row-field">
          <label class="lbl">登录密码</label>
          <input v-model="form.paypwd" type="password" class="field-input flex-1" placeholder="验证身份" />
        </div>

        <div class="space-y-1 border-t border-border/60 pt-3 text-sm">
          <div class="flex justify-between"><span class="text-muted-foreground">可转账余额</span><span class="tabular-nums">¥{{ formatMoney(balance) }}</span></div>
          <div class="flex justify-between"><span class="text-muted-foreground">手续费（{{ setup.rate }}%）</span><span class="tabular-nums text-destructive">¥{{ formatMoney(fee) }}</span></div>
          <div class="flex justify-between font-medium"><span>需支付</span><span class="tabular-nums text-primary">¥{{ formatMoney(need) }}</span></div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="addOpen = false">取消</Button>
        <Button size="sm" :disabled="!canSubmit || busy" @click="submitAdd">立即转账</Button>
      </template>
    </Drawer>
  </div>
</template>
