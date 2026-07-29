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
	// ★商户名称：填营业执照登记名称（微信必填 merchant_name，2-128字符）。与法人姓名是两回事，
	// 企业填企业名，个体户按「个体户+经营者名」（执照登记名为空/纯数字/无字号时）。不能只填人名。
	BusinessMerchantName string `json:"business_merchant_name"`
	LegalPerson   string `json:"legal_person"`   // 法定代表人/经营者姓名（非敏感，执照上公开信息）
	LicenseAddress string `json:"license_address"` // 注册地址
	PeriodBegin   string `json:"period_begin"`   // 营业期限开始 yyyy-MM-dd
	PeriodEnd     string `json:"period_end"`     // 营业期限结束 yyyy-MM-dd（长期填「长期」）

	// —— 登记证书 certificate_info（政府机关/事业单位/社会组织必填；个体户/企业不填，走营业执照）——
	CertType           string `json:"cert_type"`            // 登记证书类型 CERTIFICATE_TYPE_xxxx
	CertCopy           string `json:"cert_copy"`            // 登记证书照片 media_id
	CertNumber         string `json:"cert_number"`          // 证书号
	CertMerchantName   string `json:"cert_merchant_name"`   // 登记证书商户名称
	CertCompanyAddress string `json:"cert_company_address"` // 证书注册地址
	CertLegalPerson    string `json:"cert_legal_person"`    // 证书法定代表人
	CertPeriodBegin    string `json:"cert_period_begin"`    // 证书有效期开始
	CertPeriodEnd      string `json:"cert_period_end"`      // 证书有效期结束（长期填「长期」）
	CertLetterCopy     string `json:"cert_letter_copy"`     // 单位证明函照片 media_id（政府/事业选传，传了免汇款验证）

	// —— 经营者/法人身份（★敏感，加密）——
	IDDocType     string `json:"id_doc_type"`     // 证件类型，默认 IDENTIFICATION_TYPE_IDCARD；非身份证走 id_doc_info
	IDCardName    string `json:"id_card_name"`   // ★证件姓名（身份证）
	IDCardNumber  string `json:"id_card_number"` // ★证件号码（身份证）
	IDCardCopy    string `json:"id_card_copy"`   // 身份证人像面 media_id
	IDCardNational string `json:"id_card_national"` // 身份证国徽面 media_id
	IDCardAddress string `json:"id_card_address"` // ★身份证居住地址（企业主体必填，敏感加密）
	CardPeriodBegin string `json:"card_period_begin"` // 身份证有效期开始
	CardPeriodEnd   string `json:"card_period_end"`   // 身份证有效期结束
	// 非身份证证件（护照/通行证/居留证等，id_doc_type != IDCARD 时填）。
	IDDocCopy      string `json:"id_doc_copy"`      // 证件正面照片 media_id
	IDDocCopyBack  string `json:"id_doc_copy_back"` // 证件反面照片 media_id
	IDDocName      string `json:"id_doc_name"`      // ★证件姓名（敏感）
	IDDocNumber    string `json:"id_doc_number"`    // ★证件号码（敏感）
	IDDocAddress   string `json:"id_doc_address"`   // ★证件居住地址（企业主体必填，敏感）
	DocPeriodBegin string `json:"doc_period_begin"` // 证件有效期开始
	DocPeriodEnd   string `json:"doc_period_end"`   // 证件有效期结束

	// —— 结算银行账户（★部分敏感）——
	BankAccountType string `json:"bank_account_type"` // BANK_ACCOUNT_TYPE_CORPORATE 对公 / BANK_ACCOUNT_TYPE_PERSONAL 对私
	AccountName   string `json:"account_name"`   // ★开户名称
	AccountBank   string `json:"account_bank"`   // 开户银行（如「工商银行」，非敏感）
	BankAddressCode string `json:"bank_address_code"` // 开户银行省市编码
	// 「其他银行」时，联行号 bank_branch_id 与 支行全称 bank_name 二选一（微信规则）。
	BankBranchID  string `json:"bank_branch_id"` // 开户银行联行号
	BankName      string `json:"bank_name"`      // 开户银行全称（含支行）
	AccountNumber string `json:"account_number"` // ★银行账号

	// —— 超级管理员联系信息（★部分敏感）——
	// 超管类型：LEGAL 经营者/法定代表人（默认）/ SUPER 经办人。为 SUPER 时下面 5 个证件字段必填。
	ContactType        string `json:"contact_type"`          // LEGAL / SUPER（空默认 LEGAL）
	ContactName   string `json:"contact_name"`   // ★超管姓名
	ContactIDNumber string `json:"contact_id_number"` // ★超管证件号（SUPER 时必填，敏感加密）
	MobilePhone   string `json:"mobile_phone"`   // ★超管手机号
	ContactEmail  string `json:"contact_email"`  // ★超管邮箱
	// —— 经办人（contact_type=SUPER）专属证件信息（非敏感部分）——
	ContactIDDocType   string `json:"contact_id_doc_type"`   // 经办人证件类型 IDENTIFICATION_TYPE_xxx
	ContactIDDocCopy   string `json:"contact_id_doc_copy"`   // 经办人证件正面 media_id
	ContactIDDocCopyBack string `json:"contact_id_doc_copy_back"` // 经办人证件反面 media_id（护照免）
	ContactPeriodBegin string `json:"contact_period_begin"`  // 经办人证件有效期开始 yyyy-MM-dd
	ContactPeriodEnd   string `json:"contact_period_end"`    // 经办人证件有效期结束（长期填「长期」）

	// —— 结算规则 settlement_info（非敏感；微信 applyment4sub 必填对象）——
	// settlement_id/qualification_type 值查微信《费率结算规则对照表》(kf.qq.com FAQ)，运营手填真实值。
	SettlementID  string `json:"settlement_id"`  // 入驻结算规则ID（必填）
	QualificationType string `json:"qualification_type"` // 所属行业名称（必填）
	Qualifications []string `json:"qualifications"` // 特殊资质图片 media_id 列表（选了需资质的行业才填，≤5）
	// 优惠费率活动（选填；泛行业活动 ID=20191030111cff5b5e，费率区间 0.2%~0.6%）。
	ActivitiesID  string `json:"activities_id"`  // 优惠费率活动ID（报名优惠活动则必填）
	// ★微信 2023-09-18 起不再兼容单一 activities_rate，必须分开传借记卡/信用卡费率；填了 activities_id 则两者必填。
	DebitActivitiesRate  string `json:"debit_activities_rate"`  // 非信用卡（借记卡）活动费率值，如 "0.2"
	CreditActivitiesRate string `json:"credit_activities_rate"` // 信用卡活动费率值，如 "0.2"
	ActivitiesAdditions  []string `json:"activities_additions"` // 优惠活动补充材料 media_id 列表（≤5）

	// —— 经营场景 sales_info（business_info 下必填对象；至少勾一类场景，勾了对应子对象必填）——
	SalesScenesType []string          `json:"sales_scenes_type"` // 场景类型：SALES_SCENES_STORE/MP/MINI_PROGRAM/WEB/APP/WEWORK
	BizStore        EnrollBizStore    `json:"biz_store"`         // 线下场所（勾 STORE 必填）
	MpInfo          EnrollMpInfo      `json:"mp_info"`           // 服务号/公众号（勾 MP 必填）
	MiniProgram     EnrollMiniProgram `json:"mini_program"`      // 小程序（勾 MINI_PROGRAM 必填）
	WebInfo         EnrollWebInfo     `json:"web_info"`          // 互联网网站（勾 WEB 必填）
	AppInfo         EnrollAppInfo     `json:"app_info"`          // App（勾 APP 必填）
	WeworkInfo      EnrollWeworkInfo  `json:"wework_info"`       // 企业微信（勾 WEWORK 必填）

	// —— 最终受益人 UBO（企业/社会组织，经营者非唯一受益人时填；空则微信自动回填经营者，≤4 人）——
	UBOList []EnrollUBO `json:"ubo_list"`

	// —— 补充材料 addition_info（选填/驳回补件）——
	LegalPersonCommitment string   `json:"legal_person_commitment"` // 法定代表人开户承诺函 media_id
	LegalPersonVideo      string   `json:"legal_person_video"`      // 法定代表人开户意愿视频 media_id
	BusinessAdditionPics  []string `json:"business_addition_pics"`  // 补充材料（图片/PDF）media_id（≤5）
	BusinessAdditionMsg   string   `json:"business_addition_msg"`   // 补充说明（资金来源/用途）
}

// —— 经营场景子对象（media_id 与文本，均非敏感）——

// EnrollBizStore 线下场所场景。
type EnrollBizStore struct {
	BizStoreName    string   `json:"biz_store_name"`    // 门店名称
	BizAddressCode  string   `json:"biz_address_code"`  // 门店省市编码（省市对照表，纯数字）
	BizStoreAddress string   `json:"biz_store_address"` // 门店详细地址
	StoreEntrancePic []string `json:"store_entrance_pic"` // 门头照 media_id（≥1）
	IndoorPic       []string `json:"indoor_pic"`        // 内部照 media_id（≥1）
	BizSubAppid     string   `json:"biz_sub_appid"`     // 门店对应商家 AppID（选填）
}

// EnrollMpInfo 服务号/公众号场景。
type EnrollMpInfo struct {
	MpAppid    string   `json:"mp_appid"`     // 服务商公众号 AppID（与商家二选一）
	MpSubAppid string   `json:"mp_sub_appid"` // 商家公众号 AppID（与服务商二选一）
	MpPics     []string `json:"mp_pics"`      // 页面截图 media_id
}

// EnrollMiniProgram 小程序场景。
type EnrollMiniProgram struct {
	MiniProgramAppid    string   `json:"mini_program_appid"`     // 服务商小程序 AppID（与商家二选一）
	MiniProgramSubAppid string   `json:"mini_program_sub_appid"` // 商家小程序 AppID（与服务商二选一）
	MiniProgramPics     []string `json:"mini_program_pics"`      // 小程序截图 media_id
}

// EnrollWebInfo 互联网网站场景。
type EnrollWebInfo struct {
	Domain          string `json:"domain"`           // 网站域名
	WebAuthorisation string `json:"web_authorisation"` // 网站授权函 media_id（备案主体不一致时）
	WebAppid        string `json:"web_appid"`        // 网站对应商家 AppID（选填）
}

// EnrollAppInfo App 场景。
type EnrollAppInfo struct {
	AppAppid    string   `json:"app_appid"`     // 服务商应用 AppID（与商家二选一）
	AppSubAppid string   `json:"app_sub_appid"` // 商家应用 AppID（与服务商二选一）
	AppPics     []string `json:"app_pics"`      // App 截图 media_id
}

// EnrollWeworkInfo 企业微信场景。
type EnrollWeworkInfo struct {
	SubCorpID  string   `json:"sub_corp_id"`  // 商家企业微信 CorpID
	WeworkPics []string `json:"wework_pics"`  // 企业微信页面截图 media_id
}

// EnrollUBO 最终受益人（★证件姓名/号码/地址敏感，加密落库）。
type EnrollUBO struct {
	UBOIDDocType   string `json:"ubo_id_doc_type"`    // 证件类型
	UBOIDDocCopy   string `json:"ubo_id_doc_copy"`    // 证件正面 media_id
	UBOIDDocCopyBack string `json:"ubo_id_doc_copy_back"` // 证件反面 media_id
	UBOIDDocName   string `json:"ubo_id_doc_name"`    // ★姓名
	UBOIDDocNumber string `json:"ubo_id_doc_number"`  // ★证件号码
	UBOIDDocAddress string `json:"ubo_id_doc_address"` // ★居住地址
	UBOPeriodBegin string `json:"ubo_period_begin"`   // 证件有效期开始
	UBOPeriodEnd   string `json:"ubo_period_end"`     // 证件有效期结束
}

// EnrollMaterialView 填料回显（GET）。★敏感字段一律不回原文，只回是否已填（脱敏），防密文/明文外泄。
type EnrollMaterialView struct {
	Filled        bool   `json:"filled"`         // 是否已填过资料（material_json 非空）
	SubjectType   string `json:"subject_type"`
	MerchantShortname string `json:"merchant_shortname"`
	ServicePhone  string `json:"service_phone"`
	LicenseNumber string `json:"license_number"`
	LicenseCopy   string `json:"license_copy"`
	BusinessMerchantName string `json:"business_merchant_name"`
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
	BankBranchID  string `json:"bank_branch_id"`
	BankName      string `json:"bank_name"`
	// 敏感字段仅回「是否已填」布尔，不回值。
	HasIDCardName    bool `json:"has_id_card_name"`
	HasIDCardNumber  bool `json:"has_id_card_number"`
	HasAccountName   bool `json:"has_account_name"`
	HasAccountNumber bool `json:"has_account_number"`
	HasContactName   bool `json:"has_contact_name"`
	HasContactIDNumber bool `json:"has_contact_id_number"`
	HasMobilePhone   bool `json:"has_mobile_phone"`
	HasContactEmail  bool `json:"has_contact_email"`
	// —— 超管类型 + 经办人证件（非敏感部分原样回显；证件号已在 has_contact_id_number）——
	ContactType          string `json:"contact_type"`
	ContactIDDocType     string `json:"contact_id_doc_type"`
	ContactIDDocCopy     string `json:"contact_id_doc_copy"`
	ContactIDDocCopyBack string `json:"contact_id_doc_copy_back"`
	ContactPeriodBegin   string `json:"contact_period_begin"`
	ContactPeriodEnd     string `json:"contact_period_end"`
	// —— 结算规则（非敏感，原样回显）——
	SettlementID  string `json:"settlement_id"`
	QualificationType string `json:"qualification_type"`
	Qualifications []string `json:"qualifications"`
	ActivitiesID  string `json:"activities_id"`
	DebitActivitiesRate  string `json:"debit_activities_rate"`
	CreditActivitiesRate string `json:"credit_activities_rate"`
	ActivitiesAdditions  []string `json:"activities_additions"`

	// —— 登记证书（非敏感，原样回显）——
	CertType           string `json:"cert_type"`
	CertCopy           string `json:"cert_copy"`
	CertNumber         string `json:"cert_number"`
	CertMerchantName   string `json:"cert_merchant_name"`
	CertCompanyAddress string `json:"cert_company_address"`
	CertLegalPerson    string `json:"cert_legal_person"`
	CertPeriodBegin    string `json:"cert_period_begin"`
	CertPeriodEnd      string `json:"cert_period_end"`
	CertLetterCopy     string `json:"cert_letter_copy"`

	// —— 身份证件（非敏感部分回显；姓名/号码/地址敏感只回 has_*）——
	IDDocType      string `json:"id_doc_type"`
	IDDocCopy      string `json:"id_doc_copy"`
	IDDocCopyBack  string `json:"id_doc_copy_back"`
	DocPeriodBegin string `json:"doc_period_begin"`
	DocPeriodEnd   string `json:"doc_period_end"`
	HasIDCardAddress bool `json:"has_id_card_address"`
	HasIDDocName     bool `json:"has_id_doc_name"`
	HasIDDocNumber   bool `json:"has_id_doc_number"`
	HasIDDocAddress  bool `json:"has_id_doc_address"`

	// —— 经营场景（非敏感，原样回显）——
	SalesScenesType []string          `json:"sales_scenes_type"`
	BizStore        EnrollBizStore    `json:"biz_store"`
	MpInfo          EnrollMpInfo      `json:"mp_info"`
	MiniProgram     EnrollMiniProgram `json:"mini_program"`
	WebInfo         EnrollWebInfo     `json:"web_info"`
	AppInfo         EnrollAppInfo     `json:"app_info"`
	WeworkInfo      EnrollWeworkInfo  `json:"wework_info"`

	// —— UBO 回显（★姓名/号码/地址不回原文，仅回非敏感 + 是否已填）——
	UBOList []EnrollUBOView `json:"ubo_list"`

	// —— 补充材料（非敏感，原样回显）——
	LegalPersonCommitment string   `json:"legal_person_commitment"`
	LegalPersonVideo      string   `json:"legal_person_video"`
	BusinessAdditionPics  []string `json:"business_addition_pics"`
	BusinessAdditionMsg   string   `json:"business_addition_msg"`
}

// EnrollUBOView UBO 回显（敏感字段脱敏，只回是否已填）。
type EnrollUBOView struct {
	UBOIDDocType     string `json:"ubo_id_doc_type"`
	UBOIDDocCopy     string `json:"ubo_id_doc_copy"`
	UBOIDDocCopyBack string `json:"ubo_id_doc_copy_back"`
	UBOPeriodBegin   string `json:"ubo_period_begin"`
	UBOPeriodEnd     string `json:"ubo_period_end"`
	HasName          bool   `json:"has_name"`
	HasNumber        bool   `json:"has_number"`
	HasAddress       bool   `json:"has_address"`
}
