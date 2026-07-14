<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Search, RotateCcw, Plus, AlertCircle, FileText, RefreshCw, QrCode } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination, Drawer, Modal } from '@/components/ui'
import {
  transferRecords,
  transferApps,
  transferStatus,
  statusOptions,
  typeOptions,
  transferSetup as setup,
  type TransferRecord,
} from '@/lib/mock/merchant/transfer'
import { formatMoney } from '@/lib/utils'

// ===== 筛选 =====
const filters = ref({ value: '', type: 'all', status: -1 })
const filtered = computed(() =>
  transferRecords.filter((t) => {
    if (filters.value.type !== 'all' && t.type !== filters.value.type) return false
    if (filters.value.status > -1 && t.status !== filters.value.status) return false
    if (filters.value.value.trim()) {
      const v = filters.value.value.trim()
      if (!t.account.includes(v) && !t.username.includes(v)) return false
    }
    return true
  }),
)
function resetFilters() {
  filters.value = { value: '', type: 'all', status: -1 }
}

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
const failRow = ref<TransferRecord | null>(null)
function showFail(t: TransferRecord) {
  failRow.value = t
  failOpen.value = true
}

// ===== 发起代付抽屉 =====
const addOpen = ref(false)
const app = ref('alipay')
const form = ref({ out_biz_no: '', account: '', username: '', money: '', desc: '', paypwd: '' })
const currentApp = computed(() => transferApps.find((a) => a.key === app.value)!)
const fee = computed(() => {
  const m = Number(form.value.money) || 0
  return m > 0 ? Math.round(m * (setup.rate / 100) * 100) / 100 : 0
})
const need = computed(() => (Number(form.value.money) || 0) + fee.value)
const canSubmit = computed(() => {
  const m = Number(form.value.money)
  return form.value.out_biz_no && form.value.account && m >= setup.minMoney && m <= setup.maxMoney && form.value.paypwd
})
function openAdd() {
  // 自动生成 19 位交易号
  const now = '2026071200000'
  form.value = { out_biz_no: `${now}${String(1000 + Math.floor(total.value)).slice(0, 6)}`.slice(0, 19), account: '', username: '', money: '', desc: '', paypwd: '' }
  app.value = 'alipay'
  addOpen.value = true
}
function submitAdd() {
  if (!canSubmit.value) return
  addOpen.value = false
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
          <input v-model="filters.value" placeholder="收款账号 / 姓名" class="field-input w-48" />
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
          <Button size="sm" @click="page = 1"><Search />搜索</Button>
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
              <th class="w-[16%]">交易号 / 三方号</th>
              <th class="w-[13%]">方式 / 备注</th>
              <th class="w-[17%]">收款账号 / 姓名</th>
              <th class="num w-[13%]">金额 / 花费</th>
              <th class="w-[16%]">提交 / 付款时间</th>
              <th class="col-center w-[10%]">状态</th>
              <th class="col-center w-[9%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="t in pageRows" :key="t.id">
              <td>
                <div class="truncate font-mono text-[13px] text-primary">{{ t.biz_no }}</div>
                <div class="truncate text-xs dim">{{ t.pay_order_no || '—' }}</div>
              </td>
              <td>
                <div>{{ t.typeLabel }}</div>
                <div class="truncate text-xs dim">{{ t.desc || '—' }}</div>
              </td>
              <td>
                <div class="truncate font-mono text-[13px]">{{ t.account }}</div>
                <div class="truncate text-xs dim">{{ t.username }}</div>
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
                <Button v-if="t.status === 2" variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="showFail(t)">
                  <AlertCircle class="size-4" />
                </Button>
                <Button v-else-if="t.status === 0" variant="ghost" size="sm" title="查询状态"><RefreshCw class="size-4" /></Button>
                <Button v-else-if="t.status === 1 && t.type === 'wxpay'" variant="ghost" size="sm" title="确认收款"><QrCode class="size-4" /></Button>
                <Button v-else variant="ghost" size="sm" title="转账凭证"><FileText class="size-4" /></Button>
              </td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="7" class="py-10 text-center dim">暂无代付记录</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="safePage" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>

    <!-- 失败原因弹窗 -->
    <Modal v-model="failOpen" title="转账失败原因" width="max-w-md">
      <div v-if="failRow" class="space-y-2 text-sm">
        <div class="text-muted-foreground">{{ failRow.typeLabel }} · <b class="text-foreground">¥{{ formatMoney(failRow.money) }}</b> → {{ failRow.username }}</div>
        <div class="rounded bg-destructive/[0.08] px-3 py-2.5 text-destructive">{{ failRow.failReason }}</div>
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
          <input v-model="form.out_biz_no" class="field-input flex-1 font-mono text-[13px]" placeholder="19 位数字，防重复" />
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
          <div class="flex justify-between"><span class="text-muted-foreground">可转账余额</span><span class="tabular-nums">¥{{ formatMoney(setup.balance) }}</span></div>
          <div class="flex justify-between"><span class="text-muted-foreground">手续费（{{ setup.rate }}%）</span><span class="tabular-nums text-destructive">¥{{ formatMoney(fee) }}</span></div>
          <div class="flex justify-between font-medium"><span>需支付</span><span class="tabular-nums text-primary">¥{{ formatMoney(need) }}</span></div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="addOpen = false">取消</Button>
        <Button size="sm" :disabled="!canSubmit" @click="submitAdd">立即转账</Button>
      </template>
    </Drawer>
  </div>
</template>
