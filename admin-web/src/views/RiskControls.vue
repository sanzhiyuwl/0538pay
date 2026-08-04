<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Search, Eye, RefreshCw, ShieldAlert, ExternalLink, History } from 'lucide-vue-next'
import { Panel, Button, Badge, Drawer } from '@/components/ui'
import {
  adminListChannelControls,
  adminRefreshChannelControl,
  adminRefreshAllChannelControls,
  adminListChannelControlFlows,
  type ChannelControlView,
  type ChannelControlOverview,
  type ChannelControlFlowItem,
} from '@/lib/api/riskControl'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const list = ref<ChannelControlView[]>([])
const overview = reactive<ChannelControlOverview>({
  approved_total: 0,
  controlled: 0,
  delayed: 0,
  normal: 0,
  never_queried: 0,
})
const loading = ref(false)
const refreshingAll = ref(false)
const refreshingIds = reactive<Set<number>>(new Set())

const keyword = ref('')
const tab = ref<'all' | 'controlled' | 'delayed' | 'normal'>('all')
const tabs = [
  { v: 'all', t: '全部' },
  { v: 'controlled', t: '被管控' },
  { v: 'delayed', t: '延迟管控' },
  { v: 'normal', t: '正常' },
] as const

// 管控态徽章：未刷新过灰底提示，其余按态映射（被管控红/延迟管控橙/正常绿）。
function stateVariant(v: ChannelControlView): 'success' | 'warning' | 'destructive' | 'muted' {
  if (!v.queried) return 'muted'
  if (v.state === 'controlled') return 'destructive'
  if (v.state === 'delayed') return 'warning'
  return 'success'
}
function stateText(v: ChannelControlView): string {
  return v.queried ? v.state_text : '未刷新'
}

const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  return list.value.filter((v) => {
    if (tab.value !== 'all' && v.state !== tab.value) return false
    if (!kw) return true
    return (
      String(v.uid).includes(kw) ||
      v.sub_mchid.toLowerCase().includes(kw) ||
      (v.merchant_name || '').toLowerCase().includes(kw) ||
      (v.merchant_phone || '').includes(kw) ||
      (v.channel_name || '').toLowerCase().includes(kw)
    )
  })
})

async function load() {
  loading.value = true
  try {
    const res = await adminListChannelControls()
    list.value = res.list
    Object.assign(overview, res.overview)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

// 单个商户刷新：现查微信落快照，回填该行。
async function refreshOne(v: ChannelControlView) {
  if (refreshingIds.has(v.enroll_id)) return
  refreshingIds.add(v.enroll_id)
  try {
    const res = await adminRefreshChannelControl(v.enroll_id)
    if (res.views?.length) {
      const idx = list.value.findIndex((x) => x.enroll_id === v.enroll_id)
      if (idx >= 0) list.value[idx] = res.views[0]
      // 详情抽屉同步刷新
      if (detail.value?.enroll_id === v.enroll_id) detail.value = res.views[0]
    }
    await reloadOverview()
    toast.success('已刷新管控状态')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '刷新失败')
  } finally {
    refreshingIds.delete(v.enroll_id)
  }
}

// 批量刷新：后端串行 + 限速（避免微信 429），前端等待整体返回后重载。
async function refreshAll() {
  if (refreshingAll.value) return
  refreshingAll.value = true
  try {
    const res = await adminRefreshAllChannelControls()
    await load()
    if (res.failed > 0) toast.info(`已刷新 ${res.refreshed} 家，${res.failed} 家失败`)
    else toast.success(`已刷新全部 ${res.refreshed} 家`)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '批量刷新失败')
  } finally {
    refreshingAll.value = false
  }
}

// 概览单独轻量重载（单刷后更新概览卡，不动列表其余行）。
async function reloadOverview() {
  try {
    const res = await adminListChannelControls()
    list.value = res.list
    Object.assign(overview, res.overview)
  } catch {
    /* 概览刷新失败不打扰 */
  }
}

// —— 详情抽屉：某商户的被管控原因 + 解脱路径明细 + 处置流水时间线 ——
const drawerOpen = ref(false)
const detail = ref<ChannelControlView | null>(null)
const flows = ref<ChannelControlFlowItem[]>([])
const flowsLoading = ref(false)

async function loadFlows(enrollId: number) {
  flows.value = []
  flowsLoading.value = true
  try {
    const res = await adminListChannelControlFlows(enrollId)
    flows.value = res.list
  } catch {
    /* 流水加载失败不打扰，时间线留空 */
  } finally {
    flowsLoading.value = false
  }
}

function openDetail(v: ChannelControlView) {
  detail.value = v
  drawerOpen.value = true
  loadFlows(v.enroll_id)
}

// 流水业务态徽章：管控/延迟管控红橙，解除/撤销绿，其余灰。
function flowStateVariant(f: ChannelControlFlowItem): 'success' | 'warning' | 'destructive' | 'muted' {
  if (f.business_state === 'RECOVERY' || f.business_state === 'PUNISHMENTCANCELLED') return 'success'
  if (f.business_state === 'DELAYPUNISHMENT') return 'warning'
  if (f.business_state === 'PUNISHMENT') return 'destructive'
  // (A) 处置通知无 business_state，按机制统一按处置类红显。
  return f.mechanism === 'violation' ? 'destructive' : 'muted'
}
function flowTitle(f: ChannelControlFlowItem): string {
  if (f.mechanism === 'violation') return f.punish_plan || f.summary || '商户平台处置通知'
  return f.business_state_text || f.summary || '管控流水'
}

</script>

<template>
  <div class="space-y-2.5">
    <!-- 概览 -->
    <Panel
      title="商户风控管控"
      subtitle="已开通子商户在微信侧的被管控能力及原因（查询子商户管控情况 4012803072）。被关闭收单的商户，通道对他下单硬锁不回退平台号（守不走二清红线）"
    >
      <div class="flex flex-wrap items-center gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">已开通总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ overview.approved_total }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">被管控</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-destructive">{{ overview.controlled }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">延迟管控</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-warning">{{ overview.delayed }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">正常</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ overview.normal }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">未刷新</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-muted-foreground">{{ overview.never_queried }}</div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="管控列表" :subtitle="`共 ${filtered.length} 家`">
      <template #title-extra>
        <div class="ml-4 flex items-center gap-1">
          <button
            v-for="f in tabs"
            :key="f.v"
            class="rounded-full px-3 py-1 text-[13px] transition-colors"
            :class="tab === f.v
              ? 'bg-primary/10 font-medium text-primary'
              : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'"
            @click="tab = f.v"
          >
            {{ f.t }}
          </button>
        </div>
      </template>

      <template #actions>
        <div class="flex items-center gap-2">
          <div class="relative">
            <Search class="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <input v-model="keyword" class="field-input !pl-9 w-56" placeholder="商户 / 子商户号 / 通道" />
          </div>
          <Button size="sm" variant="outline" :disabled="refreshingAll" @click="refreshAll">
            <RefreshCw class="size-3.5" :class="{ 'animate-spin': refreshingAll }" />批量刷新
          </Button>
        </div>
      </template>

      <div class="overflow-x-auto">
        <table class="tbl w-full">
          <thead>
            <tr>
              <th class="w-[14%]">商户</th>
              <th class="w-[14%]">服务商通道</th>
              <th class="w-[13%]">子商户号</th>
              <th class="w-[9%]">管控状态</th>
              <th class="w-[18%]">被管控能力</th>
              <th class="w-[12%]">原因类型</th>
              <th class="w-[12%]">解脱路径</th>
              <th class="w-[8%] col-center">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="v in filtered" :key="v.enroll_id">
              <td>
                <div class="tabular-nums">{{ v.uid }}</div>
                <div class="mt-0.5 truncate text-xs text-muted-foreground" :title="v.merchant_name">{{ v.merchant_name || '—' }}</div>
              </td>
              <td class="truncate" :title="v.channel_name">{{ v.channel_name || '—' }}</td>
              <td class="tabular-nums">{{ v.sub_mchid || '—' }}</td>
              <td><Badge :variant="stateVariant(v)">{{ stateText(v) }}</Badge></td>
              <td>
                <template v-if="v.queried && (v.limited_function_texts?.length || v.other_limited_functions)">
                  <span
                    v-for="(t, i) in v.limited_function_texts"
                    :key="i"
                    class="mr-1 mb-1 inline-block rounded bg-destructive/10 px-1.5 py-0.5 text-xs text-destructive"
                  >{{ t }}</span>
                  <span v-if="v.other_limited_functions" class="text-xs text-muted-foreground">{{ v.other_limited_functions }}</span>
                </template>
                <span v-else class="text-xs dim">—</span>
              </td>
              <td class="text-xs">
                <template v-if="v.recovery?.length">
                  <div v-for="(r, i) in v.recovery" :key="i" class="truncate" :title="r.limitation_reason_type_text">
                    {{ r.limitation_reason_type_text || '—' }}
                  </div>
                </template>
                <span v-else class="dim">—</span>
              </td>
              <td class="text-xs text-muted-foreground">
                <template v-if="v.recovery?.length">
                  <div v-for="(r, i) in v.recovery" :key="i" class="truncate" :title="r.recover_way_text">
                    {{ r.recover_way_text || '—' }}
                  </div>
                </template>
                <span v-else class="dim">—</span>
              </td>
              <td class="col-center">
                <div class="inline-flex items-center gap-0.5">
                  <button
                    class="inline-flex size-7 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                    title="刷新管控状态"
                    :disabled="refreshingIds.has(v.enroll_id)"
                    @click="refreshOne(v)"
                  >
                    <RefreshCw class="size-4" :class="{ 'animate-spin': refreshingIds.has(v.enroll_id) }" />
                  </button>
                  <button
                    class="inline-flex size-7 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                    title="查看详情"
                    @click="openDetail(v)"
                  >
                    <Eye class="size-4" />
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && !filtered.length">
              <td colspan="8" class="py-10 text-center dim">暂无已开通子商户</td>
            </tr>
          </tbody>
        </table>
      </div>
    </Panel>

    <!-- 详情抽屉：被管控原因 + 解脱路径明细 -->
    <Drawer v-model="drawerOpen" title="管控详情" :subtitle="detail?.sub_mchid" width="max-w-2xl">
      <div v-if="detail" class="space-y-4">
        <!-- 概览行 -->
        <div class="flex min-w-0 items-center gap-x-4 whitespace-nowrap bg-muted/40 px-4 py-3 text-sm">
          <Badge :variant="stateVariant(detail)" class="shrink-0">{{ stateText(detail) }}</Badge>
          <span class="min-w-0 truncate font-medium" :title="detail.merchant_name">{{ detail.merchant_name || '—' }}</span>
          <span class="shrink-0 text-xs text-muted-foreground">商户 <span class="text-foreground tabular-nums">{{ detail.uid }}</span></span>
          <span class="shrink-0 text-xs text-muted-foreground">子商户号 <span class="text-foreground tabular-nums">{{ detail.sub_mchid }}</span></span>
          <span v-if="detail.last_query_at" class="ml-auto shrink-0 text-xs text-muted-foreground">刷新于 {{ detail.last_query_at }}</span>
        </div>

        <!-- 未刷新提示 -->
        <p v-if="!detail.queried" class="border-l-2 border-muted bg-muted/30 px-4 py-2.5 text-xs text-muted-foreground">
          尚未刷新过该商户的管控状态，点击右下角「刷新」现查微信。
        </p>

        <!-- 刷新失败提示 -->
        <div v-else-if="detail.last_error" class="border-l-2 border-destructive bg-destructive/[0.05] px-4 py-2.5 text-xs text-muted-foreground">
          最近一次刷新失败：{{ detail.last_error }}
        </div>

        <!-- 被管控能力 -->
        <div v-if="detail.queried && (detail.limited_function_texts?.length || detail.other_limited_functions)">
          <div class="flex items-center gap-2 border-b border-border/60 pb-2 text-sm font-medium">
            <ShieldAlert class="size-4 text-destructive" />被管控能力
          </div>
          <div class="mt-2 flex flex-wrap gap-1.5">
            <span
              v-for="(t, i) in detail.limited_function_texts"
              :key="i"
              class="rounded bg-destructive/10 px-2 py-0.5 text-xs text-destructive"
            >{{ t }}</span>
            <span v-if="detail.other_limited_functions" class="rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground">{{ detail.other_limited_functions }}</span>
          </div>
        </div>

        <!-- 被管控原因 + 解脱路径列表 -->
        <div v-if="detail.recovery?.length" class="space-y-3">
          <div class="text-sm font-medium">被管控原因与解脱路径</div>
          <div v-for="(r, i) in detail.recovery" :key="i" class="bg-muted/40 px-4 py-3 text-sm">
            <div class="flex items-center gap-2">
              <span class="font-medium">{{ r.limitation_reason_type_text || '—' }}</span>
              <span v-if="r.limitation_action_type === 'LIMIT_ACTION_TYPE_DELAY_CONTROL'" class="rounded bg-warning/15 px-1.5 py-0.5 text-xs text-warning">延迟管控</span>
              <span v-if="r.limitation_case_id" class="ml-auto text-xs tabular-nums text-muted-foreground">单据 {{ r.limitation_case_id }}</span>
            </div>
            <p v-if="r.limitation_reason || r.limitation_reason_describe" class="mt-1.5 text-xs leading-relaxed text-muted-foreground">
              {{ r.limitation_reason_describe || r.limitation_reason }}
            </p>
            <dl class="mt-2 space-y-1 text-xs">
              <div class="flex gap-2">
                <dt class="w-20 shrink-0 text-muted-foreground">解脱路径</dt>
                <dd class="min-w-0 flex-1">{{ r.recover_way_text || '—' }}</dd>
              </div>
              <div v-if="r.recover_way_param" class="flex gap-2">
                <dt class="w-20 shrink-0 text-muted-foreground">路径参数</dt>
                <dd class="min-w-0 flex-1 tabular-nums">{{ r.recover_way_param }}</dd>
              </div>
              <div v-if="r.relate_limitations?.length || r.other_relate_limitations" class="flex gap-2">
                <dt class="w-20 shrink-0 text-muted-foreground">关联能力</dt>
                <dd class="min-w-0 flex-1">{{ [...(r.relate_limitations || []), r.other_relate_limitations].filter(Boolean).join('、') }}</dd>
              </div>
              <div v-if="r.limitation_start_date" class="flex gap-2">
                <dt class="w-20 shrink-0 text-muted-foreground">预计管控</dt>
                <dd class="min-w-0 flex-1">{{ r.limitation_start_date }}</dd>
              </div>
              <div v-if="r.recover_help_url" class="flex gap-2">
                <dt class="w-20 shrink-0 text-muted-foreground">帮助</dt>
                <dd class="min-w-0 flex-1">
                  <a :href="r.recover_help_url" target="_blank" class="inline-flex items-center gap-1 text-primary hover:text-primary/80">
                    解脱指引<ExternalLink class="size-3" />
                  </a>
                </dd>
              </div>
            </dl>
          </div>
        </div>

        <p v-else-if="detail.queried && detail.state === 'normal'" class="py-6 text-center text-sm text-success">该商户当前无被管控能力，状态正常。</p>

        <!-- 处置/管控流水时间线（风控第三段：微信主动推送的处置通知 + 合作伙伴订阅管控流水） -->
        <div class="space-y-2">
          <div class="flex items-center gap-2 border-b border-border/60 pb-2 text-sm font-medium">
            <History class="size-4 text-muted-foreground" />处置流水
            <span v-if="flows.length" class="text-xs text-muted-foreground tabular-nums">{{ flows.length }} 条</span>
          </div>
          <p v-if="flowsLoading" class="py-4 text-center text-xs text-muted-foreground">加载中…</p>
          <p v-else-if="!flows.length" class="py-4 text-center text-xs dim">暂无微信推送的处置/管控流水</p>
          <div v-else class="space-y-2">
            <div v-for="f in flows" :key="f.id" class="bg-muted/40 px-4 py-3 text-sm">
              <div class="flex items-center gap-2">
                <Badge :variant="flowStateVariant(f)" class="shrink-0">{{ flowTitle(f) }}</Badge>
                <span class="text-xs text-muted-foreground">{{ f.mechanism === 'violation' ? '处置通知' : '管控流水' }}</span>
                <span class="ml-auto shrink-0 text-xs tabular-nums text-muted-foreground">{{ f.business_time || f.punish_time || f.created_at }}</span>
              </div>
              <p v-if="f.punish_description || f.risk_description" class="mt-1.5 text-xs leading-relaxed text-muted-foreground">
                {{ [f.risk_type, f.punish_description || f.risk_description].filter(Boolean).join('：') }}
              </p>
              <dl class="mt-2 space-y-1 text-xs">
                <div v-if="f.business_code" class="flex gap-2">
                  <dt class="w-20 shrink-0 text-muted-foreground">管控单据</dt>
                  <dd class="min-w-0 flex-1 tabular-nums">{{ f.business_code }}</dd>
                </div>
                <div v-if="f.record_id" class="flex gap-2">
                  <dt class="w-20 shrink-0 text-muted-foreground">处置编号</dt>
                  <dd class="min-w-0 flex-1 tabular-nums">{{ f.record_id }}</dd>
                </div>
                <div v-if="f.event_type" class="flex gap-2">
                  <dt class="w-20 shrink-0 text-muted-foreground">事件</dt>
                  <dd class="min-w-0 flex-1">{{ f.event_type }}</dd>
                </div>
              </dl>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <Button v-if="detail" variant="outline" :disabled="refreshingIds.has(detail.enroll_id)" @click="refreshOne(detail)">
          <RefreshCw class="size-4" :class="{ 'animate-spin': refreshingIds.has(detail.enroll_id) }" />刷新管控状态
        </Button>
      </template>
    </Drawer>
  </div>
</template>
