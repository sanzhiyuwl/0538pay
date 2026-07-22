<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { Plus, MoreHorizontal, Settings2, Pencil, Trash2, ArrowRight, Shuffle, ListOrdered, CircleCheck } from 'lucide-vue-next'
import { Panel, Button, Badge, Switch, Select, Drawer, Modal } from '@/components/ui'
import { shouldDropUp } from '@/composables/useRowMenu'
import { rollKind } from '@/lib/mock/rolls'
import { payTypes } from '@/lib/mock/orders'
import {
  fetchRolls,
  createRoll,
  updateRoll,
  setRollStatus,
  deleteRoll,
  type RollView,
  type RollChannelItem,
  type RollSaveReq,
} from '@/lib/api/rolls'
import { fetchChannels } from '@/lib/api/channels'
import type { Channel } from '@/lib/mock/channels'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const list = ref<RollView[]>([])
const loading = ref(false)
const allChannels = ref<Channel[]>([])

async function load() {
  loading.value = true
  try {
    const [rollRes, chRes] = await Promise.all([fetchRolls(), fetchChannels({ pageSize: 100 })])
    list.value = rollRes.list
    allChannels.value = chRes.list
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载轮询组失败')
    list.value = []
  } finally {
    loading.value = false
  }
}
onMounted(load)

// 轮询方式图标
const kindIcon: Record<number, any> = { 0: ListOrdered, 1: Shuffle, 2: CircleCheck }
const kindVariant: Record<number, 'default' | 'warning' | 'success'> = { 0: 'default', 1: 'warning', 2: 'success' }

const stats = computed(() => ({
  total: list.value.length,
  open: list.value.filter((r) => r.status === 1).length,
  closed: list.value.filter((r) => r.status === 0).length,
}))

const typeOptions = payTypes.map((t) => ({ value: t.id, label: t.showname }))
const kindOptions = [
  { value: 0, label: '按顺序依次轮询' },
  { value: 1, label: '按权重随机轮询' },
  { value: 2, label: '仅使用第一个已启用' },
]

// 当前支付方式下可选的通道
const channelsOfType = computed(() => allChannels.value.filter((c) => c.type === form.type))

// ===== 新增/编辑抽屉 =====
const drawer = ref(false)
const editingID = ref<number | null>(null)
const saving = ref(false)
const form = reactive({
  name: '',
  type: payTypes[0]?.id ?? 1,
  kind: 0,
  channels: [] as RollChannelItem[],
})

function resetForm() {
  Object.assign(form, { name: '', type: payTypes[0]?.id ?? 1, kind: 0, channels: [] as RollChannelItem[] })
}
function openCreate() {
  editingID.value = null
  resetForm()
  drawer.value = true
}
function openEdit(r: RollView) {
  editingID.value = r.id
  Object.assign(form, {
    name: r.name,
    type: r.type,
    kind: r.kind,
    channels: r.channels.map((c) => ({ ...c })),
  })
  drawer.value = true
}
function addChannel() {
  const first = channelsOfType.value[0]
  form.channels.push({ channel: first?.id ?? 0, channelname: first?.name ?? '', weight: 1 })
}
function removeChannel(i: number) {
  form.channels.splice(i, 1)
}

async function save() {
  if (!form.name.trim()) return toast.error('请填写显示名称')
  const channels = form.channels.filter((c) => c.channel > 0)
  const payload: RollSaveReq = {
    name: form.name.trim(),
    type: form.type,
    kind: form.kind,
    channels,
  }
  saving.value = true
  try {
    if (editingID.value !== null) {
      await updateRoll(editingID.value, payload)
      toast.success('轮询组已更新')
    } else {
      await createRoll(payload)
      toast.success('轮询组已创建')
    }
    drawer.value = false
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function toggleStatus(r: RollView) {
  const next = r.status === 1 ? 0 : 1
  try {
    await setRollStatus(r.id, next)
    r.status = next
    toast.success(next === 1 ? '已开启' : '已关闭')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  }
}

// ===== 删除 =====
const delTarget = ref<RollView | null>(null)
const deleting = ref(false)
async function confirmDelete() {
  if (!delTarget.value) return
  deleting.value = true
  try {
    await deleteRoll(delTarget.value.id)
    toast.success('轮询组已删除')
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
    <Panel title="通道轮询" subtitle="将多个同支付方式的通道编为轮询组，按顺序 / 权重 / 首个启用策略分流流量">
      <template #actions>
        <Button size="sm" @click="openCreate"><Plus />新增轮询组</Button>
      </template>
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">轮询组总数</div>
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
    <Panel title="轮询组列表" :subtitle="`${list.length} 个`">
      <div>
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[6%]">ID</th>
              <th class="w-[16%]">显示名称</th>
              <th class="w-[11%]">支付方式</th>
              <th class="w-[12%]">轮询方式</th>
              <th class="w-[32%]">轮询规则</th>
              <th class="col-center w-[8%]">状态</th>
              <th class="col-center w-[8%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in list" :key="r.id">
              <td class="font-medium tabular-nums">{{ r.id }}</td>
              <td class="truncate">{{ r.name }}</td>
              <td>{{ r.typeshowname }}</td>
              <td>
                <Badge :variant="kindVariant[r.kind]" class="inline-flex items-center gap-1">
                  <component :is="kindIcon[r.kind]" class="size-3" />
                  {{ rollKind[r.kind] }}
                </Badge>
              </td>
              <td>
                <div v-if="r.channels.length" class="flex flex-wrap items-center gap-1 text-[13px]">
                  <template v-for="(c, ci) in r.channels" :key="c.channel">
                    <span class="inline-flex items-center gap-0.5">
                      {{ c.channelname || `通道${c.channel}` }}
                      <span v-if="r.kind === 1" class="text-xs text-muted-foreground">({{ c.weight }})</span>
                    </span>
                    <ArrowRight v-if="r.kind !== 1 && ci < r.channels.length - 1" class="size-3 text-muted-foreground" />
                    <span v-else-if="ci < r.channels.length - 1" class="text-muted-foreground">/</span>
                  </template>
                </div>
                <span v-else class="dim">未配置通道</span>
              </td>
              <td class="col-center">
                <div class="flex justify-center">
                  <Switch :model-value="r.status === 1" size="sm" @update:model-value="toggleStatus(r)" />
                </div>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(r.id, $event)">
                    <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === r.id"
                    class="menu-panel absolute right-0 z-20 w-32"
                    :class="dropUp ? 'bottom-full mb-1.5' : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <button class="menu-item" @click="openEdit(r); openMenu = null">
                      <Settings2 class="size-4 shrink-0 opacity-70" /><span class="flex-1">配置通道</span>
                    </button>
                    <button class="menu-item" @click="openEdit(r); openMenu = null">
                      <Pencil class="size-4 shrink-0 opacity-70" /><span class="flex-1">编辑</span>
                    </button>
                    <div class="menu-sep" />
                    <button class="menu-item menu-item-danger" @click="delTarget = r; openMenu = null">
                      <Trash2 class="size-4 shrink-0 opacity-70" /><span class="flex-1">删除</span>
                    </button>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="!list.length">
              <td colspan="7" class="py-10 text-center dim">暂无轮询组</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        顺序轮询按配置顺序依次使用；随机轮询按权重(1-99)随机；首个启用仅用组内第一个可用通道。顺序 / 首个启用模式下权重值无效。
      </p>
    </Panel>

    <!-- 新增/编辑抽屉 -->
    <Drawer
      v-model="drawer"
      :title="editingID !== null ? '编辑轮询组' : '新增轮询组'"
      subtitle="配置支付方式、轮询策略与组内通道"
    >
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">显示名称<span class="text-destructive">*</span></label>
          <input v-model="form.name" placeholder="如：支付宝主力轮询" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">支付方式</label>
          <Select v-model="form.type" :options="typeOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">轮询方式</label>
          <Select v-model="form.kind" :options="kindOptions" class="flex-1" />
        </div>

        <!-- 组内通道 -->
        <div class="border-t border-border/60 pt-3">
          <div class="mb-2 flex items-center justify-between">
            <span class="text-sm font-medium">组内通道</span>
            <Button variant="outline" size="sm" :disabled="!channelsOfType.length" @click="addChannel"><Plus />添加通道</Button>
          </div>
          <div v-for="(c, i) in form.channels" :key="i" class="mb-2 flex items-center gap-2">
            <Select
              v-model="c.channel"
              :options="channelsOfType.map((ch) => ({ value: ch.id, label: ch.name }))"
              class="flex-1"
            />
            <div v-if="form.kind === 1" class="flex items-center gap-1">
              <input v-model.number="c.weight" type="number" min="1" max="99" placeholder="权重" class="field-input w-20" />
            </div>
            <button
              class="flex size-8 shrink-0 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-destructive"
              @click="removeChannel(i)"
            >
              <Trash2 class="size-4" />
            </button>
          </div>
          <p v-if="!channelsOfType.length" class="text-xs text-muted-foreground">该支付方式下暂无可用通道，请先在「支付通道」中添加。</p>
          <p v-else-if="!form.channels.length" class="text-xs text-muted-foreground">至少添加一个通道。顺序 / 首个启用模式下权重无效。</p>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="drawer = false">取消</Button>
        <Button size="sm" :disabled="saving" @click="save">{{ editingID !== null ? '保存' : '创建' }}</Button>
      </template>
    </Drawer>

    <!-- 删除确认 -->
    <Modal :model-value="!!delTarget" title="删除轮询组" @update:model-value="(v) => { if (!v) delTarget = null }">
      <p class="text-sm text-muted-foreground">
        确定删除轮询组 <b class="text-foreground">{{ delTarget?.name }}</b>（ID {{ delTarget?.id }}）吗？此操作不可恢复。
      </p>
      <template #footer>
        <Button variant="outline" size="sm" @click="delTarget = null">取消</Button>
        <Button variant="destructive" size="sm" :disabled="deleting" @click="confirmDelete">删除</Button>
      </template>
    </Modal>
  </div>
</template>
