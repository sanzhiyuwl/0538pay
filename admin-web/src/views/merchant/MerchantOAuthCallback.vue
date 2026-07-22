<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Loader2 } from 'lucide-vue-next'
import { oauthCallback, oauthBind } from '@/lib/api/merchantAuth'
import { useMerchantAuthStore } from '@/stores/merchantAuth'
import { useToast } from '@/composables/useToast'
import { ApiError } from '@/lib/api/client'

// 快捷登录回调页 /m/oauth/:provider：用 code 换 openid → 登录或引导绑定。
const route = useRoute()
const router = useRouter()
const auth = useMerchantAuthStore()
const toast = useToast()

const provider = route.params.provider as string
const providerLabel: Record<string, string> = { alipay: '支付宝', wx: '微信', qq: 'QQ' }
const loading = ref(true)
const needBind = ref(false)
const openid = ref('')
const bindForm = ref({ account: '', password: '' })
const binding = ref(false)

onMounted(async () => {
  const code = route.query.code as string
  if (!code) {
    toast.error('未获取到授权 code')
    router.replace('/m/login')
    return
  }
  try {
    const redirect = `${location.origin}/m/oauth/${provider}`
    const res = await oauthCallback(provider, code, redirect)
    if (res.need_bind) {
      needBind.value = true
      openid.value = res.openid || ''
    } else if (res.token && res.info) {
      auth.setSession(res.token, res.info)
      toast.success('登录成功')
      router.replace('/m')
    }
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '快捷登录失败')
    router.replace('/m/login')
  } finally {
    loading.value = false
  }
})

async function doBind() {
  if (binding.value) return
  if (!bindForm.value.account || !bindForm.value.password) {
    toast.error('请输入商户账号和密码')
    return
  }
  binding.value = true
  try {
    const res = await oauthBind({
      provider, openid: openid.value,
      account: bindForm.value.account, password: bindForm.value.password, type: 1,
    })
    if (res.token && res.info) {
      auth.setSession(res.token, res.info)
      toast.success('绑定成功，已登录')
      router.replace('/m')
    }
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '绑定失败')
  } finally {
    binding.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-muted/30 px-4">
    <div class="w-full max-w-sm rounded-lg bg-background p-8 shadow-sm">
      <div v-if="loading" class="flex flex-col items-center gap-3 py-8 text-muted-foreground">
        <Loader2 class="size-6 animate-spin" />
        <span class="text-sm">正在处理{{ providerLabel[provider] || '' }}登录…</span>
      </div>

      <div v-else-if="needBind">
        <h1 class="text-lg font-semibold">绑定商户账号</h1>
        <p class="mt-1 text-sm text-muted-foreground">
          首次使用{{ providerLabel[provider] || '第三方' }}登录，请输入已有商户账号完成绑定。
        </p>
        <div class="mt-5 space-y-3">
          <input v-model="bindForm.account" placeholder="商户账号（邮箱/手机）" class="field-input w-full" />
          <input v-model="bindForm.password" type="password" placeholder="登录密码" class="field-input w-full" @keyup.enter="doBind" />
          <button
            class="w-full bg-primary py-2.5 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-90 disabled:opacity-50"
            :disabled="binding"
            @click="doBind"
          >{{ binding ? '绑定中…' : '绑定并登录' }}</button>
          <button class="w-full py-2 text-sm text-muted-foreground hover:text-foreground" @click="router.replace('/m/login')">
            返回登录
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
