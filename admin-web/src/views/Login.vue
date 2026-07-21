<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { User, Lock, Eye, EyeOff, Smartphone, ShieldCheck } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'
import { ApiError } from '@/lib/api/client'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const toast = useToast()

// 登录方式：account 账号登录 / sms 短信登录（短信为后续接入占位）
const mode = ref<'account' | 'sms'>('account')

const form = ref({ user: '', pass: '', phone: '', code: '' })
const showPass = ref(false)
const loading = ref(false)

// 短信验证码倒计时
const countdown = ref(0)
let timer: number | undefined
function sendCode() {
  if (!form.value.phone.trim()) {
    toast.error('请先输入手机号')
    return
  }
  if (countdown.value > 0) return
  countdown.value = 60
  toast.success('验证码已发送')
  timer = window.setInterval(() => {
    countdown.value--
    if (countdown.value <= 0) window.clearInterval(timer)
  }, 1000)
}

const submitLabel = computed(() => (loading.value ? '登录中…' : '登 录'))

async function login() {
  if (mode.value === 'sms') {
    toast.info('短信登录待后端接入')
    return
  }
  if (!form.value.user.trim() || !form.value.pass) {
    toast.error('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    await auth.login(form.value.user.trim(), form.value.pass)
    toast.success('登录成功')
    const redirect = (route.query.redirect as string) || '/admin'
    router.replace(redirect)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '登录失败，请重试')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <!-- 整页背景由 login-bg-tech.jpg + 深色遮罩承担（见 .login-page 样式） -->

    <!-- 居中悬浮双栏卡片 -->
    <div class="login-card">
      <!-- 左：品牌墙 -->
      <aside class="brand-wall">
        <!-- 品牌墙装饰纹理与盾牌由 login-bg-tech.jpg 承担（见 .brand-wall 背景） -->
        <div class="brand-wall-inner">
          <div class="brand-copy">
            <h2 class="brand-slogan">三只鱼<span class="brand-pay">PAY</span></h2>
            <span class="brand-rule" aria-hidden="true"></span>
            <p class="brand-sub">一站式支付进件 · 聚合清算 · 智能风控服务</p>
          </div>

          <ul class="brand-tags">
            <li><ShieldCheck class="tag-icon" /> 持牌机构清算</li>
            <li><ShieldCheck class="tag-icon" /> 资金不经手</li>
            <li><ShieldCheck class="tag-icon" /> 全程加密对账</li>
          </ul>
        </div>
      </aside>

      <!-- 右：登录表单 -->
      <section class="form-side">
        <!-- Tab 切换 -->
        <div class="tabs">
          <button
            class="tab"
            :class="{ active: mode === 'account' }"
            @click="mode = 'account'"
          >
            账号登录
          </button>
          <button class="tab" :class="{ active: mode === 'sms' }" @click="mode = 'sms'">
            短信登录
          </button>
        </div>

        <!-- 账号登录 -->
        <form v-if="mode === 'account'" class="fields" @submit.prevent="login">
          <div class="field">
            <User class="f-icon" />
            <input
              v-model="form.user"
              class="f-input"
              placeholder="管理员账号"
              autocomplete="username"
            />
          </div>
          <div class="field">
            <Lock class="f-icon" />
            <input
              v-model="form.pass"
              :type="showPass ? 'text' : 'password'"
              class="f-input"
              placeholder="登录密码"
              autocomplete="current-password"
            />
            <button type="button" class="f-eye" @click="showPass = !showPass" tabindex="-1">
              <Eye v-if="!showPass" class="size-4" />
              <EyeOff v-else class="size-4" />
            </button>
          </div>
          <button class="submit" type="submit" :disabled="loading">{{ submitLabel }}</button>
        </form>

        <!-- 短信登录 -->
        <form v-else class="fields" @submit.prevent="login">
          <div class="field">
            <Smartphone class="f-icon" />
            <input
              v-model="form.phone"
              class="f-input"
              placeholder="手机号"
              autocomplete="tel"
            />
          </div>
          <div class="field">
            <Lock class="f-icon" />
            <input v-model="form.code" class="f-input" placeholder="短信验证码" />
            <button type="button" class="f-code" :disabled="countdown > 0" @click="sendCode">
              {{ countdown > 0 ? `${countdown}s` : '获取验证码' }}
            </button>
          </div>
          <button class="submit" type="submit" :disabled="loading">{{ submitLabel }}</button>
        </form>

        <div class="form-foot">
          <span class="tip">仅限授权运营人员访问</span>
          <a class="link" href="#" @click.prevent="toast.info('请联系系统管理员重置密码')">忘记密码</a>
        </div>
      </section>
    </div>

    <p class="page-sign">0538Pay 运营管理系统 · 技术支持 0538Pay Team</p>
  </div>
</template>

<style scoped>
.login-page {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  overflow: hidden;
  /* 整页背景图；图缺失时回退深蓝底 */
  background-color: #050e20;
  background-image: url('/assets/login-page-bg.jpg');
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
}

/* 悬浮双栏卡片 */
.login-card {
  position: relative;
  z-index: 1;
  display: grid;
  grid-template-columns: 1fr 1fr;
  width: 100%;
  max-width: 860px;
  min-height: 460px;
  border-radius: 18px;
  overflow: hidden;
  /* 卡片本身不铺白底：左右两栏各自铺色，避免圆角处透出白底形成白线 */
  background: transparent;
  box-shadow: 0 30px 80px -30px rgba(3, 12, 30, 0.65);
  animation: card-in 0.5s cubic-bezier(0.23, 1, 0.32, 1);
}
@keyframes card-in {
  from {
    opacity: 0;
    transform: translateY(12px) scale(0.98);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

/* 左品牌墙：背景图 + 轻遮罩保证文字可读；图缺失时回退品牌蓝渐变 */
.brand-wall {
  position: relative;
  color: #eaf1ff;
  background-color: #0a2456;
  background-image:
    linear-gradient(155deg, rgba(9, 26, 60, 0.15) 0%, rgba(6, 18, 44, 0.32) 100%),
    url('/assets/login-bg-tech.jpg');
  background-size: cover, cover;
  background-position: center, center;
  background-repeat: no-repeat, no-repeat;
  overflow: hidden;
}
/* 品牌墙母题插画层：SVG，铺满品牌墙，压在文案之下（P-4 视觉侧不简陋） */
.brand-wall-inner {
  position: relative;
  z-index: 1;
  height: 100%;
  padding: 56px 36px;
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  gap: 26px;
}
.brand-copy {
  margin: 0;
}
.brand-slogan {
  font-size: 36px;
  line-height: 1.2;
  font-weight: 800;
  letter-spacing: -0.01em;
  margin: 0 0 18px;
  /* 白→亮蓝渐变，呼应流星背景；深底上清晰，配轻投影增强可读 */
  background: linear-gradient(120deg, #ffffff 0%, #d6e6ff 42%, #7fb0ff 100%);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
  color: transparent;
  filter: drop-shadow(0 2px 10px rgba(4, 16, 40, 0.55));
}
.brand-pay {
  margin-left: 8px;
  letter-spacing: 0.04em;
}
/* 标题下渐变装饰横杠 */
.brand-rule {
  display: block;
  width: 56px;
  height: 3px;
  border-radius: 3px;
  margin: 0 0 18px;
  background: linear-gradient(90deg, #7fb0ff 0%, #3f7bff 55%, rgba(63, 123, 255, 0) 100%);
}
.brand-sub {
  font-size: 13px;
  line-height: 1.7;
  color: #cbdbf5;
  margin: 0;
  white-space: nowrap;
}
/* 底部信任标签条：填充下方留白 + 强化合规定位 */
.brand-tags {
  margin-top: auto;
  padding: 0;
  list-style: none;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.brand-tags li {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11.5px;
  color: #dbe7fb;
  padding: 5px 11px 5px 9px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.16);
  backdrop-filter: blur(4px);
}
.tag-icon {
  width: 13px;
  height: 13px;
  color: #8fbaff;
  flex: none;
}
/* 右表单 */
.form-side {
  padding: 44px 40px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  background: var(--background);
}
.tabs {
  display: flex;
  gap: 6px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 26px;
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
  gap: 16px;
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
  padding: 0 12px 0 38px;
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
.submit {
  height: 44px;
  margin-top: 4px;
  border: 0;
  border-radius: 10px;
  background: var(--primary);
  color: var(--primary-foreground);
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0.05em;
  cursor: pointer;
  transition:
    background 0.2s,
    transform 0.08s;
}
.submit:hover {
  background: color-mix(in oklch, var(--primary) 88%, black);
}
.submit:active {
  transform: scale(0.985);
}
.submit:disabled {
  opacity: 0.7;
  cursor: default;
}
.form-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 18px;
  font-size: 12.5px;
}
.tip {
  color: var(--muted-foreground);
}
.link {
  color: var(--primary);
  text-decoration: none;
}
.link:hover {
  text-decoration: underline;
}

.page-sign {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 1;
  margin: 0;
  padding: 18px 24px 14px;
  text-align: center;
  font-size: 12px;
  letter-spacing: 0.02em;
  color: rgba(60, 78, 110, 0.75);
  /* 顺浅色背景做柔和渐隐，无硬边框，文字融入不突兀 */
  background: linear-gradient(to top, rgba(232, 238, 248, 0.9) 0%, rgba(232, 238, 248, 0) 100%);
}

/* 响应式：窄屏收起品牌墙，表单单列 */
@media (max-width: 720px) {
  .login-card {
    grid-template-columns: 1fr;
    max-width: 400px;
    min-height: 0;
  }
  /* 窄屏品牌墙收成顶部一条图片横幅 */
  .brand-wall {
    min-height: 120px;
  }
  .brand-copy,
  .brand-tags {
    display: none;
  }
  .form-side {
    padding: 28px 28px 32px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .login-card {
    animation: none;
  }
}
</style>
