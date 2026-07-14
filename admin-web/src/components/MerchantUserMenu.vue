<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { onClickOutside } from '@vueuse/core'
import { ChevronDown, IdCard, KeyRound, SquarePen, Power } from 'lucide-vue-next'
import { Modal, Button } from '@/components/ui'

const router = useRouter()
const open = ref(false)
const root = ref<HTMLElement | null>(null)
onClickOutside(root, () => (open.value = false))

// 商户信息（原型 mock）
const merchant = { uid: 1001, name: '泰安优选商贸' }

// ===== 修改密码弹窗（商户端仅登录密码）=====
const pwdOpen = ref(false)
const pwd = reactive({ oldpwd: '', newpwd: '', newpwd2: '' })
function savePwd() {
  pwdOpen.value = false
  pwd.oldpwd = pwd.newpwd = pwd.newpwd2 = ''
}

function go(to: string) {
  open.value = false
  router.push(to)
}
function openPwd() {
  open.value = false
  pwdOpen.value = true
}
</script>

<template>
  <div ref="root" class="relative">
    <button
      class="flex items-center gap-2 rounded-lg py-1 pl-1 pr-1.5 transition-colors hover:bg-accent"
      @click="open = !open"
    >
      <img src="/images/avatar-default.png" alt="avatar" class="size-8 rounded-full object-cover" />
      <span class="hidden text-sm font-medium sm:block">{{ merchant.name }}</span>
      <ChevronDown :class="['size-4 text-muted-foreground transition-transform', open && 'rotate-180']" />
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
        class="absolute right-0 top-full z-50 mt-2 w-40 overflow-hidden rounded-md border border-border bg-popover shadow-md"
      >
        <!-- 顶部用户块 -->
        <div class="flex items-center gap-2 px-3 py-2.5">
          <img src="/images/avatar-default.png" alt="avatar" class="size-8 rounded-full object-cover" />
          <div class="min-w-0">
            <div class="truncate text-sm font-medium leading-tight text-foreground">{{ merchant.name }}</div>
            <div class="text-xs leading-tight text-muted-foreground">商户号 {{ merchant.uid }}</div>
          </div>
        </div>
        <div class="border-t border-border/70" />

        <!-- 菜单项 -->
        <div class="py-1">
          <button
            class="flex w-full items-center gap-2 px-3 py-2 text-sm text-foreground transition-colors hover:bg-accent"
            @click="go('/m/profile')"
          >
            <IdCard class="size-4 text-muted-foreground" />修改资料
          </button>
          <button
            class="flex w-full items-center gap-2 px-3 py-2 text-sm text-foreground transition-colors hover:bg-accent"
            @click="go('/m/api')"
          >
            <KeyRound class="size-4 text-muted-foreground" />API 信息
          </button>
          <button
            class="flex w-full items-center gap-2 px-3 py-2 text-sm text-foreground transition-colors hover:bg-accent"
            @click="openPwd"
          >
            <SquarePen class="size-4 text-muted-foreground" />修改密码
          </button>
          <div class="border-t border-border/70" />
          <button
            class="flex w-full items-center gap-2 px-3 py-2 text-sm text-foreground transition-colors hover:bg-accent"
            @click="go('/m/login')"
          >
            <Power class="size-4 text-muted-foreground" />退出登录
          </button>
        </div>
      </div>
    </transition>

    <!-- 修改密码弹窗 -->
    <Modal v-model="pwdOpen" title="修改登录密码">
      <div class="space-y-4">
        <div class="pwd-row">
          <label class="pwd-lbl"><span class="text-destructive">*</span> 原始密码</label>
          <div class="pwd-control">
            <input v-model="pwd.oldpwd" type="password" placeholder="请输入原始密码" class="field-input w-full" />
          </div>
        </div>
        <div class="pwd-row">
          <label class="pwd-lbl"><span class="text-destructive">*</span> 新密码</label>
          <div class="pwd-control">
            <input v-model="pwd.newpwd" type="password" placeholder="请输入新密码" class="field-input w-full" />
          </div>
        </div>
        <div class="pwd-row">
          <label class="pwd-lbl"><span class="text-destructive">*</span> 确认密码</label>
          <div class="pwd-control">
            <input v-model="pwd.newpwd2" type="password" placeholder="请再次输入新密码" class="field-input w-full" />
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="pwdOpen = false">取消</Button>
        <Button size="sm" @click="savePwd">保存</Button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.pwd-row {
  display: flex;
  align-items: center;
  gap: 1rem;
}
.pwd-lbl {
  width: 5.5rem;
  flex-shrink: 0;
  text-align: right;
  font-size: 0.875rem;
  white-space: nowrap;
  color: var(--foreground);
}
.pwd-control {
  min-width: 0;
  flex: 1 1 0%;
}
</style>
