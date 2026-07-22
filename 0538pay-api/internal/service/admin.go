package service

import (
	"strings"
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// AdminService 管理员账号 CRUD（多管理员 RBAC，超出 epay 单管理员的增强项）。
// role 用字符串承载（super=超级管理员，其余为自定义角色名）；super 账号受保护不可停用/删除。
type AdminService struct {
	repo *repository.AdminRepo
}

func NewAdminService(repo *repository.AdminRepo) *AdminService {
	return &AdminService{repo: repo}
}

func (s *AdminService) List() ([]dto.AdminView, error) {
	list, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	views := make([]dto.AdminView, 0, len(list))
	for i := range list {
		views = append(views, toAdminView(&list[i]))
	}
	return views, nil
}

// Create 新增管理员：用户名查重 + 密码必填 + bcrypt。
func (s *AdminService) Create(req dto.AdminSaveReq) error {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return aErr("用户名不能为空")
	}
	if len(req.Password) < 6 {
		return aErr("初始密码至少 6 位")
	}
	n, err := s.repo.CountByUsername(username, 0)
	if err != nil {
		return err
	}
	if n > 0 {
		return aErr("该用户名已存在")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	a := &model.Admin{
		Username: username,
		Nickname: strings.TrimSpace(req.Nickname),
		Password: string(hash),
		Role:     normRole(req.Role),
		Status:   normStatus(req.Status),
	}
	return s.repo.Create(a)
}

// Update 编辑管理员：用户名查重（排除自身）+ 密码留空则不改。super 账号角色锁定为 super。
func (s *AdminService) Update(id uint, req dto.AdminSaveReq) error {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return aErr("用户名不能为空")
	}
	n, err := s.repo.CountByUsername(username, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return aErr("该用户名已被占用")
	}
	fields := map[string]interface{}{
		"username": username,
		"nickname": strings.TrimSpace(req.Nickname),
		"status":   normStatus(req.Status),
	}
	// super 账号角色不可改（保护超管）；其余可改角色。
	if exist.Role == "super" {
		fields["role"] = "super"
		fields["status"] = int8(1) // 超管不可停用
	} else {
		fields["role"] = normRole(req.Role)
	}
	if strings.TrimSpace(req.Password) != "" {
		if len(req.Password) < 6 {
			return aErr("新密码至少 6 位")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		fields["password"] = string(hash)
	}
	return s.repo.UpdateFields(id, fields)
}

// SetStatus 启停管理员。super 账号不可停用。
func (s *AdminService) SetStatus(id uint, status int8) error {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist.Role == "super" && status != 1 {
		return aErr("超级管理员不可停用")
	}
	return s.repo.SetStatus(id, normStatus(status))
}

// Delete 删除管理员。super 账号不可删；不可删除最后一个账号。
func (s *AdminService) Delete(id uint) error {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist.Role == "super" {
		return aErr("超级管理员不可删除")
	}
	cnt, err := s.repo.Count()
	if err != nil {
		return err
	}
	if cnt <= 1 {
		return aErr("至少保留一个管理员账号")
	}
	return s.repo.Delete(id)
}

func toAdminView(a *model.Admin) dto.AdminView {
	last := ""
	if a.LastLogin != nil {
		last = a.LastLogin.Format(time.DateTime)
	}
	return dto.AdminView{
		ID:        a.ID,
		Username:  a.Username,
		Nickname:  a.Nickname,
		Role:      a.Role,
		Status:    a.Status,
		LastLogin: last,
		CreatedAt: a.CreatedAt.Format(time.DateTime),
	}
}

// normRole 归一角色名，空则默认 admin（普通管理员）。
func normRole(r string) string {
	r = strings.TrimSpace(r)
	if r == "" {
		return "admin"
	}
	return r
}

// normStatus 归一状态为 0/1，默认 1。
func normStatus(s int8) int8 {
	if s == 0 {
		return 0
	}
	return 1
}
