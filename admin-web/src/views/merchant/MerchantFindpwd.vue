<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Store, Mail, Smartphone, MessageSquareCode } from 'lucide-vue-next'
import { Button, Select } from '@/components/ui'
import { fetchCaptcha, merchantFindPwd } from '@/lib/api/merchantAuth'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const router = useRouter()
const toast = useToast()

const type = ref('email')
const typeOptions = [
  { value: 'email', label: '使用邮箱找回' },
  { value: 'phone', label: '使用手机找回' },
]
const form = ref({ account: '', code: '', pwd: '', pwd2: '' })

// 图形验证码
const captchaToken = ref('')
const captchaSvg = ref('')
async function loadCaptcha() {
  try {
    const res = await fetchCaptcha()
    captchaToken.value = res.token
    captchaSvg.value = res.svg
  } catch {
    captchaSvg.value = ''
  }
}
onMounted(loadCaptcha)

const canSubmit = computed(() => {
  const f = form.value
  return !!(f.account && f.code && f.pwd && f.pwd === f.pwd2)
})
const loading = ref(false)
async function submit() {
  if (!canSubmit.value) {
    if (form.value.pwd && form.value.pwd !== form.value.pwd2) toast.error('两次输入的密码不一致')
    return
  }
  loading.value = true
  try {
    await merchantFindPwd({
      type: type.value,
      account: form.value.account.trim(),
      password: form.value.pwd,
      captcha_token: captchaToken.value,
      captcha: form.value.code.trim(),
    })
    toast.success('密码已重置，请用新密码登录')
    router.push('/m/login')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '重置失败')
    loadCaptcha()
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-content px-4 py-8">
    <div class="w-full max-w-sm">
      <div class="mb-6 flex flex-col items-center gap-3">
        <div class="flex size-12 items-center justify-center rounded-xl bg-primary text-primary-foreground"><Store class="size-6" /></div>
        <div class="text-center">
          <div class="text-xl font-bold tracking-tight">找回密码</div>
          <div class="mt-1 text-sm text-muted-foreground">通过邮箱或手机重置登录密码</div>
        </div>
      </div>

      <div class="border border-border bg-background p-6 shadow-sm">
        <form class="space-y-3.5" @submit.prevent="submit">
          <Select v-model="type" :options="typeOptions" class="w-full" />
          <div class="af">
            <component :is="type === 'phone' ? Smartphone : Mail" class="af-icon" />
            <input v-model="form.account" :placeholder="type === 'phone' ? '注册手机号' : '注册邮箱'" class="af-input" />
          </div>
          <div class="af">
            <MessageSquareCode class="af-icon" />
            <input v-model="form.code" placeholder="图形验证码" class="af-input pr-24" />
            <button type="button" class="af-captcha" title="点击刷新" @click="loadCaptcha">
              <span v-if="captchaSvg" v-html="captchaSvg"></span>
              <span v-else class="text-xs text-muted-foreground px-3">加载中</span>
            </button>
          </div>
          <div class="af">
            <Lock class="af-icon" />
            <input v-model="form.pwd" type="password" placeholder="新密码" class="af-input" />
          </div>
          <div class="af">
            <Lock class="af-icon" />
            <input v-model="form.pwd2" type="password" placeholder="确认新密码" class="af-input" />
          </div>
          <Button class="w-full" :disabled="!canSubmit || loading" @click="submit">{{ loading ? '重置中…' : '重置密码' }}</Button>
        </form>

        <div class="mt-4 text-center text-sm text-muted-foreground">
          想起密码了？<RouterLink to="/m/login" class="text-primary hover:underline">返回登录</RouterLink>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.af { position: relative; display: flex; align-items: center; }
.af-icon { position: absolute; left: 0.75rem; width: 1rem; height: 1rem; color: var(--muted-foreground); pointer-events: none; }
.af-input { width: 100%; height: 2.5rem; padding: 0 0.75rem 0 2.25rem; border: 1px solid var(--border); background: var(--background); font-size: 0.875rem; color: var(--foreground); transition: border-color 0.15s; }
.af-input:focus { outline: none; border-color: var(--primary); }
.af-input::placeholder { color: var(--muted-foreground); }
.af-suffix { position: absolute; right: 0.5rem; font-size: 0.75rem; color: var(--primary); padding: 0 0.5rem; }
.af-suffix:disabled { color: var(--muted-foreground); }
.af-captcha { position: absolute; right: 0.5rem; height: 30px; display: flex; align-items: center; border: 1px solid var(--border); border-radius: 8px; background: #f3f4f6; overflow: hidden; cursor: pointer; }
.af-captcha :deep(svg) { display: block; height: 30px; width: 75px; }
</style>
