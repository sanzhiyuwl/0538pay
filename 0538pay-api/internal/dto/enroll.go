package dto

// 代理进件「填全套资料」表单 DTO（自研扩展）。对齐微信 APIv3 特约商户进件 applyment4sub 核心字段。
// 一期覆盖最常用主体：个体户 / 企业。敏感字段（身份证/银行账号/手机）前端明文提交，
// 后端落库前经 SubMerchantService.EncryptSensitive（RSA-OAEP）加密后组装进 material_json，不明文落库。
// 图片类字段填 MediaID（微信「图片上传」接口返回的 media_id）；一期不做图片上传，前端填 media_id 占位。

// EnrollMaterialReq 填料表单入参。
type EnrollMaterialReq struct {
	// —— 主体基础（非敏感）——
	SubjectType   string `json:"subject_type"`   // SUBJECT_TYPE_INDIVIDUAL 个体户 / SUBJECT_TYPE_ENTERPRISE 企业
	MerchantShortname string `json:"merchant_shortname"` // 商户简称（展示给顾客）
	ServicePhone  string `json:"service_phone"`  // 客服电话（非敏感）

	// —— 营业执照（非敏感）——
	LicenseNumber string `json:"license_number"` // 证照编号/统一社会信用代码
	LicenseCopy   string `json:"license_copy"`   // 营业执照照片 media_id
	LegalPerson   string `json:"legal_person"`   // 法定代表人/经营者姓名（非敏感，执照上公开信息）
	LicenseAddress string `json:"license_address"` // 注册地址
	PeriodBegin   string `json:"period_begin"`   // 营业期限开始 yyyy-MM-dd
	PeriodEnd     string `json:"period_end"`     // 营业期限结束 yyyy-MM-dd（长期填「长期」）

	// —— 经营者/法人身份（★敏感，加密）——
	IDCardName    string `json:"id_card_name"`   // ★证件姓名
	IDCardNumber  string `json:"id_card_number"` // ★证件号码
	IDCardCopy    string `json:"id_card_copy"`   // 身份证人像面 media_id
	IDCardNational string `json:"id_card_national"` // 身份证国徽面 media_id
	CardPeriodBegin string `json:"card_period_begin"` // 身份证有效期开始
	CardPeriodEnd   string `json:"card_period_end"`   // 身份证有效期结束

	// —— 结算银行账户（★部分敏感）——
	BankAccountType string `json:"bank_account_type"` // BANK_ACCOUNT_TYPE_CORPORATE 对公 / BANK_ACCOUNT_TYPE_PERSONAL 对私
	AccountName   string `json:"account_name"`   // ★开户名称
	AccountBank   string `json:"account_bank"`   // 开户银行（如「工商银行」，非敏感）
	BankAddressCode string `json:"bank_address_code"` // 开户银行省市编码
	AccountNumber string `json:"account_number"` // ★银行账号

	// —— 超级管理员联系信息（★敏感）——
	ContactName   string `json:"contact_name"`   // ★超管姓名
	ContactIDNumber string `json:"contact_id_number"` // ★超管证件号
	MobilePhone   string `json:"mobile_phone"`   // ★超管手机号
	ContactEmail  string `json:"contact_email"`  // ★超管邮箱
}

// EnrollMaterialView 填料回显（GET）。★敏感字段一律不回原文，只回是否已填（脱敏），防密文/明文外泄。
type EnrollMaterialView struct {
	Filled        bool   `json:"filled"`         // 是否已填过资料（material_json 非空）
	SubjectType   string `json:"subject_type"`
	MerchantShortname string `json:"merchant_shortname"`
	ServicePhone  string `json:"service_phone"`
	LicenseNumber string `json:"license_number"`
	LicenseCopy   string `json:"license_copy"`
	LegalPerson   string `json:"legal_person"`
	LicenseAddress string `json:"license_address"`
	PeriodBegin   string `json:"period_begin"`
	PeriodEnd     string `json:"period_end"`
	IDCardCopy    string `json:"id_card_copy"`
	IDCardNational string `json:"id_card_national"`
	CardPeriodBegin string `json:"card_period_begin"`
	CardPeriodEnd   string `json:"card_period_end"`
	BankAccountType string `json:"bank_account_type"`
	AccountBank   string `json:"account_bank"`
	BankAddressCode string `json:"bank_address_code"`
	// 敏感字段仅回「是否已填」布尔，不回值。
	HasIDCardName    bool `json:"has_id_card_name"`
	HasIDCardNumber  bool `json:"has_id_card_number"`
	HasAccountName   bool `json:"has_account_name"`
	HasAccountNumber bool `json:"has_account_number"`
	HasContactName   bool `json:"has_contact_name"`
	HasContactIDNumber bool `json:"has_contact_id_number"`
	HasMobilePhone   bool `json:"has_mobile_phone"`
	HasContactEmail  bool `json:"has_contact_email"`
}
