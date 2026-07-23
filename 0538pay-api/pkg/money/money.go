// Package money 封装金额处理。金额一律用 decimal，严禁 float 参与金额运算。
package money

import (
	"errors"

	"github.com/shopspring/decimal"
)

// Parse 把接口传入的金额字符串解析为 decimal。空串按 0 处理。
func Parse(s string) (decimal.Decimal, error) {
	if s == "" {
		return decimal.Zero, nil
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, errors.New("金额格式非法: " + s)
	}
	if d.IsNegative() {
		return decimal.Zero, errors.New("金额不能为负: " + s)
	}
	return d, nil
}

// String 把 decimal 金额格式化为固定两位小数字符串，用于对外输出。
func String(d decimal.Decimal) string {
	return d.StringFixed(2)
}

// Float 把金额格式化为 PHP (float) 等价形态：先定两位再去掉末尾多余的 0 与小数点，
// 100.00→"100"、100.50→"100.5"、100.55→"100.55"。用于异步回调 money 字段，
// 与 epay creat_callback 的 (float)money 一致（G-5，避免原生 epay 商户 SDK 复算 float 时失配）。
func Float(d decimal.Decimal) string {
	s := d.StringFixed(2)
	// 去尾零
	for len(s) > 0 && s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	return s
}
