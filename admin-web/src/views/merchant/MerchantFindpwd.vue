<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Eye, EyeOff, CheckCircle2, Zap } from 'lucide-vue-next'
import { Select } from '@/components/ui'
import { fetchCaptcha, merchantFindPwd } from '@/lib/api/merchantAuth'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import { useSiteStore } from '@/stores/site'

const router = useRouter()
const toast = useToast()

// 品牌名来自后台「网站设置 / 网站信息」，实时联动；末尾 Pay/PAY 拆出高亮
const siteStore = useSiteStore()
onMounted(() => siteStore.hydrate())
const brandName = computed(() => siteStore.config.merchantName || '三只鱼PAY')
const brand = computed(() => {
  const m = brandName.value.match(/^(.*?)(pay)$/i)
  return m ? { lead: m[1], accent: m[2] } : { lead: brandName.value, accent: '' }
})

const type = ref('email')
const typeOptions = [
  { value: 'email', label: '使用邮箱找回' },
  { value: 'phone', label: '使用手机找回' },
]
const form = ref({ account: '', code: '', pwd: '', pwd2: '' })
const showPwd = ref(false)

// 图形验证码（自研，代替短信/邮箱 OTP）
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

const highlights = [
  '邮箱 / 手机双通道，找回更便捷',
  '图形验证码校验，账户更安全',
  '重置即时生效，立即用新密码登录',
]
</script>

<template>
  <div class="auth">
    <section class="card">
      <!-- 左：品牌蓝装饰面板 -->
      <aside class="stage">
        <div class="stage-head">
          <span class="stage-logo"><Zap class="size-5" /></span>
          <span class="stage-name">{{ brand.lead }}<b v-if="brand.accent">{{ brand.accent }}</b></span>
        </div>

        <div class="stage-copy">
          <h2>忘记密码<br /><em>安全找回</em></h2>
        </div>

        <ul class="stage-list">
          <li v-for="h in highlights" :key="h"><CheckCircle2 class="size-4" />{{ h }}</li>
        </ul>
      </aside>

      <!-- 右：表单面板 -->
      <div class="panel">
        <div class="panel-inner">
          <header class="c-head">
            <h1><b>找回</b>密码</h1>
            <p class="c-sub">通过注册邮箱或手机重置你的登录密码</p>
          </header>

          <form class="fields" @submit.prevent="submit">
            <div class="field">
              <Select v-model="type" :options="typeOptions" class="w-full" />
            </div>
            <div class="field">
              <input
                v-model="form.account"
                class="f-input"
                :placeholder="type === 'phone' ? '输入注册手机号' : '输入注册邮箱'"
                autocomplete="username"
              />
            </div>
            <div class="field">
              <input v-model="form.code" class="f-input has-captcha" placeholder="输入右侧图形验证码" autocomplete="off" />
              <button type="button" class="f-captcha" title="点击刷新" @click="loadCaptcha">
                <span v-if="captchaSvg" v-html="captchaSvg"></span>
                <span v-else class="f-captcha-ph">加载中</span>
              </button>
            </div>
            <div class="field">
              <input
                v-model="form.pwd"
                :type="showPwd ? 'text' : 'password'"
                class="f-input"
                placeholder="输入新密码"
                autocomplete="new-password"
              />
              <button type="button" class="f-eye" tabindex="-1" @click="showPwd = !showPwd">
                <Eye v-if="!showPwd" class="size-4" /><EyeOff v-else class="size-4" />
              </button>
            </div>
            <div class="field">
              <input
                v-model="form.pwd2"
                :type="showPwd ? 'text' : 'password'"
                class="f-input"
                placeholder="再次输入新密码"
                autocomplete="new-password"
              />
            </div>
            <button class="submit" type="submit" :disabled="!canSubmit || loading">
              {{ loading ? '重置中…' : '重置密码' }}
            </button>
          </form>
        </div>

        <!-- 底部条 -->
        <div class="foot">
          <span>想起密码了？<RouterLink to="/m/login">返回登录</RouterLink></span>
          <RouterLink to="/m/reg">商户注册</RouterLink>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.auth {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  overflow: hidden;
  background:
    radial-gradient(100% 80% at 50% 0%, color-mix(in oklch, var(--primary) 8%, #fff) 0%, #fff 55%),
    #eef2f8;
}

/* ── 双栏卡片 ── */
.card {
  position: relative;
  display: grid;
  grid-template-columns: 300px 1fr;
  width: 100%;
  max-width: 780px;
  overflow: hidden;
  border-radius: 16px;
  background: #fff;
  box-shadow: 0 28px 70px -28px rgba(16, 42, 100, 0.32);
  animation: card-in 0.45s cubic-bezier(0.23, 1, 0.32, 1) both;
}
@keyframes card-in {
  from {
    opacity: 0;
    transform: translateY(14px) scale(0.99);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

/* ── 左：品牌蓝装饰面板 ── */
.stage {
  position: relative;
  overflow: hidden;
  padding: 34px 30px;
  display: flex;
  flex-direction: column;
  color: #eef4ff;
  background:
    radial-gradient(120% 70% at 20% 10%, rgba(255, 255, 255, 0.16) 0%, transparent 46%),
    linear-gradient(160deg, #2f7bff 0%, #2563eb 48%, #1b47c4 100%);
}
.stage::before {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background-image: radial-gradient(rgba(255, 255, 255, 0.1) 1.4px, transparent 1.5px);
  background-size: 22px 22px;
  mask-image: radial-gradient(80% 60% at 30% 18%, #000 20%, transparent 76%);
}
.stage > * {
  position: relative;
  z-index: 1;
}
.stage-head {
  display: flex;
  align-items: center;
  gap: 9px;
}
.stage-logo {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 8px;
  color: var(--primary);
  background: #fff;
  box-shadow: 0 6px 16px -6px rgba(0, 0, 0, 0.4);
}
.stage-name {
  font-size: 16px;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: #fff;
}
.stage-name b {
  color: #cfe0ff;
}
.stage-copy {
  margin-top: 34px;
}
.stage-copy h2 {
  font-size: 26px;
  line-height: 1.35;
  font-weight: 800;
  letter-spacing: 0.01em;
  margin: 0;
  color: #fff;
}
.stage-copy h2 em {
  position: relative;
  font-style: normal;
  white-space: nowrap;
}
.stage-copy h2 em::after {
  content: '';
  position: absolute;
  left: -2px;
  right: -2px;
  bottom: -6px;
  height: 7px;
  border-radius: 999px;
  background: #ffd43b;
  transform: rotate(-1.2deg);
  opacity: 0.9;
}
.stage-list {
  margin: 40px 0 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 15px;
}
.stage-list li {
  display: flex;
  align-items: center;
  gap: 9px;
  font-size: 13px;
  color: #dbe8ff;
}
.stage-list svg {
  flex-shrink: 0;
  color: #a9c6ff;
}

/* ── 右：表单面板 ── */
.panel {
  position: relative;
  display: flex;
  flex-direction: column;
}
.panel-inner {
  flex: 1;
  padding: 40px 44px 0;
}
.c-head h1 {
  font-size: 22px;
  font-weight: 700;
  letter-spacing: 0.02em;
  color: var(--foreground);
  margin: 0;
}
.c-head h1 b {
  color: var(--primary);
  font-weight: 700;
}
.c-sub {
  margin: 9px 0 0;
  font-size: 12.5px;
  color: var(--muted-foreground);
}

.fields {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-top: 26px;
}
.field {
  position: relative;
  display: flex;
  align-items: center;
}
.f-input {
  width: 100%;
  height: 46px;
  padding: 0 42px 0 16px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: #fff;
  font-size: 14px;
  color: var(--foreground);
  transition:
    border-color 0.15s,
    box-shadow 0.15s;
}
.f-input.has-captcha {
  padding-right: 104px;
}
.f-input:focus {
  outline: none;
  border-color: var(--primary);
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--primary) 14%, transparent);
}
.f-input::placeholder {
  color: var(--muted-foreground);
}
.f-eye {
  position: absolute;
  right: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 0;
  background: none;
  color: var(--muted-foreground);
  cursor: pointer;
  border-radius: 6px;
  transition: color 0.15s;
}
.f-eye:hover {
  color: var(--foreground);
}
.f-captcha {
  position: absolute;
  right: 8px;
  height: 32px;
  display: flex;
  align-items: center;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: #f3f4f6;
  overflow: hidden;
  cursor: pointer;
}
.f-captcha :deep(svg) {
  display: block;
  height: 32px;
  width: 80px;
}
.f-captcha-ph {
  padding: 0 12px;
  font-size: 12px;
  color: var(--muted-foreground);
}
.submit {
  height: 46px;
  margin-top: 4px;
  border: 0;
  border-radius: 8px;
  background: var(--primary);
  color: var(--primary-foreground);
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0.08em;
  cursor: pointer;
  transition:
    background 0.2s,
    transform 0.08s;
}
.submit:hover:not(:disabled) {
  background: color-mix(in oklch, var(--primary) 88%, black);
}
.submit:active:not(:disabled) {
  transform: scale(0.99);
}
.submit:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* 底部条：白底，链接主色 */
.foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 28px;
  padding: 16px 44px;
  font-size: 12px;
  color: var(--muted-foreground);
  background: #fff;
}
.foot a {
  color: var(--primary);
  font-weight: 500;
  text-decoration: none;
  transition: opacity 0.15s;
}
.foot a:hover {
  opacity: 0.7;
}

/* ── 响应式：窄屏收起左装饰面板 ── */
@media (max-width: 720px) {
  .card {
    grid-template-columns: 1fr;
    max-width: 420px;
  }
  .stage {
    display: none;
  }
}
@media (max-width: 480px) {
  .panel-inner {
    padding: 32px 24px 0;
  }
  .foot {
    padding: 14px 24px;
  }
}
@media (prefers-reduced-motion: reduce) {
  .card {
    animation: none;
  }
}
</style>
