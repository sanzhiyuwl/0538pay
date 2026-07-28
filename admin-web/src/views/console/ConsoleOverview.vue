<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { Users, Wallet, ListOrdered, QrCode, Receipt, Settings } from 'lucide-vue-next'
import { Panel } from '@/components/ui'
import { fetchAgents, fetchEnrolls } from '@/lib/api/console'

// 平台视角概况：代理总数 / 进件单总数 / 已完成 / 审核中，靠各列表接口 pageSize:1 取 total。
const agentTotal = ref(0)
const enrollTotal = ref(0)
const finishedTotal = ref(0)
const submittedTotal = ref(0)
const loading = ref(true)

onMounted(async () => {
  try {
    const [agents, all, fin, sub] = await Promise.all([
      fetchAgents({ pageSize: 1 }),
      fetchEnrolls({ pageSize: 1 }),
      fetchEnrolls({ status: 'finished', pageSize: 1 }),
      fetchEnrolls({ status: 'submitted', pageSize: 1 }),
    ])
    agentTotal.value = agents.total
    enrollTotal.value = all.total
    finishedTotal.value = fin.total
    submittedTotal.value = sub.total
  } catch {
    // 概况为只读聚合，拉取失败不阻塞页面
  } finally {
    loading.value = false
  }
})

const stats = [
  { label: '进件代理', get: () => agentTotal.value, cls: '' },
  { label: '进件单总数', get: () => enrollTotal.value, cls: '' },
  { label: '审核中', get: () => submittedTotal.value, cls: '' },
  { label: '已开通完成', get: () => finishedTotal.value, cls: 'text-success' },
]

// 快捷入口（微信特约商户分组下的六页）
const links = [
  { to: '/console/agents', title: '代理管理', desc: '开通代理、分配权限与名额', icon: Users },
  { to: '/console/quota', title: '名额管理', desc: '售卖/调整代理预购名额', icon: Wallet },
  { to: '/console/enroll', title: '进件申请', desc: '建单、填料、提交微信审核', icon: ListOrdered },
  { to: '/console/invites', title: '邀请链接', desc: '生成客户自助进件链接', icon: QrCode },
  { to: '/console/settlement', title: '佣金结算', desc: '进件成功的佣金结算流水', icon: Receipt },
  { to: '/console/settings', title: '进件设置', desc: '进件计价规则与凭证说明', icon: Settings },
]
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="控制台概况" subtitle="平台方视角 · 进件代理与特约商户进件总览">
      <div class="grid grid-cols-2 gap-2.5 lg:grid-cols-4">
        <div v-for="s in stats" :key="s.label" class="bg-muted/40 px-4 py-3">
          <div class="text-[11px] text-muted-foreground">{{ s.label }}</div>
          <div class="mt-1 text-2xl font-semibold tabular-nums" :class="s.cls">
            {{ loading ? '—' : s.get() }}
          </div>
        </div>
      </div>
    </Panel>

    <Panel title="快捷入口" subtitle="微信特约商户进件">
      <div class="grid grid-cols-1 gap-2.5 sm:grid-cols-2 lg:grid-cols-3">
        <RouterLink
          v-for="l in links"
          :key="l.to"
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
      </div>
    </Panel>
  </div>
</template>
