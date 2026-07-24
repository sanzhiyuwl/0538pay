<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import {
  Lock, Eye, EyeOff, Mail, Smartphone, Ticket, MessageSquareCode, ArrowRight,
} from 'lucide-vue-next'
import { useToast } from '@/composables/useToast'
import { useSiteStore } from '@/stores/site'

const router = useRouter()
const toast = useToast()

import { onMounted } from 'vue'
import { fetchCaptcha, merchantRegister } from '@/lib/api/merchantAuth'
import { ApiError } from '@/lib/api/client'

// 品牌名来自后台「网站设置 / 网站信息」，实时联动；末尾 Pay/PAY 拆出高亮
const siteStore = useSiteStore()
onMounted(() => siteStore.hydrate())
const brandName = computed(() => siteStore.config.merchantName || '三只鱼PAY')
const brand = computed(() => {
  const m = brandName.value.match(/^(.*?)(pay)$/i)
  return m ? { lead: m[1], accent: m[2] } : { lead: brandName.value, accent: '' }
})

const verifyType = ref<'phone' | 'email'>('phone')

const form = ref({ account: '', code: '', pwd: '', pwd2: '', invite: '', agree: false })
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
  return true
})

const loading = ref(false)
async function submit() {
  if (!canSubmit.value) {
    if (form.value.pwd && form.value.pwd !== form.value.pwd2) toast.error('两次输入的密码不一致')
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
    <!-- 左：开户引导舞台 -->
    <aside class="stage">
      <div class="stage-head">
        <span class="logo-name">{{ brand.lead }}<b v-if="brand.accent">{{ brand.accent }}</b></span>
        <span class="logo-badge">商户入驻</span>
      </div>

      <div class="stage-copy">
        <h2>三步开户<br />即刻开始收款</h2>
        <p>无需技术门槛 · 全渠道聚合 · 资金安全合规</p>
      </div>

      <ol class="steps" aria-hidden="true">
        <li v-for="(s, i) in steps" :key="s.n" :style="{ animationDelay: `${0.1 + i * 0.12}s` }">
          <span class="step-n">{{ s.n }}</span>
          <span class="step-body"><b>{{ s.t }}</b><em>{{ s.d }}</em></span>
        </li>
      </ol>

      <dl class="stage-stats">
        <div><dt>持牌清算</dt><dd>合规</dd></div>
        <div><dt>资金安全</dt><dd>不经手</dd></div>
        <div><dt>结算周期</dt><dd>T+1</dd></div>
      </dl>
    </aside>

    <!-- 右：纯白注册表单 -->
    <section class="panel">
      <div class="panel-inner">
        <header class="p-head">
          <h1>注册商户账户</h1>
          <p>加入 {{ brandName }}，开启你的收款之旅</p>
        </header>

        <div class="tabs">
          <button class="tab" :class="{ active: verifyType === 'phone' }" @click="verifyType = 'phone'">手机注册</button>
          <button class="tab" :class="{ active: verifyType === 'email' }" @click="verifyType = 'email'">邮箱注册</button>
        </div>

        <form class="fields" @submit.prevent="submit">
          <label class="fl">{{ verifyType === 'phone' ? '手机号码' : '邮箱地址' }}</label>
          <div class="field">
            <component :is="verifyType === 'phone' ? Smartphone : Mail" class="f-icon" />
            <input
              v-model="form.account"
              class="f-input"
              :placeholder="verifyType === 'phone' ? '请输入手机号' : '请输入邮箱'"
              autocomplete="username"
            />
          </div>

          <label class="fl">图形验证码</label>
          <div class="field">
            <MessageSquareCode class="f-icon" />
            <input v-model="form.code" class="f-input has-suffix" placeholder="请输入右侧图形验证码" autocomplete="off" />
            <button type="button" class="f-captcha" title="点击刷新" @click="loadCaptcha">
              <span v-if="captchaSvg" v-html="captchaSvg"></span>
              <span v-else class="f-captcha-ph">加载中</span>
            </button>
          </div>

          <label class="fl">登录密码</label>
          <div class="field">
            <Lock class="f-icon" />
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

          <label class="fl">确认密码</label>
          <div class="field">
            <Lock class="f-icon" />
            <input
              v-model="form.pwd2"
              :type="showPwd ? 'text' : 'password'"
              class="f-input"
              placeholder="再次输入登录密码"
              autocomplete="new-password"
            />
          </div>

          <label class="fl">邀请码</label>
          <div class="field">
            <Ticket class="f-icon" />
            <input v-model="form.invite" class="f-input" placeholder="仅邀请注册时必填，否则可留空" autocomplete="off" />
          </div>

          <label class="agree-row">
            <input v-model="form.agree" type="checkbox" class="agree-box" />
            <span>
              我已阅读并同意
              <a class="link" href="#" @click.prevent>服务协议</a> 与
              <a class="link" href="#" @click.prevent>隐私政策</a>
            </span>
          </label>

          <button class="submit" type="submit" :disabled="!canSubmit || loading">
            {{ loading ? '注册中…' : '注册' }}<ArrowRight class="size-4" />
          </button>
        </form>

        <p class="foot">已有账户？<RouterLink to="/m/login" class="link">立即登录</RouterLink></p>
      </div>
    </section>
  </div>
</template>

<style scoped>
.auth {
  position: fixed;
  inset: 0;
  display: grid;
  grid-template-columns: 1.1fr 1fr;
  background: #fff;
  overflow: hidden;
}

/* ── 左舞台 ── */
.stage {
  position: relative;
  overflow: hidden;
  padding: 52px 56px;
  display: flex;
  flex-direction: column;
  color: #eef4ff;
  /* 淡品牌蓝 tint（统一到品牌色，保清晰）+ 背景图（缺失回退纯蓝底） */
  background-color: #0a3fae;
  background-image:
    linear-gradient(152deg, rgba(16, 74, 190, 0.42) 0%, rgba(10, 52, 150, 0.34) 48%, rgba(7, 34, 104, 0.5) 100%),
    url('/assets/products-bg.jpg');
  background-size: cover, cover;
  background-position: center, center;
  background-repeat: no-repeat, no-repeat;
}
.stage::after {
  content: '';
  position: absolute;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  background:
    linear-gradient(180deg, rgba(5, 26, 82, 0.5) 0%, rgba(5, 26, 82, 0) 24%),
    linear-gradient(0deg, rgba(4, 20, 66, 0.55) 0%, rgba(4, 20, 66, 0) 22%);
}
.stage::before {
  content: '';
  position: absolute;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.025) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.025) 1px, transparent 1px);
  background-size: 46px 46px;
  mask-image: radial-gradient(80% 60% at 28% 16%, #000 20%, transparent 72%);
}
.stage > * {
  position: relative;
  z-index: 1;
}
.stage-head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.logo-name {
  font-size: 16px;
  font-weight: 700;
}
.logo-name b {
  margin-left: 3px;
  color: #bcd4ff;
  font-weight: 800;
}
.logo-badge {
  font-size: 11px;
  padding: 3px 8px;
  border-radius: 999px;
  color: #dbe8ff;
  background: rgba(255, 255, 255, 0.14);
  border: 1px solid rgba(255, 255, 255, 0.22);
}
.stage-copy {
  margin-top: 40px;
}
.stage-copy h2 {
  font-size: 36px;
  line-height: 1.24;
  font-weight: 800;
  letter-spacing: -0.02em;
  margin: 0;
  color: #fff;
}
.stage-copy p {
  margin: 16px 0 0;
  font-size: 14px;
  line-height: 1.6;
  color: #c3d6ff;
  max-width: 22em;
  text-wrap: pretty;
}

.steps {
  margin: 46px 0 0;
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
  gap: 14px;
  animation: step-in 0.5s cubic-bezier(0.23, 1, 0.32, 1) both;
}
/* 竖向连接线：贯穿相邻两步的序号中心，形成流程感 */
.steps li:not(:last-child)::before {
  content: '';
  position: absolute;
  left: 14px;
  top: 30px;
  bottom: -22px;
  width: 2px;
  transform: translateX(-50%);
  background: linear-gradient(
    180deg,
    rgba(143, 186, 255, 0.5) 0%,
    rgba(143, 186, 255, 0.14) 100%
  );
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
  color: #cfe2ff;
  /* 品牌蓝描边 + 极淡蓝底，清爽不脏；深底上有轻投影托起 */
  background: rgba(63, 123, 255, 0.22);
  border: 1.5px solid rgba(143, 186, 255, 0.7);
  box-shadow: 0 2px 8px -2px rgba(4, 20, 66, 0.6);
}
.step-body {
  display: flex;
  flex-direction: column;
  gap: 3px;
  line-height: 1.4;
}
.step-body b {
  font-size: 14.5px;
  font-weight: 600;
  color: #f6f9ff;
}
.step-body em {
  font-style: normal;
  font-size: 12.5px;
  color: #b3c8ef;
}

.stage-stats {
  position: relative;
  margin: auto 0 0;
  padding-top: 28px;
  display: grid;
  grid-template-columns: repeat(3, auto);
  gap: 34px;
}
.stage-stats::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 1px;
  background: linear-gradient(
    90deg,
    rgba(255, 255, 255, 0) 0%,
    rgba(255, 255, 255, 0.22) 32%,
    rgba(255, 255, 255, 0.22) 68%,
    rgba(255, 255, 255, 0) 100%
  );
}
.stage-stats dt {
  font-size: 11.5px;
  color: #aec6f0;
  margin-bottom: 5px;
}
.stage-stats dd {
  margin: 0;
  font-size: 17px;
  font-weight: 700;
  color: #fff;
}

/* ── 右表单 ── */
.panel {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
  overflow-y: auto;
}
.panel-inner {
  width: 100%;
  max-width: 360px;
}
.p-head h1 {
  font-size: 24px;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: var(--foreground);
  margin: 0;
}
.p-head p {
  margin: 8px 0 0;
  font-size: 13.5px;
  color: var(--muted-foreground);
}
.tabs {
  display: flex;
  gap: 6px;
  border-bottom: 1px solid var(--border);
  margin: 24px 0 18px;
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
.tab:focus-visible,
.f-eye:focus-visible,
.f-code:focus-visible {
  outline: none;
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--primary) 24%, transparent);
  border-radius: 8px;
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
}
.fl {
  font-size: 12.5px;
  font-weight: 500;
  color: var(--foreground);
  margin-bottom: 7px;
}
.fl + .field {
  margin-bottom: 14px;
}
.field {
  position: relative;
  display: flex;
  align-items: center;
}
.f-icon {
  position: absolute;
  left: 12px;
  width: 16px;
  height: 16px;
  color: var(--muted-foreground);
  pointer-events: none;
}
.f-input {
  width: 100%;
  height: 44px;
  padding: 0 40px 0 38px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--background);
  font-size: 14px;
  color: var(--foreground);
  transition:
    border-color 0.15s,
    box-shadow 0.15s;
}
.f-input.has-suffix {
  padding-right: 104px;
}
.f-input:focus {
  outline: none;
  border-color: var(--primary);
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--primary) 16%, transparent);
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
.f-code {
  position: absolute;
  right: 8px;
  height: 30px;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--background);
  color: var(--primary);
  font-size: 12.5px;
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.15s;
}
.f-code:disabled {
  color: var(--muted-foreground);
  cursor: default;
}
.f-captcha {
  position: absolute;
  right: 8px;
  height: 30px;
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
  height: 30px;
  width: 75px;
}
.f-captcha-ph {
  padding: 0 12px;
  font-size: 12px;
  color: var(--muted-foreground);
}
.pwd-meter {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0 0 14px;
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
.agree-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-top: 4px;
  font-size: 12.5px;
  line-height: 1.5;
  color: var(--muted-foreground);
}
.agree-box {
  margin-top: 2px;
  width: 14px;
  height: 14px;
  accent-color: var(--primary);
  flex: none;
}
.submit {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  height: 46px;
  margin-top: 16px;
  border: 0;
  border-radius: 10px;
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
  transform: scale(0.985);
}
.submit:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
.foot {
  margin-top: 18px;
  text-align: center;
  font-size: 13px;
  color: var(--muted-foreground);
}
.link {
  color: var(--primary);
  text-decoration: none;
}
.link:hover {
  text-decoration: underline;
}

@media (max-width: 1080px) {
  .stage {
    padding: 44px 40px;
  }
}

@media (max-width: 900px) {
  .auth {
    grid-template-columns: 1fr;
  }
  .stage {
    display: none;
  }
  .panel {
    align-items: flex-start;
    padding: 48px 24px 40px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .steps li {
    animation: none;
  }
}
</style>
