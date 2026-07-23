package service

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
	"github.com/shopspring/decimal"
)

// timeNow 便于单测替换的当前时间源。默认 time.Now。
var timeNow = time.Now

// rollMember 轮询组内一个成员（通道ID + 权重）。对齐 epay rollinfo_decode 的 {name,weight}。
type rollMember struct {
	ChannelID int
	Weight    int
}

// parseGroupInfo 解析用户组 info：{"typeid":{"type","channel","rate"}}（对齐 epay pre_group.info）。
// 键为支付方式ID（字符串），值为分配对象。空串/非法 JSON/非对象结构返回 nil（交调用方走无组随机）。
func parseGroupInfo(info string) map[int]GroupAssign {
	info = strings.TrimSpace(info)
	if info == "" {
		return nil
	}
	raw := map[string]GroupAssign{}
	if err := json.Unmarshal([]byte(info), &raw); err != nil {
		return nil
	}
	if len(raw) == 0 {
		return nil
	}
	out := make(map[int]GroupAssign, len(raw))
	for k, v := range raw {
		id, err := strconv.Atoi(strings.TrimSpace(k))
		if err != nil {
			continue
		}
		out[id] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseRollInfo 解析轮询组 info 串（对齐 epay rollinfo_decode）：
//   - "12:3,15:7" → [{12,3},{15,7}]（权重随机）
//   - "12,15,18"  → [{12,0},{15,0},{18,0}]（顺序/首个）
//
// 非法或空段跳过；通道ID解析失败的项丢弃。
func parseRollInfo(content string) []rollMember {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	parts := strings.Split(content, ",")
	out := make([]rollMember, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		seg := strings.SplitN(p, ":", 2)
		cid, err := strconv.Atoi(strings.TrimSpace(seg[0]))
		if err != nil || cid <= 0 {
			continue
		}
		w := 0
		if len(seg) == 2 {
			w, _ = strconv.Atoi(strings.TrimSpace(seg[1]))
		}
		out = append(out, rollMember{ChannelID: cid, Weight: w})
	}
	return out
}

// buildRollInfo 反向：把成员列表拼成 info 串（后台保存轮询组通道用）。
//   - kind=1（权重随机）："cid:weight,..."
//   - 其它 kind："cid,..."（权重丢弃，对齐 epay saveRollInfo）
func buildRollInfo(kind int8, members []rollMember) string {
	segs := make([]string, 0, len(members))
	for _, m := range members {
		if kind == 1 {
			w := m.Weight
			if w <= 0 {
				w = 1
			}
			segs = append(segs, strconv.Itoa(m.ChannelID)+":"+strconv.Itoa(w))
		} else {
			segs = append(segs, strconv.Itoa(m.ChannelID))
		}
	}
	return strings.Join(segs, ",")
}

// parseRateOverride 解析组级费率覆盖字符串。空串返回 (0,false) 表示无覆盖，用通道默认。
// 非法数值同样视为无覆盖（宽松，避免坏配置卡下单）。
func parseRateOverride(s string) (decimal.Decimal, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return decimal.Zero, false
	}
	d, err := decimal.NewFromString(s)
	if err != nil || d.IsNegative() {
		return decimal.Zero, false
	}
	return d, true
}

// channelFitsMoney 判断金额是否落在通道 paymin/paymax 区间（对齐 epay 的 $money>0 守卫 + 边界过滤）。
// money<=0 时不过滤（返回 true）；paymin/paymax 为空或非正数时该侧不限制。
func channelFitsMoney(c *model.Channel, money decimal.Decimal) bool {
	if money.LessThanOrEqual(decimal.Zero) {
		return true
	}
	if min, ok := parsePosDec(c.PayMin); ok && money.LessThan(min) {
		return false
	}
	if max, ok := parsePosDec(c.PayMax); ok && money.GreaterThan(max) {
		return false
	}
	return true
}

// checkChannelPayLimit 选定通道后的单笔限额硬拒绝（B1-01/B1-56，对齐 epay Pay.php:170-174 / submit2.php:64-68）。
// 与 channelFitsMoney（候选过滤，返回 bool）不同：这里是选后硬校验，越限直接返回错误中止下单，
// 文案对齐 epay「单笔最小/最大限额为X元」。paymin/paymax 为空或非正数则该侧不限制。
func checkChannelPayLimit(payMin, payMax string, money decimal.Decimal) error {
	if money.LessThanOrEqual(decimal.Zero) {
		return nil
	}
	if min, ok := parsePosDec(payMin); ok && money.LessThan(min) {
		return payErr("当前支付方式单笔最小限额为" + min.String() + "元，请选择其他支付方式")
	}
	if max, ok := parsePosDec(payMax); ok && money.GreaterThan(max) {
		return payErr("当前支付方式单笔最大限额为" + max.String() + "元，请选择其他支付方式")
	}
	return nil
}

// subFitsMoney 子通道候选的金额过滤（复用主通道 paymin/paymax）。
func subFitsMoney(p *repository.SubChannelPick, money decimal.Decimal) bool {
	if money.LessThanOrEqual(decimal.Zero) {
		return true
	}
	if min, ok := parsePosDec(p.PayMin); ok && money.LessThan(min) {
		return false
	}
	if max, ok := parsePosDec(p.PayMax); ok && money.GreaterThan(max) {
		return false
	}
	return true
}

// parsePosDec 解析正的金额字符串；空/非法/<=0 返回 ok=false（表示该侧不限制）。
func parsePosDec(s string) (decimal.Decimal, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return decimal.Zero, false
	}
	d, err := decimal.NewFromString(s)
	if err != nil || d.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, false
	}
	return d, true
}
