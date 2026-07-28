<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Loader2, AlertCircle, Store, ShieldCheck, CheckCircle2 } from 'lucide-vue-next'
import {
  fetchEnrollInfo,
  fetchEnrollCaptcha,
  submitEnroll,
  type EnrollPublicInfo,
} from '@/lib/api/enrollPublic'

// 客户自助进件公开页（/enroll/:code）：免登录，靠邀请 code 定位归属代理。
// 落地校验链接可用 → 填基础信息 + 图形验证码 → source=3 建单 → 收银台付开户费。
// PC/H5 同一套响应式布局（max-w 卡片居中，窄屏自适应）。
const route = useRoute()
const router = useRouter()
const code = (route.params.code as string) || ''

const info = ref<EnrollPublicInfo | null>(null)
const loading = ref(true)
const errMsg = ref('')
const submitting = ref(false)
const done = ref(false) // 免开户费直放行时展示成功态

// 支付方式：一期给常见几项，收开户费的渠道 plugin。
const payMethods = [
  { key: 'wxpay', label: '微信支付' },
  { key: 'alipay', label: '支付宝' },
  { key: 'mock', label: '模拟支付（测试）' },
]

const form = reactive({
  merchantName: '',
  contactPhone: '',
  plugin: 'wxpay',
  code: '',
})

const captchaToken = ref('')
const captchaSvg = ref('')

async function loadCaptcha() {
  try {
    const res = await fetchEnrollCaptcha()
    captchaToken.value = res.token
    captchaSvg.value = res.svg
  } catch {
    captchaSvg.value = ''
  }
}

onMounted(async () => {
  if (!code) {
    errMsg.value = '邀请链接无效'
    loading.value = false
    return
  }
  try {
    info.value = await fetchEnrollInfo(code)
    await loadCaptcha()
  } catch (e) {
    errMsg.value = e instanceof Error ? e.message : '邀请链接无效或已失效'
  } finally {
    loading.value = false
  }
})

async function submit() {
  errMsg.value = ''
  if (!form.merchantName.trim()) {
    errMsg.value = '请填写商户名称'
    return
  }
  if (!/^1\d{10}$/.test(form.contactPhone.trim())) {
    errMsg.value = '请填写正确的联系手机号'
    return
  }
  if (!form.code.trim()) {
    errMsg.value = '请输入图形验证码'
    return
  }
  submitting.value = true
  try {
    const res = await submitEnroll(code, {
      merchant_name: form.merchantName.trim(),
      contact_phone: form.contactPhone.trim(),
      plugin: form.plugin,
      captcha_token: captchaToken.value,
      captcha_code: form.code.trim(),
    })
    // 有收银台信息 → 跳收银台付开户费；无（免开户费）→ 展示成功态。
    if (res.pay && res.pay.trade_no) {
      router.push({ path: `/pay/mock/cashier/${res.pay.trade_no}` })
    } else {
      done.value = true
    }
  } catch (e) {
    errMsg.value = e instanceof Error ? e.message : '提交失败，请稍后重试'
    await loadCaptcha() // 提交失败刷新验证码（一次性已消耗）
    form.code = ''
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-muted/30 px-4 py-10">
    <div class="w-full max-w-md">
      <!-- 加载 -->
      <div v-if="loading" class="flex flex-col items-center gap-3 py-20 text-muted-foreground">
        <Loader2 class="size-6 animate-spin" />
        <span class="text-sm">加载中…</span>
      </div>

      <!-- 链接失效 -->
      <div v-else-if="!info" class="flex flex-col items-center gap-3 rounded-lg bg-background p-8 text-center">
        <AlertCircle class="size-8 text-destructive" />
        <div class="text-sm text-muted-foreground">{{ errMsg || '邀请链接无效' }}</div>
      </div>

      <!-- 提交成功（免开户费直放行） -->
      <div v-else-if="done" class="flex flex-col items-center gap-3 rounded-lg bg-background p-8 text-center">
        <CheckCircle2 class="size-10 text-primary" />
        <div class="text-base font-medium">提交成功</div>
        <div class="text-sm text-muted-foreground">
          您的进件申请已受理，工作人员将尽快为您办理，请保持手机 {{ form.contactPhone }} 畅通。
        </div>
      </div>

      <!-- 进件表单 -->
      <div v-else class="overflow-hidden rounded-lg bg-background shadow-sm">
        <!-- 服务方 -->
        <div class="flex flex-col items-center gap-2 bg-primary/[0.06] px-6 py-6">
          <div class="flex size-12 items-center justify-center rounded-full bg-primary/10">
            <Store class="size-6 text-primary" />
          </div>
          <div class="text-base font-medium">特约商户进件</div>
          <div class="text-xs text-muted-foreground">由 {{ info.agent_name }} 为您提供进件服务</div>
        </div>

        <div class="space-y-4 px-6 py-6">
          <div>
            <label class="mb-1.5 block text-sm text-muted-foreground">商户名称</label>
            <input
              v-model="form.merchantName"
              placeholder="请输入营业执照上的商户全称"
              class="w-full border-b-2 border-border bg-transparent py-2 text-sm outline-none focus:border-primary"
            />
          </div>

          <div>
            <label class="mb-1.5 block text-sm text-muted-foreground">联系手机</label>
            <input
              v-model="form.contactPhone"
              type="tel"
              maxlength="11"
              placeholder="用于接收进件进度通知"
              class="w-full border-b-2 border-border bg-transparent py-2 text-sm outline-none focus:border-primary tabular-nums"
            />
          </div>

          <div>
            <label class="mb-1.5 block text-sm text-muted-foreground">支付方式（开户费）</label>
            <div class="grid grid-cols-3 gap-2">
              <button
                v-for="m in payMethods"
                :key="m.key"
                type="button"
                class="border px-2 py-2 text-xs transition-colors"
                :class="form.plugin === m.key ? 'border-primary text-primary ring-1 ring-primary' : 'border-border hover:border-primary/50'"
                @click="form.plugin = m.key"
              >
                {{ m.label }}
              </button>
            </div>
          </div>

          <div>
            <label class="mb-1.5 block text-sm text-muted-foreground">图形验证码</label>
            <div class="flex items-stretch gap-2">
              <input
                v-model="form.code"
                autocomplete="off"
                placeholder="输入右侧验证码"
                class="flex-1 border-b-2 border-border bg-transparent py-2 text-sm outline-none focus:border-primary"
              />
              <button
                type="button"
                class="flex h-10 w-24 items-center justify-center overflow-hidden rounded border border-border bg-muted/40"
                title="点击刷新"
                @click="loadCaptcha"
              >
                <span v-if="captchaSvg" class="cap" v-html="captchaSvg"></span>
                <span v-else class="text-xs text-muted-foreground">加载中</span>
              </button>
            </div>
          </div>

          <p v-if="errMsg" class="text-sm text-destructive">{{ errMsg }}</p>

          <button
            class="w-full bg-primary py-3 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-90 disabled:opacity-50"
            :disabled="submitting"
            @click="submit"
          >
            <span v-if="submitting">提交中…</span>
            <span v-else>提交并支付开户费</span>
          </button>

          <div class="flex items-center justify-center gap-1.5 pt-1 text-xs text-muted-foreground">
            <ShieldCheck class="size-3.5" />
            <span>信息仅用于特约商户进件办理，全程加密传输</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.cap :deep(svg) {
  width: 100%;
  height: 100%;
}
</style>
