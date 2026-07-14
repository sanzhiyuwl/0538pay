<script setup lang="ts">
import { ref, computed } from 'vue'
import { Search, RotateCcw } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination } from '@/components/ui'
import { auditLogs, auditActionText, type AuditAction } from '@/lib/mock/console'

// ===== 筛选 =====
const actionOptions = [
  { value: '', label: '全部操作' },
  ...(Object.keys(auditActionText) as AuditAction[]).map((k) => ({ value: k, label: auditActionText[k].text })),
]
const operatorOptions = computed(() => {
  const set = Array.from(new Set(auditLogs.map((l) => l.operator)))
  return [{ value: '', label: '全部操作者' }, ...set.map((o) => ({ value: o, label: o }))]
})

const filters = ref({ kw: '', action: '', operator: '' })
const filtered = computed(() =>
  auditLogs.filter((l) => {
    if (filters.value.action && l.action !== filters.value.action) return false
    if (filters.value.operator && l.operator !== filters.value.operator) return false
    if (filters.value.kw.trim()) {
      const v = filters.value.kw.trim()
      if (!`${l.target}${l.detail}${l.ip}`.includes(v)) return false
    }
    return true
  }),
)
function resetFilters() {
  filters.value = { kw: '', action: '', operator: '' }
  page.value = 1
}

// ===== 分页 =====
const page = ref(1)
const pageSize = 12
const total = computed(() => filtered.value.length)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const safePage = computed(() => Math.min(page.value, pageCount.value))
const pageRows = computed(() =>
  filtered.value.slice((safePage.value - 1) * pageSize, safePage.value * pageSize),
)
function go(p: number) {
  page.value = Math.min(Math.max(1, p), pageCount.value)
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="操作审计" subtitle="控制台所有敏感操作留痕：创建 / 停用 / 改权限 / 免密登录等">
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">内容搜索</label>
          <input v-model="filters.kw" placeholder="目标分站 / 详情 / IP" class="field-input w-52" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">操作类型</label>
          <Select v-model="filters.action" :options="actionOptions" class="w-32" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">操作者</label>
          <Select v-model="filters.operator" :options="operatorOptions" class="w-32" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="page = 1"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 审计日志 -->
    <Panel title="审计日志" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full">
          <thead>
            <tr>
              <th class="num">日志 ID</th>
              <th>操作者</th>
              <th class="col-center">操作类型</th>
              <th>目标分站</th>
              <th>详情</th>
              <th>操作 IP</th>
              <th>时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="l in pageRows" :key="l.id">
              <td class="num tabular-nums text-muted-foreground">{{ l.id }}</td>
              <td class="font-mono text-[13px]">{{ l.operator }}</td>
              <td class="col-center"><Badge :variant="auditActionText[l.action].variant">{{ auditActionText[l.action].text }}</Badge></td>
              <td class="font-medium">{{ l.target }}</td>
              <td class="text-sm text-muted-foreground">{{ l.detail }}</td>
              <td class="font-mono text-[13px] tabular-nums text-muted-foreground">{{ l.ip }}</td>
              <td class="text-xs text-muted-foreground tabular-nums">{{ l.time }}</td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="7" class="py-10 text-center dim">没有符合条件的日志</td>
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
