<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Crown, Check, Minus, Plus, Layers, CalendarClock, Sparkles } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Modal } from '@/components/ui'
import { fetchGroups, buyGroup, type GroupPlan, type GroupCurrent } from '@/lib/api/merchantCenter'
import { ApiError } from '@/lib/api/client'
import { useMerchantAuthStore } from '@/stores/merchantAuth'
import { useToast } from '@/composables/useToast'
import { formatMoney } from '@/lib/utils'

const toast = useToast()
const auth = useMerchantAuthStore()

const plans = ref<GroupPlan[]>([])
const current = ref<GroupCurrent>({ gid: 0, name: '', expire: '—', rates: [] })
const busy = ref(false)
const loading = ref(true)

// 余额支付即时；渠道待凭证
const buyPayOptions = [{ value: 'balance', label: '余额支付' }]

async function load() {
  try {
    const res = await fetchGroups()
    plans.value = res.plans
    current.value = res.current
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载通道套餐失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

// 购买弹窗
const buyOpen = ref(false)
const plan = ref<GroupPlan | null>(null)
const num = ref(1)
const payType = ref('balance')
function openBuy(p: GroupPlan) {
  plan.value = p
  num.value = 1
  payType.value = 'balance'
  buyOpen.value = true
}
const isRenew = computed(() => !!plan.value && plan.value.id === current.value.gid)
// 默认用户组（gid=0）：epay 自带的免费基础组，随机通道+默认费率开箱即用，无到期概念。
const isDefaultGroup = computed(() => current.value.gid === 0)
const totalPrice = computed(() => {
  if (!plan.value) return 0
  return plan.value.expire === 0 ? plan.value.price : plan.value.price * num.value
})
function decNum() {
  if (num.value > 1) num.value--
}
function incNum() {
  num.value++
}
async function submitBuy() {
  if (!plan.value || busy.value) return
  busy.value = true
  try {
    await buyGroup(plan.value.id, num.value, payType.value)
    toast.success(`已开通 ${plan.value.name}`)
    buyOpen.value = false
    await Promise.all([load(), auth.refreshInfo()])
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '开通失败')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 当前套餐概览 -->
    <Panel no-header body-class="p-0">
      <div class="flex flex-col gap-4 p-5 sm:flex-row sm:items-center">
        <div class="flex items-center gap-4">
          <div class="flex size-14 items-center justify-center rounded-2xl bg-primary/[0.08] text-primary">
            <Crown class="size-7" />
          </div>
          <div>
            <div class="text-xs text-muted-foreground">当前套餐</div>
            <div class="mt-1 flex items-center gap-2">
              <span class="text-xl font-semibold tracking-tight">{{ current.name || '默认用户组' }}</span>
              <Badge :variant="isDefaultGroup ? 'muted' : 'success'">{{ isDefaultGroup ? '免费基础' : '生效中' }}</Badge>
            </div>
          </div>
        </div>
        <div class="flex items-center gap-2 text-sm text-muted-foreground sm:ml-auto">
          <template v-if="isDefaultGroup">
            <Sparkles class="size-4" />
            <span>长期有效</span>
          </template>
          <template v-else>
            <CalendarClock class="size-4" />
            <span>到期时间</span>
            <span class="font-medium text-foreground tabular-nums">{{ current.expire }}</span>
          </template>
        </div>
      </div>
      <!-- 当前组可用通道及费率 -->
      <div class="border-t border-border/60 px-5 py-4">
        <div class="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
          <Layers class="size-3.5" />
          <span>可用支付通道</span>
          <span v-if="current.rates.length" class="tabular-nums">· {{ current.rates.length }} 个</span>
        </div>
        <div v-if="current.rates.length" class="mt-3 flex flex-wrap gap-2">
          <span
            v-for="r in current.rates"
            :key="r.label"
            class="inline-flex items-center gap-1.5 bg-muted/40 px-2.5 py-1 text-sm"
          >
            <Check class="size-3.5 shrink-0 text-success" />
            <span class="text-foreground">{{ r.label }}</span>
            <span v-if="r.rate" class="font-medium tabular-nums text-primary">{{ r.rate }}%</span>
            <span v-else class="text-xs text-muted-foreground">默认费率</span>
          </span>
        </div>
        <div v-else class="mt-2 flex items-center gap-2 text-sm text-muted-foreground">
          <Minus class="size-4 shrink-0" />
          <span>暂无可用支付通道</span>
        </div>
      </div>
      <div class="flex items-center gap-2 border-t border-border/60 bg-muted/40 px-5 py-2.5 text-xs text-muted-foreground">
        <Sparkles class="size-3.5 text-primary" />
        <span>{{ isDefaultGroup ? '默认用户组已开通基础支付通道，开通更高套餐可享更低费率与更多通道。' : '开通更高套餐可解锁更多支付通道、享受更低费率。' }}</span>
      </div>
    </Panel>

    <!-- 套餐卡片 -->
    <div v-if="loading" class="grid grid-cols-1 gap-2.5 md:grid-cols-3">
      <div v-for="i in 3" :key="i" class="h-72 animate-pulse bg-card" />
    </div>

    <div v-else class="grid grid-cols-1 gap-2.5 md:grid-cols-3">
      <div
        v-for="p in plans"
        :key="p.id"
        class="flex flex-col bg-card"
        :class="p.id === current.gid ? 'ring-1 ring-primary/30' : ''"
      >
        <!-- 头部：名称 + 价格 -->
        <div class="flex flex-col gap-3 p-5">
          <div class="flex items-center gap-2">
            <span class="text-base font-semibold tracking-tight">{{ p.name }}</span>
            <Badge v-if="p.id === current.gid" variant="success">当前套餐</Badge>
          </div>
          <div class="flex items-baseline gap-1">
            <span class="text-sm text-muted-foreground">¥</span>
            <span class="text-[34px] font-semibold leading-none tabular-nums">{{ formatMoney(p.price) }}</span>
            <span class="ml-1 text-sm text-muted-foreground">/ {{ p.expire === 0 ? '永久' : '月' }}</span>
          </div>
        </div>

        <div class="border-t border-border/60" />

        <!-- 通道及费率清单 -->
        <div class="flex flex-1 flex-col p-5">
          <div class="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
            <Layers class="size-3.5" />
            <span>可用支付通道</span>
            <span v-if="p.rates.length" class="tabular-nums">· {{ p.rates.length }} 个</span>
          </div>
          <ul class="mt-3 space-y-2.5 text-sm">
            <!-- rate 为空表示随通道默认费率，只显示通道名。对齐 epay groupbuy display_info。 -->
            <li v-for="r in p.rates" :key="r.label" class="flex items-center gap-2">
              <Check class="size-4 shrink-0 text-success" />
              <span class="text-foreground">{{ r.label }}</span>
              <span v-if="r.rate" class="ml-auto font-medium tabular-nums text-primary">{{ r.rate }}%</span>
              <span v-else class="ml-auto text-xs text-muted-foreground">默认费率</span>
            </li>
            <li v-if="!p.rates.length" class="flex items-center gap-2 text-muted-foreground">
              <Minus class="size-4 shrink-0" />
              <span>未配置可用支付通道</span>
            </li>
          </ul>

          <div class="mt-4 flex items-center gap-2 border-t border-border/60 pt-3 text-sm">
            <CalendarClock class="size-4 shrink-0 text-muted-foreground" />
            <span class="text-muted-foreground">有效期</span>
            <span class="ml-auto font-medium">{{ p.expire === 0 ? '永久' : `${p.expire} 个月` }}</span>
          </div>

          <Button
            class="mt-5 w-full"
            :variant="p.id === current.gid ? 'outline' : 'default'"
            @click="openBuy(p)"
          >
            {{ p.id === current.gid ? '续期' : '立即开通' }}
          </Button>
        </div>
      </div>
    </div>

    <div v-if="!loading && !plans.length" class="bg-card px-6 py-16 text-center text-sm text-muted-foreground">
      暂无可开通的通道套餐
    </div>

    <!-- 开通弹窗 -->
    <Modal v-model="buyOpen" :title="plan ? `${isRenew ? '续期' : '开通'} ${plan.name}` : '开通套餐'" width="max-w-md">
      <div v-if="plan" class="space-y-3.5">
        <div v-if="plan.expire !== 0" class="row-field">
          <label class="lbl">购买时长</label>
          <div class="flex flex-1 items-center gap-3">
            <div class="flex items-center border border-border">
              <button class="flex size-8 items-center justify-center text-muted-foreground transition-colors hover:bg-accent disabled:opacity-40" :disabled="num <= 1" @click="decNum"><Minus class="size-4" /></button>
              <span class="w-12 text-center tabular-nums">{{ num }}</span>
              <button class="flex size-8 items-center justify-center text-muted-foreground transition-colors hover:bg-accent" @click="incNum"><Plus class="size-4" /></button>
            </div>
            <span class="text-sm text-muted-foreground">个月</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">支付方式</label>
          <Select v-model="payType" :options="buyPayOptions" class="flex-1" />
        </div>
        <div class="flex items-center justify-between border-t border-border/60 pt-3">
          <span class="text-sm text-muted-foreground">应付金额</span>
          <span class="text-xl font-semibold tabular-nums text-primary">¥{{ formatMoney(totalPrice) }}</span>
        </div>
        <p class="text-xs text-muted-foreground">余额支付即时扣款并开通/续期套餐；渠道支付待支付渠道凭证接入。</p>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="buyOpen = false">取消</Button>
        <Button size="sm" :disabled="busy" @click="submitBuy">确认支付</Button>
      </template>
    </Modal>
  </div>
</template>
