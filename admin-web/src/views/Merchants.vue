<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import {
  Search,
  RotateCcw,
  UserPlus,
  MoreHorizontal,
  Pencil,
  LogIn,
  Wallet,
  KeyRound,
  Users,
  Trash2,
  CheckCircle2,
  XCircle,
  AlertCircle,
} from 'lucide-vue-next'
import { Panel, Button, Select, Pagination } from '@/components/ui'
import {
  merchants as allMerchants,
  groups,
  settleTypes,
  searchColumns,
  statusFilters,
  type Merchant,
} from '@/lib/mock/merchants'
import { formatMoney } from '@/lib/utils'

const columnOptions = searchColumns.map((c) => ({ value: c.value, label: c.label }))
const statusOptions = statusFilters.map((s) => ({ value: s.value, label: s.label }))
const groupOptions = [{ value: -1, label: '全部用户组' }, ...groups.map((g) => ({ value: g.gid, label: g.name }))]

// ===== 筛选 =====
const filters = ref({ column: 'uid', value: '', gid: -1, dstatus: '0' })

const filtered = computed(() =>
  allMerchants.filter((m) => {
    if (filters.value.gid > -1 && m.gid !== filters.value.gid) return false
    if (filters.value.dstatus !== '0') {
      const [field, val] = filters.value.dstatus.split('_')
      if (String((m as any)[field]) !== val) return false
    }
    if (filters.value.value.trim()) {
      const v = filters.value.value.trim()
      const f = (m as any)[filters.value.column]
      if (f == null || !String(f).includes(v)) return false
    }
    return true
  }),
)

function resetFilters() {
  filters.value = { column: 'uid', value: '', gid: -1, dstatus: '0' }
  page.value = 1
}

// ===== 分页 =====
const page = ref(1)
const pageSize = 15
const total = computed(() => filtered.value.length)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const safePage = computed(() => Math.min(page.value, pageCount.value))
const pageRows = computed(() => filtered.value.slice((safePage.value - 1) * pageSize, safePage.value * pageSize))
function go(p: number) {
  page.value = Math.min(Math.max(1, p), pageCount.value)
}

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

const menuActions = [
  { key: 'edit', label: '编辑资料', icon: Pencil },
  { key: 'recharge', label: '充值 / 扣款', icon: Wallet },
  { key: 'group', label: '修改用户组', icon: Users },
  { key: 'resetkey', label: '重置密钥', icon: KeyRound },
  { key: 'sso', label: '登录商户', icon: LogIn },
  { key: '-', label: '', icon: null },
  { key: 'delete', label: '删除商户', icon: Trash2, danger: true },
]

function settlePrefix(m: Merchant) {
  return settleTypes[m.settle_id]?.prefix ?? ''
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="商户管理" :subtitle="`共 ${total} 个商户`">
      <template #actions>
        <Button size="sm"><UserPlus />添加商户</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">商户信息</label>
          <Select v-model="filters.column" :options="columnOptions" class="w-28" />
          <input v-model="filters.value" placeholder="搜索内容" class="field-input w-48" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">用户组</label>
          <Select v-model="filters.gid" :options="groupOptions" class="w-32" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">状态</label>
          <Select v-model="filters.dstatus" :options="statusOptions" class="w-36" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="page = 1"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <Panel title="商户列表" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[14%]">商户号 / 用户组</th>
              <th class="w-[11%]">余额</th>
              <th class="w-[17%]">结算账号 / 姓名</th>
              <th class="w-[15%]">联系方式</th>
              <th class="w-[18%]">域名 / 添加时间</th>
              <th class="col-center w-[15%]">状态</th>
              <th class="col-center w-[10%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in pageRows" :key="m.uid">
              <td>
                <div class="font-medium">{{ m.uid }}</div>
                <div class="cursor-pointer truncate text-xs text-primary">{{ m.groupname }}</div>
              </td>
              <td>
                <span class="cursor-pointer font-semibold tabular-nums">
                  <span class="text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(m.money) }}
                </span>
              </td>
              <td>
                <template v-if="m.account">
                  <div class="truncate">
                    <span v-if="settlePrefix(m)" class="text-success">{{ settlePrefix(m) }}</span>{{ m.account }}
                  </div>
                  <div class="truncate text-xs dim">{{ m.username }}</div>
                </template>
                <span v-else class="dim">未设置</span>
              </td>
              <td>
                <div v-if="m.qq" class="truncate">QQ：{{ m.qq }}</div>
                <div class="truncate text-xs dim">{{ m.phone || m.email }}</div>
              </td>
              <td>
                <div class="truncate">{{ m.url }}</div>
                <div class="text-xs dim">{{ m.addtime }}</div>
              </td>
              <td class="col-center">
                <div class="flex items-center justify-center gap-3">
                  <!-- 用户状态 -->
                  <span v-if="m.status === 1" class="inline-flex items-center gap-1 text-xs text-success"><CheckCircle2 class="size-3.5" />正常</span>
                  <span v-else-if="m.status === 2" class="inline-flex items-center gap-1 text-xs text-warning"><AlertCircle class="size-3.5" />未审核</span>
                  <span v-else class="inline-flex items-center gap-1 text-xs text-destructive"><XCircle class="size-3.5" />封禁</span>
                  <!-- 实名 -->
                  <span :class="['inline-flex items-center gap-1 text-xs', m.cert === 1 ? 'text-success' : 'text-muted-foreground']">
                    <component :is="m.cert === 1 ? CheckCircle2 : XCircle" class="size-3.5" />实名
                  </span>
                </div>
                <div class="mt-1 flex items-center justify-center gap-3">
                  <!-- 支付权限 -->
                  <span v-if="m.pay === 2" class="inline-flex items-center gap-1 text-xs text-warning"><AlertCircle class="size-3.5" />支付</span>
                  <span v-else :class="['inline-flex items-center gap-1 text-xs', m.pay === 1 ? 'text-success' : 'text-destructive']">
                    <component :is="m.pay === 1 ? CheckCircle2 : XCircle" class="size-3.5" />支付
                  </span>
                  <!-- 结算权限 -->
                  <span :class="['inline-flex items-center gap-1 text-xs', m.settle === 1 ? 'text-success' : 'text-destructive']">
                    <component :is="m.settle === 1 ? CheckCircle2 : XCircle" class="size-3.5" />结算
                  </span>
                </div>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(m.uid)">
                    操作 <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === m.uid"
                    class="menu-panel absolute right-0 top-full z-20 mt-1.5 w-36"
                    @click.stop
                  >
                    <template v-for="a in menuActions" :key="a.key">
                      <div v-if="a.key === '-'" class="menu-sep" />
                      <button
                        v-else
                        class="menu-item"
                        :class="a.danger && 'menu-item-danger'"
                        @click="openMenu = null"
                      >
                        <component :is="a.icon" class="size-4 shrink-0 opacity-70" />
                        <span class="flex-1">{{ a.label }}</span>
                      </button>
                    </template>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="7" class="py-10 text-center dim">没有符合条件的商户</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="safePage" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
    </Panel>
  </div>
</template>
