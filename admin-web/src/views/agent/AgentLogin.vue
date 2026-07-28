<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Eye, EyeOff, Handshake, CheckCircle2 } from 'lucide-vue-next'
import { useToast } from '@/composables/useToast'
import { useAgentAuthStore } from '@/stores/agentAuth'
import { useSiteStore } from '@/stores/site'
import { ApiError } from '@/lib/api/client'
import { splitBrand } from '@/lib/utils'

const router = useRouter()
const route = useRoute()
const toast = useToast()
const agentAuth = useAgentAuthStore()

const siteStore = useSiteStore()
onMounted(() => siteStore.hydrate())
const brand = computed(() => splitBrand(siteStore.config.sitename || 'Epvia Neo'))

const form = ref({ account: '', password: '' })
const showPass = ref(false)
const loading = ref(false)
const submitLabel = computed(() => (loading.value ? '登录中…' : '登录'))

async function login() {
  if (!form.value.account.trim() || !form.value.password) {
    toast.error('请输入登录账号和密码')
    return
  }
  loading.value = true
  try {
    await agentAuth.login(form.value.account.trim(), form.value.password)
    toast.success('登录成功')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/agent'
    router.push(redirect)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '登录失败，请重试')
  } finally {
    loading.value = false
  }
}

const highlights = [
  '名下进件一站管理，进度实时可查',
  '名额钱包 / 佣金结算，账目清楚',
  '专属邀请链接，客户自助进件',
  '数据严格隔离，只看自己名下',
]
</script>

<template>
  <div class="auth">
    <section class="card">
      <!-- 左：品牌装饰面板 -->
      <aside class="stage">
        <div class="stage-head">
          <span class="stage-logo"><Handshake class="size-5" /></span>
          <span class="stage-name">{{ brand.lead }}<b v-if="brand.accent">{{ brand.accent }}</b></span>
        </div>
        <div class="stage-copy">
          <h2><span class="slogan-line">代理进件</span><span class="slogan-line accent">工作台</span></h2>
          <p class="stage-sub">招商 · 进件 · 结算，尽在掌握</p>
        </div>
        <ul class="stage-list">
          <li v-for="h in highlights" :key="h"><CheckCircle2 class="size-4" />{{ h }}</li>
        </ul>
      </aside>

      <!-- 右：表单面板 -->
      <div class="panel">
        <div class="panel-inner">
          <header class="c-head">
            <h1><b>代理</b>登录</h1>
            <p class="c-sub">用平台分配的代理账号登录，管理你名下的进件</p>
          </header>
          <form class="fields" @submit.prevent="login">
            <div class="field">
              <input v-model="form.account" class="f-input" placeholder="输入代理登录账号" autocomplete="username" />
            </div>
            <div class="field">
              <input
                v-model="form.password"
                :type="showPass ? 'text' : 'password'"
                class="f-input"
                placeholder="输入登录密码"
                autocomplete="current-password"
              />
              <button type="button" class="f-eye" tabindex="-1" @click="showPass = !showPass">
                <Eye v-if="!showPass" class="size-4" /><EyeOff v-else class="size-4" />
              </button>
            </div>
            <button class="submit" type="submit" :disabled="loading">{{ submitLabel }}</button>
          </form>
          <p class="tip">账号由平台分配，忘记密码请联系平台运营重置。</p>
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
.card {
  position: relative;
  display: grid;
  grid-template-columns: 300px 1fr;
  width: 100%;
  max-width: 760px;
  overflow: hidden;
  border-radius: 16px;
  background: #fff;
  box-shadow: 0 28px 70px -28px rgba(16, 42, 100, 0.32);
  animation: card-in 0.45s cubic-bezier(0.23, 1, 0.32, 1) both;
}
@keyframes card-in {
  from { opacity: 0; transform: translateY(14px) scale(0.99); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}
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
.stage > * { position: relative; z-index: 1; }
.stage-head { display: flex; align-items: center; gap: 9px; }
.stage-logo {
  display: flex; align-items: center; justify-content: center;
  width: 30px; height: 30px; border-radius: 8px;
  color: var(--primary); background: #fff;
  box-shadow: 0 6px 16px -6px rgba(0, 0, 0, 0.4);
}
.stage-name { font-size: 16px; font-weight: 800; letter-spacing: -0.01em; color: #fff; }
.stage-name b { color: #cfe0ff; }
.stage-copy { margin-top: 34px; }
.stage-copy h2 { font-size: 26px; line-height: 1.35; font-weight: 800; margin: 0; color: #fff; }
.stage-sub { margin: 12px 0 0; font-size: 13px; color: #cfe0ff; }
.slogan-line { display: block; }
.slogan-line.accent { position: relative; width: fit-content; white-space: nowrap; }
.slogan-line.accent::after {
  content: '';
  position: absolute;
  left: -2px; right: -2px; bottom: -6px;
  height: 7px; border-radius: 999px;
  background: #ffd43b;
  transform: rotate(-1.2deg);
  opacity: 0.9;
}
.stage-list { margin: 36px 0 0; padding: 0; list-style: none; display: flex; flex-direction: column; gap: 14px; }
.stage-list li { display: flex; align-items: center; gap: 9px; font-size: 13px; color: #dbe8ff; }
.stage-list svg { flex-shrink: 0; color: #a9c6ff; }
.panel { position: relative; display: flex; flex-direction: column; justify-content: center; }
.panel-inner { padding: 44px 44px; }
.c-head h1 { font-size: 22px; font-weight: 700; letter-spacing: 0.02em; color: var(--foreground); margin: 0; }
.c-head h1 b { color: var(--primary); font-weight: 700; }
.c-sub { margin: 9px 0 0; font-size: 12.5px; color: var(--muted-foreground); }
.fields { display: flex; flex-direction: column; gap: 14px; margin-top: 26px; }
.field { position: relative; display: flex; align-items: center; }
.f-input {
  width: 100%; height: 46px; padding: 0 42px 0 16px;
  border: 1px solid var(--border); border-radius: 8px; background: #fff;
  font-size: 14px; color: var(--foreground);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.f-input:focus {
  outline: none; border-color: var(--primary);
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--primary) 14%, transparent);
}
.f-input::placeholder { color: var(--muted-foreground); }
.f-eye {
  position: absolute; right: 10px;
  display: flex; align-items: center; justify-content: center;
  width: 28px; height: 28px; border: 0; background: none;
  color: var(--muted-foreground); cursor: pointer; border-radius: 6px; transition: color 0.15s;
}
.f-eye:hover { color: var(--foreground); }
.submit {
  height: 46px; margin-top: 4px; border: 0; border-radius: 8px;
  background: var(--primary); color: var(--primary-foreground);
  font-size: 15px; font-weight: 600; letter-spacing: 0.08em; cursor: pointer;
  transition: background 0.2s, transform 0.08s;
}
.submit:hover:not(:disabled) { background: color-mix(in oklch, var(--primary) 88%, black); }
.submit:active:not(:disabled) { transform: scale(0.99); }
.submit:disabled { opacity: 0.7; cursor: default; }
.tip { margin: 18px 0 0; font-size: 12px; color: var(--muted-foreground); text-align: center; }
@media (max-width: 720px) {
  .card { grid-template-columns: 1fr; max-width: 420px; }
  .stage { display: none; }
}
@media (max-width: 480px) {
  .panel-inner { padding: 32px 24px; }
}
@media (prefers-reduced-motion: reduce) {
  .card { animation: none; }
}
</style>
