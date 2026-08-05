<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Search, Eye, RefreshCw, MessageSquare, History, ImagePlus, X, Send } from 'lucide-vue-next'
import { Panel, Button, Badge, Drawer, Modal, Pagination } from '@/components/ui'
import {
  merchantListComplaints,
  merchantComplaintDetail,
  merchantSyncComplaint,
  merchantComplaintHistory,
  merchantReplyComplaint,
  merchantReplyComplaintImmediate,
  merchantCompleteComplaint,
  merchantUpdateComplaintRefund,
  merchantUploadComplaintImage,
  type ComplaintListItem,
  type ComplaintDetailResult,
  type NegotiationHistory,
  type ComplaintState,
} from '@/lib/api/merchantComplaint'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'

const toast = useToast()
const confirm = useConfirm()

// —— 列表（后端强制按登录商户隔离，只看/处理自己名下的投诉） ——
const list = ref<ComplaintListItem[]>([])
const total = ref(0)
const stats = reactive<Record<string, number>>({ PENDING: 0, PROCESSING: 0, PROCESSED: 0 })
const loading = ref(false)
const keyword = ref('')
const page = ref(1)
const pageSize = 20
const tab = ref<'' | ComplaintState>('')
const tabs = [
  { v: '', t: '全部' },
  { v: 'PENDING', t: '待处理' },
  { v: 'PROCESSING', t: '处理中' },
  { v: 'PROCESSED', t: '已完成' },
] as const

const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

// 状态徽章：待处理灰 / 处理中蓝 / 已完成绿。
function stateVariant(s: string): 'default' | 'success' | 'muted' {
  if (s === 'PROCESSED') return 'success'
  if (s === 'PROCESSING') return 'default'
  return 'muted'
}
// 问题类型中文。
function problemTypeText(t: string): string {
  return { REFUND: '要求退款', SERVICE_NOT_WORK: '服务问题', OTHERS: '其他' }[t] || t || '—'
}
function yuan(cents: number): string {
  return (cents / 100).toFixed(2)
}
// 协商历史操作方中文。
function operatorText(op: string): string {
  return { USER: '用户', MERCHANT: '商户', PLATFORM: '平台' }[op] || op || '—'
}

async function load() {
  loading.value = true
  try {
    const res = await merchantListComplaints({
      keyword: keyword.value.trim() || undefined,
      state: tab.value || undefined,
      page: page.value,
      pagesize: pageSize,
    })
    list.value = res.list
    total.value = res.total
    Object.assign(stats, { PENDING: 0, PROCESSING: 0, PROCESSED: 0 }, res.stats || {})
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

function search() {
  page.value = 1
  load()
}
function switchTab(v: '' | ComplaintState) {
  tab.value = v
  page.value = 1
  load()
}
function gotoPage(p: number) {
  page.value = p
  load()
}
// —— 详情抽屉 ——
const drawerOpen = ref(false)
const detail = ref<ComplaintDetailResult | null>(null)
const detailLoading = ref(false)
const syncing = ref(false)
const acting = ref(false)

// 协商历史
const historyList = ref<NegotiationHistory[]>([])
const historyLoading = ref(false)

async function openDetail(row: ComplaintListItem) {
  drawerOpen.value = true
  detail.value = null
  detailLoading.value = true
  resetReply()
  try {
    detail.value = await merchantComplaintDetail(row.id)
    loadHistory(row.id)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载详情失败')
  } finally {
    detailLoading.value = false
  }
}

async function loadHistory(id: number) {
  historyList.value = []
  historyLoading.value = true
  try {
    const res = await merchantComplaintHistory(id)
    historyList.value = res.data || []
  } catch {
    /* 历史现查微信失败（凭证未配齐等），留空不打扰 */
  } finally {
    historyLoading.value = false
  }
}

async function syncDetail() {
  if (!detail.value || syncing.value) return
  syncing.value = true
  try {
    detail.value = await merchantSyncComplaint(detail.value.complaint.id)
    loadHistory(detail.value.complaint.id)
    toast.success('已同步微信最新状态')
    load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '同步失败')
  } finally {
    syncing.value = false
  }
}

// —— 回复 + 传图 ——
const replyText = ref('')
const replyImages = ref<string[]>([])
const uploading = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

function resetReply() {
  replyText.value = ''
  replyImages.value = []
}
function pickImage() {
  fileInput.value?.click()
}
async function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (replyImages.value.length >= 4) {
    toast.error('最多上传 4 张图片')
    return
  }
  if (file.size > 2 * 1024 * 1024) {
    toast.error('图片不能超过 2M')
    return
  }
  uploading.value = true
  try {
    const res = await merchantUploadComplaintImage(file)
    replyImages.value.push(res.media_id)
    toast.success('图片已上传')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '上传失败')
  } finally {
    uploading.value = false
  }
}
function removeImage(i: number) {
  replyImages.value.splice(i, 1)
}

const canImmediate = computed(() => detail.value?.complaint.need_immediate_service && !detail.value?.complaint.in_platform_service)

async function submitReply() {
  if (!detail.value || acting.value) return
  const content = replyText.value.trim()
  if (!content) {
    toast.error('回复内容不能为空')
    return
  }
  acting.value = true
  try {
    const id = detail.value.complaint.id
    if (canImmediate.value) {
      await merchantReplyComplaintImmediate(id, { content, images: replyImages.value })
    } else {
      await merchantReplyComplaint(id, { content, images: replyImages.value })
    }
    toast.success('已回复用户')
    resetReply()
    detail.value = await merchantComplaintDetail(id)
    loadHistory(id)
    load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '回复失败')
  } finally {
    acting.value = false
  }
}

async function complete() {
  if (!detail.value || acting.value) return
  if (!(await confirm('确认该投诉已处理完成并反馈微信？', { title: '反馈处理完成' }))) return
  acting.value = true
  try {
    const id = detail.value.complaint.id
    await merchantCompleteComplaint(id)
    toast.success('已反馈处理完成')
    detail.value = await merchantComplaintDetail(id)
    loadHistory(id)
    load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  } finally {
    acting.value = false
  }
}

// —— 退款审批 ——
const refundOpen = ref(false)
const refundForm = reactive<{ action: 'APPROVE' | 'REJECT'; launch_refund_day: number; reject_reason: string; remark: string }>({
  action: 'APPROVE',
  launch_refund_day: 1,
  reject_reason: '',
  remark: '',
})
function openRefund() {
  refundForm.action = 'APPROVE'
  refundForm.launch_refund_day = 1
  refundForm.reject_reason = ''
  refundForm.remark = ''
  refundOpen.value = true
}
async function submitRefund() {
  if (!detail.value || acting.value) return
  if (refundForm.action === 'REJECT' && !refundForm.reject_reason.trim()) {
    toast.error('驳回退款须填写驳回原因')
    return
  }
  acting.value = true
  try {
    const id = detail.value.complaint.id
    await merchantUpdateComplaintRefund(id, {
      action: refundForm.action,
      launch_refund_day: refundForm.action === 'APPROVE' ? refundForm.launch_refund_day : undefined,
      reject_reason: refundForm.action === 'REJECT' ? refundForm.reject_reason.trim() : undefined,
      remark: refundForm.remark.trim() || undefined,
    })
    toast.success('退款审批已提交')
    refundOpen.value = false
    detail.value = await merchantComplaintDetail(id)
    loadHistory(id)
    load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '提交失败')
  } finally {
    acting.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 概览 -->
    <Panel
      title="消费者投诉"
      subtitle="微信支付消费者投诉2.0：处理买家针对你名下子商户号发起的投诉。请及时回复用户、审批退款并反馈处理完成，避免影响商户信用与收款权限"
    >
      <div class="flex flex-wrap items-center gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">待处理</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-warning">{{ stats.PENDING || 0 }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">处理中</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-primary">{{ stats.PROCESSING || 0 }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已完成</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ stats.PROCESSED || 0 }}</div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="我的投诉工单" :subtitle="`共 ${total} 单`">
      <template #title-extra>
        <div class="ml-4 flex items-center gap-1">
          <button
            v-for="f in tabs"
            :key="f.v"
            class="rounded-full px-3 py-1 text-[13px] transition-colors"
            :class="tab === f.v ? 'bg-primary/10 font-medium text-primary' : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'"
            @click="switchTab(f.v)"
          >
            {{ f.t }}
          </button>
        </div>
      </template>

      <template #actions>
        <div class="relative">
          <Search class="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <input
            v-model="keyword"
            class="field-input !pl-9 w-60"
            placeholder="投诉单号 / 子商户号"
            @keyup.enter="search"
          />
        </div>
      </template>

      <div class="overflow-x-auto">
        <table class="tbl w-full">
          <thead>
            <tr>
              <th class="w-[18%]">投诉单号</th>
              <th class="w-[14%]">子商户号</th>
              <th class="w-[11%]">问题类型</th>
              <th class="w-[24%]">投诉内容</th>
              <th class="w-[10%] col-right">申诉退款</th>
              <th class="w-[9%]">状态</th>
              <th class="w-[12%]">投诉时间</th>
              <th class="w-[8%] col-center">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in list" :key="row.id">
              <td class="tabular-nums">
                <div class="truncate" :title="row.complaint_id">{{ row.complaint_id }}</div>
                <div v-if="row.user_complaint_times > 1" class="mt-0.5 text-xs text-destructive">用户第 {{ row.user_complaint_times }} 次投诉</div>
              </td>
              <td class="tabular-nums">
                <div class="truncate text-muted-foreground" :title="row.complainted_mchid">{{ row.complainted_mchid || '—' }}</div>
              </td>
              <td>{{ problemTypeText(row.problem_type) }}</td>
              <td>
                <div class="line-clamp-2 text-xs leading-relaxed text-muted-foreground" :title="row.complaint_detail">
                  {{ row.complaint_detail || row.problem_description || '—' }}
                </div>
              </td>
              <td class="col-right tabular-nums">
                <span v-if="row.apply_refund_amount"><span class="dim">¥</span>{{ yuan(row.apply_refund_amount) }}</span>
                <span v-else class="dim">—</span>
              </td>
              <td><Badge :variant="stateVariant(row.complaint_state)">{{ row.state_text }}</Badge></td>
              <td class="text-xs tabular-nums text-muted-foreground">{{ row.complaint_time || '—' }}</td>
              <td class="col-center">
                <button
                  class="inline-flex size-7 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                  title="查看 / 处理"
                  @click="openDetail(row)"
                >
                  <Eye class="size-4" />
                </button>
              </td>
            </tr>
            <tr v-if="!loading && !list.length">
              <td colspan="8" class="py-10 text-center dim">暂无投诉工单</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="pageCount > 1" class="mt-3 flex justify-end">
        <Pagination :page="page" :page-count="pageCount" :total="total" :page-size="pageSize" @change="gotoPage" />
      </div>
    </Panel>
    <!-- 详情抽屉 -->
    <Drawer v-model="drawerOpen" title="投诉详情" :subtitle="detail?.complaint.complaint_id" width="max-w-2xl">
      <p v-if="detailLoading" class="py-10 text-center text-sm text-muted-foreground">加载中…</p>
      <div v-else-if="detail" class="space-y-4">
        <!-- 概览行 -->
        <div class="flex min-w-0 flex-wrap items-center gap-x-4 gap-y-1 bg-muted/40 px-4 py-3 text-sm">
          <Badge :variant="stateVariant(detail.complaint.complaint_state)" class="shrink-0">{{ detail.state_text }}</Badge>
          <span class="shrink-0 text-xs text-muted-foreground">子商户号 <span class="text-foreground tabular-nums">{{ detail.complaint.complainted_mchid || '—' }}</span></span>
          <span v-if="canImmediate" class="shrink-0 rounded bg-destructive/10 px-1.5 py-0.5 text-xs text-destructive">需即时服务</span>
          <span v-if="detail.complaint.in_platform_service" class="shrink-0 rounded bg-warning/15 px-1.5 py-0.5 text-xs text-warning">平台介入中</span>
        </div>

        <!-- 投诉内容 -->
        <div>
          <div class="border-b border-border/60 pb-2 text-sm font-medium">投诉内容</div>
          <dl class="mt-2 space-y-1.5 text-xs">
            <div class="flex gap-2">
              <dt class="w-20 shrink-0 text-muted-foreground">问题类型</dt>
              <dd class="min-w-0 flex-1">{{ problemTypeText(detail.complaint.problem_type) }}</dd>
            </div>
            <div class="flex gap-2">
              <dt class="w-20 shrink-0 text-muted-foreground">投诉描述</dt>
              <dd class="min-w-0 flex-1 leading-relaxed">{{ detail.complaint.complaint_detail || detail.complaint.problem_description || '—' }}</dd>
            </div>
            <div class="flex gap-2">
              <dt class="w-20 shrink-0 text-muted-foreground">申诉退款</dt>
              <dd class="min-w-0 flex-1 tabular-nums">
                <span v-if="detail.complaint.apply_refund_amount"><span class="dim">¥</span>{{ yuan(detail.complaint.apply_refund_amount) }}</span>
                <span v-else class="dim">无</span>
                <span v-if="detail.complaint.complaint_full_refunded" class="ml-2 text-success">已全额退款</span>
              </dd>
            </div>
            <div class="flex gap-2">
              <dt class="w-20 shrink-0 text-muted-foreground">投诉人</dt>
              <dd class="min-w-0 flex-1 tabular-nums">{{ detail.complaint.payer_phone || '—' }}<span v-if="detail.complaint.payer_openid" class="ml-2 truncate text-muted-foreground">{{ detail.complaint.payer_openid }}</span></dd>
            </div>
            <div class="flex gap-2">
              <dt class="w-20 shrink-0 text-muted-foreground">投诉时间</dt>
              <dd class="min-w-0 flex-1 tabular-nums">{{ detail.complaint.complaint_time || '—' }}</dd>
            </div>
          </dl>
        </div>

        <!-- 关联订单 -->
        <div v-if="detail.orders?.length">
          <div class="border-b border-border/60 pb-2 text-sm font-medium">关联订单</div>
          <div class="mt-2 space-y-1.5">
            <div v-for="(o, i) in detail.orders" :key="i" class="flex items-center gap-3 bg-muted/40 px-3 py-2 text-xs">
              <span class="min-w-0 flex-1 truncate tabular-nums" :title="o.transaction_id">{{ o.transaction_id || o.out_trade_no || '—' }}</span>
              <span class="shrink-0 tabular-nums"><span class="dim">¥</span>{{ yuan(o.amount) }}</span>
              <span class="shrink-0 text-muted-foreground">{{ o.state || '—' }}</span>
            </div>
          </div>
        </div>

        <!-- 协商历史时间线 -->
        <div>
          <div class="flex items-center gap-2 border-b border-border/60 pb-2 text-sm font-medium">
            <History class="size-4 text-muted-foreground" />协商历史
            <span v-if="historyList.length" class="text-xs text-muted-foreground tabular-nums">{{ historyList.length }} 条</span>
          </div>
          <p v-if="historyLoading" class="py-4 text-center text-xs text-muted-foreground">加载中…</p>
          <p v-else-if="!historyList.length" class="py-4 text-center text-xs dim">暂无协商历史</p>
          <div v-else class="mt-2 space-y-2">
            <div v-for="h in historyList" :key="h.log_id" class="bg-muted/40 px-4 py-2.5 text-sm">
              <div class="flex items-center gap-2">
                <span class="font-medium">{{ operatorText(h.operator) }}</span>
                <span class="text-xs text-muted-foreground">{{ h.operate_type }}</span>
                <span class="ml-auto shrink-0 text-xs tabular-nums text-muted-foreground">{{ h.operate_time }}</span>
              </div>
              <p v-if="h.operate_details" class="mt-1 text-xs leading-relaxed text-muted-foreground">{{ h.operate_details }}</p>
              <div v-if="h.image_list?.length" class="mt-1.5 flex flex-wrap gap-1.5">
                <img v-for="(img, ii) in h.image_list" :key="ii" :src="img" class="size-12 rounded object-cover" />
              </div>
            </div>
          </div>
        </div>

        <!-- 回复框（未完成态可回复处理） -->
        <div v-if="detail.complaint.complaint_state !== 'PROCESSED'">
          <div class="flex items-center gap-2 border-b border-border/60 pb-2 text-sm font-medium">
            <MessageSquare class="size-4 text-muted-foreground" />回复用户
            <span v-if="canImmediate" class="text-xs text-destructive">（即时服务通道）</span>
          </div>
          <textarea
            v-model="replyText"
            rows="3"
            maxlength="200"
            class="field-input mt-2 w-full resize-none"
            placeholder="向用户说明处理情况，≤200 字符"
          />
          <div class="mt-1.5 flex items-center gap-2">
            <div v-for="(m, i) in replyImages" :key="m" class="relative">
              <span class="inline-flex items-center gap-1 rounded bg-muted px-2 py-1 text-xs tabular-nums text-muted-foreground">
                图{{ i + 1 }}
                <button class="text-muted-foreground hover:text-destructive" @click="removeImage(i)"><X class="size-3" /></button>
              </span>
            </div>
            <button
              v-if="replyImages.length < 4"
              class="inline-flex items-center gap-1 rounded border border-dashed border-border px-2 py-1 text-xs text-muted-foreground transition-colors hover:border-primary hover:text-primary disabled:opacity-50"
              :disabled="uploading"
              @click="pickImage"
            >
              <ImagePlus class="size-3.5" />{{ uploading ? '上传中…' : '传图' }}
            </button>
            <input ref="fileInput" type="file" accept="image/*" class="hidden" @change="onFileChange" />
            <span class="ml-auto text-xs text-muted-foreground tabular-nums">{{ replyText.length }}/200</span>
          </div>
        </div>
        <p v-else class="py-2 text-center text-xs text-success">该投诉已处理完成。</p>
      </div>

      <template #footer>
        <div v-if="detail" class="flex w-full items-center gap-2">
          <Button variant="outline" :disabled="syncing" @click="syncDetail">
            <RefreshCw class="size-4" :class="{ 'animate-spin': syncing }" />同步
          </Button>
          <div class="ml-auto flex items-center gap-2">
            <template v-if="detail.complaint.complaint_state !== 'PROCESSED'">
              <Button v-if="detail.complaint.apply_refund_amount" variant="outline" :disabled="acting" @click="openRefund">退款审批</Button>
              <Button variant="outline" :disabled="acting" @click="complete">反馈完成</Button>
              <Button :disabled="acting || uploading" @click="submitReply"><Send class="size-4" />回复用户</Button>
            </template>
          </div>
        </div>
      </template>
    </Drawer>

    <!-- 退款审批弹窗 -->
    <Modal v-model="refundOpen" title="退款审批">
      <div class="space-y-3">
        <div class="flex gap-2">
          <button
            class="flex-1 rounded-lg border px-3 py-2 text-sm transition-colors"
            :class="refundForm.action === 'APPROVE' ? 'border-primary bg-primary/[0.06] text-primary' : 'border-border text-muted-foreground hover:bg-muted/50'"
            @click="refundForm.action = 'APPROVE'"
          >同意退款</button>
          <button
            class="flex-1 rounded-lg border px-3 py-2 text-sm transition-colors"
            :class="refundForm.action === 'REJECT' ? 'border-destructive bg-destructive/[0.06] text-destructive' : 'border-border text-muted-foreground hover:bg-muted/50'"
            @click="refundForm.action = 'REJECT'"
          >驳回退款</button>
        </div>
        <div v-if="refundForm.action === 'APPROVE'">
          <label class="mb-1 block text-xs text-muted-foreground">承诺退款天数</label>
          <input v-model.number="refundForm.launch_refund_day" type="number" min="1" class="field-input w-full" />
        </div>
        <div v-else>
          <label class="mb-1 block text-xs text-muted-foreground">驳回原因（必填）</label>
          <textarea v-model="refundForm.reject_reason" rows="2" class="field-input w-full resize-none" placeholder="向用户说明驳回退款的原因" />
        </div>
        <div>
          <label class="mb-1 block text-xs text-muted-foreground">备注（可选）</label>
          <input v-model="refundForm.remark" class="field-input w-full" />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" :disabled="acting" @click="refundOpen = false">取消</Button>
        <Button :disabled="acting" @click="submitRefund">提交审批</Button>
      </template>
    </Modal>
  </div>
</template>



