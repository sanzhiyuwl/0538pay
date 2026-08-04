package service

import (
	"strings"

	"github.com/epvia/api/internal/dto"
)

// buildApplymentBody 组装微信特约商户进件 applyment4sub 请求体 + 非敏感明文快照。
//
// ★ 纯函数（不依赖任何 service 实例状态），代理进件线（EnrollService）与服务商通道商户进件线
//   （ChannelEnrollService）共用同一套组装与校验逻辑——这是「抽公共层，商户线复用不走代理端文件」
//   的落点：微信报文怎么组、哪些字段必填、敏感字段怎么加密全在这里，两条线各自的 service 只负责
//   状态机、落库与交付（商户线额外做占位符回填），互不牵连。
//
// 参数：
//   enc —— 敏感字段加密闭包（微信平台公钥 RSA-OAEP，见 SubMerchantService.EncryptSensitive）。
//   req —— 填料表单入参（五大块：主体/经营/银行/结算/联系人）。
// 返回：
//   body —— applyment4sub 请求体（不含 business_code，由各线 SubmitToWx 注入）。
//   meta —— 非敏感明文快照（回显编辑用，敏感只回 has_*）。
//   err  —— 校验/加密错误（用 enErr 包装的业务提示，调用方按需转成本线错误类型）。
func buildApplymentBody(enc func(string) (string, error), req dto.EnrollMaterialReq) (map[string]any, dto.EnrollMaterialView, error) {
	var zero dto.EnrollMaterialView
	// —— 基础必填校验（防组装出微信必拒的空体）——
	if strings.TrimSpace(req.MerchantShortname) == "" {
		return nil, zero, enErr("请填写商户简称")
	}
	if strings.TrimSpace(req.IDCardName) == "" || strings.TrimSpace(req.IDCardNumber) == "" {
		return nil, zero, enErr("请填写经营者/法人身份信息")
	}
	// ★身份证证件：人像面/国徽面/有效期均为微信必填（走 id_card_info 分支时），空则组装出空串必被拒。
	if dt := strings.TrimSpace(req.IDDocType); dt == "" || dt == "IDENTIFICATION_TYPE_IDCARD" {
		if strings.TrimSpace(req.IDCardCopy) == "" || strings.TrimSpace(req.IDCardNational) == "" ||
			strings.TrimSpace(req.CardPeriodBegin) == "" || strings.TrimSpace(req.CardPeriodEnd) == "" {
			return nil, zero, enErr("请完整填写身份证人像面、国徽面照片及有效期")
		}
	} else {
		// 非身份证证件：正面照 + 姓名/号码 + 有效期为必填。
		if strings.TrimSpace(req.IDDocCopy) == "" || strings.TrimSpace(req.IDDocName) == "" ||
			strings.TrimSpace(req.IDDocNumber) == "" || strings.TrimSpace(req.DocPeriodBegin) == "" ||
			strings.TrimSpace(req.DocPeriodEnd) == "" {
			return nil, zero, enErr("请完整填写证件正面照、姓名、号码及有效期")
		}
	}
	if strings.TrimSpace(req.AccountName) == "" || strings.TrimSpace(req.AccountNumber) == "" {
		return nil, zero, enErr("请填写结算银行账户信息")
	}
	// 结算规则 settlement_info：settlement_id/qualification_type 微信必填。
	if strings.TrimSpace(req.SettlementID) == "" {
		return nil, zero, enErr("请填写入驻结算规则ID（查微信费率结算规则对照表）")
	}
	if strings.TrimSpace(req.QualificationType) == "" {
		return nil, zero, enErr("请填写所属行业（查微信费率结算规则对照表）")
	}
	// ★报名优惠费率活动：填了 activities_id 则借记卡/信用卡费率两者必填（微信 2023-09-18 起口径）。
	if strings.TrimSpace(req.ActivitiesID) != "" {
		if strings.TrimSpace(req.DebitActivitiesRate) == "" || strings.TrimSpace(req.CreditActivitiesRate) == "" {
			return nil, zero, enErr("报名优惠费率活动时，借记卡与信用卡活动费率均需填写")
		}
	}
	// ★账户类型：仅个体户可用对私（经营者个人银行卡），其余主体只能对公（微信规则 4012719997）。
	if st := strings.TrimSpace(req.SubjectType); st != "" && st != "SUBJECT_TYPE_INDIVIDUAL" &&
		strings.TrimSpace(req.BankAccountType) == "BANK_ACCOUNT_TYPE_PERSONAL" {
		return nil, zero, enErr("企业/政府机关/事业单位/社会组织只能使用对公银行账户，不可选经营者个人银行卡")
	}
	// ★开户银行为「其他银行」时，联行号 bank_branch_id 与 支行全称 bank_name 二选一（微信规则）。
	if strings.TrimSpace(req.AccountBank) == "其他银行" {
		if strings.TrimSpace(req.BankBranchID) == "" && strings.TrimSpace(req.BankName) == "" {
			return nil, zero, enErr("开户银行为「其他银行」时，需填写开户银行联行号或开户银行全称（含支行）二选一")
		}
	}
	// ★主体资料：政府机关/事业单位/社会组织必传登记证书；个体户/企业必传营业执照。
	stForValidate := strings.TrimSpace(req.SubjectType)
	if stForValidate == "" {
		stForValidate = "SUBJECT_TYPE_INDIVIDUAL"
	}
	certSubject := stForValidate == "SUBJECT_TYPE_GOVERNMENT" ||
		stForValidate == "SUBJECT_TYPE_INSTITUTIONS" || stForValidate == "SUBJECT_TYPE_OTHERS"
	if certSubject {
		if strings.TrimSpace(req.CertType) == "" || strings.TrimSpace(req.CertCopy) == "" ||
			strings.TrimSpace(req.CertNumber) == "" || strings.TrimSpace(req.CertMerchantName) == "" ||
			strings.TrimSpace(req.CertCompanyAddress) == "" || strings.TrimSpace(req.CertLegalPerson) == "" ||
			strings.TrimSpace(req.CertPeriodBegin) == "" || strings.TrimSpace(req.CertPeriodEnd) == "" {
			return nil, zero, enErr("政府机关/事业单位/社会组织需完整填写登记证书信息（类型/照片/证书号/名称/地址/法人/有效期）")
		}
	} else {
		if strings.TrimSpace(req.LicenseNumber) == "" || strings.TrimSpace(req.LicenseCopy) == "" {
			return nil, zero, enErr("请填写营业执照信息（证照编号与执照照片）")
		}
		// ★商户名称必填（营业执照登记名称，2-128字符）。与法人姓名是两回事，填人名/空会被微信驳回。
		if mn := strings.TrimSpace(req.BusinessMerchantName); mn == "" {
			return nil, zero, enErr("请填写营业执照上的商户名称（营业执照登记名称，非法人姓名）")
		} else if l := len([]rune(mn)); l < 2 || l > 128 {
			return nil, zero, enErr("商户名称长度需为 2-128 个字符")
		}
	}
	// ★企业主体：身份证件需填居住地址（微信规则，仅企业必填）。
	if stForValidate == "SUBJECT_TYPE_ENTERPRISE" {
		docType := strings.TrimSpace(req.IDDocType)
		if docType == "" || docType == "IDENTIFICATION_TYPE_IDCARD" {
			if strings.TrimSpace(req.IDCardAddress) == "" {
				return nil, zero, enErr("企业主体需填写法定代表人身份证居住地址")
			}
		} else if strings.TrimSpace(req.IDDocAddress) == "" {
			return nil, zero, enErr("企业主体需填写法定代表人证件居住地址")
		}
	}
	// ★经营场景 sales_info：至少勾一类场景，勾了对应子对象关键字段必填（buildSalesInfo 内校验）。
	if len(trimNonEmpty(req.SalesScenesType)) == 0 {
		return nil, zero, enErr("请至少选择一类经营场景（线下场所/公众号/小程序/网站/App/企业微信）")
	}
	// ★超管类型：LEGAL 经营者/法人（默认）/ SUPER 经办人。为 SUPER 时证件类型/正面照/证件号/有效期必填。
	contactType := strings.TrimSpace(req.ContactType)
	if contactType == "" {
		contactType = "LEGAL"
	}
	if contactType != "LEGAL" && contactType != "SUPER" {
		return nil, zero, enErr("超级管理员类型只能为 LEGAL（经营者/法人）或 SUPER（经办人）")
	}
	if contactType == "SUPER" {
		if strings.TrimSpace(req.ContactIDDocType) == "" || strings.TrimSpace(req.ContactIDDocCopy) == "" ||
			strings.TrimSpace(req.ContactIDNumber) == "" || strings.TrimSpace(req.ContactPeriodBegin) == "" ||
			strings.TrimSpace(req.ContactPeriodEnd) == "" {
			return nil, zero, enErr("超管为经办人时，需填写经办人证件类型、证件正面照、证件号码与有效期")
		}
	}

	// —— 敏感字段逐个 RSA-OAEP 加密（空值加密返回空串，不报错）——
	idName, err := enc(req.IDCardName)
	if err != nil {
		return nil, zero, enErr("身份信息加密失败: " + err.Error())
	}
	idNumber, err := enc(req.IDCardNumber)
	if err != nil {
		return nil, zero, enErr("身份信息加密失败: " + err.Error())
	}
	acctName, err := enc(req.AccountName)
	if err != nil {
		return nil, zero, enErr("银行账户加密失败: " + err.Error())
	}
	acctNumber, err := enc(req.AccountNumber)
	if err != nil {
		return nil, zero, enErr("银行账户加密失败: " + err.Error())
	}
	contactName, err := enc(req.ContactName)
	if err != nil {
		return nil, zero, enErr("联系人信息加密失败: " + err.Error())
	}
	contactIDNum, err := enc(req.ContactIDNumber)
	if err != nil {
		return nil, zero, enErr("联系人信息加密失败: " + err.Error())
	}
	mobile, err := enc(req.MobilePhone)
	if err != nil {
		return nil, zero, enErr("联系人信息加密失败: " + err.Error())
	}
	email, err := enc(req.ContactEmail)
	if err != nil {
		return nil, zero, enErr("联系人信息加密失败: " + err.Error())
	}
	idAddress, err := enc(req.IDCardAddress)
	if err != nil {
		return nil, zero, enErr("身份信息加密失败: " + err.Error())
	}
	idDocName, err := enc(req.IDDocName)
	if err != nil {
		return nil, zero, enErr("身份信息加密失败: " + err.Error())
	}
	idDocNumber, err := enc(req.IDDocNumber)
	if err != nil {
		return nil, zero, enErr("身份信息加密失败: " + err.Error())
	}
	idDocAddress, err := enc(req.IDDocAddress)
	if err != nil {
		return nil, zero, enErr("身份信息加密失败: " + err.Error())
	}

	subjectType := strings.TrimSpace(req.SubjectType)
	if subjectType == "" {
		subjectType = "SUBJECT_TYPE_INDIVIDUAL"
	}
	// 结算规则 settlement_info：必填 settlement_id/qualification_type；资质图片与优惠活动按需带。
	settlementInfo := map[string]any{
		"settlement_id":      strings.TrimSpace(req.SettlementID),
		"qualification_type": strings.TrimSpace(req.QualificationType),
	}
	if quals := trimNonEmpty(req.Qualifications); len(quals) > 0 {
		settlementInfo["qualifications"] = quals
	}
	if aid := strings.TrimSpace(req.ActivitiesID); aid != "" {
		settlementInfo["activities_id"] = aid
		settlementInfo["debit_activities_rate"] = strings.TrimSpace(req.DebitActivitiesRate)
		settlementInfo["credit_activities_rate"] = strings.TrimSpace(req.CreditActivitiesRate)
		if adds := trimNonEmpty(req.ActivitiesAdditions); len(adds) > 0 {
			settlementInfo["activities_additions"] = adds
		}
	}
	// 结算银行账户：account_bank=「其他银行」时，联行号/支行全称二选一（微信规则），非空才带。
	bankAccountInfo := map[string]any{
		"bank_account_type": strings.TrimSpace(req.BankAccountType),
		"account_name":      acctName, // 密文
		"account_bank":      strings.TrimSpace(req.AccountBank),
		"bank_address_code": strings.TrimSpace(req.BankAddressCode),
		"account_number":    acctNumber, // 密文
	}
	if v := strings.TrimSpace(req.BankBranchID); v != "" {
		bankAccountInfo["bank_branch_id"] = v
	}
	if v := strings.TrimSpace(req.BankName); v != "" {
		bankAccountInfo["bank_name"] = v
	}
	// —— 主体资料 subject_info：个体户/企业走营业执照；政府/事业/社会组织走登记证书 ——
	isCertSubject := subjectType == "SUBJECT_TYPE_GOVERNMENT" ||
		subjectType == "SUBJECT_TYPE_INSTITUTIONS" || subjectType == "SUBJECT_TYPE_OTHERS"
	idDocType := strings.TrimSpace(req.IDDocType)
	if idDocType == "" {
		idDocType = "IDENTIFICATION_TYPE_IDCARD"
	}
	// 身份证件信息：身份证走 id_card_info，其他证件走 id_doc_info。
	identityInfo := map[string]any{"id_doc_type": idDocType}
	if idDocType == "IDENTIFICATION_TYPE_IDCARD" {
		idCardInfo := map[string]any{
			"id_card_copy":      strings.TrimSpace(req.IDCardCopy),
			"id_card_national":  strings.TrimSpace(req.IDCardNational),
			"id_card_name":      idName,   // 密文
			"id_card_number":    idNumber, // 密文
			"card_period_begin": strings.TrimSpace(req.CardPeriodBegin),
			"card_period_end":   strings.TrimSpace(req.CardPeriodEnd),
		}
		if idAddress != "" { // 企业主体必填（密文）
			idCardInfo["id_card_address"] = idAddress
		}
		identityInfo["id_card_info"] = idCardInfo
	} else {
		idDocInfo := map[string]any{
			"id_doc_copy":      strings.TrimSpace(req.IDDocCopy),
			"id_doc_name":      idDocName,   // 密文
			"id_doc_number":    idDocNumber, // 密文
			"doc_period_begin": strings.TrimSpace(req.DocPeriodBegin),
			"doc_period_end":   strings.TrimSpace(req.DocPeriodEnd),
		}
		if v := strings.TrimSpace(req.IDDocCopyBack); v != "" {
			idDocInfo["id_doc_copy_back"] = v
		}
		if idDocAddress != "" {
			idDocInfo["id_doc_address"] = idDocAddress
		}
		identityInfo["id_doc_info"] = idDocInfo
	}
	subjectInfo := map[string]any{
		"subject_type":  subjectType,
		"identity_info": identityInfo,
	}
	if isCertSubject {
		// 登记证书（政府/事业/社会组织）。
		certInfo := map[string]any{
			"cert_type":       strings.TrimSpace(req.CertType),
			"cert_copy":       strings.TrimSpace(req.CertCopy),
			"cert_number":     strings.TrimSpace(req.CertNumber),
			"merchant_name":   strings.TrimSpace(req.CertMerchantName),
			"company_address": strings.TrimSpace(req.CertCompanyAddress),
			"legal_person":    strings.TrimSpace(req.CertLegalPerson),
			"period_begin":    strings.TrimSpace(req.CertPeriodBegin),
			"period_end":      strings.TrimSpace(req.CertPeriodEnd),
		}
		subjectInfo["certificate_info"] = certInfo
		if v := strings.TrimSpace(req.CertLetterCopy); v != "" {
			subjectInfo["certificate_letter_copy"] = v // 单位证明函（政府/事业选传，免汇款验证）
		}
	} else {
		// 营业执照（个体户/企业）。
		subjectInfo["business_license_info"] = map[string]any{
			"license_number":  strings.TrimSpace(req.LicenseNumber),
			"license_copy":    strings.TrimSpace(req.LicenseCopy),
			"merchant_name":   strings.TrimSpace(req.BusinessMerchantName), // 营业执照登记名称（非法人姓名）
			"legal_person":    strings.TrimSpace(req.LegalPerson),
			"license_address": strings.TrimSpace(req.LicenseAddress),
			"period_begin":    strings.TrimSpace(req.PeriodBegin),
			"period_end":      strings.TrimSpace(req.PeriodEnd),
		}
	}
	// UBO 最终受益人（企业/社会组织可填；敏感字段加密）。
	if ubos := buildUBOList(enc, req.UBOList); len(ubos) > 0 {
		subjectInfo["ubo_info_list"] = ubos
	}

	// —— 经营场景 sales_info（至少一类；勾了哪类带哪类子对象）——
	salesInfo, err := buildSalesInfo(req)
	if err != nil {
		return nil, zero, err
	}

	// 组装 applyment4sub 请求体（business_code 提交微信时由各线 SubmitToWx 注入，这里先不填）。
	body := map[string]any{
		"subject_info": subjectInfo,
		"business_info": map[string]any{
			"merchant_shortname": strings.TrimSpace(req.MerchantShortname),
			"service_phone":      strings.TrimSpace(req.ServicePhone),
			"sales_info":         salesInfo,
		},
		"bank_account_info": bankAccountInfo,
		"settlement_info":   settlementInfo,
	}
	// —— 联系人 contact_info：contact_type 动态；证件号仅非空才带（可选敏感字段传空密文会触发 PARAM_ERROR）——
	contactInfo := map[string]any{
		"contact_type":  contactType,
		"contact_name":  contactName, // 密文
		"mobile_phone":  mobile,      // 密文
		"contact_email": email,       // 密文
	}
	if contactIDNum != "" {
		contactInfo["contact_id_number"] = contactIDNum // 密文（经办人必填）
	}
	// 经办人（SUPER）证件信息：证件类型/正面照/有效期（反面照护照免，非空才带）。
	if contactType == "SUPER" {
		contactInfo["contact_id_doc_type"] = strings.TrimSpace(req.ContactIDDocType)
		contactInfo["contact_id_doc_copy"] = strings.TrimSpace(req.ContactIDDocCopy)
		contactInfo["contact_period_begin"] = strings.TrimSpace(req.ContactPeriodBegin)
		contactInfo["contact_period_end"] = strings.TrimSpace(req.ContactPeriodEnd)
		if v := strings.TrimSpace(req.ContactIDDocCopyBack); v != "" {
			contactInfo["contact_id_doc_copy_back"] = v
		}
	}
	body["contact_info"] = contactInfo
	// 补充材料 addition_info（选填/驳回补件，全空则不带）。
	if add := buildAdditionInfo(req); len(add) > 0 {
		body["addition_info"] = add
	}

	// —— 非敏感明文快照（回显编辑用）——绝不含敏感原文 ——
	meta := dto.EnrollMaterialView{
		Filled:               true,
		SubjectType:          subjectType,
		MerchantShortname:    strings.TrimSpace(req.MerchantShortname),
		ServicePhone:         strings.TrimSpace(req.ServicePhone),
		LicenseNumber:        strings.TrimSpace(req.LicenseNumber),
		LicenseCopy:          strings.TrimSpace(req.LicenseCopy),
		BusinessMerchantName: strings.TrimSpace(req.BusinessMerchantName),
		LegalPerson:          strings.TrimSpace(req.LegalPerson),
		LicenseAddress:       strings.TrimSpace(req.LicenseAddress),
		PeriodBegin:          strings.TrimSpace(req.PeriodBegin),
		PeriodEnd:            strings.TrimSpace(req.PeriodEnd),
		IDCardCopy:           strings.TrimSpace(req.IDCardCopy),
		IDCardNational:       strings.TrimSpace(req.IDCardNational),
		CardPeriodBegin:      strings.TrimSpace(req.CardPeriodBegin),
		CardPeriodEnd:        strings.TrimSpace(req.CardPeriodEnd),
		BankAccountType:      strings.TrimSpace(req.BankAccountType),
		AccountBank:          strings.TrimSpace(req.AccountBank),
		BankAddressCode:      strings.TrimSpace(req.BankAddressCode),
		BankBranchID:         strings.TrimSpace(req.BankBranchID),
		BankName:             strings.TrimSpace(req.BankName),
		HasIDCardName:        idName != "",
		HasIDCardNumber:      idNumber != "",
		HasAccountName:       acctName != "",
		HasAccountNumber:     acctNumber != "",
		HasContactName:       contactName != "",
		ContactNameMasked:    maskNameSingle(req.ContactName),
		HasContactIDNumber:   contactIDNum != "",
		HasMobilePhone:       mobile != "",
		HasContactEmail:      email != "",
		ContactType:          contactType,
		ContactIDDocType:     strings.TrimSpace(req.ContactIDDocType),
		ContactIDDocCopy:     strings.TrimSpace(req.ContactIDDocCopy),
		ContactIDDocCopyBack: strings.TrimSpace(req.ContactIDDocCopyBack),
		ContactPeriodBegin:   strings.TrimSpace(req.ContactPeriodBegin),
		ContactPeriodEnd:     strings.TrimSpace(req.ContactPeriodEnd),
		SettlementID:         strings.TrimSpace(req.SettlementID),
		QualificationType:    strings.TrimSpace(req.QualificationType),
		Qualifications:       trimNonEmpty(req.Qualifications),
		ActivitiesID:         strings.TrimSpace(req.ActivitiesID),
		DebitActivitiesRate:  strings.TrimSpace(req.DebitActivitiesRate),
		CreditActivitiesRate: strings.TrimSpace(req.CreditActivitiesRate),
		ActivitiesAdditions:  trimNonEmpty(req.ActivitiesAdditions),
		CertType:             strings.TrimSpace(req.CertType),
		CertCopy:             strings.TrimSpace(req.CertCopy),
		CertNumber:           strings.TrimSpace(req.CertNumber),
		CertMerchantName:     strings.TrimSpace(req.CertMerchantName),
		CertCompanyAddress:   strings.TrimSpace(req.CertCompanyAddress),
		CertLegalPerson:      strings.TrimSpace(req.CertLegalPerson),
		CertPeriodBegin:      strings.TrimSpace(req.CertPeriodBegin),
		CertPeriodEnd:        strings.TrimSpace(req.CertPeriodEnd),
		CertLetterCopy:       strings.TrimSpace(req.CertLetterCopy),
		IDDocType:            idDocType,
		IDDocCopy:            strings.TrimSpace(req.IDDocCopy),
		IDDocCopyBack:        strings.TrimSpace(req.IDDocCopyBack),
		DocPeriodBegin:       strings.TrimSpace(req.DocPeriodBegin),
		DocPeriodEnd:         strings.TrimSpace(req.DocPeriodEnd),
		HasIDCardAddress:     idAddress != "",
		HasIDDocName:         idDocName != "",
		HasIDDocNumber:       idDocNumber != "",
		HasIDDocAddress:      idDocAddress != "",
		SalesScenesType:      trimNonEmpty(req.SalesScenesType),
		BizStore:             req.BizStore,
		MpInfo:               req.MpInfo,
		MiniProgram:          req.MiniProgram,
		WebInfo:              req.WebInfo,
		AppInfo:              req.AppInfo,
		WeworkInfo:           req.WeworkInfo,
		UBOList:              buildUBOViews(req.UBOList),
		LegalPersonCommitment: strings.TrimSpace(req.LegalPersonCommitment),
		LegalPersonVideo:      strings.TrimSpace(req.LegalPersonVideo),
		BusinessAdditionPics:  trimNonEmpty(req.BusinessAdditionPics),
		BusinessAdditionMsg:   strings.TrimSpace(req.BusinessAdditionMsg),
	}
	return body, meta, nil
}

// maskNameSingle 对姓名脱敏，只隐藏一个字，用于「谁来扫码签约」提示（非明文回显，不破脱敏红线）。
// 区别于 merchant_center.go 的 maskName（隐藏首字之外全部）：这里只打码一个字，尽量可辨认本人。
//   1 字：原样返回（无可隐藏）——如「王」→「王」
//   2 字：隐藏末字——如「张伟」→「张*」
//   3 字及以上：隐藏正中一个字——如「徐志坤」→「徐*坤」，「欧阳娜娜」→「欧阳*娜」
// 按 rune 处理，兼容中文。空串返回空串。
func maskNameSingle(name string) string {
	r := []rune(strings.TrimSpace(name))
	n := len(r)
	switch {
	case n == 0:
		return ""
	case n == 1:
		return string(r)
	case n == 2:
		return string(r[0]) + "*"
	default:
		mid := n / 2
		r[mid] = '*'
		return string(r)
	}
}
