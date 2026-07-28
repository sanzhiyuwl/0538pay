<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { Wallet, ListOrdered, QrCode, Receipt } from 'lucide-vue-next'
import { Panel } from '@/components/ui'
import { useAgentAuthStore } from '@/stores/agentAuth'
import { fetchMyWallet, fetchMyEnrolls } from '@/lib/api/agent'
import type { QuotaWallet } from '@/lib/api/console'

const agentAuth = useAgentAuthStore()
const wallet = ref<QuotaWallet | null>(null)
const enrollTotal = ref(0)
const finishedTotal = ref(0)

onMounted(async () => {
  if (agentAuth.has('quota')) {
    try {
      wallet.value = await fetchMyWallet()
    } catch {
      // 无权限或未开钱包，忽略
    }
  }
  if (agentAuth.has('enroll')) {
    try {
      const all = await fetchMyEnrolls({ pageSize: 1 })
      enrollTotal.value = all.total
      const fin = await fetchMyEnrolls({ status: 'finished', pageSize: 1 })
      finishedTotal.value = fin.total
    } catch {
      // ignore
    }
  }
})

// 快捷入口（按权限显示）
const links = [
  { to: '/agent/enroll', title: '进件申请', desc: '发起并管理名下进件单', icon: ListOrdered, perm: 'enroll' },
  { to: '/agent/quota', title: '名额钱包', desc: '查看名额余额与流水', icon: Wallet, perm: 'quota' },
  { to: '/agent/invites', title: '邀请链接', desc: '生成客户自助进件链接', icon: QrCode, perm: 'invite' },
  { to: '/agent/settlement', title: '佣金结算', desc: '查看进件佣金结算流水', icon: Receipt, perm: 'settlement' },
]
</script>

<template>
  <div class="space-y-2.5">
    <Panel :title="`你好，${agentAuth.name || '代理'}`" subtitle="代理工作台 · 仅展示你名下的数据">
      <div class="grid grid-cols-2 gap-2.5 lg:grid-cols-4">
        <div v-if="agentAuth.has('quota')" class="bg-muted/40 px-4 py-3">
          <div class="text-[11px] text-muted-foreground">名额余额</div>
          <div class="mt-1 text-2xl font-semibold tabular-nums">{{ wallet?.balance ?? '—' }}</div>
        </div>
        <div v-if="agentAuth.has('quota')" class="bg-muted/40 px-4 py-3">
          <div class="text-[11px] text-muted-foreground">累计购买名额</div>
          <div class="mt-1 text-2xl font-semibold tabular-nums">{{ wallet?.total_buy ?? '—' }}</div>
        </div>
        <div v-if="agentAuth.has('enroll')" class="bg-muted/40 px-4 py-3">
          <div class="text-[11px] text-muted-foreground">名下进件单</div>
          <div class="mt-1 text-2xl font-semibold tabular-nums">{{ enrollTotal }}</div>
        </div>
        <div v-if="agentAuth.has('enroll')" class="bg-muted/40 px-4 py-3">
          <div class="text-[11px] text-muted-foreground">已开通完成</div>
          <div class="mt-1 text-2xl font-semibold tabular-nums text-success">{{ finishedTotal }}</div>
        </div>
      </div>
    </Panel>

    <Panel title="快捷入口" subtitle="按平台开通的权限显示">
      <div class="grid grid-cols-1 gap-2.5 sm:grid-cols-2 lg:grid-cols-4">
        <template v-for="l in links" :key="l.to">
          <RouterLink
            v-if="agentAuth.has(l.perm)"
            :to="l.to"
            class="flex items-center gap-3 bg-muted/40 px-4 py-3.5 transition-colors hover:bg-muted/70"
          >
            <div class="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/12 text-primary">
              <component :is="l.icon" class="size-5" />
            </div>
            <div class="leading-tight">
              <div class="text-sm font-medium">{{ l.title }}</div>
              <div class="text-[11px] text-muted-foreground">{{ l.desc }}</div>
            </div>
          </RouterLink>
        </template>
      </div>
      <p class="mt-3 text-[11px] text-muted-foreground">
        没有看到某个入口，说明平台尚未给你开通该功能权限，可联系平台运营开通。
      </p>
    </Panel>
  </div>
</template>
