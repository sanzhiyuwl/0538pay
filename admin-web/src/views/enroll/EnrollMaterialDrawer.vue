<script setup lang="ts">
import { ref, reactive, watch, computed, nextTick } from 'vue'
import { Drawer, Button, Select } from '@/components/ui'
import type { EnrollMaterialReq, EnrollMaterialView } from '@/lib/api/console'
import type { OCRLicenseResult, OCRIDCardResult } from '@/lib/api/ocr'
import { ApiError } from '@/lib/api/client'
import { useToast } from '@/composables/useToast'
import {
  subjectTypeLabels,
  industryGroupNames,
  groupIndustryOptions,
  findIndustryGroup,
  splitIndustryKey,
  findIndustry,
  eligibleActivities,
  bankOptions,
  provinceOptions,
  wxProvinceOptions,
  wxCityOptions,
  wxDistrictOptions,
  wxResolveArea,
} from '@/lib/enrollCatalog'

const toast = useToast()

// 收单同款 Select 选项。主体类型覆盖对照表全部 5 类（行业下拉按主体联动）。
const subjectTypeOptions = Object.entries(subjectTypeLabels).map(([value, label]) => ({ value, label }))
// 账户类型选项随主体类型联动（微信规则，子商户进件 4012719997 bank_account_type）：
// 仅个体户可选「经营者个人银行卡（对私）」；企业/政府/事业单位/社会组织只能「对公银行账户」。
const bankAccountTypeOptions = computed(() => {
  const corporate = { value: 'BANK_ACCOUNT_TYPE_CORPORATE', label: '对公银行账户' }
  if (form.subject_type === 'SUBJECT_TYPE_INDIVIDUAL') {
    return [{ value: 'BANK_ACCOUNT_TYPE_PERSONAL', label: '经营者个人银行卡（对私）' }, corporate]
  }
  return [corporate]
})
// 开户银行选「其他银行」时，需补联行号/支行全称二选一（农信社/农商行等走这里）。
const isOtherBank = computed(() => form.account_bank === '其他银行')

// —— 主体类型分支：政府/事业/社会组织走登记证书；个体户/企业走营业执照 ——
const isCertSubject = computed(
  () =>
    form.subject_type === 'SUBJECT_TYPE_GOVERNMENT' ||
    form.subject_type === 'SUBJECT_TYPE_INSTITUTIONS' ||
    form.subject_type === 'SUBJECT_TYPE_OTHERS',
)
// 企业主体：身份证件需填居住地址。
const isEnterprise = computed(() => form.subject_type === 'SUBJECT_TYPE_ENTERPRISE')
// 企业/社会组织：可填最终受益人 UBO。
const uboEligible = computed(
  () => form.subject_type === 'SUBJECT_TYPE_ENTERPRISE' || form.subject_type === 'SUBJECT_TYPE_OTHERS',
)
// 证件类型：是否身份证（决定走 id_card_info 还是 id_doc_info 那组字段）。
const isIdCard = computed(() => !form.id_doc_type || form.id_doc_type === 'IDENTIFICATION_TYPE_IDCARD')

// 登记证书类型选项（按主体类型过滤，微信 4012719997 cert_type 枚举）。
const certTypeOptions = computed(() => {
  if (form.subject_type === 'SUBJECT_TYPE_INSTITUTIONS') {
    return [{ value: 'CERTIFICATE_TYPE_2388', label: '事业单位法人证书' }]
  }
  if (form.subject_type === 'SUBJECT_TYPE_GOVERNMENT') {
    return [{ value: 'CERTIFICATE_TYPE_2389', label: '统一社会信用代码证书' }]
  }
  // 社会组织
  return [
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
})
// 证件类型选项（政府机关仅身份证；其余主体可选任一）。
const idDocTypeOptions = computed(() => {
  const idcard = { value: 'IDENTIFICATION_TYPE_IDCARD', label: '中国大陆居民-身份证' }
  if (form.subject_type === 'SUBJECT_TYPE_GOVERNMENT') return [idcard]
  return [
    idcard,
    { value: 'IDENTIFICATION_TYPE_OVERSEA_PASSPORT', label: '其他国家或地区居民-护照' },
    { value: 'IDENTIFICATION_TYPE_HONGKONG_PASSPORT', label: '中国香港居民-来往内地通行证' },
    { value: 'IDENTIFICATION_TYPE_MACAO_PASSPORT', label: '中国澳门居民-来往内地通行证' },
    { value: 'IDENTIFICATION_TYPE_TAIWAN_PASSPORT', label: '中国台湾居民-来往大陆通行证' },
    { value: 'IDENTIFICATION_TYPE_FOREIGN_RESIDENT', label: '外国人居留证' },
    { value: 'IDENTIFICATION_TYPE_HONGKONG_MACAO_RESIDENT', label: '港澳居民居住证' },
    { value: 'IDENTIFICATION_TYPE_TAIWAN_RESIDENT', label: '台湾居民居住证' },
  ]
})
// 超管类型：经营者/法人（默认）或经办人（经办人需额外传证件资料）。
const contactTypeOptions = [
  { value: 'LEGAL', label: '经营者/法定代表人' },
  { value: 'SUPER', label: '经办人（需额外证件资料）' },
]
// 经营场景类型多选（勾选驱动对应子表单显示）。
const salesSceneDefs = [
  { value: 'SALES_SCENES_STORE', label: '线下场所' },
  { value: 'SALES_SCENES_MP', label: '服务号/公众号' },
  { value: 'SALES_SCENES_MINI_PROGRAM', label: '小程序' },
  { value: 'SALES_SCENES_WEB', label: '互联网网站' },
  { value: 'SALES_SCENES_APP', label: 'App' },
  { value: 'SALES_SCENES_WEWORK', label: '企业微信' },
]
function toggleScene(v: string) {
  const i = form.sales_scenes_type.indexOf(v)
  if (i >= 0) form.sales_scenes_type.splice(i, 1)
  else form.sales_scenes_type.push(v)
}
const hasScene = (v: string) => form.sales_scenes_type.includes(v)
// UBO 增删。
function addUBO() {
  form.ubo_list.push({
    ubo_id_doc_type: 'IDENTIFICATION_TYPE_IDCARD',
    ubo_id_doc_copy: '',
    ubo_id_doc_copy_back: '',
    ubo_id_doc_name: '',
    ubo_id_doc_number: '',
    ubo_id_doc_address: '',
    ubo_period_begin: '',
    ubo_period_end: '',
  })
}
function removeUBO(idx: number) {
  form.ubo_list.splice(idx, 1)
}

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
  uploadVideoFn: (id: number, file: File) => Promise<{ media_id: string }>
  // OCR 识别（可选）：上传执照/身份证图后自动识别回填。未传则不识别，仅上传换 media_id。
  ocrLicenseFn?: (file: File) => Promise<OCRLicenseResult>
  ocrIdcardFn?: (file: File, side?: 'front' | 'back') => Promise<OCRIDCardResult>
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
  idCardAddress: false,
  idDocName: false,
  idDocNumber: false,
  idDocAddress: false,
})

function blank(): EnrollMaterialReq {
  return {
    subject_type: 'SUBJECT_TYPE_INDIVIDUAL',
    merchant_shortname: '',
    service_phone: '',
    license_number: '',
    license_copy: '',
    business_merchant_name: '',
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
    bank_branch_id: '',
    bank_name: '',
    account_number: '',
    contact_type: 'LEGAL',
    contact_name: '',
    contact_id_number: '',
    mobile_phone: '',
    contact_email: '',
    contact_id_doc_type: 'IDENTIFICATION_TYPE_IDCARD',
    contact_id_doc_copy: '',
    contact_id_doc_copy_back: '',
    contact_period_begin: '',
    contact_period_end: '',
    settlement_id: '',
    qualification_type: '',
    qualifications: [],
    activities_id: '',
    debit_activities_rate: '',
    credit_activities_rate: '',
    activities_additions: [],
    // 登记证书（政府/事业/社会组织）
    cert_type: '',
    cert_copy: '',
    cert_number: '',
    cert_merchant_name: '',
    cert_company_address: '',
    cert_legal_person: '',
    cert_period_begin: '',
    cert_period_end: '',
    cert_letter_copy: '',
    // 身份证件
    id_doc_type: 'IDENTIFICATION_TYPE_IDCARD',
    id_card_address: '',
    id_doc_copy: '',
    id_doc_copy_back: '',
    id_doc_name: '',
    id_doc_number: '',
    id_doc_address: '',
    doc_period_begin: '',
    doc_period_end: '',
    // 经营场景
    sales_scenes_type: [],
    biz_store: { biz_store_name: '', biz_address_code: '', biz_store_address: '', store_entrance_pic: [], indoor_pic: [], biz_sub_appid: '' },
    mp_info: { mp_appid: '', mp_sub_appid: '', mp_pics: [] },
    mini_program: { mini_program_appid: '', mini_program_sub_appid: '', mini_program_pics: [] },
    web_info: { domain: '', web_authorisation: '', web_appid: '' },
    app_info: { app_appid: '', app_sub_appid: '', app_pics: [] },
    wework_info: { sub_corp_id: '', wework_pics: [] },
    // 最终受益人
    ubo_list: [],
    // 补充材料
    legal_person_commitment: '',
    legal_person_video: '',
    business_addition_pics: [],
    business_addition_msg: '',
  }
}
const form = reactive<EnrollMaterialReq>(blank())

// —— 结算规则两级联动（主体类型 → ①行业大类 → ②门店类型，选中自动带出 settlement_id/费率提示）——
// 第一级：行业大类。选中的大类名单独用一个 ref 持有（不进 form，仅用于驱动第二级）。
const industryGroup = ref('')
const industryGroupOptions = computed(() =>
  industryGroupNames(form.subject_type).map((g) => ({ value: g, label: g })),
)
// 第二级：该大类下的门店类型（具体行业）。value 为 settlement_id|qualification_type 复合键。
const storeTypeOptions = computed(() =>
  industryGroup.value ? groupIndustryOptions(form.subject_type, industryGroup.value) : [],
)
// 复合键 getter/setter：读时按 form 两字段拼，写时拆回两字段（并联动清理不再适用的活动）。
const industryKey = computed<string>({
  get: () => (form.settlement_id && form.qualification_type ? `${form.settlement_id}|${form.qualification_type}` : ''),
  set: (key) => {
    const { settlementId, qualificationType } = splitIndustryKey(String(key))
    form.settlement_id = settlementId
    form.qualification_type = qualificationType
    // 换行业后，已选活动若不在新行业白名单内则清空，避免提交非法组合。
    if (form.activities_id && !eligibleActivities(settlementId, qualificationType).some((a) => a.activities_id === form.activities_id)) {
      form.activities_id = ''
      form.debit_activities_rate = ''
      form.credit_activities_rate = ''
      form.activities_additions = []
    }
  },
})
// 切换行业大类时，清掉已选门店类型（换大类后原门店不属于新大类）。回填阶段不清。
watch(industryGroup, () => {
  if (restoring.value) return
  form.settlement_id = ''
  form.qualification_type = ''
})
// 当前选中行业项（回显 fee_desc / 是否需特殊资质提示）。
const selectedIndustry = computed(() =>
  form.settlement_id && form.qualification_type
    ? findIndustry(form.subject_type, form.settlement_id, form.qualification_type)
    : undefined,
)
// 当前行业可报名的优惠费率活动（含「不参与」空项）；不可报名时仅剩空项。
const activitiesOptions = computed(() => {
  const opts = [{ value: '', label: '不参与优惠活动' }]
  for (const a of eligibleActivities(form.settlement_id, form.qualification_type)) {
    opts.push({ value: a.activities_id, label: `${a.name}（借记 ${a.debit_min}%~${a.debit_max}% / 信用 ${a.credit_min}%~${a.credit_max}%）` })
  }
  return opts
})
// 主体类型切换：清空已选行业与活动（不同主体行业不通用）。
watch(
  () => form.subject_type,
  (n, o) => {
    // 非个体户不允许对私账户，切换后强制纠正为对公（回填阶段也要纠，防历史脏数据）。
    if (n !== 'SUBJECT_TYPE_INDIVIDUAL' && form.bank_account_type === 'BANK_ACCOUNT_TYPE_PERSONAL') {
      form.bank_account_type = 'BANK_ACCOUNT_TYPE_CORPORATE'
    }
    if (o === undefined || restoring.value) return // 初始化/回填阶段不清行业
    industryGroup.value = ''
    form.settlement_id = ''
    form.qualification_type = ''
    form.activities_id = ''
    form.debit_activities_rate = ''
    form.credit_activities_rate = ''
    form.activities_additions = []
  },
  { immediate: true },
)

// 补充视频上传状态（声明前置于 loadView，watch immediate 里会用到）。
const videoUploading = ref(false)
const videoMediaId = ref('')
// 回填期间置真，避免 subject_type watcher 把刚回填的结算规则清掉。
const restoring = ref(false)

// —— 门店省市区三级级联 state（微信 biz_address_code 要求区县级 6 位，只有 form.biz_store.biz_address_code 是真填码）——
// storeProvince/storeCity 只驱动上层下拉选项，切换上级时清空下级选中值；回填时由 loadView 反查填入。
const storeProvince = ref('')
const storeCity = ref('')
const storeCityOptions = computed(() => wxCityOptions(storeProvince.value))
const storeDistrictOptions = computed(() => wxDistrictOptions(storeCity.value))
watch(storeProvince, (_n, o) => {
  if (restoring.value || o === undefined) return
  storeCity.value = ''
  form.biz_store.biz_address_code = ''
})
watch(storeCity, (_n, o) => {
  if (restoring.value || o === undefined) return
  form.biz_store.biz_address_code = ''
})

async function loadView(id: number) {
  loading.value = true
  restoring.value = true
  try {
    const v = await props.fetchFn(id)
    Object.assign(form, blank())
    videoMediaId.value = ''
    // 非敏感字段回填
    form.subject_type = v.subject_type || 'SUBJECT_TYPE_INDIVIDUAL'
    form.merchant_shortname = v.merchant_shortname || ''
    form.service_phone = v.service_phone || ''
    form.license_number = v.license_number || ''
    form.license_copy = v.license_copy || ''
    form.business_merchant_name = v.business_merchant_name || ''
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
    form.bank_branch_id = v.bank_branch_id || ''
    form.bank_name = v.bank_name || ''
    // 结算规则（非敏感，原样回填）
    form.settlement_id = v.settlement_id || ''
    form.qualification_type = v.qualification_type || ''
    // 反查行业大类，回填第一级下拉（第二级选项即随之出现，industryKey 命中门店类型）。
    industryGroup.value = findIndustryGroup(form.subject_type, form.settlement_id, form.qualification_type)
    form.qualifications = Array.isArray(v.qualifications) ? [...v.qualifications] : []
    form.activities_id = v.activities_id || ''
    form.debit_activities_rate = v.debit_activities_rate || ''
    form.credit_activities_rate = v.credit_activities_rate || ''
    form.activities_additions = Array.isArray(v.activities_additions) ? [...v.activities_additions] : []
    // 登记证书回填
    form.cert_type = v.cert_type || ''
    form.cert_copy = v.cert_copy || ''
    form.cert_number = v.cert_number || ''
    form.cert_merchant_name = v.cert_merchant_name || ''
    form.cert_company_address = v.cert_company_address || ''
    form.cert_legal_person = v.cert_legal_person || ''
    form.cert_period_begin = v.cert_period_begin || ''
    form.cert_period_end = v.cert_period_end || ''
    form.cert_letter_copy = v.cert_letter_copy || ''
    // 身份证件（非敏感回填；敏感 has_*）
    form.id_doc_type = v.id_doc_type || 'IDENTIFICATION_TYPE_IDCARD'
    form.id_doc_copy = v.id_doc_copy || ''
    form.id_doc_copy_back = v.id_doc_copy_back || ''
    form.doc_period_begin = v.doc_period_begin || ''
    form.doc_period_end = v.doc_period_end || ''
    has.idCardAddress = v.has_id_card_address
    has.idDocName = v.has_id_doc_name
    has.idDocNumber = v.has_id_doc_number
    has.idDocAddress = v.has_id_doc_address
    // 经营场景回填
    form.sales_scenes_type = Array.isArray(v.sales_scenes_type) ? [...v.sales_scenes_type] : []
    if (v.biz_store) form.biz_store = { ...v.biz_store, store_entrance_pic: [...(v.biz_store.store_entrance_pic || [])], indoor_pic: [...(v.biz_store.indoor_pic || [])] }
    // 由已保存的区县级 biz_address_code 反查省/市，回显到三级下拉的上两级。
    {
      const area = wxResolveArea(form.biz_store.biz_address_code)
      storeProvince.value = area.province
      storeCity.value = area.city
    }
    if (v.mp_info) form.mp_info = { ...v.mp_info, mp_pics: [...(v.mp_info.mp_pics || [])] }
    if (v.mini_program) form.mini_program = { ...v.mini_program, mini_program_pics: [...(v.mini_program.mini_program_pics || [])] }
    if (v.web_info) form.web_info = { ...v.web_info }
    if (v.app_info) form.app_info = { ...v.app_info, app_pics: [...(v.app_info.app_pics || [])] }
    if (v.wework_info) form.wework_info = { ...v.wework_info, wework_pics: [...(v.wework_info.wework_pics || [])] }
    // UBO 回填（敏感字段不回原文，仅结构与非敏感；姓名/号码/地址留空表示不改）
    form.ubo_list = Array.isArray(v.ubo_list)
      ? v.ubo_list.map((u) => ({
          ubo_id_doc_type: u.ubo_id_doc_type || 'IDENTIFICATION_TYPE_IDCARD',
          ubo_id_doc_copy: u.ubo_id_doc_copy || '',
          ubo_id_doc_copy_back: u.ubo_id_doc_copy_back || '',
          ubo_id_doc_name: '',
          ubo_id_doc_number: '',
          ubo_id_doc_address: '',
          ubo_period_begin: u.ubo_period_begin || '',
          ubo_period_end: u.ubo_period_end || '',
        }))
      : []
    // 补充材料回填
    form.legal_person_commitment = v.legal_person_commitment || ''
    form.legal_person_video = v.legal_person_video || ''
    form.business_addition_pics = Array.isArray(v.business_addition_pics) ? [...v.business_addition_pics] : []
    form.business_addition_msg = v.business_addition_msg || ''
    if (v.legal_person_video) videoMediaId.value = v.legal_person_video
    // 敏感字段只回是否已填（不回原文）
    has.idCardName = v.has_id_card_name
    has.idCardNumber = v.has_id_card_number
    has.accountName = v.has_account_name
    has.accountNumber = v.has_account_number
    has.contactName = v.has_contact_name
    has.contactIdNumber = v.has_contact_id_number
    has.mobilePhone = v.has_mobile_phone
    has.contactEmail = v.has_contact_email
    // 超管类型 + 经办人证件（非敏感回填）
    form.contact_type = v.contact_type || 'LEGAL'
    form.contact_id_doc_type = v.contact_id_doc_type || 'IDENTIFICATION_TYPE_IDCARD'
    form.contact_id_doc_copy = v.contact_id_doc_copy || ''
    form.contact_id_doc_copy_back = v.contact_id_doc_copy_back || ''
    form.contact_period_begin = v.contact_period_begin || ''
    form.contact_period_end = v.contact_period_end || ''
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '加载资料失败')
  } finally {
    loading.value = false
    // 等 subject_type 变更的 watch 队列冲刷完再解锁，避免误清刚回填的结算规则。
    nextTick(() => (restoring.value = false))
  }
}

// —— 草稿自动保存（防误关丢数据）——
// 抽屉点遮罩即关（通用 Drawer 行为），未提交的填料会丢。这里把整份 form + 敏感标记 + 门店级联/视频态
// 按进件单 id 存 localStorage，打开时若有比后端更新的草稿则恢复。成功提交后清该单草稿。
// 注意：含敏感字段明文（用户已确认此取舍），仅暂存本机浏览器，提交后即清除。
const DRAFT_PREFIX = 'enroll_material_draft_'
const draftSavedAt = ref('') // 最近一次草稿保存时间（顶部提示用）
let draftDebounce: number | undefined
// 抑制草稿写入：loadView/恢复期间置真，避免把刚灌入的数据又当"用户编辑"写回。
const suppressDraft = ref(false)

function draftKey(id: number | null): string {
  return DRAFT_PREFIX + (id ?? 'new')
}

// 打包当前编辑态（form + has 敏感标记 + 门店级联 + 视频 media）。
function snapshotDraft() {
  return {
    savedAt: new Date().toISOString(),
    form: JSON.parse(JSON.stringify(form)),
    has: JSON.parse(JSON.stringify(has)),
    storeProvince: storeProvince.value,
    storeCity: storeCity.value,
    videoMediaId: videoMediaId.value,
  }
}

function writeDraft() {
  if (suppressDraft.value) return
  try {
    const snap = snapshotDraft()
    localStorage.setItem(draftKey(props.enrollId), JSON.stringify(snap))
    draftSavedAt.value = snap.savedAt
  } catch {
    /* localStorage 写满/隐私模式等，静默忽略，不打断填写 */
  }
}

function readDraft(id: number | null): ReturnType<typeof snapshotDraft> | null {
  try {
    const raw = localStorage.getItem(draftKey(id))
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

function clearDraft(id: number | null) {
  try {
    localStorage.removeItem(draftKey(id))
  } catch {
    /* 忽略 */
  }
  draftSavedAt.value = ''
}

// 把草稿快照灌回响应式状态（恢复期间抑制草稿写入与联动清值）。
function applyDraft(snap: ReturnType<typeof snapshotDraft>) {
  suppressDraft.value = true
  restoring.value = true
  Object.assign(form, snap.form)
  Object.assign(has, snap.has)
  storeProvince.value = snap.storeProvince || ''
  storeCity.value = snap.storeCity || ''
  videoMediaId.value = snap.videoMediaId || ''
  // 恢复行业大类联动（settlement_id/qualification_type 已在 form 里）。
  industryGroup.value = findIndustryGroup(form.subject_type, form.settlement_id, form.qualification_type)
  draftSavedAt.value = snap.savedAt
  nextTick(() => {
    restoring.value = false
    suppressDraft.value = false
  })
}

// 手动清除草稿并回到后端已保存态。
async function discardDraft() {
  clearDraft(props.enrollId)
  if (props.enrollId) await loadView(props.enrollId)
}

// form/has/门店级联/视频 变化时防抖写草稿。
watch(
  [() => form, () => has, storeProvince, storeCity, videoMediaId],
  () => {
    if (suppressDraft.value || !props.modelValue) return
    if (draftDebounce) window.clearTimeout(draftDebounce)
    draftDebounce = window.setTimeout(writeDraft, 600)
  },
  { deep: true },
)

// 打开抽屉：先拉后端已保存资料，再检查本地是否有更新的草稿并提示恢复。
watch(
  () => [props.modelValue, props.enrollId] as const,
  async ([open, id]) => {
    if (!open) return
    if (id) await loadView(id)
    const snap = readDraft(id)
    if (snap) applyDraft(snap)
  },
  { immediate: true },
)

const draftSavedText = computed(() => {
  if (!draftSavedAt.value) return ''
  const d = new Date(draftSavedAt.value)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`
})

const sensitivePlaceholder = (filled: boolean) => (filled ? '已填（如需修改请重新输入原文）' : '必填')

// —— 图片上传（营业执照/身份证正反面）——
// 三个 media_id 字段各自的上传中状态，按 form 字段名区分。
const uploading = reactive<Record<string, boolean>>({
  license_copy: false,
  id_card_copy: false,
  id_card_national: false,
  qualifications: false,
  activities_additions: false,
})

// OCR 识别中状态（按字段区分，用于按钮提示）。
const recognizing = reactive<Record<string, boolean>>({
  license_copy: false,
  id_card_copy: false,
  id_card_national: false,
})

// 执照识别回填：仅回填空字段/覆盖同类，日期归一由后端完成；识别值供人工核对后提交。
function fillFromLicense(r: OCRLicenseResult) {
  if (r.reg_number) form.license_number = r.reg_number
  if (r.name) form.business_merchant_name = r.name
  if (r.legal_person) form.legal_person = r.legal_person
  if (r.address) form.license_address = r.address
  if (r.valid_period_begin) form.period_begin = r.valid_period_begin
  if (r.valid_period_end) form.period_end = r.valid_period_end
}

// 身份证人像面回填（姓名/号码/住址为敏感字段，识别后填入输入框由人核对）。
function fillFromIDCardFront(r: OCRIDCardResult) {
  if (r.name) {
    form.id_card_name = r.name
    has.idCardName = false // 识别出新值，按未填态提示需核对
  }
  if (r.id_number) {
    form.id_card_number = r.id_number
    has.idCardNumber = false
  }
  if (isEnterprise.value && r.address) {
    form.id_card_address = r.address
    has.idCardAddress = false
  }
}

// 身份证国徽面回填（有效期）。
function fillFromIDCardBack(r: OCRIDCardResult) {
  if (r.valid_period_begin) form.card_period_begin = r.valid_period_begin
  if (r.valid_period_end) form.card_period_end = r.valid_period_end
}

// 选图即上传：校验类型/大小 → uploadFn 换 media_id → 回填对应字段。
// 若父组件传入 OCR 识别函数，则同时识别并回填结构化字段（识别失败不阻断上传）。
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
    // 上传成功后按字段类型尝试 OCR 识别回填（不影响已换到的 media_id）。
    await runOCR(field, file)
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '图片上传失败')
  } finally {
    uploading[field] = false
    input.value = '' // 允许重选同名文件
  }
}

// 按字段调用对应 OCR 识别并回填。识别失败仅提示、不影响上传结果。
async function runOCR(field: 'license_copy' | 'id_card_copy' | 'id_card_national', file: File) {
  try {
    if (field === 'license_copy' && props.ocrLicenseFn) {
      recognizing[field] = true
      fillFromLicense(await props.ocrLicenseFn(file))
      toast.success('已识别营业执照并回填，请核对')
    } else if (field === 'id_card_copy' && props.ocrIdcardFn) {
      recognizing[field] = true
      fillFromIDCardFront(await props.ocrIdcardFn(file, 'front'))
      toast.success('已识别身份证并回填，请核对')
    } else if (field === 'id_card_national' && props.ocrIdcardFn) {
      recognizing[field] = true
      fillFromIDCardBack(await props.ocrIdcardFn(file, 'back'))
      toast.success('已识别身份证有效期，请核对')
    }
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : 'OCR 识别失败，请手动填写')
  } finally {
    recognizing[field] = false
  }
}

// —— 多图上传（特殊资质 / 优惠活动补充材料，各 ≤5 张 media_id）——
// 选图即上传，成功 push 进对应数组字段，超 5 张拦截。
async function onPickMultiImage(field: 'qualifications' | 'activities_additions', e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (form[field].length >= 5) {
    toast.error('最多上传 5 张')
    input.value = ''
    return
  }
  if (!/\.(jpe?g|png|bmp)$/i.test(file.name)) {
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
    form[field].push(media_id)
    toast.success('图片已上传')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '图片上传失败')
  } finally {
    uploading[field] = false
    input.value = ''
  }
}
function removeMultiImage(field: 'qualifications' | 'activities_additions', idx: number) {
  form[field].splice(idx, 1)
}

// —— 通用「上传到指定数组」（经营场景截图 / 补充材料等嵌套数组字段用）——
// key 仅用于 uploading 状态区分；arr 是目标数组（reactive 引用，push 生效）。
async function onPickArrayImage(key: string, arr: string[], e: Event, max = 5) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (arr.length >= max) {
    toast.error(`最多上传 ${max} 张`)
    input.value = ''
    return
  }
  if (!/\.(jpe?g|png|bmp)$/i.test(file.name)) {
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
  uploading[key] = true
  try {
    const { media_id } = await props.uploadFn(props.enrollId, file)
    arr.push(media_id)
    toast.success('图片已上传')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '图片上传失败')
  } finally {
    uploading[key] = false
    input.value = ''
  }
}
// 单图上传到任意 media_id 字段（登记证书照/单位证明函/证件正反面等），回调回填。
async function onPickSingleTo(key: string, assign: (id: string) => void, e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (!/\.(jpe?g|png|bmp)$/i.test(file.name)) {
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
  uploading[key] = true
  try {
    const { media_id } = await props.uploadFn(props.enrollId, file)
    assign(media_id)
    toast.success('图片已上传')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '图片上传失败')
  } finally {
    uploading[key] = false
    input.value = ''
  }
}

// —— 补充视频上传（部分指定行业进件时微信要求补充；非 applyment4sub 核心字段，
//    上传换 media_id 后由运营按微信要求手动放入对应资料项，故这里只展示 media_id 供复制）——
async function onPickVideo(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  const okExt = /\.(avi|wmv|mpeg|mp4|mov|mkv|flv|f4v|m4v|rmvb)$/i.test(file.name)
  if (!okExt) {
    toast.error('视频仅支持 avi/wmv/mpeg/mp4/mov/mkv/flv/f4v/m4v/rmvb 格式')
    input.value = ''
    return
  }
  if (file.size > 5 * 1024 * 1024) {
    toast.error('视频不能超过 5M，请压缩后重试')
    input.value = ''
    return
  }
  if (!props.enrollId) return
  videoUploading.value = true
  try {
    const { media_id } = await props.uploadVideoFn(props.enrollId, file)
    videoMediaId.value = media_id
    form.legal_person_video = media_id // 组装进 addition_info.legal_person_video
    toast.success('视频已上传')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '视频上传失败')
  } finally {
    videoUploading.value = false
    input.value = ''
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
  // 结算规则：settlement_id/qualification_type 微信必填；报名活动则两费率必填。
  if (!form.settlement_id.trim()) {
    toast.error('请填写入驻结算规则ID')
    return
  }
  if (!form.qualification_type.trim()) {
    toast.error('请填写所属行业')
    return
  }
  if (form.activities_id.trim() && (!form.debit_activities_rate.trim() || !form.credit_activities_rate.trim())) {
    toast.error('报名优惠费率活动时，借记卡与信用卡费率均需填写')
    return
  }
  // 开户银行为「其他银行」时，联行号/支行全称二选一（微信规则）。
  if (isOtherBank.value && !form.bank_branch_id.trim() && !form.bank_name.trim()) {
    toast.error('开户银行为「其他银行」时，需填写联行号或支行全称二选一')
    return
  }
  // 主体资料：证书主体必填登记证书；执照主体必填执照。
  if (isCertSubject.value) {
    if (!form.cert_type.trim() || !form.cert_copy.trim() || !form.cert_number.trim() || !form.cert_merchant_name.trim() ||
        !form.cert_company_address.trim() || !form.cert_legal_person.trim() || !form.cert_period_begin.trim() || !form.cert_period_end.trim()) {
      toast.error('政府机关/事业单位/社会组织需完整填写登记证书信息')
      return
    }
  } else if (!form.license_number.trim() || !form.license_copy.trim()) {
    toast.error('请填写营业执照信息（证照编号与执照照片）')
    return
  } else if (!form.business_merchant_name.trim()) {
    toast.error('请填写营业执照上的商户名称（营业执照登记名称，非法人姓名）')
    return
  } else if (form.business_merchant_name.trim().length < 2 || form.business_merchant_name.trim().length > 128) {
    toast.error('商户名称长度需为 2-128 个字符')
    return
  }
  // 超管为经办人（SUPER）时，证件类型/正面照/证件号/有效期必填。
  if (form.contact_type === 'SUPER') {
    const idNumOk = form.contact_id_number.trim() !== '' || has.contactIdNumber
    if (!form.contact_id_doc_type.trim() || !form.contact_id_doc_copy.trim() || !idNumOk ||
        !form.contact_period_begin.trim() || !form.contact_period_end.trim()) {
      toast.error('超管为经办人时，需填经办人证件类型、正面照、证件号码与有效期')
      return
    }
  }
  // 企业主体：身份证件需填居住地址。
  if (isEnterprise.value) {
    if (isIdCard.value && !form.id_card_address.trim() && !has.idCardAddress) {
      toast.error('企业主体需填写法定代表人身份证居住地址')
      return
    }
    if (!isIdCard.value && !form.id_doc_address.trim() && !has.idDocAddress) {
      toast.error('企业主体需填写法定代表人证件居住地址')
      return
    }
  }
  // 经营场景：至少勾一类，勾了对应场景关键字段必填。
  if (form.sales_scenes_type.length === 0) {
    toast.error('请至少选择一类经营场景')
    return
  }
  if (hasScene('SALES_SCENES_STORE')) {
    const s = form.biz_store
    if (!s.biz_store_name.trim() || !s.biz_address_code.trim() || !s.biz_store_address.trim() || s.store_entrance_pic.length === 0 || s.indoor_pic.length === 0) {
      toast.error('「线下场所」需填门店名称、省市编码、地址并上传门头照与内部照')
      return
    }
  }
  if (hasScene('SALES_SCENES_MP')) {
    if (!form.mp_info.mp_appid.trim() && !form.mp_info.mp_sub_appid.trim()) {
      toast.error('「服务号/公众号」需填服务商或商家公众号 AppID（二选一）')
      return
    }
    if (form.mp_info.mp_pics.length === 0) {
      toast.error('「服务号/公众号」需上传页面截图')
      return
    }
  }
  if (hasScene('SALES_SCENES_MINI_PROGRAM') && !form.mini_program.mini_program_appid.trim() && !form.mini_program.mini_program_sub_appid.trim()) {
    toast.error('「小程序」需填服务商或商家小程序 AppID（二选一）')
    return
  }
  if (hasScene('SALES_SCENES_WEB') && !form.web_info.domain.trim()) {
    toast.error('「互联网网站」需填网站域名')
    return
  }
  if (hasScene('SALES_SCENES_APP')) {
    if (!form.app_info.app_appid.trim() && !form.app_info.app_sub_appid.trim()) {
      toast.error('「App」需填服务商或商家应用 AppID（二选一）')
      return
    }
    if (form.app_info.app_pics.length === 0) {
      toast.error('「App」需上传截图（首页/尾页/应用内/支付页）')
      return
    }
  }
  if (hasScene('SALES_SCENES_WEWORK')) {
    if (!form.wework_info.sub_corp_id.trim()) {
      toast.error('「企业微信」需填商家企业微信 CorpID')
      return
    }
    if (form.wework_info.wework_pics.length === 0) {
      toast.error('「企业微信」需上传页面截图')
      return
    }
  }

  saving.value = true
  try {
    await props.submitFn(props.enrollId, { ...form })
    clearDraft(props.enrollId) // 提交成功，清掉本地草稿
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
    width="max-w-4xl"
    title="填全套资料"
    :subtitle="merchantName ? `${merchantName} · 提交微信审核前的完整进件资料` : '提交微信审核前的完整进件资料'"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div v-if="loading" class="py-10 text-center dim">加载中…</div>
    <div v-else class="space-y-5">
      <!-- 草稿自动保存提示：填料实时暂存本地，误关抽屉不丢；提交成功后自动清除 -->
      <div v-if="draftSavedText" class="flex items-center gap-2 bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
        <span>已自动保存草稿（{{ draftSavedText }}）· 误关抽屉后重新打开可恢复，敏感信息仅暂存本机</span>
        <span class="flex-1" />
        <button type="button" class="text-destructive hover:underline" @click="discardDraft">清除草稿</button>
      </div>

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

      <!-- 营业执照（个体户/企业）-->
      <section v-if="!isCertSubject" class="space-y-3">
        <h4 class="text-sm font-medium">营业执照</h4>
        <div class="row-field">
          <label class="lbl">证照编号<span class="text-destructive">*</span></label>
          <input v-model="form.license_number" placeholder="统一社会信用代码" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">商户名称<span class="text-destructive">*</span></label>
          <input v-model="form.business_merchant_name" placeholder="营业执照登记名称（非法人姓名，个体户按「个体户+经营者名」）" class="field-input flex-1" />
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
          <label class="lbl">执照照片<span class="text-destructive">*</span></label>
          <div class="flex flex-1 items-center gap-2">
            <label class="media-btn">
              <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.license_copy || recognizing.license_copy" @change="onPickImage('license_copy', $event)" />
              {{ uploading.license_copy ? '上传中…' : recognizing.license_copy ? '识别中…' : form.license_copy ? '重新上传' : (ocrLicenseFn ? '选择图片并识别' : '选择图片') }}
            </label>
            <span v-if="form.license_copy" class="media-ok">已上传 ✓</span>
            <span v-else class="dim text-xs">JPG/PNG/BMP，≤2M</span>
          </div>
        </div>
      </section>

      <!-- 登记证书（政府机关/事业单位/社会组织）-->
      <section v-if="isCertSubject" class="space-y-3">
        <h4 class="text-sm font-medium">登记证书 <span class="text-[11px] text-muted-foreground">（政府机关/事业单位/社会组织，替代营业执照）</span></h4>
        <div class="row-field">
          <label class="lbl">证书类型<span class="text-destructive">*</span></label>
          <Select v-model="form.cert_type" :options="certTypeOptions" class="flex-1" placeholder="选择登记证书类型" />
        </div>
        <div class="row-field">
          <label class="lbl">证书号<span class="text-destructive">*</span></label>
          <input v-model="form.cert_number" placeholder="与证书类型匹配的证书号" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">商户名称<span class="text-destructive">*</span></label>
          <input v-model="form.cert_merchant_name" placeholder="登记证书上的单位名称" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">注册地址<span class="text-destructive">*</span></label>
          <input v-model="form.cert_company_address" placeholder="登记证书注册地址" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">法定代表人<span class="text-destructive">*</span></label>
          <input v-model="form.cert_legal_person" placeholder="登记证书上的法定代表人" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">证书有效期<span class="text-destructive">*</span></label>
          <div class="flex flex-1 items-center gap-2">
            <input v-model="form.cert_period_begin" placeholder="开始 yyyy-MM-dd" class="field-input flex-1" />
            <span class="dim">至</span>
            <input v-model="form.cert_period_end" placeholder="结束（长期填 长期）" class="field-input flex-1" />
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">证书照片<span class="text-destructive">*</span></label>
          <div class="flex flex-1 items-center gap-2">
            <label class="media-btn">
              <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.cert_copy" @change="onPickSingleTo('cert_copy', (id) => (form.cert_copy = id), $event)" />
              {{ uploading.cert_copy ? '上传中…' : form.cert_copy ? '重新上传' : '选择图片' }}
            </label>
            <span v-if="form.cert_copy" class="media-ok">已上传 ✓</span>
            <span v-else class="dim text-xs">JPG/PNG/BMP，≤2M</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">单位证明函</label>
          <div class="flex flex-1 items-center gap-2">
            <label class="media-btn">
              <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.cert_letter_copy" @change="onPickSingleTo('cert_letter_copy', (id) => (form.cert_letter_copy = id), $event)" />
              {{ uploading.cert_letter_copy ? '上传中…' : form.cert_letter_copy ? '重新上传' : '选择图片' }}
            </label>
            <span v-if="form.cert_letter_copy" class="media-ok">已上传 ✓</span>
            <span v-else class="dim text-xs">政府/事业选传，传了免汇款验证</span>
          </div>
        </div>
      </section>

      <!-- 经营者/法人身份（敏感）-->
      <section class="space-y-3">
        <h4 class="text-sm font-medium">经营者/法人身份 <span class="text-[11px] text-muted-foreground">（敏感，加密存储）</span></h4>
        <div class="row-field">
          <label class="lbl">证件类型</label>
          <Select v-model="form.id_doc_type" :options="idDocTypeOptions" class="flex-1" />
        </div>

        <!-- 身份证 -->
        <template v-if="isIdCard">
          <div class="row-field">
            <label class="lbl">证件姓名<span class="text-destructive">*</span></label>
            <input v-model="form.id_card_name" :placeholder="sensitivePlaceholder(has.idCardName)" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">证件号码<span class="text-destructive">*</span></label>
            <input v-model="form.id_card_number" :placeholder="sensitivePlaceholder(has.idCardNumber)" class="field-input flex-1" />
          </div>
          <div v-if="isEnterprise" class="row-field">
            <label class="lbl">居住地址<span class="text-destructive">*</span></label>
            <input v-model="form.id_card_address" :placeholder="sensitivePlaceholder(has.idCardAddress)" class="field-input flex-1" />
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
                <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.id_card_copy || recognizing.id_card_copy" @change="onPickImage('id_card_copy', $event)" />
                {{ uploading.id_card_copy ? '上传中…' : recognizing.id_card_copy ? '识别中…' : form.id_card_copy ? '重新上传' : (ocrIdcardFn ? '选择图片并识别' : '选择图片') }}
              </label>
              <span v-if="form.id_card_copy" class="media-ok">已上传 ✓</span>
              <span v-else class="dim text-xs">JPG/PNG/BMP，≤2M</span>
            </div>
          </div>
          <div class="row-field">
            <label class="lbl">国徽面照片</label>
            <div class="flex flex-1 items-center gap-2">
              <label class="media-btn">
                <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.id_card_national || recognizing.id_card_national" @change="onPickImage('id_card_national', $event)" />
                {{ uploading.id_card_national ? '上传中…' : recognizing.id_card_national ? '识别中…' : form.id_card_national ? '重新上传' : (ocrIdcardFn ? '选择图片并识别' : '选择图片') }}
              </label>
              <span v-if="form.id_card_national" class="media-ok">已上传 ✓</span>
              <span v-else class="dim text-xs">JPG/PNG/BMP，≤2M</span>
            </div>
          </div>
        </template>

        <!-- 其他证件（护照/通行证/居留证等）-->
        <template v-else>
          <div class="row-field">
            <label class="lbl">证件姓名<span class="text-destructive">*</span></label>
            <input v-model="form.id_doc_name" :placeholder="sensitivePlaceholder(has.idDocName)" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">证件号码<span class="text-destructive">*</span></label>
            <input v-model="form.id_doc_number" :placeholder="sensitivePlaceholder(has.idDocNumber)" class="field-input flex-1" />
          </div>
          <div v-if="isEnterprise" class="row-field">
            <label class="lbl">居住地址<span class="text-destructive">*</span></label>
            <input v-model="form.id_doc_address" :placeholder="sensitivePlaceholder(has.idDocAddress)" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">证件有效期</label>
            <div class="flex flex-1 items-center gap-2">
              <input v-model="form.doc_period_begin" placeholder="开始 yyyy-MM-dd" class="field-input flex-1" />
              <span class="dim">至</span>
              <input v-model="form.doc_period_end" placeholder="结束（长期填 长期）" class="field-input flex-1" />
            </div>
          </div>
          <div class="row-field">
            <label class="lbl">证件正面照</label>
            <div class="flex flex-1 items-center gap-2">
              <label class="media-btn">
                <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.id_doc_copy" @change="onPickSingleTo('id_doc_copy', (id) => (form.id_doc_copy = id), $event)" />
                {{ uploading.id_doc_copy ? '上传中…' : form.id_doc_copy ? '重新上传' : '选择图片' }}
              </label>
              <span v-if="form.id_doc_copy" class="media-ok">已上传 ✓</span>
              <span v-else class="dim text-xs">JPG/PNG/BMP，≤2M</span>
            </div>
          </div>
          <div class="row-field">
            <label class="lbl">证件反面照</label>
            <div class="flex flex-1 items-center gap-2">
              <label class="media-btn">
                <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.id_doc_copy_back" @change="onPickSingleTo('id_doc_copy_back', (id) => (form.id_doc_copy_back = id), $event)" />
                {{ uploading.id_doc_copy_back ? '上传中…' : form.id_doc_copy_back ? '重新上传' : '选择图片' }}
              </label>
              <span v-if="form.id_doc_copy_back" class="media-ok">已上传 ✓</span>
              <span v-else class="dim text-xs">护照无需反面</span>
            </div>
          </div>
        </template>
      </section>

      <!-- 最终受益人 UBO（企业/社会组织）-->
      <section v-if="uboEligible" class="space-y-3">
        <h4 class="text-sm font-medium">
          最终受益人 <span class="text-[11px] text-muted-foreground">（企业/社会组织；经营者非唯一受益人时填，留空则微信自动回填经营者，≤4 人）</span>
        </h4>
        <div v-for="(u, idx) in form.ubo_list" :key="idx" class="space-y-3 bg-muted/40 p-3">
          <div class="flex items-center justify-between">
            <span class="text-xs font-medium">受益人 {{ idx + 1 }} <span class="text-[11px] text-muted-foreground">（姓名/号码/地址敏感，加密存储）</span></span>
            <button type="button" class="text-xs text-destructive" @click="removeUBO(idx)">删除</button>
          </div>
          <div class="row-field">
            <label class="lbl">证件类型</label>
            <Select v-model="u.ubo_id_doc_type" :options="idDocTypeOptions" class="flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">证件姓名<span class="text-destructive">*</span></label>
            <input v-model="u.ubo_id_doc_name" placeholder="必填（如需修改请重新输入原文）" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">证件号码<span class="text-destructive">*</span></label>
            <input v-model="u.ubo_id_doc_number" placeholder="必填（如需修改请重新输入原文）" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">居住地址<span class="text-destructive">*</span></label>
            <input v-model="u.ubo_id_doc_address" placeholder="必填（如需修改请重新输入原文）" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">证件有效期</label>
            <div class="flex flex-1 items-center gap-2">
              <input v-model="u.ubo_period_begin" placeholder="开始 yyyy-MM-dd" class="field-input flex-1" />
              <span class="dim">至</span>
              <input v-model="u.ubo_period_end" placeholder="结束（长期填 长期）" class="field-input flex-1" />
            </div>
          </div>
          <div class="row-field">
            <label class="lbl">证件正面照</label>
            <div class="flex flex-1 items-center gap-2">
              <label class="media-btn">
                <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading[`ubo_copy_${idx}`]" @change="onPickSingleTo(`ubo_copy_${idx}`, (id) => (u.ubo_id_doc_copy = id), $event)" />
                {{ uploading[`ubo_copy_${idx}`] ? '上传中…' : u.ubo_id_doc_copy ? '重新上传' : '选择图片' }}
              </label>
              <span v-if="u.ubo_id_doc_copy" class="media-ok">已上传 ✓</span>
              <span v-else class="dim text-xs">JPG/PNG/BMP，≤2M</span>
            </div>
          </div>
          <div class="row-field">
            <label class="lbl">证件反面照</label>
            <div class="flex flex-1 items-center gap-2">
              <label class="media-btn">
                <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading[`ubo_copyback_${idx}`]" @change="onPickSingleTo(`ubo_copyback_${idx}`, (id) => (u.ubo_id_doc_copy_back = id), $event)" />
                {{ uploading[`ubo_copyback_${idx}`] ? '上传中…' : u.ubo_id_doc_copy_back ? '重新上传' : '选择图片' }}
              </label>
              <span v-if="u.ubo_id_doc_copy_back" class="media-ok">已上传 ✓</span>
              <span v-else class="dim text-xs">护照无需反面</span>
            </div>
          </div>
        </div>
        <div v-if="form.ubo_list.length < 4">
          <Button variant="outline" size="sm" @click="addUBO">+ 添加受益人</Button>
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
          <Select v-model="form.account_bank" :options="bankOptions" searchable placeholder="选择开户银行" class="flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">开户省份</label>
          <div class="flex-1 space-y-1">
            <Select v-model="form.bank_address_code" :options="provinceOptions" searchable placeholder="选择开户银行所在省份" class="w-full" />
            <p class="text-[11px] text-muted-foreground">该字段微信即将下线、非必填，精确到省即可。</p>
          </div>
        </div>
        <template v-if="isOtherBank">
          <div class="row-field">
            <label class="lbl">联行号</label>
            <div class="flex-1 space-y-1">
              <input v-model="form.bank_branch_id" placeholder="开户银行联行号（12 位数字）" class="field-input w-full" />
              <p class="text-[11px] text-muted-foreground">选「其他银行」（如农村信用社/农商行）时，联行号与支行全称二选一填写。</p>
            </div>
          </div>
          <div class="row-field">
            <label class="lbl">支行全称</label>
            <input v-model="form.bank_name" placeholder="开户银行全称（含支行），如 XX农村信用合作联社XX信用社" class="field-input flex-1" />
          </div>
        </template>
      </section>

      <!-- 结算规则 settlement_info（行业/费率，微信必填）-->
      <section class="space-y-3">
        <h4 class="text-sm font-medium">结算规则 <span class="text-[11px] text-muted-foreground">（行业与费率，微信必填；按主体类型联动选行业，自动带出结算规则ID与费率）</span></h4>
        <div class="row-field">
          <label class="lbl">行业大类<span class="text-destructive">*</span></label>
          <Select
            v-model="industryGroup"
            :options="industryGroupOptions"
            searchable
            placeholder="先选主体类型，再选行业大类"
            class="flex-1"
          />
        </div>
        <div class="row-field">
          <label class="lbl">门店类型<span class="text-destructive">*</span></label>
          <Select
            v-model="industryKey"
            :options="storeTypeOptions"
            searchable
            :placeholder="industryGroup ? '选择具体门店/经营类型' : '请先选行业大类'"
            class="flex-1"
          />
        </div>
        <div v-if="selectedIndustry" class="row-field">
          <label class="lbl">规则说明</label>
          <div class="flex-1 space-y-0.5 text-xs text-muted-foreground">
            <p>结算规则ID <span class="font-mono text-foreground">{{ form.settlement_id }}</span> · {{ selectedIndustry.fee_desc }}</p>
            <p v-if="selectedIndustry.need_qual" class="text-destructive">
              该行业需特殊资质：{{ selectedIndustry.qual_text }}（请在下方上传）
            </p>
            <p v-if="selectedIndustry.scope">经营范围：{{ selectedIndustry.scope }}</p>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">特殊资质</label>
          <div class="flex flex-1 flex-wrap items-center gap-2">
            <span v-for="(m, i) in form.qualifications" :key="i" :title="m" class="media-ok flex items-center gap-1">
              资质{{ i + 1 }} ✓
              <button type="button" class="text-destructive" @click="removeMultiImage('qualifications', i)">×</button>
            </span>
            <label v-if="form.qualifications.length < 5" class="media-btn">
              <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.qualifications" @change="onPickMultiImage('qualifications', $event)" />
              {{ uploading.qualifications ? '上传中…' : '选择图片' }}
            </label>
            <span class="dim text-xs">需特殊资质的行业才填，≤5 张</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">优惠费率活动</label>
          <Select v-model="form.activities_id" :options="activitiesOptions" class="flex-1" />
        </div>
        <template v-if="form.activities_id">
          <div class="row-field">
            <label class="lbl">借记卡费率<span class="text-destructive">*</span></label>
            <div class="flex flex-1 items-center gap-2">
              <input v-model="form.debit_activities_rate" placeholder="如 0.2" class="field-input flex-1" />
              <span class="dim text-xs">%（须在活动区间内）</span>
            </div>
          </div>
          <div class="row-field">
            <label class="lbl">信用卡费率<span class="text-destructive">*</span></label>
            <div class="flex flex-1 items-center gap-2">
              <input v-model="form.credit_activities_rate" placeholder="如 0.2" class="field-input flex-1" />
              <span class="dim text-xs">%（须在活动区间内）</span>
            </div>
          </div>
          <div class="row-field">
            <label class="lbl">活动补充材料</label>
            <div class="flex flex-1 flex-wrap items-center gap-2">
              <span v-for="(m, i) in form.activities_additions" :key="i" :title="m" class="media-ok flex items-center gap-1">
                材料{{ i + 1 }} ✓
                <button type="button" class="text-destructive" @click="removeMultiImage('activities_additions', i)">×</button>
              </span>
              <label v-if="form.activities_additions.length < 5" class="media-btn">
                <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.activities_additions" @change="onPickMultiImage('activities_additions', $event)" />
                {{ uploading.activities_additions ? '上传中…' : '选择图片' }}
              </label>
              <span class="dim text-xs">按活动要求，≤5 张</span>
            </div>
          </div>
        </template>
        <p class="text-[11px] text-muted-foreground">
          泛行业活动费率区间 0.2%~0.6%；微信要求借记卡/信用卡费率分开填写。
        </p>
      </section>

      <!-- 经营场景 sales_info（至少勾一类，勾了对应子表单必填）-->
      <section class="space-y-3">
        <h4 class="text-sm font-medium">经营场景 <span class="text-[11px] text-muted-foreground">（微信必填；至少选一类，选中后完善对应场景资料）</span></h4>
        <div class="flex flex-wrap gap-2">
          <label
            v-for="s in salesSceneDefs"
            :key="s.value"
            class="flex cursor-pointer items-center gap-1.5 bg-muted/40 px-3 py-1.5 text-xs"
            :class="hasScene(s.value) ? 'text-foreground ring-1 ring-primary' : 'text-muted-foreground'"
          >
            <input type="checkbox" :checked="hasScene(s.value)" class="accent-primary" @change="toggleScene(s.value)" />
            {{ s.label }}
          </label>
        </div>

        <!-- 线下场所 -->
        <div v-if="hasScene('SALES_SCENES_STORE')" class="space-y-3 bg-muted/40 p-3">
          <span class="text-xs font-medium">线下场所</span>
          <div class="row-field">
            <label class="lbl">门店名称<span class="text-destructive">*</span></label>
            <input v-model="form.biz_store.biz_store_name" placeholder="门店招牌名称" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">门店省市区<span class="text-destructive">*</span></label>
            <div class="flex flex-1 flex-wrap gap-2">
              <Select v-model="storeProvince" :options="wxProvinceOptions" searchable placeholder="省" class="min-w-[8rem] flex-1" />
              <Select v-model="storeCity" :options="storeCityOptions" searchable placeholder="市" class="min-w-[8rem] flex-1" :disabled="!storeProvince" />
              <Select v-model="form.biz_store.biz_address_code" :options="storeDistrictOptions" searchable placeholder="区/县" class="min-w-[8rem] flex-1" :disabled="!storeCity" />
            </div>
          </div>
          <p class="-mt-1 text-[11px] text-muted-foreground">微信要求精确到区/县级，编码取自官方省市区对照表。</p>
          <div class="row-field">
            <label class="lbl">门店地址<span class="text-destructive">*</span></label>
            <input v-model="form.biz_store.biz_store_address" placeholder="门店详细地址" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">门头照<span class="text-destructive">*</span></label>
            <div class="flex flex-1 flex-wrap items-center gap-2">
              <span v-for="(m, i) in form.biz_store.store_entrance_pic" :key="i" :title="m" class="media-ok flex items-center gap-1">
                门头{{ i + 1 }} ✓
                <button type="button" class="text-destructive" @click="form.biz_store.store_entrance_pic.splice(i, 1)">×</button>
              </span>
              <label v-if="form.biz_store.store_entrance_pic.length < 5" class="media-btn">
                <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.store_entrance_pic" @change="onPickArrayImage('store_entrance_pic', form.biz_store.store_entrance_pic, $event)" />
                {{ uploading.store_entrance_pic ? '上传中…' : '选择图片' }}
              </label>
            </div>
          </div>
          <div class="row-field">
            <label class="lbl">内部照<span class="text-destructive">*</span></label>
            <div class="flex flex-1 flex-wrap items-center gap-2">
              <span v-for="(m, i) in form.biz_store.indoor_pic" :key="i" :title="m" class="media-ok flex items-center gap-1">
                内部{{ i + 1 }} ✓
                <button type="button" class="text-destructive" @click="form.biz_store.indoor_pic.splice(i, 1)">×</button>
              </span>
              <label v-if="form.biz_store.indoor_pic.length < 5" class="media-btn">
                <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.indoor_pic" @change="onPickArrayImage('indoor_pic', form.biz_store.indoor_pic, $event)" />
                {{ uploading.indoor_pic ? '上传中…' : '选择图片' }}
              </label>
            </div>
          </div>
        </div>

        <!-- 服务号/公众号 -->
        <div v-if="hasScene('SALES_SCENES_MP')" class="space-y-3 bg-muted/40 p-3">
          <span class="text-xs font-medium">服务号/公众号 <span class="text-[11px] text-muted-foreground">（AppID 二选一）</span></span>
          <div class="row-field">
            <label class="lbl">服务商 AppID</label>
            <input v-model="form.mp_info.mp_appid" placeholder="服务商公众号 AppID" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">商家 AppID</label>
            <input v-model="form.mp_info.mp_sub_appid" placeholder="商家公众号 AppID" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">页面截图</label>
            <div class="flex flex-1 flex-wrap items-center gap-2">
              <span v-for="(m, i) in form.mp_info.mp_pics" :key="i" :title="m" class="media-ok flex items-center gap-1">
                截图{{ i + 1 }} ✓
                <button type="button" class="text-destructive" @click="form.mp_info.mp_pics.splice(i, 1)">×</button>
              </span>
              <label v-if="form.mp_info.mp_pics.length < 5" class="media-btn">
                <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.mp_pics" @change="onPickArrayImage('mp_pics', form.mp_info.mp_pics, $event)" />
                {{ uploading.mp_pics ? '上传中…' : '选择图片' }}
              </label>
            </div>
          </div>
        </div>

        <!-- 小程序 -->
        <div v-if="hasScene('SALES_SCENES_MINI_PROGRAM')" class="space-y-3 bg-muted/40 p-3">
          <span class="text-xs font-medium">小程序 <span class="text-[11px] text-muted-foreground">（AppID 二选一）</span></span>
          <div class="row-field">
            <label class="lbl">服务商 AppID</label>
            <input v-model="form.mini_program.mini_program_appid" placeholder="服务商小程序 AppID" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">商家 AppID</label>
            <input v-model="form.mini_program.mini_program_sub_appid" placeholder="商家小程序 AppID" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">小程序截图</label>
            <div class="flex flex-1 flex-wrap items-center gap-2">
              <span v-for="(m, i) in form.mini_program.mini_program_pics" :key="i" :title="m" class="media-ok flex items-center gap-1">
                截图{{ i + 1 }} ✓
                <button type="button" class="text-destructive" @click="form.mini_program.mini_program_pics.splice(i, 1)">×</button>
              </span>
              <label v-if="form.mini_program.mini_program_pics.length < 5" class="media-btn">
                <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.mini_program_pics" @change="onPickArrayImage('mini_program_pics', form.mini_program.mini_program_pics, $event)" />
                {{ uploading.mini_program_pics ? '上传中…' : '选择图片' }}
              </label>
            </div>
          </div>
        </div>

        <!-- 互联网网站 -->
        <div v-if="hasScene('SALES_SCENES_WEB')" class="space-y-3 bg-muted/40 p-3">
          <span class="text-xs font-medium">互联网网站</span>
          <div class="row-field">
            <label class="lbl">网站域名<span class="text-destructive">*</span></label>
            <input v-model="form.web_info.domain" placeholder="如 https://www.example.com" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">商家 AppID</label>
            <input v-model="form.web_info.web_appid" placeholder="网站对应商家 AppID（选填）" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">网站授权函</label>
            <div class="flex flex-1 items-center gap-2">
              <label class="media-btn">
                <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.web_authorisation" @change="onPickSingleTo('web_authorisation', (id) => (form.web_info.web_authorisation = id), $event)" />
                {{ uploading.web_authorisation ? '上传中…' : form.web_info.web_authorisation ? '重新上传' : '选择图片' }}
              </label>
              <span v-if="form.web_info.web_authorisation" class="media-ok">已上传 ✓</span>
              <span v-else class="dim text-xs">备案主体不一致时需上传</span>
            </div>
          </div>
        </div>

        <!-- App -->
        <div v-if="hasScene('SALES_SCENES_APP')" class="space-y-3 bg-muted/40 p-3">
          <span class="text-xs font-medium">App <span class="text-[11px] text-muted-foreground">（AppID 二选一）</span></span>
          <div class="row-field">
            <label class="lbl">服务商 AppID</label>
            <input v-model="form.app_info.app_appid" placeholder="服务商应用 AppID" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">商家 AppID</label>
            <input v-model="form.app_info.app_sub_appid" placeholder="商家应用 AppID" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">App 截图</label>
            <div class="flex flex-1 flex-wrap items-center gap-2">
              <span v-for="(m, i) in form.app_info.app_pics" :key="i" :title="m" class="media-ok flex items-center gap-1">
                截图{{ i + 1 }} ✓
                <button type="button" class="text-destructive" @click="form.app_info.app_pics.splice(i, 1)">×</button>
              </span>
              <label v-if="form.app_info.app_pics.length < 5" class="media-btn">
                <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.app_pics" @change="onPickArrayImage('app_pics', form.app_info.app_pics, $event)" />
                {{ uploading.app_pics ? '上传中…' : '选择图片' }}
              </label>
            </div>
          </div>
        </div>

        <!-- 企业微信 -->
        <div v-if="hasScene('SALES_SCENES_WEWORK')" class="space-y-3 bg-muted/40 p-3">
          <span class="text-xs font-medium">企业微信</span>
          <div class="row-field">
            <label class="lbl">商家 CorpID<span class="text-destructive">*</span></label>
            <input v-model="form.wework_info.sub_corp_id" placeholder="商家企业微信 CorpID" class="field-input flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">页面截图</label>
            <div class="flex flex-1 flex-wrap items-center gap-2">
              <span v-for="(m, i) in form.wework_info.wework_pics" :key="i" :title="m" class="media-ok flex items-center gap-1">
                截图{{ i + 1 }} ✓
                <button type="button" class="text-destructive" @click="form.wework_info.wework_pics.splice(i, 1)">×</button>
              </span>
              <label v-if="form.wework_info.wework_pics.length < 5" class="media-btn">
                <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.wework_pics" @change="onPickArrayImage('wework_pics', form.wework_info.wework_pics, $event)" />
                {{ uploading.wework_pics ? '上传中…' : '选择图片' }}
              </label>
            </div>
          </div>
        </div>
      </section>

      <!-- 超管联系信息（敏感）-->
      <section class="space-y-3">
        <h4 class="text-sm font-medium">超级管理员 <span class="text-[11px] text-muted-foreground">（敏感，加密存储）</span></h4>
        <div class="row-field">
          <label class="lbl">超管类型<span class="text-destructive">*</span></label>
          <Select v-model="form.contact_type" :options="contactTypeOptions" class="flex-1" />
        </div>
        <p v-if="form.contact_type === 'SUPER'" class="text-[11px] text-muted-foreground">
          经办人签约时会校验其微信绑定银行卡实名与所填证件号一致，请确保填写的是本人证件。
        </p>
        <div class="row-field">
          <label class="lbl">超管姓名<span class="text-destructive">*</span></label>
          <input v-model="form.contact_name" :placeholder="sensitivePlaceholder(has.contactName)" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">超管证件号<span v-if="form.contact_type === 'SUPER'" class="text-destructive">*</span></label>
          <input v-model="form.contact_id_number" :placeholder="sensitivePlaceholder(has.contactIdNumber)" class="field-input flex-1" />
        </div>
        <template v-if="form.contact_type === 'SUPER'">
          <div class="row-field">
            <label class="lbl">经办人证件类型<span class="text-destructive">*</span></label>
            <Select v-model="form.contact_id_doc_type" :options="idDocTypeOptions" class="flex-1" />
          </div>
          <div class="row-field">
            <label class="lbl">证件正面照<span class="text-destructive">*</span></label>
            <div class="flex flex-1 items-center gap-2">
              <label class="media-btn">
                <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.contact_id_doc_copy" @change="onPickSingleTo('contact_id_doc_copy', (id) => (form.contact_id_doc_copy = id), $event)" />
                {{ uploading.contact_id_doc_copy ? '上传中…' : form.contact_id_doc_copy ? '重新上传' : '选择图片' }}
              </label>
              <span v-if="form.contact_id_doc_copy" class="media-ok">已上传 ✓</span>
              <span v-else class="dim text-xs">身份证上传人像面</span>
            </div>
          </div>
          <div class="row-field">
            <label class="lbl">证件反面照</label>
            <div class="flex flex-1 items-center gap-2">
              <label class="media-btn">
                <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.contact_id_doc_copy_back" @change="onPickSingleTo('contact_id_doc_copy_back', (id) => (form.contact_id_doc_copy_back = id), $event)" />
                {{ uploading.contact_id_doc_copy_back ? '上传中…' : form.contact_id_doc_copy_back ? '重新上传' : '选择图片' }}
              </label>
              <span v-if="form.contact_id_doc_copy_back" class="media-ok">已上传 ✓</span>
              <span v-else class="dim text-xs">护照无需反面</span>
            </div>
          </div>
          <div class="row-field">
            <label class="lbl">证件有效期<span class="text-destructive">*</span></label>
            <div class="flex flex-1 items-center gap-2">
              <input v-model="form.contact_period_begin" placeholder="开始 yyyy-MM-dd" class="field-input flex-1" />
              <span class="dim">至</span>
              <input v-model="form.contact_period_end" placeholder="结束（长期填「长期」）" class="field-input flex-1" />
            </div>
          </div>
        </template>
        <div class="row-field">
          <label class="lbl">超管手机号<span class="text-destructive">*</span></label>
          <input v-model="form.mobile_phone" :placeholder="sensitivePlaceholder(has.mobilePhone)" class="field-input flex-1" />
        </div>
        <div class="row-field">
          <label class="lbl">超管邮箱</label>
          <input v-model="form.contact_email" :placeholder="sensitivePlaceholder(has.contactEmail)" class="field-input flex-1" />
        </div>
      </section>

      <!-- 补充材料 addition_info（选填/驳回补件）-->
      <section class="space-y-3">
        <h4 class="text-sm font-medium">补充材料 <span class="text-[11px] text-muted-foreground">（选填；驳回补件或部分指定行业进件时微信要求）</span></h4>
        <div class="row-field">
          <label class="lbl">开户承诺函</label>
          <div class="flex flex-1 items-center gap-2">
            <label class="media-btn">
              <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.legal_person_commitment" @change="onPickSingleTo('legal_person_commitment', (id) => (form.legal_person_commitment = id), $event)" />
              {{ uploading.legal_person_commitment ? '上传中…' : form.legal_person_commitment ? '重新上传' : '选择图片' }}
            </label>
            <span v-if="form.legal_person_commitment" class="media-ok">已上传 ✓</span>
            <span v-else class="dim text-xs">法定代表人开户承诺函</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">开户意愿视频</label>
          <div class="flex flex-1 items-center gap-2">
            <label class="media-btn">
              <input type="file" accept=".avi,.wmv,.mpeg,.mp4,.mov,.mkv,.flv,.f4v,.m4v,.rmvb" class="hidden" :disabled="videoUploading" @change="onPickVideo" />
              {{ videoUploading ? '上传中…' : videoMediaId ? '重新上传' : '选择视频' }}
            </label>
            <span v-if="videoMediaId" class="media-ok">已上传 ✓</span>
            <span v-else class="dim text-xs">avi/mp4/mov 等，≤5M</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">补充图片/PDF</label>
          <div class="flex flex-1 flex-wrap items-center gap-2">
            <span v-for="(m, i) in form.business_addition_pics" :key="i" :title="m" class="media-ok flex items-center gap-1">
              材料{{ i + 1 }} ✓
              <button type="button" class="text-destructive" @click="form.business_addition_pics.splice(i, 1)">×</button>
            </span>
            <label v-if="form.business_addition_pics.length < 5" class="media-btn">
              <input type="file" accept=".jpg,.jpeg,.png,.bmp" class="hidden" :disabled="uploading.business_addition_pics" @change="onPickArrayImage('business_addition_pics', form.business_addition_pics, $event)" />
              {{ uploading.business_addition_pics ? '上传中…' : '选择图片' }}
            </label>
            <span class="dim text-xs">≤5 张</span>
          </div>
        </div>
        <div class="row-field">
          <label class="lbl">补充说明</label>
          <textarea v-model="form.business_addition_msg" rows="2" placeholder="资金来源/用途等补充说明" class="field-input flex-1"></textarea>
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
