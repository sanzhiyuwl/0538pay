package service

import (
	"encoding/json"
	"strings"

	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
)

// 硬锁检查的被管控能力枚举常量（对齐微信 4012803072 limited_functions）。
const (
	ctrlFnNoTransaction            = "NO_TRANSACTION"              // 关闭收单
	ctrlFnNoTransactionAndRecharge = "NO_TRANSACTION_AND_RECHARGE" // 关闭收单和充值（含收单语义）
	ctrlFnNoRefund                 = "NO_REFUND"                   // 关闭退款
	ctrlFnNoWithdrawal             = "NO_WITHDRAWAL"               // 关闭提现
	ctrlFnNoProfitSharing          = "NO_PROFIT_SHARING"           // 关闭分账分出
)

// subMchConfigKey 通道配置里子商户号的键（服务商模式标志，与 wxbase/wxv2base 一致）。
const subMchConfigKey = "sub_mchid"

// ChannelControlGuard 风控第二段·收单硬锁：读 pay_channel_control 本地快照，命中被管控能力即拦截
// 对应操作（收单/退款/提现/分账），不回退平台号——守 0730「不走二清」红线。
//
// ★只读本地快照（刷新由 admin/risk-controls 单个/批量刷新触发落库），绝不在交易链路现查微信，
//   避免 429 RATELIMIT_EXCEEDED 与阻塞下单。
// ★只锁 state=controlled（已实际生效）；delayed（延迟管控未到点）不锁，仅风控页提示，提前引导解脱。
// ★快照缺失(nil)/DB 读错 → 放行：控股商户的钱本就直清其子商户号，即便放行微信侧仍会拦，
//   不会误落平台号；而 DB 抖动全站拦单危害更大，故基础设施错误 fail-open，仅对「明确被管控」fail-close。
type ChannelControlGuard struct {
	controls *repository.ChannelControlRepo
	enrolls  *repository.ChannelEnrollRepo
}

func NewChannelControlGuard(controls *repository.ChannelControlRepo, enrolls *repository.ChannelEnrollRepo) *ChannelControlGuard {
	return &ChannelControlGuard{controls: controls, enrolls: enrolls}
}

// ControlLockedError 硬锁拦截错误（携带被管控能力中文，供上层回显解脱指引）。
type ControlLockedError struct{ Msg string }

func (e *ControlLockedError) Error() string { return e.Msg }

// snapHitFunction 判断快照是否在 state=controlled 下命中给定能力之一，命中返回触发的能力枚举，否则空串。
func snapHitFunction(snap *model.ChannelControl, funcs ...string) string {
	if snap == nil || snap.State != model.ChannelControlControlled || snap.LimitedFunctions == "" {
		return ""
	}
	var fns []string
	if json.Unmarshal([]byte(snap.LimitedFunctions), &fns) != nil {
		return ""
	}
	want := make(map[string]struct{}, len(funcs))
	for _, f := range funcs {
		want[f] = struct{}{}
	}
	for _, f := range fns {
		if _, ok := want[f]; ok {
			return f
		}
	}
	return ""
}

// GuardSubMch 按子商户号校验给定能力是否被管控（收单/渠道退款用，sub_mchid 由订单通道配置解析）。
// subMchID 可逗号分隔多值（历史平台多租户配置），任一命中即拦截；空串（非服务商单）直接放行。
func (g *ChannelControlGuard) GuardSubMch(subMchID string, funcs ...string) error {
	if g == nil || g.controls == nil {
		return nil
	}
	subMchID = strings.TrimSpace(subMchID)
	if subMchID == "" {
		return nil
	}
	for _, sm := range strings.Split(subMchID, ",") {
		sm = strings.TrimSpace(sm)
		if sm == "" {
			continue
		}
		snap, err := g.controls.FindBySubMchID(sm)
		if err != nil {
			continue // DB 读错：fail-open（见类型注释），不因基础设施抖动全站拦单
		}
		if fn := snapHitFunction(snap, funcs...); fn != "" {
			return g.lockedErr(sm, fn)
		}
	}
	return nil
}

// GuardMerchant 按商户校验其名下任一已开通子商户是否被管控（提现/分账等无订单级 sub_mchid 的操作用）。
// 名下任一 approved 进件单快照命中即拦截（fail-closed 保守，宁严勿漏）；无进件/无快照的纯平台商户放行。
func (g *ChannelControlGuard) GuardMerchant(uid uint, funcs ...string) error {
	if g == nil || g.controls == nil || g.enrolls == nil {
		return nil
	}
	list, err := g.enrolls.ListApprovedByUID(uid)
	if err != nil || len(list) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(list))
	for i := range list {
		ids = append(ids, list[i].ID)
	}
	snaps, err := g.controls.MapByEnrollIDs(ids)
	if err != nil {
		return nil
	}
	for _, id := range ids {
		if fn := snapHitFunction(snaps[id], funcs...); fn != "" {
			return g.lockedErr(snaps[id].SubMchID, fn)
		}
	}
	return nil
}

func (g *ChannelControlGuard) lockedErr(subMchID, fn string) *ControlLockedError {
	return &ControlLockedError{Msg: "子商户 " + subMchID + " 已被微信管控（" +
		enumText(limitedFunctionText, fn) + "），该操作已被拦截，请前往「商户风控管控」查看解脱指引"}
}
