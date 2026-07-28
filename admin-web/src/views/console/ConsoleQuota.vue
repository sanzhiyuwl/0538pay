<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Plus } from 'lucide-vue-next'
import { Panel, Button, Badge, Drawer, Pagination, Select } from '@/components/ui'
import {
  fetchAgents,
  fetchQuotaLogs,
  getAgentWallet,
  adjustQuota,
  type Agent,
  type QuotaLog,
  type QuotaWallet,
} from '@/lib/api/console'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const logs = ref<QuotaLog[]>([])
const total = ref(0)
const loading = ref(false)
const agents = ref<Agent[]>([])
const filterAgent = ref('')
const page = ref(1)
const pageSize = 20
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const typeText: Record<string, { label: string; variant: 'success' | 'warning' | 'muted' }> = {
  purchase: { label: '购买', variant: 'success' },
  freeze: { label: '建单冻结', variant: 'warning' },
  consume: { label: '消耗', variant: 'warning' },
  release: { label: '释放', variant: 'success' },
  refund: { label: '退回', variant: 'muted' },
}
const agentName = (id: number) => agents.value.find((a) => a.id === id)?.name ?? `#${id}`

// 收单同款 Select 选项
const agentFilterOptions = computed(() => [
  { value: '', label: '全部代理' },
  ...agents.value.map((a) => ({ value: String(a.id), label: a.name })),
])
const agentPickOptions = computed(() => [
  { value: '', label: '请选择代理' },
  ...agents.value.map((a) => ({ value: String(a.id), label: `${a.name}（${a.account}）` })),
])

async function load() {
  loading.value = true
  try {
    const { list, total: t } = await fetchQuotaLogs({
      agent_id: filterAgent.value ? Number(filterAgent.value) : undefined,
      page: page.value,
      pageSize,
    })
    logs.value = list
    total.value = t
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载名额流水失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    const { list } = await fetchAgents({ pageSize: 500 })
    agents.value = list
  } catch {
    // ignore
  }
  await load()
})

function go(p: number) {
  page.value = p
  load()
}

// ===== 售卖/调整名额 =====
const drawer = ref(false)
const saving = ref(false)
const wallet = ref<QuotaWallet | null>(null)
const form = reactive({ agentId: '', change: 1, amount: '', remark: '' })

function openAdjust() {
  Object.assign(form, { agentId: '', change: 1, amount: '', remark: '' })
  wallet.value = null
  drawer.value = true
}

async function loadWallet() {
  if (!form.agentId) {
    wallet.value = null
    return
  }
  try {
    wallet.value = await getAgentWallet(Number(form.agentId))
  } catch {
    wallet.value = null
  }
}

async function save() {
  if (!form.agentId) {
    toast.error('请选择代理')
    return
  }
  if (!form.change) {
    toast.error('变动数量不能为 0')
    return
  }
  saving.value = true
  try {
    await adjustQuota(Number(form.agentId), {
      change: form.change,
      amount: form.amount || undefined,
      remark: form.remark || (form.change > 0 ? '平台售卖名额' : '平台扣减名额'),
    })
    drawer.value = false
    toast.success('已处理')
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="名额管理" subtitle="代理预购名额（路径一）的售卖与流水">
      <template #actions>
        <Button size="sm" @click="openAdjust"><Plus />售卖 / 调整名额</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">代理</label>
          <Select :model-value="filterAgent" :options="agentFilterOptions" searchable class="w-48" @update:model-value="(v) => { filterAgent = String(v); go(1) }" />
        </div>
      </div>
    </Panel>

    <Panel title="名额流水" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[8%]">ID</th>
              <th class="w-[18%]">代理</th>
              <th class="col-center w-[10%]">类型</th>
              <th class="num w-[10%]">变动</th>
              <th class="num w-[10%]">变动前</th>
              <th class="num w-[10%]">变动后</th>
              <th class="w-[16%]">关联单号</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="l in logs" :key="l.id">
              <td class="tabular-nums">{{ l.id }}</td>
              <td class="truncate">{{ agentName(l.agent_id) }}</td>
              <td class="col-center">
                <Badge :variant="typeText[l.type]?.variant ?? 'muted'">{{ typeText[l.type]?.label ?? l.type }}</Badge>
              </td>
              <td class="num tabular-nums" :class="l.change > 0 ? 'text-success' : 'text-destructive'">
                {{ l.change > 0 ? '+' : '' }}{{ l.change }}
              </td>
              <td class="num tabular-nums dim">{{ l.before }}</td>
              <td class="num tabular-nums">{{ l.after }}</td>
              <td class="truncate dim">{{ l.rel_no || '—' }}</td>
            </tr>
            <tr v-if="loading">
              <td colspan="7" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!logs.length">
              <td colspan="7" class="py-10 text-center dim">暂无名额流水</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="page" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>

    <Drawer v-model="drawer" title="售卖 / 调整名额" subtitle="增发为售卖批发款，扣减用于纠错；变动走流水">
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">代理<span class="text-destructive">*</span></label>
          <Select :model-value="form.agentId" :options="agentPickOptions" searchable class="flex-1" @update:model-value="(v) => { form.agentId = String(v); loadWallet() }" />
        </div>
        <div v-if="wallet" class="bg-muted/40 px-3 py-2 text-sm">
          可用名额：<span class="font-semibold tabular-nums">{{ wallet.balance }}</span>
          <span class="dim">（冻结中 {{ wallet.frozen }} / 累计购买 {{ wallet.total_buy }} / 累计消耗 {{ wallet.total_used }}）</span>
        </div>
        <div class="row-field">
          <label class="lbl">变动数量<span class="text-destructive">*</span></label>
          <input v-model.number="form.change" type="number" class="field-input flex-1" />
        </div>
        <p class="pl-[76px] text-[11px] text-muted-foreground">正数为增发（售卖），负数为扣减（纠错）。</p>
        <div class="row-field">
          <label class="lbl">批发款(元)</label>
          <input v-model="form.amount" placeholder="选填，售卖名额收取的批发款" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">备注</label>
          <input v-model="form.remark" placeholder="选填" class="field-input flex-1" />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="drawer = false">取消</Button>
        <Button :disabled="saving" @click="save">{{ saving ? '处理中…' : '确认' }}</Button>
      </template>
    </Drawer>
  </div>
</template>
