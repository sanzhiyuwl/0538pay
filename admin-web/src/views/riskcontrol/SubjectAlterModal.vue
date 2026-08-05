<script setup lang="ts">
/**
 * 解脱代办·修改主体资料（风控第二段解脱路径 recover_way=MODIFY_SUBJECT_INFORMATION）。
 * 字段与官方「修改主体信息API」(4014090649) 逐项对齐，不做自研裁剪：
 *   变更范围 alter_scope（全部/仅经营证件/仅受益人）→ 营业执照/登记证书二选一 → 法人身份信息
 *   → 金融机构许可证（按需）→ 最终受益人列表 UBO（按需）→ 补充材料（按需）。
 */
import { ref, reactive, computed } from 'vue'
import { Plus, Trash2, ImagePlus } from 'lucide-vue-next'
import { Modal, Button, Select } from '@/components/ui'
import { adminModifySubjectInfo, adminUploadChannelControlMedia, type SubjectAlterReq } from '@/lib/api/riskControl'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'

const props = defineProps<{ modelValue: boolean; enrollId: number | null }>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean]; done: [] }>()
const toast = useToast()

const scopeOptions = [
  { value: 'ALTER_SCOPE_FULL', label: '修改全部主体资料' },
  { value: 'ALTER_SCOPE_BUSINESS_CERT', label: '仅修改经营证件资料' },
  { value: 'ALTER_SCOPE_UBO', label: '仅修改受益人资料' },
]
const orgTypeOptions = [
  { value: 'SUBJECT_TYPE_ENTERPRISE', label: '企业' },
  { value: 'SUBJECT_TYPE_INDIVIDUAL', label: '个体工商户' },
  { value: 'SUBJECT_TYPE_INSTITUTIONS_CLONED', label: '事业单位' },
  { value: 'SUBJECT_TYPE_OTHERS', label: '社会组织' },
  { value: 'SUBJECT_TYPE_GOVERNMENT', label: '政府机关' },
  { value: 'SUBJECT_TYPE_MICRO', label: '小微商户或个人卖家' },
]
const idHolderOptions = [
  { value: 'LEGAL', label: '经营者/法人' },
  { value: 'SUPER', label: '经办人' },
]
const idDocTypeOptions = [
  { value: 'IDENTIFICATION_TYPE_IDCARD', label: '中国大陆居民-身份证' },
  { value: 'IDENTIFICATION_TYPE_OVERSEA_PASSPORT', label: '其他国家或地区居民-护照' },
  { value: 'IDENTIFICATION_TYPE_HONGKONG_PASSPORT', label: '中国香港居民-来往内地通行证' },
  { value: 'IDENTIFICATION_TYPE_MACAO_PASSPORT', label: '中国澳门居民-来往内地通行证' },
  { value: 'IDENTIFICATION_TYPE_TAIWAN_PASSPORT', label: '中国台湾居民-来往大陆通行证' },
  { value: 'IDENTIFICATION_TYPE_FOREIGN_RESIDENT', label: '外国人居留证' },
  { value: 'IDENTIFICATION_TYPE_HONGKONG_MACAO_RESIDENT', label: '港澳居民证' },
  { value: 'IDENTIFICATION_TYPE_TAIWAN_RESIDENT', label: '台湾居民证' },
]
const certTypeOptions = [
  { value: 'CERTIFICATE_TYPE_2388', label: '事业单位法人证书' },
  { value: 'CERTIFICATE_TYPE_2389', label: '统一社会信用代码证书' },
  { value: 'CERTIFICATE_TYPE_2394', label: '社会团体法人登记证书' },
  { value: 'CERTIFICATE_TYPE_2395', label: '民办非企业单位登记证书' },
  { value: 'CERTIFICATE_TYPE_2396', label: '基金会法人登记证书' },
  { value: 'CERTIFICATE_TYPE_2520', label: '执业许可证/执业证' },
  { value: 'CERTIFICATE_TYPE_2521', label: '基层群众性自治组织特别法人统一社会信用代码证' },
  { value: 'CERTIFICATE_TYPE_2522', label: '农村集体经济组织登记证' },
  { value: 'CERTIFICATE_TYPE_2399', label: '宗教活动场所登记证' },
  { value: 'CERTIFICATE_TYPE_2400', label: '政府部门下发的其他有效证明文件' },
]
const financeTypeOptions = [
  { value: 'FINANCE_TYPE_BANK_AGENT', label: '银行业' },
  { value: 'FINANCE_TYPE_PAYMENT_AGENT', label: '支付机构' },
  { value: 'FINANCE_TYPE_INSURANCE', label: '保险业' },
  { value: 'FINANCE_TYPE_TRADE_AND_SETTLE', label: '交易及结算类金融机构' },
  { value: 'FINANCE_TYPE_OTHER', label: '其他金融机构' },
]

// ★ SubjectAlterReq 里这几个字段是可选（string | undefined），但表单本地状态始终有值，
//   用 Required 收窄成必选 string，Select 组件的 modelValue 才能对上（不接受 undefined）。
type SubjectAlterForm = SubjectAlterReq &
  Required<Pick<SubjectAlterReq, 'alter_scope' | 'organization_type' | 'cert_type' | 'id_holder_type' | 'id_doc_type' | 'finance_type'>>

const form = reactive<SubjectAlterForm>({
  alter_scope: 'ALTER_SCOPE_FULL',
  organization_type: 'SUBJECT_TYPE_ENTERPRISE',
  finance_institution: false,
  license_number: '',
  license_copy: '',
  business_merchant_name: '',
  legal_person: '',
  company_address: '',
  license_period_begin: '',
  license_period_end: '',
  cert_type: '',
  cert_number: '',
  cert_copy: '',
  cert_merchant_name: '',
  cert_company_address: '',
  cert_legal_person: '',
  cert_period_begin: '',
  cert_period_end: '',
  finance_type: '',
  finance_license_pics: [],
  id_holder_type: 'LEGAL',
  id_doc_type: 'IDENTIFICATION_TYPE_IDCARD',
  authorize_letter_copy: '',
  card_front: '',
  card_back: '',
  card_name: '',
  card_number: '',
  card_address: '',
  card_period_begin: '',
  card_period_end: '',
  as_ubo: false,
  ubo_list: [],
})

const uboOnly = computed(() => form.alter_scope === 'ALTER_SCOPE_UBO')
const isCertSubject = computed(() =>
  ['SUBJECT_TYPE_GOVERNMENT', 'SUBJECT_TYPE_INSTITUTIONS_CLONED', 'SUBJECT_TYPE_OTHERS'].includes(form.organization_type || ''),
)

function addUBO() {
  form.ubo_list = form.ubo_list || []
  form.ubo_list.push({
    id_doc_type: 'IDENTIFICATION_TYPE_IDCARD',
    card_front: '',
    card_back: '',
    card_name: '',
    card_number: '',
    card_address: '',
    period_begin: '',
    period_end: '',
  })
}
function removeUBO(i: number) {
  form.ubo_list?.splice(i, 1)
}

// —— 图片上传（营业执照/登记证书/法人证件/UBO证件），统一走 media_id 换取 ——
const uploading = reactive<Record<string, boolean>>({})
async function pickAndUpload(key: string, assign: (mediaId: string) => void, e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (!/\.(jpe?g|png|bmp)$/i.test(file.name)) {
    toast.error('图片仅支持 JPG/PNG/BMP 格式')
    return
  }
  if (file.size > 2 * 1024 * 1024) {
    toast.error('图片不能超过 2M，请压缩后重试')
    return
  }
  uploading[key] = true
  try {
    const { media_id } = await adminUploadChannelControlMedia(file)
    assign(media_id)
    toast.success('图片已上传')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '图片上传失败')
  } finally {
    uploading[key] = false
  }
}

const submitting = ref(false)
async function submit() {
  if (!props.enrollId || submitting.value) return
  if (!uboOnly.value) {
    if (isCertSubject.value) {
      if (!form.cert_type || !form.cert_copy || !form.cert_number || !form.cert_merchant_name || !form.cert_company_address || !form.cert_legal_person || !form.cert_period_begin || !form.cert_period_end) {
        toast.error('请完整填写登记证书信息')
        return
      }
    } else if (form.organization_type === 'SUBJECT_TYPE_ENTERPRISE' || form.organization_type === 'SUBJECT_TYPE_INDIVIDUAL') {
      if (!form.license_number || !form.license_copy || !form.business_merchant_name || !form.legal_person) {
        toast.error('请完整填写营业执照信息')
        return
      }
    }
    if (form.finance_institution && (!form.finance_type || !form.finance_license_pics?.length)) {
      toast.error('主体为金融机构时需填写金融机构类型及许可证图片')
      return
    }
    if (!form.card_name || !form.card_number) {
      toast.error('请填写法人证件姓名与号码')
      return
    }
  }
  submitting.value = true
  try {
    const res = await adminModifySubjectInfo(props.enrollId, form)
    toast.success(`主体资料变更申请已提交，申请单号 ${res.apply_id}`)
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
  <Modal :model-value="modelValue" title="解脱代办·修改主体资料" width="max-w-2xl" @update:model-value="(v) => emit('update:modelValue', v)">
    <div class="max-h-[70vh] space-y-4 overflow-y-auto pr-1">
      <p class="text-xs leading-relaxed text-muted-foreground">
        代该商户向微信提交主体资料变更申请。字段与微信官方要求逐项一致，提交后由微信审核，审核期间商户经营不受影响。
      </p>

      <div class="grid grid-cols-2 gap-3">
        <div>
          <label class="mb-1 block text-xs text-muted-foreground">变更范围</label>
          <Select v-model="form.alter_scope" :options="scopeOptions" class="w-full" />
        </div>
        <div v-if="!uboOnly">
          <label class="mb-1 block text-xs text-muted-foreground">主体类型</label>
          <Select v-model="form.organization_type" :options="orgTypeOptions" class="w-full" />
        </div>
      </div>

      <template v-if="!uboOnly">
        <!-- 营业执照 / 登记证书 -->
        <div v-if="!isCertSubject" class="space-y-2 bg-muted/40 p-3">
          <div class="text-xs font-medium">营业执照信息</div>
          <div class="grid grid-cols-2 gap-2">
            <input v-model="form.license_number" class="field-input" placeholder="营业执照注册号（必填）" />
            <input v-model="form.business_merchant_name" class="field-input" placeholder="商户名称（营业执照登记名称，必填）" />
            <input v-model="form.legal_person" class="field-input" placeholder="法人姓名（必填）" />
            <input v-model="form.company_address" class="field-input" placeholder="注册地址" />
            <input v-model="form.license_period_begin" class="field-input" placeholder="有效期开始 yyyy-MM-DD" />
            <input v-model="form.license_period_end" class="field-input" placeholder="有效期结束 yyyy-MM-DD 或长期" />
          </div>
          <div class="flex items-center gap-2">
            <span class="text-xs text-muted-foreground">营业执照照片（必填）</span>
            <span v-if="form.license_copy" class="text-xs text-success">已上传 ✓</span>
            <label class="inline-flex cursor-pointer items-center gap-1 border border-dashed border-border px-2 py-1 text-xs text-muted-foreground hover:border-primary hover:text-primary">
              <ImagePlus class="size-3.5" />{{ uploading.license_copy ? '上传中…' : form.license_copy ? '重新上传' : '上传' }}
              <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.license_copy" @change="pickAndUpload('license_copy', (id) => (form.license_copy = id), $event)" />
            </label>
          </div>
        </div>
        <div v-else class="space-y-2 bg-muted/40 p-3">
          <div class="text-xs font-medium">登记证书信息</div>
          <div class="grid grid-cols-2 gap-2">
            <Select v-model="form.cert_type" :options="certTypeOptions" placeholder="证书类型（必填）" class="w-full" />
            <input v-model="form.cert_number" class="field-input" placeholder="证书编号（必填）" />
            <input v-model="form.cert_merchant_name" class="field-input" placeholder="商户名称（必填）" />
            <input v-model="form.cert_legal_person" class="field-input" placeholder="法人姓名（必填）" />
            <input v-model="form.cert_company_address" class="field-input col-span-2" placeholder="注册地址（必填）" />
            <input v-model="form.cert_period_begin" class="field-input" placeholder="有效期开始 yyyy-MM-DD（必填）" />
            <input v-model="form.cert_period_end" class="field-input" placeholder="有效期结束 yyyy-MM-DD 或长期（必填）" />
          </div>
          <div class="flex items-center gap-2">
            <span class="text-xs text-muted-foreground">登记证书照片（必填）</span>
            <label class="inline-flex cursor-pointer items-center gap-1 border border-dashed border-border px-2 py-1 text-xs text-muted-foreground hover:border-primary hover:text-primary">
              <ImagePlus class="size-3.5" />{{ uploading.cert_copy ? '上传中…' : form.cert_copy ? '重新上传' : '上传' }}
              <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.cert_copy" @change="pickAndUpload('cert_copy', (id) => (form.cert_copy = id), $event)" />
            </label>
          </div>
        </div>

        <!-- 金融机构 -->
        <div class="flex items-center gap-2 text-xs">
          <input v-model="form.finance_institution" type="checkbox" class="size-3.5" />
          <span class="text-muted-foreground">该主体是金融机构</span>
        </div>
        <div v-if="form.finance_institution" class="flex items-center gap-2 bg-muted/40 p-3">
          <Select v-model="form.finance_type" :options="financeTypeOptions" placeholder="金融机构类型" class="w-52" />
          <label class="inline-flex cursor-pointer items-center gap-1 border border-dashed border-border px-2 py-1 text-xs text-muted-foreground hover:border-primary hover:text-primary">
            <ImagePlus class="size-3.5" />许可证图片（最多5张）
            <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" @change="pickAndUpload('finance_license', (id) => form.finance_license_pics?.push(id), $event)" />
          </label>
          <span class="text-xs text-muted-foreground">已传 {{ form.finance_license_pics?.length || 0 }} 张</span>
        </div>

        <!-- 法人身份信息 -->
        <div class="space-y-2 bg-muted/40 p-3">
          <div class="text-xs font-medium">法人身份信息</div>
          <div class="grid grid-cols-2 gap-2">
            <Select v-model="form.id_holder_type" :options="idHolderOptions" class="w-full" />
            <Select v-model="form.id_doc_type" :options="idDocTypeOptions" class="w-full" />
            <input v-model="form.card_name" class="field-input" placeholder="证件姓名（必填）" />
            <input v-model="form.card_number" class="field-input" placeholder="证件号码（必填）" />
            <input v-model="form.card_address" class="field-input col-span-2" placeholder="证件居住地址（企业主体必填）" />
            <input v-model="form.card_period_begin" class="field-input" placeholder="证件有效期开始 yyyy-MM-DD" />
            <input v-model="form.card_period_end" class="field-input" placeholder="证件有效期结束 yyyy-MM-DD 或长期" />
          </div>
          <div class="flex items-center gap-3">
            <label class="inline-flex cursor-pointer items-center gap-1 border border-dashed border-border px-2 py-1 text-xs text-muted-foreground hover:border-primary hover:text-primary">
              <ImagePlus class="size-3.5" />{{ uploading.card_front ? '上传中…' : form.card_front ? '正面已传' : '证件正面照' }}
              <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.card_front" @change="pickAndUpload('card_front', (id) => (form.card_front = id), $event)" />
            </label>
            <label class="inline-flex cursor-pointer items-center gap-1 border border-dashed border-border px-2 py-1 text-xs text-muted-foreground hover:border-primary hover:text-primary">
              <ImagePlus class="size-3.5" />{{ uploading.card_back ? '上传中…' : form.card_back ? '反面已传' : '证件反面照' }}
              <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.card_back" @change="pickAndUpload('card_back', (id) => (form.card_back = id), $event)" />
            </label>
            <label v-if="form.id_holder_type === 'SUPER'" class="inline-flex cursor-pointer items-center gap-1 border border-dashed border-border px-2 py-1 text-xs text-muted-foreground hover:border-primary hover:text-primary">
              <ImagePlus class="size-3.5" />{{ uploading.authorize_letter_copy ? '上传中…' : form.authorize_letter_copy ? '说明函已传' : '法定代表人说明函' }}
              <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.authorize_letter_copy" @change="pickAndUpload('authorize_letter_copy', (id) => (form.authorize_letter_copy = id), $event)" />
            </label>
          </div>
          <div v-if="form.organization_type === 'SUBJECT_TYPE_ENTERPRISE'" class="flex items-center gap-2 text-xs">
            <input v-model="form.as_ubo" type="checkbox" class="size-3.5" />
            <span class="text-muted-foreground">经营者/法人是最终受益人</span>
          </div>
        </div>
      </template>

      <!-- 最终受益人列表 -->
      <div class="space-y-2">
        <div class="flex items-center justify-between">
          <div class="text-xs font-medium">最终受益人（UBO，最多4人；企业/社会组织且经营者/法人非受益人时必填）</div>
          <Button size="sm" variant="outline" @click="addUBO"><Plus class="size-3.5" />添加</Button>
        </div>
        <div v-for="(u, i) in form.ubo_list" :key="i" class="space-y-2 bg-muted/40 p-3">
          <div class="flex items-center justify-between">
            <span class="text-xs text-muted-foreground">受益人 {{ i + 1 }}</span>
            <button class="text-muted-foreground hover:text-destructive" @click="removeUBO(i)"><Trash2 class="size-3.5" /></button>
          </div>
          <div class="grid grid-cols-2 gap-2">
            <Select v-model="u.id_doc_type" :options="idDocTypeOptions" class="w-full" />
            <input v-model="u.card_name" class="field-input" placeholder="证件姓名" />
            <input v-model="u.card_number" class="field-input" placeholder="证件号码" />
            <input v-model="u.card_address" class="field-input" placeholder="居住地址" />
            <input v-model="u.period_begin" class="field-input" placeholder="有效期开始 yyyy-MM-DD" />
            <input v-model="u.period_end" class="field-input" placeholder="有效期结束 yyyy-MM-DD 或长期" />
          </div>
          <label class="inline-flex cursor-pointer items-center gap-1 border border-dashed border-border px-2 py-1 text-xs text-muted-foreground hover:border-primary hover:text-primary">
            <ImagePlus class="size-3.5" />{{ u.card_front ? '证件正面已传' : '证件正面照' }}
            <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" @change="pickAndUpload(`ubo_${i}`, (id) => (u.card_front = id), $event)" />
          </label>
        </div>
      </div>
    </div>

    <template #footer>
      <Button variant="outline" :disabled="submitting" @click="emit('update:modelValue', false)">取消</Button>
      <Button :disabled="submitting" @click="submit">提交申请</Button>
    </template>
  </Modal>
</template>
