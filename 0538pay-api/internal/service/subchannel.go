package service

import (
	"encoding/json"
	"strings"

	"github.com/epvia/api/internal/channel"
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

// InfoForm 返回某子通道的自定义参数动态表单（对齐 epay ajax_user.php:704-740 subChannelInfo）：
// 按其归属主通道的 config 占位符 + 插件 inputs 元数据生成字段，并回填该子通道 info 的当前值。
// 只对主 config 里值以 '[' 开头的占位字段渲染，避免管理员手填错 key。
func (s *SubChannelService) InfoForm(id uint) (*dto.SubChannelInfoForm, error) {
	sub, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, subErr("子通道不存在")
	}
	return s.buildInfoForm(sub.Channel, sub.Info)
}

// InfoFormByChannel 按主通道 ID 生成空白占位字段表单（新建子通道时预览要填哪些参数）。
func (s *SubChannelService) InfoFormByChannel(channelID int) (*dto.SubChannelInfoForm, error) {
	return s.buildInfoForm(channelID, "")
}

// buildInfoForm 核心：遍历主通道 config，值形如 "[key]" 的为占位字段，取占位 key，
// 配上插件 Inputs 元数据（Label/Type/Options/Tip，按 config 键匹配）与 info 当前值。
// 对齐 epay：foreach plugin.inputs → 判 config[key] 以 '[' 开头 → 取占位 key 渲染。
func (s *SubChannelService) buildInfoForm(channelID int, curInfo string) (*dto.SubChannelInfoForm, error) {
	ch, err := s.channels.FindByID(uint(channelID))
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, subErr("归属主通道不存在")
	}
	form := &dto.SubChannelInfoForm{
		Channel:     channelID,
		ChannelName: ch.Name,
		Plugin:      ch.Plugin,
		Fields:      []dto.SubChannelInfoField{},
	}
	// 主通道 config（键→值，值可能是 "[占位]"）。
	cfg := map[string]string{}
	if strings.TrimSpace(ch.Config) != "" {
		if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
			return nil, subErr("主通道配置不是合法的 JSON")
		}
	}
	// 子通道当前 info（编辑回填；新建为空）。
	arr := map[string]string{}
	if strings.TrimSpace(curInfo) != "" {
		_ = json.Unmarshal([]byte(curInfo), &arr) // 脏值容忍：解析失败按空回填
	}
	// 插件字段元数据：orderedInputs 保留声明顺序，inputMeta 供按 config 键查 Label/Type/Options/Tip。
	var pluginInputs []channel.FieldInput
	if c, ok := channel.Get(ch.Plugin); ok {
		if cfgr, ok := c.(channel.Configurable); ok {
			pluginInputs = cfgr.Inputs()
		}
	}
	inputMeta := map[string]channel.FieldInput{}
	for _, in := range pluginInputs {
		inputMeta[in.Name] = in
	}
	// 对齐 epay：按插件 inputs 顺序遍历，只渲染 config 值为占位符的字段（保持稳定字段顺序）。
	// 无插件元数据时退回遍历 config（仍只取占位字段），保证任意通道都能用。
	seen := map[string]bool{}
	appendField := func(configKey, placeholderKey string) {
		if placeholderKey == "" || seen[placeholderKey] {
			return
		}
		seen[placeholderKey] = true
		f := dto.SubChannelInfoField{Key: placeholderKey, Type: "text", Value: arr[placeholderKey]}
		if in, ok := inputMeta[configKey]; ok {
			f.Label = in.Label
			if in.Type != "" {
				f.Type = in.Type
			}
			f.Options = in.Options
			f.Tip = in.Tip
		}
		if f.Label == "" {
			f.Label = placeholderKey
		}
		form.Fields = append(form.Fields, f)
	}
	if len(pluginInputs) > 0 {
		for _, in := range pluginInputs {
			appendField(in.Name, placeholderKey(cfg[in.Name]))
		}
	} else {
		for k, v := range cfg {
			appendField(k, placeholderKey(v))
		}
	}
	return form, nil
}

// placeholderKey 若值形如 "[key]" 则返回去括号的 key，否则返回空串（非占位字段不渲染）。
func placeholderKey(v string) string {
	if len(v) >= 2 && v[0] == '[' && v[len(v)-1] == ']' {
		return v[1 : len(v)-1]
	}
	return ""
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
