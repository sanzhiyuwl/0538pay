<script setup lang="ts">
import { ref } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { ChevronDown, Settings, SquarePen, Power } from 'lucide-vue-next'

const open = ref(false)
const root = ref<HTMLElement | null>(null)
onClickOutside(root, () => (open.value = false))

const items = [
  { label: '账号设置', icon: Settings, to: '/settings' },
  { label: '修改密码', icon: SquarePen, to: '/settings' },
  { label: '退出登录', icon: Power, to: '' },
]
</script>

<template>
  <div ref="root" class="relative">
    <button
      class="flex items-center gap-2 rounded-lg py-1 pl-1 pr-1.5 transition-colors hover:bg-accent"
      @click="open = !open"
    >
      <img
        src="/images/avatar-default.png"
        alt="avatar"
        class="size-8 rounded-full object-cover"
      />
      <span class="hidden text-sm font-medium sm:block">admin</span>
      <ChevronDown
        :class="['size-4 text-muted-foreground transition-transform', open && 'rotate-180']"
      />
    </button>

    <!-- 下拉 -->
    <transition
      enter-active-class="transition duration-150 ease-out"
      leave-active-class="transition duration-100 ease-in"
      enter-from-class="opacity-0 translate-y-1"
      leave-to-class="opacity-0 translate-y-1"
    >
      <div
        v-if="open"
        class="absolute right-0 top-full z-50 mt-2 w-36 overflow-hidden rounded-md border border-border bg-popover shadow-md"
      >
        <!-- 顶部用户块 -->
        <div class="flex items-center gap-2 px-3 py-2.5">
          <img
            src="/images/avatar-default.png"
            alt="avatar"
            class="size-8 rounded-full object-cover"
          />
          <div class="min-w-0">
            <div class="truncate text-sm font-medium leading-tight text-foreground">admin</div>
            <div class="text-xs leading-tight text-muted-foreground">个人中心</div>
          </div>
        </div>
        <div class="border-t border-border/70" />

        <!-- 菜单项 -->
        <div class="py-1">
          <RouterLink
            v-for="it in items"
            :key="it.label"
            :to="it.to || '/'"
            class="flex items-center gap-2 px-3 py-2 text-sm text-foreground transition-colors hover:bg-accent"
            @click="open = false"
          >
            <component :is="it.icon" class="size-4 text-muted-foreground" />
            {{ it.label }}
          </RouterLink>
        </div>
      </div>
    </transition>
  </div>
</template>
