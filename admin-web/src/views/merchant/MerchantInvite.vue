<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { Copy, Check, Gift, ArrowRight } from 'lucide-vue-next'
import { Panel, Button, Pagination } from '@/components/ui'
import { inviteInfo, inviteStat, invitedUsers } from '@/lib/mock/merchant/groupbuy'
import { computed } from 'vue'
import { formatMoney } from '@/lib/utils'

const router = useRouter()

const copied = ref(false)
function copyLink() {
  navigator.clipboard?.writeText(inviteInfo.link).then(() => {
    copied.value = true
    setTimeout(() => (copied.value = false), 1500)
  }).catch(() => {})
}

// 分页
const page = ref(1)
const pageSize = 10
const total = invitedUsers.length
const pageCount = Math.max(1, Math.ceil(total / pageSize))
const pageRows = computed(() => invitedUsers.slice((page.value - 1) * pageSize, page.value * pageSize))
function go(p: number) {
  page.value = Math.min(Math.max(1, p), pageCount)
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 推广链接 + 规则 -->
    <Panel title="邀请返现" subtitle="邀请好友注册，其每笔支付你都能获得返现">
      <div class="max-w-2xl space-y-3.5">
        <div class="row-field">
          <label class="lbl">专属推广链接</label>
          <div class="flex flex-1 items-center gap-2">
            <input :value="inviteInfo.link" readonly class="field-input flex-1 bg-muted/40 font-mono text-[13px]" />
            <Button variant="outline" size="sm" @click="copyLink">
              <component :is="copied ? Check : Copy" class="size-4" />{{ copied ? '已复制' : '复制' }}
            </Button>
          </div>
        </div>
        <div class="flex items-center gap-2 rounded bg-primary/[0.06] px-3 py-2.5 text-sm">
          <Gift class="size-4 text-primary" />
          <span>返现比例 <b class="text-primary">{{ inviteInfo.rate }}%</b>，按{{ inviteInfo.orderType === 1 ? '订单手续费' : '订单金额' }}计（分成不超过订单手续费）。</span>
        </div>
      </div>
    </Panel>

    <!-- 返现统计 -->
    <Panel title="返现统计">
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">已邀请</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ inviteStat.users }} 人</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">今日返现</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success"><span class="text-sm opacity-70">¥</span>{{ formatMoney(inviteStat.incomeToday) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">昨日返现</div>
          <div class="mt-1 text-xl font-normal tabular-nums"><span class="text-sm opacity-70">¥</span>{{ formatMoney(inviteStat.incomeYesterday) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">累计返现</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-primary"><span class="text-sm opacity-70">¥</span>{{ formatMoney(inviteStat.incomeTotal) }}</div>
        </div>
        <div class="ml-auto flex items-end">
          <Button variant="outline" size="sm" @click="router.push({ path: '/m/records' })">
            查看返现记录<ArrowRight />
          </Button>
        </div>
      </div>
    </Panel>

    <!-- 已邀请用户 -->
    <Panel title="已邀请用户" :subtitle="`${total} 人`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[50%]">商户号</th>
              <th class="w-[50%]">注册时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in pageRows" :key="u.uid">
              <td class="tabular-nums text-primary">{{ u.uid }}</td>
              <td class="text-xs">{{ u.addtime }}</td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="2" class="py-10 text-center dim">还没有邀请任何用户</td>
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
