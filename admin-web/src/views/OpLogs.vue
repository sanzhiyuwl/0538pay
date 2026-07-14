<script setup lang="ts">
import { ref, computed } from 'vue'
import { Search, RotateCcw, ExternalLink } from 'lucide-vue-next'
import { Panel, Button, Badge, Select, Pagination } from '@/components/ui'
import { opLogs, opActionVariant, opActionOptions } from '@/lib/mock/admins'

// ===== 筛选 =====
const filters = ref({ kw: '', action: '' })

const filtered = computed(() => {
  return opLogs.filter((l) => {
    if (filters.value.action && l.action !== filters.value.action) return false
    if (filters.value.kw.trim()) {
      const v = filters.value.kw.trim()
      if (!`${l.admin}${l.target}`.includes(v)) return false
    }
    return true
  })
})

function resetFilters() {
  filters.value = { kw: '', action: '' }
  page.value = 1
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
</script>

<template>
  <div class="space-y-2.5">
    <!-- 筛选 -->
    <Panel title="操作日志" :subtitle="`共 ${total} 条`">
      <div class="filter-bar">
        <div class="filter-item">
          <label class="filter-label">日志搜索</label>
          <input v-model="filters.kw" placeholder="管理员 / 操作对象" class="field-input w-52" />
        </div>
        <div class="filter-item">
          <label class="text-sm text-muted-foreground">操作类型</label>
          <Select v-model="filters.action" :options="opActionOptions" class="w-32" />
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Button size="sm" @click="page = 1"><Search />搜索</Button>
          <Button variant="outline" size="sm" @click="resetFilters"><RotateCcw />重置</Button>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="操作日志列表" :subtitle="`${total} 条`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[8%]">ID</th>
              <th class="w-[14%]">管理员</th>
              <th class="w-[12%]">操作类型</th>
              <th class="w-[30%]">操作对象</th>
              <th class="w-[18%]">操作 IP</th>
              <th class="w-[18%]">时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="l in pageRows" :key="l.id">
              <td class="tabular-nums dim">{{ l.id }}</td>
              <td class="font-medium">{{ l.admin }}</td>
              <td><Badge :variant="opActionVariant[l.action] ?? 'default'">{{ l.action }}</Badge></td>
              <td class="truncate" :title="l.target">{{ l.target }}</td>
              <td>
                <a
                  :href="`https://www.ip138.com/iplookup.php?ip=${l.ip}`"
                  target="_blank"
                  rel="noreferrer"
                  class="inline-flex items-center gap-1 tabular-nums hover:text-primary"
                >
                  {{ l.ip }}<ExternalLink class="size-3 opacity-50" />
                </a>
              </td>
              <td class="text-xs">{{ l.date }}</td>
            </tr>
            <tr v-if="!pageRows.length">
              <td colspan="6" class="py-10 text-center dim">没有符合条件的操作日志</td>
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
