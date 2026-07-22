package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/0538pay/api/internal/channel"
	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/repository"
	"github.com/shopspring/decimal"
)

// itoaUID 商户号转字符串（收款方名称缺省时用）。
func itoaUID(uid uint) string { return strconv.FormatUint(uint64(uid), 10) }

// MerchantSelfService 商户中心其余自助流程：测试支付 + 聚合收款码 + 收款页公开下单。
// 测试支付复用 PayService.CreateInternalOrder 走标准收单链（对齐 epay user/test.php）。
type MerchantSelfService struct {
	merchants *repository.MerchantRepo
	channels  *repository.ChannelRepo
	paytypes  *repository.PayTypeRepo
	pay       *PayService
	cfg       *ConfigService
}

func NewMerchantSelfService(
	merchants *repository.MerchantRepo,
	channels *repository.ChannelRepo,
	paytypes *repository.PayTypeRepo,
	pay *PayService,
	cfg *ConfigService,
) *MerchantSelfService {
	return &MerchantSelfService{merchants: merchants, channels: channels, paytypes: paytypes, pay: pay, cfg: cfg}
}

// ---- 测试支付（对齐 epay user/test.php）----

// TestPayInfo 测试支付页信息：开关 + 金额上下限 + 可选支付方式。
func (s *MerchantSelfService) TestPayInfo() dto.TestPayInfoResp {
	return dto.TestPayInfoResp{
		Open:     s.cfg.Int("test_open", 0) == 1,
		MinMoney: s.cfg.Dec("pay_minmoney", decimal.NewFromFloat(0.01)).String(),
		MaxMoney: s.cfg.Dec("pay_maxmoney", decimal.NewFromInt(50000)).String(),
		Types:    s.enabledPayTypes(),
	}
}

// TestPay 测试支付：用固定测试收款商户(test_pay_uid，0 则用当前商户)下一笔真实订单走收单链。
// 复用 CreateInternalOrder（tid=3 标记测试单，对齐 epay test.php tid=3）。返回收银台可用的下单信息。
func (s *MerchantSelfService) TestPay(uid uint, req dto.TestPayReq) (*dto.SubmitResp, error) {
	if s.cfg.Int("test_open", 0) != 1 {
		return nil, maErr("测试支付未开放")
	}
	money, err := decimal.NewFromString(strings.TrimSpace(req.Money))
	if err != nil || money.LessThanOrEqual(decimal.Zero) {
		return nil, maErr("支付金额不合法")
	}
	// 金额上下限（对齐 epay pay_maxmoney/pay_minmoney）。
	minM := s.cfg.Dec("pay_minmoney", decimal.NewFromFloat(0.01))
	maxM := s.cfg.Dec("pay_maxmoney", decimal.NewFromInt(50000))
	if minM.GreaterThan(decimal.Zero) && money.LessThan(minM) {
		return nil, maErr("支付金额低于最小限额")
	}
	if maxM.GreaterThan(decimal.Zero) && money.GreaterThan(maxM) {
		return nil, maErr("支付金额超过最大限额")
	}
	plugin := strings.TrimSpace(req.Type)
	if plugin == "" {
		return nil, maErr("请选择支付方式")
	}
	// 收款方：test_pay_uid 配置了则用固定测试商户，否则下到当前商户（对齐 epay test_pay_uid）。
	payUID := uid
	if tp := s.cfg.Int("test_pay_uid", 0); tp > 0 {
		payUID = uint(tp)
	}
	// tid=3 测试支付标记（对齐 epay test.php）。
	return s.pay.CreateInternalOrder(context.Background(), payUID, 3, "支付测试", money, plugin)
}

// ---- 聚合收款码（对齐 epay user/onecode.php）----

// OnecodeInfo 商户聚合收款码信息：可用性 + 固定收款页 URL + 收款方名称。
func (s *MerchantSelfService) OnecodeInfo(uid uint, siteURL string) (*dto.OnecodeInfo, error) {
	m, err := s.merchants.FindByUID(uid)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, maErr("商户不存在")
	}
	// 全局 onecode 开关 或 该商户单独开启(open_code=1)（对齐 epay onecode.php）。
	open := s.cfg.Int("onecode", 0) == 1 || m.OpenCode == 1
	codeName := m.CodeName
	if codeName == "" {
		codeName = "商户 " + itoaUID(uid)
	}
	payURL := strings.TrimRight(siteURL, "/") + "/paypage?merchant=" + encodeInviteUID(uid)
	return &dto.OnecodeInfo{
		Open:     open,
		PayURL:   payURL,
		CodeName: codeName,
	}, nil
}

// SaveCodeName 保存收款方名称（对齐 epay ajax2 edit_codename）。
func (s *MerchantSelfService) SaveCodeName(uid uint, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return maErr("收款方名称不能为空")
	}
	if len([]rune(name)) > 32 {
		return maErr("收款方名称过长")
	}
	return s.merchants.UpdateFields(uid, map[string]interface{}{"codename": name})
}

// ---- 公开收款页（paypage，对齐 epay paypage/index.php）----

// PaypageInfo 收款页信息（解密 merchant 得 uid → 展示收款方 + 可选支付方式）。
func (s *MerchantSelfService) PaypageInfo(merchant string) (*dto.PaypageInfo, error) {
	uid := decodeInviteUID(merchant)
	if uid == 0 {
		return nil, maErr("收款码无效")
	}
	m, err := s.merchants.FindByUID(uid)
	if err != nil {
		return nil, err
	}
	if m == nil || m.Status == 0 {
		return nil, maErr("收款码无效或已停用")
	}
	if s.cfg.Int("onecode", 0) != 1 && m.OpenCode != 1 {
		return nil, maErr("该商户未开启聚合收款")
	}
	codeName := m.CodeName
	if codeName == "" {
		codeName = "商户 " + itoaUID(uid)
	}
	return &dto.PaypageInfo{
		CodeName: codeName,
		SiteName: s.cfg.Str("sitename"),
		Types:    s.enabledPayTypes(),
	}, nil
}

// PaypageSubmit 收款页下单：解密 merchant 得收款商户 → 输入金额 + 选支付方式 → 走收单链。
func (s *MerchantSelfService) PaypageSubmit(req dto.PaypageSubmitReq) (*dto.SubmitResp, error) {
	uid := decodeInviteUID(req.Merchant)
	if uid == 0 {
		return nil, maErr("收款码无效")
	}
	m, err := s.merchants.FindByUID(uid)
	if err != nil {
		return nil, err
	}
	if m == nil || m.Status == 0 {
		return nil, maErr("收款码无效或已停用")
	}
	if s.cfg.Int("onecode", 0) != 1 && m.OpenCode != 1 {
		return nil, maErr("该商户未开启聚合收款")
	}
	money, err := decimal.NewFromString(strings.TrimSpace(req.Money))
	if err != nil || money.LessThanOrEqual(decimal.Zero) {
		return nil, maErr("付款金额不合法")
	}
	minM := s.cfg.Dec("pay_minmoney", decimal.NewFromFloat(0.01))
	maxM := s.cfg.Dec("pay_maxmoney", decimal.NewFromInt(50000))
	if minM.GreaterThan(decimal.Zero) && money.LessThan(minM) {
		return nil, maErr("付款金额低于最小限额")
	}
	if maxM.GreaterThan(decimal.Zero) && money.GreaterThan(maxM) {
		return nil, maErr("付款金额超过最大限额")
	}
	plugin := strings.TrimSpace(req.Type)
	if plugin == "" {
		return nil, maErr("请选择支付方式")
	}
	// 聚合收款是真实收款到该商户余额，走普通订单（tid=0），复用收单链。
	return s.pay.CreateInternalOrder(context.Background(), uid, 0, "聚合收款", money, plugin)
}

// enabledPayTypes 取当前有已开启通道的支付方式（收银可选项）。
// 以启用通道的 plugin 去重（阶段A/B 下单按 plugin 名定位通道）。
func (s *MerchantSelfService) enabledPayTypes() []dto.PayTypeOption {
	types, err := s.paytypes.All()
	if err != nil {
		return nil
	}
	out := make([]dto.PayTypeOption, 0, len(types))
	seen := map[string]bool{}
	for i := range types {
		t := &types[i]
		if t.Status != 1 {
			continue
		}
		// 该支付方式下需存在已开启通道，且该通道插件已在渠道注册表实现（否则下单会失败）。
		ch, err := s.channels.FindEnabledByPlugin(t.Name)
		if err != nil || ch == nil {
			continue
		}
		if _, ok := channel.Get(ch.Plugin); !ok {
			continue // 通道 plugin 未实现（如 seed 里 alipay/wxpay 显示名），不作为测试/收款可选项
		}
		if seen[ch.Plugin] {
			continue
		}
		seen[ch.Plugin] = true
		// Type 用通道 plugin（下单 CreateInternalOrder 按 plugin 定位并 dispatch），ShowName 用支付方式友好名。
		out = append(out, dto.PayTypeOption{Type: ch.Plugin, ShowName: t.ShowName})
	}
	return out
}
