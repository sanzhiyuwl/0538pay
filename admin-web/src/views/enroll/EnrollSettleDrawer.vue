<script setup lang="ts">
import { ref, reactive, watch } from 'vue'
import { Drawer, Button, Badge, Select } from '@/components/ui'
import type { SettleModifyReq, SettlementView, SettleApplicationView } from '@/lib/api/console'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

/**
 * 「结算账户管理」抽屉（console / agent 共用；自研扩展，微信接口 6/7/8）。
 * 仅 finished（已拿到 sub_mchid）的进件单可用：查当前结算账户 / 修改结算账户 / 查改单审核状态。
 * 敏感字段（银行账号/开户名）明文提交，后端 RSA-OAEP 加密。所有 API 由父组件按端注入。
 */
const props = defineProps<{
  modelValue: boolean
  enrollId: number | null
  merchantName?: string
  subMchId?: string
  getFn: (id: number) => Promise<SettlementView>
  modifyFn: (id: number, body: SettleModifyReq) => Promise<{ application_no: string }>
  getApplicationFn: (id: number) => Promise<SettleApplicationView>
}>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean]; changed: [] }>()

const accountTypeOptions = [
  { value: 'ACCOUNT_TYPE_BUSINESS', label: '对公银行账户' },
  { value: 'ACCOUNT_TYPE_PRIVATE', label: '经营者个人银行卡' },
]

// 查询当前结算账户
const querying = ref(false)
const current = ref<SettlementView | null>(null)
const verifyMeta: Record<string, { label: string; variant: 'success' | 'warning' | 'destructive' | 'muted' }> = {
  VERIFY_SUCCESS: { label: '验证成功', variant: 'success' },
  VERIFYING: { label: '验证中', variant: 'warning' },
  VERIFY_FAIL: { label: '验证失败', variant: 'destructive' },
}
// 改单审核状态
const auditMeta: Record<string, { label: string; variant: 'success' | 'warning' | 'destructive' | 'muted' }> = {
  AUDIT_SUCCESS: { label: '审核成功', variant: 'success' },
  AUDITING: { label: '审核中', variant: 'warning' },
  AUDIT_FAIL: { label: '审核驳回', variant: 'destructive' },
}

async function doQuery() {
  if (!props.enrollId) return
  querying.value = true
  try {
    current.value = await props.getFn(props.enrollId)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '查询结算账户失败')
  } finally {
    querying.value = false
  }
}

// 修改结算账户
const modifying = ref(false)
const form = reactive<SettleModifyReq>({
  account_type: 'ACCOUNT_TYPE_BUSINESS',
  account_bank: '',
  bank_name: '',
  bank_branch_id: '',
  account_number: '',
  account_name: '',
})
function resetForm() {
  Object.assign(form, {
    account_type: 'ACCOUNT_TYPE_BUSINESS',
    account_bank: '',
    bank_name: '',
    bank_branch_id: '',
    account_number: '',
    account_name: '',
  })
}
async function doModify() {
  if (!props.enrollId) return
  if (!form.account_bank.trim()) {
    toast.error('请填写开户银行')
    return
  }
  if (!form.account_number.trim()) {
    toast.error('请填写银行账号')
    return
  }
  modifying.value = true
  try {
    const { application_no } = await props.modifyFn(props.enrollId, { ...form })
    toast.success(`修改申请已提交，申请单号 ${application_no}`)
    resetForm()
    emit('changed')
    await doQueryApplication()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '修改结算账户失败')
  } finally {
    modifying.value = false
  }
}

// 查改单审核状态
const queryingApp = ref(false)
const application = ref<SettleApplicationView | null>(null)
async function doQueryApplication() {
  if (!props.enrollId) return
  queryingApp.value = true
  try {
    application.value = await props.getApplicationFn(props.enrollId)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '查询改单状态失败')
  } finally {
    queryingApp.value = false
  }
}

// 打开抽屉时清空上次结果，自动拉一次当前账户
watch(
  () => [props.modelValue, props.enrollId] as const,
  ([open, id]) => {
    if (open && id) {
      current.value = null
      application.value = null
      resetForm()
      doQuery()
    }
  },
  { immediate: true },
)
</script>

<template>
  <Drawer
    :model-value="modelValue"
    width="max-w-xl"
    title="结算账户管理"
    :subtitle="merchantName ? `${merchantName} · 特约商户号 ${subMchId || '—'}` : '进件成功后管理结算银行账户'"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="space-y-5">
      <!-- 当前生效的结算账户 -->
      <section class="space-y-2">
        <div class="flex items-center justify-between">
          <h4 class="text-sm font-medium">当前结算账户</h4>
          <Button size="sm" variant="outline" :disabled="querying" @click="doQuery">
            {{ querying ? '查询中…' : '刷新' }}
          </Button>
        </div>
        <div v-if="current" class="space-y-1.5 bg-muted/40 px-3 py-2.5 text-sm">
          <div class="flex justify-between"><span class="dim">账户类型</span><span>{{ current.account_type === 'ACCOUNT_TYPE_BUSINESS' ? '对公银行账户' : '经营者个人银行卡' }}</span></div>
          <div class="flex justify-between"><span class="dim">开户银行</span><span>{{ current.account_bank }}</span></div>
          <div v-if="current.bank_name" class="flex justify-between"><span class="dim">开户支行</span><span>{{ current.bank_name }}</span></div>
          <div class="flex justify-between"><span class="dim">银行账号</span><span class="tabular-nums">{{ current.account_number }}</span></div>
          <div class="flex justify-between">
            <span class="dim">验证结果</span>
            <Badge :variant="verifyMeta[current.verify_result]?.variant ?? 'muted'">
              {{ verifyMeta[current.verify_result]?.label ?? current.verify_result }}
            </Badge>
          </div>
          <div v-if="current.verify_fail_reason" class="text-destructive text-xs">{{ current.verify_fail_reason }}</div>
        </div>
        <div v-else class="dim py-3 text-center text-xs">点「刷新」查询当前生效的结算账户</div>
      </section>

      <!-- 修改结算账户 -->
      <section class="space-y-3">
        <h4 class="text-sm font-medium">修改结算账户 <span class="text-[11px] text-muted-foreground">（银行账号/开户名加密提交）</span></h4>
        <div class="row-field">
          <label class="lbl">账户类型</label>
          <Select v-model="form.account_type" :options="accountTypeOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">开户银行<span class="text-destructive">*</span></label>
          <input v-model="form.account_bank" placeholder="如 招商银行" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">开户支行</label>
          <input v-model="form.bank_name" placeholder="开户银行全称（含支行），按需" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">联行号</label>
          <input v-model="form.bank_branch_id" placeholder="开户银行联行号，按需" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">银行账号<span class="text-destructive">*</span></label>
          <input v-model="form.account_number" placeholder="数字，不超过 32 位" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">开户名称</label>
          <input v-model="form.account_name" placeholder="不改可留空；对公=主体名称，对私=经营者名称" class="field-input flex-1" />
        </div>
        <div class="flex justify-end">
          <Button :disabled="modifying" @click="doModify">{{ modifying ? '提交中…' : '提交修改' }}</Button>
        </div>
      </section>

      <!-- 改单审核状态 -->
      <section class="space-y-2">
        <div class="flex items-center justify-between">
          <h4 class="text-sm font-medium">最近改单审核状态</h4>
          <Button size="sm" variant="outline" :disabled="queryingApp" @click="doQueryApplication">
            {{ queryingApp ? '查询中…' : '查改单状态' }}
          </Button>
        </div>
        <div v-if="application" class="space-y-1.5 bg-muted/40 px-3 py-2.5 text-sm">
          <div class="flex justify-between"><span class="dim">开户名称</span><span>{{ application.account_name }}</span></div>
          <div class="flex justify-between"><span class="dim">开户银行</span><span>{{ application.account_bank }}</span></div>
          <div class="flex justify-between"><span class="dim">银行账号</span><span class="tabular-nums">{{ application.account_number }}</span></div>
          <div class="flex justify-between">
            <span class="dim">审核状态</span>
            <Badge :variant="auditMeta[application.verify_result]?.variant ?? 'muted'">
              {{ auditMeta[application.verify_result]?.label ?? application.verify_result }}
            </Badge>
          </div>
          <div v-if="application.verify_fail_reason" class="text-destructive text-xs">{{ application.verify_fail_reason }}</div>
          <div v-if="application.verify_finish_time" class="flex justify-between"><span class="dim">更新时间</span><span class="tabular-nums text-xs">{{ application.verify_finish_time }}</span></div>
        </div>
        <div v-else class="dim py-3 text-center text-xs">提交修改后点「查改单状态」查看审核进度</div>
      </section>

      <p class="text-[11px] text-muted-foreground">
        修改结算账户直接影响特约商户实际收款到账银行卡。审核期间系统可能向新账户打款 0.01 元验证；
        每个商户每天仅能提交 5 次修改申请。银行账号/开户名经加密传输，不明文落库。
      </p>
    </div>
    <template #footer>
      <Button variant="outline" @click="emit('update:modelValue', false)">关闭</Button>
    </template>
  </Drawer>
</template>
