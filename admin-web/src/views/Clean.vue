<script setup lang="ts">
import { reactive } from 'vue'
import { Trash2, DatabaseZap, RefreshCw, TriangleAlert } from 'lucide-vue-next'
import { Panel, Button } from '@/components/ui'
import { cleanTables } from '@/lib/mock/settings'

// 每个表的自定义天数
const days = reactive<Record<string, string>>(
  Object.fromEntries(cleanTables.map((t) => [t.key, '30'])),
)

function cleanCache() {
  // 原型：仅提示
}
function cleanData() {
  // 原型：仅提示（真实环境需 confirm + 后端删除）
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 缓存清理 -->
    <Panel title="数据清理" subtitle="定期清理历史数据有助于提升数据库性能，删除后不可恢复">
      <div class="flex items-center justify-between bg-muted/40 p-4">
        <div class="flex items-center gap-3">
          <div class="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <RefreshCw class="size-[18px]" />
          </div>
          <div>
            <div class="text-sm font-medium">系统设置缓存</div>
            <div class="text-xs text-muted-foreground">清理配置缓存与 OPcache，修改设置后未生效时使用</div>
          </div>
        </div>
        <Button variant="outline" size="sm" @click="cleanCache"><RefreshCw />清理缓存</Button>
      </div>
    </Panel>

    <!-- 历史数据清理 -->
    <Panel title="历史数据清理" subtitle="按数据类型删除指定天数之前的记录">
      <div class="mb-3 flex items-start gap-2 bg-destructive/[0.06] p-3 text-xs text-destructive">
        <TriangleAlert class="mt-0.5 size-4 shrink-0" />
        <span>删除操作不可恢复，请提前备份数据库。建议保留至少 30 天的近期数据以便对账。</span>
      </div>
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[28%]">数据类型</th>
              <th class="w-[24%]">数据表</th>
              <th class="w-[28%]">清理范围</th>
              <th class="col-center w-[20%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="t in cleanTables" :key="t.key">
              <td>
                <div class="flex items-center gap-2">
                  <DatabaseZap class="size-4 shrink-0 text-muted-foreground" />
                  <span class="font-medium">{{ t.label }}</span>
                </div>
              </td>
              <td class="font-mono text-[13px] dim">{{ t.table }}</td>
              <td>
                <div class="flex items-center gap-1.5 text-sm">
                  <span class="text-muted-foreground">删除</span>
                  <input v-model="days[t.key]" class="field-input w-20 text-center" />
                  <span class="text-muted-foreground">天前的记录</span>
                </div>
              </td>
              <td class="col-center">
                <Button variant="outline" size="sm" class="text-destructive hover:text-destructive" @click="cleanData">
                  <Trash2 />立即清理
                </Button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </Panel>
  </div>
</template>
