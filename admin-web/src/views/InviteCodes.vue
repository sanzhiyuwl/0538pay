<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Search, RotateCcw, Plus, Trash2, Eraser } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination, Drawer } from '@/components/ui'
import { inviteCodes, inviteStatus, statusOptions, calcInviteStats } from '@/lib/mock/invitecodes'

// ===== 筛选 =====
const filters = ref({ kw: '', status: -1 })

const filtered = computed(() => {
  return inviteCodes.filter((c) => {
    if (filters.value.status > -1 && c.status !== filters.value.status) return false
    if (filters.value.kw.trim() && !c.code.includes(filters.value.kw.trim())) return false
    return true
  })
})

function resetFilters() {
  filters.value = { kw: '', status: -1 }
}

// ===== 分页 =====
const page = ref(1)
const pageSize = 15
const total = computed(() => filtered.value.length)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const safePage = computed(() => Math.min(page.value, pageCount.value))
const pageRows = computed(() =>
  filtered.value.slice((safePage.value - 1) * pageSize, safePage.value * pageSize),
)
function go(p: number) {
  page.value = Math.min(Math.max(1, p), pageCount.value)
}

const stats = computed(() => calcInviteStats(filtered.value))

// 筛选变化回到第 1 页
watch(filters, () => { page.value = 1 }, { deep: true })

// ===== 生成邀请码（原型：仅弹窗交互，不落库）=====
const genOpen = ref(false)
const genNum = ref('10')
function submitGen() {
  genOpen.value = false
  genNum.value = '10'
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="邀请码管理" :subtitle="`共 ${total} 个`">
      <template #actions>
        <Button size="sm" @click="genOpen = true"><Plus />生成邀请码</Button>
      </template>
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">邀请码</label>
          <input v-model="filters.kw" placeholder="邀请码" class="field-input w-52" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">状态</label>
          <Select v-model="filters.status" :options="statusOptions" class="w-32" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 概况 -->
    <Panel title="邀请码概况" subtitle="按当前筛选条件">
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">邀请码总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.total }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">未使用</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ stats.unused }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已使用</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.used }}</div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="邀请码列表" :subtitle="`${total} 个`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[20%]">邀请码</th>
              <th class="w-[10%]">状态</th>
              <th class="w-[18%]">生成时间</th>
              <th class="w-[18%]">使用时间</th>
              <th class="w-[14%]">使用者</th>
              <th class="col-center w-[10%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in pageRows" :key="c.id">
              <td class="font-mono text-[13px] font-medium">{{ c.code }}</td>
              <td>
                <Badge :variant="inviteStatus[c.status].variant">{{ inviteStatus[c.status].text }}</Badge>
              </td>
              <td class="text-xs">{{ c.addtime }}</td>
              <td class="text-xs">
                <span v-if="c.usetime">{{ c.usetime }}</span>
                <span v-else class="dim">—</span>
              </td>
              <td class="tabular-nums">
                <span v-if="c.uid" class="text-primary">{{ c.uid }}</span>
                <span v-else class="dim">—</span>
              </td>
              <td class="col-center">
                <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive">
                  <Trash2 class="size-4" />
                </Button>
              </td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="6" class="py-10 text-center dim">没有符合条件的邀请码</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="mt-4 border-t border-border/60 pt-4">
        <Pagination :page="safePage" :page-count="pageCount" :total="total" :page-size="pageSize" @change="go" />
      </div>
      <p class="mt-3 flex items-center gap-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        <span>仅邀请注册模式（注册登录设置开启「仅邀请注册」）下生效，商户注册时需填写有效邀请码。</span>
        <Button variant="ghost" size="sm" class="ml-auto shrink-0 text-destructive hover:text-destructive">
          <Eraser class="size-4" />清空已使用
        </Button>
      </p>
    </Panel>

    <!-- 生成邀请码抽屉 -->
    <Drawer v-model="genOpen" title="生成邀请码" subtitle="批量生成新的未使用邀请码" width="max-w-md">
      <div class="space-y-3.5">
        <div class="row-field">
          <label class="lbl">生成个数</label>
          <input v-model="genNum" type="number" min="1" max="1000" placeholder="生成的个数" class="field-input flex-1" />
        </div>
        <p class="text-xs text-muted-foreground">
          单次最多生成 1000 个。生成后邀请码状态为「未使用」，可在列表中查看与分发。
        </p>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="genOpen = false">取消</Button>
        <Button size="sm" @click="submitGen"><Plus />确认生成</Button>
      </template>
    </Drawer>
  </div>
</template>
