package service

import (
	"strings"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
)

// RollService 通道轮询组管理（对齐 epay pay_roll.php + ajax_pay saveRoll/saveRollInfo）。
type RollService struct {
	repo     *repository.RollRepo
	channels *repository.ChannelRepo
}

func NewRollService(repo *repository.RollRepo, channels *repository.ChannelRepo) *RollService {
	return &RollService{repo: repo, channels: channels}
}

// RollError 携带业务错误码与提示。
type RollError struct {
	Code int
	Msg  string
}

func (e *RollError) Error() string { return e.Msg }

func rollErr(msg string) *RollError { return &RollError{Code: 1105, Msg: msg} }

// List 返回全部轮询组，info 串解析为通道列表并派生通道名/支付方式名。
func (s *RollService) List() ([]dto.RollView, error) {
	list, err := s.repo.All()
	if err != nil {
		return nil, err
	}
	// 预取所有涉及的通道名（一次查全，避免 N+1）。
	nameCache := map[int]string{}
	views := make([]dto.RollView, 0, len(list))
	for i := range list {
		r := &list[i]
		members := parseRollInfo(r.Info)
		items := make([]dto.RollChannelItem, 0, len(members))
		for _, m := range members {
			name, ok := nameCache[m.ChannelID]
			if !ok {
				if ch, _ := s.channels.FindByID(uint(m.ChannelID)); ch != nil {
					name = ch.Name
				}
				nameCache[m.ChannelID] = name
			}
			items = append(items, dto.RollChannelItem{
				Channel: m.ChannelID, ChannelName: name, Weight: m.Weight,
			})
		}
		views = append(views, dto.RollView{
			ID:           r.ID,
			Name:         r.Name,
			Type:         r.Type,
			TypeShowName: channelTypeMeta(r.Type).showname,
			Kind:         r.Kind,
			Channels:     items,
			Status:       r.Status,
		})
	}
	return views, nil
}

// applyForm 校验轮询组表单并落到 model 的可编辑字段（info 由 channels 拼串）。
func (s *RollService) applyForm(r *model.Roll, req dto.RollSaveReq) error {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return rollErr("显示名称不能为空")
	}
	if req.Kind < 0 || req.Kind > 2 {
		return rollErr("轮询方式不合法")
	}
	members := make([]rollMember, 0, len(req.Channels))
	for _, c := range req.Channels {
		if c.Channel <= 0 {
			continue
		}
		w := c.Weight
		if req.Kind == 1 && w <= 0 {
			w = 1 // 权重随机下权重至少 1
		}
		members = append(members, rollMember{ChannelID: c.Channel, Weight: w})
	}
	r.Name = name
	r.Type = req.Type
	r.Kind = req.Kind
	r.Info = buildRollInfo(req.Kind, members)
	return nil
}

// Create 新增轮询组（默认关闭，配好后手动开启）。
func (s *RollService) Create(req dto.RollSaveReq) (uint, error) {
	var r model.Roll
	if err := s.applyForm(&r, req); err != nil {
		return 0, err
	}
	r.Status = 0
	if err := s.repo.Create(&r); err != nil {
		return 0, err
	}
	return r.ID, nil
}

// Update 编辑轮询组（改名/支付方式/轮询方式/组内通道）。切换 kind 时 info 串按新 kind 重拼。
func (s *RollService) Update(id uint, req dto.RollSaveReq) error {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return rollErr("轮询组不存在")
	}
	if err := s.applyForm(exist, req); err != nil {
		return err
	}
	fields := map[string]interface{}{
		"name": exist.Name,
		"type": exist.Type,
		"kind": exist.Kind,
		"info": exist.Info,
		"idx":  0, // 组内通道变更后游标归零，避免越界漂移
	}
	return s.repo.Update(id, fields)
}

// SetStatus 切换轮询组开关。
func (s *RollService) SetStatus(id uint, status int8) error {
	if status != 0 && status != 1 {
		return rollErr("状态值不合法")
	}
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return rollErr("轮询组不存在")
	}
	return s.repo.SetStatus(id, status)
}

// Delete 删除轮询组。
func (s *RollService) Delete(id uint) error {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return rollErr("轮询组不存在")
	}
	return s.repo.Delete(id)
}
