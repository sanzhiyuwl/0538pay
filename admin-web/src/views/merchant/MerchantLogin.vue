<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { Store, User, Lock, KeyRound, Hash } from 'lucide-vue-next'
import { Button } from '@/components/ui'

const router = useRouter()

// 登录方式：password 密码登录 / key 密钥登录
const mode = ref<'password' | 'key'>('password')
const form = ref({ user: '', pass: '', mid: '', mkey: '' })

function login() {
  // 原型：直接进工作台
  router.push('/m')
}

// 第三方快捷登录（原型占位）
const socials = [
  { key: 'alipay', label: '支付宝', color: 'text-[#1677ff]' },
  { key: 'wx', label: '微信', color: 'text-[#07c160]' },
  { key: 'qq', label: 'QQ', color: 'text-[#12b7f5]' },
]
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-content px-4">
    <div class="w-full max-w-sm">
      <!-- 品牌 -->
      <div class="mb-8 flex flex-col items-center gap-3">
        <div class="flex size-12 items-center justify-center rounded-xl bg-primary text-primary-foreground">
          <Store class="size-6" />
        </div>
        <div class="text-center">
          <div class="text-xl font-bold tracking-tight">0538<span class="text-primary">Pay</span> 商户中心</div>
          <div class="mt-1 text-sm text-muted-foreground">登录你的商户账户</div>
        </div>
      </div>

      <!-- 卡片 -->
      <div class="border border-border bg-background p-6 shadow-sm">
        <!-- 登录方式 Tab -->
        <div class="mb-5 flex gap-1 border-b border-border">
          <button
            class="-mb-px border-b-2 px-4 py-2 text-sm transition-colors"
            :class="mode === 'password' ? 'border-primary font-medium text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'"
            @click="mode = 'password'"
          >
            密码登录
          </button>
          <button
            class="-mb-px border-b-2 px-4 py-2 text-sm transition-colors"
            :class="mode === 'key' ? 'border-primary font-medium text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'"
            @click="mode = 'key'"
          >
            密钥登录
          </button>
        </div>

        <!-- 密码登录 -->
        <form v-if="mode === 'password'" class="space-y-3.5" @submit.prevent="login">
          <div class="login-field">
            <User class="login-icon" />
            <input v-model="form.user" placeholder="邮箱 / 手机号" class="login-input" />
          </div>
          <div class="login-field">
            <Lock class="login-icon" />
            <input v-model="form.pass" type="password" placeholder="登录密码" class="login-input" />
          </div>
          <!-- 极验验证码占位 -->
          <div class="flex h-10 items-center justify-center rounded border border-dashed border-border bg-muted/30 text-xs text-muted-foreground">
            滑动验证占位（极验）
          </div>
          <Button class="w-full" @click="login">登录</Button>
        </form>

        <!-- 密钥登录 -->
        <form v-else class="space-y-3.5" @submit.prevent="login">
          <div class="login-field">
            <Hash class="login-icon" />
            <input v-model="form.mid" placeholder="商户 ID" class="login-input" />
          </div>
          <div class="login-field">
            <KeyRound class="login-icon" />
            <input v-model="form.mkey" type="password" placeholder="商户密钥" class="login-input" />
          </div>
          <Button class="w-full" @click="login">登录</Button>
        </form>

        <!-- 辅助链接 -->
        <div class="mt-4 flex items-center justify-between text-sm">
          <RouterLink to="/m/findpwd" class="text-muted-foreground hover:text-primary">找回密码</RouterLink>
          <RouterLink to="/m/reg" class="text-primary hover:underline">注册商户</RouterLink>
        </div>

        <!-- 第三方快捷登录 -->
        <div class="mt-6">
          <div class="flex items-center gap-3 text-xs text-muted-foreground">
            <div class="h-px flex-1 bg-border" />
            <span>快捷登录</span>
            <div class="h-px flex-1 bg-border" />
          </div>
          <div class="mt-4 flex justify-center gap-3">
            <button
              v-for="s in socials"
              :key="s.key"
              class="flex size-10 items-center justify-center rounded-full border border-border transition-colors hover:border-primary/40 hover:bg-accent"
              :title="`${s.label}登录`"
            >
              <span :class="['text-xs font-semibold', s.color]">{{ s.label.slice(0, 1) }}</span>
            </button>
          </div>
        </div>
      </div>

      <p class="mt-6 text-center text-xs text-muted-foreground">
        登录即表示同意 <a class="text-primary hover:underline" href="#">服务协议</a> 与 <a class="text-primary hover:underline" href="#">隐私政策</a>
      </p>
    </div>
  </div>
</template>

<style scoped>
.login-field {
  position: relative;
  display: flex;
  align-items: center;
}
.login-icon {
  position: absolute;
  left: 0.75rem;
  width: 1rem;
  height: 1rem;
  color: var(--muted-foreground);
  pointer-events: none;
}
.login-input {
  width: 100%;
  height: 2.5rem;
  padding: 0 0.75rem 0 2.25rem;
  border: 1px solid var(--border);
  background: var(--background);
  font-size: 0.875rem;
  color: var(--foreground);
  transition: border-color 0.15s;
}
.login-input:focus {
  outline: none;
  border-color: var(--primary);
}
.login-input::placeholder {
  color: var(--muted-foreground);
}
</style>
