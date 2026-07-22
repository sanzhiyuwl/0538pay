<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { Plus, MoreHorizontal, Pencil, Trash2, FlaskConical, MessageSquare, Smartphone } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Drawer, Modal } from '@/components/ui'
import { shouldDropUp } from '@/composables/useRowMenu'
import { weixinType } from '@/lib/mock/weixin'
import {
  fetchWeixins,
  createWeixin,
  updateWeixin,
  deleteWeixin,
  type WeixinView,
  type WeixinSaveReq,
} from '@/lib/api/paycfg'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const list = ref<WeixinView[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await fetchWeixins()
    list.value = res.list
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载配置失败')
    list.value = []
  } finally {
    loading.value = false
  }
}
onMounted(load)

const stats = computed(() => ({
  total: list.value.length,
  official: list.value.filter((w) => w.type === 0).length,
  mini: list.value.filter((w) => w.type === 1).length,
}))

const typeIcon: Record<number, any> = { 0: MessageSquare, 1: Smartphone }
const typeOptions = [
  { value: 0, label: '微信服务号' },
  { value: 1, label: '微信小程序' },
]

// ===== 新增/编辑 =====
const drawer = ref(false)
const editingID = ref<number | null>(null)
const saving = ref(false)
const form = reactive({ type: 0, name: '', appid: '', appsecret: '' })

function openCreate() {
  editingID.value = null
  Object.assign(form, { type: 0, name: '', appid: '', appsecret: '' })
  drawer.value = true
}
function openEdit(w: WeixinView) {
  editingID.value = w.id
  // appsecret 脱敏值不回填（留空表示不修改）
  Object.assign(form, { type: w.type, name: w.name, appid: w.appid, appsecret: '' })
  drawer.value = true
}
async function save() {
  if (!form.name.trim()) return toast.error('请填写名称')
  if (!form.appid.trim()) return toast.error('请填写 APPID')
  const payload: WeixinSaveReq = {
    type: form.type, name: form.name.trim(), appid: form.appid.trim(), appsecret: form.appsecret.trim(),
  }
  saving.value = true
  try {
    if (editingID.value !== null) {
      await updateWeixin(editingID.value, payload)
      toast.success('配置已更新')
    } else {
      await createWeixin(payload)
      toast.success('配置已创建')
    }
    drawer.value = false
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

function testWeixin() {
  toast.info('测试需调用微信接口换取 access_token，依赖真实 APPID/APPSECRET 凭证')
}

// ===== 删除 =====
const delTarget = ref<WeixinView | null>(null)
const deleting = ref(false)
async function confirmDelete() {
  if (!delTarget.value) return
  deleting.value = true
  try {
    await deleteWeixin(delTarget.value.id)
    toast.success('配置已删除')
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
    <Panel title="公众号 / 小程序" subtitle="配置微信服务号 / 小程序的 APPID 与密钥，用于 JSAPI 支付、网页授权等场景">
      <template #actions>
        <Button size="sm" @click="openCreate"><Plus />新增配置</Button>
      </template>
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">配置总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.total }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">微信服务号</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-primary">{{ stats.official }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">微信小程序</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ stats.mini }}</div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="配置列表" :subtitle="`${list.length} 个`">
      <div>
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[8%]">ID</th>
              <th class="w-[16%]">类别</th>
              <th class="w-[24%]">名称</th>
              <th class="w-[24%]">APPID</th>
              <th class="w-[18%]">APPSECRET</th>
              <th class="col-center w-[10%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="w in list" :key="w.id">
              <td class="font-medium tabular-nums">{{ w.id }}</td>
              <td>
                <Badge :variant="w.type === 1 ? 'success' : 'default'" class="inline-flex items-center gap-1">
                  <component :is="typeIcon[w.type]" class="size-3" />
                  {{ weixinType[w.type] }}
                </Badge>
              </td>
              <td class="truncate">{{ w.name }}</td>
              <td class="font-mono text-[13px] text-primary">{{ w.appid }}</td>
              <td class="font-mono text-xs dim">{{ w.appsecret || '—' }}</td>
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
                    <button class="menu-item" @click="openEdit(w); openMenu = null">
                      <Pencil class="size-4 shrink-0 opacity-70" /><span class="flex-1">编辑</span>
                    </button>
                    <button class="menu-item" @click="testWeixin(); openMenu = null">
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
              <td colspan="6" class="py-10 text-center dim">暂无公众号 / 小程序配置</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        服务号需在【公众平台→功能设置】配置网页授权域名，支付还需在商户平台设置 JSAPI 支付授权目录；小程序需在【小程序后台→开发设置】配置 request 合法域名并在微信支付后台绑定。
      </p>
    </Panel>

    <!-- 新增/编辑抽屉 -->
    <Drawer
      v-model="drawer"
      :title="editingID !== null ? '编辑配置' : '新增配置'"
      subtitle="APPSECRET 明文保存；编辑时留空表示不修改"
    >
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">类别</label>
          <Select v-model="form.type" :options="typeOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">名称<span class="text-destructive">*</span></label>
          <input v-model="form.name" placeholder="如 主商城服务号" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">APPID<span class="text-destructive">*</span></label>
          <input v-model="form.appid" placeholder="wx..." class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">APPSECRET</label>
          <input v-model="form.appsecret" :placeholder="editingID !== null ? '留空不修改' : '应用密钥'" class="field-input flex-1" />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="drawer = false">取消</Button>
        <Button size="sm" :disabled="saving" @click="save">{{ editingID !== null ? '保存' : '创建' }}</Button>
      </template>
    </Drawer>

    <!-- 删除确认 -->
    <Modal :model-value="!!delTarget" title="删除配置" @update:model-value="(v) => { if (!v) delTarget = null }">
      <p class="text-sm text-muted-foreground">
        确定删除 <b class="text-foreground">{{ delTarget?.name }}</b>（{{ delTarget?.appid }}）吗？此操作不可恢复。
      </p>
      <template #footer>
        <Button variant="outline" size="sm" @click="delTarget = null">取消</Button>
        <Button variant="destructive" size="sm" :disabled="deleting" @click="confirmDelete">删除</Button>
      </template>
    </Modal>
  </div>
</template>
