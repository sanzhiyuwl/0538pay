// Package fuiou2 实现富友支付（合作方版）渠道，1:1 对齐 epay plugins/fuiou2。
//
// 支持支付宝/微信/云闪付扫码下单（/preCreate 取 qr_code）、异步回调验签入账（req XML）、
// 原路退款（/commonRefund）。协议原语（RSA-MD5 签名、GBK、XML、双重 urlencode）见 pkg/fuiou。
//
// 配置字段（对齐 epay $info['inputs']，经 buildChannelConfigFromKV 落位）：
//   - appid       → 机构号 ins_cd（cfg.AppID）
//   - appmchid    → 商户号 mchnt_cd（Extra）
//   - appsecret   → 商户私钥（Extra，RSA 私钥，请求签名）
//   - appkey      → 富友公钥（Extra，RSA 公钥，应答/回调验签）
//   - appurl      → 订单号前缀（Extra，拼在系统订单号前作为 mchnt_order_no）
//   - appswitch   → 环境：0 生产 / 1 测试（Extra）
//
// key = "fuiou2"。
package fuiou2

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/pkg/fuiou"
)

const (
	gatewayProd = "https://spay-mc.fuioupay.com"
	gatewayTest = "https://fundwx.fuiou.com"
	version     = "1.0"
	termID      = "88888888"
)

type Channel struct{}

func (Channel) Key() string { return "fuiou2" }

// conf 从 Config 解出富友要素。
type fuiouConf struct {
	insCd, mchntCd, privKey, pubKey, prefix, gateway string
}

func resolve(cfg channel.Config) (fuiouConf, error) {
	c := fuiouConf{
		insCd:   cfg.AppID,
		mchntCd: cfg.ExtraOr("appmchid", ""),
		privKey: cfg.ExtraOr("appsecret", cfg.PrivateKey),
		pubKey:  cfg.ExtraOr("appkey", cfg.PublicKey),
		prefix:  cfg.ExtraOr("appurl", ""),
		gateway: gatewayProd,
	}
	if cfg.ExtraOr("appswitch", "0") == "1" {
		c.gateway = gatewayTest
	}
	if c.insCd == "" || c.mchntCd == "" || c.privKey == "" {
		return c, fmt.Errorf("富友渠道缺少机构号/商户号/商户私钥配置")
	}
	return c, nil
}

// orderType 把内部支付方式映射为富友 order_type（对齐 fuiou2 addOrder：ALIPAY/WECHAT/UNIONPAY）。
func orderType(typename string) (string, error) {
	switch typename {
	case "alipay":
		return "ALIPAY", nil
	case "wxpay":
		return "WECHAT", nil
	case "bank":
		return "UNIONPAY", nil
	default:
		return "", fmt.Errorf("富友渠道不支持的支付方式: %s", typename)
	}
}

// Create 扫码下单 /preCreate，取 qr_code 渲染二维码（对齐 fuiou2 addOrder）。
func (Channel) Create(ctx context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	c, err := resolve(cfg)
	if err != nil {
		return channel.CreateResp{}, err
	}
	typename := req.Extra["typename"]
	if typename == "" {
		typename = "alipay"
	}
	ot, err := orderType(typename)
	if err != nil {
		return channel.CreateResp{}, err
	}
	notify := req.NotifyURL
	if notify == "" {
		notify = cfg.NotifyURL
	}
	if notify == "" {
		return channel.CreateResp{}, fmt.Errorf("富友渠道缺少 notify_url 回调地址")
	}
	params := map[string]string{
		"order_type":     ot,
		"order_amt":      fenStr(req.Money),
		"mchnt_order_no": c.prefix + req.TradeNo,
		"txn_begin_ts":   nowYmdHis(),
		"goods_des":      defaultStr(req.Subject, "商品支付"),
		"term_ip":        defaultStr(req.ClientIP, "127.0.0.1"),
		"notify_url":     notify,
		"curr_type":      "CNY",
	}
	result, err := c.execute(ctx, "/preCreate", params)
	if err != nil {
		return channel.CreateResp{}, err
	}
	qr := result["qr_code"]
	if qr == "" {
		return channel.CreateResp{}, fmt.Errorf("富友下单未返回 qr_code")
	}
	return channel.CreateResp{
		PayType: channel.PayTypeQRCode,
		QRCode:  qr,
		PayURL:  qr,
	}, nil
}

// Query 主动查单：富友无独立 orderQuery，暂以回调/后续查单接口为准，这里返回未支付（对齐 epay 无查单实现）。
func (Channel) Query(_ context.Context, _ channel.Config, _ string) (bool, error) {
	return false, nil
}

// Notify 异步回调验签 + 判 result_code=000000（对齐 fuiou2 notify）。
// raw[RawBody] 为原始 body（req=urlencode(urlencode(xml))）；富友 POST 表单键 req 已由 handler 收进 params["req"]。
func (Channel) Notify(_ context.Context, cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	c, err := resolve(cfg)
	if err != nil {
		return channel.NotifyResult{}, err
	}
	// 富友回调 body = req=urlencode(urlencode(xml))。handler 的 ParseForm 解一层，
	// 这里再解一层还原出原始 XML（对齐 epay urldecode($_POST['req'])）。
	reqXML := raw["req"]
	if reqXML == "" {
		return channel.NotifyResult{}, fmt.Errorf("富友回调缺少 req 参数")
	}
	if dec, derr := url.QueryUnescape(reqXML); derr == nil {
		reqXML = dec
	}
	data, err := fuiou.ParseResponseXML(reqXML)
	if err != nil {
		return channel.NotifyResult{}, err
	}
	if !fuiou.VerifyMap(data, c.pubKey) {
		return channel.NotifyResult{}, fmt.Errorf("富友回调验签失败")
	}
	success := data["result_code"] == "000000"
	// 去订单号前缀还原系统订单号（对齐 substr($mchnt_order_no, strlen($prefix))）。
	tradeNo := strings.TrimPrefix(data["mchnt_order_no"], c.prefix)
	res := channel.NotifyResult{
		TradeNo:    tradeNo,
		ChannelNo:  data["transaction_id"],
		Success:    success,
		AckContent: "1", // 富友要求回 "1" 确认（对齐 return ['type'=>'html','data'=>'1']）
	}
	if amt := data["order_amt"]; amt != "" {
		res.Money = fenToYuan(amt)
	}
	return res, nil
}

// Refund 原路退款 /commonRefund（对齐 fuiou2 refund）。
func (Channel) Refund(ctx context.Context, cfg channel.Config, req channel.RefundReq) (channel.RefundResp, error) {
	c, err := resolve(cfg)
	if err != nil {
		return channel.RefundResp{}, err
	}
	ot, err := orderType(req.TypeName)
	if err != nil {
		// typename 缺省时默认按支付宝，不阻断（对齐 epay 按 $order['type'] 映射）。
		ot = "ALIPAY"
	}
	params := map[string]string{
		"mchnt_order_no":  c.prefix + req.TradeNo,
		"refund_order_no": req.OutRefundNo,
		"order_type":      ot,
		"total_amt":       fenStr(req.TotalMoney),
		"refund_amt":      fenStr(req.Money),
	}
	result, err := c.execute(ctx, "/commonRefund", params)
	if err != nil {
		return channel.RefundResp{}, err
	}
	return channel.RefundResp{
		RefundNo: result["mchnt_order_no"],
		Money:    req.Money,
		Success:  true,
	}, nil
}

func (Channel) Inputs() []channel.FieldInput {
	return []channel.FieldInput{
		{Name: "appid", Label: "机构号", Type: "text", Require: true, Tip: "富友分配的机构号 ins_cd"},
		{Name: "appmchid", Label: "商户号", Type: "text", Require: true, Tip: "富友商户号 mchnt_cd"},
		{Name: "appsecret", Label: "商户私钥", Type: "textarea", Require: true, Tip: "商户 RSA 私钥（请求签名，单行 base64 或 PEM）"},
		{Name: "appkey", Label: "富友公钥", Type: "textarea", Require: true, Tip: "富友平台 RSA 公钥（应答/回调验签）"},
		{Name: "appurl", Label: "订单号前缀", Type: "text", Require: false, Tip: "拼在系统订单号前作为富友商户订单号（可空）"},
		{Name: "appswitch", Label: "环境", Type: "select", Options: []string{"生产环境", "测试环境"}, Require: false, Tip: "0 生产 / 1 测试"},
		{Name: "notify_url", Label: "回调基址", Type: "text", Require: true, Tip: "系统会自动拼接 /系统订单号 作为富友回调地址"},
	}
}

func (Channel) Products() []channel.ProductType {
	return []channel.ProductType{
		{Code: "alipay_qr", Name: "支付宝扫码", Group: "alipay"},
		{Code: "wxpay_qr", Name: "微信扫码", Group: "wxpay"},
		{Code: "bank_qr", Name: "云闪付扫码", Group: "bank"},
	}
}

func init() { channel.Register(Channel{}) }
