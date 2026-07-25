package fuiou

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// genKeys 生成一对 RSA 密钥，返回 epay 单行 base64 格式（PKCS#8 私钥 / PKIX 公钥）。
func genKeys(t *testing.T) (priv, pub string) {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privDER, _ := x509.MarshalPKCS8PrivateKey(k)
	pubDER, _ := x509.MarshalPKIXPublicKey(&k.PublicKey)
	return base64.StdEncoding.EncodeToString(privDER), base64.StdEncoding.EncodeToString(pubDER)
}

// TestSignContent 对齐 getSignContent：ksort、跳过 sign 与 reserved* 前缀、k=v& 去尾。
func TestSignContent(t *testing.T) {
	params := map[string]string{
		"version":         "1.0",
		"ins_cd":          "08M0",
		"mchnt_cd":        "0002",
		"sign":            "SHOULD_SKIP",
		"reserved_fy_no":  "SHOULD_SKIP",
		"reserved_extra":  "SHOULD_SKIP",
		"order_amt":       "100",
	}
	got := SignContent(params)
	want := "ins_cd=08M0&mchnt_cd=0002&order_amt=100&version=1.0"
	if got != want {
		t.Fatalf("待签名串不对:\n got=%s\nwant=%s", got, want)
	}
}

// TestSignVerifyRoundTrip 商户私钥 RSA(MD5) 签名 → 富友公钥验签闭环。
func TestSignVerifyRoundTrip(t *testing.T) {
	priv, pub := genKeys(t)
	content := "ins_cd=08M0&mchnt_cd=0002&order_amt=100"
	sig, err := Sign(content, priv)
	if err != nil {
		t.Fatalf("签名失败: %v", err)
	}
	if !Verify(content, sig, pub) {
		t.Fatal("自签自验应通过")
	}
	if Verify(content+"x", sig, pub) {
		t.Fatal("内容篡改后验签应失败")
	}
	if Verify(content, "bad-base64!!", pub) {
		t.Fatal("非法签名应验签失败")
	}
	if Verify(content, "", pub) {
		t.Fatal("空签名应验签失败")
	}
}

// TestVerifyMap 端到端：构造一份 UTF-8 map（含中文）→ 用 GBK 值签名 → VerifyMap 应通过。
func TestVerifyMap(t *testing.T) {
	priv, pub := genKeys(t)
	params := map[string]string{
		"result_code":    "000000",
		"mchnt_order_no": "T20260725",
		"transaction_id": "fy-txn-1",
		"result_msg":     "交易成功", // 含中文，验证 GBK 还原路径
	}
	// 模拟富友：值 UTF-8→GBK 后签名
	gbk := map[string]string{}
	for k, v := range params {
		gbk[k] = ToGBK(v)
	}
	sig, err := Sign(SignContent(gbk), priv)
	if err != nil {
		t.Fatal(err)
	}
	params["sign"] = sig
	if !VerifyMap(params, pub) {
		t.Fatal("含中文报文的 VerifyMap 应通过")
	}
	params["result_msg"] = "被篡改"
	if VerifyMap(params, pub) {
		t.Fatal("篡改后 VerifyMap 应失败")
	}
}

func TestGBKRoundTrip(t *testing.T) {
	s := "支付宝扫码123ABC"
	gbk := ToGBK(s)
	// GBK 编码后应与直接用 encoder 一致
	want, _ := simplifiedchinese.GBK.NewEncoder().String(s)
	if gbk != want {
		t.Fatal("ToGBK 结果不一致")
	}
	if FromGBK(gbk) != s {
		t.Fatalf("GBK 往返不一致: got=%s want=%s", FromGBK(gbk), s)
	}
}

// TestBuildRequestBody 组装请求体：以 req= 开头、双重 urlencode（原始 XML 里的 < 变成 %253C）。
func TestBuildRequestBody(t *testing.T) {
	priv, _ := genKeys(t)
	params := map[string]string{
		"version":        "1.0",
		"ins_cd":         "08M0",
		"mchnt_cd":       "0002",
		"order_amt":      "100",
		"goods_des":      "测试商品",
	}
	body, err := BuildRequestBody(params, priv)
	if err != nil {
		t.Fatalf("组装失败: %v", err)
	}
	if !strings.HasPrefix(body, "req=") {
		t.Fatalf("body 应以 req= 开头: %s", body[:20])
	}
	// 双重编码：'<' → 第一次 %3C → 第二次 %253C
	if !strings.Contains(body, "%253C") {
		t.Fatalf("应双重 urlencode（含 %%253C）: %s", body[:60])
	}
}

// TestParseResponseXML 解析平铺富友应答 XML → map，忽略 <?xml?> 声明与 xml 包裹。
func TestParseResponseXML(t *testing.T) {
	raw := `<?xml version="1.0" encoding="GBK" standalone="yes"?><xml><result_code>000000</result_code><qr_code>https://qr.fuiou.com/abc</qr_code><mchnt_order_no>T1</mchnt_order_no></xml>`
	m, err := ParseResponseXML(raw)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if m["result_code"] != "000000" || m["qr_code"] != "https://qr.fuiou.com/abc" || m["mchnt_order_no"] != "T1" {
		t.Fatalf("解析结果不对: %v", m)
	}
	if _, err := ParseResponseXML(""); err == nil {
		t.Fatal("空应答应报错")
	}
}
