<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { Eye, EyeOff, ArrowRight, Zap, Ban } from 'lucide-vue-next'
import { useToast } from '@/composables/useToast'
import { useSiteStore } from '@/stores/site'
import { splitBrand } from '@/lib/utils'
import { fetchCaptcha, merchantRegister, fetchRegMethods, sendSmsCode, sendEmailCode, type RegMethods } from '@/lib/api/merchantAuth'
import { ApiError } from '@/lib/api/client'

const router = useRouter()
const toast = useToast()

// 品牌名来自后台「网站设置 / 网站信息」，实时联动；末尾一个词高亮
const siteStore = useSiteStore()
onMounted(() => siteStore.hydrate())
const brand = computed(() => splitBrand(siteStore.config.merchantName))

const verifyType = ref<'phone' | 'email'>('phone')

// 注册方式开关 + 是否需真实验证码（后台 reg_phone/reg_email/reg_otp）
const methods = ref<RegMethods>({ open: true, invite_only: false, phone: true, email: true, otp: false })
onMounted(async () => {
  try {
    methods.value = await fetchRegMethods()
    // 默认 tab 落在开放的方式上：手机没开就切邮箱
    if (!methods.value.phone && methods.value.email) verifyType.value = 'email'
    else if (methods.value.phone) verifyType.value = 'phone'
  } catch {
    /* 拉取失败按默认全开处理 */
  }
})
// 后台关闭注册（reg_open=0），或注册方式全关，都不可注册 → 出暂停注册页
const regPaused = computed(() => !methods.value.open)
const regClosed = computed(() => !methods.value.phone && !methods.value.email)

const form = ref({ account: '', code: '', otp: '', pwd: '', pwd2: '', invite: '', agree: false })
const showPwd = ref(false)

// —— 真实验证码：发送 + 60s 倒计时 ——
const otpSending = ref(false)
const otpCountdown = ref(0)
let otpTimer: ReturnType<typeof setInterval> | null = null
function startOtpCountdown() {
  otpCountdown.value = 60
  otpTimer = setInterval(() => {
    otpCountdown.value--
    if (otpCountdown.value <= 0 && otpTimer) {
      clearInterval(otpTimer)
      otpTimer = null
    }
  }, 1000)
}
onUnmounted(() => { if (otpTimer) clearInterval(otpTimer) })

async function sendOtp() {
  const account = form.value.account.trim()
  if (!account) {
    toast.error(verifyType.value === 'phone' ? '请先输入手机号' : '请先输入邮箱')
    return
  }
  otpSending.value = true
  try {
    if (verifyType.value === 'phone') await sendSmsCode(account)
    else await sendEmailCode(account)
    toast.success('验证码已发送，请注意查收')
    startOtpCountdown()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '验证码发送失败')
  } finally {
    otpSending.value = false
  }
}

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

const pwdScore = computed(() => {
  const p = form.value.pwd
  if (!p) return 0
  let s = 0
  if (p.length >= 8) s++
  if (/[A-Z]/.test(p) && /[a-z]/.test(p)) s++
  if (/\d/.test(p) && /[^A-Za-z0-9]/.test(p)) s++
  return Math.min(s, 3)
})
const pwdLabel = computed(() => ['', '弱', '中', '强'][pwdScore.value])

const canSubmit = computed(() => {
  const f = form.value
  if (!f.account || !f.code || !f.pwd || f.pwd !== f.pwd2 || !f.agree) return false
  if (methods.value.otp && !f.otp) return false // 真实验证码开启时必填
  return true
})

const loading = ref(false)
async function submit() {
  if (!canSubmit.value) {
    if (form.value.pwd && form.value.pwd !== form.value.pwd2) toast.error('两次输入的密码不一致')
    else if (!form.value.agree) toast.error('请先勾选同意服务协议、隐私政策')
    return
  }
  loading.value = true
  try {
    const res = await merchantRegister({
      verifytype: verifyType.value === 'phone' ? 1 : 0,
      account: form.value.account.trim(),
      password: form.value.pwd,
      invite: form.value.invite.trim() || undefined,
      captcha_token: captchaToken.value,
      captcha: form.value.code.trim(),
      otp_code: methods.value.otp ? form.value.otp.trim() : undefined,
    })
    // 付费注册（reg_pay=1）：先支付注册费，回调成功后后端建号。跳收银台完成支付。
    if (res.need_pay && res.pay) {
      toast.info(res.msg || '请完成支付以完成注册')
      router.push(`/pay/mock/cashier/${res.pay.trade_no}`)
      return
    }
    toast.success(res.msg || '注册成功，请登录')
    router.push('/m/login')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '注册失败')
    loadCaptcha() // 刷新验证码
  } finally {
    loading.value = false
  }
}

const steps = [
  { n: '1', t: '注册账户', d: '手机 / 邮箱验证，即刻开通' },
  { n: '2', t: '提交进件', d: '完善主体资料，等待审核' },
  { n: '3', t: '开始收款', d: '接入渠道，资金自动结算' },
]
</script>

<template>
  <div class="auth">
    <section class="card">
      <!-- 左：品牌蓝装饰面板 + 三步引导 -->
      <aside class="stage">
        <div class="stage-head">
          <span class="stage-logo"><Zap class="size-5" /></span>
          <span class="stage-name">{{ brand.lead }}<b v-if="brand.accent">{{ brand.accent }}</b></span>
        </div>

        <div class="stage-copy">
          <h2>三步开户<br /><em>即刻收款</em></h2>
        </div>

        <ol class="steps">
          <li v-for="(s, i) in steps" :key="s.n" :style="{ animationDelay: `${0.1 + i * 0.12}s` }">
            <span class="step-n">{{ s.n }}</span>
            <span class="step-body"><b>{{ s.t }}</b><em>{{ s.d }}</em></span>
          </li>
        </ol>
      </aside>

      <!-- 右：注册表单面板 -->
      <div class="panel">
        <!-- 后台关闭注册：整体暂停注册页 -->
        <div v-if="regPaused" class="panel-inner paused">
          <div class="paused-badge"><Ban class="size-7" /></div>
          <h1 class="paused-title">暂停注册</h1>
          <p class="paused-desc">平台当前已关闭商户注册通道，如需开通账户请联系平台客服或稍后再来。</p>
          <RouterLink to="/m/login" class="paused-btn">返回登录<ArrowRight class="size-4" /></RouterLink>
        </div>

        <div v-else class="panel-inner">
          <header class="c-head">
            <h1><b>注册</b>商户账户</h1>
            <p class="c-sub">加入 {{ siteStore.config.merchantName }}，开启你的收款之旅</p>
          </header>

          <div v-if="!regClosed" class="tabs">
            <button v-if="methods.phone" class="tab" :class="{ active: verifyType === 'phone' }" @click="verifyType = 'phone'">手机注册</button>
            <button v-if="methods.email" class="tab" :class="{ active: verifyType === 'email' }" @click="verifyType = 'email'">邮箱注册</button>
          </div>

          <p v-if="regClosed" class="reg-closed">当前已开放注册，但未开启任何注册方式，请联系平台管理员。</p>

          <form v-else class="fields" @submit.prevent="submit">
            <div class="field">
              <input
                v-model="form.account"
                class="f-input"
                :placeholder="verifyType === 'phone' ? '输入手机号' : '输入邮箱'"
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

            <!-- 真实验证码：手机短信 / 邮箱验证码（后台 reg_otp=1 才出现） -->
            <div v-if="methods.otp" class="field">
              <input
                v-model="form.otp"
                class="f-input has-captcha"
                :placeholder="verifyType === 'phone' ? '输入短信验证码' : '输入邮箱验证码'"
                autocomplete="one-time-code"
              />
              <button
                type="button"
                class="f-otp"
                :disabled="otpSending || otpCountdown > 0"
                @click="sendOtp"
              >
                {{ otpCountdown > 0 ? `${otpCountdown}s` : otpSending ? '发送中' : '获取验证码' }}
              </button>
            </div>

            <div class="field">
              <input
                v-model="form.pwd"
                :type="showPwd ? 'text' : 'password'"
                class="f-input"
                placeholder="≥8 位，含大小写与符号更安全"
                autocomplete="new-password"
              />
              <button type="button" class="f-eye" tabindex="-1" @click="showPwd = !showPwd">
                <Eye v-if="!showPwd" class="size-4" /><EyeOff v-else class="size-4" />
              </button>
            </div>
            <div v-if="form.pwd" class="pwd-meter" :data-score="pwdScore">
              <span class="bar" :class="{ on: pwdScore >= 1 }"></span>
              <span class="bar" :class="{ on: pwdScore >= 2 }"></span>
              <span class="bar" :class="{ on: pwdScore >= 3 }"></span>
              <em class="pwd-label">密码强度 {{ pwdLabel }}</em>
            </div>

            <div class="field">
              <input
                v-model="form.pwd2"
                :type="showPwd ? 'text' : 'password'"
                class="f-input"
                placeholder="再次输入登录密码"
                autocomplete="new-password"
              />
            </div>

            <div class="field">
              <input v-model="form.invite" class="f-input" placeholder="邀请码（仅邀请注册时必填）" autocomplete="off" />
            </div>

            <label class="agree">
              <input v-model="form.agree" type="checkbox" class="agree-box" />
              <span>我已阅读并同意 <a href="#" @click.prevent>服务协议</a> 与 <a href="#" @click.prevent>隐私政策</a></span>
            </label>

            <button class="submit" type="submit" :disabled="!canSubmit || loading">
              {{ loading ? '注册中…' : '注册' }}<ArrowRight class="size-4" />
            </button>
          </form>
        </div>

        <!-- 底部条 -->
        <div class="foot">
          <span>已有账户？<RouterLink to="/m/login">立即登录</RouterLink></span>
          <RouterLink to="/m/findpwd">找回密码</RouterLink>
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
  overflow: auto;
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
  max-width: 800px;
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
  margin-top: 30px;
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

/* 三步引导 */
.steps {
  margin: 38px 0 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.steps li {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 13px;
  animation: step-in 0.5s cubic-bezier(0.23, 1, 0.32, 1) both;
}
.steps li:not(:last-child)::before {
  content: '';
  position: absolute;
  left: 14px;
  top: 30px;
  bottom: -22px;
  width: 2px;
  transform: translateX(-50%);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.4) 0%, rgba(255, 255, 255, 0.1) 100%);
}
@keyframes step-in {
  from {
    opacity: 0;
    transform: translateX(-10px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}
.step-n {
  position: relative;
  z-index: 1;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  font-size: 13px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: #fff;
  background: rgba(255, 255, 255, 0.16);
  border: 1.5px solid rgba(255, 255, 255, 0.55);
}
.step-body {
  display: flex;
  flex-direction: column;
  gap: 3px;
  line-height: 1.4;
}
.step-body b {
  font-size: 14px;
  font-weight: 600;
  color: #fff;
}
.step-body em {
  font-style: normal;
  font-size: 12px;
  color: #cfe0ff;
}

/* ── 右：表单面板 ── */
.panel {
  position: relative;
  display: flex;
  flex-direction: column;
}
.panel-inner {
  flex: 1;
  padding: 36px 44px 0;
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

.tabs {
  display: flex;
  gap: 6px;
  border-bottom: 1px solid var(--border);
  margin: 22px 0 4px;
}
.tab {
  position: relative;
  padding: 8px 14px 12px;
  font-size: 15px;
  color: var(--muted-foreground);
  background: none;
  border: 0;
  cursor: pointer;
  transition: color 0.15s;
}
.tab.active {
  color: var(--primary);
  font-weight: 600;
}
.tab.active::after {
  content: '';
  position: absolute;
  left: 14px;
  right: 14px;
  bottom: -1px;
  height: 2px;
  border-radius: 2px;
  background: var(--primary);
}

.fields {
  display: flex;
  flex-direction: column;
  gap: 13px;
  margin-top: 18px;
}
.field {
  position: relative;
  display: flex;
  align-items: center;
}
.f-input {
  width: 100%;
  height: 44px;
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

/* 发送短信/邮箱验证码按钮 */
.f-otp {
  position: absolute;
  right: 8px;
  height: 32px;
  min-width: 92px;
  padding: 0 12px;
  font-size: 13px;
  font-weight: 500;
  color: var(--primary);
  background: color-mix(in oklch, var(--primary) 8%, #fff);
  border: 1px solid color-mix(in oklch, var(--primary) 24%, transparent);
  border-radius: 8px;
  cursor: pointer;
  transition:
    background 0.15s,
    opacity 0.15s;
}
.f-otp:hover:not(:disabled) {
  background: color-mix(in oklch, var(--primary) 14%, #fff);
}
.f-otp:disabled {
  color: var(--muted-foreground);
  background: #f3f4f6;
  border-color: var(--border);
  cursor: not-allowed;
}

/* 关闭注册提示 */
.reg-closed {
  margin: 26px 0 8px;
  padding: 16px;
  font-size: 13.5px;
  text-align: center;
  color: var(--muted-foreground);
  background: var(--muted);
  border-radius: 8px;
}

/* 暂停注册整页态 */
.panel-inner.paused {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 56px 44px;
  min-height: 420px;
}
.paused-badge {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 68px;
  height: 68px;
  border-radius: 50%;
  color: var(--muted-foreground);
  background: var(--muted);
}
.paused-title {
  margin: 22px 0 0;
  font-size: 22px;
  font-weight: 700;
  letter-spacing: 0.02em;
  color: var(--foreground);
}
.paused-desc {
  margin: 12px 0 0;
  max-width: 20rem;
  font-size: 13.5px;
  line-height: 1.7;
  color: var(--muted-foreground);
}
.paused-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 28px;
  height: 44px;
  padding: 0 24px;
  border-radius: 8px;
  background: var(--primary);
  color: var(--primary-foreground);
  font-size: 14px;
  font-weight: 600;
  text-decoration: none;
  transition:
    background 0.2s,
    transform 0.08s;
}
.paused-btn:hover {
  background: color-mix(in oklch, var(--primary) 88%, black);
}
.paused-btn:active {
  transform: scale(0.99);
}

/* 密码强度条 */
.pwd-meter {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: -4px 0 0;
}
.pwd-meter .bar {
  flex: 1;
  height: 4px;
  border-radius: 2px;
  background: var(--border);
  transition: background 0.2s;
}
.pwd-meter[data-score='1'] .bar.on {
  background: #ef4444;
}
.pwd-meter[data-score='2'] .bar.on {
  background: #f59e0b;
}
.pwd-meter[data-score='3'] .bar.on {
  background: #10b981;
}
.pwd-label {
  flex: none;
  font-style: normal;
  font-size: 11.5px;
  color: var(--muted-foreground);
}

/* 同意勾选 */
.agree {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-top: 2px;
  font-size: 12.5px;
  line-height: 1.5;
  color: var(--muted-foreground);
  cursor: pointer;
}
.agree-box {
  margin-top: 2px;
  width: 14px;
  height: 14px;
  flex: none;
  accent-color: var(--primary);
  cursor: pointer;
}
.agree a {
  color: var(--primary);
  text-decoration: none;
}
.agree a:hover {
  text-decoration: underline;
}

.submit {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  height: 46px;
  margin-top: 8px;
  border: 0;
  border-radius: 8px;
  background: var(--primary);
  color: var(--primary-foreground);
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0.02em;
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
  opacity: 0.55;
  cursor: not-allowed;
}

/* 底部条：白底，链接主色 */
.foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 24px;
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
  .card,
  .steps li {
    animation: none;
  }
}
</style>
