<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Plus, MoreHorizontal, Settings2, Pencil, Trash2, ArrowRight, Shuffle, ListOrdered, CircleCheck } from 'lucide-vue-next'
import { Panel, Button, Badge, Switch } from '@/components/ui'
import { rolls, rollKind } from '@/lib/mock/rolls'

const list = rolls

// 轮询方式图标
const kindIcon: Record<number, any> = {
  0: ListOrdered,
  1: Shuffle,
  2: CircleCheck,
}
const kindVariant: Record<number, 'default' | 'warning' | 'success'> = {
  0: 'default',
  1: 'warning',
  2: 'success',
}

function toggleStatus(id: number) {
  const r = list.find((x) => x.id === id)
  if (r) r.status = r.status === 1 ? 0 : 1
}

const stats = computed(() => ({
  total: list.length,
  open: list.filter((r) => r.status === 1).length,
  closed: list.filter((r) => r.status === 0).length,
}))

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
    <Panel title="通道轮询" subtitle="将多个同支付方式的通道编为轮询组，按顺序 / 权重 / 首个启用策略分流流量">
      <template #actions>
        <Button size="sm"><Plus />新增轮询组</Button>
      </template>
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">轮询组总数</div>
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
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="轮询组列表" :subtitle="`${list.length} 个`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[6%]">ID</th>
              <th class="w-[16%]">显示名称</th>
              <th class="w-[11%]">支付方式</th>
              <th class="w-[12%]">轮询方式</th>
              <th class="w-[32%]">轮询规则</th>
              <th class="col-center w-[8%]">状态</th>
              <th class="col-center w-[8%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(r, si) in list" :key="r.id">
              <td class="font-medium tabular-nums">{{ r.id }}</td>
              <td class="truncate">{{ r.name }}</td>
              <td>{{ r.typeshowname }}</td>
              <td>
                <Badge :variant="kindVariant[r.kind]" class="inline-flex items-center gap-1">
                  <component :is="kindIcon[r.kind]" class="size-3" />
                  {{ rollKind[r.kind] }}
                </Badge>
              </td>
              <td>
                <div v-if="r.channels.length" class="flex flex-wrap items-center gap-1 text-[13px]">
                  <template v-for="(c, ci) in r.channels" :key="c.channel">
                    <span class="inline-flex items-center gap-0.5">
                      {{ c.channelname }}
                      <span v-if="r.kind === 1" class="text-xs text-muted-foreground">({{ c.weight }})</span>
                    </span>
                    <ArrowRight v-if="r.kind !== 1 && ci < r.channels.length - 1" class="size-3 text-muted-foreground" />
                    <span v-else-if="ci < r.channels.length - 1" class="text-muted-foreground">/</span>
                  </template>
                </div>
                <span v-else class="dim">未配置通道</span>
              </td>
              <td class="col-center">
                <div class="flex justify-center">
                  <Switch :model-value="r.status === 1" size="sm" @update:model-value="toggleStatus(r.id)" />
                </div>
              </td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(r.id)">
                    <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === r.id"
                    class="menu-panel absolute right-0 z-20 w-32"
                    :class="si >= list.length - 3 && list.length > 3
                      ? 'bottom-full mb-1.5'
                      : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <button class="menu-item" @click="openMenu = null">
                      <Settings2 class="size-4 shrink-0 opacity-70" /><span class="flex-1">配置通道</span>
                    </button>
                    <button class="menu-item" @click="openMenu = null">
                      <Pencil class="size-4 shrink-0 opacity-70" /><span class="flex-1">编辑</span>
                    </button>
                    <div class="menu-sep" />
                    <button class="menu-item menu-item-danger" @click="openMenu = null">
                      <Trash2 class="size-4 shrink-0 opacity-70" /><span class="flex-1">删除</span>
                    </button>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="!list.length">
              <td colspan="7" class="py-10 text-center dim">暂无轮询组</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        顺序轮询按配置顺序依次使用；随机轮询按权重(1-99)随机；首个启用仅用组内第一个可用通道。顺序 / 首个启用模式下权重值无效。
      </p>
    </Panel>
  </div>
</template>
