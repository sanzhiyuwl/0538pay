<script setup lang="ts">
import { reactive, ref } from 'vue'
import { Trash2, DatabaseZap, TriangleAlert } from 'lucide-vue-next'
import { Panel, Button, Modal } from '@/components/ui'
import { cleanData as apiClean } from '@/lib/api/clean'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

// 清理目标（表名对齐后端白名单 pay_*）
const cleanTables = [
  { key: 'order', label: '订单记录', table: 'pay_order' },
  { key: 'settle', label: '结算记录', table: 'pay_settle' },
  { key: 'record', label: '资金明细', table: 'pay_record' },
  { key: 'transfer', label: '付款记录', table: 'pay_transfer' },
  { key: 'psorder', label: '分账记录', table: 'pay_ps_order' },
]

// 每个表的自定义天数
const days = reactive<Record<string, string>>(
  Object.fromEntries(cleanTables.map((t) => [t.key, '90'])),
)

// 二次确认
const confirmOpen = ref(false)
const busy = ref(false)
const target = ref<{ key: string; label: string; days: number } | null>(null)

function askClean(t: { key: string; label: string }) {
  const d = Number(days[t.key])
  if (!(d >= 7)) {
    toast.error('清理天数不得小于 7 天')
    return
  }
  target.value = { key: t.key, label: t.label, days: d }
  confirmOpen.value = true
}

async function doClean() {
  if (!target.value || busy.value) return
  busy.value = true
  try {
    const res = await apiClean(target.value.key, target.value.days)
    toast.success(`已清理 ${res.deleted} 条${target.value.label}`)
    confirmOpen.value = false
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '清理失败')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
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
                <Button variant="outline" size="sm" class="text-destructive hover:text-destructive" @click="askClean(t)">
                  <Trash2 />立即清理
                </Button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </Panel>

    <!-- 二次确认 -->
    <Modal v-model="confirmOpen" title="确认清理数据" width="max-w-md">
      <div class="space-y-2 text-sm">
        <p>即将删除 <span class="font-medium text-foreground">{{ target?.label }}</span> 中 <span class="font-medium text-destructive">{{ target?.days }}</span> 天前的所有记录。</p>
        <p class="text-destructive">此操作不可恢复，请确认已备份数据库。</p>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="confirmOpen = false">取消</Button>
        <Button variant="destructive" size="sm" :disabled="busy" @click="doClean">确认清理</Button>
      </template>
    </Modal>
  </div>
</template>
