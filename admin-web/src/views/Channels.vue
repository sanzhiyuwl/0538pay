<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  Search,
  RotateCcw,
  Plus,
  MoreHorizontal,
  KeyRound,
  Pencil,
  Trash2,
  ReceiptText,
  Copy,
  FlaskConical,
  ShoppingCart,
  Coins,
} from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Switch, Pagination, Drawer, Modal } from '@/components/ui'
import {
  channelMode,
  typeOptions,
  pluginsByType,
  calcChannelStats,
  type Channel,
} from '@/lib/mock/channels'
import {
  fetchChannels,
  createChannel,
  updateChannel,
  deleteChannel,
  setChannelStatus,
  fetchChannelConfig,
  saveChannelConfig,
  fetchPluginMeta,
  channelTestPay,
  type ChannelSaveReq,
  type PluginFieldInput,
  type PluginMeta,
} from '@/lib/api/channels'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { shouldDropUp } from '@/composables/useRowMenu'
import { formatMoney } from '@/lib/utils'

const toast = useToast()
const router = useRouter()

// 通道数据（真接口，一次拉取全部后客户端筛选/分页）。
// 对齐 epay：pay_channel 列表本就全量返回、无服务端分页；通道数量有限，
// 全量拉取才能保证概况卡（总数/开启/今昨收款）统计准确。pageSize=500 覆盖实际规模上限。
const allChannels = ref<Channel[]>([])
async function loadChannels() {
  try {
    const res = await fetchChannels({ page: 1, pageSize: 500 })
    allChannels.value = res.list
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '通道加载失败')
    allChannels.value = []
  }
}

const statusOptions = [
  { value: -1, label: '全部状态' },
  { value: 1, label: '状态已开启' },
  { value: 0, label: '状态已关闭' },
]

// ===== 筛选 =====
const filters = ref({ kw: '', plugin: '', type: 0, dstatus: -1 })

const filtered = computed(() => {
  return allChannels.value.filter((c) => {
    if (filters.value.type && c.type !== filters.value.type) return false
    if (filters.value.dstatus > -1 && c.status !== filters.value.dstatus) return false
    if (filters.value.plugin.trim() && !c.plugin.includes(filters.value.plugin.trim())) return false
    if (filters.value.kw.trim()) {
      const v = filters.value.kw.trim()
      if (!`${c.id}${c.name}`.includes(v)) return false
    }
    return true
  })
})

function resetFilters() {
  filters.value = { kw: '', plugin: '', type: 0, dstatus: -1 }
  page.value = 1
}

// ===== 分页 =====
const page = ref(1)
const pageSize = 15
const total = computed(() => filtered.value.length)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const safePage = computed(() => Math.min(page.value, pageCount.value))
const pageRows = computed(() =>
  filtered.value.slice((safePage.value - 1) * pageSize, safePage.value * pageSize),
)
function go(p: number) {
  page.value = Math.min(Math.max(1, p), pageCount.value)
}

// ===== 概况 =====
const stats = computed(() => calcChannelStats(filtered.value))

// ===== 状态切换（调真接口，失败回滚）=====
async function toggleStatus(id: number) {
  const c = allChannels.value.find((x) => x.id === id)
  if (!c) return
  const prev = c.status
  const next = (prev === 1 ? 0 : 1) as 0 | 1
  c.status = next // 乐观更新，先给即时反馈
  try {
    await setChannelStatus(id, next)
    toast.success(next === 1 ? '通道已开启' : '通道已关闭')
  } catch (e) {
    c.status = prev // 失败回滚
    toast.error(e instanceof ApiError ? e.message : '状态切换失败')
  }
}

// ===== 行操作菜单 =====
const openMenu = ref<number | null>(null)
const dropUp = ref(false)
function toggleMenu(id: number, ev?: MouseEvent) {
  if (openMenu.value === id) { openMenu.value = null; return }
  openMenu.value = id
  dropUp.value = shouldDropUp(ev)
}
function closeMenu() {
  openMenu.value = null
}

// ===== 行操作：查看该通道订单（对齐 epay pay_channel「订单」→ order.php?channel=id）=====
function viewOrders(c: Channel) {
  openMenu.value = null
  router.push({ path: '/admin/orders', query: { channel: String(c.id) } })
}

// ===== 行操作：测试支付（对齐 epay pay_channel「测试支付」→ ajax_pay.php act=testpay）=====
const testDialog = ref(false)
const testTarget = ref<Channel | null>(null)
const testForm = reactive({ name: '支付测试', money: '1' })
const testSubmitting = ref(false)
function openTestPay(c: Channel) {
  openMenu.value = null
  testTarget.value = c
  testForm.name = '支付测试'
  testForm.money = '1'
  testDialog.value = true
}
async function submitTestPay() {
  if (!testTarget.value || testSubmitting.value) return
  const money = Number(testForm.money)
  if (!(money > 0)) {
    toast.error('请输入有效金额')
    return
  }
  testSubmitting.value = true
  try {
    const res = await channelTestPay({
      channel: testTarget.value.id,
      name: testForm.name.trim() || '支付测试',
      money: testForm.money.trim(),
    })
    testDialog.value = false
    // 下单成功 → 跳收银台（对齐 epay window.open(testsubmit.php)），mock 渠道走模拟支付页
    router.push({ path: `/pay/mock/cashier/${res.trade_no}` })
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '测试支付下单失败')
  } finally {
    testSubmitting.value = false
  }
}

// 插件元数据（后端 /channels/plugins 自声明的密钥字段/能力），驱动密钥表单动态渲染
const pluginMeta = ref<Record<string, PluginMeta>>({})
async function loadPluginMeta() {
  try {
    const list = await fetchPluginMeta()
    const map: Record<string, PluginMeta> = {}
    for (const m of list) map[m.key] = m
    pluginMeta.value = map
  } catch {
    // 元数据拉取失败不阻塞列表；密钥抽屉会退回通用 key-value 编辑
    pluginMeta.value = {}
  }
}

onMounted(() => {
  window.addEventListener('click', closeMenu)
  loadChannels()
  loadPluginMeta()
})
onUnmounted(() => window.removeEventListener('click', closeMenu))

// ===== 新增/编辑抽屉 =====
const channelDrawer = ref(false)
const editingId = ref<number | null>(null)
const saving = ref(false)
const form = reactive<ChannelSaveReq>({
  name: '',
  type: 1,
  plugin: '',
  mode: 0,
  rate: '',
  costrate: '',
  daytop: 0,
  paymin: '',
  paymax: '',
})

// 支付方式下拉（去掉“所有支付方式”那项，表单必须选具体方式）
const typeSelectOptions = computed(() => typeOptions.filter((o) => o.value !== 0))
// 按当前 type 联动插件候选
const pluginOptions = computed(() => {
  const list = pluginsByType[form.type] || []
  return list.map((p) => ({ value: p.name, label: `${p.showname} (${p.name})` }))
})

function resetForm() {
  Object.assign(form, {
    name: '', type: 1, plugin: '', mode: 0, rate: '',
    costrate: '', daytop: 0, paymin: '', paymax: '',
  })
}

function openCreate() {
  editingId.value = null
  resetForm()
  channelDrawer.value = true
}

function openEdit(c: Channel) {
  editingId.value = c.id
  Object.assign(form, {
    name: c.name,
    type: c.type || 1,
    plugin: c.plugin,
    mode: c.mode,
    rate: c.rate,
    costrate: c.costrate,
    daytop: c.daytop,
    paymin: c.paymin,
    paymax: c.paymax,
  })
  channelDrawer.value = true
  openMenu.value = null
}

function openCopy(c: Channel) {
  // 复制：带入源通道字段但清空名称，走新增（对齐 epay copy）
  editingId.value = null
  Object.assign(form, {
    name: c.name + ' 副本',
    type: c.type || 1,
    plugin: c.plugin,
    mode: c.mode,
    rate: c.rate,
    costrate: c.costrate,
    daytop: c.daytop,
    paymin: c.paymin,
    paymax: c.paymax,
  })
  channelDrawer.value = true
  openMenu.value = null
}

// type 变化时，若当前 plugin 不在新方式的候选内，清空 plugin 待重选
function onTypeChange() {
  const list = pluginsByType[form.type] || []
  if (!list.some((p) => p.name === form.plugin)) form.plugin = ''
}

async function saveChannel() {
  if (!form.name.trim()) return toast.error('请填写显示名称')
  if (!form.plugin) return toast.error('请选择支付插件')
  if (!form.rate.trim()) return toast.error('请填写分成比例')
  const payload: ChannelSaveReq = {
    name: form.name.trim(),
    type: form.type,
    plugin: form.plugin,
    mode: form.mode,
    rate: form.rate.trim(),
    costrate: form.costrate.trim(),
    daytop: form.mode === 1 ? 0 : Number(form.daytop) || 0,
    paymin: form.paymin.trim(),
    paymax: form.paymax.trim(),
  }
  saving.value = true
  try {
    if (editingId.value) {
      await updateChannel(editingId.value, payload)
      toast.success('通道已更新')
    } else {
      await createChannel(payload)
      toast.success('通道已创建（默认关闭，配置好密钥后再开启）')
    }
    channelDrawer.value = false
    await loadChannels()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

// ===== 删除确认 =====
const delTarget = ref<Channel | null>(null)
const deleting = ref(false)
function askDelete(c: Channel) {
  delTarget.value = c
  openMenu.value = null
}
async function confirmDelete() {
  if (!delTarget.value) return
  deleting.value = true
  try {
    await deleteChannel(delTarget.value.id)
    toast.success('通道已删除')
    delTarget.value = null
    await loadChannels()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '删除失败')
  } finally {
    deleting.value = false
  }
}

// ===== 密钥配置抽屉 =====
// 插件元数据(pluginMeta)声明了 inputs 字段的走专用表单，否则退回通用 key-value 编辑。
const configDrawer = ref(false)
const configTarget = ref<Channel | null>(null)
const configLoading = ref(false)
const configSaving = ref(false)
// 通用模式：key-value 行编辑
const configRows = ref<{ key: string; value: string }[]>([])
// 预设模式：字段定义(来自后端插件元数据) + 值表
const presetFields = ref<PluginFieldInput[]>([])
const presetForm = ref<Record<string, string>>({})

// 把 config JSON 字符串解析为 key-value 行（值统一转字符串展示）
function parseConfigToRows(raw: string): { key: string; value: string }[] {
  if (!raw || !raw.trim()) return []
  try {
    const obj = JSON.parse(raw)
    if (obj && typeof obj === 'object' && !Array.isArray(obj)) {
      return Object.entries(obj).map(([k, v]) => ({
        key: k,
        value: typeof v === 'string' ? v : JSON.stringify(v),
      }))
    }
  } catch {
    // 非法 JSON：留空让用户重填
  }
  return []
}

// 当前通道是否有插件预设表单
const hasPreset = computed(() => presetFields.value.length > 0)

// 把 config JSON 解析为对象（供预设表单回填）
function parseConfigToObject(raw: string): Record<string, string> {
  if (!raw || !raw.trim()) return {}
  try {
    const obj = JSON.parse(raw)
    if (obj && typeof obj === 'object' && !Array.isArray(obj)) {
      const out: Record<string, string> = {}
      for (const [k, v] of Object.entries(obj)) {
        out[k] = typeof v === 'string' ? v : JSON.stringify(v)
      }
      return out
    }
  } catch {
    // 非法 JSON 忽略
  }
  return {}
}

async function openConfig(c: Channel) {
  configTarget.value = c
  configRows.value = []
  presetForm.value = {}
  presetFields.value = pluginMeta.value[c.plugin]?.inputs ?? []
  configDrawer.value = true
  openMenu.value = null
  configLoading.value = true
  try {
    const res = await fetchChannelConfig(c.id)
    if (presetFields.value.length) {
      // 预设模式：按字段定义回填，未知键也保留（合并保存时不丢）
      presetForm.value = parseConfigToObject(res.config)
    } else {
      configRows.value = parseConfigToRows(res.config)
    }
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '读取密钥配置失败')
  } finally {
    configLoading.value = false
  }
}

function addConfigRow() {
  configRows.value.push({ key: '', value: '' })
}
function removeConfigRow(i: number) {
  configRows.value.splice(i, 1)
}

async function saveConfig() {
  if (!configTarget.value) return
  let obj: Record<string, string> = {}
  if (hasPreset.value) {
    // 预设模式：先以已回填对象为底（保留未知键），再覆盖预设字段值
    obj = { ...presetForm.value }
    for (const f of presetFields.value) {
      const v = (presetForm.value[f.name] ?? '').trim()
      if (f.require && !v) {
        toast.error(`请填写「${f.label}」`)
        return
      }
      obj[f.name] = presetForm.value[f.name] ?? ''
    }
  } else {
    // 通用模式：忽略空 key，key 去重（后者覆盖前者）
    for (const row of configRows.value) {
      const k = row.key.trim()
      if (!k) continue
      obj[k] = row.value
    }
  }
  configSaving.value = true
  try {
    await saveChannelConfig(configTarget.value.id, JSON.stringify(obj))
    toast.success('密钥配置已保存')
    configDrawer.value = false
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    configSaving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="支付通道" :subtitle="`共 ${total} 个通道`">
      <template #actions>
        <Button size="sm" @click="openCreate"><Plus />新增通道</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">通道搜索</label>
          <input v-model="filters.kw" placeholder="通道 ID / 名称" class="field-input w-44" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">支付插件</label>
          <input v-model="filters.plugin" placeholder="插件标识" class="field-input w-32" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">支付方式</label>
          <Select v-model="filters.type" :options="typeOptions" class="w-32" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">状态</label>
          <Select v-model="filters.dstatus" :options="statusOptions" class="w-32" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="page = 1"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 概况 -->
    <Panel title="通道概况" subtitle="按当前筛选条件">
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">通道总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.total }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已开启</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ stats.open }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已关闭</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-muted-foreground">{{ stats.closed }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">今日收款合计</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-primary"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.todayTotal) }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">昨日收款合计</div>
          <div class="mt-1 text-xl font-normal tabular-nums"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.yesterdayTotal) }}</div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="通道列表" :subtitle="`${total} 条`">
      <div>
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[6%]">ID</th>
              <th class="w-[16%]">显示名称</th>
              <th class="w-[10%]">通道模式</th>
              <th class="w-[10%]">支付方式</th>
              <th class="w-[12%]">支付插件</th>
              <th class="num w-[9%]">分成比例</th>
              <th class="num w-[12%]">今日收款</th>
              <th class="num w-[12%]">昨日收款</th>
              <th class="col-center w-[7%]">状态</th>
              <th class="col-center w-[6%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in pageRows" :key="c.id">
              <td class="font-medium tabular-nums">{{ c.id }}</td>
              <td class="truncate">{{ c.name }}</td>
              <td>
                <Badge :variant="c.mode === 1 ? 'warning' : 'muted'">{{ channelMode[c.mode] }}</Badge>
              </td>
              <td>{{ c.typeshowname }}</td>
              <td>
                <span class="truncate text-primary" :title="c.plugin">{{ c.plugin }}</span>
              </td>
              <td class="num tabular-nums">{{ c.rate }}%</td>
              <td class="num tabular-nums">
                <span v-if="c.status === 1"><span class="dim text-xs">¥</span>{{ c.today }}</span>
                <span v-else class="dim">—</span>
              </td>
              <td class="num tabular-nums">
                <span v-if="+c.yesterday > 0"><span class="dim text-xs">¥</span>{{ c.yesterday }}</span>
                <span v-else class="dim">—</span>
              </td>
              <td class="col-center">
                <div class="flex justify-center">
                  <Switch :model-value="c.status === 1" size="sm" @update:model-value="toggleStatus(c.id)" />
                </div>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(c.id, $event)">
                    <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === c.id"
                    class="menu-panel absolute right-0 z-20 w-32"
                    :class="dropUp ? 'bottom-full mb-1.5' : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <button class="menu-item" @click="openConfig(c)">
                      <KeyRound class="size-4 shrink-0 opacity-70" /><span class="flex-1">配置密钥</span>
                    </button>
                    <button class="menu-item" @click="openEdit(c)">
                      <Pencil class="size-4 shrink-0 opacity-70" /><span class="flex-1">编辑</span>
                    </button>
                    <button class="menu-item" @click="openCopy(c)">
                      <Copy class="size-4 shrink-0 opacity-70" /><span class="flex-1">复制</span>
                    </button>
                    <button class="menu-item" @click="viewOrders(c)">
                      <ReceiptText class="size-4 shrink-0 opacity-70" /><span class="flex-1">订单</span>
                    </button>
                    <button class="menu-item" @click="openTestPay(c)">
                      <FlaskConical class="size-4 shrink-0 opacity-70" /><span class="flex-1">测试支付</span>
                    </button>
                    <div class="menu-sep" />
                    <button class="menu-item menu-item-danger" @click="askDelete(c)">
                      <Trash2 class="size-4 shrink-0 opacity-70" /><span class="flex-1">删除</span>
                    </button>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="10" class="py-10 text-center dim">没有符合条件的支付通道</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="safePage" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>

    <!-- 新增/编辑通道抽屉（对齐 epay pay_channel 编辑表单）-->
    <Drawer
      v-model="channelDrawer"
      :title="editingId ? '编辑通道' : '新增通道'"
      subtitle="配置支付方式、插件、费率与限额"
    >
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">显示名称<span class="text-destructive">*</span></label>
          <input v-model="form.name" placeholder="如：支付宝官方直连" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">支付方式<span class="text-destructive">*</span></label>
          <Select v-model="form.type" :options="typeSelectOptions" class="flex-1" @update:model-value="onTypeChange" />
        </div>
        <div class="row-field">
          <label class="lbl">支付插件<span class="text-destructive">*</span></label>
          <Select
            v-model="form.plugin"
            :options="pluginOptions"
            placeholder="请选择插件"
            class="flex-1"
          />
        </div>
        <div class="row-field">
          <label class="lbl">通道模式</label>
          <Select
            v-model="form.mode"
            :options="[{ value: 0, label: '平台代收（扣手续费入余额）' }, { value: 1, label: '商户直清（不入余额）' }]"
            class="flex-1"
          />
        </div>
        <div class="row-field">
          <label class="lbl">分成比例<span class="text-destructive">*</span></label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.rate" placeholder="如 0.38" class="field-input flex-1" />
            <span class="text-sm text-muted-foreground">%</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">通道成本</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.costrate" placeholder="留空为 0" class="field-input flex-1" />
            <span class="text-sm text-muted-foreground">%</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">单日限额</label>
          <input
            v-model.number="form.daytop"
            type="number"
            :disabled="form.mode === 1"
            placeholder="0 为不限"
            class="field-input flex-1 disabled:opacity-50"
          />
        </div>
        <div class="row-field">
          <label class="lbl">单笔最小</label>
          <input v-model="form.paymin" placeholder="留空为不限" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">单笔最大</label>
          <input v-model="form.paymax" placeholder="留空为不限" class="field-input flex-1" />
        </div>
        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          商户直清模式下资金不入平台余额，单日限额将被忽略。新增通道默认关闭，配置好密钥后在列表手动开启。
        </p>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="channelDrawer = false">取消</Button>
        <Button size="sm" :disabled="saving" @click="saveChannel">
          {{ editingId ? '保存' : '创建' }}
        </Button>
      </template>
    </Drawer>

    <!-- 密钥配置抽屉（通用 key-value 编辑；B 阶段按插件定制专用表单）-->
    <Drawer
      v-model="configDrawer"
      title="配置通道密钥"
      :subtitle="configTarget ? `${configTarget.name} · ${configTarget.plugin}` : ''"
    >
      <div v-if="configLoading" class="py-10 text-center text-sm dim">加载中…</div>

      <!-- 预设模式：按插件字段定义渲染专用表单 -->
      <div v-else-if="hasPreset" class="space-y-3.5">
        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          按 {{ configTarget?.plugin }} 插件所需字段填写密钥参数，保存后即可用于真实收款。
          私钥/公钥请粘贴完整 PEM 内容。
        </p>
        <div v-for="f in presetFields" :key="f.name" class="row-field items-start">
          <label class="lbl pt-2">
            {{ f.label }}<span v-if="f.require" class="text-destructive">*</span>
          </label>
          <div class="flex-1 space-y-1">
            <textarea
              v-if="f.type === 'textarea'"
              v-model="presetForm[f.name]"
              :placeholder="f.tip"
              rows="4"
              class="field-input w-full resize-y font-mono text-xs"
            />
            <select
              v-else-if="f.type === 'select'"
              v-model="presetForm[f.name]"
              class="field-input w-full"
            >
              <option v-for="opt in f.options ?? []" :key="opt" :value="opt">{{ opt }}</option>
            </select>
            <input
              v-else
              v-model="presetForm[f.name]"
              :type="f.type === 'password' ? 'password' : 'text'"
              :placeholder="f.tip"
              class="field-input w-full"
            />
            <p v-if="f.tip" class="text-xs text-muted-foreground">{{ f.tip }}</p>
          </div>
        </div>
      </div>

      <!-- 通用模式：key-value 编辑（无预设的插件）-->
      <div v-else class="space-y-3">
        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          按「键 / 值」编辑渠道密钥参数（如 appid、mch_id、私钥等），保存为 JSON。
          该插件暂无专用表单预设。
        </p>
        <div v-for="(row, i) in configRows" :key="i" class="flex items-center gap-2">
          <input v-model="row.key" placeholder="键，如 appid" class="field-input w-36" />
          <input v-model="row.value" placeholder="值" class="field-input flex-1" />
          <button
            class="flex size-8 shrink-0 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-destructive"
            title="删除此项"
            @click="removeConfigRow(i)"
          >
            <Trash2 class="size-4" />
          </button>
        </div>
        <div v-if="!configRows.length" class="py-4 text-center text-sm dim">
          暂无参数，点击下方按钮添加
        </div>
        <Button variant="outline" size="sm" @click="addConfigRow"><Plus />添加参数</Button>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="configDrawer = false">取消</Button>
        <Button size="sm" :disabled="configSaving || configLoading" @click="saveConfig">保存</Button>
      </template>
    </Drawer>

    <!-- 删除确认 -->
    <Modal :model-value="!!delTarget" title="删除通道" @update:model-value="(v) => { if (!v) delTarget = null }">
      <p class="text-sm text-muted-foreground">
        确定删除通道
        <b class="text-foreground">{{ delTarget?.name }}</b>
        （ID {{ delTarget?.id }}）吗？删除后该通道将无法用于下单，此操作不可恢复。
      </p>
      <template #footer>
        <Button variant="outline" size="sm" @click="delTarget = null">取消</Button>
        <Button variant="destructive" size="sm" :disabled="deleting" @click="confirmDelete">删除</Button>
      </template>
    </Modal>

    <!-- 测试支付（对齐 epay pay_channel testpay：订单名 + 金额，下单后跳收银台）-->
    <Modal
      :model-value="testDialog"
      title="测试支付"
      @update:model-value="(v) => { if (!v) testDialog = false }"
    >
      <p class="mb-3 text-sm text-muted-foreground">
        对通道
        <b class="text-foreground">{{ testTarget?.name }}</b>
        （ID {{ testTarget?.id }}）发起一笔真实测试订单，验证密钥配置与收款是否正常。
      </p>
      <div class="space-y-3">
        <div>
          <label class="mb-1 block text-sm text-muted-foreground">订单名称</label>
          <div class="relative">
            <ShoppingCart class="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <input v-model="testForm.name" placeholder="支付测试" class="field-input w-full !pl-9" />
          </div>
        </div>
        <div>
          <label class="mb-1 block text-sm text-muted-foreground">订单金额</label>
          <div class="relative">
            <Coins class="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <input v-model="testForm.money" placeholder="1" class="field-input w-full !pl-9 tabular-nums" />
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="testDialog = false">取消</Button>
        <Button size="sm" :disabled="testSubmitting" @click="submitTestPay">发起支付</Button>
      </template>
    </Modal>
  </div>
</template>
