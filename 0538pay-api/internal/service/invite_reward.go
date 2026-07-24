package service

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/repository"
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
	groups    *repository.GroupRepo // 上级所在组的 invite_* 覆盖配置（可空，SetGroupRepo 注入）
	cfg       *ConfigService
}

func NewInviteRewardService(m *repository.MerchantRepo, a *repository.AccountRepo, r *repository.RecordRepo, cfg *ConfigService) *InviteRewardService {
	return &InviteRewardService{merchants: m, accounts: a, records: r, cfg: cfg}
}

// SetGroupRepo 注入用户组仓储，使邀请返现读取「上级所在组」的组级 invite_* 配置覆盖全局。
// nil 则一律使用全局配置（向后兼容）。
func (s *InviteRewardService) SetGroupRepo(g *repository.GroupRepo) { s.groups = g }

// inviteConf 单笔返现结算时生效的邀请配置（组级覆盖全局后的最终值，对齐 epay array_merge(conf, groupconfig)）。
type inviteConf struct {
	open      bool
	rate      decimal.Decimal
	orderType int
	orderFee  bool
}

// resolveInviteConf 计算某上级(upgid=上级所在组)生效的邀请配置：
// 以全局为底，组级 config 里非空的 invite_open/invite_rate/invite_order_type/invite_order_fee 覆盖之
// （对齐 epay getGroupConfig + array_merge，只覆盖组里"非空"的键）。
func (s *InviteRewardService) resolveInviteConf(upgid int) inviteConf {
	c := inviteConf{
		open:      s.cfg.Int("invite_open", 0) == 1,
		rate:      s.cfg.Dec("invite_rate", decimal.Zero),
		orderType: s.cfg.Int("invite_order_type", 0),
		orderFee:  s.cfg.Int("invite_order_fee", 0) == 1,
	}
	if s.groups == nil {
		return c
	}
	g, err := s.groups.FindByID(upgid)
	if err != nil || g == nil || strings.TrimSpace(g.Config) == "" {
		return c
	}
	var gc map[string]interface{}
	if json.Unmarshal([]byte(g.Config), &gc) != nil {
		return c
	}
	if v, ok := groupConfStr(gc, "invite_open"); ok {
		c.open = v == "1"
	}
	if v, ok := groupConfStr(gc, "invite_rate"); ok {
		if d, err := decimal.NewFromString(v); err == nil {
			c.rate = d
		}
	}
	if v, ok := groupConfStr(gc, "invite_order_type"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			c.orderType = n
		}
	}
	if v, ok := groupConfStr(gc, "invite_order_fee"); ok {
		c.orderFee = v == "1"
	}
	return c
}

// groupConfStr 从组 config map 里取键并转字符串，空值(""/nil)视为"未设置"返回 ok=false
// （对齐 epay getGroupConfig 里 isNullOrEmpty 跳过 + 非空才覆盖）。
func groupConfStr(m map[string]interface{}, key string) (string, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		return "", false
	}
	var s string
	switch t := v.(type) {
	case string:
		s = strings.TrimSpace(t)
	case float64:
		s = strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			s = "1"
		} else {
			s = "0"
		}
	default:
		return "", false
	}
	if s == "" {
		return "", false
	}
	return s, true
}

// SettleOnPaid 订单支付成功后结算邀请返现（实时发放，对齐 epay functions.php 结算钩子）。
// uid=下单商户；money=订单金额；getMoney=商户实得(分成)；reduceMoney=手续费(平台+商户差额)。
// 返现发到该商户上级(upid)的余额并写流水。任一环节不满足则静默跳过（不影响主链）。
func (s *InviteRewardService) SettleOnPaid(uid uint, money, getMoney, profitMoney decimal.Decimal, tradeNo string) {
	buyer, err := s.merchants.FindByUID(uid)
	if err != nil || buyer == nil || buyer.UpID <= 0 {
		return
	}
	// 上级必须存在且未封禁。
	up, err := s.merchants.FindByUID(uint(buyer.UpID))
	if err != nil || up == nil || up.Status == 0 {
		return
	}

	// 读「上级所在组」生效的邀请配置（组级覆盖全局，对齐 epay getGroupConfig(upgid)）。
	ic := s.resolveInviteConf(up.GID)
	if !ic.open || ic.rate.LessThanOrEqual(decimal.Zero) {
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
	// 平台利润 profitmoney 由回调侧 calcProfitMoney 算好传入（可为负，成本高于毛利时）。
	// 直接使用，不回落手续费——对齐 epay functions.php:638 invite_order_type==2 用 $profitmoney 原值，
	// 负值经末尾 invite_money>0 门槛（calcReward 返回后 reward<=0 即跳过）自然丢弃。
	reward := s.calcReward(ic, money, reduceMoney, profitMoney)
	if reward.LessThanOrEqual(decimal.Zero) {
		return
	}
	// 发放到上级余额 + 写"邀请返现"流水（幂等由调用方保证——每笔订单只结算一次）。
	_ = s.accounts.ChangeUserMoney(up.UID, reward, true, inviteRewardType, tradeNo)
}

// calcReward 三种返现口径（对齐 epay invite_order_type，配置取自上级所在组的生效值 ic）：
//
//	0(默认)：按订单金额 money × rate/100，且 invite_order_fee=0 时封顶不超过手续费 reduceMoney
//	1：按订单手续费 reduceMoney × rate/100
//	2：按平台真实利润 profitMoney(=手续费 - 通道成本 costrate) × rate/100（对齐 epay functions.php:638-639）
func (s *InviteRewardService) calcReward(ic inviteConf, money, reduceMoney, profitMoney decimal.Decimal) decimal.Decimal {
	hundred := decimal.NewFromInt(100)
	switch ic.orderType {
	case 1:
		return reduceMoney.Mul(ic.rate).Div(hundred).Round(2)
	case 2:
		// 平台真实利润 = 手续费扣除通道成本 costrate 后的余额（回调侧算好传入）。
		return profitMoney.Mul(ic.rate).Div(hundred).Round(2)
	default:
		reward := money.Mul(ic.rate).Div(hundred).Round(2)
		// invite_order_fee=0：返现不超过手续费（对齐 epay "分成金额最多不超过订单手续费"）。
		if !ic.orderFee && reward.GreaterThan(reduceMoney) {
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
