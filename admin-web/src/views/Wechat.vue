<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Plus, MoreHorizontal, Pencil, Trash2, FlaskConical, MessageSquare, Smartphone } from 'lucide-vue-next'
import { Panel, Button, Badge } from '@/components/ui'
import { weixinApps, weixinType, calcWeixinStats } from '@/lib/mock/weixin'

const list = weixinApps
const stats = computed(() => calcWeixinStats(list))

const typeIcon: Record<number, any> = {
  0: MessageSquare,
  1: Smartphone,
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
    <Panel title="公众号 / 小程序" subtitle="配置微信服务号 / 小程序的 APPID 与密钥，用于 JSAPI 支付、网页授权等场景">
      <template #actions>
        <Button size="sm"><Plus />新增配置</Button>
      </template>
      <div class="flex flex-wrap gap-x-10 gap-y-4">
        <div>
          <div class="text-[13px] text-muted-foreground">配置总数</div>
          <div class="mt-1 text-xl font-normal tabular-nums">{{ stats.total }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">微信服务号</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-primary">{{ stats.official }}</div>
        </div>
        <div>
          <div class="text-[13px] text-muted-foreground">微信小程序</div>
          <div class="mt-1 text-xl font-normal tabular-nums text-success">{{ stats.mini }}</div>
        </div>
      </div>
    </Panel>

    <!-- 列表 -->
    <Panel title="配置列表" :subtitle="`${list.length} 个`">
      <div class="overflow-x-auto">
        <table class="tbl w-full table-fixed">
          <thead>
            <tr>
              <th class="w-[8%]">ID</th>
              <th class="w-[16%]">类别</th>
              <th class="w-[24%]">名称</th>
              <th class="w-[24%]">APPID</th>
              <th class="w-[18%]">APPSECRET</th>
              <th class="col-center w-[10%]">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(w, si) in list" :key="w.id">
              <td class="font-medium tabular-nums">{{ w.id }}</td>
              <td>
                <Badge :variant="w.type === 1 ? 'success' : 'default'" class="inline-flex items-center gap-1">
                  <component :is="typeIcon[w.type]" class="size-3" />
                  {{ weixinType[w.type] }}
                </Badge>
              </td>
              <td class="truncate">{{ w.name }}</td>
              <td class="font-mono text-[13px] text-primary">{{ w.appid }}</td>
              <td class="font-mono text-xs dim">{{ w.appsecret }}</td>
              <td class="col-center">
                <div class="relative inline-block">
                  <Button variant="ghost" size="sm" @click.stop="toggleMenu(w.id)">
                    <MoreHorizontal class="size-4" />
                  </Button>
                  <div
                    v-if="openMenu === w.id"
                    class="menu-panel absolute right-0 z-20 w-32"
                    :class="si >= list.length - 3 && list.length > 3
                      ? 'bottom-full mb-1.5'
                      : 'top-full mt-1.5'"
                    @click.stop
                  >
                    <button class="menu-item" @click="openMenu = null">
                      <Pencil class="size-4 shrink-0 opacity-70" /><span class="flex-1">编辑</span>
                    </button>
                    <button class="menu-item" @click="openMenu = null">
                      <FlaskConical class="size-4 shrink-0 opacity-70" /><span class="flex-1">测试</span>
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
              <td colspan="6" class="py-10 text-center dim">暂无公众号 / 小程序配置</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="mt-3 border-t border-border/60 pt-3 text-xs text-muted-foreground">
        服务号需在【公众平台→功能设置】配置网页授权域名，支付还需在商户平台设置 JSAPI 支付授权目录；小程序需在【小程序后台→开发设置】配置 request 合法域名并在微信支付后台绑定。
      </p>
    </Panel>
  </div>
</template>
