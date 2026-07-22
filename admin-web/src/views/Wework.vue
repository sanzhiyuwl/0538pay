<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Plus, MoreHorizontal, Pencil, Trash2, FlaskConical, RefreshCw } from 'lucide-vue-next'
import { Panel, Button, Badge, Drawer } from '@/components/ui'
import { shouldDropUp } from '@/composables/useRowMenu'
import { weworkAccounts, calcWeworkStats, type WeworkAccount } from '@/lib/mock/wework'

const list = ref([...weworkAccounts])
const stats = computed(() => calcWeworkStats(list.value))

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

function toggleStatus(w: WeworkAccount) {
  w.status = w.status === 1 ? 0 : 1
  openMenu.value = null
}

// 新增 / 编辑抽屉
const drawerOpen = ref(false)
const editing = ref(false)
const form = ref({ id: 0, name: '', appid: '', appsecret: '' })
function openAdd() {
  editing.value = false
  form.value = { id: 0, name: '', appid: '', appsecret: '' }
  drawerOpen.value = true
}
function openEdit(w: WeworkAccount) {
  editing.value = true
  form.value = { id: w.id, name: w.name, appid: w.appid, appsecret: '' }
  drawerOpen.value = true
  openMenu.value = null
}
function submit() {
  drawerOpen.value = false
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
                <button class="ml-1 text-xs text-primary hover:underline">
                  <RefreshCw class="inline size-3" />
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
                    <button class="menu-item" @click="openMenu = null">
                      <FlaskConical class="size-4 shrink-0 opacity-70" /><span class="flex-1">测试</span>
                    </button>
                    <div class="menu-sep" />
                    <button class="menu-item menu-item-danger" @click="openMenu = null">
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
        <Button size="sm" @click="submit">保存</Button>
      </template>
    </Drawer>
  </div>
</template>
