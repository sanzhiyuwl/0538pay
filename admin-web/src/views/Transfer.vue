<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { Send, RefreshCw, Info } from 'lucide-vue-next'
import { Panel, Button, Select } from '@/components/ui'
import { transferApps, transferChannels } from '@/lib/mock/transfer'
import { createTransfer, createTransferBatch, type TransferCreateReq, type BatchItemResult } from '@/lib/api/transfer'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const router = useRouter()

// 当前付款方式 Tab
const activeApp = ref<'alipay' | 'wxpay' | 'qqpay' | 'bank'>('alipay')
const currentApp = computed(() => transferApps.find((a) => a.key === activeApp.value)!)
const channelOptions = computed(() =>
  transferChannels[activeApp.value].map((c) => ({ value: c.id, label: c.name })),
)

// 生成 19 位交易号（YmdHis + 5位随机）
function genBizNo() {
  const now = new Date()
  const p = (n: number, l = 2) => String(n).padStart(l, '0')
  const ymdhis =
    now.getFullYear() +
    p(now.getMonth() + 1) +
    p(now.getDate()) +
    p(now.getHours()) +
    p(now.getMinutes()) +
    p(now.getSeconds())
  const rand = String(Math.floor(11111 + Math.random() * 88888))
  return (ymdhis + rand).slice(0, 19)
}

const form = ref({
  channel: channelOptions.value[0]?.value ?? 0,
  bizNo: genBizNo(),
  account: '',
  realName: '',
  money: '',
  desc: '',
  paypwd: '',
})
const busy = ref(false)

// 切 Tab：重置通道 + 重新生成交易号
function switchApp(key: typeof activeApp.value) {
  activeApp.value = key
  form.value.channel = transferChannels[key][0]?.id ?? 0
  form.value.bizNo = genBizNo()
}

function regenBizNo() {
  form.value.bizNo = genBizNo()
}

async function submit() {
  if (busy.value) return
  const money = Number(form.value.money)
  if (!form.value.account.trim()) return toast.error('请填写收款账号')
  if (!(money > 0)) return toast.error('请输入有效的转账金额')
  if (!form.value.paypwd) return toast.error('请输入支付密码')

  busy.value = true
  try {
    const body: TransferCreateReq = {
      biz_no: form.value.bizNo,
      type: activeApp.value,
      channel: form.value.channel,
      account: form.value.account.trim(),
      username: form.value.realName.trim(),
      money: String(form.value.money),
      desc: form.value.desc.trim(),
      password: form.value.paypwd,
    }
    const res = await createTransfer(body)
    toast.success(`已提交代付，交易号 ${res.biz_no}`)
    router.push('/admin/transfer-records')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '提交失败')
  } finally {
    busy.value = false
  }
}

// ===== 批量代付（C-2）=====
const batchApp = ref<'alipay' | 'wxpay' | 'qqpay' | 'bank'>('alipay')
const batchChannelOptions = computed(() =>
  transferChannels[batchApp.value].map((c) => ({ value: c.id, label: c.name })),
)
const batchForm = ref({ channel: batchChannelOptions.value[0]?.value ?? 0, text: '', desc: '', paypwd: '' })
const batchBusy = ref(false)
const batchResults = ref<BatchItemResult[]>([])

function switchBatchApp(key: typeof batchApp.value) {
  batchApp.value = key
  batchForm.value.channel = transferChannels[key][0]?.id ?? 0
}

// 解析批量文本：每行「账号,姓名,金额,[备注]」
function parseBatch(): TransferCreateReq[] {
  const items: TransferCreateReq[] = []
  for (const raw of batchForm.value.text.split('\n')) {
    const line = raw.trim()
    if (!line) continue
    const parts = line.split(/[,，\t]/).map((s) => s.trim())
    items.push({
      type: batchApp.value,
      channel: batchForm.value.channel,
      account: parts[0] || '',
      username: parts[1] || '',
      money: parts[2] || '',
      desc: parts[3] || batchForm.value.desc.trim(),
      password: '',
    })
  }
  return items
}

async function submitBatch() {
  if (batchBusy.value) return
  const items = parseBatch()
  if (items.length === 0) return toast.error('请填写批量代付明细（每行：账号,姓名,金额,备注）')
  if (!batchForm.value.paypwd) return toast.error('请输入支付密码')
  batchBusy.value = true
  batchResults.value = []
  try {
    const res = await createTransferBatch(batchForm.value.paypwd, items)
    batchResults.value = res.results
    toast.success(`批量代付完成：成功 ${res.success}/${res.total} 条`)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '批量提交失败')
  } finally {
    batchBusy.value = false
  }
}
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="转账付款" subtitle="后台主动发起对外转账（代付 / 提现打款），支持支付宝 / 微信 / QQ钱包 / 银行卡">
      <!-- 付款方式 Tab -->
      <div class="mb-4 flex gap-1 border-b border-border">
        <button
          v-for="a in transferApps"
          :key="a.key"
          class="relative -mb-px border-b-2 px-4 py-2 text-sm transition-colors"
          :class="
            activeApp === a.key
              ? 'border-primary font-medium text-primary'
              : 'border-transparent text-muted-foreground hover:text-foreground'
          "
          @click="switchApp(a.key)"
        >
          {{ a.label }}
        </button>
      </div>

      <!-- 表单 -->
      <div class="max-w-2xl space-y-3.5">
        <div class="flex items-center gap-3">
          <label class="w-24 shrink-0 text-right text-sm text-muted-foreground">通道选择</label>
          <Select v-model="form.channel" :options="channelOptions" class="flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <label class="w-24 shrink-0 text-right text-sm text-muted-foreground">交易号</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.bizNo" class="field-input flex-1" />
            <Button variant="outline" size="sm" @click="regenBizNo"><RefreshCw />重新生成</Button>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <label class="w-24 shrink-0 text-right text-sm text-muted-foreground">{{ currentApp.accountLabel }}</label>
          <input v-model="form.account" :placeholder="currentApp.accountPlaceholder" class="field-input flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <label class="w-24 shrink-0 text-right text-sm text-muted-foreground">{{ currentApp.nameLabel }}</label>
          <input v-model="form.realName" placeholder="不填写则不校验真实姓名" class="field-input flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <label class="w-24 shrink-0 text-right text-sm text-muted-foreground">转账金额</label>
          <div class="relative flex-1">
            <span class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sm text-muted-foreground">¥</span>
            <input v-model="form.money" placeholder="RMB / 元" class="field-input w-full !pl-7" />
          </div>
        </div>
        <div class="flex items-center gap-3">
          <label class="w-24 shrink-0 text-right text-sm text-muted-foreground">转账备注</label>
          <input v-model="form.desc" placeholder="可留空，最多 32 字" maxlength="32" class="field-input flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <label class="w-24 shrink-0 text-right text-sm text-muted-foreground">支付密码</label>
          <input v-model="form.paypwd" type="password" placeholder="请输入管理员支付密码" class="field-input flex-1" />
        </div>

        <div class="flex items-center gap-2 pl-[7.5rem] pt-1">
          <Button :disabled="busy" @click="submit"><Send />立即转账</Button>
        </div>
      </div>

      <!-- 说明 -->
      <div class="mt-5 flex items-start gap-2 border-t border-border/60 pt-4 text-xs text-muted-foreground">
        <Info class="mt-0.5 size-3.5 shrink-0" />
        <p>交易号用于防止重复转账，同一交易号只能提交一次。后台发起不收手续费、不扣款。转账结果可在「付款记录」页面查看；真实渠道打款待渠道凭证接入，当前提交后进入处理中状态。</p>
      </div>
    </Panel>

    <!-- 批量代付（C-2，对齐 epay transfer_batch）-->
    <Panel title="批量代付" subtitle="一次性对多个收款人发起代付（每行一条，逗号分隔）">
      <div class="mb-4 flex gap-1 border-b border-border">
        <button
          v-for="a in transferApps"
          :key="a.key"
          class="relative -mb-px border-b-2 px-4 py-2 text-sm transition-colors"
          :class="batchApp === a.key ? 'border-primary font-medium text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'"
          @click="switchBatchApp(a.key)"
        >
          {{ a.label }}
        </button>
      </div>
      <div class="max-w-3xl space-y-3.5">
        <div class="flex items-center gap-3">
          <label class="w-24 shrink-0 text-right text-sm text-muted-foreground">通道选择</label>
          <Select v-model="batchForm.channel" :options="batchChannelOptions" class="flex-1" />
        </div>
        <div class="flex items-start gap-3">
          <label class="w-24 shrink-0 pt-2 text-right text-sm text-muted-foreground">批量明细</label>
          <textarea
            v-model="batchForm.text"
            rows="6"
            placeholder="每行一条：收款账号,收款姓名,金额,备注（可选）&#10;示例：&#10;2088xxxx,张三,100.00,货款&#10;2088yyyy,李四,58.5"
            class="field-input flex-1 font-mono text-[13px]"
          />
        </div>
        <div class="flex items-center gap-3">
          <label class="w-24 shrink-0 text-right text-sm text-muted-foreground">统一备注</label>
          <input v-model="batchForm.desc" placeholder="行内未填备注时用此值，最多32字" maxlength="32" class="field-input flex-1" />
        </div>
        <div class="flex items-center gap-3">
          <label class="w-24 shrink-0 text-right text-sm text-muted-foreground">支付密码</label>
          <input v-model="batchForm.paypwd" type="password" placeholder="请输入管理员支付密码" class="field-input flex-1" />
        </div>
        <div class="flex items-center gap-2 pl-[7.5rem] pt-1">
          <Button :disabled="batchBusy" @click="submitBatch"><Send />批量提交</Button>
        </div>
      </div>

      <!-- 批量结果 -->
      <div v-if="batchResults.length" class="mt-5 border-t border-border/60 pt-4">
        <div class="mb-2 text-sm font-medium">提交结果（{{ batchResults.filter((r) => r.success).length }}/{{ batchResults.length }} 成功）</div>
        <div class="overflow-x-auto">
          <table class="tbl w-full">
            <thead>
              <tr><th class="w-12">#</th><th>收款账号</th><th class="w-24">结果</th><th>交易号 / 失败原因</th></tr>
            </thead>
            <tbody>
              <tr v-for="r in batchResults" :key="r.index">
                <td class="dim tabular-nums">{{ r.index + 1 }}</td>
                <td class="font-mono text-[13px]">{{ r.account }}</td>
                <td>
                  <span v-if="r.success" class="text-success">成功</span>
                  <span v-else class="text-destructive">失败</span>
                </td>
                <td class="text-xs">{{ r.success ? r.biz_no : r.msg }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="mt-5 flex items-start gap-2 border-t border-border/60 pt-4 text-xs text-muted-foreground">
        <Info class="mt-0.5 size-3.5 shrink-0" />
        <p>每行格式：收款账号,收款姓名,金额,备注。逐条独立处理，单条失败不影响其余；成功条目进入「付款记录」处理中。单次上限 200 条。真实渠道打款待凭证。</p>
      </div>
    </Panel>
  </div>
</template>
