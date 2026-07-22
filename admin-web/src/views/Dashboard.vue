<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { RefreshCw, ShieldAlert } from 'lucide-vue-next'
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
// 图表配色收口到设计令牌：订单量走品牌蓝、交易额走语义绿（收敛饱和度，
// 去掉原先游离在系统外的 #4b7bec / #07c160 突兀硬编码）
const orderTrend = computed(() => [
  { name: '订单量', color: '#0062ef', data: data.value?.trend.orders ?? [] },
])
const amountTrend = computed(() => [
  { name: '交易额', color: '#3a9e4f', data: (data.value?.trend.amounts ?? []).map((a) => Number(a)) },
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

// 安全告警（弱密码/默认密码等）
const alerts = computed(() => data.value?.alerts ?? [])

// 支付方式手续费利润交叉表（今+近6日 × 各支付方式）
const feeProfit = computed(() => data.value?.fee_profit ?? null)
const hasFeeProfit = computed(() => !!feeProfit.value && feeProfit.value.paytypes.length > 0)
</script>

<template>
  <div class="space-y-2.5">
    <!-- ===== 安全告警条（弱密码/默认密码等，多条合并到同一横幅，未来可承载站点公告）=====
         配色抄 Badge warning 的 Element UI 精确 hex（#fdf6ec 底 / #faecd8 边 / #985f0d 文），
         淡黄底横幅 + 安全盾牌图标，明暗主题均高对比可读，不加阴影/动效（后台告警求稳）。 -->
    <div
      v-if="alerts.length"
      class="flex gap-2.5 rounded border px-4 py-3"
      style="background:#fdf6ec;border-color:#faecd8"
    >
      <ShieldAlert class="mt-px size-[17px] shrink-0" style="color:#E6A23C" />
      <div class="min-w-0 space-y-1">
        <p
          v-for="(a, i) in alerts"
          :key="i"
          class="text-[13px] leading-relaxed"
          style="color:#985f0d"
        >{{ a }}</p>
      </div>
    </div>

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
            <div class="mt-2.5 text-[28px] font-semibold leading-none tracking-tight tabular-nums text-foreground">{{ s.today }}</div>
            <div class="mt-2 text-xs text-muted-foreground/60">昨日:{{ s.yesterday }}</div>
            <div class="mt-5">
              <div class="text-[13px] text-muted-foreground">{{ s.total_label }}</div>
              <div class="mt-2 text-[22px] font-semibold leading-none tracking-tight tabular-nums text-foreground">{{ s.total }}</div>
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
            <div class="mt-2.5 text-[24px] font-semibold tracking-tight tabular-nums">{{ data?.merchants ?? 0 }}</div>
          </div>
          <div>
            <div class="text-[13px] text-muted-foreground">订单总数</div>
            <div class="mt-2.5 text-[24px] font-semibold tracking-tight tabular-nums">{{ data?.orders_total ?? 0 }}</div>
          </div>
          <div>
            <div class="text-[13px] text-muted-foreground">今日成功率</div>
            <div class="mt-2.5 text-[24px] font-semibold tracking-tight tabular-nums text-success">{{ data?.success_rate ?? '0' }}%</div>
          </div>
          <div>
            <div class="text-[13px] text-muted-foreground">商户余额总额</div>
            <div class="mt-2.5 text-[24px] font-semibold tracking-tight tabular-nums"><span class="text-sm font-normal text-muted-foreground">¥</span>{{ data?.total_money ?? '0.00' }}</div>
          </div>
          <div>
            <div class="text-[13px] text-muted-foreground">已结算总额</div>
            <div class="mt-2.5 text-[24px] font-semibold tracking-tight tabular-nums"><span class="text-sm font-normal text-muted-foreground">¥</span>{{ data?.settled_sum ?? '0.00' }}</div>
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
              class="mt-2.5 text-[26px] font-semibold leading-none tracking-tight tabular-nums transition-colors group-hover:text-primary"
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

    <!-- ===== 支付方式手续费利润（已扣通道成本，今+近6日）===== -->
    <Card v-if="hasFeeProfit && feeProfit">
      <div class="px-6 py-4">
        <h3 class="text-[15px] font-semibold tracking-tight">支付方式手续费利润</h3>
        <span class="text-xs text-muted-foreground/70">已扣除通道成本，收入取实付额，利润取平台利润</span>
      </div>
      <div class="border-t border-border/70" />
      <div class="px-6 py-4">
        <div class="overflow-x-auto">
          <table class="tbl w-full">
            <thead>
              <tr>
                <th class="whitespace-nowrap">支付方式</th>
                <th v-for="d in feeProfit.days" :key="d" class="num whitespace-nowrap">{{ d }}</th>
              </tr>
            </thead>
            <tbody>
              <template v-for="pt in feeProfit.paytypes" :key="pt">
                <tr>
                  <td class="whitespace-nowrap">{{ pt }} <span class="dim text-xs">收入</span></td>
                  <td v-for="(v, i) in feeProfit.income[pt]" :key="i" class="num tabular-nums">
                    <span class="dim text-xs">¥</span>{{ v }}
                  </td>
                </tr>
                <tr>
                  <td class="whitespace-nowrap">{{ pt }} <span class="text-xs text-primary">利润</span></td>
                  <td v-for="(v, i) in feeProfit.profit[pt]" :key="i" class="num tabular-nums text-primary">
                    <span class="dim text-xs">¥</span>{{ v }}
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>
      </div>
    </Card>

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
