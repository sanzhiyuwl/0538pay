<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  Plus,
  Pencil,
  Trash2,
  Users,
  Check,
  X,
  ArrowUpFromLine,
  ArrowDownToLine,
  Store,
} from 'lucide-vue-next'
import { Panel, Button, Badge, Switch } from '@/components/ui'
import {
  groups as allGroups,
  settleOpenText,
  groupBuyEnabled,
  capabilityList,
} from '@/lib/mock/groups'

const buyEnabled = ref(groupBuyEnabled)

// 按排序展示（sort 越小越靠前）
const groups = computed(() => [...allGroups].sort((a, b) => a.sort - b.sort))

const totalMerchants = computed(() => allGroups.reduce((a, g) => a + g.merchantCount, 0))
const onSaleCount = computed(() => allGroups.filter((g) => g.isbuy === 1).length)

function expireText(month: number) {
  return month === 0 ? '永久有效' : `${month} 个月`
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 概览 + 购买开关 -->
    <Panel title="用户组 / 套餐" subtitle="每个用户组即一个套餐：定义通道费率、结算规则与功能权限，可上架供商户购买">
      <template #actions>
        <Button size="sm"><Plus />新增套餐</Button>
      </template>
      <div class="flex flex-wrap items-center gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">套餐总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ allGroups.length }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">已上架</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ onSaleCount }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">覆盖商户</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ totalMerchants }}</div>
        </div>
      </div>
      <div class="mt-4 flex items-center justify-between border-t border-border/60 pt-3.5">
        <div>
          <div class="text-sm font-medium">套餐购买开关</div>
          <div class="text-xs text-muted-foreground">关闭后商户中心不展示套餐购买入口</div>
        </div>
        <Switch v-model="buyEnabled" />
      </div>
    </Panel>

    <!-- 套餐卡片 -->
    <div class="grid grid-cols-1 gap-2.5 lg:grid-cols-2 xl:grid-cols-4">
      <Panel v-for="g in groups" :key="g.gid" flush>
        <div class="p-4">
          <!-- 头部：名称 + 上架状态 -->
          <div class="flex items-start justify-between">
            <div>
              <div class="flex items-center gap-2">
                <span class="text-base font-semibold">{{ g.name }}</span>
                <span class="text-xs text-muted-foreground">GID {{ g.gid }}</span>
              </div>
              <Badge v-if="g.isbuy" variant="success" class="mt-1.5">已上架</Badge>
              <Badge v-else variant="muted" class="mt-1.5">未上架</Badge>
            </div>
            <div class="flex items-center gap-1 text-xs text-muted-foreground">
              <Store class="size-3.5" />{{ g.merchantCount }}
            </div>
          </div>

          <!-- 定价 -->
          <div class="mt-3 flex items-baseline gap-1.5">
            <span class="text-sm text-muted-foreground">¥</span>
            <span class="text-2xl font-semibold tabular-nums">{{ g.price }}</span>
            <span class="text-sm text-muted-foreground">/ {{ expireText(g.expire) }}</span>
          </div>

          <!-- 通道费率 -->
          <div class="mt-3.5 border-t border-border/60 pt-3">
            <div class="mb-1.5 text-xs font-medium text-muted-foreground">通道与费率</div>
            <div class="space-y-1">
              <div v-for="r in g.rates" :key="r.typename" class="flex items-center justify-between text-sm">
                <span>{{ r.typename }}</span>
                <span v-if="r.channel === '关闭'" class="text-xs text-muted-foreground">已关闭</span>
                <span v-else class="tabular-nums">
                  <span class="text-xs text-muted-foreground">{{ r.channel }}</span>
                  <b class="ml-1.5 text-primary">{{ r.rate }}%</b>
                </span>
              </div>
            </div>
          </div>

          <!-- 结算规则 -->
          <div class="mt-3 border-t border-border/60 pt-3 text-sm">
            <div class="mb-1.5 text-xs font-medium text-muted-foreground">结算</div>
            <div class="flex flex-wrap gap-x-4 gap-y-1 text-[13px]">
              <span class="text-muted-foreground">方式 <b class="text-foreground">{{ settleOpenText[g.config.settleOpen] }}</b></span>
              <span class="text-muted-foreground">周期 <b class="text-foreground">{{ g.config.settleType === '1' ? 'D+0' : 'D+1' }}</b></span>
              <span class="text-muted-foreground">费率 <b class="text-foreground tabular-nums">{{ g.config.settleRate }}%</b></span>
            </div>
          </div>

          <!-- 能力开关 -->
          <div class="mt-3 border-t border-border/60 pt-3">
            <div class="mb-1.5 text-xs font-medium text-muted-foreground">功能权限</div>
            <div class="grid grid-cols-2 gap-1">
              <div
                v-for="cap in capabilityList(g.config)"
                :key="cap.label"
                class="flex items-center gap-1.5 text-[13px]"
                :class="cap.on ? 'text-foreground' : 'text-muted-foreground/60'"
              >
                <Check v-if="cap.on" class="size-3.5 text-success" />
                <X v-else class="size-3.5" />
                {{ cap.label }}
              </div>
            </div>
          </div>

          <!-- 操作 -->
          <div class="mt-4 flex items-center gap-1.5 border-t border-border/60 pt-3">
            <Button variant="outline" size="sm"><Users />商户</Button>
            <Button variant="outline" size="sm"><Pencil />编辑</Button>
            <Button v-if="g.isbuy" variant="ghost" size="sm"><ArrowDownToLine />下架</Button>
            <Button v-else variant="ghost" size="sm"><ArrowUpFromLine />上架</Button>
            <Button
              v-if="g.gid !== 0"
              variant="ghost"
              size="icon"
              class="ml-auto text-destructive hover:text-destructive"
            >
              <Trash2 />
            </Button>
          </div>
        </div>
      </Panel>
    </div>

    <p class="px-1 text-xs text-muted-foreground">
      未设置用户组的商户归为「默认用户组」(GID 0)，自动使用可用支付通道与通道默认费率，该组不可删除。
    </p>
  </div>
</template>
