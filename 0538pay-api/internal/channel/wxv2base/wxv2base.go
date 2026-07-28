// Package wxv2base 为微信支付 APIv2 各支付形态（Native/JSAPI/H5/APP）提供共享逻辑：
// 统一下单 execute（拼公共参数+签名+XML+POST+验应答签）、查单、回调解析、退款（带 mTLS 证书）。
//
// 逐条对齐 epay 内置 cccyun/wechatpay-sdk（BaseService/PaymentService）：
//   - 公共参数：appid/mch_id/nonce_str/sign_type=MD5；服务商模式追加 sub_mch_id/sub_appid。
//   - execute：merge 公共参数 → 签名 → array2Xml → POST → xml2array → return_code/result_code=SUCCESS 判定 + 验应答签。
//   - 下单：/pay/unifiedorder，据 trade_type=NATIVE/JSAPI/MWEB/APP 取 code_url/prepay_id/mweb_url。
//   - 查单：/pay/orderquery，trade_state=SUCCESS 判已付。
//   - 退款：/secapi/pay/refund，需商户证书 mTLS。
//   - 回调：验签 + total_fee/out_trade_no 校验（各形态通用）。
//
// 与 V3 分离：V2 走 XML+MD5，V3 走 JSON+RSA，两套 base 并存（对齐 epay wxpay 系 vs wxpayn 系）。
package wxv2base

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/pkg/wxpayv2"
	"github.com/shopspring/decimal"
)

// APIHost 微信支付 V2 网关基址（var 而非 const，便于单测用 httptest 替换）。
var APIHost = "https://api.mch.weixin.qq.com"

const httpTimeout = 15 * time.Second

// 回调保留键（复用 channel 通用常量）。
const (
	KeyBody = channel.RawBody
)

// 服务商模式与证书相关 Extra 键（对齐 epay wxpaysl config：sub_mchid=子商户号、sub_appid=子商户appid；
// cert_pem/key_pem=商户 API 证书 PEM 内容，用于退款/撤单的 mTLS）。
const (
	extraSubMchID = "sub_mchid"
	extraSubAppID = "sub_appid"
	extraCertPEM  = "cert_pem"
	extraKeyPEM   = "key_pem"
)

// SubMchID 子商户号（多个逗号分隔时取第一个；V2 无随机分配需求，取首个足够）。
func SubMchID(cfg channel.Config) string {
	v := cfg.ExtraOr(extraSubMchID, "")
	if v == "" {
		return ""
	}
	if parts := strings.Split(v, ","); len(parts) > 1 {
		return strings.TrimSpace(parts[0])
	}
	return strings.TrimSpace(v)
}

// SubAppID 子商户 appid（可空）。
func SubAppID(cfg channel.Config) string { return cfg.ExtraOr(extraSubAppID, "") }

// IsPartner 是否服务商模式（配了子商户号）。
func IsPartner(cfg channel.Config) bool { return SubMchID(cfg) != "" }

// PublicParams 组装 V2 公共请求参数（对齐 PaymentService 构造的 publicParams）。
func PublicParams(cfg channel.Config) (map[string]string, error) {
	if cfg.AppID == "" || cfg.MchID == "" || cfg.Key == "" {
		return nil, fmt.Errorf("微信 V2 通道缺少 appid/mch_id/apikey")
	}
	nonce, err := wxpayv2.NonceStr(32)
	if err != nil {
		return nil, err
	}
	p := map[string]string{
		"appid":     cfg.AppID,
		"mch_id":    cfg.MchID,
		"nonce_str": nonce,
		"sign_type": wxpayv2.SignTypeMD5,
	}
	if sub := SubMchID(cfg); sub != "" {
		p["sub_mch_id"] = sub
	}
	if sub := SubAppID(cfg); sub != "" {
		p["sub_appid"] = sub
	}
	return p, nil
}

// Execute 通用请求：merge 公共参数 → 签名 → XML → POST → 解析 → 判 return_code/result_code + 验应答签。
// useCert=true 时用商户证书做 mTLS（退款/撤单必需）。对齐 BaseService::execute。
func Execute(ctx context.Context, cfg channel.Config, url string, params map[string]string, useCert bool) (map[string]string, error) {
	pub, err := PublicParams(cfg)
	if err != nil {
		return nil, err
	}
	for k, v := range params {
		pub[k] = v
	}
	pub["sign"] = wxpayv2.MakeSign(pub, cfg.Key)
	reqXML := wxpayv2.MapToXML(pub)

	client, err := httpClient(cfg, useCert)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(reqXML))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("微信 V2 请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	result, err := wxpayv2.XMLToMap(string(respBody))
	if err != nil {
		return nil, fmt.Errorf("微信 V2 应答解析失败: %w (原文 %s)", err, string(respBody))
	}
	if result["return_code"] != "SUCCESS" {
		return nil, fmt.Errorf("微信 V2 通信失败: %s", result["return_msg"])
	}
	if result["result_code"] != "SUCCESS" {
		// 返回结构化业务错误（含 err_code），供付款码轮询/撤单据 USERPAYING/SYSTEMERROR 判定
		// （对齐 epay WeChatPayException::getErrCode）。
		return nil, &BizError{ErrCode: result["err_code"], ErrCodeDes: result["err_code_des"], Raw: result}
	}
	// 验应答签（对齐 BaseService::execute：有 sign 则校验，防伪造）。
	if _, ok := result["sign"]; ok && !wxpayv2.CheckSign(result, cfg.Key) {
		return nil, fmt.Errorf("微信 V2 应答验签失败")
	}
	return result, nil
}

// UnifiedOrder 统一下单 /pay/unifiedorder（对齐 PaymentService::unifiedOrder）。
func UnifiedOrder(ctx context.Context, cfg channel.Config, params map[string]string) (map[string]string, error) {
	if params["out_trade_no"] == "" || params["body"] == "" || params["total_fee"] == "" || params["trade_type"] == "" {
		return nil, fmt.Errorf("统一下单缺少必填参数 out_trade_no/body/total_fee/trade_type")
	}
	return Execute(ctx, cfg, APIHost+"/pay/unifiedorder", params, false)
}

// BaseOrderParams 组装下单公共业务参数（body/out_trade_no/total_fee/spbill_create_ip/notify_url）。
func BaseOrderParams(cfg channel.Config, req channel.CreateReq) (map[string]string, error) {
	notify := req.NotifyURL
	if notify == "" {
		notify = cfg.NotifyURL
	}
	if notify == "" {
		return nil, fmt.Errorf("微信 V2 通道缺少 notify_url 回调地址")
	}
	body := req.Subject
	if body == "" {
		body = "商品支付"
	}
	ip := req.ClientIP
	if ip == "" {
		ip = "127.0.0.1"
	}
	return map[string]string{
		"body":             body,
		"out_trade_no":     req.TradeNo,
		"total_fee":        YuanToFenStr(req.Money),
		"spbill_create_ip": ip,
		"notify_url":       notify,
	}, nil
}

// BizError 微信 V2 业务错误（result_code!=SUCCESS，含 err_code），对齐 epay WeChatPayException。
// 付款码支付据 err_code=USERPAYING/SYSTEMERROR 走轮询查单 + 撤单容错。
type BizError struct {
	ErrCode    string            // 业务错误码（USERPAYING/SYSTEMERROR/ORDERPAID…）
	ErrCodeDes string            // 错误码描述
	Raw        map[string]string // 原始应答
}

func (e *BizError) Error() string {
	if e.ErrCodeDes != "" {
		return e.ErrCode + " " + e.ErrCodeDes
	}
	return e.ErrCode
}

// QueryPaid 主动查单 /pay/orderquery，trade_state=SUCCESS 判已付（对齐 orderQuery）。
func QueryPaid(ctx context.Context, cfg channel.Config, tradeNo string) (bool, error) {
	result, err := QueryOrder(ctx, cfg, tradeNo)
	if err != nil {
		return false, err
	}
	return result["trade_state"] == "SUCCESS", nil
}

// QueryOrder 主动查单 /pay/orderquery，返回完整应答（trade_state/transaction_id/openid/total_fee…）。
// 付款码轮询用（对齐 epay orderQuery 返回全量结果供 processNotify）。
func QueryOrder(ctx context.Context, cfg channel.Config, tradeNo string) (map[string]string, error) {
	return Execute(ctx, cfg, APIHost+"/pay/orderquery", map[string]string{"out_trade_no": tradeNo}, false)
}

// Reverse 撤销订单 /secapi/pay/reverse（需商户证书 mTLS，对齐 epay PaymentService::reverse）。
// 付款码轮询超时后撤单，避免掉单（用户后续再扫无法重复扣款）。
func Reverse(ctx context.Context, cfg channel.Config, tradeNo string) error {
	_, err := Execute(ctx, cfg, APIHost+"/secapi/pay/reverse", map[string]string{"out_trade_no": tradeNo}, true)
	return err
}

// ParseNotify 支付结果回调解析（对齐 PaymentService::notify）：解析 XML → 验签 → 判 return_code。
// raw[KeyBody] 为 handler 注入的原始 XML 报文体。返回结果含 AckContent（V2 需回 <xml> 应答）。
func ParseNotify(cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	body := raw[KeyBody]
	if body == "" {
		return channel.NotifyResult{}, fmt.Errorf("回调报文为空")
	}
	data, err := wxpayv2.XMLToMap(body)
	if err != nil {
		return channel.NotifyResult{}, err
	}
	if data["return_code"] != "SUCCESS" {
		return channel.NotifyResult{}, fmt.Errorf("回调 return_code 非 SUCCESS: %s", data["return_msg"])
	}
	if !wxpayv2.CheckSign(data, cfg.Key) {
		return channel.NotifyResult{}, fmt.Errorf("回调签名校验失败")
	}
	res := channel.NotifyResult{
		TradeNo:    data["out_trade_no"],
		ChannelNo:  data["transaction_id"],
		Success:    data["result_code"] == "SUCCESS",
		AckContent: ReplyXML(true, ""),
	}
	if fen := data["total_fee"]; fen != "" {
		res.Money = FenStrToYuan(fen)
	}
	return res, nil
}

// ReplyXML 生成回复微信的应答报文（对齐 replyNotify）。
func ReplyXML(ok bool, msg string) string {
	if ok {
		return "<xml><return_code><![CDATA[SUCCESS]]></return_code><return_msg><![CDATA[OK]]></return_msg></xml>"
	}
	return "<xml><return_code><![CDATA[FAIL]]></return_code><return_msg><![CDATA[" + msg + "]]></return_msg></xml>"
}

// Refund 申请退款 /secapi/pay/refund（需商户证书 mTLS，对齐 PaymentService::refund）。
func Refund(ctx context.Context, cfg channel.Config, req channel.RefundReq) (channel.RefundResp, error) {
	params := map[string]string{
		"out_refund_no": req.OutRefundNo,
		"total_fee":     YuanToFenStr(req.TotalMoney),
		"refund_fee":    YuanToFenStr(req.Money),
	}
	if req.ChannelNo != "" {
		params["transaction_id"] = req.ChannelNo
	} else {
		params["out_trade_no"] = req.TradeNo
	}
	result, err := Execute(ctx, cfg, APIHost+"/secapi/pay/refund", params, true)
	if err != nil {
		return channel.RefundResp{}, err
	}
	return channel.RefundResp{
		RefundNo: result["refund_id"],
		Money:    FenStrToYuan(result["refund_fee"]),
		Success:  true,
	}, nil
}

// httpClient 构造 HTTP 客户端；useCert=true 时加载商户证书做双向 TLS。
func httpClient(cfg channel.Config, useCert bool) (*http.Client, error) {
	if !useCert {
		return &http.Client{Timeout: httpTimeout}, nil
	}
	certPEM := cfg.ExtraOr(extraCertPEM, "")
	keyPEM := cfg.ExtraOr(extraKeyPEM, cfg.PrivateKey)
	if certPEM == "" || keyPEM == "" {
		return nil, fmt.Errorf("退款/撤单需配置商户 API 证书（cert_pem + key_pem）")
	}
	cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		return nil, fmt.Errorf("商户证书加载失败: %w", err)
	}
	return &http.Client{
		Timeout: httpTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{cert}},
		},
	}, nil
}

// YuanToFenStr 元→分字符串（对齐 epay strval($money*100)）。
func YuanToFenStr(yuan decimal.Decimal) string {
	return yuan.Mul(decimal.NewFromInt(100)).Round(0).String()
}

// FenStrToYuan 分字符串→元。
func FenStrToYuan(fen string) decimal.Decimal {
	d, err := decimal.NewFromString(strings.TrimSpace(fen))
	if err != nil {
		return decimal.Zero
	}
	return d.Div(decimal.NewFromInt(100))
}

// BuildAppParams 生成 V2 APP 支付拉起参数并 MD5 签名（对齐 PaymentService::getAppParameters）。
// 字段 appid/partnerid/prepayid/package=Sign=WXPay/noncestr/timestamp，sign 走 MakeSign(MD5)。
func BuildAppParams(cfg channel.Config, prepayID string) (map[string]string, error) {
	nonce, err := wxpayv2.NonceStr(32)
	if err != nil {
		return nil, err
	}
	partnerID := cfg.MchID
	if sub := SubMchID(cfg); sub != "" {
		partnerID = sub
	}
	params := map[string]string{
		"appid":     cfg.AppID,
		"partnerid": partnerID,
		"prepayid":  prepayID,
		"package":   "Sign=WXPay",
		"noncestr":  nonce,
		"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
	}
	params["sign"] = wxpayv2.MakeSign(params, cfg.Key)
	return params, nil
}

// ---- 企业付款（V2 mmpaymkttransfers/mmpaysptrans，对齐 epay wechatpay-sdk TransferService）----
//
// V2 企业付款的公共参数与统一下单不同：仅 nonce_str（到零钱额外带 mch_appid/mchid，到银行卡带 mch_id），
// 均需商户证书 mTLS，且响应不带 sign（execCert 不验应答签）。故不复用 PublicParams/Execute。

// Transfer 企业付款到零钱 /mmpaymkttransfers/promotion/transfers（对齐 TransferService::transfer）。
// req.Account 为收款用户 openid；req.Name 非空则强制实名校验（check_name=FORCE_CHECK）。
func Transfer(ctx context.Context, cfg channel.Config, req channel.TransferReq) (channel.TransferResp, error) {
	if cfg.AppID == "" || cfg.MchID == "" || cfg.Key == "" {
		return channel.TransferResp{}, fmt.Errorf("企业付款缺少 appid/mch_id/apikey")
	}
	if req.Account == "" {
		return channel.TransferResp{}, fmt.Errorf("企业付款到零钱缺少收款用户 openid")
	}
	nonce, err := wxpayv2.NonceStr(32)
	if err != nil {
		return channel.TransferResp{}, err
	}
	params := map[string]string{
		"mch_appid":        cfg.AppID,
		"mchid":            cfg.MchID,
		"nonce_str":        nonce,
		"partner_trade_no": req.OutBizNo,
		"openid":           req.Account,
		"amount":           YuanToFenStr(req.Money),
		"desc":             defaultStr(req.Remark, "企业付款"),
	}
	if req.Name != "" {
		params["check_name"] = "FORCE_CHECK"
		params["re_user_name"] = req.Name
	}
	result, err := execCert(ctx, cfg, APIHost+"/mmpaymkttransfers/promotion/transfers", params)
	if err != nil {
		return channel.TransferResp{}, err
	}
	// 到零钱成功即视为付款成功（同步返回 payment_no/payment_time）。
	return channel.TransferResp{
		TransferNo: result["payment_no"],
		Status:     1,
		Message:    result["payment_time"],
	}, nil
}

// TransferToBank 企业付款到银行卡 /mmpaysptrans/pay_bank（对齐 TransferService::transferToBank）。
// 卡号/姓名用微信 RSA 公钥 OAEP 加密；bank_code 由银行卡号推导。
// RSA 公钥取 cfg.Extra["wx_pubkey"]（PEM），为空则报错提示先获取（对齐 epay publickey_path 文件缓存）。
func TransferToBank(ctx context.Context, cfg channel.Config, req channel.TransferReq, bankCode string) (channel.TransferResp, error) {
	if cfg.MchID == "" || cfg.Key == "" {
		return channel.TransferResp{}, fmt.Errorf("企业付款到银行卡缺少 mch_id/apikey")
	}
	pubPEM := cfg.ExtraOr("wx_pubkey", "")
	if pubPEM == "" {
		return channel.TransferResp{}, fmt.Errorf("企业付款到银行卡缺少 RSA 加密公钥（wx_pubkey），请先在微信商户平台获取")
	}
	encBankNo, err := wxpayv2.RSAEncryptOAEP(req.Account, pubPEM)
	if err != nil {
		return channel.TransferResp{}, fmt.Errorf("银行卡号加密失败: %w", err)
	}
	encName, err := wxpayv2.RSAEncryptOAEP(req.Name, pubPEM)
	if err != nil {
		return channel.TransferResp{}, fmt.Errorf("收款人姓名加密失败: %w", err)
	}
	nonce, err := wxpayv2.NonceStr(32)
	if err != nil {
		return channel.TransferResp{}, err
	}
	params := map[string]string{
		"mch_id":           cfg.MchID,
		"nonce_str":        nonce,
		"partner_trade_no": req.OutBizNo,
		"enc_bank_no":      encBankNo,
		"enc_true_name":    encName,
		"bank_code":        bankCode,
		"amount":           YuanToFenStr(req.Money),
		"desc":             defaultStr(req.Remark, "企业付款"),
	}
	result, err := execCert(ctx, cfg, APIHost+"/mmpaysptrans/pay_bank", params)
	if err != nil {
		return channel.TransferResp{}, err
	}
	// 到银行卡受理成功但到账异步，返回处理中（对齐 epay：需 queryBank 查最终状态）。
	return channel.TransferResp{
		TransferNo: result["payment_no"],
		Status:     0,
		Message:    result["cmms_amt"],
	}, nil
}

// TransferQuery 查询企业付款到零钱状态 /mmpaymkttransfers/gettransferinfo（对齐 transferQuery）。
func TransferQuery(ctx context.Context, cfg channel.Config, outBizNo string) (channel.TransferResp, error) {
	if cfg.AppID == "" || cfg.MchID == "" {
		return channel.TransferResp{}, fmt.Errorf("企业付款查询缺少 appid/mch_id")
	}
	nonce, err := wxpayv2.NonceStr(32)
	if err != nil {
		return channel.TransferResp{}, err
	}
	params := map[string]string{
		"mch_appid":        cfg.AppID,
		"mchid":            cfg.MchID,
		"nonce_str":        nonce,
		"partner_trade_no": outBizNo,
	}
	result, err := execCert(ctx, cfg, APIHost+"/mmpaymkttransfers/gettransferinfo", params)
	if err != nil {
		return channel.TransferResp{}, err
	}
	return channel.TransferResp{
		TransferNo: result["detail_id"],
		Status:     transferBankStatus(result["status"]),
		Message:    result["reason"],
	}, nil
}

// QueryBank 查询企业付款到银行卡状态 /mmpaysptrans/query_bank（对齐 queryBank）。
func QueryBank(ctx context.Context, cfg channel.Config, outBizNo string) (channel.TransferResp, error) {
	if cfg.MchID == "" {
		return channel.TransferResp{}, fmt.Errorf("企业付款到银行卡查询缺少 mch_id")
	}
	nonce, err := wxpayv2.NonceStr(32)
	if err != nil {
		return channel.TransferResp{}, err
	}
	params := map[string]string{
		"mch_id":           cfg.MchID,
		"nonce_str":        nonce,
		"partner_trade_no": outBizNo,
	}
	result, err := execCert(ctx, cfg, APIHost+"/mmpaysptrans/query_bank", params)
	if err != nil {
		return channel.TransferResp{}, err
	}
	return channel.TransferResp{
		TransferNo: result["payment_no"],
		Status:     transferBankStatus(result["status"]),
		Message:    result["reason"],
	}, nil
}

// bankCodeMap 银行英文简码 → 微信企业付款到银行卡 bank_code（对齐 epay wxpay/inc/bankcode.json）。
var bankCodeMap = map[string]string{
	"ICBC": "1002", "ABC": "1005", "CCB": "1003", "BOC": "1026", "COMM": "1020",
	"CMB": "1001", "PSBC": "1066", "CMBC": "1006", "SPABANK": "1010", "CITIC": "1021",
	"SPDB": "1004", "CIB": "1009", "CEB": "1022", "GDB": "1027", "HXBANK": "1025",
	"NBBANK": "1056", "BJBANK": "4836", "SHBANK": "1024", "NJCB": "1054", "RHCZBANK": "4755",
	"CSCB": "4216", "ZJTLCB": "4051", "ZYB": "4753", "IBK": "4761", "SDEB": "4036",
	"HSBK": "4752", "CZCCB": "4756", "DTCBANK": "4767", "HNRCU": "4115", "NXRCU": "4150",
	"SXRCU": "4156", "ARCU": "4166", "GSRCU": "4157", "TRCB": "4153", "GXRCU": "4113",
	"SXRCCU": "4108", "SRCB": "4076", "NBYZ": "4052", "ZJNX": "4764", "JSRCU": "4217",
	"JZRCBANK": "4072", "ZGCBANK": "4769", "DBS": "4778", "ZZYH": "4766", "UBCHN": "4758",
	"NYSYBANK": "4763",
}

// BankCode 按银行英文简码取微信 bank_code（对齐 epay getBankCode）；未知返回空串。
// 调用方（transfer 主链）据卡号 BIN 解析出银行简码后传入。
func BankCode(bankAbbr string) string { return bankCodeMap[strings.ToUpper(bankAbbr)] }

// transferBankStatus 微信付款状态 → 统一 status（0处理中/1成功/2失败，对齐 epay transfer_query）。
func transferBankStatus(status string) int8 {
	switch status {
	case "SUCCESS":
		return 1
	case "FAILED", "BANK_FAIL", "CLOSED":
		return 2
	default: // PROCESSING/WAITING 等
		return 0
	}
}

// execCert V2 企业付款专用请求：merge nonce_str 已由调用方带入，签名后 mTLS POST，
// 判 return_code/result_code=SUCCESS（企业付款响应无 sign，不验应答签，对齐 SDK execute 分支）。
func execCert(ctx context.Context, cfg channel.Config, url string, params map[string]string) (map[string]string, error) {
	params["sign"] = wxpayv2.MakeSign(params, cfg.Key)
	reqXML := wxpayv2.MapToXML(params)
	client, err := httpClient(cfg, true)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(reqXML))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("微信 V2 企业付款请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	result, err := wxpayv2.XMLToMap(string(respBody))
	if err != nil {
		return nil, fmt.Errorf("微信 V2 企业付款应答解析失败: %w (原文 %s)", err, string(respBody))
	}
	if result["return_code"] != "SUCCESS" {
		return nil, fmt.Errorf("微信 V2 企业付款通信失败: %s", result["return_msg"])
	}
	if result["result_code"] != "SUCCESS" {
		return nil, &BizError{ErrCode: result["err_code"], ErrCodeDes: result["err_code_des"], Raw: result}
	}
	return result, nil
}

// MicroPay 付款码支付 /pay/micropay（对齐 PaymentService::microPay）。同步返回支付结果（无异步回调）。
// 成功返回 result_code=SUCCESS + transaction_id/openid；USERPAYING/SYSTEMERROR 需上层轮询查单。
// micropay 无需 notify_url（同步结果），故不复用 BaseOrderParams，直接组必要字段。
func MicroPay(ctx context.Context, cfg channel.Config, req channel.CreateReq) (map[string]string, error) {
	if req.AuthCode == "" {
		return nil, fmt.Errorf("付款码支付缺少 auth_code")
	}
	params := map[string]string{
		"body":             defaultStr(req.Subject, "商品支付"),
		"out_trade_no":     req.TradeNo,
		"total_fee":        YuanToFenStr(req.Money),
		"spbill_create_ip": defaultStr(req.ClientIP, "127.0.0.1"),
		"auth_code":        req.AuthCode,
	}
	return Execute(ctx, cfg, APIHost+"/pay/micropay", params, false)
}

func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// Inputs V2 渠道共用配置字段（元数据驱动后台密钥表单）。
func Inputs() []channel.FieldInput {
	return []channel.FieldInput{
		{Name: "appid", Label: "AppID", Type: "text", Require: true, Tip: "服务号/小程序/开放平台 appid"},
		{Name: "mch_id", Label: "商户号", Type: "text", Require: true, Tip: "微信支付商户号 mch_id"},
		{Name: "api_key", Label: "APIv2 密钥", Type: "password", Require: true, Tip: "商户平台设置的 32 位 APIv2 密钥（用于签名与回调验签）"},
		{Name: "cert_pem", Label: "商户证书", Type: "textarea", Require: false, Tip: "apiclient_cert.pem 内容，退款/撤单需要（双向 TLS）"},
		{Name: "key_pem", Label: "商户证书私钥", Type: "textarea", Require: false, Tip: "apiclient_key.pem 内容，退款/撤单需要"},
		{Name: "sub_mchid", Label: "子商户号", Type: "text", Require: false, Tip: "服务商模式填写特约商户号；直连留空"},
		{Name: "sub_appid", Label: "子商户 AppID", Type: "text", Require: false, Tip: "服务商模式可选子商户 appid"},
		{Name: "notify_url", Label: "回调基址", Type: "text", Require: true, Tip: "系统会自动拼接 /系统订单号 作为微信回调地址"},
	}
}
