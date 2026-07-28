<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Panel, Badge, Pagination } from '@/components/ui'
import { fetchMySettlements } from '@/lib/api/agent'
import type { Settlement } from '@/lib/api/console'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const rows = ref<Settlement[]>([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const pageSize = 20
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const pathText: Record<number, string> = { 1: '预购名额（全额归我）', 2: '商户自付（分账）' }

async function load() {
  loading.value = true
  try {
    const { list, total: t } = await fetchMySettlements({ page: page.value, pageSize })
    rows.value = list
    total.value = t
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载佣金结算失败')
  } finally {
    loading.value = false
  }
}

onMounted(load)

function go(p: number) {
  page.value = p
  load()
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="佣金结算" subtitle="你名下进件成功后的佣金结算流水（进件成功自动触发）">
      <template #actions>
        <span />
      </template>
    </Panel>

    <Panel title="结算流水" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[14%]">ID</th>
              <th class="w-[18%]">进件单</th>
              <th class="w-[36%]">资金路径</th>
              <th class="w-[28%]">关联收款单号</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in rows" :key="r.id">
              <td class="tabular-nums">{{ r.id }}</td>
              <td class="tabular-nums dim">#{{ r.enroll_id }}</td>
              <td><Badge variant="muted">{{ pathText[r.path] ?? '—' }}</Badge></td>
              <td class="truncate tabular-nums dim">{{ r.pay_order_no || '—' }}</td>
            </tr>
            <tr v-if="loading">
              <td colspan="4" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!rows.length">
              <td colspan="4" class="py-10 text-center dim">暂无结算流水（进件成功后自动生成）</td>
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
