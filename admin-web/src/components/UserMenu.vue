<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { useRouter } from 'vue-router'
import { onClickOutside } from '@vueuse/core'
import { ChevronDown, Settings, SquarePen, KeyRound, Power, Plus } from 'lucide-vue-next'
import { Modal, Button } from '@/components/ui'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'
import { fetchProfile, updateProfile, changePassword, changePayPassword } from '@/lib/api/auth'
import { ApiError } from '@/lib/api/client'

const open = ref(false)
const root = ref<HTMLElement | null>(null)
onClickOutside(root, () => (open.value = false))

const router = useRouter()
const auth = useAuthStore()
const toast = useToast()

// 顶栏与菜单显示的账号名：优先真实昵称，回退 admin
const displayName = computed(() => auth.nickname || 'admin')

function logout() {
  open.value = false
  auth.logout()
  toast.success('已退出登录')
  router.replace('/login')
}

// ===== 账号设置弹窗 =====
const accountOpen = ref(false)
const account = reactive({ username: auth.nickname || 'admin', name: '' })
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
const savingAccount = ref(false)
async function saveAccount() {
  if (savingAccount.value) return
  savingAccount.value = true
  try {
    await updateProfile({ nickname: account.name.trim(), username: account.username.trim() })
    auth.setNickname(account.name.trim())
    toast.success('账号资料已保存')
    accountOpen.value = false
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    savingAccount.value = false
  }
}

// ===== 修改登录密码弹窗 =====
// 登录密码用于后台登录；支付密码（下方）独立，用于转账/结算/退款二次校验，对齐 epay admin_paypwd。
const pwdOpen = ref(false)
const pwd = reactive({ oldpwd: '', newpwd: '', newpwd2: '' })
const savingPwd = ref(false)
async function savePwd() {
  if (savingPwd.value) return
  if (!pwd.oldpwd) return toast.error('请输入原始密码')
  if (!pwd.newpwd || pwd.newpwd.length < 6) return toast.error('新密码至少 6 位')
  if (pwd.newpwd !== pwd.newpwd2) return toast.error('两次输入的新密码不一致')
  savingPwd.value = true
  try {
    await changePassword({ oldpwd: pwd.oldpwd, newpwd: pwd.newpwd, newpwd2: pwd.newpwd2 })
    toast.success('密码修改成功，请重新登录')
    pwd.oldpwd = pwd.newpwd = pwd.newpwd2 = ''
    pwdOpen.value = false
    // 改密后强制重新登录（对齐 epay "请重新登录"）
    auth.logout()
    router.replace('/login')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '修改失败')
  } finally {
    savingPwd.value = false
  }
}

// ===== 修改支付密码弹窗 =====
// 支付密码用于转账/结算/API 退款二次校验，独立于登录密码，出厂默认 123456（对齐 epay admin_paypwd）。
const payPwdOpen = ref(false)
const payPwd = reactive({ oldpwd: '', newpwd: '', newpwd2: '' })
const savingPayPwd = ref(false)
async function savePayPwd() {
  if (savingPayPwd.value) return
  if (!payPwd.oldpwd) return toast.error('请输入原支付密码')
  if (!payPwd.newpwd || payPwd.newpwd.length < 6) return toast.error('新支付密码至少 6 位')
  if (payPwd.newpwd !== payPwd.newpwd2) return toast.error('两次输入的新支付密码不一致')
  savingPayPwd.value = true
  try {
    await changePayPassword({ oldpwd: payPwd.oldpwd, newpwd: payPwd.newpwd, newpwd2: payPwd.newpwd2 })
    toast.success('支付密码修改成功')
    payPwd.oldpwd = payPwd.newpwd = payPwd.newpwd2 = ''
    payPwdOpen.value = false
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '修改失败')
  } finally {
    savingPayPwd.value = false
  }
}

// ===== 菜单动作 =====
async function openAccount() {
  open.value = false
  accountOpen.value = true
  // 拉真实资料回填（用户名 + 昵称）
  try {
    const p = await fetchProfile()
    account.username = p.username
    account.name = p.nickname
  } catch { /* 保留默认 */ }
}
function openPwd() {
  open.value = false
  pwdOpen.value = true
}
function openPayPwd() {
  open.value = false
  payPwdOpen.value = true
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
      <span class="hidden text-sm font-medium sm:block">{{ displayName }}</span>
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
        class="absolute right-0 top-full z-50 mt-2 w-40 overflow-hidden rounded-md border border-border bg-popover shadow-md"
      >
        <!-- 顶部用户块 -->
        <div class="flex items-center gap-2 px-3 py-2.5">
          <img
            src="/images/avatar-default.png"
            alt="avatar"
            class="size-8 rounded-full object-cover"
          />
          <div class="min-w-0">
            <div class="truncate text-sm font-medium leading-tight text-foreground">{{ displayName }}</div>
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
            @click="openPayPwd"
          >
            <KeyRound class="size-4 text-muted-foreground" />修改支付密码
          </button>
          <button
            class="flex w-full items-center gap-2 px-3 py-2 text-sm text-foreground transition-colors hover:bg-accent"
            @click="logout"
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

    <!-- 修改登录密码弹窗 -->
    <Modal v-model="pwdOpen" title="修改登录密码" width="max-w-lg">
      <div class="space-y-6">
        <!-- 登录密码 -->
        <section class="space-y-3">
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
              <p class="pwd-hint">至少 6 位。修改成功后需重新登录。</p>
            </div>
          </div>
          <div class="pwd-row">
            <label class="pwd-lbl"><span class="text-destructive">*</span> 确认密码</label>
            <div class="pwd-control">
              <input v-model="pwd.newpwd2" type="password" placeholder="请再次输入新密码" class="field-input w-full" />
            </div>
          </div>
        </section>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="pwdOpen = false">取消</Button>
        <Button size="sm" :disabled="savingPwd" @click="savePwd">保存</Button>
      </template>
    </Modal>

    <!-- 修改支付密码弹窗 -->
    <Modal v-model="payPwdOpen" title="修改支付密码" width="max-w-lg">
      <div class="space-y-6">
        <section class="space-y-3">
          <div class="pwd-row">
            <label class="pwd-lbl"><span class="text-destructive">*</span> 原支付密码</label>
            <div class="pwd-control">
              <input v-model="payPwd.oldpwd" type="password" placeholder="请输入原支付密码" class="field-input w-full" />
            </div>
          </div>
          <div class="pwd-row">
            <label class="pwd-lbl"><span class="text-destructive">*</span> 新支付密码</label>
            <div class="pwd-control">
              <input v-model="payPwd.newpwd" type="password" placeholder="请输入新支付密码" class="field-input w-full" />
              <p class="pwd-hint">至少 6 位，用于转账 / 结算 / 退款二次校验。</p>
            </div>
          </div>
          <div class="pwd-row">
            <label class="pwd-lbl"><span class="text-destructive">*</span> 确认密码</label>
            <div class="pwd-control">
              <input v-model="payPwd.newpwd2" type="password" placeholder="请再次输入新支付密码" class="field-input w-full" />
            </div>
          </div>
        </section>
      </div>
      <template #footer>
        <Button variant="outline" size="sm" @click="payPwdOpen = false">取消</Button>
        <Button size="sm" :disabled="savingPayPwd" @click="savePayPwd">保存</Button>
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
