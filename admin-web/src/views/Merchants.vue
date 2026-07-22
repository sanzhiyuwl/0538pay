<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import {
  Search,
  RotateCcw,
  UserPlus,
  MoreHorizontal,
  Pencil,
  Wallet,
  KeyRound,
  Users,
  Trash2,
  CheckCircle2,
  XCircle,
  AlertCircle,
  Network,
  Plus,
  LogIn,
} from 'lucide-vue-next'
import { Panel, Button, Select, Pagination, Drawer, Modal, Switch } from '@/components/ui'
import {
  settleTypes,
  searchColumns,
  statusFilters,
  type Merchant,
} from '@/lib/mock/merchants'
import { formatMoney } from '@/lib/utils'
import {
  fetchMerchants,
  createMerchant,
  updateMerchant,
  rechargeMerchant,
  setMerchantGroup,
  setMerchantStatus,
  resetMerchantKey,
  deleteMerchant,
  ssoMerchant,
  type MerchantCreateReq,
  type MerchantEditReq,
} from '@/lib/api/merchants'
import { setMerchantToken } from '@/lib/api/client'
import { fetchGroups, type GroupView } from '@/lib/api/groups'
import {
  fetchSubChannels,
  createSubChannel,
  updateSubChannel,
  setSubChannelStatus,
  deleteSubChannel,
  type SubChannelView,
  type SubChannelSaveReq,
} from '@/lib/api/subchannels'
import { fetchChannels } from '@/lib/api/channels'
import type { Channel } from '@/lib/mock/channels'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { shouldDropUp } from '@/composables/useRowMenu'

const toast = useToast()

const columnOptions = searchColumns.map((c) => ({ value: c.value, label: c.label }))
const statusOptions = statusFilters.map((s) => ({ value: s.value, label: s.label }))

// 用户组下拉从后端拉取（真接口）
const groupList = ref<GroupView[]>([])
const groupOptions = computed(() => [
  { value: -1, label: '全部用户组' },
  { value: 0, label: '默认用户组' },
  ...groupList.value.map((g) => ({ value: g.gid, label: g.name })),
])
// 表单用（不含“全部”项，含默认组）
const groupFormOptions = computed(() => [
  { value: 0, label: '默认用户组' },
  ...groupList.value.map((g) => ({ value: g.gid, label: g.name })),
])
async function loadGroups() {
  try {
    const res = await fetchGroups()
    groupList.value = res.list
  } catch {
    groupList.value = []
  }
}

const settleIdOptions = [
  { value: 1, label: '支付宝' },
  { value: 2, label: '微信' },
  { value: 3, label: 'QQ钱包' },
  { value: 4, label: '银行卡' },
  { value: 5, label: '支付机构' },
]

// ===== 筛选 =====
const filters = ref({ column: 'uid', value: '', gid: -1, dstatus: '0' })

// ===== 数据源：从后端 API 加载 =====
const allMerchants = ref<Merchant[]>([])
const loading = ref(false)

async function loadMerchants() {
  loading.value = true
  try {
    const res = await fetchMerchants({ page: 1, pageSize: 100 })
    allMerchants.value = res.list
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载商户失败')
    allMerchants.value = []
  } finally {
    loading.value = false
  }
}

const filtered = computed(() =>
  allMerchants.value.filter((m) => {
    if (filters.value.gid > -1 && m.gid !== filters.value.gid) return false
    if (filters.value.dstatus !== '0') {
      const [field, val] = filters.value.dstatus.split('_')
      if (String((m as any)[field]) !== val) return false
    }
    if (filters.value.value.trim()) {
      const v = filters.value.value.trim()
      const f = (m as any)[filters.value.column]
      if (f == null || !String(f).includes(v)) return false
    }
    return true
  }),
)

function resetFilters() {
  filters.value = { column: 'uid', value: '', gid: -1, dstatus: '0' }
  page.value = 1
}

// ===== 分页 =====
const page = ref(1)
const pageSize = 15
const total = computed(() => filtered.value.length)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const safePage = computed(() => Math.min(page.value, pageCount.value))
const pageRows = computed(() => filtered.value.slice((safePage.value - 1) * pageSize, safePage.value * pageSize))
function go(p: number) {
  page.value = Math.min(Math.max(1, p), pageCount.value)
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
onMounted(() => {
  window.addEventListener('click', closeMenu)
  loadGroups()
  loadMerchants()
})
onUnmounted(() => window.removeEventListener('click', closeMenu))

function settlePrefix(m: Merchant) {
  return settleTypes[m.settle_id]?.prefix ?? ''
}

// ===== 新增/编辑抽屉 =====
const editDrawer = ref(false)
const editingUID = ref<number | null>(null)
const saving = ref(false)
const form = reactive({
  gid: 0,
  upid: 0,
  settle_id: 1,
  account: '',
  username: '',
  money: '',
  url: '',
  email: '',
  qq: '',
  phone: '',
  mode: 0,
  pay: 1,
  settle: 1,
  status: 1,
  password: '',
  ordername: '',
  open_code: 0,
  remain_money: '',
  deposit: '',
})

function resetForm() {
  Object.assign(form, {
    gid: 0, upid: 0, settle_id: 1, account: '', username: '', money: '',
    url: '', email: '', qq: '', phone: '', mode: 0, pay: 1, settle: 1, status: 1, password: '',
    ordername: '', open_code: 0, remain_money: '', deposit: '',
  })
}

function openCreate() {
  editingUID.value = null
  resetForm()
  editDrawer.value = true
}

function openEdit(m: Merchant) {
  editingUID.value = m.uid
  Object.assign(form, {
    gid: m.gid,
    upid: m.upid,
    settle_id: m.settle_id,
    account: m.account,
    username: m.username,
    money: m.money,
    url: m.url,
    email: m.email,
    qq: m.qq,
    phone: m.phone,
    mode: m.mode,
    pay: m.pay === 2 ? 1 : m.pay,
    settle: m.settle,
    status: m.status === 2 ? 1 : m.status,
    password: '',
    ordername: m.ordername ?? '',
    open_code: m.open_code ?? 0,
    remain_money: m.remain_money ?? '',
    deposit: m.deposit ?? '',
  })
  editDrawer.value = true
  openMenu.value = null
}

async function saveMerchant() {
  if (!form.phone.trim() && !form.email.trim()) {
    return toast.error('手机号和邮箱不能都为空')
  }
  saving.value = true
  try {
    if (editingUID.value) {
      const payload: MerchantEditReq = {
        gid: form.gid, upid: Number(form.upid) || 0, settle_id: form.settle_id,
        account: form.account.trim(), username: form.username.trim(), money: form.money.trim(),
        url: form.url.trim(), email: form.email.trim(), qq: form.qq.trim(), phone: form.phone.trim(),
        mode: form.mode, pay: form.pay, settle: form.settle, status: form.status,
        password: form.password.trim() || undefined,
        ordername: form.ordername.trim(), open_code: form.open_code,
        remain_money: form.remain_money.trim(), deposit: form.deposit.trim(),
      }
      await updateMerchant(editingUID.value, payload)
      toast.success('商户已更新')
    } else {
      const payload: MerchantCreateReq = {
        gid: form.gid, settle_id: form.settle_id,
        account: form.account.trim(), username: form.username.trim(),
        url: form.url.trim(), email: form.email.trim(), qq: form.qq.trim(), phone: form.phone.trim(),
        mode: form.mode, pay: form.pay, settle: form.settle, status: form.status,
        password: form.password.trim() || undefined,
      }
      const res = await createMerchant(payload)
      toast.success(`商户已创建：ID ${res.uid}，密钥 ${res.key}`)
    }
    editDrawer.value = false
    await loadMerchants()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

// ===== 充值/扣款弹窗 =====
const rechargeTarget = ref<Merchant | null>(null)
const rechargeForm = reactive({ action: 0, amount: '' })
const recharging = ref(false)
function openRecharge(m: Merchant) {
  rechargeTarget.value = m
  rechargeForm.action = 0
  rechargeForm.amount = ''
  openMenu.value = null
}
async function confirmRecharge() {
  if (!rechargeTarget.value) return
  const amt = rechargeForm.amount.trim()
  if (!amt || Number(amt) <= 0) return toast.error('请输入有效金额')
  recharging.value = true
  try {
    await rechargeMerchant(rechargeTarget.value.uid, rechargeForm.action, amt)
    toast.success(rechargeForm.action === 0 ? '充值成功' : '扣款成功')
    rechargeTarget.value = null
    await loadMerchants()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  } finally {
    recharging.value = false
  }
}

// ===== 子通道管理抽屉（对齐 epay uset.php?my=subchannel）=====
const subDrawer = ref(false)
const subMerchant = ref<Merchant | null>(null)
const subList = ref<SubChannelView[]>([])
const subChannels = ref<Channel[]>([])
const subLoading = ref(false)
const subEditingID = ref<number | null>(null)
const subSaving = ref(false)
const subForm = reactive({ channel: 0, name: '', info: '' })

async function openSubChannels(m: Merchant) {
  subMerchant.value = m
  subEditingID.value = null
  Object.assign(subForm, { channel: 0, name: '', info: '' })
  subDrawer.value = true
  openMenu.value = null
  await loadSubChannels()
  if (!subChannels.value.length) {
    try {
      const res = await fetchChannels({ pageSize: 100 })
      subChannels.value = res.list
    } catch { /* 忽略 */ }
  }
}
async function loadSubChannels() {
  if (!subMerchant.value) return
  subLoading.value = true
  try {
    const res = await fetchSubChannels(subMerchant.value.uid)
    subList.value = res.list
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载子通道失败')
    subList.value = []
  } finally {
    subLoading.value = false
  }
}
function subEdit(sc: SubChannelView) {
  subEditingID.value = sc.id
  Object.assign(subForm, { channel: sc.channel, name: sc.name, info: sc.info })
}
function subResetForm() {
  subEditingID.value = null
  Object.assign(subForm, { channel: subChannels.value[0]?.id ?? 0, name: '', info: '' })
}
async function subSave() {
  if (!subMerchant.value) return
  if (!subForm.name.trim()) return toast.error('请填写子通道名称')
  if (!subForm.channel) return toast.error('请选择归属主通道')
  const payload: SubChannelSaveReq = {
    channel: subForm.channel,
    name: subForm.name.trim(),
    info: subForm.info.trim(),
  }
  subSaving.value = true
  try {
    if (subEditingID.value !== null) {
      await updateSubChannel(subEditingID.value, payload)
      toast.success('子通道已更新')
    } else {
      await createSubChannel(subMerchant.value.uid, payload)
      toast.success('子通道已创建')
    }
    subResetForm()
    await loadSubChannels()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    subSaving.value = false
  }
}
async function subToggle(sc: SubChannelView) {
  const next = sc.status === 1 ? 0 : 1
  try {
    await setSubChannelStatus(sc.id, next)
    sc.status = next
    toast.success(next === 1 ? '已开启' : '已关闭')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  }
}
async function subDelete(sc: SubChannelView) {
  try {
    await deleteSubChannel(sc.id)
    toast.success('子通道已删除')
    if (subEditingID.value === sc.id) subResetForm()
    await loadSubChannels()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '删除失败')
  }
}

// ===== 修改用户组弹窗 =====
const groupTarget = ref<Merchant | null>(null)
const groupForm = reactive({ gid: 0, endtime: '' })
const savingGroup = ref(false)
function openGroup(m: Merchant) {
  groupTarget.value = m
  groupForm.gid = m.gid
  groupForm.endtime = m.endtime ? m.endtime.slice(0, 10) : ''
  openMenu.value = null
}
async function confirmGroup() {
  if (!groupTarget.value) return
  savingGroup.value = true
  try {
    await setMerchantGroup(groupTarget.value.uid, groupForm.gid, groupForm.endtime)
    toast.success('用户组已修改')
    groupTarget.value = null
    await loadMerchants()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  } finally {
    savingGroup.value = false
  }
}

// ===== 状态切换（正常/封禁、支付、结算）=====
async function toggle(m: Merchant, field: 'user' | 'pay' | 'settle' | 'refund' | 'transfer') {
  let next: number
  if (field === 'user') next = m.status === 1 ? 0 : 1
  else if (field === 'pay') next = m.pay === 1 ? 0 : 1
  else if (field === 'settle') next = m.settle === 1 ? 0 : 1
  else if (field === 'refund') next = (m.refund ?? 1) === 1 ? 0 : 1
  else next = (m.transfer ?? 0) === 1 ? 0 : 1
  try {
    await setMerchantStatus(m.uid, field, next)
    if (field === 'user') m.status = next as Merchant['status']
    else if (field === 'pay') m.pay = next as Merchant['pay']
    else if (field === 'settle') m.settle = next as Merchant['settle']
    else if (field === 'refund') m.refund = next as Merchant['refund']
    else m.transfer = next as Merchant['transfer']
    toast.success('已更新')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  }
  openMenu.value = null
}

// ===== 重置密钥 =====
const keyTarget = ref<Merchant | null>(null)
const resetting = ref(false)
function askResetKey(m: Merchant) {
  keyTarget.value = m
  openMenu.value = null
}
async function confirmResetKey() {
  if (!keyTarget.value) return
  resetting.value = true
  try {
    const res = await resetMerchantKey(keyTarget.value.uid)
    toast.success(`新密钥：${res.key}`)
    keyTarget.value = null
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '重置失败')
  } finally {
    resetting.value = false
  }
}

// ===== SSO 免密进入商户端 =====
async function ssoInto(m: Merchant) {
  openMenu.value = null
  try {
    const res = await ssoMerchant(m.uid)
    // 写商户端 token（与后台 token 隔离），新窗口打开商户中心工作台。
    setMerchantToken(res.token)
    window.open('/m', '_blank')
    toast.success(`已以商户 ${res.name} 身份登录商户端`)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '进入商户端失败')
  }
}

// ===== 删除商户 =====
const delTarget = ref<Merchant | null>(null)
const deleting = ref(false)
function askDelete(m: Merchant) {
  delTarget.value = m
  openMenu.value = null
}
async function confirmDelete() {
  if (!delTarget.value) return
  deleting.value = true
  try {
    await deleteMerchant(delTarget.value.uid)
    toast.success('商户已删除')
    delTarget.value = null
    await loadMerchants()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '删除失败')
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="商户管理" :subtitle="`共 ${total} 个商户`">
      <template #actions>
        <Button size="sm" @click="openCreate"><UserPlus />添加商户</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">商户信息</label>
          <Select v-model="filters.column" :options="columnOptions" class="w-28" />
          <input v-model="filters.value" placeholder="搜索内容" class="field-input w-48" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">用户组</label>
          <Select v-model="filters.gid" :options="groupOptions" class="w-32" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">状态</label>
          <Select v-model="filters.dstatus" :options="statusOptions" class="w-36" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="page = 1"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <Panel title="商户列表" :subtitle="`${total} 条`">
      <div>
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[14%]">商户号 / 用户组</th>
              <th class="w-[11%]">余额</th>
              <th class="w-[17%]">结算账号 / 姓名</th>
              <th class="w-[15%]">联系方式</th>
              <th class="w-[18%]">域名 / 添加时间</th>
              <th class="col-center w-[15%]">状态</th>
              <th class="col-center w-[10%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in pageRows" :key="m.uid">
              <td>
                <div class="font-medium">{{ m.uid }}</div>
                <div class="cursor-pointer truncate text-xs text-primary" @click="openGroup(m)">{{ m.groupname }}</div>
              </td>
              <td>
                <span class="font-semibold tabular-nums">
                  <span class="text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(m.money) }}
                </span>
              </td>
              <td>
                <template v-if="m.account">
                  <div class="truncate">
                    <span v-if="settlePrefix(m)" class="text-success">{{ settlePrefix(m) }}</span>{{ m.account }}
                  </div>
                  <div class="truncate text-xs dim">{{ m.username }}</div>
                </template>
                <span v-else class="dim">未设置</span>
              </td>
              <td>
                <div v-if="m.qq" class="truncate">QQ：{{ m.qq }}</div>
                <div class="truncate text-xs dim">{{ m.phone || m.email }}</div>
              </td>
              <td>
                <div class="truncate">{{ m.url }}</div>
                <div class="text-xs dim">{{ m.addtime }}</div>
              </td>
              <td class="col-center">
                <div class="flex items-center justify-center gap-3">
                  <span v-if="m.status === 1" class="inline-flex items-center gap-1 text-xs text-success"><CheckCircle2 class="size-3.5" />正常</span>
                  <span v-else-if="m.status === 2" class="inline-flex items-center gap-1 text-xs text-warning"><AlertCircle class="size-3.5" />未审核</span>
                  <span v-else class="inline-flex items-center gap-1 text-xs text-destructive"><XCircle class="size-3.5" />封禁</span>
                  <span :class="['inline-flex items-center gap-1 text-xs', m.cert === 1 ? 'text-success' : 'text-muted-foreground']">
                    <component :is="m.cert === 1 ? CheckCircle2 : XCircle" class="size-3.5" />实名
                  </span>
                </div>
                <div class="mt-1 flex items-center justify-center gap-3">
                  <span v-if="m.pay === 2" class="inline-flex items-center gap-1 text-xs text-warning"><AlertCircle class="size-3.5" />支付</span>
                  <span v-else :class="['inline-flex items-center gap-1 text-xs', m.pay === 1 ? 'text-success' : 'text-destructive']">
                    <component :is="m.pay === 1 ? CheckCircle2 : XCircle" class="size-3.5" />支付
                  </span>
                  <span :class="['inline-flex items-center gap-1 text-xs', m.settle === 1 ? 'text-success' : 'text-destructive']">
                    <component :is="m.settle === 1 ? CheckCircle2 : XCircle" class="size-3.5" />结算
                  </span>
                </div>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(m.uid, $event)">
                    操作 <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === m.uid"
                    class="menu-panel absolute right-0 z-20 w-40"
                    :class="dropUp ? 'bottom-full mb-1.5' : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <button class="menu-item" @click="openEdit(m)">
                      <Pencil class="size-4 shrink-0 opacity-70" /><span class="flex-1">编辑资料</span>
                    </button>
                    <button class="menu-item" @click="openRecharge(m)">
                      <Wallet class="size-4 shrink-0 opacity-70" /><span class="flex-1">充值 / 扣款</span>
                    </button>
                    <button class="menu-item" @click="openGroup(m)">
                      <Users class="size-4 shrink-0 opacity-70" /><span class="flex-1">修改用户组</span>
                    </button>
                    <button class="menu-item" @click="askResetKey(m)">
                      <KeyRound class="size-4 shrink-0 opacity-70" /><span class="flex-1">重置密钥</span>
                    </button>
                    <button class="menu-item" @click="openSubChannels(m)">
                      <Network class="size-4 shrink-0 opacity-70" /><span class="flex-1">子通道管理</span>
                    </button>
                    <button class="menu-item" @click="ssoInto(m)">
                      <LogIn class="size-4 shrink-0 opacity-70" /><span class="flex-1">进入商户端</span>
                    </button>
                    <div class="menu-sep" />
                    <button class="menu-item" @click="toggle(m, 'user')">
                      <component :is="m.status === 1 ? XCircle : CheckCircle2" class="size-4 shrink-0 opacity-70" />
                      <span class="flex-1">{{ m.status === 1 ? '封禁账号' : '解封账号' }}</span>
                    </button>
                    <button class="menu-item" @click="toggle(m, 'pay')">
                      <component :is="m.pay === 1 ? XCircle : CheckCircle2" class="size-4 shrink-0 opacity-70" />
                      <span class="flex-1">{{ m.pay === 1 ? '关闭支付' : '开启支付' }}</span>
                    </button>
                    <button class="menu-item" @click="toggle(m, 'settle')">
                      <component :is="m.settle === 1 ? XCircle : CheckCircle2" class="size-4 shrink-0 opacity-70" />
                      <span class="flex-1">{{ m.settle === 1 ? '关闭结算' : '开启结算' }}</span>
                    </button>
                    <button class="menu-item" @click="toggle(m, 'refund')">
                      <component :is="(m.refund ?? 1) === 1 ? XCircle : CheckCircle2" class="size-4 shrink-0 opacity-70" />
                      <span class="flex-1">{{ (m.refund ?? 1) === 1 ? '关闭退款API' : '开启退款API' }}</span>
                    </button>
                    <button class="menu-item" @click="toggle(m, 'transfer')">
                      <component :is="(m.transfer ?? 0) === 1 ? XCircle : CheckCircle2" class="size-4 shrink-0 opacity-70" />
                      <span class="flex-1">{{ (m.transfer ?? 0) === 1 ? '关闭代付API' : '开启代付API' }}</span>
                    </button>
                    <div class="menu-sep" />
                    <button class="menu-item menu-item-danger" @click="askDelete(m)">
                      <Trash2 class="size-4 shrink-0 opacity-70" /><span class="flex-1">删除商户</span>
                    </button>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="7" class="py-10 text-center dim">没有符合条件的商户</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="safePage" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>

    <!-- 新增/编辑商户抽屉（对齐 epay uset.php）-->
    <Drawer
      v-model="editDrawer"
      :title="editingUID ? '编辑商户' : '添加商户'"
      subtitle="配置结算账号、联系方式、用户组与权限"
    >
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">用户组</label>
          <Select v-model="form.gid" :options="groupFormOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">结算方式</label>
          <Select v-model="form.settle_id" :options="settleIdOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">结算账号</label>
          <input v-model="form.account" placeholder="收款账号" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">结算姓名</label>
          <input v-model="form.username" placeholder="真实姓名" class="field-input flex-1" />
        </div>
        <div v-if="editingUID" class="row-field">
          <label class="lbl">余额</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.money" placeholder="直接覆盖余额（不产生流水）" class="field-input flex-1" />
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">手机号</label>
          <input v-model="form.phone" placeholder="手机号与邮箱至少填一项" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">邮箱</label>
          <input v-model="form.email" placeholder="登录账号之一" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">QQ</label>
          <input v-model="form.qq" placeholder="选填" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">域名</label>
          <input v-model="form.url" placeholder="商户网站域名" class="field-input flex-1" />
        </div>
        <div v-if="editingUID" class="row-field">
          <label class="lbl">邀请方 UID</label>
          <input v-model.number="form.upid" type="number" placeholder="0 为无" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">扣费模式</label>
          <Select
            v-model="form.mode"
            :options="[{ value: 0, label: '余额扣费' }, { value: 1, label: '订单加费' }]"
            class="flex-1"
          />
        </div>
        <div class="row-field">
          <label class="lbl">支付权限</label>
          <Select v-model="form.pay" :options="[{ value: 1, label: '开启' }, { value: 0, label: '关闭' }]" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">结算权限</label>
          <Select v-model="form.settle" :options="[{ value: 1, label: '开启' }, { value: 0, label: '关闭' }]" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">账号状态</label>
          <Select v-model="form.status" :options="[{ value: 1, label: '正常' }, { value: 0, label: '封禁' }]" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">登录密码</label>
          <input v-model="form.password" type="password" :placeholder="editingUID ? '留空则不修改' : '选填，商户可用密码登录'" class="field-input flex-1" />
        </div>

        <!-- 进阶字段（仅编辑时；对齐 epay ajax_user.php edit）-->
        <template v-if="editingUID">
          <div class="row-field">
            <label class="lbl">订单名模板</label>
            <input v-model="form.ordername" placeholder="回传给下游的商品名，留空用原始名" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">聚合收款</label>
            <Select v-model="form.open_code" :options="[{ value: 0, label: '关闭' }, { value: 1, label: '开启' }]" class="flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">预留余额</label>
            <div class="flex flex-1 items-center gap-2">
              <input v-model="form.remain_money" placeholder="自动结算不参与的金额，留空为 0" class="field-input flex-1" />
            </div>
          </div>
          <div class="row-field">
            <label class="lbl">保证金</label>
            <div class="flex flex-1 items-center gap-2">
              <input v-model="form.deposit" placeholder="商户保证金，留空为 0" class="field-input flex-1" />
            </div>
          </div>
        </template>

        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          手机号和邮箱不能都为空，且各自不可与已有商户重复。新增成功后系统自动生成 32 位通信密钥。
        </p>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="editDrawer = false">取消</Button>
        <Button size="sm" :disabled="saving" @click="saveMerchant">{{ editingUID ? '保存' : '创建' }}</Button>
      </template>
    </Drawer>

    <!-- 充值/扣款弹窗 -->
    <Modal :model-value="!!rechargeTarget" title="余额充值 / 扣款" @update:model-value="(v) => { if (!v) rechargeTarget = null }">
      <div class="space-y-3.5">
        <p class="text-sm text-muted-foreground">
          商户 <b class="text-foreground">{{ rechargeTarget?.uid }}</b>，当前余额
          <b class="text-foreground tabular-nums">¥{{ rechargeTarget ? formatMoney(rechargeTarget.money) : '0.00' }}</b>
        </p>
        <div class="row-field">
          <label class="lbl">操作类型</label>
          <Select v-model="rechargeForm.action" :options="[{ value: 0, label: '充值（加款）' }, { value: 1, label: '扣除（扣款）' }]" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">金额</label>
          <div class="flex flex-1 items-center gap-2">
            <span class="text-sm text-muted-foreground">¥</span>
            <input v-model="rechargeForm.amount" placeholder="如 100.00" class="field-input flex-1" />
          </div>
        </div>
        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          充值/扣款均产生资金流水。扣款金额若超过当前余额，将自动截断为当前余额。
        </p>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="rechargeTarget = null">取消</Button>
        <Button size="sm" :disabled="recharging" @click="confirmRecharge">确定</Button>
      </template>
    </Modal>

    <!-- 修改用户组弹窗 -->
    <Modal :model-value="!!groupTarget" title="修改用户组" @update:model-value="(v) => { if (!v) groupTarget = null }">
      <div class="space-y-3.5">
        <p class="text-sm text-muted-foreground">商户 <b class="text-foreground">{{ groupTarget?.uid }}</b></p>
        <div class="row-field">
          <label class="lbl">用户组</label>
          <Select v-model="groupForm.gid" :options="groupFormOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">到期时间</label>
          <input v-model="groupForm.endtime" type="date" class="field-input flex-1" />
        </div>
        <p class="rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          到期时间留空表示永久有效。到期后该商户自动回落默认用户组。
        </p>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="groupTarget = null">取消</Button>
        <Button size="sm" :disabled="savingGroup" @click="confirmGroup">保存</Button>
      </template>
    </Modal>

    <!-- 重置密钥确认 -->
    <Modal :model-value="!!keyTarget" title="重置通信密钥" @update:model-value="(v) => { if (!v) keyTarget = null }">
      <p class="text-sm text-muted-foreground">
        确定重置商户 <b class="text-foreground">{{ keyTarget?.uid }}</b> 的 MD5 通信密钥吗？
        原密钥将立即失效，商户对接代码需同步更新新密钥。
      </p>
      <template #footer>
        <Button variant="outline" size="sm" @click="keyTarget = null">取消</Button>
        <Button variant="destructive" size="sm" :disabled="resetting" @click="confirmResetKey">重置</Button>
      </template>
    </Modal>

    <!-- 删除确认 -->
    <Modal :model-value="!!delTarget" title="删除商户" @update:model-value="(v) => { if (!v) delTarget = null }">
      <p class="text-sm text-muted-foreground">
        确定删除商户 <b class="text-foreground">{{ delTarget?.uid }}</b> 吗？此操作不可恢复。
      </p>
      <template #footer>
        <Button variant="outline" size="sm" @click="delTarget = null">取消</Button>
        <Button variant="destructive" size="sm" :disabled="deleting" @click="confirmDelete">删除</Button>
      </template>
    </Modal>

    <!-- 子通道管理抽屉 -->
    <Drawer
      v-model="subDrawer"
      :title="`子通道管理 · 商户 ${subMerchant?.uid ?? ''}`"
      subtitle="子通道为该商户在某主通道下的一份参数覆盖，用户组分配为「用户自定义子通道」时按顺序调度使用"
    >
      <div class="space-y-3.5">
        <!-- 已有子通道列表 -->
        <div>
          <div class="mb-2 text-sm font-medium">已有子通道</div>
          <table class="tbl w-full">
            <thead>
              <tr>
                <th>名称</th>
                <th>归属通道</th>
                <th class="col-center w-16">状态</th>
                <th class="col-center w-20">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="sc in subList" :key="sc.id">
                <td class="truncate">{{ sc.name }}</td>
                <td class="text-[13px] text-muted-foreground">{{ sc.channelname || `通道${sc.channel}` }}</td>
                <td class="col-center">
                  <div class="flex justify-center">
                    <Switch :model-value="sc.status === 1" size="sm" @update:model-value="subToggle(sc)" />
                  </div>
                </td>
                <td class="col-center">
                  <div class="flex justify-center gap-1">
                    <button class="text-muted-foreground hover:text-foreground" title="编辑" @click="subEdit(sc)">
                      <Pencil class="size-4" />
                    </button>
                    <button class="text-muted-foreground hover:text-destructive" title="删除" @click="subDelete(sc)">
                      <Trash2 class="size-4" />
                    </button>
                  </div>
                </td>
              </tr>
              <tr v-if="!subLoading && !subList.length">
                <td colspan="4" class="py-6 text-center dim">暂无子通道</td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 新增/编辑表单 -->
        <div class="border-t border-border/60 pt-3 space-y-3.5">
          <div class="text-sm font-medium">{{ subEditingID !== null ? '编辑子通道' : '新增子通道' }}</div>
          <div class="row-field">
            <label class="lbl">归属主通道<span class="text-destructive">*</span></label>
            <Select
              v-model="subForm.channel"
              :options="subChannels.map((c) => ({ value: c.id, label: `${c.name}（${c.typeshowname}）` }))"
              class="flex-1"
            />
          </div>
          <div class="row-field">
            <label class="lbl">子通道名称<span class="text-destructive">*</span></label>
            <input v-model="subForm.name" placeholder="同一商户内唯一" class="field-input flex-1" />
          </div>
          <div>
            <label class="mb-1.5 block text-sm">自定义参数（JSON，可空）</label>
            <textarea
              v-model="subForm.info"
              rows="4"
              placeholder='如 {"appid":"xxx"}，替换主通道 config 中形如 [appid] 的占位变量'
              class="field-input w-full font-mono text-xs"
            />
          </div>
          <div class="flex items-center gap-2">
            <Button size="sm" :disabled="subSaving" @click="subSave">
              <Plus v-if="subEditingID === null" />{{ subEditingID !== null ? '保存修改' : '添加子通道' }}
            </Button>
            <Button v-if="subEditingID !== null" variant="outline" size="sm" @click="subResetForm">取消编辑</Button>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="subDrawer = false">关闭</Button>
      </template>
    </Drawer>
  </div>
</template>
