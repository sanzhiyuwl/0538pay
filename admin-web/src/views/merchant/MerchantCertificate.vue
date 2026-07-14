<script setup lang="ts">
import { ref, computed } from 'vue'
import { Check, ScanLine, ShieldCheck, User, Building2, ArrowUpCircle } from 'lucide-vue-next'
import { Panel, Button, Badge } from '@/components/ui'

// 认证状态：mock 开关。true=已认证（展示成功态），false=未认证（走流程）
const certified = ref(false)

// 已认证信息（成功态展示）
const certInfo = {
  certtype: 0, // 0 个人 1 企业
  certname: '张*伟',
  certno: '370902********1234',
  certtime: '2026-06-20 14:32:10',
  method: '支付宝身份验证',
}

// ===== 认证流程 =====
// 认证方式（由平台配置，mock 固定为支付宝扫码）
const certMethod = '支付宝身份验证'
// 主体类型：0 个人 1 企业（平台开启企业认证时可选）
const corpEnabled = true
const certType = ref<0 | 1>(0)
// 步骤：1 填信息 2 扫码 3 完成
const step = ref(1)

const form = ref({
  certname: '',
  certno: '',
  certcorpname: '',
  certcorpno: '',
})

const canNext = computed(() => {
  if (certType.value === 1) {
    return form.value.certcorpname && form.value.certcorpno && form.value.certname && form.value.certno
  }
  return form.value.certname && form.value.certno
})

function toScan() {
  if (!canNext.value) return
  step.value = 2
}
function finishCert() {
  step.value = 3
}
function resetFlow() {
  step.value = 1
}

const steps = [
  { n: 1, label: '填写认证信息' },
  { n: 2, label: '扫码快捷认证' },
  { n: 3, label: '认证完成' },
]
</script>

<template>
  <div class="space-y-2.5">
    <!-- 已认证：成功态 -->
    <Panel v-if="certified" title="实名认证" subtitle="您的账户已完成实名认证">
      <div class="max-w-xl">
        <div class="flex items-center gap-3 border border-success/30 bg-success/[0.06] px-4 py-3">
          <ShieldCheck class="size-6 text-success" />
          <div>
            <div class="text-sm font-medium text-success">已认证</div>
            <div class="text-xs text-muted-foreground">认证时间 {{ certInfo.certtime }}</div>
          </div>
        </div>
        <div class="mt-4 space-y-3 text-sm">
          <div class="flex"><span class="w-24 text-muted-foreground">认证类型</span><span>{{ certInfo.certtype === 1 ? '企业认证' : '个人认证' }}</span></div>
          <div class="flex"><span class="w-24 text-muted-foreground">认证方式</span><span>{{ certInfo.method }}</span></div>
          <div class="flex"><span class="w-24 text-muted-foreground">真实姓名</span><span>{{ certInfo.certname }}</span></div>
          <div class="flex"><span class="w-24 text-muted-foreground">证件号码</span><span class="font-mono">{{ certInfo.certno }}</span></div>
        </div>
        <div v-if="certInfo.certtype === 0" class="mt-5 border-t border-border/60 pt-4">
          <Button variant="outline"><ArrowUpCircle />升级到企业认证</Button>
        </div>
      </div>
    </Panel>

    <!-- 未认证：认证流程 -->
    <Panel v-else title="实名认证" :subtitle="`当前认证方式：${certMethod}`">
      <!-- 步骤条 -->
      <div class="mb-6 flex items-center">
        <template v-for="(s, i) in steps" :key="s.n">
          <div class="flex items-center gap-2">
            <div
              class="flex size-7 items-center justify-center rounded-full text-xs font-medium"
              :class="step >= s.n ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground'"
            >
              <Check v-if="step > s.n" class="size-4" />
              <span v-else>{{ s.n }}</span>
            </div>
            <span class="text-sm" :class="step >= s.n ? 'font-medium text-foreground' : 'text-muted-foreground'">{{ s.label }}</span>
          </div>
          <div v-if="i < steps.length - 1" class="mx-3 h-px w-12 flex-none" :class="step > s.n ? 'bg-primary' : 'bg-border'" />
        </template>
      </div>

      <!-- 步骤1：填写信息 -->
      <div v-if="step === 1" class="max-w-2xl space-y-3.5">
        <!-- 主体类型 -->
        <div v-if="corpEnabled" class="flex gap-3">
          <button
            class="flex flex-1 items-center gap-3 border p-4 text-left transition-colors"
            :class="certType === 0 ? 'border-primary ring-1 ring-primary' : 'border-border hover:border-primary/50'"
            @click="certType = 0"
          >
            <User class="size-5" :class="certType === 0 ? 'text-primary' : 'text-muted-foreground'" />
            <div>
              <div class="text-sm font-medium">个人认证</div>
              <div class="text-xs text-muted-foreground">以个人身份实名</div>
            </div>
          </button>
          <button
            class="flex flex-1 items-center gap-3 border p-4 text-left transition-colors"
            :class="certType === 1 ? 'border-primary ring-1 ring-primary' : 'border-border hover:border-primary/50'"
            @click="certType = 1"
          >
            <Building2 class="size-5" :class="certType === 1 ? 'text-primary' : 'text-muted-foreground'" />
            <div>
              <div class="text-sm font-medium">企业认证</div>
              <div class="text-xs text-muted-foreground">以企业主体实名</div>
            </div>
          </button>
        </div>

        <!-- 企业信息 -->
        <template v-if="certType === 1">
          <div class="row-field">
            <label class="lbl">公司名称</label>
            <input v-model="form.certcorpname" class="field-input flex-1" placeholder="营业执照上的公司全称" />
          </div>
          <div class="row-field">
            <label class="lbl">统一社会信用代码</label>
            <input v-model="form.certcorpno" class="field-input flex-1" placeholder="营业执照编号" />
          </div>
          <div class="border-t border-border/60 pt-3.5 text-sm font-medium text-muted-foreground">法人信息</div>
        </template>

        <!-- 个人 / 法人信息 -->
        <div class="row-field">
          <label class="lbl">{{ certType === 1 ? '法人姓名' : '真实姓名' }}</label>
          <input v-model="form.certname" class="field-input flex-1" placeholder="请输入真实姓名" />
        </div>
        <div class="row-field">
          <label class="lbl">身份证号</label>
          <input v-model="form.certno" class="field-input flex-1" placeholder="请输入身份证号码" />
        </div>

        <p class="text-xs text-muted-foreground">实名认证需支付认证费用，认证失败不扣费。请确保信息真实准确。</p>
        <div class="border-t border-border/60 pt-4">
          <Button :disabled="!canNext" @click="toScan"><ScanLine />下一步：扫码认证</Button>
        </div>
      </div>

      <!-- 步骤2：扫码 -->
      <div v-else-if="step === 2" class="flex max-w-2xl flex-col items-center gap-4 py-4">
        <div class="text-sm text-muted-foreground">请使用支付宝扫描下方二维码完成快捷认证</div>
        <div class="flex size-48 items-center justify-center border border-border bg-muted/30 text-muted-foreground/40">
          <ScanLine class="size-16" />
        </div>
        <div class="text-xs text-muted-foreground">正在等待扫码认证结果…</div>
        <div class="flex gap-2">
          <Button variant="outline" size="sm" @click="resetFlow">返回修改</Button>
          <Button size="sm" @click="finishCert">（模拟）认证成功</Button>
        </div>
      </div>

      <!-- 步骤3：完成 -->
      <div v-else class="flex max-w-2xl flex-col items-center gap-3 py-8">
        <div class="flex size-16 items-center justify-center rounded-full bg-success/10 text-success">
          <Check class="size-8" />
        </div>
        <div class="text-lg font-semibold">认证成功</div>
        <div class="text-sm text-muted-foreground">您的账户已完成实名认证</div>
        <Badge variant="success">已认证</Badge>
      </div>
    </Panel>
  </div>
</template>
