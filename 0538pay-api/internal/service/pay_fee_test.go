package service

import (
	"testing"

	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/pkg/sign"
	"github.com/shopspring/decimal"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// TestCalcFeeMode0 平台代收 mode=0：getmoney=money*rate/100，realmoney=money（无随机时）。
func TestCalcFeeMode0(t *testing.T) {
	s := &PayService{} // cfg=nil → 无兜底/无随机
	m := &model.Merchant{Mode: 0, Money: dec("1000")}
	get, real := s.calcFee(m, dec("100"), dec("2"))
	if !get.Equal(dec("2")) {
		t.Errorf("mode0 getmoney=%s 期望 2", get)
	}
	if !real.Equal(dec("100")) {
		t.Errorf("mode0 realmoney=%s 期望 100", real)
	}
}

// TestCalcFeeMode1 商户直清 mode=1：realmoney=money*(200-rate)/100（买家加费），getmoney=money。
// 对齐 epay Pay.php:142-143：realmoney=round(money*(100+100-rate)/100,2)。
func TestCalcFeeMode1(t *testing.T) {
	s := &PayService{}
	m := &model.Merchant{Mode: 1, Money: dec("1000")}
	// rate=2 → realmoney=100*(198)/100=198? 不对：应为 100*(200-2)/100=198。买家多付 98 手续费。
	get, real := s.calcFee(m, dec("100"), dec("2"))
	if !get.Equal(dec("100")) {
		t.Errorf("mode1 getmoney=%s 期望 100(商户全额到账)", get)
	}
	if !real.Equal(dec("198")) {
		t.Errorf("mode1 realmoney=%s 期望 198(money*(200-2)/100)", real)
	}
}

// TestCalcInternalFee B1-53：内部订单按 tid 维度加费（对齐 epay submit2.php:35
// `if(mode==1 && tid!=4 || tid==2)`）。money=100,rate=2。
func TestCalcInternalFee(t *testing.T) {
	s := &PayService{}
	// tid=2 余额充值：无论商户 mode 都加费 → realmoney=198(买家多付), getmoney=100(全额到账)。
	for _, mode := range []int8{0, 1} {
		m := &model.Merchant{Mode: mode}
		get, real := s.calcInternalFee(m, 2, dec("100"), dec("2"))
		if !get.Equal(dec("100")) || !real.Equal(dec("198")) {
			t.Errorf("tid=2 mode=%d get=%s real=%s 期望 100/198(强制加费)", mode, get, real)
		}
	}
	// tid=4 购买用户组：即便 mode==1 也强制平台代收（不加费）→ realmoney=100, getmoney=2。
	get, real := s.calcInternalFee(&model.Merchant{Mode: 1}, 4, dec("100"), dec("2"))
	if !get.Equal(dec("2")) || !real.Equal(dec("100")) {
		t.Errorf("tid=4 mode=1 get=%s real=%s 期望 2/100(强制代收,排除加费)", get, real)
	}
	// tid=3 聚合收款：按商户 mode。mode=1 → 加费(198/100)；mode=0 → 代收(100/2)。
	g1, r1 := s.calcInternalFee(&model.Merchant{Mode: 1}, 3, dec("100"), dec("2"))
	if !g1.Equal(dec("100")) || !r1.Equal(dec("198")) {
		t.Errorf("tid=3 mode=1 get=%s real=%s 期望 100/198(随商户加费)", g1, r1)
	}
	g0, r0 := s.calcInternalFee(&model.Merchant{Mode: 0}, 3, dec("100"), dec("2"))
	if !g0.Equal(dec("2")) || !r0.Equal(dec("100")) {
		t.Errorf("tid=3 mode=0 get=%s real=%s 期望 2/100(随商户代收)", g0, r0)
	}
	// m=nil（收款商户查不到）→ 按 mode=0 代收安全默认。
	gn, rn := s.calcInternalFee(nil, 3, dec("100"), dec("2"))
	if !gn.Equal(dec("2")) || !rn.Equal(dec("100")) {
		t.Errorf("tid=3 m=nil get=%s real=%s 期望 2/100(安全默认代收)", gn, rn)
	}
}

// TestParseInternalParam B1-46：内部订单 param JSON 解析 uid/gid/endtime，兼容数字与字符串编码。
func TestParseInternalParam(t *testing.T) {
	// 数字编码
	if got := parseParamUID(`{"uid":1000}`); got != 1000 {
		t.Errorf("uid 数字编码 got=%d 期望 1000", got)
	}
	// 字符串编码
	if got := parseParamUID(`{"uid":"1001"}`); got != 1001 {
		t.Errorf("uid 字符串编码 got=%d 期望 1001", got)
	}
	// 空/非法 → 0
	if got := parseParamUID(``); got != 0 {
		t.Errorf("空 param got=%d 期望 0", got)
	}
	if got := parseParamUID(`not json`); got != 0 {
		t.Errorf("非法 param got=%d 期望 0", got)
	}
	// endtime 解析
	if parseParamEndTime("") != nil {
		t.Error("空 endtime 期望 nil(永久组)")
	}
	if tm := parseParamEndTime("2026-08-01"); tm == nil || tm.Year() != 2026 {
		t.Errorf("endtime 日期解析失败: %v", tm)
	}
	if tm := parseParamEndTime("2026-08-01 12:30:00"); tm == nil || tm.Hour() != 12 {
		t.Errorf("endtime 时间解析失败: %v", tm)
	}
}

// TestCheckChannelPayLimit B1-01/B1-56：选定通道后单笔限额硬拒绝（对齐 epay Pay.php:170-174）。
func TestCheckChannelPayLimit(t *testing.T) {
	// 区间 [10,100]
	if err := checkChannelPayLimit("10", "100", dec("50")); err != nil {
		t.Errorf("50 在 [10,100] 内应放行，得 %v", err)
	}
	// 低于最小 → 拒
	if err := checkChannelPayLimit("10", "100", dec("5")); err == nil {
		t.Error("5 < 最小10 应拒绝")
	}
	// 高于最大 → 拒
	if err := checkChannelPayLimit("10", "100", dec("200")); err == nil {
		t.Error("200 > 最大100 应拒绝")
	}
	// 边界值放行（epay 用 < / > 严格比较，等于不拒）
	if err := checkChannelPayLimit("10", "100", dec("10")); err != nil {
		t.Errorf("等于最小10 应放行，得 %v", err)
	}
	if err := checkChannelPayLimit("10", "100", dec("100")); err != nil {
		t.Errorf("等于最大100 应放行，得 %v", err)
	}
	// 空/0 限额 → 该侧不限制
	if err := checkChannelPayLimit("", "", dec("99999")); err != nil {
		t.Errorf("空限额应不限制，得 %v", err)
	}
	if err := checkChannelPayLimit("0", "0", dec("99999")); err != nil {
		t.Errorf("0 限额应不限制，得 %v", err)
	}
	// money<=0 不校验
	if err := checkChannelPayLimit("10", "100", dec("0")); err != nil {
		t.Errorf("money=0 不校验，得 %v", err)
	}
}

// TestTruncateRunes A-11：按 UTF-8 边界截断，不切碎中文。
func TestTruncateRunes(t *testing.T) {
	// 42 个中文 = 126 字节 ≤127，全保留
	s42 := ""
	for i := 0; i < 42; i++ {
		s42 += "中"
	}
	if got := truncateRunes(s42, 127); got != s42 {
		t.Errorf("126字节应全保留，得 %d 字节", len(got))
	}
	// 43 个中文 = 129 字节 >127，截到 42 个（126 字节），不产生半个字符
	s43 := s42 + "文"
	got := truncateRunes(s43, 127)
	if len(got) != 126 {
		t.Errorf("截断后应 126 字节，得 %d", len(got))
	}
	// 截断结果必须是合法 UTF-8（无半个字符）
	for _, r := range got {
		if r == '�' {
			t.Error("截断切碎了中文字符（出现替换符）")
		}
	}
}

// TestCalcRefundReduce 退款 reducemoney 四分支（对齐 epay Order::refund）。
// channels=nil + Channel=0 → channelDirect=false，专测非直清的费率分支（不再按比例折算）。
func TestCalcRefundReduce(t *testing.T) {
	real := dec("100")   // 实付
	getMoney := dec("98") // 商户实得
	base := &model.Order{Channel: 0, GetMoney: getMoney}

	// refund_fee_type=0(平台承担)
	s0 := &MapiService{cfg: &ConfigService{cache: map[string]string{"refund_fee_type": "0"}}}
	// 全额退 → 扣 getmoney
	if got := s0.calcRefundReduce(base, dec("100"), real); !got.Equal(getMoney) {
		t.Errorf("平台承担全额退 reduce=%s 期望 98(getmoney)", got)
	}
	// 部分退且 ≥getmoney → 扣 getmoney
	if got := s0.calcRefundReduce(base, dec("99"), real); !got.Equal(getMoney) {
		t.Errorf("平台承担部分退99(≥getmoney) reduce=%s 期望 98(getmoney)", got)
	}
	// 部分退且 <getmoney → 按退款额扣（关键：不再按比例折算）
	if got := s0.calcRefundReduce(base, dec("30"), real); !got.Equal(dec("30")) {
		t.Errorf("平台承担部分退30(<getmoney) reduce=%s 期望 30(按退款额,非折算)", got)
	}

	// refund_fee_type=1(商户承担手续费)
	s1 := &MapiService{cfg: &ConfigService{cache: map[string]string{"refund_fee_type": "1"}}}
	// 全额退 → 扣 realmoney
	if got := s1.calcRefundReduce(base, dec("100"), real); !got.Equal(real) {
		t.Errorf("商户承担全额退 reduce=%s 期望 100(realmoney)", got)
	}
	// 部分退 → 按退款额扣
	if got := s1.calcRefundReduce(base, dec("30"), real); !got.Equal(dec("30")) {
		t.Errorf("商户承担部分退30 reduce=%s 期望 30", got)
	}

	// status=3(已冻结) → 0（不扣）
	frozen := &model.Order{Channel: 0, GetMoney: getMoney, Status: 3}
	if got := s0.calcRefundReduce(frozen, dec("100"), real); !got.IsZero() {
		t.Errorf("已冻结单 reduce=%s 期望 0", got)
	}
}

// TestGroupConfStr 组配置取值：空值/nil 视为"未设置"(ok=false)，数字/布尔转字符串。
// 这是邀请返现"上级组覆盖全局"只覆盖非空键的关键（对齐 epay getGroupConfig isNullOrEmpty 跳过）。
func TestGroupConfStr(t *testing.T) {
	m := map[string]interface{}{
		"invite_rate":       "1.5",
		"invite_open":       float64(1),
		"invite_order_fee":  true,
		"invite_order_type": "",  // 空串 → 未设置
		"empty_num":         nil, // nil → 未设置
	}
	if v, ok := groupConfStr(m, "invite_rate"); !ok || v != "1.5" {
		t.Errorf("invite_rate 应为 (1.5,true) 得 (%s,%v)", v, ok)
	}
	if v, ok := groupConfStr(m, "invite_open"); !ok || v != "1" {
		t.Errorf("invite_open(float64 1) 应为 (1,true) 得 (%s,%v)", v, ok)
	}
	if v, ok := groupConfStr(m, "invite_order_fee"); !ok || v != "1" {
		t.Errorf("invite_order_fee(true) 应为 (1,true) 得 (%s,%v)", v, ok)
	}
	if _, ok := groupConfStr(m, "invite_order_type"); ok {
		t.Error("空串键应视为未设置(ok=false)")
	}
	if _, ok := groupConfStr(m, "empty_num"); ok {
		t.Error("nil 键应视为未设置(ok=false)")
	}
	if _, ok := groupConfStr(m, "not_exist"); ok {
		t.Error("不存在的键应 ok=false")
	}
}

// TestOrderVersion A-1：_version=1 → V2(1)，否则 V1(0)。
func TestOrderVersion(t *testing.T) {
	if orderVersion(map[string]string{"_version": "1"}) != 1 {
		t.Error("_version=1 应判为 V2(1)")
	}
	if orderVersion(map[string]string{}) != 0 {
		t.Error("无 _version 应判为 V1(0)")
	}
}

// TestBuildCallbackV1MD5 A-1：version=0 用商户 key MD5 签，sign_type=MD5。
func TestBuildCallbackV1MD5(t *testing.T) {
	s := &PayService{} // cfg=nil
	o := &model.Order{Version: 0, UID: 1, TradeNo: "T1", OutTradeNo: "O1", TypeName: "alipay", Name: "商品", Money: dec("100")}
	m := &model.Merchant{UID: 1, AppKey: "testkey123"}
	p := s.buildCallbackParams(o, m)
	if p["sign_type"] != "MD5" {
		t.Errorf("V1 sign_type=%s 期望 MD5", p["sign_type"])
	}
	if !sign.VerifyMD5(p, m.AppKey) {
		t.Error("V1 回调 MD5 签名应能被商户 key 验证通过")
	}
}

// TestBuildCallbackV2RSA A-1：version=1 且配置了平台私钥 → 平台私钥 RSA 签 + timestamp，
// 商户用平台公钥验签通过（这是原缺口：之前恒 MD5，V2 商户验签必失败）。
func TestBuildCallbackV2RSA(t *testing.T) {
	priv, pub, err := sign.GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}
	// 构造带平台私钥的 config（同包可直接填 cache）
	cfg := &ConfigService{cache: map[string]string{keySysRSAPrivate: priv, keySysRSAPublic: pub}}
	s := &PayService{cfg: cfg}
	o := &model.Order{Version: 1, UID: 1, TradeNo: "T2", OutTradeNo: "O2", TypeName: "alipay", Name: "商品", Money: dec("100")}
	m := &model.Merchant{UID: 1, AppKey: "irrelevant"}
	p := s.buildCallbackParams(o, m)
	if p["sign_type"] != "RSA" {
		t.Fatalf("V2 sign_type=%s 期望 RSA", p["sign_type"])
	}
	if p["timestamp"] == "" {
		t.Error("V2 回调应带 timestamp")
	}
	// 商户用平台公钥验签（对齐 epay：平台私钥签→商户平台公钥验）
	if !sign.VerifyRSA(p, pub) {
		t.Error("V2 回调 RSA 签名应能被平台公钥验证通过（原缺口修复验证）")
	}
}
