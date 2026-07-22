<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { Plus, MoreHorizontal, Pencil, Trash2, Monitor, Smartphone, MonitorSmartphone } from 'lucide-vue-next'
import { Panel, Button, Switch, Select, Drawer, Modal } from '@/components/ui'
import { shouldDropUp } from '@/composables/useRowMenu'
import { deviceText } from '@/lib/mock/paytypes'
import {
  fetchPayTypes,
  createPayType,
  updatePayType,
  setPayTypeStatus,
  deletePayType,
  type PayTypeView,
  type PayTypeSaveReq,
} from '@/lib/api/paycfg'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const types = ref<PayTypeView[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await fetchPayTypes()
    types.value = res.list
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载支付方式失败')
    types.value = []
  } finally {
    loading.value = false
  }
}
onMounted(load)

const stats = computed(() => ({
  total: types.value.length,
  open: types.value.filter((t) => t.status === 1).length,
  closed: types.value.filter((t) => t.status === 0).length,
}))

const deviceIcon: Record<number, any> = { 0: MonitorSmartphone, 1: Monitor, 2: Smartphone }
const deviceOptions = [
  { value: 0, label: 'PC + Mobile' },
  { value: 1, label: 'PC' },
  { value: 2, label: 'Mobile' },
]

async function toggleStatus(t: PayTypeView) {
  const next = t.status === 1 ? 0 : 1
  try {
    await setPayTypeStatus(t.id, next)
    t.status = next
    toast.success(next === 1 ? '已开启' : '已关闭')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  }
}

// ===== 新增/编辑 =====
const drawer = ref(false)
const editingID = ref<number | null>(null)
const saving = ref(false)
const form = reactive({ name: '', showname: '', device: 0 })

function openCreate() {
  editingID.value = null
  Object.assign(form, { name: '', showname: '', device: 0 })
  drawer.value = true
}
function openEdit(t: PayTypeView) {
  editingID.value = t.id
  Object.assign(form, { name: t.name, showname: t.showname, device: t.device })
  drawer.value = true
}
async function save() {
  if (!form.name.trim()) return toast.error('请填写调用值')
  if (!form.showname.trim()) return toast.error('请填写显示名称')
  const payload: PayTypeSaveReq = { name: form.name.trim(), showname: form.showname.trim(), device: form.device }
  saving.value = true
  try {
    if (editingID.value !== null) {
      await updatePayType(editingID.value, payload)
      toast.success('支付方式已更新')
    } else {
      await createPayType(payload)
      toast.success('支付方式已创建')
    }
    drawer.value = false
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

// ===== 删除 =====
const delTarget = ref<PayTypeView | null>(null)
const deleting = ref(false)
async function confirmDelete() {
  if (!delTarget.value) return
  deleting.value = true
  try {
    await deletePayType(delTarget.value.id)
    toast.success('支付方式已删除')
    delTarget.value = null
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '删除失败')
  } finally {
    deleting.value = false
  }
}

// 行操作菜单
const openMenu = ref<number | null>(null)
const dropUp = ref(false)
function toggleMenu(id: number, ev?: MouseEvent) {
  if (openMenu.value === id) { openMenu.value = null; return }
  openMenu.value = id
  dropUp.value = shouldDropUp(ev)
}
function closeMenu() { openMenu.value = null }
onMounted(() => window.addEventListener('click', closeMenu))
onUnmounted(() => window.removeEventListener('click', closeMenu))
</script>

<template>
  <div class="space-y-2.5">
    <!-- 概况 -->
    <Panel title="支付方式" subtitle="前端可选的收款方式，每个支付方式可关联多个支付通道">
      <template #actions>
        <Button size="sm" @click="openCreate"><Plus />新增支付方式</Button>
      </template>
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">方式总数</div>
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
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="支付方式列表" :subtitle="`${types.length} 个`">
      <div>
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[10%]">ID</th>
              <th class="w-[22%]">调用值</th>
              <th class="w-[28%]">显示名称</th>
              <th class="w-[20%]">支持设备</th>
              <th class="col-center w-[10%]">状态</th>
              <th class="col-center w-[10%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="t in types" :key="t.id">
              <td class="font-medium tabular-nums">{{ t.id }}</td>
              <td class="font-mono text-[13px] text-primary">{{ t.name }}</td>
              <td>
                <div class="flex items-center gap-2">
                  <span class="grid size-6 place-items-center rounded bg-primary/10 text-xs font-medium text-primary">
                    {{ t.showname[0] }}
                  </span>
                  {{ t.showname }}
                </div>
              </td>
              <td>
                <span class="inline-flex items-center gap-1.5 text-muted-foreground">
                  <component :is="deviceIcon[t.device]" class="size-4" />
                  {{ deviceText[t.device] }}
                </span>
              </td>
              <td class="col-center">
                <div class="flex justify-center">
                  <Switch :model-value="t.status === 1" size="sm" @update:model-value="toggleStatus(t)" />
                </div>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(t.id, $event)">
                    <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === t.id"
                    class="menu-panel absolute right-0 z-20 w-32"
                    :class="dropUp ? 'bottom-full mb-1.5' : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <button class="menu-item" @click="openEdit(t); openMenu = null">
                      <Pencil class="size-4 shrink-0 opacity-70" /><span class="flex-1">编辑</span>
                    </button>
                    <div class="menu-sep" />
                    <button
                      class="menu-item"
                      :class="t.id < 4 ? 'cursor-not-allowed opacity-40' : 'menu-item-danger'"
                      :disabled="t.id < 4"
                      :title="t.id < 4 ? '系统自带支付方式不支持删除' : ''"
                      @click="t.id >= 4 ? (delTarget = t, openMenu = null) : null"
                    >
                      <Trash2 class="size-4 shrink-0 opacity-70" /><span class="flex-1">删除</span>
                    </button>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="!types.length">
              <td colspan="6" class="py-10 text-center dim">暂无支付方式</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        系统自带的支付方式（ID &lt; 4）不支持删除；被支付通道引用的支付方式也不可删除。同一「调用值 + 支持设备」不能重复。
      </p>
    </Panel>

    <!-- 新增/编辑抽屉 -->
    <Drawer
      v-model="drawer"
      :title="editingID !== null ? '编辑支付方式' : '新增支付方式'"
      subtitle="调用值为英文标识（与支付文档一致），显示名称面向用户"
    >
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">调用值<span class="text-destructive">*</span></label>
          <input v-model="form.name" placeholder="如 alipay（字母数字）" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">显示名称<span class="text-destructive">*</span></label>
          <input v-model="form.showname" placeholder="如 支付宝" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">支持设备</label>
          <Select v-model="form.device" :options="deviceOptions" class="flex-1" />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="drawer = false">取消</Button>
        <Button size="sm" :disabled="saving" @click="save">{{ editingID !== null ? '保存' : '创建' }}</Button>
      </template>
    </Drawer>

    <!-- 删除确认 -->
    <Modal :model-value="!!delTarget" title="删除支付方式" @update:model-value="(v) => { if (!v) delTarget = null }">
      <p class="text-sm text-muted-foreground">
        确定删除支付方式 <b class="text-foreground">{{ delTarget?.showname }}</b>（{{ delTarget?.name }}）吗？此操作不可恢复。
      </p>
      <template #footer>
        <Button variant="outline" size="sm" @click="delTarget = null">取消</Button>
        <Button variant="destructive" size="sm" :disabled="deleting" @click="confirmDelete">删除</Button>
      </template>
    </Modal>
  </div>
</template>
