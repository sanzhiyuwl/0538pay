<script setup lang="ts">
import { ref, computed } from 'vue'
import { Crown, Check, Minus, Plus } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Modal } from '@/components/ui'
import { groupPlans, currentGroup, buyPayOptions, type GroupPlan } from '@/lib/mock/merchant/groupbuy'
import { formatMoney } from '@/lib/utils'

// 购买弹窗
const buyOpen = ref(false)
const plan = ref<GroupPlan | null>(null)
const num = ref(1) // 购买月数
const payType = ref('0')
function openBuy(p: GroupPlan) {
  plan.value = p
  num.value = 1
  payType.value = '0'
  buyOpen.value = true
}
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
function submitBuy() {
  buyOpen.value = false
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 当前等级 -->
    <Panel title="购买会员" subtitle="升级会员享受更低费率与更多支付通道">
      <div class="flex items-center gap-4">
        <div class="flex size-12 items-center justify-center rounded-xl bg-primary/[0.08] text-primary">
          <Crown class="size-6" />
        </div>
        <div>
          <div class="text-sm text-muted-foreground">当前等级</div>
          <div class="mt-0.5 text-lg font-semibold">{{ currentGroup.name }}</div>
        </div>
        <div class="ml-auto text-right">
          <div class="text-sm text-muted-foreground">到期时间</div>
          <div class="mt-0.5 text-sm">{{ currentGroup.expire }}</div>
        </div>
      </div>
    </Panel>

    <!-- 套餐卡片 -->
    <div class="grid grid-cols-1 gap-2.5 md:grid-cols-3">
      <Panel v-for="p in groupPlans" :key="p.id" :title="p.name">
        <template #actions>
          <Badge v-if="p.recommended" variant="success">推荐</Badge>
        </template>
        <div class="flex flex-col">
          <div class="flex items-baseline gap-1">
            <span class="text-3xl font-semibold tabular-nums">¥{{ formatMoney(p.price) }}</span>
            <span class="text-sm text-muted-foreground">/ {{ p.expire === 0 ? '永久' : '月' }}</span>
          </div>
          <ul class="mt-4 space-y-2 text-sm">
            <li v-for="r in p.rates" :key="r.label" class="flex items-center gap-2">
              <Check class="size-4 text-success" />
              <span class="text-muted-foreground">{{ r.label }}费率</span>
              <span class="ml-auto font-medium tabular-nums">{{ r.rate }}%</span>
            </li>
            <li class="flex items-center gap-2">
              <Check class="size-4 text-success" />
              <span class="text-muted-foreground">有效期</span>
              <span class="ml-auto">{{ p.expire === 0 ? '永久' : `${p.expire} 个月` }}</span>
            </li>
          </ul>
          <Button class="mt-5 w-full" :variant="p.recommended ? 'default' : 'outline'" @click="openBuy(p)">
            立即购买
          </Button>
        </div>
      </Panel>
    </div>

    <!-- 购买弹窗 -->
    <Modal v-model="buyOpen" :title="plan ? `购买 ${plan.name}` : '购买会员'" width="max-w-md">
      <div v-if="plan" class="space-y-3.5">
        <div v-if="plan.expire !== 0" class="row-field">
          <label class="lbl">购买时长</label>
          <div class="flex flex-1 items-center gap-3">
            <div class="flex items-center border border-border">
              <button class="flex size-8 items-center justify-center text-muted-foreground hover:bg-accent" @click="decNum"><Minus class="size-4" /></button>
              <span class="w-12 text-center tabular-nums">{{ num }}</span>
              <button class="flex size-8 items-center justify-center text-muted-foreground hover:bg-accent" @click="incNum"><Plus class="size-4" /></button>
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
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="buyOpen = false">取消</Button>
        <Button size="sm" @click="submitBuy">确认支付</Button>
      </template>
    </Modal>
  </div>
</template>
