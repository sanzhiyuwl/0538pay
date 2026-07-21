package service

import (
	"strings"
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
)

// RiskError 携带业务错误码与提示（风控/黑名单/域名共用）。
type RiskError struct {
	Code int
	Msg  string
}

func (e *RiskError) Error() string { return e.Msg }

func rkErr(msg string) *RiskError { return &RiskError{Code: 1108, Msg: msg} }

// ===== 风控（只读列表 + 系统写入）=====

// RiskService 风控记录：只读列表 + 系统触发写入（关键词拦截等）。
type RiskService struct {
	repo *repository.RiskRepo
}

func NewRiskService(repo *repository.RiskRepo) *RiskService { return &RiskService{repo: repo} }

// List 风控记录列表（只读）。
func (s *RiskService) List(q dto.RiskQuery) ([]dto.RiskView, int64, error) {
	q.Normalize()
	list, total, err := s.repo.List(q)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.RiskView, 0, len(list))
	for i := range list {
		r := &list[i]
		views = append(views, dto.RiskView{
			ID: r.ID, UID: r.UID, Type: r.Type,
			Content: r.Content, URL: r.URL, Date: r.Date.Format(timeLayout),
		})
	}
	return views, total, nil
}

// RecordKeyword 记录一条关键词屏蔽命中（type=0，收单链拦截时调用）。
func (s *RiskService) RecordKeyword(uid uint, url, content string) {
	_ = s.repo.Create(&model.RiskRecord{
		UID: uid, Type: 0, URL: url, Content: content, Date: time.Now(),
	})
}

// ===== 黑名单 =====

// BlacklistService 黑名单：CRUD + 下单命中校验。
type BlacklistService struct {
	repo *repository.BlacklistRepo
}

func NewBlacklistService(repo *repository.BlacklistRepo) *BlacklistService {
	return &BlacklistService{repo: repo}
}

// List 黑名单列表。
func (s *BlacklistService) List(q dto.BlacklistQuery) ([]dto.BlacklistView, int64, error) {
	q.Normalize()
	list, total, err := s.repo.List(q)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.BlacklistView, 0, len(list))
	for i := range list {
		views = append(views, toBlacklistView(&list[i]))
	}
	return views, total, nil
}

// Stats 黑名单概况（总数/账号/IP/永久）。
func (s *BlacklistService) Stats() (total, account, ip, permanent int64, err error) {
	if total, err = s.repo.CountByType(-1); err != nil {
		return
	}
	if account, err = s.repo.CountByType(0); err != nil {
		return
	}
	if ip, err = s.repo.CountByType(1); err != nil {
		return
	}
	permanent, err = s.repo.CountPermanent()
	return
}

// Add 添加黑名单：校验内容非空 → (type,content) 查重 → 计算过期时间 → 落库。
func (s *BlacklistService) Add(req dto.BlacklistAddReq) error {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return rkErr("请填写黑名单内容")
	}
	if req.Type != 0 && req.Type != 1 {
		return rkErr("黑名单类型不合法")
	}
	exist, err := s.repo.Exist(req.Type, content)
	if err != nil {
		return err
	}
	if exist {
		return rkErr("该黑名单已存在")
	}
	b := &model.Blacklist{
		Type: req.Type, Content: content, AddTime: time.Now(), Remark: req.Remark,
	}
	if req.Days > 0 {
		end := time.Now().AddDate(0, 0, req.Days)
		b.EndTime = &end
	}
	return s.repo.Create(b)
}

// Delete 删除黑名单。
func (s *BlacklistService) Delete(id uint) error {
	return s.repo.Delete(id)
}

// BatchDelete 批量删除，返回删除条数。
func (s *BlacklistService) BatchDelete(ids []uint) (int64, error) {
	if len(ids) == 0 {
		return 0, rkErr("请选择要删除的记录")
	}
	return s.repo.BatchDelete(ids)
}

// IsBlocked 下单命中校验：type+content 命中未过期黑名单则返回 true。
func (s *BlacklistService) IsBlocked(typ int8, content string) bool {
	if content == "" {
		return false
	}
	blocked, err := s.repo.ExistActive(typ, content, time.Now())
	return err == nil && blocked
}

func toBlacklistView(b *model.Blacklist) dto.BlacklistView {
	v := dto.BlacklistView{
		ID: b.ID, Type: b.Type, Content: b.Content,
		AddTime: b.AddTime.Format(timeLayout), Remark: b.Remark,
	}
	if b.EndTime != nil {
		t := b.EndTime.Format(timeLayout)
		v.EndTime = &t
	}
	return v
}

// ===== 授权域名 =====

// DomainService 授权域名：CRUD + 审核 + 下单白名单校验。
type DomainService struct {
	repo *repository.DomainRepo
}

func NewDomainService(repo *repository.DomainRepo) *DomainService {
	return &DomainService{repo: repo}
}

// List 域名列表。
func (s *DomainService) List(q dto.DomainQuery) ([]dto.DomainView, int64, error) {
	q.Normalize()
	list, total, err := s.repo.List(q)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.DomainView, 0, len(list))
	for i := range list {
		views = append(views, toDomainView(&list[i]))
	}
	return views, total, nil
}

// Stats 域名概况（总数/待审/正常/拒绝）。
func (s *DomainService) Stats() (total, pending, normal, rejected int64, err error) {
	if total, err = s.repo.CountByStatus(-1); err != nil {
		return
	}
	if pending, err = s.repo.CountByStatus(0); err != nil {
		return
	}
	if normal, err = s.repo.CountByStatus(1); err != nil {
		return
	}
	rejected, err = s.repo.CountByStatus(2)
	return
}

// Add 后台添加域名（免审 status=1）：校验格式 → (uid,domain) 查重 → 落库。
func (s *DomainService) Add(req dto.DomainAddReq) error {
	domain := strings.TrimSpace(strings.ToLower(req.Domain))
	if !validDomain(domain) {
		return rkErr("域名格式不正确")
	}
	exist, err := s.repo.Exist(req.UID, domain)
	if err != nil {
		return err
	}
	if exist {
		return rkErr("该商户已存在此授权域名")
	}
	now := time.Now()
	return s.repo.Create(&model.Domain{
		UID: req.UID, Domain: domain, Status: 1, AddTime: now, EndTime: &now,
	})
}

// SetStatus 审核/状态变更（1通过 2拒绝）。
func (s *DomainService) SetStatus(id uint, status int8) error {
	if status != 1 && status != 2 {
		return rkErr("状态值不合法")
	}
	d, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if d == nil {
		return rkErr("域名记录不存在")
	}
	return s.repo.SetStatus(id, status, time.Now())
}

// Delete 删除域名。
func (s *DomainService) Delete(id uint) error {
	return s.repo.Delete(id)
}

// BatchOp 批量操作：status 1通过/2拒绝/3删除。返回影响条数。
func (s *DomainService) BatchOp(ids []uint, status int8) (int64, error) {
	if len(ids) == 0 {
		return 0, rkErr("请选择要操作的记录")
	}
	if status != 1 && status != 2 && status != 3 {
		return 0, rkErr("操作类型不合法")
	}
	return s.repo.BatchOp(ids, status, time.Now())
}

// IsAllowed 下单域名白名单校验：域名或 *.主域名 命中 status=1 记录则放行。
func (s *DomainService) IsAllowed(uid uint, domain string) bool {
	domain = strings.ToLower(domain)
	wildcard := "*." + rootDomain(domain)
	ok, err := s.repo.MatchWhitelist(uid, domain, wildcard)
	return err == nil && ok
}

// HasAnyDomain 判断商户是否配置了任何授权域名（决定是否启用白名单校验；无记录则视为未开启，放行）。
func (s *DomainService) HasAnyDomain(uid uint) bool {
	n, err := s.repo.CountByUID(uid)
	return err == nil && n > 0
}

func toDomainView(d *model.Domain) dto.DomainView {
	v := dto.DomainView{
		ID: d.ID, UID: d.UID, Domain: d.Domain, Status: d.Status,
		AddTime: d.AddTime.Format(timeLayout),
	}
	if d.EndTime != nil {
		t := d.EndTime.Format(timeLayout)
		v.EndTime = &t
	}
	return v
}

// validDomain 简单域名格式校验：支持 *. 通配前缀 + 至少一个点。
func validDomain(d string) bool {
	if d == "" || len(d) > 128 {
		return false
	}
	body := strings.TrimPrefix(d, "*.")
	if body == "" || !strings.Contains(body, ".") {
		return false
	}
	for _, c := range body {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '-') {
			return false
		}
	}
	return true
}

// rootDomain 取主域名（最后两段，如 pay.shop.example.com → example.com）。
func rootDomain(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) <= 2 {
		return host
	}
	return strings.Join(parts[len(parts)-2:], ".")
}
