/**
 * 证件 OCR 识别 API（营业执照/身份证）。
 * 收单后台(/admin)、商户中心(/merchant)、代理进件(console:/console, agent:/agent)
 * 各挂同名子路由，仅前缀不同——调用方传入 base 前缀，鉴权由各自 token 负责。
 */
import { upload } from './client'

/** 营业执照识别结果（后端归一字段）。 */
export interface OCRLicenseResult {
  reg_number: string
  name: string
  legal_person: string
  address: string
  company_type: string
  establish_date: string
  valid_period_begin: string
  valid_period_end: string
  business: string
}

/** 身份证识别结果（人像面出姓名/号码/住址，国徽面出有效期）。 */
export interface OCRIDCardResult {
  name: string
  id_number: string
  sex: string
  nation: string
  birth: string
  address: string
  authority: string
  valid_period_begin: string
  valid_period_end: string
}

/** 识别营业执照。base 如 '/admin'、'/merchant'、'/console'、'/agent'。 */
export function recognizeLicense(base: string, file: File): Promise<OCRLicenseResult> {
  const form = new FormData()
  form.append('file', file)
  return upload<OCRLicenseResult>(`${base}/ocr/license`, form)
}

/** 识别身份证。side 默认人像面，'back' 识别国徽面（出有效期）。 */
export function recognizeIDCard(base: string, file: File, side?: 'front' | 'back'): Promise<OCRIDCardResult> {
  const form = new FormData()
  form.append('file', file)
  if (side === 'back') form.append('side', 'back')
  return upload<OCRIDCardResult>(`${base}/ocr/idcard`, form)
}
