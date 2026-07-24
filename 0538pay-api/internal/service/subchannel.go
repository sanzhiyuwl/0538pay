package service

import (
	"encoding/json"
	"strings"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
)

// SubChannelService 子通道管理（后台商户维度，对齐 epay ajax_user saveSubChannel/subChannelInfo）。
type SubChannelService struct {
	repo     *repository.SubChannelRepo
	channels *repository.ChannelRepo
	merchants *repository.MerchantRepo
}

func NewSubChannelService(repo *repository.SubChannelRepo, channels *repository.ChannelRepo, merchants *repository.MerchantRepo) *SubChannelService {
	return &SubChannelService{repo: repo, channels: channels, merchants: merchants}
}

// SubChannelError 携带业务错误码与提示。
type SubChannelError struct {
	Code int
	Msg  string
}

func (e *SubChannelError) Error() string { return e.Msg }

func subErr(msg string) *SubChannelError { return &SubChannelError{Code: 1106, Msg: msg} }

// ListByMerchant 列出某商户的全部子通道，派生主通道名与使用时间文案。
func (s *SubChannelService) ListByMerchant(uid uint) ([]dto.SubChannelView, error) {
	list, err := s.repo.ListByMerchant(uid)
	if err != nil {
		return nil, err
	}
	nameCache := map[int]string{}
	views := make([]dto.SubChannelView, 0, len(list))
	for i := range list {
		sc := &list[i]
		name, ok := nameCache[sc.Channel]
		if !ok {
			if ch, _ := s.channels.FindByID(uint(sc.Channel)); ch != nil {
				name = ch.Name
			}
			nameCache[sc.Channel] = name
		}
		useTime := "—"
		if sc.UseTime != nil {
			useTime = sc.UseTime.Format(timeLayout)
		}
		views = append(views, dto.SubChannelView{
			ID:          sc.ID,
			Channel:     sc.Channel,
			ChannelName: name,
			UID:         sc.UID,
			Name:        sc.Name,
			Status:      sc.Status,
			Info:        sc.Info,
			UseTime:     useTime,
		})
	}
	return views, nil
}

// validateSubForm 校验子通道表单（主通道存在、名称非空且组内唯一、info 合法 JSON）。
func (s *SubChannelService) validateSubForm(uid uint, req dto.SubChannelSaveReq, excludeID uint) (string, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return "", subErr("子通道名称不能为空")
	}
	ch, err := s.channels.FindByID(uint(req.Channel))
	if err != nil {
		return "", err
	}
	if ch == nil {
		return "", subErr("归属主通道不存在")
	}
	info := strings.TrimSpace(req.Info)
	if info != "" && !json.Valid([]byte(info)) {
		return "", subErr("自定义参数不是合法的 JSON")
	}
	n, err := s.repo.CountByName(uid, name, excludeID)
	if err != nil {
		return "", err
	}
	if n > 0 {
		return "", subErr("该商户下已存在同名子通道")
	}
	return info, nil
}

// Create 新增子通道（对齐 epay saveSubChannel add：写 channel/uid/name/addtime/usetime）。
func (s *SubChannelService) Create(uid uint, req dto.SubChannelSaveReq) (uint, error) {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return 0, err
	}
	if m == nil {
		return 0, subErr("商户不存在")
	}
	info, err := s.validateSubForm(uid, req, 0)
	if err != nil {
		return 0, err
	}
	now := timeNow()
	sc := &model.SubChannel{
		Channel: req.Channel,
		UID:     uid,
		Name:    strings.TrimSpace(req.Name),
		Status:  0, // 新建默认关闭
		Info:    info,
		AddTime: now,
		UseTime: &now, // 初始 usetime=创建时间，参与顺序调度
	}
	if err := s.repo.Create(sc); err != nil {
		return 0, err
	}
	return sc.ID, nil
}

// Update 编辑子通道（改归属通道/名称/自定义参数）。
func (s *SubChannelService) Update(id uint, req dto.SubChannelSaveReq) error {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return subErr("子通道不存在")
	}
	info, err := s.validateSubForm(exist.UID, req, id)
	if err != nil {
		return err
	}
	fields := map[string]interface{}{
		"channel": req.Channel,
		"name":    strings.TrimSpace(req.Name),
		"info":    info,
	}
	return s.repo.Update(id, fields)
}

// SetStatus 切换子通道开关。
func (s *SubChannelService) SetStatus(id uint, status int8) error {
	if status != 0 && status != 1 {
		return subErr("状态值不合法")
	}
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return subErr("子通道不存在")
	}
	return s.repo.Update(id, map[string]interface{}{"status": status})
}

// Delete 删除子通道。
func (s *SubChannelService) Delete(id uint) error {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return subErr("子通道不存在")
	}
	return s.repo.Delete(id)
}
