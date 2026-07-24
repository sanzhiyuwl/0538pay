package service

import (
	"regexp"
	"strings"
	"time"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
)

// nameAlnumRe 支付方式调用值：字母数字（对齐 epay savePayType 正则）。
var nameAlnumRe = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

// maskSecret 脱敏密钥：保留首尾各 4 位，中间用 * 覆盖。太短则全 *。
func maskSecret(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) <= 8 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", 12) + s[len(s)-4:]
}

// ===== 支付方式 PayTypeService =====

type PayTypeService struct {
	repo     *repository.PayTypeRepo
	channels *repository.ChannelRepo
}

func NewPayTypeService(repo *repository.PayTypeRepo, channels *repository.ChannelRepo) *PayTypeService {
	return &PayTypeService{repo: repo, channels: channels}
}

type PayTypeError struct {
	Code int
	Msg  string
}

func (e *PayTypeError) Error() string { return e.Msg }
func ptErr(msg string) *PayTypeError  { return &PayTypeError{Code: 1107, Msg: msg} }

func (s *PayTypeService) List() ([]dto.PayTypeView, error) {
	list, err := s.repo.All()
	if err != nil {
		return nil, err
	}
	views := make([]dto.PayTypeView, 0, len(list))
	for i := range list {
		t := &list[i]
		views = append(views, dto.PayTypeView{
			ID: t.ID, Name: t.Name, ShowName: t.ShowName,
			Device: t.Device, Today: "0.00", Status: t.Status,
		})
	}
	return views, nil
}

func (s *PayTypeService) validate(req dto.PayTypeSaveReq, excludeID uint) error {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ptErr("调用值不能为空")
	}
	if !nameAlnumRe.MatchString(name) {
		return ptErr("调用值只能是字母和数字")
	}
	if strings.TrimSpace(req.ShowName) == "" {
		return ptErr("显示名称不能为空")
	}
	if req.Device < 0 || req.Device > 2 {
		return ptErr("支持设备值不合法")
	}
	n, err := s.repo.CountByNameDevice(name, req.Device, excludeID)
	if err != nil {
		return err
	}
	if n > 0 {
		return ptErr("同一「调用值 + 支持设备」已存在")
	}
	return nil
}

func (s *PayTypeService) Create(req dto.PayTypeSaveReq) (uint, error) {
	if err := s.validate(req, 0); err != nil {
		return 0, err
	}
	m := &model.PayType{
		Name: strings.TrimSpace(req.Name), ShowName: strings.TrimSpace(req.ShowName),
		Device: req.Device, Status: 1, // 新增默认开启（对齐 epay add status=1）
	}
	if err := s.repo.Create(m); err != nil {
		return 0, err
	}
	return m.ID, nil
}

func (s *PayTypeService) Update(id uint, req dto.PayTypeSaveReq) error {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return ptErr("支付方式不存在")
	}
	if err := s.validate(req, id); err != nil {
		return err
	}
	return s.repo.Update(id, map[string]interface{}{
		"name": strings.TrimSpace(req.Name), "show_name": strings.TrimSpace(req.ShowName),
		"device": req.Device,
	})
}

func (s *PayTypeService) SetStatus(id uint, status int8) error {
	if status != 0 && status != 1 {
		return ptErr("状态值不合法")
	}
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return ptErr("支付方式不存在")
	}
	return s.repo.Update(id, map[string]interface{}{"status": status})
}

// Delete 删除支付方式。系统自带(id<4)不可删；被通道引用拒删（对齐 epay delPayType）。
func (s *PayTypeService) Delete(id uint) error {
	if id < 4 {
		return ptErr("系统自带支付方式不支持删除")
	}
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return ptErr("支付方式不存在")
	}
	cnt, err := s.channels.CountByType(int(id))
	if err != nil {
		return err
	}
	if cnt > 0 {
		return ptErr("该支付方式已被支付通道使用，无法删除")
	}
	return s.repo.Delete(id)
}

// ===== 微信公众号/小程序 WeixinService =====

type WeixinService struct {
	repo *repository.WeixinRepo
}

func NewWeixinService(repo *repository.WeixinRepo) *WeixinService { return &WeixinService{repo: repo} }

type WeixinError struct {
	Code int
	Msg  string
}

func (e *WeixinError) Error() string { return e.Msg }
func wxErr(msg string) *WeixinError  { return &WeixinError{Code: 1108, Msg: msg} }

func (s *WeixinService) List() ([]dto.WeixinView, error) {
	list, err := s.repo.All()
	if err != nil {
		return nil, err
	}
	views := make([]dto.WeixinView, 0, len(list))
	for i := range list {
		w := &list[i]
		views = append(views, dto.WeixinView{
			ID: w.ID, Type: w.Type, Name: w.Name, AppID: w.AppID,
			AppSecret: maskSecret(w.AppSecret),
		})
	}
	return views, nil
}

func (s *WeixinService) validate(req dto.WeixinSaveReq, excludeID uint) error {
	if strings.TrimSpace(req.Name) == "" {
		return wxErr("名称不能为空")
	}
	if strings.TrimSpace(req.AppID) == "" {
		return wxErr("APPID 不能为空")
	}
	if req.Type != 0 && req.Type != 1 {
		return wxErr("类别值不合法")
	}
	n, err := s.repo.CountByName(strings.TrimSpace(req.Name), excludeID)
	if err != nil {
		return err
	}
	if n > 0 {
		return wxErr("名称重复")
	}
	n, err = s.repo.CountByAppID(strings.TrimSpace(req.AppID), excludeID)
	if err != nil {
		return err
	}
	if n > 0 {
		return wxErr("APPID 重复")
	}
	return nil
}

func (s *WeixinService) Create(req dto.WeixinSaveReq) (uint, error) {
	if err := s.validate(req, 0); err != nil {
		return 0, err
	}
	now := time.Now()
	m := &model.Weixin{
		Type: req.Type, Name: strings.TrimSpace(req.Name),
		AppID: strings.TrimSpace(req.AppID), AppSecret: strings.TrimSpace(req.AppSecret),
		Status: 1, AddTime: &now,
	}
	if err := s.repo.Create(m); err != nil {
		return 0, err
	}
	return m.ID, nil
}

func (s *WeixinService) Update(id uint, req dto.WeixinSaveReq) error {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return wxErr("账号不存在")
	}
	if err := s.validate(req, id); err != nil {
		return err
	}
	fields := map[string]interface{}{
		"type": req.Type, "name": strings.TrimSpace(req.Name), "app_id": strings.TrimSpace(req.AppID),
	}
	// appsecret 留空表示不修改（避免脱敏回填覆盖真实值）。
	if strings.TrimSpace(req.AppSecret) != "" && !strings.Contains(req.AppSecret, "*") {
		fields["app_secret"] = strings.TrimSpace(req.AppSecret)
	}
	return s.repo.Update(id, fields)
}

func (s *WeixinService) Delete(id uint) error {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return wxErr("账号不存在")
	}
	// epay 校验被 channel.appwxmp/appwxa 引用拒删；我方 channel 模型无该字段，无引用可能，直接删。
	return s.repo.Delete(id)
}

// ===== 企业微信 WeworkService =====

type WeworkService struct {
	repo *repository.WeworkRepo
}

func NewWeworkService(repo *repository.WeworkRepo) *WeworkService { return &WeworkService{repo: repo} }

type WeworkError struct {
	Code int
	Msg  string
}

func (e *WeworkError) Error() string { return e.Msg }
func wwErr(msg string) *WeworkError  { return &WeworkError{Code: 1109, Msg: msg} }

func (s *WeworkService) List() ([]dto.WeworkView, error) {
	list, err := s.repo.All()
	if err != nil {
		return nil, err
	}
	counts, err := s.repo.CountKfByWork()
	if err != nil {
		return nil, err
	}
	views := make([]dto.WeworkView, 0, len(list))
	for i := range list {
		w := &list[i]
		views = append(views, dto.WeworkView{
			ID: w.ID, Name: w.Name, AppID: w.AppID,
			AppSecret: maskSecret(w.AppSecret), KfNum: counts[w.ID], Status: w.Status,
		})
	}
	return views, nil
}

func (s *WeworkService) validate(req dto.WeworkSaveReq, excludeID uint) error {
	if strings.TrimSpace(req.Name) == "" {
		return wwErr("名称不能为空")
	}
	if strings.TrimSpace(req.AppID) == "" {
		return wwErr("企业ID 不能为空")
	}
	n, err := s.repo.CountByName(strings.TrimSpace(req.Name), excludeID)
	if err != nil {
		return err
	}
	if n > 0 {
		return wwErr("名称重复")
	}
	n, err = s.repo.CountByAppID(strings.TrimSpace(req.AppID), excludeID)
	if err != nil {
		return err
	}
	if n > 0 {
		return wwErr("企业ID 重复")
	}
	return nil
}

func (s *WeworkService) Create(req dto.WeworkSaveReq) (uint, error) {
	if err := s.validate(req, 0); err != nil {
		return 0, err
	}
	now := time.Now()
	m := &model.Wework{
		Name: strings.TrimSpace(req.Name), AppID: strings.TrimSpace(req.AppID),
		AppSecret: strings.TrimSpace(req.AppSecret), Status: 1, AddTime: &now,
	}
	if err := s.repo.Create(m); err != nil {
		return 0, err
	}
	return m.ID, nil
}

func (s *WeworkService) Update(id uint, req dto.WeworkSaveReq) error {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return wwErr("账号不存在")
	}
	if err := s.validate(req, id); err != nil {
		return err
	}
	fields := map[string]interface{}{
		"name": strings.TrimSpace(req.Name), "app_id": strings.TrimSpace(req.AppID),
	}
	if strings.TrimSpace(req.AppSecret) != "" && !strings.Contains(req.AppSecret, "*") {
		fields["app_secret"] = strings.TrimSpace(req.AppSecret)
	}
	return s.repo.Update(id, fields)
}

func (s *WeworkService) SetStatus(id uint, status int8) error {
	if status != 0 && status != 1 {
		return wwErr("状态值不合法")
	}
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return wwErr("账号不存在")
	}
	return s.repo.Update(id, map[string]interface{}{"status": status})
}

func (s *WeworkService) Delete(id uint) error {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return wwErr("账号不存在")
	}
	return s.repo.Delete(id) // 级联删客服账号在 repo 事务内
}
