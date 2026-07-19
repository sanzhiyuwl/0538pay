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
}

func NewAuthService(repo *repository.AdminRepo, jm *jwtauth.Manager) *AuthService {
	return &AuthService{repo: repo, jm: jm}
}

// Login 校验凭据并签发 token。
func (s *AuthService) Login(req dto.LoginReq) (*dto.LoginResp, error) {
	admin, err := s.repo.FindByUsername(req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredential
		}
		return nil, err
	}
	if admin.Status != 1 {
		return nil, errors.New("账号已停用")
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.Password)) != nil {
		return nil, ErrInvalidCredential
	}

	token, err := s.jm.Generate(admin.ID, admin.Username, admin.Role, "admin")
	if err != nil {
		return nil, err
	}
	return &dto.LoginResp{Token: token, Nickname: admin.Nickname, Role: admin.Role}, nil
}
