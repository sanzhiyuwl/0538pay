package ocr

import (
	"regexp"
	"strings"
)

// truncate 截断字符串用于错误信息，避免把整段响应塞进日志。
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

var dateSep = regexp.MustCompile(`[年月./]`)

// datePat 匹配一个日期（2020-01-01 / 2020.1.1 / 2020年1月1日）或「长期」。
var datePat = regexp.MustCompile(`\d{4}[.\-/年]\d{1,2}[.\-/月]\d{1,2}日?|长期`)

// normalizeDate 把「2020年01月01日」「2020.01.01」「2020/01/01」等归一到「2020-01-01」。
// 「长期」原样保留；无法识别的原样返回。
func normalizeDate(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.Contains(s, "长期") {
		return s
	}
	// 统一分隔符为 -，去掉「日」尾字
	s = dateSep.ReplaceAllString(s, "-")
	s = strings.TrimRight(strings.TrimSuffix(s, "日"), "-")
	// 补零：2020-1-1 → 2020-01-01
	parts := strings.Split(s, "-")
	if len(parts) == 3 {
		if len(parts[1]) == 1 {
			parts[1] = "0" + parts[1]
		}
		if len(parts[2]) == 1 {
			parts[2] = "0" + parts[2]
		}
		return strings.Join(parts, "-")
	}
	return s
}

// splitAliyunPeriod 从营业期限/有效期字段拆出起止日期。
// 阿里云不同版本可能返回合并串（如「2020-01-01至2040-01-01」或「2020-01-01-长期」）
// 或拆开的 from/to；优先用拆开的，其次解析合并串。
func splitAliyunPeriod(period, from, to string) (begin, end string) {
	from, to = strings.TrimSpace(from), strings.TrimSpace(to)
	if from != "" || to != "" {
		return normalizeDate(from), normalizeDate(to)
	}
	period = strings.TrimSpace(period)
	if period == "" {
		return "", ""
	}
	// 直接从串里抽取所有日期 token（兼容「2020.1.1-2040.1.1」「2020-01-01至长期」等各种分隔）。
	tokens := datePat.FindAllString(period, -1)
	if len(tokens) >= 2 {
		return normalizeDate(tokens[0]), normalizeDate(tokens[1])
	}
	if len(tokens) == 1 {
		return normalizeDate(tokens[0]), ""
	}
	return normalizeDate(period), ""
}
