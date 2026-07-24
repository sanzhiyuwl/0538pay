<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Eye, EyeOff, KeyRound, MonitorSmartphone, CheckCircle2, Zap } from 'lucide-vue-next'
import { useToast } from '@/composables/useToast'
import { useMerchantAuthStore } from '@/stores/merchantAuth'
import { useSiteStore } from '@/stores/site'
import AlipayIcon from '@/components/site/icons/AlipayIcon.vue'
import WechatIcon from '@/components/site/icons/WechatIcon.vue'
import QQIcon from '@/components/site/icons/QQIcon.vue'
import { ApiError } from '@/lib/api/client'
import { fetchOAuthURL } from '@/lib/api/merchantAuth'

const router = useRouter()
const route = useRoute()
const toast = useToast()
const merchantAuth = useMerchantAuthStore()

// 品牌名来自后台「网站设置 / 网站信息」，实时联动；末尾 Pay/PAY 拆出高亮
const siteStore = useSiteStore()
onMounted(() => siteStore.hydrate())
const brandName = computed(() => siteStore.config.merchantName || '三只鱼PAY')
const brand = computed(() => {
  const m = brandName.value.match(/^(.*?)(pay)$/i)
  return m ? { lead: m[1], accent: m[2] } : { lead: brandName.value, accent: '' }
})

// 登录方式：password 密码登录 / key 密钥登录（商户特有），由右上角折角切换
const mode = ref<'password' | 'key'>('password')
function toggleMode() {
  mode.value = mode.value === 'password' ? 'key' : 'password'
}
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
  { key: 'alipay', label: '支付宝', icon: AlipayIcon, color: '#1677ff' },
  { key: 'wx', label: '微信', icon: WechatIcon, color: '#07c160' },
  { key: 'qq', label: 'QQ', icon: QQIcon, color: '#12b7f5' },
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

const highlights = [
  '多渠道聚合，一次对接全支持',
  '交易实时到账，秒级入账',
  '资金持牌清算，安全不经手',
  '自动 T+1 结算，省心对账',
  '开放 API，最快 1 天上线',
  '多端管理，PC / 移动随时看',
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
          <h2>让每一笔收款<br /><em>稳稳到账</em></h2>
        </div>

        <ul class="stage-list">
          <li v-for="h in highlights" :key="h"><CheckCircle2 class="size-4" />{{ h }}</li>
        </ul>
      </aside>

      <!-- 右：表单面板 -->
      <div class="panel">
        <!-- 右上角折角：切换 密码登录 ⇄ 密钥登录 -->
        <button class="corner" type="button" :title="mode === 'password' ? '切换密钥登录' : '切换密码登录'" @click="toggleMode">
          <span class="corner-pill">{{ mode === 'password' ? '密钥登录在这里' : '密码登录在这里' }}</span>
          <span class="corner-fold"><component :is="mode === 'password' ? KeyRound : MonitorSmartphone" class="size-5" /></span>
        </button>

        <div class="panel-inner">
          <header class="c-head">
            <h1 v-if="mode === 'password'"><b>账号密码</b>登录</h1>
            <h1 v-else><b>商户密钥</b>登录</h1>
            <p class="c-sub" v-if="mode === 'password'">欢迎回来，继续管理你的收款与结算</p>
            <p class="c-sub" v-else>用商户 ID 与密钥登录，可在「商户中心 · API 接口」查看</p>
          </header>

          <form v-if="mode === 'password'" class="fields" @submit.prevent="login">
            <div class="field">
              <input v-model="form.user" class="f-input" placeholder="输入邮箱 / 手机号" autocomplete="username" />
            </div>
            <div class="field">
              <input
                v-model="form.pass"
                :type="showPass ? 'text' : 'password'"
                class="f-input"
                placeholder="输入密码"
                autocomplete="current-password"
              />
              <button type="button" class="f-eye" tabindex="-1" @click="showPass = !showPass">
                <Eye v-if="!showPass" class="size-4" /><EyeOff v-else class="size-4" />
              </button>
            </div>
            <button class="submit" type="submit" :disabled="loading">{{ submitLabel }}</button>
            <div class="links">
              <RouterLink to="/m/reg" class="link">注册商户</RouterLink>
              <RouterLink to="/m/findpwd" class="link muted">忘记密码</RouterLink>
            </div>
          </form>

          <form v-else class="fields" @submit.prevent="login">
            <div class="field">
              <input v-model="form.mid" class="f-input" placeholder="输入商户 ID" autocomplete="off" />
            </div>
            <div class="field">
              <input
                v-model="form.mkey"
                :type="showKey ? 'text' : 'password'"
                class="f-input"
                placeholder="输入商户密钥"
                autocomplete="off"
              />
              <button type="button" class="f-eye" tabindex="-1" @click="showKey = !showKey">
                <Eye v-if="!showKey" class="size-4" /><EyeOff v-else class="size-4" />
              </button>
            </div>
            <button class="submit" type="submit" :disabled="loading">{{ submitLabel }}</button>
            <div class="links">
              <RouterLink to="/m/reg" class="link">注册商户</RouterLink>
              <RouterLink to="/m/findpwd" class="link muted">忘记密码</RouterLink>
            </div>
          </form>

          <!-- 其他登录方式 -->
          <div class="other">
            <div class="sep"><span>其他登录方式</span></div>
            <div class="social-row">
              <button
                v-for="s in socials"
                :key="s.key"
                type="button"
                class="sbtn"
                :title="`${s.label}登录`"
                :style="{ '--brand': s.color }"
                @click="startOAuth(s.key, s.label)"
              >
                <component :is="s.icon" class="smark" />
              </button>
            </div>
          </div>
        </div>

        <!-- 底部协议条 -->
        <div class="foot">
          <span>登录即同意 <a href="#" @click.prevent>服务协议</a>、<a href="#" @click.prevent>隐私政策</a></span>
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
/* 淡色圆点装饰纹理 */
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
/* 手绘感黄色下划线 */
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

/* 右上角折角切换 */
.corner {
  position: absolute;
  top: 0;
  right: 0;
  z-index: 2;
  display: flex;
  align-items: flex-start;
  border: 0;
  background: none;
  cursor: pointer;
  padding: 0;
}
.corner-pill {
  margin-right: 10px;
  margin-top: 18px;
  padding: 5px 13px;
  font-size: 12.5px;
  font-weight: 500;
  color: var(--primary);
  background: color-mix(in oklch, var(--primary) 8%, #fff);
  border: 1px solid color-mix(in oklch, var(--primary) 24%, transparent);
  border-radius: 999px;
  box-shadow: 0 4px 12px -6px color-mix(in oklch, var(--primary) 50%, transparent);
  transition: background 0.15s, transform 0.15s;
  white-space: nowrap;
}
.corner:hover .corner-pill {
  background: color-mix(in oklch, var(--primary) 15%, #fff);
  transform: translateY(-1px);
}
.corner-fold {
  position: relative;
  display: flex;
  align-items: flex-start;
  justify-content: flex-end;
  width: 78px;
  height: 78px;
  padding: 9px 9px 0 0;
  color: var(--primary-foreground);
  background: linear-gradient(225deg, color-mix(in oklch, var(--primary) 82%, black) 0%, var(--primary) 60%);
  clip-path: polygon(100% 0, 0 0, 100% 100%);
  transition: filter 0.15s;
}
.corner:hover .corner-fold {
  filter: brightness(1.06);
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
  opacity: 0.7;
  cursor: default;
}
.links {
  display: flex;
  align-items: center;
  justify-content: space-between;
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

/* 其他登录方式 */
.other {
  margin-top: 28px;
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
  border-top: 1px dashed var(--border);
}
.social-row {
  display: flex;
  justify-content: center;
  gap: 16px;
  margin-top: 18px;
}
.sbtn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: 0;
  background: none;
  padding: 0;
  cursor: pointer;
  /* 默认灰色图标，hover 显品牌本色 */
  color: color-mix(in oklch, var(--muted-foreground) 72%, transparent);
  transition:
    color 0.15s,
    transform 0.12s;
}
.sbtn:hover {
  color: var(--brand);
  transform: translateY(-2px);
}
.sbtn:active {
  transform: scale(0.92);
}
.smark {
  width: 24px;
  height: 24px;
}

/* 底部协议条：文字统一灰色，无分隔线 */
.foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 28px;
  padding: 14px 44px;
  font-size: 12px;
  color: var(--muted-foreground);
  background: var(--muted);
}
.foot a {
  color: var(--muted-foreground);
  text-decoration: none;
  transition: color 0.15s;
}
.foot a:hover {
  color: var(--foreground);
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
