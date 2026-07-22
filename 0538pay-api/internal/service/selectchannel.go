package service

import (
	"math/rand"
	"strconv"
	"strings"

	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
	"github.com/shopspring/decimal"
)

// ChannelSelector 下单选通道核心分发。1:1 移植 epay includes/lib/Channel.php 的
// getSubmitInfo + getChannelFromRoll：按 商户用户组(gid) 对某支付方式(typeid) 的分配
// (0关/-1随机/-2子通道/正整数固定或轮询组) 选出最终主通道 + 可选子通道，并算出组级费率覆盖。
type ChannelSelector struct {
	channels    *repository.ChannelRepo
	rolls       *repository.RollRepo
	subchannels *repository.SubChannelRepo
	groups      *repository.GroupRepo

	// randIntn 注入随机源（单测可替换为确定序列）。默认 rand.Intn。
	randIntn func(n int) int
}

func NewChannelSelector(ch *repository.ChannelRepo, rl *repository.RollRepo, sc *repository.SubChannelRepo, gr *repository.GroupRepo) *ChannelSelector {
	return &ChannelSelector{channels: ch, rolls: rl, subchannels: sc, groups: gr, randIntn: rand.Intn}
}

// SelectResult 选通道结果（对齐 epay getSubmitInfo 返回的 channel/subchannel/rate）。
type SelectResult struct {
	ChannelID  int             // 命中主通道ID
	Subchannel int             // 命中子通道ID（0=无）
	Plugin     string          // 主通道插件
	Rate       decimal.Decimal // 最终费率%（组覆盖优先，空则通道默认）
	SubInfo    string          // 子通道自定义参数 JSON（占位替换用，无则空）
}

// GroupAssign 用户组 info 里单个支付方式的分配配置（对齐 epay pre_group.info 的值对象）。
// epay 里 channel 以字符串存（如 "-1"），故用 string 承接再转换。
type GroupAssign struct {
	Type    string `json:"type"`    // "channel" | "roll" | ""（区分正整数是通道还是轮询组）
	Channel string `json:"channel"` // "0"关 / "-1"随机 / "-2"子通道 / 正整数(通道ID或轮询组ID)
	Rate    string `json:"rate"`    // 组级费率覆盖（百分数字符串，空=用通道默认）
}

// selErr 选通道失败错误（对齐 epay getSubmitInfo 返回 false 的各情形）。
func selErr(msg string) *PayError { return &PayError{Code: 1101, Msg: msg} }

// Select 选通道主入口。money<=0 时不做金额区间过滤（对齐 epay `$money>0` 守卫）。
// 返回 error 表示无可用通道/被关闭（对齐 epay return false，下单应拒绝）。
func (s *ChannelSelector) Select(uid uint, gid, typeID int, money decimal.Decimal) (*SelectResult, error) {
	assign, hasGroup, err := s.loadAssign(gid, typeID)
	if err != nil {
		return nil, err
	}
	if !hasGroup {
		// 未设置用户组 info → 该 type 下所有启用通道中随机（对齐 epay else 分支的 SQL rand()）。
		return s.pickRandomOfType(typeID, money, decimal.Decimal{}, false)
	}

	channelVal, _ := strconv.Atoi(strings.TrimSpace(assign.Channel))
	rateOverride, hasRate := parseRateOverride(assign.Rate)

	switch {
	case channelVal == 0:
		// (A) 关闭：该商户禁用此支付方式。
		return nil, selErr("当前支付方式已关闭")
	case channelVal == -1:
		// (B) 随机可用通道。
		return s.pickRandomOfType(typeID, money, rateOverride, hasRate)
	case channelVal == -2:
		// (C) 用户自定义子通道（顺序调度）。
		return s.pickSubChannel(uid, typeID, money, rateOverride, hasRate)
	default:
		// (D) 正整数：固定通道 或 轮询组（由 type 字段判定）。
		return s.pickFixedOrRoll(assign, channelVal, money, rateOverride, hasRate)
	}
}

// loadAssign 载入某组对某支付方式的分配配置。gid>0 查该组，缺失回退默认组 gid=0（对齐 epay）。
// 返回 hasGroup=false 表示两级组的 info 都为空/无该 type 键 → 交由调用方走无组随机。
func (s *ChannelSelector) loadAssign(gid, typeID int) (GroupAssign, bool, error) {
	info := ""
	if gid > 0 {
		g, err := s.groups.FindByID(gid)
		if err != nil {
			return GroupAssign{}, false, err
		}
		if g != nil {
			info = g.Info
		}
	}
	if strings.TrimSpace(info) == "" {
		g0, err := s.groups.FindByID(0)
		if err != nil {
			return GroupAssign{}, false, err
		}
		if g0 != nil {
			info = g0.Info
		}
	}
	m := parseGroupInfo(info)
	if m == nil {
		return GroupAssign{}, false, nil
	}
	assign, ok := m[typeID]
	if !ok {
		// 有 info 但无该 type 键：epay 当作数组缺失 → channel=-1 随机处理。
		return GroupAssign{Channel: "-1"}, true, nil
	}
	return assign, true, nil
}

// pickRandomOfType 对齐 epay -1 分支：取该 type 全部启用通道，金额过滤后等概率随机(array_rand)；
// 全被金额过滤则在原集里随机（epay 同样兜底）。
func (s *ChannelSelector) pickRandomOfType(typeID int, money, rateOverride decimal.Decimal, hasRate bool) (*SelectResult, error) {
	list, err := s.channels.ListEnabledByType(typeID)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, selErr("暂无可用支付通道")
	}
	filtered := make([]model.Channel, 0, len(list))
	for i := range list {
		if channelFitsMoney(&list[i], money) {
			filtered = append(filtered, list[i])
		}
	}
	pool := filtered
	if len(pool) == 0 {
		pool = list // 全被金额过滤 → 原集随机（对齐 epay 兜底）
	}
	ch := pool[s.randIntn(len(pool))]
	return s.buildResult(&ch, 0, "", rateOverride, hasRate), nil
}

// pickSubChannel 对齐 epay -2 分支：该商户该 type 下按 use_time 升序取第一个可用子通道，
// 命中后回写 use_time。金额过滤后取第一个，全被过滤则取原集第一个。
func (s *ChannelSelector) pickSubChannel(uid uint, typeID int, money, rateOverride decimal.Decimal, hasRate bool) (*SelectResult, error) {
	rows, err := s.subchannels.FindPickable(uid, typeID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, selErr("暂无可用子通道，请联系管理员配置")
	}
	var chosen *repository.SubChannelPick
	for i := range rows {
		if subFitsMoney(&rows[i], money) {
			chosen = &rows[i]
			break
		}
	}
	if chosen == nil {
		chosen = &rows[0] // 全被金额过滤 → 取原集第一个（对齐 epay 兜底）
	}
	rate := chosen.Rate
	if hasRate {
		rate = rateOverride.String()
	}
	rateDec, _ := decimal.NewFromString(rate)
	// 命中后刷新调度时间（尽力而为，失败不阻断下单）。
	_ = s.subchannels.TouchUseTime(chosen.SubID, timeNow())
	return &SelectResult{
		ChannelID:  chosen.ChannelID,
		Subchannel: int(chosen.SubID),
		Plugin:     chosen.Plugin,
		Rate:       rateDec,
		SubInfo:    chosen.Info,
	}, nil
}

// pickFixedOrRoll 对齐 epay (D) 分支：type=roll 时正整数是轮询组ID，先经 getChannelFromRoll
// 解析成真实通道ID；否则正整数直接是通道ID。随后校验通道启用。
func (s *ChannelSelector) pickFixedOrRoll(assign GroupAssign, channelVal int, money, rateOverride decimal.Decimal, hasRate bool) (*SelectResult, error) {
	channelID := channelVal
	if assign.Type == "roll" {
		resolved, err := s.getChannelFromRoll(uint(channelVal), money)
		if err != nil {
			return nil, err
		}
		if resolved <= 0 {
			return nil, selErr("轮询组未启用或无可用通道")
		}
		channelID = resolved
	}
	ch, err := s.channels.FindByID(uint(channelID))
	if err != nil {
		return nil, err
	}
	if ch == nil || ch.Status != 1 || ch.DayStatus != 0 {
		return nil, selErr("指定支付通道不可用")
	}
	return s.buildResult(ch, 0, "", rateOverride, hasRate), nil
}

// buildResult 组装选通道结果，套用组级费率覆盖（非空优先，否则用通道 rate）。
func (s *ChannelSelector) buildResult(ch *model.Channel, subID int, subInfo string, rateOverride decimal.Decimal, hasRate bool) *SelectResult {
	rate := ch.Rate
	if hasRate {
		rate = rateOverride
	}
	return &SelectResult{
		ChannelID:  int(ch.ID),
		Subchannel: subID,
		Plugin:     ch.Plugin,
		Rate:       rate,
		SubInfo:    subInfo,
	}
}

// getChannelFromRoll 1:1 移植 epay Channel::getChannelFromRoll：
// 载入轮询组 → 金额过滤组内可用通道 → 按 kind 选：2首个/1权重随机/0顺序游标。
// 返回命中的真实通道ID；组未启用/无可用返回 0。
func (s *ChannelSelector) getChannelFromRoll(rollID uint, money decimal.Decimal) (int, error) {
	roll, err := s.rolls.FindByID(rollID)
	if err != nil {
		return 0, err
	}
	if roll == nil || roll.Status != 1 {
		return 0, nil
	}
	members := parseRollInfo(roll.Info)
	if len(members) == 0 {
		return 0, nil
	}
	ids := make([]int, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.ChannelID)
	}
	chMap, err := s.channels.FindManyByIDs(ids)
	if err != nil {
		return 0, err
	}
	// 保留原顺序中「启用且金额合规」的成员（对齐 epay 先过滤再按 kind 选）。
	avail := make([]rollMember, 0, len(members))
	for _, m := range members {
		ch := chMap[m.ChannelID]
		if ch == nil || ch.Status != 1 || ch.DayStatus != 0 {
			continue
		}
		if !channelFitsMoney(ch, money) {
			continue
		}
		avail = append(avail, m)
	}
	if len(avail) == 0 {
		return 0, nil
	}

	switch roll.Kind {
	case 2: // 首个启用
		return avail[0].ChannelID, nil
	case 1: // 权重随机
		return s.randomWeight(avail), nil
	default: // 0 顺序轮询：游标对「过滤后列表」取模，命中后 +1 回写（对齐 epay index 语义）。
		idx := roll.Idx
		if idx < 0 || idx >= len(avail) {
			idx = 0
		}
		chosen := avail[idx].ChannelID
		next := (idx + 1) % len(avail)
		_ = s.rolls.AdvanceIndex(roll.ID, roll.Idx, next)
		return chosen, nil
	}
}

// randomWeight 1:1 移植 epay random_weight：按权重线性命中；权重和<=0 返回 0。
func (s *ChannelSelector) randomWeight(members []rollMember) int {
	sum := 0
	for _, m := range members {
		if m.Weight > 0 {
			sum += m.Weight
		}
	}
	if sum <= 0 {
		return 0
	}
	r := s.randIntn(sum) + 1 // mt_rand(1, sum) 等价
	for _, m := range members {
		w := m.Weight
		if w < 0 {
			w = 0
		}
		if r <= w {
			return m.ChannelID
		}
		r -= w
	}
	return 0
}
