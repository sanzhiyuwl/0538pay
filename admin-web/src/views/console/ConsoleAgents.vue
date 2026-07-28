<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Plus, Pencil, Trash2, Users, Wallet } from 'lucide-vue-next'
import { Panel, Button, Switch, Drawer, Pagination, Select } from '@/components/ui'
import {
  fetchAgents,
  createAgent,
  updateAgent,
  setAgentStatus,
  deleteAgent,
  fetchAgentPermissions,
  type Agent,
  type AgentPermission,
} from '@/lib/api/console'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'

const toast = useToast()
const confirm = useConfirm()

// 状态筛选下拉选项（收单同款 Select）
const statusOptions = [
  { value: '', label: '全部' },
  { value: '1', label: '启用' },
  { value: '0', label: '停用' },
]

const agents = ref<Agent[]>([])
const total = ref(0)
const loading = ref(false)
const permCatalog = ref<AgentPermission[]>([])

const filters = reactive({ keyword: '', status: '' as string })
const page = ref(1)
const pageSize = 20
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

// 权限点按分组聚合，便于抽屉里分块勾选
const permGroups = computed(() => {
  const map = new Map<string, AgentPermission[]>()
  for (const p of permCatalog.value) {
    if (!map.has(p.group)) map.set(p.group, [])
    map.get(p.group)!.push(p)
  }
  return [...map.entries()].map(([group, items]) => ({ group, items }))
})

function permText(permissions: string): string {
  const keys = permissions.split(',').map((s) => s.trim()).filter(Boolean)
  if (!keys.length) return '未开通任何权限'
  return keys
    .map((k) => permCatalog.value.find((p) => p.key === k)?.name ?? k)
    .join(' · ')
}

async function load() {
  loading.value = true
  try {
    const { list, total: t } = await fetchAgents({
      keyword: filters.keyword || undefined,
      status: filters.status === '' ? undefined : Number(filters.status),
      page: page.value,
      pageSize,
    })
    agents.value = list
    total.value = t
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载代理失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    permCatalog.value = await fetchAgentPermissions()
  } catch {
    // 权限清单拉取失败不阻塞列表
  }
  await load()
})

function search() {
  page.value = 1
  load()
}
function resetFilters() {
  filters.keyword = ''
  filters.status = ''
  search()
}
function go(p: number) {
  page.value = p
  load()
}

// ===== 新增/编辑抽屉 =====
const drawer = ref(false)
const editing = ref<Agent | null>(null)
const saving = ref(false)
const form = reactive({
  name: '',
  account: '',
  password: '',
  contact: '',
  remark: '',
  permissions: [] as string[],
})

function openCreate() {
  editing.value = null
  Object.assign(form, {
    name: '',
    account: '',
    password: '',
    contact: '',
    remark: '',
    permissions: ['enroll'],
  })
  drawer.value = true
}
function openEdit(a: Agent) {
  editing.value = a
  Object.assign(form, {
    name: a.name,
    account: a.account,
    password: '',
    contact: a.contact,
    remark: a.remark,
    permissions: a.permissions.split(',').map((s) => s.trim()).filter(Boolean),
  })
  drawer.value = true
}
function togglePerm(key: string) {
  const i = form.permissions.indexOf(key)
  if (i > -1) form.permissions.splice(i, 1)
  else form.permissions.push(key)
}

async function save() {
  if (!form.name.trim() || !form.account.trim()) {
    toast.error('请填写代理名称和登录账号')
    return
  }
  if (!editing.value && !form.password) {
    toast.error('请设置登录密码')
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      await updateAgent(editing.value.id, {
        name: form.name,
        account: form.account,
        password: form.password || undefined,
        contact: form.contact,
        remark: form.remark,
        permissions: form.permissions,
      })
    } else {
      await createAgent({
        name: form.name,
        account: form.account,
        password: form.password,
        contact: form.contact,
        remark: form.remark,
        permissions: form.permissions,
      })
    }
    drawer.value = false
    toast.success('已保存')
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function toggleStatus(a: Agent) {
  const next = a.status === 1 ? 0 : 1
  try {
    await setAgentStatus(a.id, next)
    a.status = next
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '操作失败')
  }
}

async function remove(a: Agent) {
  if (!(await confirm(`确定删除代理「${a.name}」？仅无名额记录、无进件单和邀请的代理可删；有资金/业务记录的请改用「停用」。`, { title: '删除代理', danger: true }))) return
  try {
    await deleteAgent(a.id)
    toast.success('已删除')
    await load()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '删除失败')
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="代理管理" :subtitle="`共 ${total} 个代理`">
      <template #actions>
        <Button size="sm" @click="openCreate"><Plus />开通代理</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">搜索</label>
          <input
            v-model="filters.keyword"
            placeholder="代理名称 / 登录账号"
            class="field-input w-52"
            @keyup.enter="search"
          />
        </div>
        <div class="filter-item">
          <label class="filter-label">状态</label>
          <Select v-model="filters.status" :options="statusOptions" class="w-28" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="search">搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters">重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="代理列表" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[6%]">ID</th>
              <th class="w-[16%]">代理</th>
              <th class="w-[12%]">登录账号</th>
              <th class="w-[12%]">联系方式</th>
              <th class="w-[30%]">已开通权限</th>
              <th class="col-center w-[8%]">状态</th>
              <th class="col-center w-[16%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in agents" :key="a.id">
              <td class="font-medium tabular-nums">{{ a.id }}</td>
              <td>
                <div class="flex items-center gap-1.5 font-medium">
                  <Users class="size-4 shrink-0 text-primary" />{{ a.name }}
                </div>
              </td>
              <td class="truncate">{{ a.account }}</td>
              <td class="truncate dim">{{ a.contact || '—' }}</td>
              <td class="text-[13px]">{{ permText(a.permissions) }}</td>
              <td class="col-center">
                <div class="flex justify-center">
                  <Switch :model-value="a.status === 1" size="sm" @update:model-value="toggleStatus(a)" />
                </div>
              </td>
              <td class="col-center">
                <div class="flex items-center justify-center gap-1">
                  <Button variant="ghost" size="sm" @click="openEdit(a)"><Pencil class="size-4" />编辑</Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    class="text-destructive hover:text-destructive"
                    @click="remove(a)"
                  >
                    <Trash2 class="size-4" />
                  </Button>
                </div>
              </td>
            </tr>
            <tr v-if="loading">
              <td colspan="7" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!agents.length">
              <td colspan="7" class="py-10 text-center dim">暂无代理，点右上角开通</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="page" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>

    <!-- 新增/编辑抽屉 -->
    <Drawer
      v-model="drawer"
      :title="editing ? '编辑代理' : '开通代理'"
      subtitle="权限开通啥，代理端就有啥；数据只看自己名下"
    >
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">代理名称<span class="text-destructive">*</span></label>
          <input v-model="form.name" placeholder="代理名称" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">登录账号<span class="text-destructive">*</span></label>
          <input v-model="form.account" placeholder="代理端登录账号" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">登录密码<span v-if="!editing" class="text-destructive">*</span></label>
          <input
            v-model="form.password"
            type="password"
            :placeholder="editing ? '留空表示不修改' : '设置登录密码'"
            class="field-input flex-1"
          />
        </div>
        <div class="row-field">
          <label class="lbl">联系方式</label>
          <input v-model="form.contact" placeholder="手机号 / 微信 / QQ" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">备注</label>
          <input v-model="form.remark" placeholder="选填" class="field-input flex-1" />
        </div>
        <div v-if="!editing">
          <div class="mb-2 flex items-center gap-1.5 text-xs font-semibold text-muted-foreground">
            <Wallet class="size-3.5" />初始权限
          </div>
          <div v-for="g in permGroups" :key="g.group" class="mb-3">
            <div class="mb-1.5 text-[11px] text-muted-foreground">{{ g.group }}</div>
            <div class="grid grid-cols-2 gap-2">
              <label
                v-for="p in g.items"
                :key="p.key"
                class="flex cursor-pointer items-start gap-2 bg-muted/40 px-3 py-2 text-sm transition-colors hover:bg-accent/60"
                :title="p.desc"
              >
                <input
                  type="checkbox"
                  class="mt-0.5"
                  :checked="form.permissions.includes(p.key)"
                  @change="togglePerm(p.key)"
                />
                <span>
                  {{ p.name }}
                  <span class="block text-[11px] text-muted-foreground">{{ p.desc }}</span>
                </span>
              </label>
            </div>
          </div>
        </div>
        <p v-else class="bg-muted/40 px-3 py-2 text-[11px] text-muted-foreground">
          功能权限请在左侧「权限分配」页调整（改后代理下次操作即刻生效）。
        </p>
      </div>
      <template #footer>
        <Button variant="outline" @click="drawer = false">取消</Button>
        <Button :disabled="saving" @click="save">{{ saving ? '保存中…' : '保存' }}</Button>
      </template>
    </Drawer>
  </div>
</template>
