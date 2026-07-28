<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Users, ShieldCheck, Search, ChevronDown, Check, Minus } from 'lucide-vue-next'
import { Panel, Button } from '@/components/ui'
import {
  fetchAgents,
  fetchAgentPermissions,
  setAgentPermissions,
  type Agent,
  type AgentPermission,
} from '@/lib/api/console'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const agents = ref<Agent[]>([])
const permCatalog = ref<AgentPermission[]>([])
const loading = ref(false)
const saving = ref(false)
const keyword = ref('')

// 当前选中的代理 + 其权限勾选集合（工作副本，保存前不回写列表）
const selectedId = ref<number | null>(null)
const checked = reactive(new Set<string>())
// 折叠的分组（组名集合，命中即折叠）
const collapsed = reactive(new Set<string>())

const selected = computed(() => agents.value.find((a) => a.id === selectedId.value) ?? null)

// 权限点按分组聚合
const permGroups = computed(() => {
  const map = new Map<string, AgentPermission[]>()
  for (const p of permCatalog.value) {
    if (!map.has(p.group)) map.set(p.group, [])
    map.get(p.group)!.push(p)
  }
  return [...map.entries()].map(([group, items]) => ({ group, items }))
})

const filteredAgents = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return agents.value
  return agents.value.filter((a) => a.name.toLowerCase().includes(kw) || a.account.toLowerCase().includes(kw))
})

// 选中代理的当前权限（用于「未保存」比对）
const savedKeys = computed(() => {
  const a = selected.value
  if (!a) return [] as string[]
  return a.permissions.split(',').map((s) => s.trim()).filter(Boolean)
})
const dirty = computed(() => {
  if (!selected.value) return false
  const now = [...checked].sort().join(',')
  const was = [...savedKeys.value].sort().join(',')
  return now !== was
})

async function loadAll() {
  loading.value = true
  try {
    const [permList, agentPage] = await Promise.all([
      fetchAgentPermissions(),
      fetchAgents({ page: 1, pageSize: 200 }),
    ])
    permCatalog.value = permList
    agents.value = agentPage.list
    // 默认选中第一个
    if (agents.value.length && selectedId.value === null) selectAgent(agents.value[0])
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}
onMounted(loadAll)

function selectAgent(a: Agent) {
  selectedId.value = a.id
  checked.clear()
  for (const k of a.permissions.split(',').map((s) => s.trim()).filter(Boolean)) checked.add(k)
}

function toggle(key: string) {
  if (checked.has(key)) checked.delete(key)
  else checked.add(key)
}

// —— 组级分配：整组一键开/关，勾选态三态（全选 / 半选 / 未选）——
function groupState(items: AgentPermission[]): 'all' | 'some' | 'none' {
  const on = items.filter((p) => checked.has(p.key)).length
  if (on === 0) return 'none'
  if (on === items.length) return 'all'
  return 'some'
}
function toggleGroup(items: AgentPermission[]) {
  // 只要不是「全选」就整组开；已全选则整组关。
  const turnOn = groupState(items) !== 'all'
  for (const p of items) {
    if (turnOn) checked.add(p.key)
    else checked.delete(p.key)
  }
}

function toggleCollapse(group: string) {
  if (collapsed.has(group)) collapsed.delete(group)
  else collapsed.add(group)
}

function reset() {
  if (selected.value) selectAgent(selected.value)
}

async function save() {
  if (!selected.value) return
  saving.value = true
  try {
    const keys = [...checked]
    await setAgentPermissions(selected.value.id, keys)
    // 回写本地列表，保持「已保存」基准同步
    selected.value.permissions = keys.join(',')
    toast.success('权限已保存，代理下次操作即刻生效')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

function permText(permissions: string): string {
  const keys = permissions.split(',').map((s) => s.trim()).filter(Boolean)
  if (!keys.length) return '未开通'
  return keys.map((k) => permCatalog.value.find((p) => p.key === k)?.name ?? k).join(' · ')
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="权限分配" subtitle="按代理逐个分配功能权限；权限开通啥，代理端就有啥，改后代理下次操作即刻生效（无需重登）">
      <div v-if="loading" class="py-10 text-center dim">加载中…</div>
      <div v-else class="grid grid-cols-[16rem_1fr] gap-4">
        <!-- 左：代理列表 -->
        <div class="border-r border-border/60 pr-4">
          <div class="mb-2 flex items-center gap-1.5 rounded bg-muted/40 px-2.5 py-1.5">
            <Search class="size-3.5 shrink-0 text-muted-foreground" />
            <input v-model="keyword" placeholder="搜索代理名称 / 账号" class="w-full bg-transparent text-sm outline-none" />
          </div>
          <div class="max-h-[60vh] space-y-1 overflow-y-auto">
            <button
              v-for="a in filteredAgents"
              :key="a.id"
              type="button"
              class="w-full px-2.5 py-2 text-left text-sm transition-colors"
              :class="a.id === selectedId ? 'bg-primary/[0.08] text-primary' : 'hover:bg-muted/60'"
              @click="selectAgent(a)"
            >
              <div class="flex items-center gap-1.5 font-medium">
                <Users class="size-3.5 shrink-0" />{{ a.name }}
                <span v-if="a.status !== 1" class="text-[10px] text-destructive">停用</span>
              </div>
              <div class="mt-0.5 truncate text-[11px] text-muted-foreground">{{ permText(a.permissions) }}</div>
            </button>
            <div v-if="!filteredAgents.length" class="py-8 text-center text-sm dim">无匹配代理</div>
          </div>
        </div>

        <!-- 右：权限勾选 -->
        <div v-if="selected">
          <div class="mb-3 flex items-center gap-1.5 text-sm font-semibold">
            <ShieldCheck class="size-4 text-primary" />
            {{ selected.name }}
            <span class="text-[11px] font-normal text-muted-foreground">（{{ selected.account }}）</span>
          </div>
          <div v-for="g in permGroups" :key="g.group" class="mb-3 last:mb-0">
            <!-- 组头：勾选框整组三态开/关，点标题区折叠，右侧一键整组 -->
            <div class="flex items-center gap-2.5 bg-muted/40 px-3 py-2">
              <button
                type="button"
                class="flex size-4 shrink-0 items-center justify-center rounded border transition-colors"
                :class="groupState(g.items) === 'none'
                  ? 'border-input bg-background hover:border-primary/60'
                  : 'border-primary bg-primary text-primary-foreground'"
                @click="toggleGroup(g.items)"
              >
                <Check v-if="groupState(g.items) === 'all'" class="size-3" :stroke-width="3" />
                <Minus v-else-if="groupState(g.items) === 'some'" class="size-3" :stroke-width="3" />
              </button>
              <button
                type="button"
                class="flex flex-1 items-center gap-1.5 text-left"
                @click="toggleCollapse(g.group)"
              >
                <ChevronDown
                  class="size-3.5 shrink-0 text-muted-foreground transition-transform"
                  :class="{ '-rotate-90': collapsed.has(g.group) }"
                />
                <span class="text-sm font-semibold">{{ g.group }}</span>
                <span class="rounded-full bg-background px-1.5 text-[11px] tabular-nums text-muted-foreground">
                  {{ g.items.filter((p) => checked.has(p.key)).length }}/{{ g.items.length }}
                </span>
              </button>
              <button
                type="button"
                class="shrink-0 text-[11px] font-medium text-primary hover:underline"
                @click="toggleGroup(g.items)"
              >
                {{ groupState(g.items) === 'all' ? '取消整组' : '开通整组' }}
              </button>
            </div>
            <!-- 组内各权限点：自适应两列，空位不占格；选中态主色描边 -->
            <div v-show="!collapsed.has(g.group)" class="mt-2 grid grid-cols-2 gap-2">
              <button
                v-for="p in g.items"
                :key="p.key"
                type="button"
                class="group flex items-start gap-2.5 border px-3 py-2.5 text-left transition-colors"
                :class="checked.has(p.key)
                  ? 'border-primary/40 bg-primary/[0.06]'
                  : 'border-border/70 bg-background hover:border-primary/30 hover:bg-muted/40'"
                @click="toggle(p.key)"
              >
                <span
                  class="mt-0.5 flex size-4 shrink-0 items-center justify-center rounded border transition-colors"
                  :class="checked.has(p.key)
                    ? 'border-primary bg-primary text-primary-foreground'
                    : 'border-input bg-background group-hover:border-primary/60'"
                >
                  <Check v-if="checked.has(p.key)" class="size-3" :stroke-width="3" />
                </span>
                <span class="min-w-0">
                  <span class="text-sm font-medium" :class="checked.has(p.key) ? 'text-primary' : 'text-foreground'">{{ p.name }}</span>
                  <span class="mt-0.5 block truncate text-[11px] text-muted-foreground">{{ p.desc }}</span>
                </span>
              </button>
            </div>
          </div>
          <div class="flex items-center gap-2 border-t border-border/60 pt-3">
            <Button :disabled="saving || !dirty" @click="save">{{ saving ? '保存中…' : '保存权限' }}</Button>
            <Button variant="outline" :disabled="!dirty" @click="reset">重置</Button>
            <span v-if="dirty" class="text-[11px] text-[#E6A23C]">有未保存的改动</span>
          </div>
        </div>
        <div v-else class="flex items-center justify-center text-sm dim">从左侧选择一个代理分配权限</div>
      </div>
    </Panel>
  </div>
</template>
