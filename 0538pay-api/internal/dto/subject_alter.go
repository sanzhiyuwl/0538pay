package dto

// 修改主体信息（服务商为商户发起主体资料变更申请）。
// 接口：POST /v3/mchalterapply/mchsubjectalterapplyment（官方 doc_id 4014090649，
// 知识库暂未同步该产品线文档，字段已对官方页面逐项核对，非凭记忆编造）。
// 场景：风控第二段 recover_way=MODIFY_SUBJECT_INFORMATION（修改主体资料）时，
// 服务商可直接代商户提交此申请，比引导商户去「微信支付商家助手」小程序自助处理更高效。
//
// ★ 字段命名与「提交申请单」(applyment4sub) 不同——UBO 用 card_* 前缀而非 ubo_id_doc_*，
//   两个产品各自独立建模，不与 EnrollUBO/buildUBOList 混用。

// SubjectAlterUBO 最终受益人（本产品字段名 card_*，与 applyment4sub 的 ubo_id_doc_* 不同）。
type SubjectAlterUBO struct {
	IDDocType   string `json:"id_doc_type"`   // 证件类型
	CardFront   string `json:"card_front"`    // 证件正面 media_id
	CardBack    string `json:"card_back"`     // 证件反面 media_id（选填）
	CardName    string `json:"card_name"`     // ★姓名
	CardNumber  string `json:"card_number"`   // ★证件号码
	CardAddress string `json:"card_address"`  // ★居住地址
	PeriodBegin string `json:"period_begin"`  // 证件有效期开始
	PeriodEnd   string `json:"period_end"`    // 证件有效期结束
}

// SubjectAlterReq 主体资料变更申请入参（明文，敏感字段由 service 加密后组装）。
type SubjectAlterReq struct {
	AlterScope       string `json:"alter_scope"`       // ALTER_SCOPE_FULL / _BUSINESS_CERT / _UBO（空=全部）
	OrganizationType string `json:"organization_type"` // 主体类型（变更范围未指定仅受益人资料时必填）
	FinanceInstitution *bool `json:"finance_institution,omitempty"` // 是否金融机构

	// 营业执照（个体户/企业）
	LicenseNumber  string `json:"license_number"`
	LicenseCopy    string `json:"license_copy"`
	BusinessMerchantName string `json:"business_merchant_name"` // 商户名称（营业执照登记名称）
	LegalPerson    string `json:"legal_person"`
	CompanyAddress string `json:"company_address"`
	LicensePeriodBegin string `json:"license_period_begin"`
	LicensePeriodEnd   string `json:"license_period_end"`

	// 登记证书（政府机关/事业单位/社会组织）
	CertType           string `json:"cert_type"`
	CertNumber         string `json:"cert_number"`
	CertCopy           string `json:"cert_copy"`
	CertMerchantName   string `json:"cert_merchant_name"`
	CertCompanyAddress string `json:"cert_company_address"`
	CertLegalPerson    string `json:"cert_legal_person"`
	CertPeriodBegin    string `json:"cert_period_begin"`
	CertPeriodEnd      string `json:"cert_period_end"`

	// 金融机构许可证（当主体是金融机构时必填）
	FinanceType        string   `json:"finance_type"`
	FinanceLicensePics []string `json:"finance_license_pics"`

	// 法人身份信息
	IDHolderType         string `json:"id_holder_type"`          // LEGAL 经营者/法人 / SUPER 经办人
	IDDocType            string `json:"id_doc_type"`
	AuthorizeLetterCopy  string `json:"authorize_letter_copy"` // 经办人时必传
	CardFront            string `json:"card_front"`
	CardBack             string `json:"card_back"`
	CardName             string `json:"card_name"`   // ★
	CardNumber           string `json:"card_number"` // ★
	CardAddress          string `json:"card_address"` // ★
	CardPeriodBegin      string `json:"card_period_begin"`
	CardPeriodEnd        string `json:"card_period_end"`
	AsUBO                *bool  `json:"as_ubo,omitempty"` // 经营者/法人是否为受益人

	// 最终受益人列表（企业/社会组织，经营者/法人非受益人时必填）
	UBOList []SubjectAlterUBO `json:"ubo_list"`

	// 补充材料
	BankOpenAccountLicense []string `json:"bank_openaccount_license"`
	OpenAccountApproval    []string `json:"openaccount_approval"`
	LegalOtherProve        []string `json:"legal_other_prove"`
	AgencyProve            []string `json:"agency_prove"`
	UBOProve               []string `json:"ubo_prove"`
}

// SubjectAlterView 主体资料变更申请回显（GET 表单预填 + 风控页展示进度用）。★敏感字段不回原文，只回是否已填。
type SubjectAlterView struct {
	AlterScope         string `json:"alter_scope"`
	OrganizationType   string `json:"organization_type"`
	LicenseNumber      string `json:"license_number"`
	LicenseCopy        string `json:"license_copy"`
	BusinessMerchantName string `json:"business_merchant_name"`
	LegalPerson        string `json:"legal_person"`
	CompanyAddress     string `json:"company_address"`
	LicensePeriodBegin string `json:"license_period_begin"`
	LicensePeriodEnd   string `json:"license_period_end"`
	CertType           string `json:"cert_type"`
	IDHolderType       string `json:"id_holder_type"`
	IDDocType          string `json:"id_doc_type"`
	CardFront          string `json:"card_front"`
	CardBack           string `json:"card_back"`
	HasCardName        bool   `json:"has_card_name"`
	HasCardNumber      bool   `json:"has_card_number"`
	HasCardAddress     bool   `json:"has_card_address"`
	CardPeriodBegin    string `json:"card_period_begin"`
	CardPeriodEnd      string `json:"card_period_end"`
	UBOCount           int    `json:"ubo_count"`
}
