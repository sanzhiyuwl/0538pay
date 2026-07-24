package service

import (
	"strconv"
	"strings"
	"time"

	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
	"github.com/epvia/api/pkg/sign"
)

// MapiService 承载 V2 REST 接口族（对齐 epay api.php?s= → ApiHelper → lib/api/{Pay,Merchant,Transfer}）。
// 统一验签(pid查商户/keytype/timestamp防重放/MD5或RSA) + 回包 RSA 签名(平台私钥)。
// 复用既有 PayService/OrderService/MerchantCenterService/TransferService 的核心逻辑，
// 仅在此层做 JSON 契约包装与签名，不重复实现业务。
type MapiService struct {
	merchants   *repository.MerchantRepo
	orders      *repository.OrderRepo
	refunds     *repository.RefundOrderRepo
	accounts    *repository.AccountRepo
	channels    *repository.ChannelRepo
	cfg         *ConfigService
	pay         *PayService
	transfer    *TransferService
}

func NewMapiService(m *repository.MerchantRepo, o *repository.OrderRepo, rf *repository.RefundOrderRepo,
	a *repository.AccountRepo, ch *repository.ChannelRepo, cfg *ConfigService,
	pay *PayService, transfer *TransferService) *MapiService {
	return &MapiService{
		merchants: m, orders: o, refunds: rf, accounts: a, channels: ch,
		cfg: cfg, pay: pay, transfer: transfer,
	}
}

// MapiError 携带 epay 风格错误码（负数）。handler 据此返回 {code,msg}。
type MapiError struct {
	Code int
	Msg  string
}

func (e *MapiError) Error() string { return e.Msg }

// mapiErr 通用业务失败（对齐 epay echojsonmsg 默认 -1）。
func mapiErr(msg string) *MapiError { return &MapiError{Code: -1, Msg: msg} }

// mapiErrCode 指定错误码（-3签名/-4参数/-5路由）。
func mapiErrCode(code int, msg string) *MapiError { return &MapiError{Code: code, Msg: msg} }

// verify 统一验签（对齐 epay ApiHelper::verify + api_verify）。
// 返回校验通过的商户。规则：pid 必填→查商户→状态守卫→timestamp ±300s 防重放→
// keytype=1 强制 RSA；sign_type 缺省 MD5；MD5 用商户 key、RSA 用商户公钥验签。
func (s *MapiService) Verify(params map[string]string) (*model.Merchant, error) {
	// B1-39：对齐 epay ApiHelper::verify 的 pid 分支与错误码。
	//   ① 完全未传 pid 参数 → -4 '未传入任何参数'（epay isset($_POST['pid']) 为假的分支）。
	//   ② 传了 pid 但 intval 后为空/0（空串/非数字/0）→ '商户ID不能为空'，code 归 -1
	//      （epay throw 无 code，load_api 行34 $code!=0?$code:-1 → -1）。
	pidStr, ok := params["pid"]
	if !ok {
		return nil, mapiErrCode(-4, "未传入任何参数")
	}
	pid, err := strconv.ParseUint(strings.TrimSpace(pidStr), 10, 64)
	if err != nil || pid == 0 {
		return nil, mapiErr("商户ID不能为空") // mapiErr => code -1
	}
	m, err := s.merchants.FindByUIDSafe(uint(pid))
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, mapiErr("商户不存在！")
	}
	if m.Status == 0 {
		return nil, mapiErr("商户已被封禁")
	}

	signType := params["sign_type"]
	if signType == "" {
		signType = "MD5"
	}
	if m.KeyType == 1 && signType != "RSA" {
		return nil, mapiErrCode(-3, "该商户只能使用RSA签名类型")
	}
	// V2 REST 强制 timestamp 防重放（对齐 epay API_INIT 分支）。
	if err := checkTimestamp(params["timestamp"]); err != nil {
		// 统一为 -3（签名/校验域）错误码
		if pe, ok := err.(*PayError); ok {
			return nil, mapiErrCode(-3, pe.Msg)
		}
		return nil, mapiErrCode(-3, "时间戳校验失败")
	}
	if signType == "RSA" {
		if m.PublicKey == "" {
			return nil, mapiErrCode(-3, "该商户未配置 RSA 公钥")
		}
		if !sign.VerifyRSA(params, m.PublicKey) {
			return nil, mapiErrCode(-3, "RSA签名校验失败")
		}
	} else {
		if !sign.VerifyMD5(params, m.AppKey) {
			return nil, mapiErrCode(-3, "MD5签名校验失败")
		}
	}
	return m, nil
}

// signResponse 给回包追加 timestamp/sign_type=RSA/sign（平台私钥 RSA 签，对齐 epay ApiHelper 行27-29）。
// 平台私钥缺失时跳过签名（不阻断返回，仅回包无签名）。data 为最终返回的 map（含 code）。
func (s *MapiService) SignResponse(data map[string]string) map[string]string {
	priv := s.cfg.PlatformPrivateKey()
	data["timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)
	// B1-40：sign_type 无条件设 RSA（对齐 epay ApiHelper 行28 恒设，私钥缺失时也保留该字段，
	// 仅 sign 缺省）。商户侧据 sign_type 决定验签分支，缺该键会走偏。
	data["sign_type"] = "RSA"
	if priv == "" {
		return data
	}
	sig, err := sign.MakeRSA(data, priv)
	if err == nil {
		data["sign"] = sig
	}
	return data
}
