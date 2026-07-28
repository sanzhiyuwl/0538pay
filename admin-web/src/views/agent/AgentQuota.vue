<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Panel, Badge, Pagination } from '@/components/ui'
import { fetchMyWallet, fetchMyQuotaLogs } from '@/lib/api/agent'
import type { QuotaLog, QuotaWallet } from '@/lib/api/console'
import { ApiError } from '@/lib/api/client'

const wallet = ref<QuotaWallet | null>(null)
const logs = ref<QuotaLog[]>([])
const total = ref(0)
const loading = ref(false)
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

async function load() {
  loading.value = true
  try {
    const { list, total: t } = await fetchMyQuotaLogs({ page: page.value, pageSize })
    logs.value = list
    total.value = t
  } catch (e) {
    alert(e instanceof ApiError ? e.message : '加载名额流水失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    wallet.value = await fetchMyWallet()
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
    <Panel title="名额钱包" subtitle="预购名额（路径一）建单即冻结预占，进件成功转消耗、失败释放回可用；名额售卖由平台操作">
      <div class="grid grid-cols-4 gap-2.5">
        <div class="bg-muted/40 px-4 py-3">
          <div class="text-[11px] text-muted-foreground">可用余额</div>
          <div class="mt-1 text-2xl font-semibold tabular-nums">{{ wallet?.balance ?? '—' }}</div>
        </div>
        <div class="bg-muted/40 px-4 py-3">
          <div class="text-[11px] text-muted-foreground">冻结中</div>
          <div class="mt-1 text-2xl font-semibold tabular-nums">{{ wallet?.frozen ?? '—' }}</div>
        </div>
        <div class="bg-muted/40 px-4 py-3">
          <div class="text-[11px] text-muted-foreground">累计购买</div>
          <div class="mt-1 text-2xl font-semibold tabular-nums">{{ wallet?.total_buy ?? '—' }}</div>
        </div>
        <div class="bg-muted/40 px-4 py-3">
          <div class="text-[11px] text-muted-foreground">累计消耗</div>
          <div class="mt-1 text-2xl font-semibold tabular-nums">{{ wallet?.total_used ?? '—' }}</div>
        </div>
      </div>
    </Panel>

    <Panel title="名额流水" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[10%]">ID</th>
              <th class="col-center w-[14%]">类型</th>
              <th class="num w-[14%]">变动</th>
              <th class="num w-[14%]">变动前</th>
              <th class="num w-[14%]">变动后</th>
              <th class="w-[24%]">关联单号</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="l in logs" :key="l.id">
              <td class="tabular-nums">{{ l.id }}</td>
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
              <td colspan="6" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!logs.length">
              <td colspan="6" class="py-10 text-center dim">暂无名额流水</td>
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
