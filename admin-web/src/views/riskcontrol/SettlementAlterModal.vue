<script setup lang="ts">
/** 解脱代办·修改结算账户（风控第二段解脱路径 recover_way=MODIFY_SETTLE_ACCOUNT_INFORMATION）。 */
import { ref, reactive } from 'vue'
import { Modal, Button, Select } from '@/components/ui'
import { adminModifySettlement, type ModifySettlementReq } from '@/lib/api/riskControl'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const props = defineProps<{ modelValue: boolean; enrollId: number | null }>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean]; done: [] }>()
const toast = useToast()

const accountTypeOptions = [
  { value: 'ACCOUNT_TYPE_BUSINESS', label: '对公账户' },
  { value: 'ACCOUNT_TYPE_PRIVATE', label: '对私账户（经营者个人银行卡）' },
]

const form = reactive<ModifySettlementReq>({
  account_type: 'ACCOUNT_TYPE_BUSINESS',
  account_bank: '',
  bank_name: '',
  bank_branch_id: '',
  account_number: '',
  account_name: '',
})

const submitting = ref(false)
async function submit() {
  if (!props.enrollId || submitting.value) return
  if (!form.account_bank.trim() || !form.account_number.trim()) {
    toast.error('请填写开户银行与银行账号')
    return
  }
  submitting.value = true
  try {
    const res = await adminModifySettlement(props.enrollId, form)
    toast.success(`结算账户变更申请已提交，申请单号 ${res.application_no}`)
    emit('update:modelValue', false)
    emit('done')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '提交失败')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Modal :model-value="modelValue" title="解脱代办·修改结算账户" @update:model-value="(v) => emit('update:modelValue', v)">
    <div class="space-y-3">
      <p class="text-xs leading-relaxed text-muted-foreground">
        代该商户向微信提交结算银行账户变更申请，提交后由微信审核变更。
      </p>
      <div>
        <label class="mb-1 block text-xs text-muted-foreground">账户类型</label>
        <Select v-model="form.account_type" :options="accountTypeOptions" class="w-full" />
      </div>
      <div>
        <label class="mb-1 block text-xs text-muted-foreground">开户银行</label>
        <input v-model="form.account_bank" class="field-input w-full" placeholder="如：中国工商银行 / 其他银行" />
      </div>
      <div v-if="form.account_bank === '其他银行'" class="grid grid-cols-2 gap-2">
        <input v-model="form.bank_branch_id" class="field-input" placeholder="联行号（与开户行全称二选一）" />
        <input v-model="form.bank_name" class="field-input" placeholder="开户行全称（含支行）" />
      </div>
      <div>
        <label class="mb-1 block text-xs text-muted-foreground">银行账号</label>
        <input v-model="form.account_number" class="field-input w-full" placeholder="必填" />
      </div>
      <div>
        <label class="mb-1 block text-xs text-muted-foreground">开户名称</label>
        <input v-model="form.account_name" class="field-input w-full" placeholder="对私账户需与法人姓名一致" />
      </div>
    </div>
    <template #footer>
      <Button variant="outline" :disabled="submitting" @click="emit('update:modelValue', false)">取消</Button>
      <Button :disabled="submitting" @click="submit">提交申请</Button>
    </template>
  </Modal>
</template>
