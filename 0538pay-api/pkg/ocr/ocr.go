// Package ocr 提供证件文字识别（OCR）能力，支持阿里云与腾讯云两家引擎。
//
// 收单侧商户实名认证与代理进件填料共用同一套识别能力：
// 上传营业执照 / 身份证图片 → 识别出结构化字段 → 回填表单由人工核对后提交。
//
// 阿里云对齐 epay includes/lib/AliyunRecognize.php（RPC 风格 + HMAC-SHA1 签名）；
// 腾讯云走 TC3-HMAC-SHA256 签名的 OCR 接口（BizLicenseOCR / IDCardOCR）。
package ocr

import "context"

// LicenseResult 营业执照识别结果（各引擎字段归一到此）。
type LicenseResult struct {
	// 统一社会信用代码 / 注册号
	RegNumber string `json:"reg_number"`
	// 营业执照登记名称（单位名称）
	Name string `json:"name"`
	// 法定代表人 / 经营者
	LegalPerson string `json:"legal_person"`
	// 注册地址 / 经营场所
	Address string `json:"address"`
	// 类型（有限责任公司 / 个体工商户 等，可空）
	CompanyType string `json:"company_type"`
	// 成立日期（yyyy-MM-dd，可空）
	EstablishDate string `json:"establish_date"`
	// 营业期限起（yyyy-MM-dd，可空）
	ValidPeriodBegin string `json:"valid_period_begin"`
	// 营业期限止（yyyy-MM-dd 或「长期」，可空）
	ValidPeriodEnd string `json:"valid_period_end"`
	// 经营范围（可空）
	Business string `json:"business"`
}

// IDCardResult 身份证识别结果（各引擎字段归一到此）。
// 身份证正面（人像面）出姓名/性别/民族/生日/住址/号码；反面（国徽面）出有效期。
type IDCardResult struct {
	// 姓名
	Name string `json:"name"`
	// 公民身份号码
	IDNumber string `json:"id_number"`
	// 性别（男/女，可空）
	Sex string `json:"sex"`
	// 民族（可空）
	Nation string `json:"nation"`
	// 出生日期（yyyy-MM-dd，可空）
	Birth string `json:"birth"`
	// 住址
	Address string `json:"address"`
	// 签发机关（反面，可空）
	Authority string `json:"authority"`
	// 有效期起（yyyy-MM-dd，反面，可空）
	ValidPeriodBegin string `json:"valid_period_begin"`
	// 有效期止（yyyy-MM-dd 或「长期」，反面，可空）
	ValidPeriodEnd string `json:"valid_period_end"`
}

// IDCardSide 身份证识别的证件面。
type IDCardSide string

const (
	// IDCardFront 人像面（出姓名/号码/住址）。
	IDCardFront IDCardSide = "front"
	// IDCardBack 国徽面（出有效期/签发机关）。
	IDCardBack IDCardSide = "back"
)

// Recognizer 统一的证件识别引擎接口。两家引擎各自实现。
type Recognizer interface {
	// RecognizeBusinessLicense 识别营业执照。image 为图片原始字节。
	RecognizeBusinessLicense(ctx context.Context, image []byte) (*LicenseResult, error)
	// RecognizeIDCard 识别身份证。side 指定人像面或国徽面。
	RecognizeIDCard(ctx context.Context, image []byte, side IDCardSide) (*IDCardResult, error)
}
