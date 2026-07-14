<script setup lang="ts">
import { ref } from 'vue'
import { Check, LayoutTemplate } from 'lucide-vue-next'
import { Panel, Button, Badge } from '@/components/ui'
import { templates, currentTemplate } from '@/lib/mock/sysconfig'

const active = ref(currentTemplate)
function pick(name: string) {
  active.value = name
}
function save() {}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="首页模板配置" :subtitle="`共 ${templates.length} 套官网首页模板，当前使用：${active}`">
      <template #actions>
        <Button size="sm" @click="save"><Check />应用模板</Button>
      </template>
      <div class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
        <button
          v-for="t in templates"
          :key="t.name"
          type="button"
          class="group relative flex flex-col overflow-hidden border text-left transition-colors"
          :class="active === t.name ? 'border-primary ring-1 ring-primary' : 'border-border hover:border-primary/50'"
          @click="pick(t.name)"
        >
          <!-- 预览缩略图占位（原型无真实截图，用占位块） -->
          <div class="flex aspect-[16/10] items-center justify-center bg-muted/40 text-muted-foreground/40">
            <LayoutTemplate class="size-8" />
          </div>
          <div class="flex items-center justify-between gap-1 border-t border-border px-3 py-2">
            <span class="truncate text-sm">{{ t.label }}</span>
            <Badge v-if="active === t.name" variant="success" class="shrink-0">当前</Badge>
          </div>
          <div
            v-if="active === t.name"
            class="absolute right-2 top-2 flex size-6 items-center justify-center rounded-full bg-primary text-primary-foreground"
          >
            <Check class="size-4" />
          </div>
        </button>
      </div>
      <p class="mt-4 border-t border-border/60 pt-4 text-xs text-muted-foreground">
        模板对应 template/ 目录下的皮肤（default 及 index1~10）。选择后点「应用模板」切换官网首页外观。
      </p>
    </Panel>
  </div>
</template>
