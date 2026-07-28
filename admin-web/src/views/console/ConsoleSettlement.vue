<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Panel, Badge, Pagination, Select } from '@/components/ui'
import { fetchAgents, fetchSettlements, type Agent, type Settlement } from '@/lib/api/console'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const rows = ref<Settlement[]>([])
const total = ref(0)
const loading = ref(false)
const agents = ref<Agent[]>([])
const filterAgent = ref('')
const page = ref(1)
const pageSize = 20
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const pathText: Record<number, string> = { 1: '预购名额（全额归代理）', 2: '商户自付（分账）' }
const agentName = (id: number) => agents.value.find((a) => a.id === id)?.name ?? `#${id}`

// 代理筛选下拉（收单同款 Select）
const agentFilterOptions = computed(() => [
  { value: '', label: '全部代理' },
  ...agents.value.map((a) => ({ value: String(a.id), label: a.name })),
])

async function load() {
  loading.value = true
  try {
    const { list, total: t } = await fetchSettlements({
      agent_id: filterAgent.value ? Number(filterAgent.value) : undefined,
      page: page.value,
      pageSize,
    })
    rows.value = list
    total.value = t
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载佣金结算失败')
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
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="佣金结算" subtitle="进件成功后的佣金结算流水（进件成功触发）">
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">代理</label>
          <Select :model-value="filterAgent" :options="agentFilterOptions" searchable class="w-48" @update:model-value="(v) => { filterAgent = String(v); go(1) }" />
        </div>
      </div>
    </Panel>

    <Panel title="结算流水" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[10%]">ID</th>
              <th class="w-[12%]">进件单</th>
              <th class="w-[20%]">代理</th>
              <th class="w-[28%]">资金路径</th>
              <th class="w-[20%]">关联收款单号</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in rows" :key="r.id">
              <td class="tabular-nums">{{ r.id }}</td>
              <td class="tabular-nums dim">#{{ r.enroll_id }}</td>
              <td class="truncate">{{ agentName(r.agent_id) }}</td>
              <td><Badge variant="muted">{{ pathText[r.path] ?? '—' }}</Badge></td>
              <td class="truncate tabular-nums dim">{{ r.pay_order_no || '—' }}</td>
            </tr>
            <tr v-if="loading">
              <td colspan="5" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!rows.length">
              <td colspan="5" class="py-10 text-center dim">暂无结算流水（进件成功后自动生成）</td>
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
