package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/epvia/api/internal/config"
	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
)

// TestFillMaterialApplymentBody 端到端校验 applyment4sub 组装体的字段正确性（连本地 MySQL）。
// 覆盖本轮「对照微信官方文档清提交被拒隐患」修复的 5 处，按主体分支断言组装出的 material_json：
//   ① 营业执照 merchant_name = 商户名称（BusinessMerchantName）而非法人姓名（LegalPerson）；
//   ② contact_type=SUPER 时带经办人证件字段，LEGAL 时不带且证件号空时不下发 contact_id_number；
//   ③ MP/APP/WEWORK 场景带 *_pics；
//   ④ 敏感字段（身份证号/银行账号/联系人）全部加密（≠明文）且明文不出现在 body；
//   ⑤ 顶层结构与嵌套（sales_info 在 business_info 下）正确。
// 用自签 RSA 公钥让 submch.Configured() 过并真实走 RSA-OAEP 加密；高位 test id，跑完清库。
func TestFillMaterialApplymentBody(t *testing.T) {
	db, err := model.NewDB(config.DatabaseConfig{
		DSN:     "pay0538:pay0538@tcp(127.0.0.1:3306)/pay0538?charset=utf8mb4&parseTime=True&loc=Local",
		MaxOpen: 4, MaxIdle: 2,
	})
	if err != nil {
		t.Skipf("跳过：连不上本地 MySQL（%v）", err)
	}

	// 自签 RSA：公钥供 EncryptSensitive 加密，私钥供本测试解回密文验证。
	privPEM, pubPEM := testRSAKeyPair(t)
	rsaPriv := parseTestPriv(t, privPEM)
	submch := &SubMerchantService{cfg: &ConfigService{cache: map[string]string{
		"wx_partner_sp_mchid":    "1900000001",
		"wx_partner_serial_no":   "TESTSERIAL",
		"wx_partner_private_key": privPEM,
		"wx_partner_public_key":  pubPEM,
	}}}
	if !submch.Configured() {
		t.Fatal("测试凭证应判为已配置")
	}

	repo := repository.NewEnrollRepo(db)
	// FillMaterial 不读 config，给个空 cache 的 ConfigService 即可（与其它 service 单测同款构造）。
	svc := NewEnrollService(repo, &ConfigService{cache: map[string]string{}})
	svc.SetSubMerchant(submch)

	const testID uint = 991001
	cleanup := func() { db.Where("id = ?", testID).Delete(&model.SubMerchantEnroll{}) }
	cleanup()
	t.Cleanup(cleanup)

	// 每个用例：重置一张 paid 状态进件单 → FillMaterial → 反序列化 material_json 断言。
	fill := func(t *testing.T, req dto.EnrollMaterialReq) map[string]any {
		t.Helper()
		cleanup()
		if err := db.Create(&model.SubMerchantEnroll{
			ID: testID, EnrollNo: "TYTEST0001", AgentID: 0, MerchantName: "组装测试",
			Status: model.EnrollStatusPaid, AddTime: time.Now(),
		}).Error; err != nil {
			t.Fatalf("建测试进件单失败: %v", err)
		}
		if _, err := svc.FillMaterial(testID, nil, req); err != nil {
			t.Fatalf("FillMaterial 失败: %v", err)
		}
		e, err := repo.FindEnroll(testID)
		if err != nil {
			t.Fatalf("回查进件单失败: %v", err)
		}
		var body map[string]any
		if err := json.Unmarshal([]byte(e.MaterialJSON), &body); err != nil {
			t.Fatalf("material_json 非法 JSON: %v", err)
		}
		return body
	}

	// —— 通用最小合法资料构造器（个体户 + 身份证 + 对私 + 线下场所）——
	baseReq := func() dto.EnrollMaterialReq {
		return dto.EnrollMaterialReq{
			SubjectType:          "SUBJECT_TYPE_INDIVIDUAL",
			MerchantShortname:    "测试小店",
			ServicePhone:         "0538-1234567",
			LicenseNumber:        "91370000MA3XXXXXXX",
			LicenseCopy:          "MEDIA_LICENSE",
			BusinessMerchantName: "个体户张三便利店",
			LegalPerson:          "张三",
			LicenseAddress:       "山东省泰安市XX路1号",
			PeriodBegin:          "2020-01-01",
			PeriodEnd:            "长期",
			IDDocType:            "IDENTIFICATION_TYPE_IDCARD",
			IDCardName:           "张三",
			IDCardNumber:         "370900199001011234",
			IDCardCopy:           "MEDIA_ID_FRONT",
			IDCardNational:       "MEDIA_ID_BACK",
			CardPeriodBegin:      "2015-01-01",
			CardPeriodEnd:        "2035-01-01",
			BankAccountType:      "BANK_ACCOUNT_TYPE_PERSONAL",
			AccountName:          "张三",
			AccountBank:          "工商银行",
			BankAddressCode:      "370900",
			AccountNumber:        "6222021234567890123",
			ContactType:          "LEGAL",
			ContactName:          "张三",
			MobilePhone:          "13800138000",
			ContactEmail:         "zs@example.com",
			SettlementID:         "719",
			QualificationType:    "餐饮",
			SalesScenesType:      []string{"SALES_SCENES_STORE"},
			BizStore: dto.EnrollBizStore{
				BizStoreName: "测试小店", BizAddressCode: "370900", BizStoreAddress: "泰安市XX路1号",
				StoreEntrancePic: []string{"MEDIA_ENTRANCE"}, IndoorPic: []string{"MEDIA_INDOOR"},
			},
		}
	}

	// ① 营业执照 merchant_name = 商户名称，legal_person = 法人姓名（两者不同，防回退到填人名）。
	t.Run("营业执照商户名称非法人姓名", func(t *testing.T) {
		body := fill(t, baseReq())
		lic := digInto(t, body, "subject_info", "business_license_info")
		if lic["merchant_name"] != "个体户张三便利店" {
			t.Errorf("business_license_info.merchant_name=%v 期望「个体户张三便利店」（营业执照登记名称）", lic["merchant_name"])
		}
		if lic["legal_person"] != "张三" {
			t.Errorf("business_license_info.legal_person=%v 期望「张三」", lic["legal_person"])
		}
		if lic["merchant_name"] == lic["legal_person"] {
			t.Error("merchant_name 不应等于 legal_person（不能把法人姓名当商户名称）")
		}
	})

	// ② LEGAL 超管：不带经办人证件字段；证件号为空时不下发 contact_id_number（避免空密文触发 PARAM_ERROR）。
	t.Run("LEGAL超管不带经办人证件与空证件号", func(t *testing.T) {
		body := fill(t, baseReq())
		contact := digInto(t, body, "contact_info")
		if contact["contact_type"] != "LEGAL" {
			t.Errorf("contact_type=%v 期望 LEGAL", contact["contact_type"])
		}
		if _, ok := contact["contact_id_number"]; ok {
			t.Error("LEGAL 且未填证件号时不应下发 contact_id_number（空密文会被拒）")
		}
		for _, k := range []string{"contact_id_doc_type", "contact_id_doc_copy", "contact_period_begin", "contact_period_end"} {
			if _, ok := contact[k]; ok {
				t.Errorf("LEGAL 超管不应带经办人字段 %s", k)
			}
		}
	})

	// ② SUPER 超管：带经办人证件类型/正面照/有效期与证件号（加密）。
	t.Run("SUPER超管带经办人证件", func(t *testing.T) {
		req := baseReq()
		req.ContactType = "SUPER"
		req.ContactIDNumber = "370900198801019876"
		req.ContactIDDocType = "IDENTIFICATION_TYPE_IDCARD"
		req.ContactIDDocCopy = "MEDIA_CONTACT_FRONT"
		req.ContactPeriodBegin = "2015-01-01"
		req.ContactPeriodEnd = "2035-01-01"
		body := fill(t, req)
		contact := digInto(t, body, "contact_info")
		if contact["contact_type"] != "SUPER" {
			t.Errorf("contact_type=%v 期望 SUPER", contact["contact_type"])
		}
		if contact["contact_id_doc_type"] != "IDENTIFICATION_TYPE_IDCARD" {
			t.Errorf("缺 contact_id_doc_type，值=%v", contact["contact_id_doc_type"])
		}
		if contact["contact_id_doc_copy"] != "MEDIA_CONTACT_FRONT" {
			t.Errorf("缺 contact_id_doc_copy，值=%v", contact["contact_id_doc_copy"])
		}
		if contact["contact_period_begin"] != "2015-01-01" || contact["contact_period_end"] != "2035-01-01" {
			t.Errorf("经办人证件有效期不对：begin=%v end=%v", contact["contact_period_begin"], contact["contact_period_end"])
		}
		enc, _ := contact["contact_id_number"].(string)
		if enc == "" || enc == req.ContactIDNumber {
			t.Fatalf("contact_id_number 应加密，值=%q", enc)
		}
		if got := decryptOAEP(t, rsaPriv, enc); got != req.ContactIDNumber {
			t.Errorf("contact_id_number 解密=%q 期望 %q", got, req.ContactIDNumber)
		}
	})

	// ③ MP/APP/WEWORK 场景带 *_pics，且 sales_info 挂在 business_info 下。
	t.Run("经营场景截图与sales_info嵌套", func(t *testing.T) {
		req := baseReq()
		req.SalesScenesType = []string{"SALES_SCENES_MP", "SALES_SCENES_APP", "SALES_SCENES_WEWORK"}
		req.MpInfo = dto.EnrollMpInfo{MpSubAppid: "wxMP", MpPics: []string{"MEDIA_MP"}}
		req.AppInfo = dto.EnrollAppInfo{AppSubAppid: "appid", AppPics: []string{"MEDIA_APP"}}
		req.WeworkInfo = dto.EnrollWeworkInfo{SubCorpID: "corp", WeworkPics: []string{"MEDIA_WW"}}
		body := fill(t, req)
		sales := digInto(t, body, "business_info", "sales_info")
		mp := asMap(t, sales["mp_info"], "mp_info")
		if !hasPic(mp["mp_pics"], "MEDIA_MP") {
			t.Errorf("mp_info.mp_pics 缺 MEDIA_MP，值=%v", mp["mp_pics"])
		}
		app := asMap(t, sales["app_info"], "app_info")
		if !hasPic(app["app_pics"], "MEDIA_APP") {
			t.Errorf("app_info.app_pics 缺 MEDIA_APP，值=%v", app["app_pics"])
		}
		ww := asMap(t, sales["wework_info"], "wework_info")
		if !hasPic(ww["wework_pics"], "MEDIA_WW") {
			t.Errorf("wework_info.wework_pics 缺 MEDIA_WW，值=%v", ww["wework_pics"])
		}
	})

	// ④ 敏感字段加密且明文不出现在 body 里。
	t.Run("敏感字段加密无明文泄露", func(t *testing.T) {
		req := baseReq()
		body := fill(t, req)
		raw, _ := json.Marshal(body)
		for _, plain := range []string{req.IDCardNumber, req.AccountNumber, req.MobilePhone, req.ContactEmail} {
			if strings.Contains(string(raw), plain) {
				t.Errorf("material_json 不应含敏感明文 %q", plain)
			}
		}
		bank := digInto(t, body, "bank_account_info")
		encAcct, _ := bank["account_number"].(string)
		if encAcct == "" || encAcct == req.AccountNumber {
			t.Fatalf("account_number 应加密，值=%q", encAcct)
		}
		if got := decryptOAEP(t, rsaPriv, encAcct); got != req.AccountNumber {
			t.Errorf("account_number 解密=%q 期望 %q", got, req.AccountNumber)
		}
	})

	// ⑤ 事业单位主体走登记证书 certificate_info（而非营业执照），merchant_name 用证书商户名称。
	t.Run("事业单位走登记证书", func(t *testing.T) {
		req := baseReq()
		req.SubjectType = "SUBJECT_TYPE_INSTITUTIONS"
		req.BankAccountType = "BANK_ACCOUNT_TYPE_CORPORATE"
		req.AccountName = "泰安市XX事业单位"
		req.CertType = "CERTIFICATE_TYPE_2388"
		req.CertCopy = "MEDIA_CERT"
		req.CertNumber = "CERT12345"
		req.CertMerchantName = "泰安市XX事业单位"
		req.CertCompanyAddress = "泰安市XX路2号"
		req.CertLegalPerson = "李四"
		req.CertPeriodBegin = "2018-01-01"
		req.CertPeriodEnd = "长期"
		body := fill(t, req)
		subject := digInto(t, body, "subject_info")
		if _, ok := subject["business_license_info"]; ok {
			t.Error("登记证书主体不应带 business_license_info")
		}
		cert := asMap(t, subject["certificate_info"], "certificate_info")
		if cert["merchant_name"] != "泰安市XX事业单位" {
			t.Errorf("certificate_info.merchant_name=%v 不对", cert["merchant_name"])
		}
	})
}

// —— 断言辅助 ——

// digInto 逐层取嵌套 map，任一层缺失或非对象即 Fatal。
func digInto(t *testing.T, m map[string]any, path ...string) map[string]any {
	t.Helper()
	cur := m
	for _, k := range path {
		cur = asMap(t, cur[k], k)
	}
	return cur
}

func asMap(t *testing.T, v any, name string) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("%s 不是对象，值=%v", name, v)
	}
	return m
}

// hasPic 判断 JSON 数组（[]any）里是否含某 media_id 字符串。
func hasPic(v any, want string) bool {
	arr, ok := v.([]any)
	if !ok {
		return false
	}
	for _, it := range arr {
		if s, _ := it.(string); s == want {
			return true
		}
	}
	return false
}

