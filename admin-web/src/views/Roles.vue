<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { Plus, Pencil, Trash2, ShieldCheck } from 'lucide-vue-next'
import { Panel, Button, Badge, Drawer } from '@/components/ui'
import { roles as allRoles, permModules, rolePermText, type Role } from '@/lib/mock/admins'

const roles = ref<Role[]>([...allRoles])

// ===== 角色权限抽屉 =====
const roleDrawer = ref(false)
const editingRoleId = ref<number | null>(null)
const roleForm = reactive({ name: '', desc: '', permissions: [] as string[] })

function openRoleCreate() {
  editingRoleId.value = null
  Object.assign(roleForm, { name: '', desc: '', permissions: ['dashboard'] })
  roleDrawer.value = true
}
function openRoleEdit(r: Role) {
  editingRoleId.value = r.id
  Object.assign(roleForm, {
    name: r.name,
    desc: r.desc,
    permissions: r.permissions.includes('*') ? permModules.map((m) => m.key) : [...r.permissions],
  })
  roleDrawer.value = true
}
function togglePerm(key: string) {
  const i = roleForm.permissions.indexOf(key)
  if (i > -1) roleForm.permissions.splice(i, 1)
  else roleForm.permissions.push(key)
}
function saveRole() {
  roleDrawer.value = false
}

const isSuperAdminRole = computed(() => editingRoleId.value === 1)
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="角色管理" :subtitle="`${roles.length} 个角色`">
      <template #actions>
        <Button size="sm" @click="openRoleCreate"><Plus />新增角色</Button>
      </template>
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[14%]">角色</th>
              <th class="w-[22%]">说明</th>
              <th class="w-[44%]">权限范围</th>
              <th class="col-center w-[20%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in roles" :key="r.id">
              <td>
                <div class="flex items-center gap-1.5 font-medium">
                  <ShieldCheck class="size-4 shrink-0 text-primary" />{{ r.name }}
                  <Badge v-if="r.builtin" variant="muted" class="ml-1">内置</Badge>
                </div>
              </td>
              <td class="text-xs dim">{{ r.desc }}</td>
              <td>
                <span v-if="r.permissions.includes('*')" class="text-sm text-primary">全部权限</span>
                <span v-else class="text-[13px]">{{ rolePermText(r) }}</span>
              </td>
              <td class="col-center">
                <div class="flex items-center justify-center gap-1">
                  <Button variant="ghost" size="sm" @click="openRoleEdit(r)"><Pencil class="size-4" />权限</Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    :class="r.builtin ? 'cursor-not-allowed opacity-40' : 'text-destructive hover:text-destructive'"
                    :disabled="r.builtin"
                    :title="r.builtin ? '内置角色不可删除' : ''"
                  >
                    <Trash2 class="size-4" />
                  </Button>
                </div>
              </td>
            </tr>
            <tr v-if="!roles.length">
              <td colspan="4" class="py-10 text-center dim">暂无角色</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        角色决定管理员可访问的功能模块。超级管理员拥有全部权限且不可删除；内置角色可改权限但不可删除。
      </p>
    </Panel>

    <!-- 角色权限抽屉 -->
    <Drawer v-model="roleDrawer" :title="editingRoleId ? '编辑角色权限' : '新增角色'" subtitle="勾选该角色可访问的功能模块">
      <div class="space-y-4">
        <div class="flex items-center gap-3">
          <label class="w-16 shrink-0 text-right text-sm text-muted-foreground">角色名</label>
          <input v-model="roleForm.name" :disabled="isSuperAdminRole" placeholder="角色名称" class="field-input flex-1 disabled:opacity-60" />
        </div>
        <div class="flex items-center gap-3">
          <label class="w-16 shrink-0 text-right text-sm text-muted-foreground">说明</label>
          <input v-model="roleForm.desc" placeholder="角色说明" class="field-input flex-1" />
        </div>
        <div>
          <div class="mb-2 text-xs font-semibold text-muted-foreground">功能模块权限</div>
          <div v-if="isSuperAdminRole" class="bg-primary/[0.06] p-3 text-sm text-primary">
            超级管理员固定拥有全部权限，不可调整。
          </div>
          <div v-else class="grid grid-cols-2 gap-2">
            <label
              v-for="m in permModules"
              :key="m.key"
              class="flex cursor-pointer items-center gap-2 bg-muted/40 px-3 py-2 text-sm transition-colors hover:bg-accent/60"
            >
              <input type="checkbox" :checked="roleForm.permissions.includes(m.key)" @change="togglePerm(m.key)" />
              {{ m.label }}
            </label>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="roleDrawer = false">取消</Button>
        <Button @click="saveRole">保存</Button>
      </template>
    </Drawer>
  </div>
</template>
