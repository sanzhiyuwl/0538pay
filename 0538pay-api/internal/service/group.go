package service

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
	"github.com/shopspring/decimal"
)

// GroupService 用户组管理业务（对齐 epay glist/gedit/group.php + ajax_user saveGroup/delGroup）。
type GroupService struct {
	repo      *repository.GroupRepo
	merchants *repository.MerchantRepo
}

func NewGroupService(repo *repository.GroupRepo, merchants *repository.MerchantRepo) *GroupService {
	return &GroupService{repo: repo, merchants: merchants}
}

// GroupError 携带业务错误码与提示。
type GroupError struct {
	Code int
	Msg  string
}

func (e *GroupError) Error() string { return e.Msg }

func grpErr(msg string) *GroupError { return &GroupError{Code: 1011, Msg: msg} }

// List 返回全部用户组（按 sort/gid 升序），派生费率说明与该组商户数。
func (s *GroupService) List() ([]dto.GroupView, error) {
	list, err := s.repo.All()
	if err != nil {
		return nil, err
	}
	counts, err := s.merchants.CountByGroup()
	if err != nil {
		return nil, err
	}
	views := make([]dto.GroupView, 0, len(list))
	for i := range list {
		views = append(views, toGroupView(&list[i], counts[list[i].GID]))
	}
	return views, nil
}

func toGroupView(g *model.Group, count int64) dto.GroupView {
	return dto.GroupView{
		GID:           g.GID,
		Name:          g.Name,
		IsBuy:         g.IsBuy,
		Price:         g.Price.StringFixed(2),
		Expire:        g.Expire,
		Sort:          g.Sort,
		Visible:       g.Visible,
		Rates:         parseGroupRates(g.Info),
		Info:          g.Info,
		Config:        g.Config,
		Settings:      g.Settings,
		MerchantCount: count,
	}
}

// validateJSON 校验字符串为合法 JSON（空串放行，视为无配置）。
func validateJSON(s, field string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if !json.Valid([]byte(s)) {
		return "", grpErr(field + "不是合法的 JSON")
	}
	return s, nil
}

// applyGroupForm 校验表单并落到 model.Group 的可编辑字段（新增/编辑共用）。
func (s *GroupService) applyGroupForm(g *model.Group, req dto.GroupSaveReq) error {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return grpErr("用户组名称不能为空")
	}
	price := decimal.Zero
	if strings.TrimSpace(req.Price) != "" {
		p, err := decimal.NewFromString(strings.TrimSpace(req.Price))
		if err != nil || p.IsNegative() {
			return grpErr("售价填写错误")
		}
		price = p
	}
	info, err := validateJSON(req.Info, "通道费率配置")
	if err != nil {
		return err
	}
	config, err := validateJSON(req.Config, "功能配置")
	if err != nil {
		return err
	}
	g.Name = name
	g.IsBuy = req.IsBuy
	g.Price = price
	g.Expire = req.Expire
	g.Sort = req.Sort
	g.Visible = strings.ReplaceAll(strings.TrimSpace(req.Visible), "，", ",") // 中文逗号转英文
	g.Info = info
	g.Config = config
	g.Settings = strings.TrimSpace(req.Settings)
	return nil
}

// Create 新增用户组（对齐 epay saveGroup action=add）。组名唯一。
func (s *GroupService) Create(req dto.GroupSaveReq) (int, error) {
	n, err := s.repo.CountByName(strings.TrimSpace(req.Name), nil)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		return 0, grpErr("用户组名称重复")
	}
	var g model.Group
	if err := s.applyGroupForm(&g, req); err != nil {
		return 0, err
	}
	if err := s.repo.Create(&g); err != nil {
		return 0, err
	}
	return g.GID, nil
}

// Update 编辑用户组（对齐 epay saveGroup else 分支）。组名唯一（排除自身）。
func (s *GroupService) Update(gid int, req dto.GroupSaveReq) error {
	exist, err := s.repo.FindByID(gid)
	if err != nil {
		return err
	}
	if exist == nil {
		return grpErr("用户组不存在")
	}
	n, err := s.repo.CountByName(strings.TrimSpace(req.Name), &gid)
	if err != nil {
		return err
	}
	if n > 0 {
		return grpErr("用户组名称重复")
	}
	if err := s.applyGroupForm(exist, req); err != nil {
		return err
	}
	fields := map[string]interface{}{
		"name":     exist.Name,
		"is_buy":   exist.IsBuy,
		"price":    exist.Price,
		"expire":   exist.Expire,
		"sort":     exist.Sort,
		"visible":  exist.Visible,
		"info":     exist.Info,
		"config":   exist.Config,
		"settings": exist.Settings,
	}
	return s.repo.Update(gid, fields)
}

// SetBuy 用户组上/下架（对齐 epay saveGroup action=changebuy）。
func (s *GroupService) SetBuy(gid int, isbuy int8) error {
	exist, err := s.repo.FindByID(gid)
	if err != nil {
		return err
	}
	if exist == nil {
		return grpErr("用户组不存在")
	}
	if isbuy != 0 && isbuy != 1 {
		return grpErr("上架状态不合法")
	}
	return s.repo.Update(gid, map[string]interface{}{"is_buy": isbuy})
}

// Delete 删除用户组（对齐 epay delGroup）。默认组 gid=0 不可删；删除后该组商户回落默认组。
func (s *GroupService) Delete(gid int) error {
	if gid == 0 {
		return grpErr("系统自带默认用户组不支持删除")
	}
	exist, err := s.repo.FindByID(gid)
	if err != nil {
		return err
	}
	if exist == nil {
		return grpErr("用户组不存在")
	}
	if err := s.repo.Delete(gid); err != nil {
		return err
	}
	// 级联：该组下商户回落默认组（对齐 epay delGroup 的 UPDATE pre_user SET gid=0）
	return s.merchants.ResetGroupToDefault(gid)
}

// GetAssigns 读取某组的通道分配（解析 info 的 {typeid:{type,channel,rate}}）。
// 供后台「通道分配」编辑回填。缺失键的支付方式不返回项（前端按可用支付方式补默认）。
func (s *GroupService) GetAssigns(gid int) ([]dto.GroupAssignItem, error) {
	g, err := s.repo.FindByID(gid)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, grpErr("用户组不存在")
	}
	m := parseGroupInfo(g.Info)
	items := make([]dto.GroupAssignItem, 0, len(m))
	for typeID, a := range m {
		items = append(items, dto.GroupAssignItem{
			Type:    typeID,
			Kind:    a.Type,
			Channel: strings.TrimSpace(a.Channel),
			Rate:    strings.TrimSpace(a.Rate),
		})
	}
	return items, nil
}

// SaveAssigns 保存某组的通道分配，序列化为 epay 格式 info JSON（对齐 saveGroup 的 info 字段）。
// 整组一次性覆盖：每个支付方式一项 {type,channel,rate}。
func (s *GroupService) SaveAssigns(gid int, req dto.GroupAssignSaveReq) error {
	g, err := s.repo.FindByID(gid)
	if err != nil {
		return err
	}
	if g == nil {
		return grpErr("用户组不存在")
	}
	info, err := buildGroupInfo(req.Assigns)
	if err != nil {
		return err
	}
	return s.repo.Update(gid, map[string]interface{}{"info": info})
}

// buildGroupInfo 把分配项列表拼成 epay 格式 info JSON：{"typeid":{"type","channel","rate"}}。
// channel 校验：0/-1/-2 或正整数；kind 归一为 channel|roll（仅正整数分配才有意义）。
func buildGroupInfo(items []dto.GroupAssignItem) (string, error) {
	out := map[string]GroupAssign{}
	for _, it := range items {
		if it.Type < 0 {
			return "", grpErr("支付方式ID不合法")
		}
		channel := strings.TrimSpace(it.Channel)
		if channel == "" {
			channel = "-1" // 空视为随机可用通道（对齐 epay 默认）
		}
		n, err := strconv.Atoi(channel)
		if err != nil {
			return "", grpErr("通道分配值不合法")
		}
		kind := ""
		if n > 0 {
			// 正整数才需 kind 区分通道/轮询组；非法值归一为 channel。
			kind = strings.TrimSpace(it.Kind)
			if kind != "roll" {
				kind = "channel"
			}
		}
		rate := strings.TrimSpace(it.Rate)
		if rate != "" {
			if _, ok := parseRateOverride(rate); !ok {
				return "", grpErr("费率覆盖值不合法")
			}
		}
		out[strconv.Itoa(it.Type)] = GroupAssign{Type: kind, Channel: channel, Rate: rate}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
