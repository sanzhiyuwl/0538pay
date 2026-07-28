<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Plus, Copy, Trash2 } from 'lucide-vue-next'
import { Panel, Button, Badge, Switch, Drawer, Pagination } from '@/components/ui'
import {
  fetchAgents,
  fetchInvites,
  createInvite,
  setInviteStatus,
  deleteInvite,
  type Agent,
  type Invite,
} from '@/lib/api/console'
import { ApiError } from '@/lib/api/client'

const invites = ref<Invite[]>([])
const total = ref(0)
const loading = ref(false)
const agents = ref<Agent[]>([])
const filterAgent = ref('')
const page = ref(1)
const pageSize = 20
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const agentName = (id: number) => (id === 0 ? '平台' : agents.value.find((a) => a.id === id)?.name ?? `#${id}`)

// 公开自助进件页地址（客户扫码/点开的落地页；公开页在后续批次实现）
const enrollUrl = (code: string) => `${location.origin}/enroll/${code}`

async function load() {
  loading.value = true
  try {
    const { list, total: t } = await fetchInvites({
      agent_id: filterAgent.value ? Number(filterAgent.value) : undefined,
      page: page.value,
      pageSize,
    })
    invites.value = list
    total.value = t
  } catch (e) {
    alert(e instanceof ApiError ? e.message : '加载邀请链接失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    const { list } = await fetchAgents({ pageSize: 500 })
    agents.value = list
  } catch {
    // ignore
  }
  await load()
})

function go(p: number) {
  page.value = p
  load()
}

async function copyLink(code: string) {
  try {
    await navigator.clipboard.writeText(enrollUrl(code))
    alert('链接已复制')
  } catch {
    alert(enrollUrl(code))
  }
}

async function toggle(v: Invite) {
  if (v.status === 2) {
    alert('已失效的链接不可再启用，请新建替换')
    return
  }
  const next = v.status === 1 ? 0 : 1
  try {
    await setInviteStatus(v.id, next)
    v.status = next
  } catch (e) {
    alert(e instanceof ApiError ? e.message : '操作失败')
  }
}

async function remove(v: Invite) {
  if (!confirm(`确定删除邀请链接「${v.name || v.code}」？`)) return
  try {
    await deleteInvite(v.id)
    await load()
  } catch (e) {
    alert(e instanceof ApiError ? e.message : '删除失败')
  }
}

// ===== 新建 =====
const drawer = ref(false)
const saving = ref(false)
const form = reactive({ agentId: '', name: '' })

function openCreate() {
  Object.assign(form, { agentId: '', name: '' })
  drawer.value = true
}

async function save() {
  saving.value = true
  try {
    await createInvite({
      agent_id: form.agentId ? Number(form.agentId) : 0,
      name: form.name || undefined,
    })
    drawer.value = false
    await load()
  } catch (e) {
    alert(e instanceof ApiError ? e.message : '生成失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="邀请链接" subtitle="客户扫码/点链接自助进件，归属绑定代理">
      <template #actions>
        <Button size="sm" @click="openCreate"><Plus />生成链接</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">归属代理</label>
          <select v-model="filterAgent" class="field-input w-48" @change="go(1)">
            <option value="">全部</option>
            <option v-for="a in agents" :key="a.id" :value="a.id">{{ a.name }}</option>
          </select>
        </div>
      </div>
    </Panel>

    <Panel title="链接列表" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[14%]">Code</th>
              <th class="w-[16%]">备注</th>
              <th class="w-[14%]">归属代理</th>
              <th class="num w-[9%]">打开数</th>
              <th class="num w-[9%]">提交数</th>
              <th class="col-center w-[10%]">状态</th>
              <th class="col-center w-[18%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="v in invites" :key="v.id">
              <td class="truncate font-medium tabular-nums">{{ v.code }}</td>
              <td class="truncate">{{ v.name || '—' }}</td>
              <td class="truncate dim">{{ agentName(v.agent_id) }}</td>
              <td class="num tabular-nums">{{ v.open_count }}</td>
              <td class="num tabular-nums">{{ v.submit_count }}</td>
              <td class="col-center">
                <div v-if="v.status === 2" class="flex justify-center">
                  <Badge variant="destructive">已失效</Badge>
                </div>
                <div v-else class="flex justify-center">
                  <Switch :model-value="v.status === 1" size="sm" @update:model-value="toggle(v)" />
                </div>
              </td>
              <td class="col-center">
                <div class="flex items-center justify-center gap-1">
                  <Button variant="ghost" size="sm" @click="copyLink(v.code)"><Copy class="size-4" />复制</Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    class="text-destructive hover:text-destructive"
                    @click="remove(v)"
                  >
                    <Trash2 class="size-4" />
                  </Button>
                </div>
              </td>
            </tr>
            <tr v-if="loading">
              <td colspan="7" class="py-10 text-center dim">加载中…</td>
            </tr>
            <tr v-else-if="!invites.length">
              <td colspan="7" class="py-10 text-center dim">暂无邀请链接</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="page" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        链接有效期锚定终态事件（关单/驳回/退款完成后 24h 起算，审核中不倒计时）。二维码与自助进件公开页在后续批次实现。
      </p>
    </Panel>

    <Drawer v-model="drawer" title="生成邀请链接" subtitle="选择归属代理，客户从该链接进来的进件单自动归属该代理">
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">归属代理</label>
          <select v-model="form.agentId" class="field-input flex-1">
            <option value="">平台自己（无代理）</option>
            <option v-for="a in agents" :key="a.id" :value="a.id">{{ a.name }}（{{ a.account }}）</option>
          </select>
        </div>
        <div class="row-field">
          <label class="lbl">备注</label>
          <input v-model="form.name" placeholder="选填，便于区分给了哪个客户" class="field-input flex-1" />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="drawer = false">取消</Button>
        <Button :disabled="saving" @click="save">{{ saving ? '生成中…' : '生成' }}</Button>
      </template>
    </Drawer>
  </div>
</template>
