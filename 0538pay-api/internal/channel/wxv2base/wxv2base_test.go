package wxv2base

import (
	"testing"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/pkg/wxpayv2"
	"github.com/shopspring/decimal"
)

// TestPublicParamsDirect 直连公共参数：appid/mch_id/nonce_str/sign_type，无 sub_mch_id。
func TestPublicParamsDirect(t *testing.T) {
	cfg := channel.Config{AppID: "wx1", MchID: "160", Key: "k"}
	p, err := PublicParams(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if p["appid"] != "wx1" || p["mch_id"] != "160" || p["sign_type"] != "MD5" {
		t.Fatalf("公共参数不对: %v", p)
	}
	if p["nonce_str"] == "" {
		t.Fatal("应有 nonce_str")
	}
	if _, ok := p["sub_mch_id"]; ok {
		t.Fatal("直连不应带 sub_mch_id")
	}
}

// TestPublicParamsPartner 服务商公共参数：追加 sub_mch_id / sub_appid。
func TestPublicParamsPartner(t *testing.T) {
	cfg := channel.Config{
		AppID: "sp1", MchID: "sp160", Key: "k",
		Extra: map[string]string{"sub_mchid": "sub200", "sub_appid": "subapp"},
	}
	if !IsPartner(cfg) {
		t.Fatal("配了 sub_mchid 应判服务商模式")
	}
	p, err := PublicParams(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if p["sub_mch_id"] != "sub200" || p["sub_appid"] != "subapp" {
		t.Fatalf("服务商公共参数不对: %v", p)
	}
}

func TestPublicParamsMissing(t *testing.T) {
	if _, err := PublicParams(channel.Config{AppID: "x", MchID: "y"}); err == nil {
		t.Fatal("缺 apikey 应报错")
	}
}

func TestBaseOrderParams(t *testing.T) {
	cfg := channel.Config{NotifyURL: "https://x/n"}
	p, err := BaseOrderParams(cfg, channel.CreateReq{
		TradeNo: "T1", Money: decimal.RequireFromString("1.50"), Subject: "商品", ClientIP: "1.2.3.4",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p["out_trade_no"] != "T1" || p["total_fee"] != "150" || p["body"] != "商品" || p["spbill_create_ip"] != "1.2.3.4" {
		t.Fatalf("下单业务参数不对: %v", p)
	}
	// 缺 notify_url 报错
	if _, err := BaseOrderParams(channel.Config{}, channel.CreateReq{TradeNo: "T2"}); err == nil {
		t.Fatal("缺 notify_url 应报错")
	}
}

func TestYuanFenConv(t *testing.T) {
	if YuanToFenStr(decimal.RequireFromString("88.88")) != "8888" {
		t.Fatal("元转分错误")
	}
	if !FenStrToYuan("8888").Equal(decimal.RequireFromString("88.88")) {
		t.Fatal("分转元错误")
	}
	if !FenStrToYuan("bad").Equal(decimal.Zero) {
		t.Fatal("非法分应返回 0")
	}
}

// TestParseNotifySuccess 端到端：自签一份支付成功回调 XML，ParseNotify 验签通过并解出字段。
func TestParseNotifySuccess(t *testing.T) {
	key := "01234567890123456789012345678901"
	cfg := channel.Config{Key: key}
	data := map[string]string{
		"return_code":    "SUCCESS",
		"result_code":    "SUCCESS",
		"out_trade_no":   "T20260725",
		"transaction_id": "wx-txn-9",
		"total_fee":      "10000",
		"openid":         "o123",
	}
	data["sign"] = wxpayv2.MakeSign(data, key)
	body := wxpayv2.MapToXML(data)

	res, err := ParseNotify(cfg, map[string]string{KeyBody: body})
	if err != nil {
		t.Fatalf("解析成功回调失败: %v", err)
	}
	if !res.Success {
		t.Fatal("result_code=SUCCESS 应判成功")
	}
	if res.TradeNo != "T20260725" || res.ChannelNo != "wx-txn-9" {
		t.Fatalf("字段不对: %+v", res)
	}
	if !res.Money.Equal(decimal.RequireFromString("100")) {
		t.Fatalf("金额应为 100 元, 实际 %s", res.Money)
	}
	if res.AckContent == "" {
		t.Fatal("V2 回调应回 <xml> 应答")
	}
}

// TestParseNotifyBadSign 篡改后验签失败。
func TestParseNotifyBadSign(t *testing.T) {
	key := "01234567890123456789012345678901"
	data := map[string]string{"return_code": "SUCCESS", "result_code": "SUCCESS", "out_trade_no": "T1", "total_fee": "100"}
	data["sign"] = wxpayv2.MakeSign(data, key)
	data["total_fee"] = "999" // 篡改
	body := wxpayv2.MapToXML(data)
	if _, err := ParseNotify(channel.Config{Key: key}, map[string]string{KeyBody: body}); err == nil {
		t.Fatal("篡改后应验签失败")
	}
}

func TestParseNotifyEmpty(t *testing.T) {
	if _, err := ParseNotify(channel.Config{Key: "k"}, map[string]string{}); err == nil {
		t.Fatal("空报文应报错")
	}
}

func TestReplyXML(t *testing.T) {
	if ReplyXML(true, "") == "" || ReplyXML(false, "err") == "" {
		t.Fatal("应答报文不应为空")
	}
}

// TestBankCode 银行简码 → 微信 bank_code 映射（对齐 epay bankcode.json）。
func TestBankCode(t *testing.T) {
	if BankCode("ICBC") != "1002" || BankCode("CMB") != "1001" || BankCode("BOC") != "1026" {
		t.Fatal("银行编码映射不对")
	}
	if BankCode("icbc") != "1002" {
		t.Fatal("银行简码应大小写不敏感")
	}
	if BankCode("NOTEXIST") != "" {
		t.Fatal("未知银行应返回空串")
	}
}

// TestBizErrorMessage 业务错误码格式化。
func TestBizErrorMessage(t *testing.T) {
	e := &BizError{ErrCode: "USERPAYING", ErrCodeDes: "用户支付中"}
	if e.Error() != "USERPAYING 用户支付中" {
		t.Fatalf("错误信息不对: %s", e.Error())
	}
	e2 := &BizError{ErrCode: "SYSTEMERROR"}
	if e2.Error() != "SYSTEMERROR" {
		t.Fatalf("无描述时应只回错误码: %s", e2.Error())
	}
}

// TestBuildAppParams V2 APP 拉起参数：字段齐全 + sign 为自洽 MD5（对齐 getAppParameters）。
func TestBuildAppParams(t *testing.T) {
	cfg := channel.Config{AppID: "wxapp1", MchID: "160", Key: "01234567890123456789012345678901"}
	p, err := BuildAppParams(cfg, "prepay_abc")
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"appid", "partnerid", "prepayid", "package", "noncestr", "timestamp", "sign"} {
		if p[k] == "" {
			t.Fatalf("缺字段 %s: %v", k, p)
		}
	}
	if p["partnerid"] != "160" || p["prepayid"] != "prepay_abc" || p["package"] != "Sign=WXPay" {
		t.Fatalf("字段值不对: %v", p)
	}
	// sign 自洽校验
	if !wxpayv2.CheckSign(p, cfg.Key) {
		t.Fatal("APP 参数 sign 自校验应通过")
	}
	// 服务商模式 partnerid 用子商户号
	cfg2 := channel.Config{AppID: "sp", MchID: "sp160", Key: "k", Extra: map[string]string{"sub_mchid": "sub9"}}
	p2, _ := BuildAppParams(cfg2, "pp")
	if p2["partnerid"] != "sub9" {
		t.Fatalf("服务商 partnerid 应为子商户号, 实际 %s", p2["partnerid"])
	}
}
