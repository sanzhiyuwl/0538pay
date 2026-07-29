package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/epvia/api/pkg/faceid"
)

// CertVerifyService 实名认证第三方核验（对齐 epay user/ajax2 certificate 的 cert_open 分派）。
//
// cert_open：1=支付宝身份认证(人脸) / 2=手机号三要素 / 3=支付宝实名比对 / 4=腾讯云实名 / 5=阿里云金融级实人。
// 各方式请求构造/签名 1:1 对齐 epay，真实核验需对应第三方凭证（APPCODE/SecretKey/渠道私钥）。
// 手机三要素(方式2,阿里云市场 APPCODE)为同步核验，凭证到位即可端到端；其余为异步/需渠道私钥，凭证到位后接回调。
type CertVerifyService struct {
	cfg *ConfigService
}

func NewCertVerifyService(cfg *ConfigService) *CertVerifyService {
	return &CertVerifyService{cfg: cfg}
}

const certHTTPTimeout = 10 * time.Second

// Verify 按 cert_open 方式核验姓名+证件号(+手机号)。返回 (通过, error)。
// 无法同步判定（人脸类异步）时返回明确的待凭证/待回调错误。
func (s *CertVerifyService) Verify(ctx context.Context, name, idNo, phone string) (bool, error) {
	switch s.cfg.Int("cert_open", 0) {
	case 0:
		return false, maErr("平台未开启实名认证")
	case 2:
		return s.verifyPhone3(ctx, name, idNo, phone)
	case 1, 3:
		return false, maErr("支付宝实名认证需配置支付宝应用私钥(RSA2 网关签名)并接入异步回调，待真实凭证")
	case 4:
		// 腾讯云人脸核身是异步扫码流程，不走同步 Verify：由 CertSubmit 调 InitFace 发起、回调查 QueryFaceResult。
		return false, maErr("腾讯云扫码实名为异步流程，请通过扫码认证入口发起")
	case 5:
		return false, maErr("阿里云金融级实人认证需配置 AccessKey 并接入异步回调，待真实凭证")
	}
	return false, maErr("未知的实名认证方式")
}

// verifyPhone3 手机号三要素核验（阿里云云市场「手机三要素」接口 + APPCODE）。
//
// 对接上海神度「手机三要素实时版」：POST https://sdmobile3.market.alicloudapi.com/mobile_three/check，
// Header Authorization: APPCODE {cert_appcode}，Body 表单 name/idcard/mobile。
// 返回 {code, success, msg, data:{result, desc}}：顶层 code==200 表示接口调通，
// data.result 为字符串——"0"=一致（通过），其余（"1"不一致/查无等）为不通过，desc 给人看的原因。
//
// 【可配】调用地址后台可配（cert_phone3_url），默认对齐上海神度「手机三要素实时版」。
// 参数名各家云市场三要素接口统一为 name/idcard/mobile，写死；换服务商只需在实名设置页改调用地址，无需改代码。
// 返回判定用兼容多家的通用规则（见 judgePhone3Result）：先看 data.result==0，再回退顶层 code==200 / success。
func (s *CertVerifyService) verifyPhone3(ctx context.Context, name, idNo, phone string) (bool, error) {
	appcode := strings.TrimSpace(s.cfg.Str("cert_appcode"))
	if appcode == "" {
		return false, maErr("手机三要素认证未配置 APPCODE(cert_appcode)，待真实凭证")
	}
	if phone == "" {
		return false, maErr("请先绑定手机号再做三要素认证")
	}
	// 可配调用地址（配置空则回退神度默认）。参数名各家云市场三要素接口统一为 name/idcard/mobile，固定即可。
	apiURL := firstNonEmpty(s.cfg.Str("cert_phone3_url"), "https://sdmobile3.market.alicloudapi.com/mobile_three/check")

	form := url.Values{}
	form.Set("name", name)
	form.Set("idcard", idNo)
	form.Set("mobile", phone)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "APPCODE "+appcode)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	resp, err := (&http.Client{Timeout: certHTTPTimeout}).Do(req)
	if err != nil {
		return false, fmt.Errorf("请求三要素核验失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// 阿里云 API 网关鉴权失败（APPCODE 无效/未订阅/欠费/超限）会返 40x，且响应头带 X-Ca-Error-Message。
	// 这类是"凭证/配额问题"，与"姓名证件号不一致"截然不同，单独报清楚方便排查。
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		if em := strings.TrimSpace(resp.Header.Get("X-Ca-Error-Message")); em != "" {
			return false, maErr("APPCODE 鉴权失败：" + em + "（请检查已订阅该接口且 APPCODE 正确、额度未用尽）")
		}
		return false, maErr("APPCODE 无效或未订阅该接口，请核对后台配置与阿里云云市场订阅状态")
	}
	return judgePhone3Result(body)
}

// judgePhone3Result 兼容多家云市场三要素接口的返回判定：
//   - 顶层 code 非 200 视为接口调用失败（把服务商 msg 透出）；
//   - 有 data.result 字段：以它为准（"0"/0=一致通过，其余为不一致，用 data.desc 作原因）；
//   - 无 data.result：回退顶层 code==200 || success==true 即视为一致（对齐 epay check_cert）。
func judgePhone3Result(body []byte) (bool, error) {
	// result 用 json.Number 兼容字符串 "0" 与数字 0 两种写法。
	var r struct {
		Code    int    `json:"code"`
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
		Message string `json:"message"`
		Data    *struct {
			Result json.Number `json:"result"`
			Desc   string      `json:"desc"`
			Msg    string      `json:"msg"`
		} `json:"data"`
	}
	if jerr := json.Unmarshal(body, &r); jerr != nil {
		return false, maErr("三要素接口返回解析失败：" + truncateBody(string(body)))
	}
	topMsg := firstNonEmpty(r.Msg, r.Message)
	// 接口调用未成功（顶层 code 非 200）：把服务商原始 msg 透出来。
	if r.Code != 200 && !r.Success {
		if topMsg != "" {
			return false, maErr("三要素核验失败：" + topMsg)
		}
		return false, maErr("三要素接口调用失败(code=" + fmt.Sprint(r.Code) + ")")
	}
	// 有业务结果字段：以 data.result 为准。
	if r.Data != nil && r.Data.Result != "" {
		if res := strings.TrimSpace(r.Data.Result.String()); res == "0" {
			return true, nil
		}
		desc := firstNonEmpty(r.Data.Desc, r.Data.Msg, "实名信息核验不一致，请核对姓名、身份证号与手机号是否为本人")
		return false, maErr("实名核验未通过：" + desc)
	}
	// 无业务结果字段：顶层成功即视为一致（对齐 epay check_cert）。
	return true, nil
}

// VerifyCorp 企业工商三要素核验（对齐 epay check_corp_cert，阿里云云市场企业三要素 + cert_appcode2）。
//
// 校验「公司名称 + 统一社会信用代码/营业执照号 + 法定代表人姓名」三者是否在工商库一致。
// 默认对接 companythree.shumaidata.com/companythree/check（POST 表单 companyName/creditNo/legalPerson），
// 调用地址后台可配（cert_corp_url）。返回判定复用 judgePhone3Result（同为神度系 data.result 格式）。
func (s *CertVerifyService) VerifyCorp(ctx context.Context, corpName, corpNo, legalPerson string) (bool, error) {
	appcode := strings.TrimSpace(s.cfg.Str("cert_appcode2"))
	if appcode == "" {
		return false, maErr("企业认证未配置企业校验 APPCODE(cert_appcode2)，待真实凭证")
	}
	apiURL := firstNonEmpty(s.cfg.Str("cert_corp_url"), "https://companythree.shumaidata.com/companythree/check")

	form := url.Values{}
	form.Set("companyName", corpName)
	form.Set("creditNo", corpNo)
	form.Set("legalPerson", legalPerson)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "APPCODE "+appcode)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	resp, err := (&http.Client{Timeout: certHTTPTimeout}).Do(req)
	if err != nil {
		return false, fmt.Errorf("请求企业核验失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		if em := strings.TrimSpace(resp.Header.Get("X-Ca-Error-Message")); em != "" {
			return false, maErr("企业校验 APPCODE 鉴权失败：" + em + "（请检查已订阅企业三要素接口且 APPCODE 正确、额度未用尽）")
		}
		return false, maErr("企业校验 APPCODE 无效或未订阅该接口，请核对后台配置")
	}
	ok, jerr := judgePhone3Result(body)
	if jerr != nil {
		// judgePhone3Result 通用文案面向个人，这里替换成企业语义。
		return false, maErr(strings.NewReplacer(
			"三要素核验失败", "企业信息核验失败",
			"实名核验未通过", "企业信息核验未通过",
			"实名信息核验不一致，请核对姓名、身份证号与手机号是否为本人", "公司名称、营业执照号与法人信息不一致",
		).Replace(jerr.Error()))
	}
	return ok, nil
}

// truncateBody 截断返回体用于错误提示，避免把整段响应塞进业务错误。
func truncateBody(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}

// IsAsyncFace 当前实名方式是否为腾讯云扫码人脸核身（cert_open=4）。
// 为真时 CertSubmit 走异步发起（InitFace），而非同步 Verify。
func (s *CertVerifyService) IsAsyncFace() bool {
	return s.cfg.Int("cert_open", 0) == 4
}

// FaceInit 腾讯云扫码核身发起的返回：AuthToken 落库用于回调对账，RedirectURL 给前端弹二维码。
type FaceInit struct {
	AuthToken   string
	RedirectURL string
}

// InitFace 发起腾讯云人脸核身（对齐 epay ajax2 cert_open==4 发起分支）。
// callbackURL 为扫脸完成后腾讯云回跳地址（已带 state=uid）。返回 AuthToken + 扫码链接。
func (s *CertVerifyService) InitFace(ctx context.Context, name, idNo, callbackURL string) (*FaceInit, error) {
	id := strings.TrimSpace(s.cfg.Str("cert_qcloudid"))
	key := strings.TrimSpace(s.cfg.Str("cert_qcloudkey"))
	if id == "" || key == "" {
		return nil, maErr("未配置腾讯云 SecretId/SecretKey")
	}
	cli := faceid.New(id, key, "")
	r, err := cli.GetRealNameAuthToken(ctx, name, idNo, callbackURL)
	if err != nil {
		return nil, maErr(err.Error())
	}
	return &FaceInit{AuthToken: r.AuthToken, RedirectURL: r.RedirectURL}, nil
}

// QueryFaceResult 用 AuthToken 查腾讯云核身结果（对齐 epay alipaycertok.php cert_open==4 回调）。
// 返回 (通过, error)。未通过时 error 携带具体原因（姓名证件号不一致 / 微信未实名等）。
func (s *CertVerifyService) QueryFaceResult(ctx context.Context, authToken string) (bool, error) {
	id := strings.TrimSpace(s.cfg.Str("cert_qcloudid"))
	key := strings.TrimSpace(s.cfg.Str("cert_qcloudkey"))
	if id == "" || key == "" {
		return false, maErr("未配置腾讯云 SecretId/SecretKey")
	}
	cli := faceid.New(id, key, "")
	r, err := cli.GetRealNameAuthResult(ctx, authToken)
	if err != nil {
		return false, maErr(err.Error())
	}
	if r.ResultType != "0" {
		return false, maErr(faceid.ResultMessage(r.ResultType))
	}
	return true, nil
}
