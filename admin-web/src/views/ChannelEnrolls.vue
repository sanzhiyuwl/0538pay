<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { Search, Eye, CheckCircle2, XCircle, RefreshCw, ShieldCheck, QrCode, ArrowUp, ArrowDown, Copy } from 'lucide-vue-next'
import QRCodeLib from 'qrcode'
import { Panel, Button, Badge, Drawer, Pagination } from '@/components/ui'
import {
  adminListChannelEnrolls,
  adminGetChannelEnroll,
  adminApproveChannelEnroll,
  adminRejectChannelEnroll,
  adminSyncChannelEnroll,
  type ChannelEnrollView,
  type ChannelEnrollDetail,
} from '@/lib/api/channelEnroll'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const list = ref<ChannelEnrollView[]>([])
const total = ref(0)
const loading = ref(false)
const query = reactive({ keyword: '', status: '', page: 1, pagesize: 20, sort: 'id_desc' })

// 按编号(ID)排序切换：点表头在 降序↔升序 间切换，回第一页重拉。
function toggleSort() {
  query.sort = query.sort === 'id_desc' ? 'id_asc' : 'id_desc'
  query.page = 1
  load()
}

const statusVariant: Record<string, 'default' | 'success' | 'warning' | 'destructive' | 'muted'> = {
  draft: 'muted',
  pending: 'warning',
  submitted: 'warning',
  approved: 'success',
  rejected: 'destructive',
}

const statusFilters = [
  { v: '', t: '全部' },
  { v: 'submitted', t: '处理中' },
  { v: 'approved', t: '已开通' },
  { v: 'rejected', t: '已驳回' },
  { v: 'draft', t: '草稿' },
  { v: 'pending', t: '待处理(历史)' },
]

// 概览统计：处理中 / 待签约 / 已开通 / 已驳回 / 全部总数（各取 total，轻量并行）。
// 待签约是微信侧细状态（wx_state=APPLYMENT_STATE_TO_BE_SIGNED），从「处理中(submitted)」里进一步细分出来。
const stats = reactive({ total: 0, submitted: 0, toBeSigned: 0, approved: 0, rejected: 0 })
async function loadStats() {
  try {
    const [all, sub, signing, appr, rej] = await Promise.all([
      adminListChannelEnrolls({ page: 1, pagesize: 1 }),
      adminListChannelEnrolls({ page: 1, pagesize: 1, status: 'submitted' }),
      adminListChannelEnrolls({ page: 1, pagesize: 1, wx_state: 'APPLYMENT_STATE_TO_BE_SIGNED' }),
      adminListChannelEnrolls({ page: 1, pagesize: 1, status: 'approved' }),
      adminListChannelEnrolls({ page: 1, pagesize: 1, status: 'rejected' }),
    ])
    stats.total = all.total
    stats.submitted = sub.total
    stats.toBeSigned = signing.total
    stats.approved = appr.total
    stats.rejected = rej.total
  } catch {
    /* 统计失败不打扰主列表 */
  }
}

async function load() {
  loading.value = true
  try {
    const res = await adminListChannelEnrolls({ ...query })
    list.value = res.list
    total.value = res.total
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}
onMounted(() => {
  load()
  loadStats()
})

function setStatus(v: string) {
  query.status = v
  query.page = 1
  load()
}
function search() {
  query.page = 1
  load()
}

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / query.pagesize)))
function goPage(p: number) {
  const np = Math.min(Math.max(1, p), totalPages.value)
  if (np === query.page) return
  query.page = np
  load()
}

// —— 详情 / 手动兜底 ——
const drawerOpen = ref(false)
const detail = ref<ChannelEnrollDetail | null>(null)
const busy = ref(false)
const material = computed(() => detail.value?.material)

// 待签约二维码：内嵌显示，超管扫码即签约。
// 微信有两种 sign_url 形态：
//   ① 图片直链：`https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=...` —— 本身就是二维码 PNG，直接当图显示，
//      切勿再 QR 编码这个 URL（那会把「图片地址」编成一个扫不出真意图的二维码）。
//   ② 普通签约页链接：用 qrcode 库把 URL 编成二维码 PNG。
const signQrImageURL = ref('') // 直链图片形态
const signQrDataURL = ref('')  // 前端编码 dataURL 形态
watch(
  () => detail.value?.sign_url,
  async (url) => {
    signQrImageURL.value = ''
    signQrDataURL.value = ''
    if (!url) return
    if (/mp\.weixin\.qq\.com\/cgi-bin\/showqrcode/i.test(url)) {
      signQrImageURL.value = url
      return
    }
    try {
      signQrDataURL.value = await QRCodeLib.toDataURL(url, { width: 176, margin: 1 })
    } catch {
      /* 生成失败则回退到文字链接 */
    }
  },
  { immediate: true },
)
async function copySignURL() {
  if (!detail.value?.sign_url) return
  try {
    await navigator.clipboard.writeText(detail.value.sign_url)
    toast.success('签约链接已复制')
  } catch {
    toast.error('复制失败，请手动选择链接')
  }
}

async function openDetail(id: number) {
  try {
    detail.value = await adminGetChannelEnroll(id)
    drawerOpen.value = true
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载详情失败')
  }
}

// 刷新微信状态（审核中→开通/驳回，全自动主链）。
async function doSync() {
  if (!detail.value) return
  busy.value = true
  try {
    const r = await adminSyncChannelEnroll(detail.value.id)
    if (r.status === 'approved') toast.success(`微信已开通，子商户号 ${r.sub_mchid}`)
    else if (r.status === 'rejected') toast.error(`微信驳回：${r.reject_reason || '见详情'}`)
    else toast.info(`当前微信状态：${r.wx_state_text || '处理中'}`)
    detail.value = await adminGetChannelEnroll(detail.value.id)
    await load()
    loadStats()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '刷新失败')
  } finally {
    busy.value = false
  }
}

// 手动交付（兜底）：微信已开通但自动同步异常、或历史 pending 单人工补号。
async function doApprove() {
  if (!detail.value) return
  const subMchID = window.prompt('手动交付：输入微信开出的子商户号（将写入该商户子通道，收款直清到此号）：', detail.value.sub_mchid || '')
  if (subMchID == null) return
  if (!subMchID.trim()) {
    toast.error('子商户号不能为空')
    return
  }
  busy.value = true
  try {
    await adminApproveChannelEnroll(detail.value.id, subMchID.trim())
    toast.success('已手动交付，为该商户开通子通道。请确认其所属用户组已将此支付方式设为「子通道(-2)」')
    drawerOpen.value = false
    await load()
    loadStats()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '交付失败')
  } finally {
    busy.value = false
  }
}

async function doReject() {
  if (!detail.value) return
  const reason = window.prompt('请输入驳回原因（商户可据此修改重提）：', '')
  if (reason == null) return
  if (!reason.trim()) {
    toast.error('驳回原因不能为空')
    return
  }
  busy.value = true
  try {
    await adminRejectChannelEnroll(detail.value.id, reason.trim())
    toast.success('已驳回')
    drawerOpen.value = false
    await load()
    loadStats()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '驳回失败')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 概览 -->
    <Panel
      title="通道进件"
      subtitle="商户在服务商通道下的进件（全自动 applyment4sub 直提交微信）：微信审核通过后系统自动为商户开通子通道并回填子商户号（只走商户进件不走二清）"
    >
      <div class="flex flex-wrap items-center gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">进件总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.total }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">处理中</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-warning">{{ stats.submitted }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">待签约</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-warning">{{ stats.toBeSigned }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已开通</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ stats.approved }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已驳回</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-destructive">{{ stats.rejected }}</div>
        </div>
      </div>
    </Panel>

    <!-- 列表：与 OpLogs.vue 同款结构（不再 flush/自定义 padding，标准 Panel + tbl w-full + Pagination） -->
    <Panel title="进件申请" :subtitle="`共 ${total} 条`">
      <!-- 状态筛选 tab 放进标题行 -->
      <template #title-extra>
        <div class="ml-4 flex items-center gap-1">
          <button
            v-for="f in statusFilters"
            :key="f.v"
            class="rounded-full px-3 py-1 text-[13px] transition-colors"
            :class="query.status === f.v
              ? 'bg-primary/10 font-medium text-primary'
              : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'"
            @click="setStatus(f.v)"
          >
            {{ f.t }}
          </button>
        </div>
      </template>

      <template #actions>
        <div class="flex items-center gap-2">
          <div class="relative">
            <Search class="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <input
              v-model="query.keyword"
              class="field-input !pl-9 w-56"
              placeholder="商户名 / 单号 / 手机"
              @keyup.enter="search"
            />
          </div>
          <Button size="sm" variant="outline" @click="search"><Search class="size-3.5" />查询</Button>
        </div>
      </template>

      <div class="overflow-x-auto">
        <table class="tbl w-full">
          <thead>
            <tr>
              <th class="w-[6%]">
                <button class="inline-flex select-none items-center gap-1 hover:text-foreground" @click="toggleSort">
                  ID
                  <ArrowUp v-if="query.sort === 'id_asc'" class="size-3" />
                  <ArrowDown v-else class="size-3" />
                </button>
              </th>
              <th class="w-[12%]">进件单号</th>
              <th class="w-[7%]">商户</th>
              <th class="w-[14%]">服务商通道</th>
              <th class="w-[19%]">主体名称</th>
              <th class="w-[8%]">状态</th>
              <th class="w-[10%]">微信状态</th>
              <th class="w-[14%]">子商户号</th>
              <th class="w-[10%] col-center">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="e in list" :key="e.id">
              <td class="tabular-nums dim">{{ e.id }}</td>
              <td class="tabular-nums">{{ e.enroll_no }}</td>
              <td class="tabular-nums">{{ e.uid }}</td>
              <td class="truncate" :title="e.channel_name">{{ e.channel_name || '—' }}</td>
              <td class="truncate" :title="e.merchant_name">{{ e.merchant_name || '—' }}</td>
              <td><Badge :variant="statusVariant[e.status] || 'muted'">{{ e.status_text }}</Badge></td>
              <td class="text-xs text-muted-foreground">{{ e.wx_state_text || '—' }}</td>
              <td class="tabular-nums">{{ e.sub_mchid || '—' }}</td>
              <td class="col-center">
                <button
                  class="inline-flex size-7 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                  title="查看详情"
                  @click="openDetail(e.id)"
                >
                  <Eye class="size-4" />
                </button>
              </td>
            </tr>
            <tr v-if="!loading && !list.length">
              <td colspan="9" class="py-10 text-center dim">暂无进件申请</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="totalPages > 1" class="mt-4 border-t border-border/60 pt-4">
        <Pagination
          :page="query.page"
          :page-count="totalPages"
          :total="total"
          :page-size="query.pagesize"
          @change="goPage"
        />
      </div>
    </Panel>

    <!-- 详情抽屉 -->
    <Drawer v-model="drawerOpen" title="进件详情" :subtitle="detail?.enroll_no" width="max-w-2xl">
      <div v-if="detail" class="space-y-4">
        <!-- 状态 + 通道概览：单行显示，主状态与关键字段一字排开，右侧子商户号 -->
        <div class="flex min-w-0 items-center gap-x-4 whitespace-nowrap bg-muted/40 px-4 py-3 text-sm">
          <Badge :variant="statusVariant[detail.status] || 'muted'" class="shrink-0">{{ detail.status_text }}</Badge>
          <span class="min-w-0 truncate font-medium" :title="detail.channel_name">{{ detail.channel_name }}</span>
          <span class="shrink-0 text-xs text-muted-foreground">商户 <span class="text-foreground tabular-nums">{{ detail.uid }}</span></span>
          <span class="shrink-0 text-xs text-muted-foreground">微信状态 <span class="text-foreground">{{ detail.wx_state_text || '—' }}</span></span>
          <span v-if="detail.sub_mchid" class="ml-auto shrink-0 tabular-nums">
            <span class="text-muted-foreground">子商户号 </span>{{ detail.sub_mchid }}
          </span>
        </div>

        <!-- 驳回原因回显 -->
        <div v-if="detail.status === 'rejected' && detail.reject_reason" class="flex gap-2 border-l-2 border-destructive bg-destructive/[0.05] px-4 py-2.5 text-xs text-muted-foreground">
          <XCircle class="size-4 shrink-0 text-destructive" />
          <span>微信驳回：{{ detail.reject_reason }}</span>
        </div>

        <!-- 待签约：二维码直接内嵌，超管扫码即签约，无需另开网页 -->
        <div v-if="detail.sign_url" class="flex gap-4 bg-warning/[0.08] px-4 py-4">
          <div class="shrink-0">
            <img v-if="signQrImageURL" :src="signQrImageURL" alt="签约二维码" class="size-32 rounded bg-white p-1.5" referrerpolicy="no-referrer" />
            <img v-else-if="signQrDataURL" :src="signQrDataURL" alt="签约二维码" class="size-32 rounded bg-white p-1.5" />
            <div v-else class="flex size-32 items-center justify-center rounded bg-white/60 text-xs text-muted-foreground">生成中…</div>
          </div>
          <div class="min-w-0 flex-1 text-sm">
            <div class="flex items-center gap-1.5 font-medium">
              <QrCode class="size-4 text-warning shrink-0" />需超级管理员扫码签约激活
            </div>
            <p class="mt-1.5 text-xs leading-relaxed text-muted-foreground">
              请商户的超级管理员<span v-if="material?.contact_name_masked" class="text-foreground">（{{ material.contact_name_masked }}）</span>用<span class="text-foreground">微信</span>扫描左侧二维码，关注「微信支付商家助手」公众号后，按指引完成核对信息、账户验证与签约。
            </p>
            <div class="mt-2.5 flex items-center gap-3">
              <button class="inline-flex items-center gap-1 text-xs text-primary transition-colors hover:text-primary/80" @click="copySignURL">
                <Copy class="size-3.5" />复制链接
              </button>
              <a :href="detail.sign_url" target="_blank" class="text-xs text-muted-foreground underline transition-colors hover:text-foreground">在新窗口打开</a>
            </div>
          </div>
        </div>

        <!-- 资料（脱敏，敏感字段已用微信平台公钥加密直传微信，平台不可解密） -->
        <div v-if="material?.filled">
          <div class="flex items-center gap-2 border-b border-border/60 pb-2 text-sm font-medium">
            <ShieldCheck class="size-4 text-muted-foreground" />进件资料（脱敏）
          </div>
          <dl class="grid grid-cols-1 gap-x-8 gap-y-px bg-muted/40 px-4 py-1 text-sm sm:grid-cols-2 mt-2">
            <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">主体类型</dt><dd class="font-medium">{{ material.subject_type || '—' }}</dd></div>
            <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">商户简称</dt><dd class="font-medium">{{ material.merchant_shortname || '—' }}</dd></div>
            <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">执照名称</dt><dd class="font-medium">{{ material.business_merchant_name || material.cert_merchant_name || '—' }}</dd></div>
            <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">法人/经营者</dt><dd class="font-medium">{{ material.legal_person || material.cert_legal_person || '—' }}</dd></div>
            <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">证照编号</dt><dd class="font-medium">{{ material.license_number || material.cert_number || '—' }}</dd></div>
            <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">证件号码</dt><dd class="font-medium">{{ material.has_id_card_name || material.has_id_doc_name ? '已加密直传微信' : '—' }}</dd></div>
            <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">开户银行</dt><dd class="font-medium">{{ material.account_bank || '—' }}</dd></div>
            <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">银行账号</dt><dd class="font-medium">{{ material.has_account_number ? '已加密直传微信' : '—' }}</dd></div>
            <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">客服电话</dt><dd class="font-medium tabular-nums">{{ material.service_phone || '—' }}</dd></div>
            <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">联系手机</dt><dd class="font-medium tabular-nums">{{ material.contact_phone || '—' }}</dd></div>
            <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">结算行业</dt><dd class="font-medium">{{ material.qualification_type || '—' }}</dd></div>
            <div class="flex border-b border-border/60 py-2.5"><dt class="w-24 shrink-0 text-muted-foreground">经营场景</dt><dd class="font-medium">{{ (material.sales_scenes_type || []).length }} 类</dd></div>
            <div v-if="material.remark" class="flex py-2.5 sm:col-span-2"><dt class="w-24 shrink-0 text-muted-foreground">备注</dt><dd class="font-medium">{{ material.remark }}</dd></div>
          </dl>
          <p class="mt-2 text-xs text-muted-foreground">敏感字段（证件号/银行账号）已用微信平台公钥加密后直传微信，平台侧不留明文、不可解密。</p>
        </div>
        <p v-else class="py-6 text-center text-sm text-muted-foreground">商户尚未填写进件资料</p>
      </div>

      <template #footer>
        <Button v-if="detail && detail.status === 'submitted'" variant="outline" :disabled="busy" @click="doSync">
          <RefreshCw class="size-4" :class="{ 'animate-spin': busy }" />刷新微信状态
        </Button>
        <Button v-if="detail && (detail.status === 'submitted' || detail.status === 'pending')" variant="outline" :disabled="busy" @click="doReject">
          <XCircle class="size-4" />驳回
        </Button>
        <Button v-if="detail && detail.status !== 'approved' && detail.status !== 'draft'" :disabled="busy" @click="doApprove">
          <CheckCircle2 class="size-4" />手动交付
        </Button>
      </template>
    </Drawer>
  </div>
</template>
