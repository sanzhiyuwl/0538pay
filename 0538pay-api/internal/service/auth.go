package service

import (
	"errors"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/repository"
	"github.com/0538pay/api/pkg/jwtauth"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrInvalidCredential = errors.New("用户名或密码错误")

// AuthService 处理后台登录。
type AuthService struct {
	repo *repository.AdminRepo
	jm   *jwtauth.Manager
	log  *LogService // 登录日志（可空）
}

func NewAuthService(repo *repository.AdminRepo, jm *jwtauth.Manager) *AuthService {
	return &AuthService{repo: repo, jm: jm}
}

// SetLogService 注入登录日志服务。
func (s *AuthService) SetLogService(l *LogService) { s.log = l }

// Login 校验凭据并签发 token。ip 用于登录日志。
func (s *AuthService) Login(req dto.LoginReq, ip string) (*dto.LoginResp, error) {
	admin, err := s.repo.FindByUsername(req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.recordFail(ip)
			return nil, ErrInvalidCredential
		}
		return nil, err
	}
	if admin.Status != 1 {
		return nil, errors.New("账号已停用")
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.Password)) != nil {
		s.recordFail(ip)
		return nil, ErrInvalidCredential
	}

	token, err := s.jm.Generate(admin.ID, admin.Username, admin.Role, "admin")
	if err != nil {
		return nil, err
	}
	if s.log != nil {
		s.log.Record(0, "登录后台", ip, "") // uid=0 表示管理员
	}
	return &dto.LoginResp{Token: token, Nickname: admin.Nickname, Role: admin.Role}, nil
}

// recordFail 记录一条管理员登录失败日志（对齐 epay 防爆破日志）。
func (s *AuthService) recordFail(ip string) {
	if s.log != nil {
		s.log.Record(0, "登录失败", ip, "")
	}
}
