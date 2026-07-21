<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Wallet as WalletIcon, CheckSquare, AreaChart, ShoppingCart, IdCard, KeyRound, ShieldAlert, X, Megaphone } from 'lucide-vue-next'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import TrendChart from '@/components/TrendChart.vue'
import { merchantStatusMeta } from '@/lib/mock/merchant/dashboard'
import { fetchDashboard, type MerchantDashboard } from '@/lib/api/merchantCenter'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { formatMoney } from '@/lib/utils'

const router = useRouter()
const toast = useToast()

// ===== 工作台聚合数据（真接口）=====
const dash = ref<MerchantDashboard | null>(null)
async function loadDashboard() {
  try {
    dash.value = await fetchDashboard()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '工作台加载失败')
  }
}
onMounted(loadDashboard)

const m = computed(() => dash.value?.merchantInfo)
const channelStats = computed(() => dash.value?.channels ?? [])
const announces = computed(() => dash.value?.announces ?? [])
const settleTrend = computed(() => dash.value?.trend ?? { labels: [], data: [] })

// 四数据卡片
const cards = computed(() => [
  { label: '商户当前余额', value: formatMoney(m.value?.money ?? 0), money: true, icon: WalletIcon, color: 'text-primary' },
  { label: '已结算余额', value: formatMoney(m.value?.settleMoney ?? 0), money: true, icon: CheckSquare, color: 'text-success' },
  { label: '订单总数', value: String(m.value?.orders ?? 0), money: false, icon: AreaChart, color: 'text-foreground' },
  { label: '今日订单', value: String(m.value?.ordersToday ?? 0), money: false, icon: ShoppingCart, color: 'text-warning' },
])

const statusMeta = computed(() => merchantStatusMeta[m.value?.status ?? 'normal'] ?? merchantStatusMeta.normal)

// 结算趋势图
const settleSeries = computed(() => [
  { name: '结算金额', color: '#4b7bec', data: settleTrend.value.data },
])

// 提醒横幅（可关闭）
const closed = ref<Set<string>>(new Set())
const alerts = computed(() => {
  const al = dash.value?.alerts
  if (!al) return []
  const list: { key: string; text: string }[] = []
  if (al.needCert) list.push({ key: 'cert', text: '您的账户尚未完成实名认证，部分功能受限，请尽快认证。' })
  if (al.noSecurity) list.push({ key: 'security', text: '您还未绑定密保手机/邮箱，账户存在安全风险，建议前往账户设置绑定。' })
  if (al.noLoginPwd) list.push({ key: 'pwd', text: '您还未设置登录密码，仅能使用密钥登录，建议设置登录密码。' })
  return list.filter((a) => !closed.value.has(a.key))
})
function closeAlert(key: string) {
  closed.value.add(key)
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 顶部提醒横幅 -->
    <div v-for="a in alerts" :key="a.key" class="flex items-center gap-2 border border-warning/30 bg-warning/[0.08] px-4 py-2.5 text-sm text-warning">
      <ShieldAlert class="size-4 shrink-0" />
      <span class="flex-1">{{ a.text }}</span>
      <button class="text-warning/60 hover:text-warning" @click="closeAlert(a.key)"><X class="size-4" /></button>
    </div>

    <!-- 四数据卡片 -->
    <div class="grid grid-cols-2 gap-2.5 lg:grid-cols-4">
      <Card v-for="c in cards" :key="c.label">
        <div class="flex items-center gap-4 px-5 py-5">
          <div class="flex size-12 shrink-0 items-center justify-center rounded-xl bg-primary/[0.08]">
            <component :is="c.icon" :class="['size-6', c.color]" />
          </div>
          <div class="min-w-0">
            <div class="text-[13px] text-muted-foreground">{{ c.label }}</div>
            <div class="mt-1 text-2xl font-normal leading-none tabular-nums">
              <span v-if="c.money" class="mr-0.5 text-sm text-muted-foreground">¥</span>{{ c.value }}
            </div>
          </div>
        </div>
      </Card>
    </div>

    <div class="grid grid-cols-1 gap-2.5 xl:grid-cols-2">
      <!-- 商户资料卡 -->
      <Card>
        <div class="flex items-center justify-between px-6 py-4">
          <h3 class="text-[15px] font-semibold tracking-tight">商户资料</h3>
          <div class="flex gap-2">
            <button
              class="rounded border border-border px-2.5 py-1 text-xs text-muted-foreground transition-colors hover:border-primary/40 hover:text-primary"
              @click="router.push('/m/profile')"
            >
              <IdCard class="mr-1 inline size-3.5" />修改资料
            </button>
            <button
              class="rounded border border-border px-2.5 py-1 text-xs text-muted-foreground transition-colors hover:border-primary/40 hover:text-primary"
              @click="router.push('/m/api')"
            >
              <KeyRound class="mr-1 inline size-3.5" />API 信息
            </button>
          </div>
        </div>
        <div class="border-t border-border/70" />
        <div class="px-6 py-5">
          <div class="flex items-center gap-4">
            <img src="/images/avatar-default.png" alt="avatar" class="size-14 rounded-full object-cover" />
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="text-base font-semibold">欢迎您，{{ m?.name }}</span>
                <Badge :variant="statusMeta.variant">{{ statusMeta.text }}</Badge>
              </div>
              <div class="mt-1 text-sm text-muted-foreground">商户号 {{ m?.uid }} · {{ m?.groupName }}</div>
            </div>
          </div>
          <div class="mt-5 grid grid-cols-2 gap-4 border-t border-border/60 pt-5">
            <div>
              <div class="text-[13px] text-muted-foreground">今日收入</div>
              <div class="mt-1.5 text-xl font-normal tabular-nums text-success">
                <span class="text-sm opacity-70">¥</span>{{ formatMoney(m?.todayIncome ?? 0) }}
              </div>
            </div>
            <div>
              <div class="text-[13px] text-muted-foreground">昨日收入</div>
              <div class="mt-1.5 text-xl font-normal tabular-nums">
                <span class="text-sm opacity-70">¥</span>{{ formatMoney(m?.yesterdayIncome ?? 0) }}
              </div>
            </div>
          </div>
        </div>
      </Card>

      <!-- 公告通知 -->
      <Card>
        <div class="flex items-center gap-2 px-6 py-4">
          <Megaphone class="size-4 text-primary" />
          <h3 class="text-[15px] font-semibold tracking-tight">公告通知</h3>
        </div>
        <div class="border-t border-border/70" />
        <div class="divide-y divide-border/60">
          <div v-for="a in announces" :key="a.id" class="flex items-start gap-3 px-6 py-3.5">
            <span class="mt-1.5 size-1.5 shrink-0 rounded-full bg-primary/60" />
            <div class="min-w-0 flex-1">
              <p class="text-sm" :style="a.color ? { color: a.color } : {}">{{ a.content }}</p>
              <p class="mt-1 text-xs text-muted-foreground">{{ a.time }}</p>
            </div>
          </div>
        </div>
      </Card>
    </div>

    <!-- 收入统计与通道费率 -->
    <Card>
      <div class="px-6 py-4">
        <h3 class="text-[15px] font-semibold tracking-tight">收入统计与通道费率</h3>
      </div>
      <div class="border-t border-border/70" />
      <div class="px-6 py-4">
        <div class="overflow-x-auto">
          <table class="tbl w-full">
            <thead>
              <tr>
                <th>支付方式</th>
                <th class="num">今日金额</th>
                <th class="num">昨日金额</th>
                <th class="num">成功率</th>
                <th class="num">费率</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="c in channelStats" :key="c.typename">
                <td>
                  <div class="flex items-center gap-1.5">
                    <img :src="`/assets/icon/${c.typename}.ico`" class="size-4" onerror="this.style.display='none'" />
                    <span>{{ c.showname }}</span>
                  </div>
                </td>
                <td class="num tabular-nums font-medium"><span class="dim text-xs">¥</span>{{ formatMoney(c.today) }}</td>
                <td class="num tabular-nums"><span class="dim text-xs">¥</span>{{ formatMoney(c.yesterday) }}</td>
                <td class="num tabular-nums">{{ c.successRate }}%</td>
                <td class="num tabular-nums text-muted-foreground">{{ c.rate }}%</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </Card>

    <!-- 结算金额趋势 -->
    <Card>
      <div class="px-6 py-4">
        <h3 class="text-[15px] font-semibold tracking-tight">结算金额趋势</h3>
      </div>
      <div class="border-t border-border/70" />
      <div class="px-6 py-5"><TrendChart :labels="settleTrend.labels" :series="settleSeries" /></div>
    </Card>
  </div>
</template>
