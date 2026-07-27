package service

import (
	"strings"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
)

// RoleError 角色业务错误（带前端可读消息）。
type RoleError struct{ Msg string }

func (e *RoleError) Error() string { return e.Msg }

// permAll 全部权限的特殊标记。
const permAll = "*"

// RoleService 后台角色管理（RBAC 增强）。
type RoleService struct {
	repo *repository.RoleRepo
}

func NewRoleService(repo *repository.RoleRepo) *RoleService {
	return &RoleService{repo: repo}
}

// builtinRoles 内置角色（首次访问时若表为空则播种）。code 与 Admin.Role 对应。
// super 为超级管理员，固定全部权限、不可删除、不可改权限。
var builtinRoles = []model.Role{
	{Code: "super", Name: "超级管理员", Description: "拥有全部权限，不可删除", Permissions: permAll, Builtin: true, Sort: 0},
	{Code: "operator", Name: "运营管理员", Description: "日常交易、商户、风控运营", Permissions: "dashboard,trade,merchant,risk", Builtin: false, Sort: 1},
	{Code: "finance", Name: "财务专员", Description: "结算、付款、账单等财务操作", Permissions: "dashboard,finance,trade", Builtin: false, Sort: 2},
	{Code: "service", Name: "客服专员", Description: "只读查看订单与商户", Permissions: "dashboard,trade,merchant", Builtin: false, Sort: 3},
}

// ensureSeed 表为空时播种内置角色（幂等：非空直接返回）。
func (s *RoleService) ensureSeed() error {
	n, err := s.repo.Count()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	for i := range builtinRoles {
		r := builtinRoles[i]
		if err := s.repo.Create(&r); err != nil {
			return err
		}
	}
	return nil
}

// toView model → DTO。
func toRoleView(r *model.Role) dto.RoleView {
	perms := []string{}
	if strings.TrimSpace(r.Permissions) != "" {
		perms = strings.Split(r.Permissions, ",")
	}
	return dto.RoleView{
		ID:          r.ID,
		Code:        r.Code,
		Name:        r.Name,
		Description: r.Description,
		Permissions: perms,
		Builtin:     r.Builtin,
	}
}

// List 列出全部角色（首次访问播种内置角色）。
func (s *RoleService) List() ([]dto.RoleView, error) {
	if err := s.ensureSeed(); err != nil {
		return nil, err
	}
	list, err := s.repo.All()
	if err != nil {
		return nil, err
	}
	views := make([]dto.RoleView, 0, len(list))
	for i := range list {
		views = append(views, toRoleView(&list[i]))
	}
	return views, nil
}

// normPerms 归一权限列表：含 "*" 则收敛为 "*"；去空去重后逗号拼接。
func normPerms(perms []string) string {
	seen := map[string]bool{}
	out := []string{}
	for _, p := range perms {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == permAll {
			return permAll
		}
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return strings.Join(out, ",")
}

// Create 新增角色。code 唯一，不可用保留字 super。
func (s *RoleService) Create(req dto.RoleSaveReq) error {
	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" {
		return &RoleError{"角色代码和名称不能为空"}
	}
	if code == "super" {
		return &RoleError{"super 为超级管理员保留代码，不可用于新增"}
	}
	n, err := s.repo.CountByCode(code, 0)
	if err != nil {
		return err
	}
	if n > 0 {
		return &RoleError{"角色代码已存在"}
	}
	return s.repo.Create(&model.Role{
		Code:        code,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		Permissions: normPerms(req.Permissions),
		Builtin:     false,
	})
}

// Update 编辑角色。super 固定全部权限、code 不可改；内置角色可改权限但 code 锁定。
func (s *RoleService) Update(id uint, req dto.RoleSaveReq) error {
	role, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if role == nil {
		return &RoleError{"角色不存在"}
	}
	fields := map[string]interface{}{
		"name":        strings.TrimSpace(req.Name),
		"description": strings.TrimSpace(req.Description),
	}
	if fields["name"] == "" {
		return &RoleError{"角色名称不能为空"}
	}
	if role.Code == "super" {
		// 超级管理员固定全部权限，不接受权限调整。
		fields["permissions"] = permAll
	} else {
		fields["permissions"] = normPerms(req.Permissions)
	}
	return s.repo.Update(id, fields)
}

// Delete 删除角色。内置角色不可删；仍有管理员使用该角色时不可删。
func (s *RoleService) Delete(id uint) error {
	role, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if role == nil {
		return &RoleError{"角色不存在"}
	}
	if role.Builtin {
		return &RoleError{"内置角色不可删除"}
	}
	used, err := s.repo.CountAdminsByRole(role.Code)
	if err != nil {
		return err
	}
	if used > 0 {
		return &RoleError{"仍有管理员使用该角色，请先改派后再删除"}
	}
	return s.repo.Delete(id)
}
