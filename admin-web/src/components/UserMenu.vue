<script setup lang="ts">
import { ref, reactive } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { ChevronDown, Settings, SquarePen, Power, Plus } from 'lucide-vue-next'
import { Modal, Button } from '@/components/ui'

const open = ref(false)
const root = ref<HTMLElement | null>(null)
onClickOutside(root, () => (open.value = false))

// ===== 账号设置弹窗 =====
const accountOpen = ref(false)
const account = reactive({ username: 'admin', name: '' })
const avatar = ref('')
const avatarInput = ref<HTMLInputElement | null>(null)
function pickAvatar() {
  avatarInput.value?.click()
}
function onAvatarChange(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file || !file.type.startsWith('image/')) return
  const reader = new FileReader()
  reader.onload = () => (avatar.value = reader.result as string)
  reader.readAsDataURL(file)
}
function saveAccount() {
  accountOpen.value = false
}

// ===== 修改密码弹窗（登录密码 + 支付密码）=====
const pwdOpen = ref(false)
// 登录密码
const pwd = reactive({ oldpwd: '', newpwd: '', newpwd2: '' })
// 支付密码（对齐 epay set.php mod=paypwd，用于转账接口与 API 退款，默认 123456）
const paypwd = reactive({ oldpwd: '', newpwd: '', newpwd2: '' })
function savePwd() {
  pwdOpen.value = false
  pwd.oldpwd = pwd.newpwd = pwd.newpwd2 = ''
  paypwd.oldpwd = paypwd.newpwd = paypwd.newpwd2 = ''
}

// ===== 菜单动作 =====
function openAccount() {
  open.value = false
  accountOpen.value = true
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
          <button
            class="flex w-full items-center gap-2 px-3 py-2 text-sm text-foreground transition-colors hover:bg-accent"
            @click="openAccount"
          >
            <Settings class="size-4 text-muted-foreground" />账号设置
          </button>
          <button
            class="flex w-full items-center gap-2 px-3 py-2 text-sm text-foreground transition-colors hover:bg-accent"
            @click="openPwd"
          >
            <SquarePen class="size-4 text-muted-foreground" />修改密码
          </button>
          <button
            class="flex w-full items-center gap-2 px-3 py-2 text-sm text-foreground transition-colors hover:bg-accent"
            @click="open = false"
          >
            <Power class="size-4 text-muted-foreground" />退出登录
          </button>
        </div>
      </div>
    </transition>

    <!-- 账号设置弹窗 -->
    <Modal v-model="accountOpen" title="账号设置">
      <div class="space-y-5">
        <!-- 头像 -->
        <div class="flex items-start gap-6">
          <label class="w-16 shrink-0 pt-1 text-right text-sm text-muted-foreground">头像</label>
          <button
            type="button"
            class="group flex size-24 flex-col items-center justify-center gap-1 overflow-hidden rounded border border-dashed border-border bg-muted/20 text-muted-foreground/60 transition-colors hover:border-primary hover:text-primary"
            @click="pickAvatar"
          >
            <img v-if="avatar" :src="avatar" alt="头像" class="size-full object-cover" />
            <template v-else>
              <Plus class="size-6" />
              <span class="text-xs">上传</span>
            </template>
          </button>
          <input ref="avatarInput" type="file" accept="image/*" class="hidden" @change="onAvatarChange" />
        </div>
        <!-- 用户名（只读展示） -->
        <div class="flex items-center gap-6">
          <label class="w-16 shrink-0 text-right text-sm text-muted-foreground">用户名</label>
          <span class="text-sm text-foreground">{{ account.username }}</span>
        </div>
        <!-- 名称 -->
        <div class="flex items-center gap-6">
          <label class="w-16 shrink-0 text-right text-sm text-muted-foreground">名称</label>
          <input v-model="account.name" placeholder="请输入用户名称" class="field-input flex-1" />
        </div>
      </div>
      <template #footer>
        <Button size="sm" @click="saveAccount">保存</Button>
        <Button variant="outline" size="sm" @click="accountOpen = false">取消</Button>
      </template>
    </Modal>

    <!-- 修改密码弹窗（登录密码 + 支付密码）-->
    <Modal v-model="pwdOpen" title="修改密码" width="max-w-lg">
      <div class="space-y-6">
        <!-- 登录密码 -->
        <section class="space-y-3">
          <h4 class="pwd-group-title">登录密码</h4>
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
              <p class="pwd-hint">修改密码时必填，不修改密码时留空</p>
            </div>
          </div>
          <div class="pwd-row">
            <label class="pwd-lbl"><span class="text-destructive">*</span> 确认密码</label>
            <div class="pwd-control">
              <input v-model="pwd.newpwd2" type="password" placeholder="请再次输入新密码" class="field-input w-full" />
            </div>
          </div>
        </section>

        <!-- 支付密码 -->
        <section class="space-y-3 border-t border-border/60 pt-5">
          <div>
            <h4 class="pwd-group-title">支付密码</h4>
            <p class="mt-1 text-xs text-muted-foreground">用于转账接口及 API 退款，默认为 123456</p>
          </div>
          <div class="pwd-row">
            <label class="pwd-lbl">原支付密码</label>
            <div class="pwd-control">
              <input v-model="paypwd.oldpwd" type="password" placeholder="修改支付密码时填写" class="field-input w-full" />
            </div>
          </div>
          <div class="pwd-row">
            <label class="pwd-lbl">新支付密码</label>
            <div class="pwd-control">
              <input v-model="paypwd.newpwd" type="password" placeholder="请输入新支付密码" class="field-input w-full" />
              <p class="pwd-hint">修改支付密码时必填，不修改时留空</p>
            </div>
          </div>
          <div class="pwd-row">
            <label class="pwd-lbl">确认支付密码</label>
            <div class="pwd-control">
              <input v-model="paypwd.newpwd2" type="password" placeholder="请再次输入新支付密码" class="field-input w-full" />
            </div>
          </div>
        </section>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="pwdOpen = false">取消</Button>
        <Button size="sm" @click="savePwd">保存</Button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
/* 修改密码弹窗：统一每行结构，标签定宽右对齐、与输入框顶部对齐，
   所有输入框左边缘对齐，带说明的行说明文字跟随输入框下方，不影响整体节奏 */
.pwd-group-title {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--foreground);
}
.pwd-row {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
}
.pwd-lbl {
  width: 6rem;
  flex-shrink: 0;
  padding-top: 0.5rem;
  text-align: right;
  font-size: 0.875rem;
  line-height: 1.25rem;
  white-space: nowrap;
  color: var(--foreground);
}
.pwd-control {
  min-width: 0;
  flex: 1 1 0%;
}
.pwd-hint {
  margin-top: 0.375rem;
  font-size: 0.75rem;
  color: var(--muted-foreground);
}
</style>
