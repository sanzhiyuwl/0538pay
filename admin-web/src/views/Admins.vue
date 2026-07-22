<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { Plus, MoreHorizontal, Pencil, Trash2, Power } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Drawer } from '@/components/ui'
import { shouldDropUp } from '@/composables/useRowMenu'
import { adminStatus, roleOptions, roleLabel } from '@/lib/mock/admins'
import {
  fetchAdmins, createAdmin, updateAdmin, setAdminStatus, deleteAdmin,
  type AdminAccount,
} from '@/lib/api/admins'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const admins = ref<AdminAccount[]>([])

async function load() {
  try {
    admins.value = await fetchAdmins()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载管理员失败')
    admins.value = []
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
onMounted(() => { load(); window.addEventListener('click', closeMenu) })
onUnmounted(() => window.removeEventListener('click', closeMenu))

async function toggleAdminStatus(a: AdminAccount) {
  openMenu.value = null
  try {
    await setAdminStatus(a.id, a.status === 1 ? 0 : 1)
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  }
}

async function removeAdmin(a: AdminAccount) {
  openMenu.value = null
  if (a.role === 'super') return
  if (!confirm(`确认删除管理员「${a.username}」？`)) return
  try {
    await deleteAdmin(a.id)
    toast.success('已删除')
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '删除失败')
  }
}

// ===== 管理员抽屉 =====
const adminDrawer = ref(false)
const editingId = ref<number | null>(null)
const editingSuper = ref(false)
const adminForm = reactive({ username: '', nickname: '', role: 'admin', password: '', status: 1 })
const saving = ref(false)
function openAdminCreate() {
  editingId.value = null
  editingSuper.value = false
  Object.assign(adminForm, { username: '', nickname: '', role: 'admin', password: '', status: 1 })
  adminDrawer.value = true
}
function openAdminEdit(a: AdminAccount) {
  editingId.value = a.id
  editingSuper.value = a.role === 'super'
  Object.assign(adminForm, { username: a.username, nickname: a.nickname, role: a.role, password: '', status: a.status })
  adminDrawer.value = true
  openMenu.value = null
}
async function saveAdmin() {
  if (saving.value) return
  if (!adminForm.username.trim()) return toast.error('请输入账号')
  if (editingId.value === null && adminForm.password.length < 6) return toast.error('初始密码至少 6 位')
  saving.value = true
  try {
    const body = { ...adminForm }
    if (editingId.value === null) await createAdmin(body)
    else await updateAdmin(editingId.value, body)
    toast.success(editingId.value === null ? '管理员已创建' : '已保存')
    adminDrawer.value = false
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 管理员列表 -->
    <Panel title="管理员" :subtitle="`共 ${admins.length} 个账号`">
      <template #actions>
        <Button size="sm" @click="openAdminCreate"><Plus />新增管理员</Button>
      </template>
      <div>
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[8%]">ID</th>
              <th class="w-[16%]">账号</th>
              <th class="w-[14%]">昵称</th>
              <th class="w-[14%]">角色</th>
              <th class="w-[32%]">最后登录</th>
              <th class="col-center w-[8%]">状态</th>
              <th class="col-center w-[8%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in admins" :key="a.id">
              <td class="tabular-nums dim">{{ a.id }}</td>
              <td class="font-mono text-[13px] font-medium">{{ a.username }}</td>
              <td>{{ a.nickname }}</td>
              <td><Badge variant="outline">{{ roleLabel(a.role) }}</Badge></td>
              <td class="text-xs">{{ a.last_login || '从未登录' }}</td>
              <td class="col-center">
                <Badge :variant="adminStatus[a.status].variant">{{ adminStatus[a.status].text }}</Badge>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(a.id, $event)">
                    <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === a.id"
                    class="menu-panel absolute right-0 z-20 w-32"
                    :class="dropUp ? 'bottom-full mb-1.5' : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <button class="menu-item" @click="openAdminEdit(a)">
                      <Pencil class="size-4 shrink-0 opacity-70" /><span class="flex-1">编辑</span>
                    </button>
                    <button
                      class="menu-item"
                      :class="a.role === 'super' ? 'cursor-not-allowed opacity-40' : ''"
                      :disabled="a.role === 'super'"
                      :title="a.role === 'super' ? '超级管理员不可停用' : ''"
                      @click="a.role !== 'super' && toggleAdminStatus(a)"
                    >
                      <Power class="size-4 shrink-0 opacity-70" />
                      <span class="flex-1">{{ a.status === 1 ? '禁用' : '启用' }}</span>
                    </button>
                    <div class="menu-sep" />
                    <button
                      class="menu-item"
                      :class="a.role === 'super' ? 'cursor-not-allowed opacity-40' : 'menu-item-danger'"
                      :disabled="a.role === 'super'"
                      :title="a.role === 'super' ? '超级管理员不可删除' : ''"
                      @click="removeAdmin(a)"
                    >
                      <Trash2 class="size-4 shrink-0 opacity-70" /><span class="flex-1">删除</span>
                    </button>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="!admins.length">
              <td colspan="7" class="py-10 text-center dim">暂无管理员</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        管理员的功能权限由其所属「角色」决定，可在「角色管理」中查看角色定义。超级管理员不可停用或删除。
      </p>
    </Panel>

    <!-- 管理员抽屉 -->
    <Drawer v-model="adminDrawer" :title="editingId ? '编辑管理员' : '新增管理员'">
      <div class="space-y-3.5">
        <div class="flex items-center gap-3">
          <label class="w-20 shrink-0 text-right text-sm text-muted-foreground">账号</label>
          <input v-model="adminForm.username" placeholder="登录账号" class="field-input flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <label class="w-20 shrink-0 text-right text-sm text-muted-foreground">昵称</label>
          <input v-model="adminForm.nickname" placeholder="显示名称" class="field-input flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <label class="w-20 shrink-0 text-right text-sm text-muted-foreground">角色</label>
          <Select v-if="!editingSuper" v-model="adminForm.role" :options="roleOptions" class="flex-1" />
          <span v-else class="flex-1 text-sm text-muted-foreground">超级管理员（角色锁定）</span>
        </div>
        <div class="flex items-center gap-3">
          <label class="w-20 shrink-0 text-right text-sm text-muted-foreground">状态</label>
          <Select
            v-model="adminForm.status"
            :options="[{ value: 1, label: '正常' }, { value: 0, label: '停用' }]"
            :disabled="editingSuper"
            class="flex-1"
          />
        </div>
        <div class="flex items-center gap-3">
          <label class="w-20 shrink-0 text-right text-sm text-muted-foreground">{{ editingId ? '重置密码' : '初始密码' }}</label>
          <input v-model="adminForm.password" type="password" :placeholder="editingId ? '留空则不修改' : '登录密码(至少6位)'" class="field-input flex-1" />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="adminDrawer = false">取消</Button>
        <Button :disabled="saving" @click="saveAdmin">{{ editingId ? '保存' : '创建' }}</Button>
      </template>
    </Drawer>
  </div>
</template>
