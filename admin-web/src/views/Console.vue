<script setup lang="ts">
import { ref, computed, reactive, onMounted, onUnmounted } from 'vue'
import {
  Search,
  RotateCcw,
  Plus,
  MoreHorizontal,
  Pencil,
  CalendarPlus,
  Power,
  LogIn,
  Trash2,
  Server,
} from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Drawer, Pagination } from '@/components/ui'
import { shouldDropUp } from '@/composables/useRowMenu'
import {
  sites as allSites,
  siteStatus,
  sitePlanText,
  allModules,
  allChannels,
  planQuota,
  planPrice,
  calcConsoleStats,
  type Site,
  type SitePlan,
} from '@/lib/mock/sites'
import { formatMoney } from '@/lib/utils'

const planOptions = [
  { value: '', label: '全部套餐' },
  ...(Object.keys(sitePlanText) as SitePlan[]).map((k) => ({ value: k, label: sitePlanText[k] })),
]
const statusOptions = [
  { value: -1, label: '全部状态' },
  ...Object.entries(siteStatus).map(([k, s]) => ({ value: Number(k), label: s.text })),
]
const planSelectOptions = (Object.keys(sitePlanText) as SitePlan[]).map((k) => ({
  value: k,
  label: `${sitePlanText[k]}（¥${planPrice[k]}/年）`,
}))

// ===== 筛选 =====
const filters = ref({ kw: '', plan: '', status: -1 })
const filtered = computed(() =>
  allSites.filter((s) => {
    if (filters.value.plan && s.plan !== filters.value.plan) return false
    if (filters.value.status > -1 && s.status !== filters.value.status) return false
    if (filters.value.kw.trim()) {
      const v = filters.value.kw.trim()
      if (!`${s.name}${s.domain}${s.adminUser}`.includes(v)) return false
    }
    return true
  }),
)
function resetFilters() {
  filters.value = { kw: '', plan: '', status: -1 }
  page.value = 1
}

// ===== 分页 =====
const page = ref(1)
const pageSize = 10
const total = computed(() => filtered.value.length)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const safePage = computed(() => Math.min(page.value, pageCount.value))
const pageRows = computed(() =>
  filtered.value.slice((safePage.value - 1) * pageSize, safePage.value * pageSize),
)
function go(p: number) {
  page.value = Math.min(Math.max(1, p), pageCount.value)
}

const stats = computed(() => calcConsoleStats(allSites))

// ===== 用量进度 =====
function usagePct(used: number, max: number) {
  return max > 0 ? Math.min(100, Math.round((used / max) * 100)) : 0
}
function pctColor(pct: number) {
  if (pct >= 90) return 'bg-destructive'
  if (pct >= 70) return 'bg-warning'
  return 'bg-primary'
}

// ===== 创建 / 编辑抽屉 =====
const drawerOpen = ref(false)
const editingId = ref<number | null>(null)
const blankForm = () => ({
  name: '',
  domain: '',
  adminUser: '',
  password: '',
  plan: 'basic' as SitePlan,
  months: 12,
  permissions: allModules.map((m) => m.key),
  channelScope: [1, 2] as number[],
  quota: { ...planQuota.basic },
})
const form = reactive(blankForm())

function openCreate() {
  editingId.value = null
  Object.assign(form, blankForm())
  drawerOpen.value = true
}
function openEdit(s: Site) {
  editingId.value = s.id
  Object.assign(form, {
    name: s.name,
    domain: s.domain,
    adminUser: s.adminUser,
    password: '',
    plan: s.plan,
    months: 12,
    permissions: [...s.permissions],
    channelScope: [...s.channelScope],
    quota: { ...s.quota },
  })
  drawerOpen.value = true
}
// 切套餐时用套餐默认配额回填
function onPlanChange() {
  form.quota = { ...planQuota[form.plan] }
}
function toggleModule(key: string) {
  const i = form.permissions.indexOf(key)
  if (i > -1) form.permissions.splice(i, 1)
  else form.permissions.push(key)
}
function toggleChannel(id: number) {
  const i = form.channelScope.indexOf(id)
  if (i > -1) form.channelScope.splice(i, 1)
  else form.channelScope.push(id)
}
// 原型阶段：保存仅关闭抽屉（不落库）
function save() {
  drawerOpen.value = false
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
onMounted(() => window.addEventListener('click', closeMenu))
onUnmounted(() => window.removeEventListener('click', closeMenu))

// 停用 / 启用（本地 mock 直改）
function toggleSiteStatus(s: Site) {
  s.status = s.status === 1 ? 2 : 1
  openMenu.value = null
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 概况 -->
    <Panel title="控制台" subtitle="平台方视角：创建、售卖、管理子站点租户（分站），分配权限、配额与使用期限">
      <template #actions>
        <Button size="sm" @click="openCreate"><Plus />创建租户</Button>
      </template>
      <div class="grid grid-cols-2 gap-x-8 gap-y-5 sm:grid-cols-3 lg:grid-cols-5">
        <div>
          <div class="text-[13px] text-muted-foreground">租户总数</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums">{{ stats.total }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">运行中</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-success">{{ stats.running }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">即将到期</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-warning">{{ stats.expiringSoon }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已停用 / 过期</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-muted-foreground">{{ stats.stopped }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">租户累计收入</div>
          <div class="mt-1.5 text-xl font-normal tabular-nums text-primary"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.revenue) }}</div>
        </div>
      </div>
    </Panel>

    <!-- 筛选 -->
    <Panel title="租户筛选" :subtitle="`共 ${total} 个`">
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">租户搜索</label>
          <input v-model="filters.kw" placeholder="站点名 / 域名 / 管理员" class="field-input w-52" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">套餐</label>
          <Select v-model="filters.plan" :options="planOptions" class="w-32" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">状态</label>
          <Select v-model="filters.status" :options="statusOptions" class="w-28" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="page = 1"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 租户列表 -->
    <Panel title="租户列表" :subtitle="`${total} 个`">
      <div>
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[18%]">站点 / 域名</th>
              <th class="w-[11%]">管理员</th>
              <th class="w-[9%]">套餐</th>
              <th class="w-[18%]">商户用量</th>
              <th class="w-[18%]">月交易额用量</th>
              <th class="w-[13%]">到期时间</th>
              <th class="col-center w-[8%]">状态</th>
              <th class="col-center w-[6%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="s in pageRows" :key="s.id">
              <td>
                <div class="flex items-center gap-2">
                  <Server class="size-4 shrink-0 text-primary" />
                  <div class="min-w-0">
                    <div class="truncate font-medium">{{ s.name }}</div>
                    <div class="truncate text-xs dim">{{ s.domain }}</div>
                  </div>
                </div>
              </td>
              <td class="truncate font-mono text-[13px]">{{ s.adminUser }}</td>
              <td><Badge variant="outline">{{ sitePlanText[s.plan] }}</Badge></td>
              <td>
                <div class="flex items-center justify-between text-xs">
                  <span class="tabular-nums">{{ s.usage.merchants }} / {{ s.quota.maxMerchants }}</span>
                  <span class="dim">{{ usagePct(s.usage.merchants, s.quota.maxMerchants) }}%</span>
                </div>
                <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-muted">
                  <div
                    class="h-full rounded-full"
                    :class="pctColor(usagePct(s.usage.merchants, s.quota.maxMerchants))"
                    :style="{ width: usagePct(s.usage.merchants, s.quota.maxMerchants) + '%' }"
                  />
                </div>
              </td>
              <td>
                <div class="flex items-center justify-between text-xs">
                  <span class="tabular-nums">¥{{ formatMoney(s.usage.monthlyAmount) }}</span>
                  <span class="dim">{{ usagePct(s.usage.monthlyAmount, s.quota.monthlyAmount) }}%</span>
                </div>
                <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-muted">
                  <div
                    class="h-full rounded-full"
                    :class="pctColor(usagePct(s.usage.monthlyAmount, s.quota.monthlyAmount))"
                    :style="{ width: usagePct(s.usage.monthlyAmount, s.quota.monthlyAmount) + '%' }"
                  />
                </div>
              </td>
              <td class="text-xs">{{ s.expireTime.slice(0, 10) }}</td>
              <td class="col-center">
                <Badge :variant="siteStatus[s.status].variant">{{ siteStatus[s.status].text }}</Badge>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(s.id, $event)">
                    <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === s.id"
                    class="menu-panel absolute right-0 z-20 w-36"
                    :class="dropUp ? 'bottom-full mb-1.5' : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <button class="menu-item" @click="openEdit(s); openMenu = null">
                      <Pencil class="size-4 shrink-0 opacity-70" /><span class="flex-1">编辑配置</span>
                    </button>
                    <button class="menu-item" @click="openMenu = null">
                      <CalendarPlus class="size-4 shrink-0 opacity-70" /><span class="flex-1">续期</span>
                    </button>
                    <button class="menu-item" @click="openMenu = null">
                      <LogIn class="size-4 shrink-0 opacity-70" /><span class="flex-1">免密登录</span>
                    </button>
                    <button class="menu-item" @click="toggleSiteStatus(s)">
                      <Power class="size-4 shrink-0 opacity-70" />
                      <span class="flex-1">{{ s.status === 1 ? '停用' : '启用' }}</span>
                    </button>
                    <div class="menu-sep" />
                    <button class="menu-item menu-item-danger" @click="openMenu = null">
                      <Trash2 class="size-4 shrink-0 opacity-70" /><span class="flex-1">删除租户</span>
                    </button>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="8" class="py-10 text-center dim">没有符合条件的租户</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="safePage" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>

    <!-- 创建 / 编辑抽屉 -->
    <Drawer
      v-model="drawerOpen"
      :title="editingId ? '编辑租户配置' : '创建租户'"
      subtitle="配置分站的基础信息、套餐、权限、配额与期限"
      width="max-w-xl"
    >
      <div class="space-y-5">
        <!-- 基础信息 -->
        <section>
          <h4 class="mb-2.5 text-xs font-semibold text-muted-foreground">基础信息</h4>
          <div class="space-y-3">
            <div class="flex items-center gap-3">
              <label class="w-20 shrink-0 text-right text-sm text-muted-foreground">站点名称</label>
              <input v-model="form.name" placeholder="如：云支付分站" class="field-input flex-1" />
            </div>
            <div class="flex items-center gap-3">
              <label class="w-20 shrink-0 text-right text-sm text-muted-foreground">绑定域名</label>
              <input v-model="form.domain" placeholder="如：pay.example.com" class="field-input flex-1" />
            </div>
            <div class="flex items-center gap-3">
              <label class="w-20 shrink-0 text-right text-sm text-muted-foreground">管理员账号</label>
              <input v-model="form.adminUser" placeholder="分站管理员登录账号" class="field-input flex-1" />
            </div>
            <div v-if="!editingId" class="flex items-center gap-3">
              <label class="w-20 shrink-0 text-right text-sm text-muted-foreground">初始密码</label>
              <input v-model="form.password" type="password" placeholder="分站管理员初始密码" class="field-input flex-1" />
            </div>
          </div>
        </section>

        <!-- 套餐与期限 -->
        <section class="border-t border-border/60 pt-4">
          <h4 class="mb-2.5 text-xs font-semibold text-muted-foreground">套餐与期限</h4>
          <div class="space-y-3">
            <div class="flex items-center gap-3">
              <label class="w-20 shrink-0 text-right text-sm text-muted-foreground">租户套餐</label>
              <Select v-model="form.plan" :options="planSelectOptions" class="flex-1" @update:model-value="onPlanChange" />
            </div>
            <div class="flex items-center gap-3">
              <label class="w-20 shrink-0 text-right text-sm text-muted-foreground">使用期限</label>
              <div class="flex flex-1 items-center gap-2">
                <input v-model.number="form.months" type="number" min="1" class="field-input w-24" />
                <span class="text-sm text-muted-foreground">个月</span>
              </div>
            </div>
          </div>
        </section>

        <!-- 功能权限 -->
        <section class="border-t border-border/60 pt-4">
          <h4 class="mb-2.5 text-xs font-semibold text-muted-foreground">功能模块权限</h4>
          <div class="grid grid-cols-2 gap-2">
            <label
              v-for="m in allModules"
              :key="m.key"
              class="flex cursor-pointer items-center gap-2 bg-muted/40 px-3 py-2 text-sm transition-colors hover:bg-accent/60"
            >
              <input type="checkbox" :checked="form.permissions.includes(m.key)" @change="toggleModule(m.key)" />
              {{ m.label }}
            </label>
          </div>
        </section>

        <!-- 通道范围 -->
        <section class="border-t border-border/60 pt-4">
          <h4 class="mb-2.5 text-xs font-semibold text-muted-foreground">可用支付通道</h4>
          <div class="grid grid-cols-2 gap-2">
            <label
              v-for="c in allChannels"
              :key="c.id"
              class="flex cursor-pointer items-center gap-2 bg-muted/40 px-3 py-2 text-sm transition-colors hover:bg-accent/60"
            >
              <input type="checkbox" :checked="form.channelScope.includes(c.id)" @change="toggleChannel(c.id)" />
              {{ c.name }}
            </label>
          </div>
        </section>

        <!-- 资源配额 -->
        <section class="border-t border-border/60 pt-4">
          <h4 class="mb-2.5 text-xs font-semibold text-muted-foreground">资源配额</h4>
          <div class="space-y-3">
            <div class="flex items-center gap-3">
              <label class="w-20 shrink-0 text-right text-sm text-muted-foreground">最大商户数</label>
              <input v-model.number="form.quota.maxMerchants" type="number" class="field-input flex-1" />
            </div>
            <div class="flex items-center gap-3">
              <label class="w-20 shrink-0 text-right text-sm text-muted-foreground">最大通道数</label>
              <input v-model.number="form.quota.maxChannels" type="number" class="field-input flex-1" />
            </div>
            <div class="flex items-center gap-3">
              <label class="w-20 shrink-0 text-right text-sm text-muted-foreground">月交易额上限</label>
              <input v-model.number="form.quota.monthlyAmount" type="number" class="field-input flex-1" />
            </div>
          </div>
        </section>
      </div>

      <template #footer>
        <Button variant="outline" @click="drawerOpen = false">取消</Button>
        <Button @click="save">{{ editingId ? '保存配置' : '创建租户' }}</Button>
      </template>
    </Drawer>
  </div>
</template>
