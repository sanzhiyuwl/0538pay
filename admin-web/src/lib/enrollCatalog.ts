/**
 * 微信特约商户进件「结算规则 + 优惠费率活动」对照表（自研扩展）。
 * 数据源：微信官方《费率结算规则对照表》(kf.qq.com) + 《优惠费率活动对照表》(4012082816)，
 * 由 wepay 项目抓取解析成 JSON，拷贝进本项目 src/data/ 作为静态资源（微信调价后手动更新该 json 即可）。
 *
 * 用途：填料抽屉「结算规则」区做「主体类型 → 行业」联动下拉，选中自动带出 settlement_id / fee_desc /
 * 是否需特殊资质；优惠费率活动按白名单（settlement_id + 行业名）过滤是否可报名。
 */
import settlementCatalog from '@/data/settlement_catalog.json'
import activityCatalog from '@/data/activity_catalog.json'

// —— 结算规则对照表 ——
export interface IndustryItem {
  settlement_id: string
  industry_id: string
  qualification_type: string // 所属行业名称（即微信 qualification_type 字段值）
  need_qual: boolean // 是否需要特殊资质
  qual_text: string // 资质说明
  fee_desc: string // 费率与入账周期说明
  scope: string // 业务范围说明
  activity_eligible: boolean // 该行业是否可报名优惠费率活动
}
type BySubject = Record<string, Record<string, IndustryItem[]>>
const bySubject = (settlementCatalog as { by_subject: BySubject }).by_subject

// 主体类型选项（与填料表单 subject_type 对齐；catalog 覆盖 5 类主体）。
export const subjectTypeLabels: Record<string, string> = {
  SUBJECT_TYPE_INDIVIDUAL: '个体户',
  SUBJECT_TYPE_ENTERPRISE: '企业',
  SUBJECT_TYPE_GOVERNMENT: '政府机关',
  SUBJECT_TYPE_INSTITUTIONS: '事业单位',
  SUBJECT_TYPE_OTHERS: '社会组织',
}

/** 某主体类型下的行业分组（组名 → 行业项列表）。无匹配返回空对象。 */
export function industryGroups(subjectType: string): Record<string, IndustryItem[]> {
  return bySubject[subjectType] ?? {}
}

/**
 * 某主体类型下的行业下拉选项（扁平化，带 optgroup 语义）。
 * value 用 `settlement_id|qualification_type` 复合键（同一行业名在不同规则下 settlement_id 不同，需一起锁定）。
 */
export interface IndustryOption {
  value: string // settlement_id|qualification_type
  label: string
  group: string
}
export function industryOptions(subjectType: string): IndustryOption[] {
  const groups = industryGroups(subjectType)
  const out: IndustryOption[] = []
  for (const [group, items] of Object.entries(groups)) {
    for (const it of items) {
      out.push({
        value: `${it.settlement_id}|${it.qualification_type}`,
        label: `${it.qualification_type}（${it.fee_desc}）`,
        group,
      })
    }
  }
  return out
}

/** 某主体下的行业大类（分组名）列表，供第一级下拉。 */
export function industryGroupNames(subjectType: string): string[] {
  return Object.keys(industryGroups(subjectType))
}

/** 某主体 + 某大类下的具体行业（门店类型）选项，供第二级下拉。value 为复合键。 */
export function groupIndustryOptions(subjectType: string, group: string): IndustryOption[] {
  const items = industryGroups(subjectType)[group] ?? []
  return items.map((it) => ({
    value: `${it.settlement_id}|${it.qualification_type}`,
    label: `${it.qualification_type}（${it.fee_desc}）`,
    group,
  }))
}

/** 反查某行业所属大类名（回填时用来定位第一级下拉当前值）。无匹配返回空串。 */
export function findIndustryGroup(subjectType: string, settlementId: string, qualificationType: string): string {
  for (const [group, items] of Object.entries(industryGroups(subjectType))) {
    if (items.some((x) => x.settlement_id === settlementId && x.qualification_type === qualificationType)) return group
  }
  return ''
}

/** 按复合键拆出 settlement_id / qualification_type。 */
export function splitIndustryKey(key: string): { settlementId: string; qualificationType: string } {
  const i = key.indexOf('|')
  if (i < 0) return { settlementId: '', qualificationType: '' }
  return { settlementId: key.slice(0, i), qualificationType: key.slice(i + 1) }
}

/** 在某主体下按 settlement_id + qualification_type 精确查一条行业项（回显/校验用）。 */
export function findIndustry(subjectType: string, settlementId: string, qualificationType: string): IndustryItem | undefined {
  for (const items of Object.values(industryGroups(subjectType))) {
    const hit = items.find((x) => x.settlement_id === settlementId && x.qualification_type === qualificationType)
    if (hit) return hit
  }
  return undefined
}

// —— 优惠费率活动对照表 ——
export interface ActivityDef {
  activities_id: string
  name: string
  desc: string
  credit_min: string
  credit_max: string
  debit_min: string
  debit_max: string
  valid_until: string
  detail_url: string
  whitelist: { settlement_ids: string[]; industry_names: string[] }
}
const activities = (activityCatalog as { activities: ActivityDef[] }).activities

/**
 * 给定已选行业（settlement_id + qualification_type），返回可报名的优惠费率活动。
 * 白名单规则：settlement_id 命中 或 行业名命中即视为可报名（对齐 wepay 判定）。
 */
export function eligibleActivities(settlementId: string, qualificationType: string): ActivityDef[] {
  if (!settlementId || !qualificationType) return []
  return activities.filter((a) => {
    const sids = a.whitelist?.settlement_ids ?? []
    const names = a.whitelist?.industry_names ?? []
    return sids.includes(settlementId) || names.includes(qualificationType)
  })
}

// —— 开户银行对照表（account_bank 取值）——
// 数据源：微信官方《开户银行对照表》(合作伙伴 4012082813，更新 2025.02.19)。
// account_bank 传的就是这里的「银行名」文本；17 家常用银行 + 兜底「其他银行」。
export const bankOptions: { value: string; label: string }[] = [
  '工商银行',
  '农业银行',
  '中国银行',
  '建设银行',
  '交通银行',
  '招商银行',
  '邮政储蓄银行',
  '民生银行',
  '中信银行',
  '浦发银行',
  '兴业银行',
  '光大银行',
  '广发银行',
  '平安银行',
  '北京银行',
  '华夏银行',
  '宁波银行',
  '其他银行',
].map((n) => ({ value: n, label: n }))

// —— 开户银行省市编码（bank_address_code，省级）——
// 数据源：微信官方《省市区编号对照表》(合作伙伴 4012082815)。该字段官方标注「即将下线、无需传入、非必填、至少精确到市」，
// 这里只做省级（GB/T 2260 省级行政区划码，34 个），够用不做区县级几千项级联。
export const provinceOptions: { value: string; label: string }[] = [
  { value: '110000', label: '北京市' },
  { value: '120000', label: '天津市' },
  { value: '130000', label: '河北省' },
  { value: '140000', label: '山西省' },
  { value: '150000', label: '内蒙古自治区' },
  { value: '210000', label: '辽宁省' },
  { value: '220000', label: '吉林省' },
  { value: '230000', label: '黑龙江省' },
  { value: '310000', label: '上海市' },
  { value: '320000', label: '江苏省' },
  { value: '330000', label: '浙江省' },
  { value: '340000', label: '安徽省' },
  { value: '350000', label: '福建省' },
  { value: '360000', label: '江西省' },
  { value: '370000', label: '山东省' },
  { value: '410000', label: '河南省' },
  { value: '420000', label: '湖北省' },
  { value: '430000', label: '湖南省' },
  { value: '440000', label: '广东省' },
  { value: '450000', label: '广西壮族自治区' },
  { value: '460000', label: '海南省' },
  { value: '500000', label: '重庆市' },
  { value: '510000', label: '四川省' },
  { value: '520000', label: '贵州省' },
  { value: '530000', label: '云南省' },
  { value: '540000', label: '西藏自治区' },
  { value: '610000', label: '陕西省' },
  { value: '620000', label: '甘肃省' },
  { value: '630000', label: '青海省' },
  { value: '640000', label: '宁夏回族自治区' },
  { value: '650000', label: '新疆维吾尔自治区' },
  { value: '710000', label: '台湾省' },
  { value: '810000', label: '香港特别行政区' },
  { value: '820000', label: '澳门特别行政区' },
]

// —— 门店省市区编码（biz_store_info.biz_address_code，区县级必填）——
// 微信「线下场所」的省市编码不同于开户银行：必须精确到区/县级 6 位码，且取值必须落在
// 微信官方《省市区编号对照表》(合作伙伴 4012082815) 内，省级码（xx0000）会被 PARAM_ERROR 拒绝。
// 数据源：该文档的全量 xlsx 附件解析而来（微信口径，非纯 GB/T 2260；微信调价/调整时手动重跑更新 json）。
import wxAreaCatalog from '@/data/wx_area_catalog.json'

interface AreaNode {
  code: string
  name: string
}
interface WxAreaCatalog {
  updated: string
  source: string
  provinces: AreaNode[]
  cities: Record<string, AreaNode[]> // 省code → 市列表
  districts: Record<string, AreaNode[]> // 市code → 区县列表
}
const wxArea = wxAreaCatalog as WxAreaCatalog

const toOptions = (list: AreaNode[]): { value: string; label: string }[] =>
  list.map((a) => ({ value: a.code, label: a.name }))

/** 门店省份下拉选项（区县级级联的第一级）。 */
export const wxProvinceOptions = toOptions(wxArea.provinces)

/** 某省下的市级下拉选项；无匹配返回空数组。 */
export function wxCityOptions(provinceCode: string): { value: string; label: string }[] {
  if (!provinceCode) return []
  return toOptions(wxArea.cities[provinceCode] ?? [])
}

/** 某市下的区县下拉选项（biz_address_code 最终取这一级的 code）；无匹配返回空数组。 */
export function wxDistrictOptions(cityCode: string): { value: string; label: string }[] {
  if (!cityCode) return []
  return toOptions(wxArea.districts[cityCode] ?? [])
}

/**
 * 由一个区县级 biz_address_code 反查其所属的省code、市code（用于回显已保存的值到三级下拉）。
 * 找不到时对应项返回空串。
 */
export function wxResolveArea(districtCode: string): { province: string; city: string } {
  const empty = { province: '', city: '' }
  if (!districtCode) return empty
  for (const [cityCode, list] of Object.entries(wxArea.districts)) {
    if (list.some((d) => d.code === districtCode)) {
      for (const [provCode, cities] of Object.entries(wxArea.cities)) {
        if (cities.some((c) => c.code === cityCode)) {
          return { province: provCode, city: cityCode }
        }
      }
      return { province: '', city: cityCode }
    }
  }
  return empty
}
