<script setup lang="ts">
import { ref, reactive, watch } from 'vue'
import { Drawer, Button, Select } from '@/components/ui'
import type { EnrollMaterialReq, EnrollMaterialView } from '@/lib/api/console'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const toast = useToast()

// 收单同款 Select 选项
const subjectTypeOptions = [
  { value: 'SUBJECT_TYPE_INDIVIDUAL', label: '个体户' },
  { value: 'SUBJECT_TYPE_ENTERPRISE', label: '企业' },
]
const bankAccountTypeOptions = [
  { value: 'BANK_ACCOUNT_TYPE_PERSONAL', label: '对私（个人）' },
  { value: 'BANK_ACCOUNT_TYPE_CORPORATE', label: '对公（企业）' },
]

/**
 * 「填全套资料」抽屉（console / agent 共用；自研扩展）。
 * 对齐微信 APIv3 特约商户进件 applyment4sub 核心字段，一期覆盖个体户/企业。
 * 敏感字段（身份证/银行账号/超管手机等）明文提交，后端 RSA-OAEP 加密落库；
 * 回显时敏感字段一律不回原文，仅显示「已填/未填」，改填要重新输入原文。
 * 图片类字段（营业执照/身份证正反面）走图片上传：选图 → 上传微信 media/upload → 拿 media_id 回填。
 */
const props = defineProps<{
  modelValue: boolean
  enrollId: number | null
  merchantName?: string
  fetchFn: (id: number) => Promise<EnrollMaterialView>
  submitFn: (id: number, body: EnrollMaterialReq) => Promise<unknown>
  uploadFn: (id: number, file: File) => Promise<{ media_id: string }>
}>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean]; saved: [] }>()

const loading = ref(false)
const saving = ref(false)
// 敏感字段是否已填（回显）——用于给输入框加「已填，留空则不改」提示
const has = reactive({
  idCardName: false,
  idCardNumber: false,
  accountName: false,
  accountNumber: false,
  contactName: false,
  contactIdNumber: false,
  mobilePhone: false,
  contactEmail: false,
})

function blank(): EnrollMaterialReq {
  return {
    subject_type: 'SUBJECT_TYPE_INDIVIDUAL',
    merchant_shortname: '',
    service_phone: '',
    license_number: '',
    license_copy: '',
    legal_person: '',
    license_address: '',
    period_begin: '',
    period_end: '',
    id_card_name: '',
    id_card_number: '',
    id_card_copy: '',
    id_card_national: '',
    card_period_begin: '',
    card_period_end: '',
    bank_account_type: 'BANK_ACCOUNT_TYPE_PERSONAL',
    account_name: '',
    account_bank: '',
    bank_address_code: '',
    account_number: '',
    contact_name: '',
    contact_id_number: '',
    mobile_phone: '',
    contact_email: '',
  }
}
const form = reactive<EnrollMaterialReq>(blank())

async function loadView(id: number) {
  loading.value = true
  try {
    const v = await props.fetchFn(id)
    Object.assign(form, blank())
    // 非敏感字段回填
    form.subject_type = v.subject_type || 'SUBJECT_TYPE_INDIVIDUAL'
    form.merchant_shortname = v.merchant_shortname || ''
    form.service_phone = v.service_phone || ''
    form.license_number = v.license_number || ''
    form.license_copy = v.license_copy || ''
    form.legal_person = v.legal_person || ''
    form.license_address = v.license_address || ''
    form.period_begin = v.period_begin || ''
    form.period_end = v.period_end || ''
    form.id_card_copy = v.id_card_copy || ''
    form.id_card_national = v.id_card_national || ''
    form.card_period_begin = v.card_period_begin || ''
    form.card_period_end = v.card_period_end || ''
    form.bank_account_type = v.bank_account_type || 'BANK_ACCOUNT_TYPE_PERSONAL'
    form.account_bank = v.account_bank || ''
    form.bank_address_code = v.bank_address_code || ''
    // 敏感字段只回是否已填（不回原文）
    has.idCardName = v.has_id_card_name
    has.idCardNumber = v.has_id_card_number
    has.accountName = v.has_account_name
    has.accountNumber = v.has_account_number
    has.contactName = v.has_contact_name
    has.contactIdNumber = v.has_contact_id_number
    has.mobilePhone = v.has_mobile_phone
    has.contactEmail = v.has_contact_email
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载资料失败')
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.modelValue, props.enrollId] as const,
  ([open, id]) => {
    if (open && id) loadView(id)
  },
  { immediate: true },
)

const sensitivePlaceholder = (filled: boolean) => (filled ? '已填（如需修改请重新输入原文）' : '必填')

// —— 图片上传（营业执照/身份证正反面）——
// 三个 media_id 字段各自的上传中状态，按 form 字段名区分。
const uploading = reactive<Record<string, boolean>>({
  license_copy: false,
  id_card_copy: false,
  id_card_national: false,
})

// 选图即上传：校验类型/大小 → uploadFn 换 media_id → 回填对应字段。
async function onPickImage(field: 'license_copy' | 'id_card_copy' | 'id_card_national', e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  const okExt = /\.(jpe?g|png|bmp)$/i.test(file.name)
  if (!okExt) {
    toast.error('图片仅支持 JPG/PNG/BMP 格式')
    input.value = ''
    return
  }
  if (file.size > 2 * 1024 * 1024) {
    toast.error('图片不能超过 2M，请压缩后重试')
    input.value = ''
    return
  }
  if (!props.enrollId) return
  uploading[field] = true
  try {
    const { media_id } = await props.uploadFn(props.enrollId, file)
    form[field] = media_id
    toast.success('图片已上传')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '图片上传失败')
  } finally {
    uploading[field] = false
    input.value = '' // 允许重选同名文件
  }
}

async function doSave() {
  if (!props.enrollId) return
  if (!form.merchant_shortname.trim()) {
    toast.error('请填写商户简称')
    return
  }
  // 敏感字段：未填过则本次必须填；填过可留空表示不改（后端未收到值时保留原密文）。
  const requireIfNew = (val: string, filled: boolean, label: string): boolean => {
    if (!filled && !val.trim()) {
      toast.error(`请填写${label}`)
      return false
    }
    return true
  }
  if (!requireIfNew(form.id_card_name, has.idCardName, '证件姓名')) return
  if (!requireIfNew(form.id_card_number, has.idCardNumber, '证件号码')) return
  if (!requireIfNew(form.account_name, has.accountName, '开户名称')) return
  if (!requireIfNew(form.account_number, has.accountNumber, '银行账号')) return
  if (!requireIfNew(form.contact_name, has.contactName, '超管姓名')) return
  if (!requireIfNew(form.mobile_phone, has.mobilePhone, '超管手机号')) return

  saving.value = true
  try {
    await props.submitFn(props.enrollId, { ...form })
    toast.success('资料已保存')
    emit('saved')
    emit('update:modelValue', false)
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '保存资料失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <Drawer
    :model-value="modelValue"
    width="max-w-2xl"
    title="填全套资料"
    :subtitle="merchantName ? `${merchantName} · 提交微信审核前的完整进件资料` : '提交微信审核前的完整进件资料'"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div v-if="loading" class="py-10 text-center dim">加载中…</div>
    <div v-else class="space-y-5">
      <!-- 主体基础 -->
      <section class="space-y-3">
        <h4 class="text-sm font-medium">主体基础</h4>
        <div class="row-field">
          <label class="lbl">主体类型</label>
          <Select v-model="form.subject_type" :options="subjectTypeOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">商户简称<span class="text-destructive">*</span></label>
          <input v-model="form.merchant_shortname" placeholder="展示给顾客的简称" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">客服电话</label>
          <input v-model="form.service_phone" placeholder="对外客服电话" class="field-input flex-1" />
        </div>
      </section>

      <!-- 营业执照 -->
      <section class="space-y-3">
        <h4 class="text-sm font-medium">营业执照</h4>
        <div class="row-field">
          <label class="lbl">证照编号</label>
          <input v-model="form.license_number" placeholder="统一社会信用代码" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">经营者/法人</label>
          <input v-model="form.legal_person" placeholder="执照上的经营者/法人姓名" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">注册地址</label>
          <input v-model="form.license_address" placeholder="执照注册地址" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">营业期限</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.period_begin" placeholder="开始 yyyy-MM-dd" class="field-input flex-1" />
            <span class="dim">至</span>
            <input v-model="form.period_end" placeholder="结束（长期填 长期）" class="field-input flex-1" />
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">执照照片</label>
          <div class="flex flex-1 items-center gap-2">
            <label class="media-btn">
              <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.license_copy" @change="onPickImage('license_copy', $event)" />
              {{ uploading.license_copy ? '上传中…' : form.license_copy ? '重新上传' : '选择图片' }}
            </label>
            <span v-if="form.license_copy" class="media-ok">已上传 ✓</span>
            <span v-else class="dim text-xs">JPG/PNG/BMP，≤2M</span>
          </div>
        </div>
      </section>

      <!-- 经营者/法人身份（敏感）-->
      <section class="space-y-3">
        <h4 class="text-sm font-medium">经营者/法人身份 <span class="text-[11px] text-muted-foreground">（敏感，加密存储）</span></h4>
        <div class="row-field">
          <label class="lbl">证件姓名<span class="text-destructive">*</span></label>
          <input v-model="form.id_card_name" :placeholder="sensitivePlaceholder(has.idCardName)" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">证件号码<span class="text-destructive">*</span></label>
          <input v-model="form.id_card_number" :placeholder="sensitivePlaceholder(has.idCardNumber)" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">身份证有效期</label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.card_period_begin" placeholder="开始 yyyy-MM-dd" class="field-input flex-1" />
            <span class="dim">至</span>
            <input v-model="form.card_period_end" placeholder="结束（长期填 长期）" class="field-input flex-1" />
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">人像面照片</label>
          <div class="flex flex-1 items-center gap-2">
            <label class="media-btn">
              <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.id_card_copy" @change="onPickImage('id_card_copy', $event)" />
              {{ uploading.id_card_copy ? '上传中…' : form.id_card_copy ? '重新上传' : '选择图片' }}
            </label>
            <span v-if="form.id_card_copy" class="media-ok">已上传 ✓</span>
            <span v-else class="dim text-xs">JPG/PNG/BMP，≤2M</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">国徽面照片</label>
          <div class="flex flex-1 items-center gap-2">
            <label class="media-btn">
              <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.id_card_national" @change="onPickImage('id_card_national', $event)" />
              {{ uploading.id_card_national ? '上传中…' : form.id_card_national ? '重新上传' : '选择图片' }}
            </label>
            <span v-if="form.id_card_national" class="media-ok">已上传 ✓</span>
            <span v-else class="dim text-xs">JPG/PNG/BMP，≤2M</span>
          </div>
        </div>
      </section>

      <!-- 结算银行账户 -->
      <section class="space-y-3">
        <h4 class="text-sm font-medium">结算银行账户 <span class="text-[11px] text-muted-foreground">（账号/户名敏感，加密存储）</span></h4>
        <div class="row-field">
          <label class="lbl">账户类型</label>
          <Select v-model="form.bank_account_type" :options="bankAccountTypeOptions" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">开户名称<span class="text-destructive">*</span></label>
          <input v-model="form.account_name" :placeholder="sensitivePlaceholder(has.accountName)" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">银行账号<span class="text-destructive">*</span></label>
          <input v-model="form.account_number" :placeholder="sensitivePlaceholder(has.accountNumber)" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">开户银行</label>
          <input v-model="form.account_bank" placeholder="如 工商银行" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">省市编码</label>
          <input v-model="form.bank_address_code" placeholder="开户银行省市编码" class="field-input flex-1" />
        </div>
      </section>

      <!-- 超管联系信息（敏感）-->
      <section class="space-y-3">
        <h4 class="text-sm font-medium">超级管理员 <span class="text-[11px] text-muted-foreground">（敏感，加密存储）</span></h4>
        <div class="row-field">
          <label class="lbl">超管姓名<span class="text-destructive">*</span></label>
          <input v-model="form.contact_name" :placeholder="sensitivePlaceholder(has.contactName)" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">超管证件号</label>
          <input v-model="form.contact_id_number" :placeholder="sensitivePlaceholder(has.contactIdNumber)" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">超管手机号<span class="text-destructive">*</span></label>
          <input v-model="form.mobile_phone" :placeholder="sensitivePlaceholder(has.mobilePhone)" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">超管邮箱</label>
          <input v-model="form.contact_email" :placeholder="sensitivePlaceholder(has.contactEmail)" class="field-input flex-1" />
        </div>
      </section>

      <p class="text-[11px] text-muted-foreground">
        敏感字段（身份证、银行账号、超管手机等）经 RSA-OAEP 加密后组装进 material_json，不明文落库；
        回显只显示「已填」不回原文。已填过的敏感字段留空表示不修改，保留原密文。
        执照/身份证图片选图后即上传至微信换取 media_id（原图不落库），需先在系统设置配好服务商凭证。
      </p>
    </div>
    <template #footer>
      <Button variant="outline" @click="emit('update:modelValue', false)">取消</Button>
      <Button :disabled="saving || loading" @click="doSave">{{ saving ? '保存中…' : '保存资料' }}</Button>
    </template>
  </Drawer>
</template>
