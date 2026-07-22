<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { RefreshCw } from 'lucide-vue-next'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import TrendChart from '@/components/TrendChart.vue'
import { fetchAdminDashboard, type AdminDashboard } from '@/lib/api/dashboard'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const data = ref<AdminDashboard | null>(null)
const loading = ref(false)
const updatedAt = ref('')

async function load() {
  loading.value = true
  try {
    data.value = await fetchAdminDashboard()
    updatedAt.value = new Date().toLocaleString('zh-CN', { hour12: false })
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载仪表盘失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

const overview = computed(() => data.value?.overview ?? [])
const trendLabels = computed(() => data.value?.trend.labels ?? [])
const orderTrend = computed(() => [
  { name: '订单量', color: '#4b7bec', data: data.value?.trend.orders ?? [] },
])
const amountTrend = computed(() => [
  { name: '交易额', color: '#07c160', data: (data.value?.trend.amounts ?? []).map((a) => Number(a)) },
])

const todoCards = computed(() => {
  const t = data.value?.todo
  if (!t) return []
  return [
    { label: '待结算', value: t.pending_settle, urgent: false, to: '/admin/settle' },
    { label: '待审域名', value: t.pending_domain, urgent: true, to: '/admin/domains' },
    { label: '待分账', value: t.pending_profit, urgent: false, to: '/admin/profit-sharing' },
    { label: '今日未付单', value: t.unpaid_orders, urgent: false, to: '/admin/orders' },
  ]
})

const statusMap: Record<number, { text: string; variant: 'success' | 'warning' | 'destructive' | 'muted' }> = {
  0: { text: '待支付', variant: 'warning' },
  1: { text: '已支付', variant: 'success' },
  2: { text: '已退款', variant: 'destructive' },
  3: { text: '已冻结', variant: 'muted' },
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- ===== 实时概况 ===== -->
    <Card>
      <div class="flex items-center gap-2 px-6 py-4">
        <h3 class="text-[15px] font-semibold tracking-tight">实时概况</h3>
        <span class="text-xs text-muted-foreground/70">更新时间:{{ updatedAt || '—' }}</span>
        <div class="flex-1" />
        <button class="flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-primary" @click="load">
          <RefreshCw class="size-3.5" :class="loading && 'animate-spin'" /> 刷新
        </button>
      </div>
      <div class="border-t border-border/70" />
      <div class="px-6 py-6">
        <div class="grid grid-cols-2 gap-x-8 gap-y-7 lg:grid-cols-4">
          <div v-for="s in overview" :key="s.label">
            <div class="text-[13px] text-muted-foreground">{{ s.label }}</div>
            <div class="mt-2.5 text-[28px] font-normal leading-none tabular-nums text-foreground">{{ s.today }}</div>
            <div class="mt-2 text-xs text-muted-foreground/60">昨日:{{ s.yesterday }}</div>
            <div class="mt-5">
              <div class="text-[13px] text-muted-foreground">{{ s.total_label }}</div>
              <div class="mt-2 text-[22px] font-normal leading-none tabular-nums text-foreground">{{ s.total }}</div>
            </div>
          </div>
        </div>
      </div>
    </Card>

    <!-- ===== 关键指标 ===== -->
    <Card>
      <div class="px-6 py-4"><h3 class="text-[15px] font-semibold tracking-tight">平台指标</h3></div>
      <div class="border-t border-border/70" />
      <div class="px-6 py-6">
        <div class="grid grid-cols-2 gap-x-8 gap-y-6 lg:grid-cols-5">
          <div>
            <div class="text-[13px] text-muted-foreground">商户总数</div>
            <div class="mt-2.5 text-[24px] font-normal tabular-nums">{{ data?.merchants ?? 0 }}</div>
          </div>
          <div>
            <div class="text-[13px] text-muted-foreground">订单总数</div>
            <div class="mt-2.5 text-[24px] font-normal tabular-nums">{{ data?.orders_total ?? 0 }}</div>
          </div>
          <div>
            <div class="text-[13px] text-muted-foreground">今日成功率</div>
            <div class="mt-2.5 text-[24px] font-normal tabular-nums text-success">{{ data?.success_rate ?? '0' }}%</div>
          </div>
          <div>
            <div class="text-[13px] text-muted-foreground">商户余额总额</div>
            <div class="mt-2.5 text-[24px] font-normal tabular-nums"><span class="text-sm text-muted-foreground">¥</span>{{ data?.total_money ?? '0.00' }}</div>
          </div>
          <div>
            <div class="text-[13px] text-muted-foreground">已结算总额</div>
            <div class="mt-2.5 text-[24px] font-normal tabular-nums"><span class="text-sm text-muted-foreground">¥</span>{{ data?.settled_sum ?? '0.00' }}</div>
          </div>
        </div>
      </div>
    </Card>

    <!-- ===== 待办事项 ===== -->
    <Card>
      <div class="px-6 py-4"><h3 class="text-[15px] font-semibold tracking-tight">待办事项</h3></div>
      <div class="border-t border-border/70" />
      <div class="px-6 py-6">
        <div class="grid grid-cols-2 gap-x-8 gap-y-6 sm:grid-cols-4">
          <RouterLink v-for="t in todoCards" :key="t.label" :to="t.to" class="group">
            <div class="text-[13px] text-muted-foreground">
              {{ t.label }}
              <span v-if="t.urgent && t.value > 0" class="ml-1 inline-block size-1.5 rounded-full bg-destructive align-middle" />
            </div>
            <div
              class="mt-2.5 text-[26px] font-normal leading-none tabular-nums transition-colors group-hover:text-primary"
              :class="t.urgent && t.value > 0 ? 'text-destructive' : 'text-foreground'"
            >{{ t.value }}</div>
          </RouterLink>
        </div>
      </div>
    </Card>

    <!-- ===== 两张趋势图 ===== -->
    <div class="grid grid-cols-1 gap-2.5 xl:grid-cols-2">
      <Card>
        <div class="px-6 py-4"><h3 class="text-[15px] font-semibold tracking-tight">近 7 日订单量</h3></div>
        <div class="border-t border-border/70" />
        <div class="px-6 py-5"><TrendChart :labels="trendLabels" :series="orderTrend" /></div>
      </Card>
      <Card>
        <div class="px-6 py-4"><h3 class="text-[15px] font-semibold tracking-tight">近 7 日交易额（元）</h3></div>
        <div class="border-t border-border/70" />
        <div class="px-6 py-5"><TrendChart :labels="trendLabels" :series="amountTrend" /></div>
      </Card>
    </div>

    <!-- ===== 实时订单 ===== -->
    <Card>
      <div class="flex items-center justify-between px-6 py-4">
        <h3 class="text-[15px] font-semibold tracking-tight">实时订单</h3>
        <RouterLink to="/admin/orders" class="text-xs text-muted-foreground transition-colors hover:text-primary">查看全部</RouterLink>
      </div>
      <div class="border-t border-border/70" />
      <div class="px-6 py-4">
        <div class="overflow-x-auto">
          <table class="tbl w-full table-fixed">
            <thead>
              <tr>
                <th>商户号</th>
                <th>订单号</th>
                <th>支付方式</th>
                <th>金额</th>
                <th class="col-center">状态</th>
                <th>时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="o in data?.recent ?? []" :key="o.trade_no">
                <td class="tabular-nums text-primary">{{ o.uid }}</td>
                <td class="text-xs tabular-nums tracking-wide text-muted-foreground">{{ o.trade_no }}</td>
                <td class="text-muted-foreground">{{ o.typeshowname || '—' }}</td>
                <td class="tabular-nums font-medium whitespace-nowrap">
                  <span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ o.money }}
                </td>
                <td class="col-center">
                  <Badge :variant="statusMap[o.status]?.variant || 'muted'">{{ statusMap[o.status]?.text || '未知' }}</Badge>
                </td>
                <td class="text-muted-foreground tabular-nums whitespace-nowrap">{{ o.time }}</td>
              </tr>
              <tr v-if="!loading && !(data?.recent?.length)">
                <td colspan="6" class="py-10 text-center dim">暂无订单</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </Card>
  </div>
</template>
