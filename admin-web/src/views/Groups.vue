<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  Plus,
  Pencil,
  Trash2,
  Users,
  Check,
  X,
  ArrowUpFromLine,
  ArrowDownToLine,
  Store,
} from 'lucide-vue-next'
import { Panel, Button, Badge, Switch, Select, Drawer, Modal } from '@/components/ui'
import { settleOpenText, capabilityList, type GroupConfig } from '@/lib/mock/groups'
import {
  fetchGroups,
  createGroup,
  updateGroup,
  setGroupBuy,
  deleteGroup,
  fetchGroupAssigns,
  saveGroupAssigns,
  type GroupView,
  type GroupSaveReq,
  type GroupAssignItem,
} from '@/lib/api/groups'
import { fetchRolls, type RollView } from '@/lib/api/rolls'
import { fetchChannels } from '@/lib/api/channels'
import type { Channel } from '@/lib/mock/channels'
import { payTypes } from '@/lib/mock/orders'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const router = useRouter()

// 查看该用户组下的商户（对齐 epay glist「用户」→ ulist.php?gid=）
function viewMerchants(g: GroupView) {
  router.push({ path: '/admin/merchants', query: { gid: String(g.gid) } })
}

const allGroups = ref<GroupView[]>([])
const loading = ref(false)
const allRolls = ref<RollView[]>([])
const allChannels = ref<Channel[]>([])

async function loadGroups() {
  loading.value = true
  try {
    const [res, rollRes, chRes] = await Promise.all([
      fetchGroups(),
      fetchRolls(),
      fetchChannels({ pageSize: 100 }),
    ])
    allGroups.value = res.list
    allRolls.value = rollRes.list
    allChannels.value = chRes.list
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载用户组失败')
    allGroups.value = []
  } finally {
    loading.value = false
  }
}

onMounted(loadGroups)

// 通道分配下拉：0关闭 / -1随机 / -2子通道 / 各固定通道(kind=channel) / 各轮询组(kind=roll)。
// 每项按支付方式(type)过滤。value 编码为 "channel:数值" 便于携带 kind。
function assignOptions(typeID: number) {
  const opts = [
    { value: 'channel:0', label: '关闭' },
    { value: 'channel:-1', label: '随机可用通道' },
    { value: 'channel:-2', label: '用户自定义子通道' },
  ]
  for (const c of allChannels.value.filter((x) => x.type === typeID)) {
    opts.push({ value: `channel:${c.id}`, label: `通道：${c.name}` })
  }
  for (const r of allRolls.value.filter((x) => x.type === typeID)) {
    opts.push({ value: `roll:${r.id}`, label: `轮询组：${r.name}` })
  }
  return opts
}

// assign 项在表单里的编辑态：value 编码 "kind:channel"，rate 单列。
interface AssignRow {
  type: number
  typeName: string
  value: string // "kind:channel"
  rate: string
}

function encodeAssign(a: GroupAssignItem): string {
  const n = Number(a.channel)
  // 0/-1/-2 一律 channel 前缀；正整数按 kind
  if (n > 0) return `${a.kind === 'roll' ? 'roll' : 'channel'}:${a.channel}`
  return `channel:${a.channel}`
}
function decodeAssign(row: AssignRow): GroupAssignItem {
  const [kind, channel] = row.value.split(':')
  return { type: row.type, kind, channel, rate: row.rate.trim() }
}

// 按排序展示（sort 越小越靠前）
const groups = computed(() => [...allGroups.value].sort((a, b) => a.sort - b.sort))

const totalMerchants = computed(() => allGroups.value.reduce((a, g) => a + g.merchantCount, 0))
const onSaleCount = computed(() => allGroups.value.filter((g) => g.isbuy === 1).length)

function expireText(month: number) {
  return month === 0 ? '永久有效' : `${month} 个月`
}

// 默认能力配置
function defaultConfig(): GroupConfig {
  return {
    settleOpen: '3',
    settleType: '2',
    settleRate: '0.1',
    recharge: false,
    userTransfer: false,
    inviteOpen: false,
    userDeposit: false,
  }
}

// 解析 config JSON → 结构化能力配置（容错，缺字段用默认）
function parseConfig(raw: string): GroupConfig {
  const d = defaultConfig()
  if (!raw || !raw.trim()) return d
  try {
    const o = JSON.parse(raw)
    return {
      settleOpen: o.settleOpen ?? d.settleOpen,
      settleType: o.settleType ?? d.settleType,
      settleRate: o.settleRate ?? d.settleRate,
      recharge: !!o.recharge,
      userTransfer: !!o.userTransfer,
      inviteOpen: !!o.inviteOpen,
      userDeposit: !!o.userDeposit,
    }
  } catch {
    return d
  }
}

// 卡片渲染的 config（解析自后端 JSON）
function cardConfig(g: GroupView): GroupConfig {
  return parseConfig(g.config)
}

// 卡片通道分配摘要：解析 info 的 {typeid:{type,channel,rate}}，每种支付方式一行文案。
interface AssignSummary {
  typeName: string
  target: string // 分配目标文案
  rate: string // 费率覆盖（空=默认）
}
function cardAssigns(g: GroupView): AssignSummary[] {
  if (!g.info || !g.info.trim()) return []
  let obj: Record<string, { type?: string; channel?: string; rate?: string }>
  try {
    obj = JSON.parse(g.info)
  } catch {
    return []
  }
  if (Array.isArray(obj) || typeof obj !== 'object') return [] // 旧格式忽略
  const out: AssignSummary[] = []
  for (const t of payTypes) {
    const a = obj[String(t.id)]
    if (!a) continue
    out.push({ typeName: t.showname, target: assignTargetText(a), rate: (a.rate ?? '').trim() })
  }
  return out
}
function assignTargetText(a: { type?: string; channel?: string }): string {
  const n = Number(a.channel)
  if (n === 0) return '关闭'
  if (n === -1) return '随机'
  if (n === -2) return '子通道'
  if (n > 0) {
    if (a.type === 'roll') {
      const r = allRolls.value.find((x) => x.id === n)
      return `轮询组：${r?.name ?? n}`
    }
    const c = allChannels.value.find((x) => x.id === n)
    return `通道：${c?.name ?? n}`
  }
  return '随机'
}

// ===== 新增/编辑抽屉 =====
const drawer = ref(false)
const editingGID = ref<number | null>(null)
const saving = ref(false)
const form = reactive({
  name: '',
  isbuy: 0,
  price: '0.00',
  expire: 0,
  sort: 0,
  visible: '',
  config: defaultConfig(),
  infoRaw: '', // 现有 info 原文（通道分配），基本信息保存时原样保留，避免被清空
  assigns: [] as AssignRow[],
})

// 按可用支付方式构建分配行；existing 为该组已存分配（缺项补默认「随机可用通道」）。
function buildAssignRows(existing: GroupAssignItem[]): AssignRow[] {
  const byType = new Map(existing.map((a) => [a.type, a]))
  return payTypes.map((t) => {
    const a = byType.get(t.id)
    return {
      type: t.id,
      typeName: t.showname,
      value: a ? encodeAssign(a) : 'channel:-1',
      rate: a?.rate ?? '',
    }
  })
}

function resetForm() {
  Object.assign(form, {
    name: '', isbuy: 0, price: '0.00', expire: 0, sort: 0, visible: '',
    config: defaultConfig(), infoRaw: '', assigns: buildAssignRows([]),
  })
}

function openCreate() {
  editingGID.value = null
  resetForm()
  drawer.value = true
}

async function openEdit(g: GroupView) {
  editingGID.value = g.gid
  Object.assign(form, {
    name: g.name,
    isbuy: g.isbuy,
    price: g.price,
    expire: g.expire,
    sort: g.sort,
    visible: g.visible,
    config: parseConfig(g.config),
    infoRaw: g.info,
    assigns: buildAssignRows([]),
  })
  drawer.value = true
  // 拉取该组现有通道分配填充编辑态
  try {
    const res = await fetchGroupAssigns(g.gid)
    form.assigns = buildAssignRows(res.list)
  } catch {
    // 拉取失败保持默认行，不阻断编辑
  }
}

async function save() {
  if (!form.name.trim()) return toast.error('请填写用户组名称')
  const payload: GroupSaveReq = {
    name: form.name.trim(),
    isbuy: form.isbuy,
    price: form.price.trim() || '0',
    expire: Number(form.expire) || 0,
    sort: Number(form.sort) || 0,
    visible: form.visible.trim(),
    info: form.infoRaw, // 保留现有通道分配，由 saveGroupAssigns 权威写入
    config: JSON.stringify(form.config),
    settings: '',
  }
  const assigns = form.assigns.map(decodeAssign)
  saving.value = true
  try {
    let gid: number
    if (editingGID.value !== null) {
      await updateGroup(editingGID.value, payload)
      gid = editingGID.value
    } else {
      const res = await createGroup(payload)
      gid = res.gid
    }
    // 通道分配权威写入（写 info）。放在基本信息保存之后，确保覆盖生效。
    await saveGroupAssigns(gid, assigns)
    toast.success(editingGID.value !== null ? '用户组已更新' : '用户组已创建')
    drawer.value = false
    await loadGroups()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

// ===== 上/下架 =====
async function toggleBuy(g: GroupView) {
  const next = g.isbuy === 1 ? 0 : 1
  try {
    await setGroupBuy(g.gid, next)
    g.isbuy = next
    toast.success(next === 1 ? '已上架' : '已下架')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  }
}

// ===== 删除 =====
const delTarget = ref<GroupView | null>(null)
const deleting = ref(false)
function askDelete(g: GroupView) {
  delTarget.value = g
}
async function confirmDelete() {
  if (!delTarget.value) return
  deleting.value = true
  try {
    await deleteGroup(delTarget.value.gid)
    toast.success('用户组已删除，该组商户已回落默认组')
    delTarget.value = null
    await loadGroups()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '删除失败')
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 概览 -->
    <Panel title="用户组 / 套餐" subtitle="每个用户组即一个套餐：定义通道费率、结算规则与功能权限，可上架供商户购买">
      <template #actions>
        <Button size="sm" @click="openCreate"><Plus />新增套餐</Button>
      </template>
      <div class="flex flex-wrap items-center gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">套餐总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ allGroups.length }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已上架</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ onSaleCount }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">覆盖商户</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ totalMerchants }}</div>
        </div>
      </div>
    </Panel>

    <!-- 套餐卡片 -->
    <div class="grid grid-cols-1 gap-2.5 lg:grid-cols-2 xl:grid-cols-4">
      <Panel v-for="g in groups" :key="g.gid" flush>
        <div class="p-4">
          <!-- 头部：名称 + 上架状态 -->
          <div class="flex items-start justify-between">
            <div>
              <div class="flex items-center gap-2">
                <span class="text-base font-semibold">{{ g.name }}</span>
                <span class="text-xs text-muted-foreground">GID {{ g.gid }}</span>
              </div>
              <Badge v-if="g.isbuy" variant="success" class="mt-1.5">已上架</Badge>
              <Badge v-else variant="muted" class="mt-1.5">未上架</Badge>
            </div>
            <div class="flex items-center gap-1 text-xs text-muted-foreground">
              <Store class="size-3.5" />{{ g.merchantCount }}
            </div>
          </div>

          <!-- 定价 -->
          <div class="mt-3 flex items-baseline gap-1.5">
            <span class="text-sm text-muted-foreground">¥</span>
            <span class="text-2xl font-semibold tabular-nums">{{ g.price }}</span>
            <span class="text-sm text-muted-foreground">/ {{ expireText(g.expire) }}</span>
          </div>

          <!-- 通道分配 -->
          <div class="mt-3.5 border-t border-border/60 pt-3">
            <div class="mb-1.5 text-xs font-medium text-muted-foreground">通道分配</div>
            <div v-if="cardAssigns(g).length" class="space-y-1">
              <div v-for="a in cardAssigns(g)" :key="a.typeName" class="flex items-center justify-between text-sm">
                <span>{{ a.typeName }}</span>
                <span class="flex items-center gap-1.5">
                  <span class="text-[13px] text-muted-foreground">{{ a.target }}</span>
                  <b v-if="a.rate" class="tabular-nums text-primary">{{ a.rate }}%</b>
                </span>
              </div>
            </div>
            <div v-else class="text-xs text-muted-foreground">随机分配可用通道，使用通道默认费率</div>
          </div>

          <!-- 结算规则 -->
          <div class="mt-3 border-t border-border/60 pt-3 text-sm">
            <div class="mb-1.5 text-xs font-medium text-muted-foreground">结算</div>
            <div class="flex flex-wrap gap-x-4 gap-y-1 text-[13px]">
              <span class="text-muted-foreground">方式 <b class="text-foreground">{{ settleOpenText[cardConfig(g).settleOpen] }}</b></span>
              <span class="text-muted-foreground">周期 <b class="text-foreground">{{ cardConfig(g).settleType === '1' ? 'D+0' : 'D+1' }}</b></span>
              <span class="text-muted-foreground">费率 <b class="text-foreground tabular-nums">{{ cardConfig(g).settleRate }}%</b></span>
            </div>
          </div>

          <!-- 能力开关 -->
          <div class="mt-3 border-t border-border/60 pt-3">
            <div class="mb-1.5 text-xs font-medium text-muted-foreground">功能权限</div>
            <div class="grid grid-cols-2 gap-1">
              <div
                v-for="cap in capabilityList(cardConfig(g))"
                :key="cap.label"
                class="flex items-center gap-1.5 text-[13px]"
                :class="cap.on ? 'text-foreground' : 'text-muted-foreground/60'"
              >
                <Check v-if="cap.on" class="size-3.5 text-success" />
                <X v-else class="size-3.5" />
                {{ cap.label }}
              </div>
            </div>
          </div>

          <!-- 操作 -->
          <div class="mt-4 flex items-center gap-1.5 border-t border-border/60 pt-3">
            <Button variant="outline" size="sm" @click="viewMerchants(g)"><Users />商户</Button>
            <Button variant="outline" size="sm" @click="openEdit(g)"><Pencil />编辑</Button>
            <Button v-if="g.isbuy" variant="ghost" size="sm" @click="toggleBuy(g)"><ArrowDownToLine />下架</Button>
            <Button v-else variant="ghost" size="sm" @click="toggleBuy(g)"><ArrowUpFromLine />上架</Button>
            <Button
              v-if="g.gid !== 0"
              variant="ghost"
              size="icon"
              class="ml-auto text-destructive hover:text-destructive"
              @click="askDelete(g)"
            >
              <Trash2 />
            </Button>
          </div>
        </div>
      </Panel>
    </div>

    <p class="px-1 text-xs text-muted-foreground">
      未设置用户组的商户归为「默认用户组」(GID 0)，自动使用可用支付通道与通道默认费率，该组不可删除。
    </p>

    <!-- 新增/编辑抽屉 -->
    <Drawer
      v-model="drawer"
      :title="editingGID !== null ? '编辑用户组' : '新增用户组'"
      subtitle="配置售价、有效期、通道费率与功能权限"
    >
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">组名称<span class="text-destructive">*</span></label>
          <input v-model="form.name" placeholder="如：白银会员" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">上架售卖</label>
          <Switch :model-value="form.isbuy === 1" @update:model-value="(v: boolean) => form.isbuy = v ? 1 : 0" />
        </div>
        <div class="row-field">
          <label class="lbl">售价</label>
          <div class="flex flex-1 items-center gap-2">
            <span class="text-sm text-muted-foreground">¥</span>
            <input v-model="form.price" placeholder="0.00" class="field-input flex-1" />
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">有效期(月)</label>
          <input v-model.number="form.expire" type="number" placeholder="0 为永久" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">排序</label>
          <input v-model.number="form.sort" type="number" placeholder="越小越靠前" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">可见范围</label>
          <input v-model="form.visible" placeholder="GID 列表逗号分隔，空=全部可见" class="field-input flex-1" />
        </div>

        <!-- 通道分配 -->
        <div class="border-t border-border/60 pt-3">
          <div class="mb-2 text-sm font-medium">通道分配</div>
          <p class="mb-2.5 text-xs text-muted-foreground">
            为每种支付方式指定通道来源：关闭 / 随机可用通道 / 用户自定义子通道 / 固定某通道或轮询组。费率留空则用通道默认费率。
          </p>
          <div v-for="row in form.assigns" :key="row.type" class="mb-2 flex items-center gap-2">
            <span class="w-20 shrink-0 text-sm">{{ row.typeName }}</span>
            <Select v-model="row.value" :options="assignOptions(row.type)" class="flex-1" />
            <div class="flex items-center gap-1">
              <input v-model="row.rate" placeholder="费率" class="field-input w-16" />
              <span class="text-sm text-muted-foreground">%</span>
            </div>
          </div>
        </div>

        <!-- 结算规则 -->
        <div class="border-t border-border/60 pt-3 space-y-3.5">
          <div class="row-field">
            <label class="lbl">结算方式</label>
            <Select
              v-model="form.config.settleOpen"
              :options="[{ value: '1', label: '仅自动结算' }, { value: '2', label: '仅手动申请' }, { value: '3', label: '自动+手动' }]"
              class="flex-1"
            />
          </div>
          <div class="row-field">
            <label class="lbl">结算周期</label>
            <Select
              v-model="form.config.settleType"
              :options="[{ value: '1', label: 'D+0' }, { value: '2', label: 'D+1' }]"
              class="flex-1"
            />
          </div>
          <div class="row-field">
            <label class="lbl">结算费率</label>
            <div class="flex flex-1 items-center gap-2">
              <input v-model="form.config.settleRate" placeholder="0.1" class="field-input flex-1" />
              <span class="text-sm text-muted-foreground">%</span>
            </div>
          </div>
        </div>

        <!-- 能力开关 -->
        <div class="border-t border-border/60 pt-3">
          <div class="mb-2 text-sm font-medium">功能权限</div>
          <div class="space-y-2.5">
            <div class="flex items-center justify-between">
              <span class="text-sm">余额充值</span>
              <Switch v-model="form.config.recharge" />
            </div>
            <div class="flex items-center justify-between">
              <span class="text-sm">代付功能</span>
              <Switch v-model="form.config.userTransfer" />
            </div>
            <div class="flex items-center justify-between">
              <span class="text-sm">邀请返现</span>
              <Switch v-model="form.config.inviteOpen" />
            </div>
            <div class="flex items-center justify-between">
              <span class="text-sm">商户保证金</span>
              <Switch v-model="form.config.userDeposit" />
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="drawer = false">取消</Button>
        <Button size="sm" :disabled="saving" @click="save">{{ editingGID !== null ? '保存' : '创建' }}</Button>
      </template>
    </Drawer>

    <!-- 删除确认 -->
    <Modal :model-value="!!delTarget" title="删除用户组" @update:model-value="(v) => { if (!v) delTarget = null }">
      <p class="text-sm text-muted-foreground">
        确定删除用户组 <b class="text-foreground">{{ delTarget?.name }}</b>（GID {{ delTarget?.gid }}）吗？
        该组下 <b class="text-foreground">{{ delTarget?.merchantCount }}</b> 个商户将回落默认用户组，此操作不可恢复。
      </p>
      <template #footer>
        <Button variant="outline" size="sm" @click="delTarget = null">取消</Button>
        <Button variant="destructive" size="sm" :disabled="deleting" @click="confirmDelete">删除</Button>
      </template>
    </Modal>
  </div>
</template>
