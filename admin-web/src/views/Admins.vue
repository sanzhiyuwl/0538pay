<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { Plus, MoreHorizontal, Pencil, Trash2, KeyRound, Power } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Drawer } from '@/components/ui'
import {
  admins as allAdmins,
  roles as allRoles,
  adminStatus,
  roleName,
  type Admin,
} from '@/lib/mock/admins'

const admins = ref<Admin[]>([...allAdmins])
const roleOptions = computed(() => allRoles.map((r) => ({ value: r.id, label: r.name })))

// ===== 行操作菜单 =====
const openMenu = ref<number | null>(null)
function toggleMenu(id: number) {
  openMenu.value = openMenu.value === id ? null : id
}
function closeMenu() {
  openMenu.value = null
}
onMounted(() => window.addEventListener('click', closeMenu))
onUnmounted(() => window.removeEventListener('click', closeMenu))

function toggleAdminStatus(a: Admin) {
  a.status = a.status === 1 ? 0 : 1
  openMenu.value = null
}

// ===== 管理员抽屉 =====
const adminDrawer = ref(false)
const editingAdminId = ref<number | null>(null)
const adminForm = reactive({ username: '', nickname: '', roleId: 2, password: '', status: 1 as 0 | 1 })
function openAdminCreate() {
  editingAdminId.value = null
  Object.assign(adminForm, { username: '', nickname: '', roleId: 2, password: '', status: 1 })
  adminDrawer.value = true
}
function openAdminEdit(a: Admin) {
  editingAdminId.value = a.id
  Object.assign(adminForm, { username: a.username, nickname: a.nickname, roleId: a.roleId, password: '', status: a.status })
  adminDrawer.value = true
  openMenu.value = null
}
function saveAdmin() {
  adminDrawer.value = false
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 管理员列表 -->
    <Panel title="管理员" :subtitle="`共 ${admins.length} 个账号`">
      <template #actions>
        <Button size="sm" @click="openAdminCreate"><Plus />新增管理员</Button>
      </template>
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[8%]">ID</th>
              <th class="w-[16%]">账号</th>
              <th class="w-[14%]">昵称</th>
              <th class="w-[14%]">角色</th>
              <th class="w-[16%]">最后登录 IP</th>
              <th class="w-[16%]">最后登录时间</th>
              <th class="col-center w-[8%]">状态</th>
              <th class="col-center w-[8%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(a, si) in admins" :key="a.id">
              <td class="tabular-nums dim">{{ a.id }}</td>
              <td class="font-mono text-[13px] font-medium">{{ a.username }}</td>
              <td>{{ a.nickname }}</td>
              <td><Badge variant="outline">{{ roleName(a.roleId) }}</Badge></td>
              <td class="tabular-nums text-xs">{{ a.lastLoginIp }}</td>
              <td class="text-xs">{{ a.lastLoginTime ?? '从未登录' }}</td>
              <td class="col-center">
                <Badge :variant="adminStatus[a.status].variant">{{ adminStatus[a.status].text }}</Badge>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(a.id)">
                    <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === a.id"
                    class="menu-panel absolute right-0 z-20 w-32"
                    :class="si >= admins.length - 3 && admins.length > 3 ? 'bottom-full mb-1.5' : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <button class="menu-item" @click="openAdminEdit(a)">
                      <Pencil class="size-4 shrink-0 opacity-70" /><span class="flex-1">编辑</span>
                    </button>
                    <button class="menu-item" @click="openMenu = null">
                      <KeyRound class="size-4 shrink-0 opacity-70" /><span class="flex-1">重置密码</span>
                    </button>
                    <button class="menu-item" @click="toggleAdminStatus(a)">
                      <Power class="size-4 shrink-0 opacity-70" />
                      <span class="flex-1">{{ a.status === 1 ? '禁用' : '启用' }}</span>
                    </button>
                    <div class="menu-sep" />
                    <button
                      class="menu-item"
                      :class="a.id === 1 ? 'cursor-not-allowed opacity-40' : 'menu-item-danger'"
                      :disabled="a.id === 1"
                      :title="a.id === 1 ? '超级管理员不可删除' : ''"
                      @click="openMenu = null"
                    >
                      <Trash2 class="size-4 shrink-0 opacity-70" /><span class="flex-1">删除</span>
                    </button>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="!admins.length">
              <td colspan="8" class="py-10 text-center dim">暂无管理员</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        管理员的功能权限由其所属「角色」决定，可在「角色管理」中配置。超级管理员（ID 1）不可删除。
      </p>
    </Panel>

    <!-- 管理员抽屉 -->
    <Drawer v-model="adminDrawer" :title="editingAdminId ? '编辑管理员' : '新增管理员'">
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
          <Select v-model="adminForm.roleId" :options="roleOptions" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <label class="w-20 shrink-0 text-right text-sm text-muted-foreground">{{ editingAdminId ? '重置密码' : '初始密码' }}</label>
          <input v-model="adminForm.password" type="password" :placeholder="editingAdminId ? '留空则不修改' : '登录密码'" class="field-input flex-1" />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="adminDrawer = false">取消</Button>
        <Button @click="saveAdmin">{{ editingAdminId ? '保存' : '创建' }}</Button>
      </template>
    </Drawer>
  </div>
</template>
