package service

import (
	"github.com/shopspring/decimal"
)

// CombineConfig 合单拆单配置（对齐 epay conf wxcombine_open/minmoney/submoney）。
type CombineConfig struct {
	Open     bool            // wxcombine_open：是否开启合单
	MinMoney decimal.Decimal // wxcombine_minmoney：达此金额（元）才拆单
	SubMoney decimal.Decimal // wxcombine_submoney：单个子单金额（元）上限，超过则增加子单数
}

// combineConfigFromCfg 从 ConfigService 读取合单配置。cfg 为 nil 时返回未开启。
func combineConfigFromCfg(cfg *ConfigService) CombineConfig {
	if cfg == nil {
		return CombineConfig{}
	}
	return CombineConfig{
		Open:     cfg.Bool("wxcombine_open"),
		MinMoney: cfg.Dec("wxcombine_minmoney", decimal.Zero),
		SubMoney: cfg.Dec("wxcombine_submoney", decimal.Zero),
	}
}

// CombineSubMoneys 把订单金额拆成多个子单金额（分为单位的整数），1:1 移植 epay combinepay_submoneys。
//
// money 为订单金额（分）。规则：
//  1. 未开启 / minmoney 或 submoney 未配 → 返回 nil（不拆）。
//  2. money < minmoney*100 → 返回 nil（不够拆单阈值）。
//  3. 从 3 个子单起，子单额 = money/subnum 向下取整；若子单额仍 > submoney*100，
//     增加子单数直到子单额 ≤ 上限或子单数达 50。
//  4. 余数（money % subnum）平摊到前若干个子单各 +1，保证子单求和 = 总额。
//
// 返回的每个元素为对应子单金额（分）；调用方转元入库/下单。
func CombineSubMoneys(money int64, c CombineConfig) []int64 {
	if !c.Open || c.MinMoney.IsZero() || c.SubMoney.IsZero() {
		return nil
	}
	// minmoney/submoney 单位为元，转分（intval(minmoney*100)，对齐 epay 的整数截断）。
	minFen := c.MinMoney.Mul(decimal.NewFromInt(100)).IntPart()
	subFen := c.SubMoney.Mul(decimal.NewFromInt(100)).IntPart()
	if money < minFen {
		return nil
	}

	subnum := int64(3)
	submoney := money / subnum
	for submoney > subFen {
		subnum++
		submoney = money / subnum
		if subnum == 50 {
			break
		}
	}

	out := make([]int64, subnum)
	for i := range out {
		out[i] = submoney
	}
	// 余数平摊到前 mod 个子单，保证求和 = money（对齐 epay $submoneys[$i] += 1）。
	mod := money % subnum
	for i := int64(0); i < mod; i++ {
		out[i]++
	}
	return out
}
