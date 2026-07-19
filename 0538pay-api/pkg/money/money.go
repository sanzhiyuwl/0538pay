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
