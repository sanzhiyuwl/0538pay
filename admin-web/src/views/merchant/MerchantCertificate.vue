<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ShieldCheck, User, Building2, Send, CheckCircle2, Info } from 'lucide-vue-next'
import { Panel, Button } from '@/components/ui'
import { fetchCertInfo, submitCert, type CertInfo } from '@/lib/api/merchantCenter'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const info = ref<CertInfo | null>(null)
const busy = ref(false)
const certified = computed(() => info.value?.cert === 1)

const certType = ref<0 | 1>(0)
const form = reactive({ certname: '', certno: '', certcorp: '' })

const canSubmit = computed(() => {
  if (certType.value === 1) return form.certcorp && form.certname && form.certno
  return form.certname && form.certno
})

async function load() {
  try {
    info.value = await fetchCertInfo()
    certType.value = (info.value.certtype === 1 ? 1 : 0)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载实名信息失败')
  }
}
onMounted(load)

async function submit() {
  if (!canSubmit.value || busy.value) return
  busy.value = true
  try {
    await submitCert({
      certtype: certType.value,
      certname: form.certname.trim(),
      certno: form.certno.trim(),
      certcorp: form.certcorp.trim(),
    })
    toast.success('实名信息已提交')
    await load()
  } catch (e) {
    // 后端对"第三方认证待凭证"返回业务错误，如实提示
    toast.error(e instanceof ApiError ? e.message : '提交失败')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <!-- 已认证：成功态 -->
    <Panel v-if="certified && info" title="实名认证" subtitle="您的账户已完成实名认证">
      <div class="mx-auto max-w-xl space-y-4">
        <div class="flex items-center gap-3 border border-success/30 bg-success/[0.06] px-4 py-3.5">
          <span class="flex size-10 shrink-0 items-center justify-center bg-success/[0.12] text-success">
            <ShieldCheck class="size-5" />
          </span>
          <div>
            <div class="text-sm font-semibold text-success">已完成实名认证</div>
            <div class="text-xs text-muted-foreground">认证时间 {{ info.certtime || '—' }}</div>
          </div>
        </div>
        <dl class="divide-y divide-border/60 bg-muted/40 px-4 text-sm">
          <div class="flex py-2.5"><dt class="w-24 text-muted-foreground">认证类型</dt><dd>{{ info.certtype === 1 ? '企业认证' : '个人认证' }}</dd></div>
          <div class="flex py-2.5"><dt class="w-24 text-muted-foreground">认证方式</dt><dd>{{ info.method }}</dd></div>
          <div class="flex py-2.5"><dt class="w-24 text-muted-foreground">真实姓名</dt><dd>{{ info.certname }}</dd></div>
          <div class="flex py-2.5"><dt class="w-24 text-muted-foreground">证件号码</dt><dd class="font-mono tabular-nums">{{ info.certno }}</dd></div>
          <div v-if="info.certcorp" class="flex py-2.5"><dt class="w-24 text-muted-foreground">企业名称</dt><dd>{{ info.certcorp }}</dd></div>
        </dl>
      </div>
    </Panel>

    <!-- 未认证：提交表单 -->
    <Panel v-else title="实名认证" :subtitle="info ? `认证方式：${info.method}` : ''">
      <div class="mx-auto max-w-2xl space-y-4">
        <!-- 主体类型 -->
        <div v-if="info?.corpopen" class="grid grid-cols-2 gap-3">
          <button
            v-for="opt in [
              { t: 0 as const, icon: User, name: '个人认证', desc: '以个人身份实名' },
              { t: 1 as const, icon: Building2, name: '企业认证', desc: '以企业主体实名' },
            ]"
            :key="opt.t"
            class="relative flex items-center gap-3 border px-4 py-3.5 text-left transition-colors"
            :class="certType === opt.t
              ? 'border-primary bg-primary/[0.04]'
              : 'border-border hover:border-primary/40'"
            @click="certType = opt.t"
          >
            <component
              :is="opt.icon"
              class="size-5 shrink-0"
              :class="certType === opt.t ? 'text-primary' : 'text-muted-foreground'"
            />
            <div class="min-w-0">
              <div class="text-sm font-medium" :class="certType === opt.t && 'text-primary'">{{ opt.name }}</div>
              <div class="mt-0.5 text-xs text-muted-foreground">{{ opt.desc }}</div>
            </div>
            <CheckCircle2
              v-if="certType === opt.t"
              class="absolute right-3.5 top-3.5 size-4 text-primary"
            />
          </button>
        </div>

        <!-- 企业信息 -->
        <template v-if="certType === 1">
          <div class="row-field">
            <label class="lbl">公司名称</label>
            <input v-model="form.certcorp" class="field-input flex-1" placeholder="营业执照上的公司全称" />
          </div>
          <div class="border-t border-border/60 pt-3.5 text-sm font-medium text-muted-foreground">法人信息</div>
        </template>

        <div class="row-field">
          <label class="lbl">{{ certType === 1 ? '法人姓名' : '真实姓名' }}</label>
          <input v-model="form.certname" class="field-input flex-1" placeholder="请输入真实姓名" />
        </div>
        <div class="row-field">
          <label class="lbl">身份证号</label>
          <input v-model="form.certno" class="field-input flex-1" placeholder="18 位身份证号码" />
        </div>

        <div class="flex gap-2.5 bg-muted/40 px-3.5 py-3 text-xs leading-relaxed text-muted-foreground">
          <Info class="mt-0.5 size-4 shrink-0 text-muted-foreground/70" />
          <p>
            实名认证需通过第三方（支付宝/微信/阿里云）核验，工本费
            <b class="text-foreground">¥{{ info?.certmoney ?? 0 }}</b>，认证成功才扣，失败不扣费。
            当前第三方认证渠道待接入凭证，提交后信息将暂存待核验。
          </p>
        </div>
        <div class="border-t border-border/60 pt-4">
          <Button :disabled="!canSubmit || busy" @click="submit"><Send />提交实名信息</Button>
        </div>
      </div>
    </Panel>
  </div>
</template>
