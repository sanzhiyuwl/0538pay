<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Eye, Plus } from 'lucide-vue-next'
import { Panel, Button, Badge, Drawer, Pagination } from '@/components/ui'
import {
  fetchAgents,
  fetchEnrolls,
  getEnroll,
  createEnroll,
  submitEnroll,
  syncEnroll,
  type Agent,
  type Enroll,
} from '@/lib/api/console'
import { ApiError } from '@/lib/api/client'

const enrolls = ref<Enroll[]>([])
const total = ref(0)
const loading = ref(false)
const agents = ref<Agent[]>([])

const filters = reactive({ keyword: '', status: '', agentId: '', source: '' })
const page = ref(1)
const pageSize = 20
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

// 本地状态机文案（对齐 docs-代理进件/01 第五节）
const statusMeta: Record<string, { label: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'muted' }> = {
  pending_pay: { label: '待支付', variant: 'warning' },
  paid: { label: '已支付待完善', variant: 'default' },
  submitted: { label: '审核中', variant: 'default' },
  finished: { label: '已完成', variant: 'success' },
  rejected: { label: '已驳回', variant: 'destructive' },
  closed: { label: '已关单', variant: 'muted' },
  refunded: { label: '已退款', variant: 'muted' },
}
const sourceText: Record<number, string> = { 1: '平台代填', 2: '代理代填', 3: '客户自助' }
const pathText: Record<number, string> = { 1: '预购名额', 2: '商户自付' }
const agentName = (id: number) => (id === 0 ? '平台' : agents.value.find((a) => a.id === id)?.name ?? `#${id}`)

async function load() {
  loading.value = true
  try {
    const { list, total: t } = await fetchEnrolls({
      keyword: filters.keyword || undefined,
      status: filters.status || undefined,
      agent_id: filters.agentId ? Number(filters.agentId) : undefined,
      source: filters.source ? Number(filters.source) : undefined,
      page: page.value,
      pageSize,
    })
    enrolls.value = list
    total.value = t
  } catch (e) {
    alert(e instanceof ApiError ? e.message : '加载进件单失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    const { list } = await fetchAgents({ pageSize: 500 })
    agents.value = list
  } catch {
    // ignore
  }
  await load()
})

function search() {
  page.value = 1
  load()
}
function resetFilters() {
  filters.keyword = ''
  filters.status = ''
  filters.agentId = ''
  filters.source = ''
  search()
}
function go(p: number) {
  page.value = p
  load()
}

// ===== 详情抽屉 =====
const drawer = ref(false)
const detail = ref<Enroll | null>(null)
const acting = ref(false)
async function openDetail(e: Enroll) {
  detail.value = e
  drawer.value = true
  try {
    detail.value = await getEnroll(e.id)
  } catch {
    // 用列表行数据兜底
  }
}

// ===== 建单抽屉（付费前置：建单即下开户费收款）=====
const createDrawer = ref(false)
const creating = ref(false)
const form = reactive({ merchantName: '', contactPhone: '', path: 2, agentId: '', plugin: 'mock' })
function openCreate() {
  Object.assign(form, { merchantName: '', contactPhone: '', path: 2, agentId: '', plugin: 'mock' })
  createDrawer.value = true
}
async function doCreate() {
  if (!form.merchantName.trim()) {
    alert('请填写商户名称')
    return
  }
  creating.value = true
  try {
    const r = await createEnroll({
      merchant_name: form.merchantName.trim(),
      contact_phone: form.contactPhone.trim() || undefined,
      path: form.path,
      agent_id: form.agentId ? Number(form.agentId) : undefined,
      plugin: form.plugin,
    })
    createDrawer.value = false
    await load()
    if (r.pay?.qrcode || r.pay?.pay_url) {
      alert(`进件单已建，待支付开户费 ¥${r.pay.money}。收款单号：${r.pay.trade_no}\n付款链接：${r.pay.pay_url || r.pay.qrcode}`)
    } else {
      alert('进件单已建（无需收费，已放行填料）')
    }
  } catch (e) {
    alert(e instanceof ApiError ? e.message : '建单失败')
  } finally {
    creating.value = false
  }
}

// 已支付待完善 / 被驳回可重提 → 提交微信审核
const canSubmit = computed(() => detail.value?.status === 'paid' || detail.value?.status === 'rejected')
// 审核中 → 可主动查微信最新状态
const canSync = computed(() => detail.value?.status === 'submitted')

async function doSubmit() {
  if (!detail.value) return
  acting.value = true
  try {
    detail.value = await submitEnroll(detail.value.id)
    await load()
  } catch (e) {
    alert(e instanceof ApiError ? e.message : '提交微信失败')
  } finally {
    acting.value = false
  }
}
async function doSync() {
  if (!detail.value) return
  acting.value = true
  try {
    detail.value = await syncEnroll(detail.value.id)
    await load()
  } catch (e) {
    alert(e instanceof ApiError ? e.message : '查询微信状态失败')
  } finally {
    acting.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="进件申请" :subtitle="`共 ${total} 笔进件单`">
      <template #actions>
        <Button size="sm" @click="openCreate"><Plus />建进件单</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">搜索</label>
          <input
            v-model="filters.keyword"
            placeholder="商户名 / 进件单号"
            class="field-input w-48"
            @keyup.enter="search"
          />
        </div>
        <div class="filter-item">
          <label class="filter-label">状态</label>
          <select v-model="filters.status" class="field-input w-32">
            <option value="">全部</option>
            <option v-for="(m, k) in statusMeta" :key="k" :value="k">{{ m.label }}</option>
          </select>
        </div>
        <div class="filter-item">
          <label class="filter-label">代理</label>
          <select v-model="filters.agentId" class="field-input w-40">
            <option value="">全部</option>
            <option v-for="a in agents" :key="a.id" :value="a.id">{{ a.name }}</option>
          </select>
        </div>
        <div class="filter-item">
          <label class="filter-label">来源</label>
          <select v-model="filters.source" class="field-input w-28">
            <option value="">全部</option>
            <option value="1">平台代填</option>
            <option value="2">代理代填</option>
            <option value="3">客户自助</option>
          </select>
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="search">搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters">重置</Button>
        </div>
      </div>
    </Panel>

    <Panel title="进件单列表" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[15%]">进件单号</th>
              <th class="w-[18%]">商户名称</th>
              <th class="w-[12%]">归属代理</th>
              <th class="col-center w-[9%]">来源</th>
              <th class="col-center w-[9%]">路径</th>
              <th class="col-center w-[12%]">状态</th>
              <th class="w-[13%]">sub_mchid</th>
              <th class="col-center w-[8%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="e in enrolls" :key="e.id">
              <td class="truncate font-medium tabular-nums">{{ e.enroll_no }}</td>
              <td class="truncate">{{ e.merchant_name }}</td>
              <td class="truncate dim">{{ agentName(e.agent_id) }}</td>
              <td class="col-center text-xs">{{ sourceText[e.source] ?? '—' }}</td>
              <td class="col-center text-xs">{{ pathText[e.path] ?? '—' }}</td>
              <td class="col-center">
                <Badge :variant="statusMeta[e.status]?.variant ?? 'muted'">
                  {{ statusMeta[e.status]?.label ?? e.status }}
                </Badge>
              </td>
              <td class="truncate tabular-nums dim">{{ e.wx_sub_mchid || '—' }}</td>
              <td class="col-center">
                <Button variant="ghost" size="sm" @click="openDetail(e)"><Eye class="size-4" /></Button>
              </td>
            </tr>
            <tr v-if="loading">
              <td colspan="8" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!enrolls.length">
              <td colspan="8" class="py-10 text-center dim">暂无进件单</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="page" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>

    <!-- 详情抽屉 -->
    <Drawer v-model="drawer" title="进件单详情">
      <div v-if="detail" class="space-y-2 text-sm">
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">进件单号</span><span class="tabular-nums">{{ detail.enroll_no }}</span>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">商户名称</span><span>{{ detail.merchant_name }}</span>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">主体类型</span><span>{{ detail.subject_type || '—' }}</span>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">联系手机</span><span class="tabular-nums">{{ detail.contact_phone || '—' }}</span>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">归属代理</span><span>{{ agentName(detail.agent_id) }}</span>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">发起来源</span><span>{{ sourceText[detail.source] ?? '—' }}</span>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">资金路径</span><span>{{ pathText[detail.path] ?? '—' }}</span>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">状态</span>
          <Badge :variant="statusMeta[detail.status]?.variant ?? 'muted'">
            {{ statusMeta[detail.status]?.label ?? detail.status }}
          </Badge>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">收款单号</span><span class="tabular-nums">{{ detail.pay_order_no || '—' }}</span>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">微信申请单号</span><span class="tabular-nums">{{ detail.wx_applyment_id || '—' }}</span>
        </div>
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="dim">sub_mchid</span><span class="tabular-nums">{{ detail.wx_sub_mchid || '—' }}</span>
        </div>
        <div v-if="detail.reject_reason" class="py-2">
          <div class="dim mb-1">驳回原因</div>
          <div class="bg-destructive/[0.06] px-3 py-2 text-destructive">{{ detail.reject_reason }}</div>
        </div>
      </div>
      <p class="mt-4 text-xs text-muted-foreground">
        提交微信调用服务商 APIv3 进件接口（applyment4sub）；审核中可点“查状态”拉取微信最新进度。
        退款（自动/手动）与 sub_mchid 硬锁在后续批次接入。
      </p>
      <template #footer>
        <Button variant="outline" @click="drawer = false">关闭</Button>
        <Button v-if="canSync" :disabled="acting" variant="outline" @click="doSync">
          {{ acting ? '查询中…' : '查状态' }}
        </Button>
        <Button v-if="canSubmit" :disabled="acting" @click="doSubmit">
          {{ acting ? '提交中…' : detail?.status === 'rejected' ? '重新提交' : '提交微信' }}
        </Button>
      </template>
    </Drawer>

    <!-- 建单抽屉（付费前置）-->
    <Drawer v-model="createDrawer" title="建进件单" subtitle="第一步只填基础信息，建单即收开户费，付款成功后放行填全套资料">
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">商户名称<span class="text-destructive">*</span></label>
          <input v-model="form.merchantName" placeholder="特约商户主体名称" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">联系手机</label>
          <input v-model="form.contactPhone" placeholder="用于进度查询匹配" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">归属代理</label>
          <select v-model="form.agentId" class="field-input flex-1">
            <option value="">平台自己（无代理）</option>
            <option v-for="a in agents" :key="a.id" :value="a.id">{{ a.name }}</option>
          </select>
        </div>
        <div class="row-field">
          <label class="lbl">资金路径</label>
          <select v-model.number="form.path" class="field-input flex-1">
            <option :value="2">商户自付（分账）</option>
            <option :value="1">预购名额（全额归代理）</option>
          </select>
        </div>
        <div class="row-field">
          <label class="lbl">收款方式</label>
          <input v-model="form.plugin" placeholder="收开户费的渠道 plugin，如 mock / alipay / wxpay" class="field-input flex-1" />
        </div>
        <p class="text-[11px] text-muted-foreground">
          开户零售价读“进件设置”，程序不硬编码。路径一若后台配置“客户免付开户费”，建单直接放行不收款。
        </p>
      </div>
      <template #footer>
        <Button variant="outline" @click="createDrawer = false">取消</Button>
        <Button :disabled="creating" @click="doCreate">{{ creating ? '创建中…' : '创建并收款' }}</Button>
      </template>
    </Drawer>
  </div>
</template>
