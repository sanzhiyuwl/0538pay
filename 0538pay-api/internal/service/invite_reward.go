package service

import (
	"strconv"
	"strings"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/repository"
	"github.com/shopspring/decimal"
)

// 邀请返现记录类型（对齐 epay pre_record.type='邀请返现'）。
const inviteRewardType = "邀请返现"

// inviteSalt 邀请码混淆盐（对齐 epay get_invite_code 用 SYS_KEY 异或的思路，我方用固定盐 + 进制变换）。
const inviteSalt = 0x5A538

// encodeInviteUID 把商户 uid 编码为推广邀请码（异或混淆后 36 进制，避免直接暴露 uid）。
func encodeInviteUID(uid uint) string {
	return strings.ToUpper(strconv.FormatUint(uint64(uid)^inviteSalt, 36))
}

// decodeInviteUID 反解邀请码为 uid。非法返回 0。
func decodeInviteUID(code string) uint {
	code = strings.TrimSpace(strings.ToLower(code))
	if code == "" {
		return 0
	}
	n, err := strconv.ParseUint(code, 36, 64)
	if err != nil {
		return 0
	}
	return uint(n ^ inviteSalt)
}

// InviteRewardService 邀请返现结算与统计（对齐 epay user/invite.php + functions.php 结算钩子）。
// 挂在订单支付成功链路上：被邀请人（下单商户）支付成功 → 按比例返现到其上级（upid）余额。
type InviteRewardService struct {
	merchants *repository.MerchantRepo
	accounts  *repository.AccountRepo
	records   *repository.RecordRepo
	cfg       *ConfigService
}

func NewInviteRewardService(m *repository.MerchantRepo, a *repository.AccountRepo, r *repository.RecordRepo, cfg *ConfigService) *InviteRewardService {
	return &InviteRewardService{merchants: m, accounts: a, records: r, cfg: cfg}
}

// SettleOnPaid 订单支付成功后结算邀请返现（实时发放，对齐 epay functions.php 结算钩子）。
// uid=下单商户；money=订单金额；getMoney=商户实得(分成)；reduceMoney=手续费(平台+商户差额)。
// 返现发到该商户上级(upid)的余额并写流水。任一环节不满足则静默跳过（不影响主链）。
func (s *InviteRewardService) SettleOnPaid(uid uint, money, getMoney decimal.Decimal, tradeNo string) {
	if s.cfg.Int("invite_open", 0) != 1 {
		return
	}
	rate := s.cfg.Dec("invite_rate", decimal.Zero)
	if rate.LessThanOrEqual(decimal.Zero) {
		return
	}
	buyer, err := s.merchants.FindByUID(uid)
	if err != nil || buyer == nil || buyer.UpID <= 0 {
		return
	}
	// 上级必须存在且未封禁。
	up, err := s.merchants.FindByUID(uint(buyer.UpID))
	if err != nil || up == nil || up.Status == 0 {
		return
	}

	// 手续费(reducemoney) = 订单金额 - 商户实得(getmoney)。getmoney<=0（阶段A无费率）时视为 0。
	reduceMoney := decimal.Zero
	if getMoney.GreaterThan(decimal.Zero) {
		reduceMoney = money.Sub(getMoney)
		if reduceMoney.LessThan(decimal.Zero) {
			reduceMoney = decimal.Zero
		}
	}

	reward := s.calcReward(money, getMoney, reduceMoney, rate)
	if reward.LessThanOrEqual(decimal.Zero) {
		return
	}
	// 发放到上级余额 + 写"邀请返现"流水（幂等由调用方保证——每笔订单只结算一次）。
	_ = s.accounts.ChangeUserMoney(up.UID, reward, true, inviteRewardType, tradeNo)
}

// calcReward 三种返现口径（对齐 epay invite_order_type）：
//
//	0(默认)：按订单金额 money × rate/100，且 invite_order_fee=0 时封顶不超过手续费 reduceMoney
//	1：按订单手续费 reduceMoney × rate/100
//	2：按平台利润 profit(=reduceMoney，我方无独立利润字段，等同手续费) × rate/100
func (s *InviteRewardService) calcReward(money, getMoney, reduceMoney, rate decimal.Decimal) decimal.Decimal {
	hundred := decimal.NewFromInt(100)
	switch s.cfg.Int("invite_order_type", 0) {
	case 1:
		return reduceMoney.Mul(rate).Div(hundred).Round(2)
	case 2:
		// 平台利润：我方口径 = 手续费（无独立成本字段可扣，对齐 epay profitmoney 缺省行为）。
		return reduceMoney.Mul(rate).Div(hundred).Round(2)
	default:
		reward := money.Mul(rate).Div(hundred).Round(2)
		// invite_order_fee=0：返现不超过手续费（对齐 epay "分成金额最多不超过订单手续费"）。
		if s.cfg.Int("invite_order_fee", 0) != 1 && reward.GreaterThan(reduceMoney) {
			reward = reduceMoney
		}
		return reward
	}
}

// Info 返现设置 + 推广信息（供商户端 /m/invite 渲染）。
func (s *InviteRewardService) Info(uid uint, siteURL string) dto.InviteRewardInfo {
	rate := s.cfg.Dec("invite_rate", decimal.Zero)
	link := strings.TrimRight(siteURL, "/") + "/m/reg?invite=" + encodeInviteUID(uid)
	return dto.InviteRewardInfo{
		Open:      s.cfg.Int("invite_open", 0) == 1,
		Rate:      rate.String(),
		OrderType: s.cfg.Int("invite_order_type", 0),
		OrderFee:  s.cfg.Int("invite_order_fee", 0) == 1,
		Link:      link,
		Code:      encodeInviteUID(uid),
	}
}

// Stats 返现统计：已邀请人数 + 今日/昨日/累计返现额。
func (s *InviteRewardService) Stats(uid uint) (dto.InviteRewardStat, error) {
	var st dto.InviteRewardStat
	users, err := s.merchants.CountByUpID(uid)
	if err != nil {
		return st, err
	}
	st.Users = users

	today, yesterday, total, err := s.records.SumInviteReward(uid)
	if err != nil {
		return st, err
	}
	st.IncomeToday = today.StringFixed(2)
	st.IncomeYesterday = yesterday.StringFixed(2)
	st.IncomeTotal = total.StringFixed(2)
	return st, nil
}

// List 已邀请下级商户列表（分页）。
func (s *InviteRewardService) List(uid uint, page, pageSize int) ([]dto.InvitedUserView, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	list, total, err := s.merchants.ListByUpID(uid, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.InvitedUserView, 0, len(list))
	for i := range list {
		m := &list[i]
		views = append(views, dto.InvitedUserView{
			UID:     m.UID,
			AddTime: m.AddTime.Format(timeLayout),
			Status:  m.Status,
		})
	}
	return views, total, nil
}
