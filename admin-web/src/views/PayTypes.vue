<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Plus, MoreHorizontal, Pencil, Trash2, ReceiptText, Monitor, Smartphone, MonitorSmartphone } from 'lucide-vue-next'
import { Panel, Button, Switch } from '@/components/ui'
import { payTypeList, deviceText, calcPayTypeStats } from '@/lib/mock/paytypes'
import { formatMoney } from '@/lib/utils'

const types = payTypeList
const stats = computed(() => calcPayTypeStats(types))

// 设备图标
const deviceIcon: Record<number, any> = {
  0: MonitorSmartphone,
  1: Monitor,
  2: Smartphone,
}

function toggleStatus(id: number) {
  const t = types.find((x) => x.id === id)
  if (t) t.status = t.status === 1 ? 0 : 1
}

// 行操作菜单
const openMenu = ref<number | null>(null)
function toggleMenu(id: number) {
  openMenu.value = openMenu.value === id ? null : id
}
function closeMenu() {
  openMenu.value = null
}
onMounted(() => window.addEventListener('click', closeMenu))
onUnmounted(() => window.removeEventListener('click', closeMenu))
</script>

<template>
  <div class="space-y-2.5">
    <!-- 概况 -->
    <Panel title="支付方式" subtitle="前端可选的收款方式，每个支付方式可关联多个支付通道">
      <template #actions>
        <Button size="sm"><Plus />新增支付方式</Button>
      </template>
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">方式总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.total }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已开启</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ stats.open }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已关闭</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-muted-foreground">{{ stats.closed }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">今日收款合计</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-primary"><span class="mr-0.5 text-xs font-normal text-muted-foreground">¥</span>{{ formatMoney(stats.todayTotal) }}</div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="支付方式列表" :subtitle="`${types.length} 个`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[8%]">ID</th>
              <th class="w-[16%]">调用值</th>
              <th class="w-[20%]">显示名称</th>
              <th class="w-[16%]">支持设备</th>
              <th class="num w-[16%]">今日收款</th>
              <th class="col-center w-[10%]">状态</th>
              <th class="col-center w-[10%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(t, si) in types" :key="t.id">
              <td class="font-medium tabular-nums">{{ t.id }}</td>
              <td class="font-mono text-[13px] text-primary">{{ t.name }}</td>
              <td>
                <div class="flex items-center gap-2">
                  <span class="grid size-6 place-items-center rounded bg-primary/10 text-xs font-medium text-primary">
                    {{ t.showname[0] }}
                  </span>
                  {{ t.showname }}
                </div>
              </td>
              <td>
                <span class="inline-flex items-center gap-1.5 text-muted-foreground">
                  <component :is="deviceIcon[t.device]" class="size-4" />
                  {{ deviceText[t.device] }}
                </span>
              </td>
              <td class="num tabular-nums">
                <span v-if="+t.today > 0"><span class="dim text-xs">¥</span>{{ t.today }}</span>
                <span v-else class="dim">—</span>
              </td>
              <td class="col-center">
                <div class="flex justify-center">
                  <Switch :model-value="t.status === 1" size="sm" @update:model-value="toggleStatus(t.id)" />
                </div>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(t.id)">
                    <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === t.id"
                    class="menu-panel absolute right-0 z-20 w-32"
                    :class="si >= types.length - 3 && types.length > 3
                      ? 'bottom-full mb-1.5'
                      : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <button class="menu-item" @click="openMenu = null">
                      <Pencil class="size-4 shrink-0 opacity-70" /><span class="flex-1">编辑</span>
                    </button>
                    <button class="menu-item" @click="openMenu = null">
                      <ReceiptText class="size-4 shrink-0 opacity-70" /><span class="flex-1">订单</span>
                    </button>
                    <div class="menu-sep" />
                    <button
                      class="menu-item"
                      :class="t.id < 4 ? 'cursor-not-allowed opacity-40' : 'menu-item-danger'"
                      :disabled="t.id < 4"
                      :title="t.id < 4 ? '系统自带支付方式不支持删除' : ''"
                      @click="openMenu = null"
                    >
                      <Trash2 class="size-4 shrink-0 opacity-70" /><span class="flex-1">删除</span>
                    </button>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="!types.length">
              <td colspan="7" class="py-10 text-center dim">暂无支付方式</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        系统自带的支付方式（ID &lt; 4）不支持删除。同一「调用值 + 支持设备」不能重复。
      </p>
    </Panel>
  </div>
</template>
