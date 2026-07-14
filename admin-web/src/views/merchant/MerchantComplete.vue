<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { Store } from 'lucide-vue-next'
import { Button, Select } from '@/components/ui'

const router = useRouter()

// 完善信息（对齐 epay user/completeinfo.php），强制补全后进工作台
const stypeOptions = [
  { value: '1', label: '支付宝结算' },
  { value: '2', label: '微信结算' },
  { value: '3', label: 'QQ钱包结算' },
  { value: '4', label: '银行卡结算' },
]
const form = ref({ stype: '1', account: '', username: '', qq: '', url: '' })

const accountLabel = computed(() => {
  switch (form.value.stype) {
    case '1': return '支付宝账号'
    case '2': return '微信 OpenId / 微信号'
    case '3': return 'QQ 号码'
    case '4': return '银行卡号'
    default: return '收款账号'
  }
})
const canSubmit = computed(() => form.value.account && form.value.username)
function submit() {
  if (!canSubmit.value) return
  router.push('/m')
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-content px-4 py-8">
    <div class="w-full max-w-md">
      <div class="mb-6 flex flex-col items-center gap-3">
        <div class="flex size-12 items-center justify-center rounded-xl bg-primary text-primary-foreground"><Store class="size-6" /></div>
        <div class="text-center">
          <div class="text-xl font-bold tracking-tight">完善账户信息</div>
          <div class="mt-1 text-sm text-muted-foreground">首次登录需完善收款信息才能开始收款</div>
        </div>
      </div>

      <div class="border border-border bg-background p-6 shadow-sm">
        <div class="space-y-3.5">
          <div class="row-field">
            <label class="lbl">结算方式</label>
            <Select v-model="form.stype" :options="stypeOptions" class="flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">{{ accountLabel }}</label>
            <input v-model="form.account" class="field-input flex-1" placeholder="收款到账账号" />
          </div>
          <div class="row-field">
            <label class="lbl">真实姓名</label>
            <input v-model="form.username" class="field-input flex-1" placeholder="收款账户实名" />
          </div>
          <div class="row-field">
            <label class="lbl">QQ</label>
            <input v-model="form.qq" class="field-input flex-1" placeholder="选填" />
          </div>
          <div class="row-field">
            <label class="lbl">网站域名</label>
            <input v-model="form.url" class="field-input flex-1" placeholder="选填，如 shop.abc.com" />
          </div>
          <div class="border-t border-border/60 pt-4">
            <Button class="w-full" :disabled="!canSubmit" @click="submit">保存并进入商户中心</Button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
