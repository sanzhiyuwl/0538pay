<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  User, Lock, Eye, EyeOff, Hash, KeyRound, ArrowRight, CheckCircle2, TrendingUp,
} from 'lucide-vue-next'
import { useToast } from '@/composables/useToast'
import { useMerchantAuthStore } from '@/stores/merchantAuth'
import { ApiError } from '@/lib/api/client'
import { fetchOAuthURL } from '@/lib/api/merchantAuth'

const router = useRouter()
const route = useRoute()
const toast = useToast()
const merchantAuth = useMerchantAuthStore()

// 登录方式：password 密码登录 / key 密钥登录（商户特有）
const mode = ref<'password' | 'key'>('password')
const form = ref({ user: '', pass: '', mid: '', mkey: '' })
const showPass = ref(false)
const showKey = ref(false)
const loading = ref(false)

const submitLabel = computed(() => (loading.value ? '登录中…' : '登录'))

async function login() {
  // type=1 密码登录 / type=0 密钥登录，与后端 dto.MerchantLoginReq 对齐
  let type: 0 | 1
  let account: string
  let password: string
  if (mode.value === 'password') {
    if (!form.value.user.trim() || !form.value.pass) {
      toast.error('请输入账号和密码')
      return
    }
    type = 1
    account = form.value.user.trim()
    password = form.value.pass
  } else {
    if (!form.value.mid.trim() || !form.value.mkey.trim()) {
      toast.error('请输入商户 ID 和密钥')
      return
    }
    type = 0
    account = form.value.mid.trim()
    password = form.value.mkey.trim()
  }
  loading.value = true
  try {
    await merchantAuth.login(type, account, password)
    toast.success('登录成功')
    // 支持登录后跳回原目标（路由守卫带的 redirect）
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/m'
    router.push(redirect)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '登录失败，请重试')
  } finally {
    loading.value = false
  }
}

const socials = [
  { key: 'alipay', label: '支付宝', short: '支', color: '#1677ff' },
  { key: 'wx', label: '微信', short: '微', color: '#07c160' },
  { key: 'qq', label: 'QQ', short: 'Q', color: '#12b7f5' },
]

// 快捷登录：取第三方授权 URL 后跳转；回调页(/m/oauth/:provider)处理 code 换登录态。
async function startOAuth(provider: string, label: string) {
  try {
    const redirect = `${location.origin}/m/oauth/${provider}`
    const state = Math.random().toString(36).slice(2)
    const { url } = await fetchOAuthURL(provider, redirect, state)
    location.href = url
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : `${label}登录未开启`)
  }
}
</script>

<template>
  <div class="auth">
    <!-- 左：收款场景舞台（品牌蓝，铺到边，无浮卡） -->
    <aside class="stage">
      <div class="stage-head">
        <span class="logo-name">三只鱼<b>PAY</b></span>
        <span class="logo-badge">商户中心</span>
      </div>

      <div class="stage-copy">
        <h2>每一笔收款<br />都稳稳到账</h2>
        <p>聚合多渠道收款 · 交易实时对账 · 资金自动结算</p>
      </div>

      <!-- CSS 合成的「到账/趋势」浮层卡片，作真实产品感 -->
      <div class="mock" aria-hidden="true">
        <div class="mock-card mock-arrive">
          <div class="mc-ico ok"><CheckCircle2 class="size-4" /></div>
          <div class="mc-body">
            <span class="mc-t">收款到账</span>
            <span class="mc-s">微信支付 · 实时</span>
          </div>
          <b class="mc-amt">+¥1,280.00</b>
        </div>

        <div class="mock-card mock-trend">
          <div class="mc-ico up"><TrendingUp class="size-4" /></div>
          <div class="mc-body">
            <span class="mc-t">今日交易额</span>
            <span class="mc-s">较昨日 +12.4%</span>
          </div>
          <b class="mc-amt strong">¥ 1,284,730</b>
          <div class="spark">
            <span style="height: 32%"></span><span style="height: 54%"></span>
            <span style="height: 40%"></span><span style="height: 72%"></span>
            <span style="height: 60%"></span><span style="height: 90%"></span>
            <span style="height: 76%"></span>
          </div>
        </div>
      </div>

      <dl class="stage-stats">
        <div><dt>服务商户</dt><dd>12,800+</dd></div>
        <div><dt>累计交易额</dt><dd>¥38.4 亿</dd></div>
        <div><dt>资金到账</dt><dd><span class="live" />实时</dd></div>
      </dl>
    </aside>

    <!-- 右：纯白表单区（无浮卡，大留白） -->
    <section class="panel">
      <div class="panel-inner">
        <header class="p-head">
          <h1>登录商户账户</h1>
          <p>欢迎回来，继续管理你的收款与结算</p>
        </header>

        <div class="tabs">
          <button class="tab" :class="{ active: mode === 'password' }" @click="mode = 'password'">密码登录</button>
          <button class="tab" :class="{ active: mode === 'key' }" @click="mode = 'key'">密钥登录</button>
        </div>

        <form v-if="mode === 'password'" class="fields" @submit.prevent="login">
          <label class="fl">账号</label>
          <div class="field">
            <User class="f-icon" />
            <input v-model="form.user" class="f-input" placeholder="邮箱 / 手机号" autocomplete="username" />
          </div>
          <label class="fl">密码</label>
          <div class="field">
            <Lock class="f-icon" />
            <input
              v-model="form.pass"
              :type="showPass ? 'text' : 'password'"
              class="f-input"
              placeholder="登录密码"
              autocomplete="current-password"
            />
            <button type="button" class="f-eye" tabindex="-1" @click="showPass = !showPass">
              <Eye v-if="!showPass" class="size-4" /><EyeOff v-else class="size-4" />
            </button>
          </div>
          <button class="submit" type="submit" :disabled="loading">
            {{ submitLabel }}<ArrowRight class="size-4" />
          </button>
        </form>

        <form v-else class="fields" @submit.prevent="login">
          <label class="fl">商户 ID</label>
          <div class="field">
            <Hash class="f-icon" />
            <input v-model="form.mid" class="f-input" placeholder="商户 ID" autocomplete="off" />
          </div>
          <label class="fl">商户密钥</label>
          <div class="field">
            <KeyRound class="f-icon" />
            <input
              v-model="form.mkey"
              :type="showKey ? 'text' : 'password'"
              class="f-input"
              placeholder="商户密钥"
              autocomplete="off"
            />
            <button type="button" class="f-eye" tabindex="-1" @click="showKey = !showKey">
              <Eye v-if="!showKey" class="size-4" /><EyeOff v-else class="size-4" />
            </button>
          </div>
          <p class="hint">商户 ID 与密钥可在「商户中心 · API 接口」页面查看</p>
          <button class="submit" type="submit" :disabled="loading">
            {{ submitLabel }}<ArrowRight class="size-4" />
          </button>
        </form>

        <div class="row">
          <RouterLink to="/m/findpwd" class="link muted">找回密码</RouterLink>
          <RouterLink to="/m/reg" class="link">注册商户 →</RouterLink>
        </div>

        <div class="social">
          <div class="sep"><span>快捷登录</span></div>
          <div class="social-row">
            <button
              v-for="s in socials"
              :key="s.key"
              type="button"
              class="sbtn"
              :title="`${s.label}登录`"
              @click="startOAuth(s.key, s.label)"
            >
              <span class="smark" :style="{ color: s.color }">{{ s.short }}</span>
            </button>
          </div>
        </div>

        <p class="agree">
          登录即表示同意
          <a class="link" href="#" @click.prevent>服务协议</a> 与
          <a class="link" href="#" @click.prevent>隐私政策</a>
        </p>
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

/* ── 左：收款场景舞台 ── */
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
/* 顶/底文字区柔和压暗（标题+数据条可读），中间插画区不压 */
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
/* 极淡网格纹理，隐约可见即可，不抢插画 */
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
  font-size: 38px;
  line-height: 1.22;
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

/* 浮层卡片 */
.mock {
  margin-top: 44px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 380px;
}
.mock-card {
  position: relative;
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 12px;
  padding: 16px 18px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.12);
  border: 1px solid rgba(255, 255, 255, 0.2);
  backdrop-filter: blur(10px);
  box-shadow: 0 18px 40px -22px rgba(3, 16, 50, 0.7);
  animation: float-in 0.6s cubic-bezier(0.23, 1, 0.32, 1) both;
}
.mock-arrive {
  margin-left: 8px;
  animation-delay: 0.1s;
}
.mock-trend {
  margin-left: 40px;
  animation-delay: 0.24s;
}
@keyframes float-in {
  from {
    opacity: 0;
    transform: translateY(14px) scale(0.97);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}
.mc-ico {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border-radius: 10px;
}
.mc-ico.ok {
  color: #34d399;
  background: rgba(52, 211, 153, 0.18);
}
.mc-ico.up {
  color: #7fb0ff;
  background: rgba(127, 176, 255, 0.2);
}
.mc-body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.mc-t {
  font-size: 13px;
  font-weight: 600;
  color: #f2f7ff;
}
.mc-s {
  font-size: 11.5px;
  color: #b8ccf0;
}
.mc-amt {
  font-size: 15px;
  font-weight: 700;
  color: #d9fce9;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.mc-amt.strong {
  color: #fff;
}
.spark {
  grid-column: 1 / -1;
  display: flex;
  align-items: flex-end;
  gap: 5px;
  height: 34px;
  margin-top: 4px;
}
.spark span {
  flex: 1;
  border-radius: 3px 3px 0 0;
  background: linear-gradient(to top, rgba(127, 176, 255, 0.35), #7fb0ff);
}

.stage-stats {
  position: relative;
  margin: auto 0 0;
  padding-top: 28px;
  display: grid;
  grid-template-columns: repeat(3, auto);
  gap: 34px;
}
/* 两端渐隐的柔和分隔，替代生硬实线 */
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
  font-variant-numeric: tabular-nums;
  display: flex;
  align-items: center;
  gap: 6px;
}
.live {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #34d399;
  box-shadow: 0 0 0 0 rgba(52, 211, 153, 0.6);
  animation: pulse 1.8s ease-out infinite;
}
@keyframes pulse {
  to {
    box-shadow: 0 0 0 7px rgba(52, 211, 153, 0);
  }
}

/* ── 右：纯白表单 ── */
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
  margin: 26px 0 22px;
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
.sbtn:focus-visible,
.f-eye:focus-visible {
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
  margin-bottom: 16px;
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
.hint {
  margin: 0 0 16px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--muted-foreground);
}
.submit {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  height: 46px;
  margin-top: 4px;
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
  opacity: 0.7;
  cursor: default;
}
.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 16px;
  font-size: 13px;
}
.link {
  color: var(--primary);
  text-decoration: none;
}
.link:hover {
  text-decoration: underline;
}
.link.muted {
  color: var(--muted-foreground);
}
.link.muted:hover {
  color: var(--primary);
}
.social {
  margin-top: 24px;
}
.sep {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
  color: var(--muted-foreground);
}
.sep::before,
.sep::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--border);
}
.social-row {
  display: flex;
  justify-content: center;
  gap: 14px;
  margin-top: 16px;
}
.sbtn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--background);
  cursor: pointer;
  transition:
    border-color 0.15s,
    background 0.15s,
    transform 0.08s;
}
.sbtn:hover {
  border-color: color-mix(in oklch, var(--primary) 40%, var(--border));
  background: var(--accent);
}
.sbtn:active {
  transform: scale(0.94);
}
.smark {
  font-size: 14px;
  font-weight: 700;
}
.agree {
  margin: 20px 0 0;
  text-align: center;
  font-size: 12px;
  color: var(--muted-foreground);
}

/* 中等宽度：左舞台内边距收一档，避免走势图卡贴边 */
@media (max-width: 1080px) {
  .stage {
    padding: 44px 40px;
  }
  .mock-trend {
    margin-left: 20px;
  }
}

/* ── 响应式：窄屏收起左舞台 ── */
@media (max-width: 900px) {
  .auth {
    grid-template-columns: 1fr;
  }
  .stage {
    display: none;
  }
  .panel {
    align-items: flex-start;
    padding: 56px 24px 40px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .mock-card,
  .live {
    animation: none;
  }
}
</style>
