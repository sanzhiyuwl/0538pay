package service

import (
	"testing"

	"github.com/shopspring/decimal"
)

// TestSplitProfit P0-a：多接收方 | 拆分 + 逐接收方独立费率，1:1 对齐 epay Wxpay.php submit()。
//   - account/name/rate 均以 | 分隔，接收方个数以 account 段数为准。
//   - rate 段缺省复用首段（$rates[0] 兜底）。
//   - 每接收方金额 = round(floor(realmoney*rate)/100, 2)（向下取整到分）。
func TestSplitProfit(t *testing.T) {
	rm := func(s string) decimal.Decimal { return decimal.RequireFromString(s) }

	t.Run("单接收方", func(t *testing.T) {
		items := splitProfit("2088001", "渠道A", "10", rm("100"))
		if len(items) != 1 {
			t.Fatalf("接收方数=%d 期望1", len(items))
		}
		if !items[0].Money.Equal(rm("10")) {
			t.Errorf("金额=%s 期望10", items[0].Money)
		}
	})

	t.Run("多接收方逐段独立费率", func(t *testing.T) {
		// realmoney=200, rate 30|20 → 60 / 40
		items := splitProfit("2088001|oABC", "渠道A|个人B", "30|20", rm("200"))
		if len(items) != 2 {
			t.Fatalf("接收方数=%d 期望2", len(items))
		}
		if items[0].Account != "2088001" || !items[0].Money.Equal(rm("60")) {
			t.Errorf("接收方0: account=%s money=%s 期望 2088001/60", items[0].Account, items[0].Money)
		}
		if items[1].Account != "oABC" || items[1].Name != "个人B" || !items[1].Money.Equal(rm("40")) {
			t.Errorf("接收方1: account=%s name=%s money=%s 期望 oABC/个人B/40", items[1].Account, items[1].Name, items[1].Money)
		}
	})

	t.Run("rate段缺省复用首段", func(t *testing.T) {
		// 3个账号 rate 只给1段 15 → 三方均按 15% 分
		items := splitProfit("a|b|c", "", "15", rm("100"))
		if len(items) != 3 {
			t.Fatalf("接收方数=%d 期望3", len(items))
		}
		for i, it := range items {
			if !it.Money.Equal(rm("15")) {
				t.Errorf("接收方%d 金额=%s 期望15（复用首段rate）", i, it.Money)
			}
		}
	})

	t.Run("floor向下取整到分", func(t *testing.T) {
		// realmoney=33.33, rate=10 → 33.33*10=333.3, floor=333, /100=3.33
		items := splitProfit("a", "", "10", rm("33.33"))
		if !items[0].Money.Equal(rm("3.33")) {
			t.Errorf("金额=%s 期望3.33（floor(333.3)/100）", items[0].Money)
		}
		// realmoney=9.99, rate=33 → 9.99*33=329.67, floor=329, /100=3.29
		items = splitProfit("a", "", "33", rm("9.99"))
		if !items[0].Money.Equal(rm("3.29")) {
			t.Errorf("金额=%s 期望3.29（floor(329.67)/100）", items[0].Money)
		}
	})

	t.Run("空账号段跳过", func(t *testing.T) {
		items := splitProfit("a||b", "", "10|20|30", rm("100"))
		if len(items) != 2 {
			t.Fatalf("接收方数=%d 期望2（空段跳过）", len(items))
		}
		// 注意：空段跳过后 name/rate 仍按原索引取，b 对应 rate 索引2=30
		if items[1].Account != "b" || !items[1].Money.Equal(rm("30")) {
			t.Errorf("接收方1: account=%s money=%s 期望 b/30", items[1].Account, items[1].Money)
		}
	})

	t.Run("sumProfit汇总", func(t *testing.T) {
		items := splitProfit("a|b", "", "30|20", rm("200"))
		if !sumProfit(items).Equal(rm("100")) {
			t.Errorf("汇总=%s 期望100（60+40）", sumProfit(items))
		}
	})
}

// TestProfitSupportPlugins B1-55：仅支持分账的插件集合才尝试匹配分账规则。
// 对齐 epay ProfitSharing\CommUtil::$plugins = [alipay,alipaysl,alipayd,wxpayn,wxpaynp,yeepay,yseqt,chinaums,dinpay,adapay]。
func TestProfitSupportPlugins(t *testing.T) {
	support := []string{"alipay", "alipaysl", "alipayd", "wxpayn", "wxpaynp", "yeepay", "yseqt", "chinaums", "dinpay", "adapay"}
	for _, p := range support {
		if !profitSupportPlugins[p] {
			t.Errorf("插件 %s 应在支持分账集合内", p)
		}
	}
	// 不支持分账的插件（免签/自研/收银台等）不应命中
	for _, p := range []string{"epay", "epayn", "vmq", "wxh5", "wxjsapi", "mock", "", "unknown"} {
		if profitSupportPlugins[p] {
			t.Errorf("插件 %s 不应在支持分账集合内", p)
		}
	}
	if len(profitSupportPlugins) != len(support) {
		t.Errorf("支持集合大小=%d 期望 %d（与 epay CommUtil::$plugins 一致）", len(profitSupportPlugins), len(support))
	}
}

// TestProfitOrderStatusDelay B1-42/B1-70：分账单初始 status/delay 由插件与延迟结算开关决定。
// 对齐 epay functions.php:667-669：no_order_plugins(chinaums/dinpay)建单即成功(status=2)；
// wxpaynp/alipayd + direct_settle_time=1 → delay=1(24小时)，其余 delay=0(60秒冷却)。
func TestProfitOrderStatusDelay(t *testing.T) {
	cases := []struct {
		name         string
		plugin       string
		directSettle bool
		wantStatus   int8
		wantDelay    int8
	}{
		{"普通渠道立即结算→待分账0/delay0", "alipay", false, 0, 0},
		{"chinaums→建单即成功2/delay0", "chinaums", false, 2, 0},
		{"dinpay→建单即成功2/delay0", "dinpay", true, 2, 0}, // no_order 优先，delay 对其无意义仍算
		{"wxpaynp+延迟结算→待分账0/delay1", "wxpaynp", true, 0, 1},
		{"alipayd+延迟结算→待分账0/delay1", "alipayd", true, 0, 1},
		{"wxpaynp+立即结算→待分账0/delay0", "wxpaynp", false, 0, 0},
		{"alipayd+立即结算→待分账0/delay0", "alipayd", false, 0, 0},
		{"其它插件+延迟结算→delay不触发", "alipay", true, 0, 0},
	}
	for _, c := range cases {
		st, dl := profitOrderStatusDelay(c.plugin, c.directSettle)
		if st != c.wantStatus || dl != c.wantDelay {
			t.Errorf("%s: got status=%d delay=%d, 期望 status=%d delay=%d", c.name, st, dl, c.wantStatus, c.wantDelay)
		}
	}
}
