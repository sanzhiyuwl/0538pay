package service

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/0538pay/api/internal/channel"
	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
	"github.com/shopspring/decimal"
)

// ChannelService 支付通道业务逻辑。
type ChannelService struct {
	repo        *repository.ChannelRepo
	subchannels *repository.SubChannelRepo // 删通道级联删其子通道（可空，SetSubChannelRepo 注入）
	orders      *repository.OrderRepo      // 通道今昨收款/成功率聚合（可空，SetOrderRepo 注入）
}

func NewChannelService(repo *repository.ChannelRepo) *ChannelService {
	return &ChannelService{repo: repo}
}

// SetSubChannelRepo 注入子通道 repo，删主通道时级联删其子通道（对齐 epay 删通道级联）。
func (s *ChannelService) SetSubChannelRepo(r *repository.SubChannelRepo) { s.subchannels = r }

// SetOrderRepo 注入订单 repo，通道列表补今昨收款额 + 今日成功率（对齐 epay pay_channel 刷新聚合）。
func (s *ChannelService) SetOrderRepo(r *repository.OrderRepo) { s.orders = r }

// List 返回分页通道（转对外 View：费率格式化 + 今昨收款/成功率实时聚合）。
func (s *ChannelService) List(q dto.ChannelQuery) ([]dto.ChannelView, int64, error) {
	q.Normalize()
	list, total, err := s.repo.List(q)
	if err != nil {
		return nil, 0, err
	}
	// 今昨收款 + 今日成功率按通道聚合（orders 未注入则留 0，向后兼容）。
	var todayMap, ydayMap map[int]decimal.Decimal
	var rateMap map[int][2]int64
	if s.orders != nil {
		now := time.Now()
		today := dayStart(now)
		yday := today.AddDate(0, 0, -1)
		todayMap, _ = s.orders.SumPaidMoneyByChannel(today, today.AddDate(0, 0, 1))
		ydayMap, _ = s.orders.SumPaidMoneyByChannel(yday, today)
		rateMap, _ = s.orders.ChannelPaidRate(today, today.AddDate(0, 0, 1))
	}
	views := make([]dto.ChannelView, 0, len(list))
	for i := range list {
		v := toChannelView(&list[i])
		cid := int(list[i].ID)
		if m, ok := todayMap[cid]; ok {
			v.Today = m.StringFixed(2)
		}
		if m, ok := ydayMap[cid]; ok {
			v.Yesterday = m.StringFixed(2)
		}
		if r, ok := rateMap[cid]; ok && r[0] > 0 {
			rate := decimal.NewFromInt(r[1]).Mul(decimal.NewFromInt(100)).Div(decimal.NewFromInt(r[0]))
			v.SuccessRate = rate.StringFixed(2)
		}
		views = append(views, v)
	}
	return views, total, nil
}

// PluginMeta 返回所有已注册渠道插件的能力与配置元数据（后台按插件动态渲染密钥表单/展示能力）。
// 对齐 epay 插件 $info（inputs/transtypes/是否支持退款代付）由插件自声明的思路。
func (s *ChannelService) PluginMeta() []channel.PluginMeta {
	return channel.AllMeta()
}

// ChannelError 携带业务错误码与提示，handler 据此返回 code+msg。
type ChannelError struct {
	Code int
	Msg  string
}

func (e *ChannelError) Error() string { return e.Msg }

func chErr(msg string) *ChannelError { return &ChannelError{Code: 1104, Msg: msg} }

// parseRate 解析费率百分比字符串为 decimal，校验非负且 <100（对齐 epay 分成比例范围）。
// empty 为 true 时允许空串（返回 0）。
func parseRate(s string, allowEmpty bool, field string) (decimal.Decimal, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		if allowEmpty {
			return decimal.Zero, nil
		}
		return decimal.Zero, chErr(field + "不能为空")
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, chErr(field + "格式不正确")
	}
	if d.IsNegative() || d.GreaterThanOrEqual(decimal.NewFromInt(100)) {
		return decimal.Zero, chErr(field + "须在 0~100 之间")
	}
	return d, nil
}

// applyForm 把表单入参校验并落到 model.Channel 的可编辑字段（新增/编辑共用）。
// typeMeta 派生 typename/typeshowname，与前端 mock 的 typeMeta 保持一致。
func applyForm(c *model.Channel, req dto.ChannelSaveReq) error {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return chErr("显示名称不能为空")
	}
	if strings.TrimSpace(req.Plugin) == "" {
		return chErr("请选择支付插件")
	}
	rate, err := parseRate(req.Rate, false, "分成比例")
	if err != nil {
		return err
	}
	costRate, err := parseRate(req.CostRate, true, "通道成本")
	if err != nil {
		return err
	}
	c.Name = name
	c.Type = req.Type
	c.Plugin = strings.TrimSpace(req.Plugin)
	c.Mode = req.Mode
	c.Rate = rate
	c.CostRate = costRate
	c.PayMin = strings.TrimSpace(req.PayMin)
	c.PayMax = strings.TrimSpace(req.PayMax)
	// 商户直清(mode=1)不加入余额，单日限额无意义 → 置 0（对齐 epay 表单 mode>0 时禁用 daytop）
	if req.Mode == 1 {
		c.DayTop = 0
	} else {
		c.DayTop = req.DayTop
	}
	// 支付方式元信息（图标名/中文名）
	tm := channelTypeMeta(req.Type)
	c.TypeName = tm.name
	c.TypeShow = tm.showname
	return nil
}

type typeMetaEntry struct {
	name     string
	showname string
}

// channelTypeMeta 支付方式 ID → (英文名/中文名)，对齐前端 mock/orders.ts 的 payTypes + channels.ts typeMeta。
func channelTypeMeta(typeID int) typeMetaEntry {
	switch typeID {
	case 1:
		return typeMetaEntry{"alipay", "支付宝"}
	case 2:
		return typeMetaEntry{"wxpay", "微信支付"}
	case 3:
		return typeMetaEntry{"qqpay", "QQ钱包"}
	case 4:
		return typeMetaEntry{"bank", "云闪付"}
	default:
		return typeMetaEntry{"mock", "模拟支付"}
	}
}

// Create 新增通道，返回新建 ID。
func (s *ChannelService) Create(req dto.ChannelSaveReq) (uint, error) {
	var c model.Channel
	if err := applyForm(&c, req); err != nil {
		return 0, err
	}
	c.Status = 0 // 新建默认关闭，配置好密钥后手动开启（对齐 epay 新增后需启用）
	if err := s.repo.Create(&c); err != nil {
		return 0, err
	}
	return c.ID, nil
}

// Update 编辑通道的可编辑字段（不含 status/config，各由专用接口维护）。
func (s *ChannelService) Update(id uint, req dto.ChannelSaveReq) error {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return chErr("通道不存在")
	}
	if err := applyForm(exist, req); err != nil {
		return err
	}
	fields := map[string]interface{}{
		"name":      exist.Name,
		"type":      exist.Type,
		"type_name": exist.TypeName,
		"type_show": exist.TypeShow,
		"plugin":    exist.Plugin,
		"mode":      exist.Mode,
		"rate":      exist.Rate,
		"cost_rate": exist.CostRate,
		"day_top":   exist.DayTop,
		"pay_min":   exist.PayMin,
		"pay_max":   exist.PayMax,
	}
	return s.repo.Update(id, fields)
}

// Delete 删除通道。
func (s *ChannelService) Delete(id uint) error {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return chErr("通道不存在")
	}
	if err := s.repo.Delete(id); err != nil {
		return err
	}
	// 级联删该主通道下的子通道（对齐 epay 删通道级联）。
	if s.subchannels != nil {
		_ = s.subchannels.DeleteByChannel(int(id))
	}
	return nil
}

// SetStatus 切换通道开关状态。
func (s *ChannelService) SetStatus(id uint, status int8) error {
	if status != 0 && status != 1 {
		return chErr("状态值不合法")
	}
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return chErr("通道不存在")
	}
	return s.repo.SetStatus(id, status)
}

// GetConfig 读取通道密钥配置（供配置抽屉回填）。
func (s *ChannelService) GetConfig(id uint) (*dto.ChannelConfigView, error) {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if exist == nil {
		return nil, chErr("通道不存在")
	}
	return &dto.ChannelConfigView{
		ID:     exist.ID,
		Name:   exist.Name,
		Plugin: exist.Plugin,
		Config: exist.Config,
	}, nil
}

// SaveConfig 保存通道密钥配置。校验 config 为合法 JSON（空串视为清空为 {}）。
func (s *ChannelService) SaveConfig(id uint, config string) error {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return chErr("通道不存在")
	}
	config = strings.TrimSpace(config)
	if config == "" {
		config = "{}"
	}
	if !json.Valid([]byte(config)) {
		return chErr("密钥配置不是合法的 JSON")
	}
	return s.repo.SaveConfig(id, config)
}

func toChannelView(c *model.Channel) dto.ChannelView {
	return dto.ChannelView{
		ID:        c.ID,
		Name:      c.Name,
		Type:      c.Type,
		TypeName:  c.TypeName,
		TypeShow:  c.TypeShow,
		Plugin:    c.Plugin,
		Mode:      c.Mode,
		Rate:      c.Rate.StringFixed(2),
		CostRate:  c.CostRate.StringFixed(2),
		DayTop:    c.DayTop,
		PayMin:    c.PayMin,
		PayMax:    c.PayMax,
		Today:     "0.00", // 派生统计，接订单聚合后补
		Yesterday: "0.00",
		Status:    c.Status,
	}
}
