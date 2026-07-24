package epayn

import (
	"context"
	"testing"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/pkg/sign"
)

// buildCfg 造一个带平台公私钥的 config（平台私钥用于模拟上游签回调）。
func buildCfg(t *testing.T) (channel.Config, string) {
	t.Helper()
	// 上游平台密钥对：私钥模拟上游签回调，公钥配到本站验签
	platPriv, platPub, err := sign.GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("生成平台密钥失败: %v", err)
	}
	// 本站商户密钥对（下单签名用；测试 Notify 用不到，占位即可）
	merPriv, _, _ := sign.GenerateRSAKeyPair()
	cfg := channel.Config{
		AppID: "1",
		Extra: map[string]string{
			"appurl":                "http://127.0.0.1:8080",
			"platform_public_key":   platPub,
			"merchant_private_key":  merPriv,
		},
	}
	return cfg, platPriv
}

// TestNotifySuccess 上游用平台私钥签的成功回调，应验签通过并判定成功。
func TestNotifySuccess(t *testing.T) {
	cfg, platPriv := buildCfg(t)
	raw := map[string]string{
		"pid": "1", "trade_no": "UP20260721001", "out_trade_no": "V2ORDER001",
		"type": "alipay", "name": "VIP", "money": "1.00",
		"trade_status": "TRADE_SUCCESS", "timestamp": "1784600000",
	}
	sig, err := sign.MakeRSA(raw, platPriv)
	if err != nil {
		t.Fatalf("签名失败: %v", err)
	}
	raw["sign"] = sig
	raw["sign_type"] = "RSA"

	nr, err := Channel{}.Notify(context.Background(), cfg, raw)
	if err != nil {
		t.Fatalf("回调验签应通过: %v", err)
	}
	if !nr.Success || nr.TradeNo != "V2ORDER001" || nr.ChannelNo != "UP20260721001" {
		t.Fatalf("回调解析不符: %+v", nr)
	}
}

// TestNotifyTamperFails 篡改金额后回调验签应失败。
func TestNotifyTamperFails(t *testing.T) {
	cfg, platPriv := buildCfg(t)
	raw := map[string]string{
		"out_trade_no": "V2ORDER002", "money": "1.00", "trade_status": "TRADE_SUCCESS",
	}
	raw["sign"], _ = sign.MakeRSA(raw, platPriv)
	raw["sign_type"] = "RSA"
	raw["money"] = "9999.00" // 篡改
	if _, err := (Channel{}).Notify(context.Background(), cfg, raw); err == nil {
		t.Fatal("篡改金额后验签应失败")
	}
}

// TestConfMissing 缺配置应报错。
func TestConfMissing(t *testing.T) {
	_, err := (Channel{}).Notify(context.Background(), channel.Config{Extra: map[string]string{}}, map[string]string{})
	if err == nil {
		t.Fatal("缺配置应报错")
	}
}
