<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Plus, MoreHorizontal, Pencil, Trash2, FlaskConical, RefreshCw } from 'lucide-vue-next'
import { Panel, Button, Badge, Drawer, Modal } from '@/components/ui'
import { shouldDropUp } from '@/composables/useRowMenu'
import {
  fetchWeworks,
  createWework,
  updateWework,
  setWeworkStatus,
  deleteWework,
  refreshWeworkKf,
  type WeworkView,
  type WeworkSaveReq,
} from '@/lib/api/paycfg'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const list = ref<WeworkView[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await fetchWeworks()
    list.value = res.list
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载企业微信失败')
    list.value = []
  } finally {
    loading.value = false
  }
}
onMounted(load)

const stats = computed(() => ({
  total: list.value.length,
  active: list.value.filter((w) => w.status === 1).length,
  kfTotal: list.value.reduce((a, w) => a + w.kfnum, 0),
}))

// 行操作菜单
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
onMounted(() => window.addEventListener('click', closeMenu))
onUnmounted(() => window.removeEventListener('click', closeMenu))

async function toggleStatus(w: WeworkView) {
  const next = w.status === 1 ? 0 : 1
  try {
    await setWeworkStatus(w.id, next)
    w.status = next
    toast.success(next === 1 ? '已开启' : '已关闭')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  }
}

const refreshingId = ref(0)
async function refreshKf(w: WeworkView) {
  if (refreshingId.value) return
  refreshingId.value = w.id
  try {
    const r = await refreshWeworkKf(w.id)
    w.kfnum = r.count
    toast.success(`已同步 ${r.count} 个客服账号`)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '刷新失败')
  } finally {
    refreshingId.value = 0
  }
}
function testWework() {
  toast.info('测试需调用企业微信接口换取 access_token，依赖真实凭证')
}

// 新增 / 编辑抽屉
const drawerOpen = ref(false)
const editing = ref(false)
const saving = ref(false)
const form = ref({ id: 0, name: '', appid: '', appsecret: '' })
function openAdd() {
  editing.value = false
  form.value = { id: 0, name: '', appid: '', appsecret: '' }
  drawerOpen.value = true
}
function openEdit(w: WeworkView) {
  editing.value = true
  form.value = { id: w.id, name: w.name, appid: w.appid, appsecret: '' }
  drawerOpen.value = true
  openMenu.value = null
}
async function submit() {
  if (!form.value.name.trim()) return toast.error('请填写名称')
  if (!form.value.appid.trim()) return toast.error('请填写企业ID')
  const payload: WeworkSaveReq = {
    name: form.value.name.trim(), appid: form.value.appid.trim(), appsecret: form.value.appsecret.trim(),
  }
  saving.value = true
  try {
    if (editing.value) {
      await updateWework(form.value.id, payload)
      toast.success('企业微信已更新')
    } else {
      await createWework(payload)
      toast.success('企业微信已创建')
    }
    drawerOpen.value = false
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

// 删除
const delTarget = ref<WeworkView | null>(null)
const deleting = ref(false)
async function confirmDelete() {
  if (!delTarget.value) return
  deleting.value = true
  try {
    await deleteWework(delTarget.value.id)
    toast.success('企业微信已删除，关联客服账号一并清除')
    delTarget.value = null
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '删除失败')
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 概况 -->
    <Panel title="企业微信账号" subtitle="配置企业微信应用，用于 H5 跳转微信客服支付">
      <template #actions>
        <Button size="sm" @click="openAdd"><Plus />新增企业微信</Button>
      </template>
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">企业微信总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.total }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已开启</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ stats.active }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">客服账号总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-primary">{{ stats.kfTotal }}</div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="账号列表" :subtitle="`${list.length} 个`">
      <div>
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[8%]">ID</th>
              <th class="w-[24%]">名称</th>
              <th class="w-[26%]">企业ID</th>
              <th class="num w-[14%]">客服账号数</th>
              <th class="col-center w-[12%]">状态</th>
              <th class="col-center w-[10%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="w in list" :key="w.id">
              <td class="font-medium tabular-nums">{{ w.id }}</td>
              <td class="truncate">{{ w.name }}</td>
              <td class="font-mono text-[13px] text-primary">{{ w.appid }}</td>
              <td class="num tabular-nums">
                {{ w.kfnum }}
                <button class="ml-1 text-xs text-primary hover:underline disabled:opacity-50" title="刷新客服账号" :disabled="refreshingId === w.id" @click="refreshKf(w)">
                  <RefreshCw class="inline size-3" :class="refreshingId === w.id ? 'animate-spin' : ''" />
                </button>
              </td>
              <td class="col-center">
                <button @click="toggleStatus(w)">
                  <Badge :variant="w.status === 1 ? 'success' : 'warning'">
                    {{ w.status === 1 ? '已开启' : '已关闭' }}
                  </Badge>
                </button>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(w.id, $event)">
                    <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === w.id"
                    class="menu-panel absolute right-0 z-20 w-32"
                    :class="dropUp ? 'bottom-full mb-1.5' : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <button class="menu-item" @click="openEdit(w)">
                      <Pencil class="size-4 shrink-0 opacity-70" /><span class="flex-1">编辑</span>
                    </button>
                    <button class="menu-item" @click="testWework(); openMenu = null">
                      <FlaskConical class="size-4 shrink-0 opacity-70" /><span class="flex-1">测试</span>
                    </button>
                    <div class="menu-sep" />
                    <button class="menu-item menu-item-danger" @click="delTarget = w; openMenu = null">
                      <Trash2 class="size-4 shrink-0 opacity-70" /><span class="flex-1">删除</span>
                    </button>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="!list.length">
              <td colspan="6" class="py-10 text-center dim">暂无企业微信账号</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        企业微信需完成企业认证，否则接待客户数量受限。如需在一个企业下添加多个客服账号，需先在开发配置停用 API 后再添加。
      </p>
    </Panel>

    <!-- 新增/编辑抽屉 -->
    <Drawer v-model="drawerOpen" :title="editing ? '修改企业微信' : '新增企业微信'" width="max-w-md">
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">名称</label>
          <input v-model="form.name" placeholder="仅用于显示，不要重复" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">企业ID</label>
          <input v-model="form.appid" placeholder="corpid（ww 开头）" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">Secret</label>
          <input v-model="form.appsecret" :placeholder="editing ? '不修改请留空' : '应用 Secret'" class="field-input flex-1" />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="drawerOpen = false">取消</Button>
        <Button size="sm" :disabled="saving" @click="submit">保存</Button>
      </template>
    </Drawer>

    <!-- 删除确认 -->
    <Modal :model-value="!!delTarget" title="删除企业微信" @update:model-value="(v) => { if (!v) delTarget = null }">
      <p class="text-sm text-muted-foreground">
        确定删除 <b class="text-foreground">{{ delTarget?.name }}</b>（{{ delTarget?.appid }}）吗？其下
        <b class="text-foreground">{{ delTarget?.kfnum }}</b> 个客服账号将一并删除，此操作不可恢复。
      </p>
      <template #footer>
        <Button variant="outline" size="sm" @click="delTarget = null">取消</Button>
        <Button variant="destructive" size="sm" :disabled="deleting" @click="confirmDelete">删除</Button>
      </template>
    </Modal>
  </div>
</template>
