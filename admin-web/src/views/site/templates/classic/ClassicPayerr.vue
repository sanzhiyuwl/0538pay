<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { XCircle } from 'lucide-vue-next'
import { Button } from '@/components/ui'

const route = useRoute()
const msg = computed(() => (route.query.msg as string) || '支付未完成或已取消，请重新发起支付')
const returnUrl = computed(() => (route.query.return_url as string) || '')

function retry() {
  window.history.back()
}
function goBack() {
  if (returnUrl.value) window.location.href = returnUrl.value
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-content px-4">
    <div class="w-full max-w-md rounded-2xl border border-border bg-background p-8 text-center shadow-sm">
      <div class="mx-auto flex size-16 items-center justify-center rounded-full bg-destructive/10 text-destructive">
        <XCircle class="size-9" />
      </div>
      <h1 class="mt-5 text-2xl font-bold tracking-tight">支付失败</h1>
      <p class="mt-2 text-sm text-muted-foreground">{{ msg }}</p>

      <div class="mt-6 flex justify-center gap-3">
        <Button @click="retry">重新支付</Button>
        <Button v-if="returnUrl" variant="outline" @click="goBack">返回商户</Button>
        <Button v-else variant="outline" @click="$router.push('/')">返回首页</Button>
      </div>

      <p class="mt-6 text-xs text-muted-foreground">
        如已扣款但显示失败，请稍后在商户中心「订单记录」核对，或联系客服。
      </p>
    </div>
  </div>
</template>
