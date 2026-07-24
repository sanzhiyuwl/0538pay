package service

import (
	"errors"
	"strings"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/repository"
	"github.com/epvia/api/pkg/jwtauth"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrInvalidCredential = errors.New("用户名或密码错误")

// ErrLoginLocked 登录失败次数超限（对齐 epay 防爆破锁定）。
var ErrLoginLocked = errors.New("多次登录失败，账号已被暂时锁定，请 24 小时后重试或联系管理员")

// AuthService 处理后台登录。
type AuthService struct {
	repo *repository.AdminRepo
	jm   *jwtauth.Manager
	log  *LogService    // 登录日志（可空）
	cfg  *ConfigService // 登录失败次数上限 login_limit_count（可空，默认 5）
}

func NewAuthService(repo *repository.AdminRepo, jm *jwtauth.Manager) *AuthService {
	return &AuthService{repo: repo, jm: jm}
}

// SetLogService 注入登录日志服务。
func (s *AuthService) SetLogService(l *LogService) { s.log = l }

// SetConfigService 注入配置域（读取 login_limit_count 登录失败次数上限）。
func (s *AuthService) SetConfigService(c *ConfigService) { s.cfg = c }

// loginLimitCount 登录失败次数上限（对齐 epay login_limit_count，默认 5；0 或负=不限）。
func (s *AuthService) loginLimitCount() int {
	if s.cfg == nil {
		return 5
	}
	return s.cfg.Int("login_limit_count", 5)
}

// Login 校验凭据并签发 token。ip 用于登录日志。
// 防爆破：同 IP 近 24h 登录失败达 login_limit_count 次即锁定（对齐 epay admin/login.php:23-26）。
func (s *AuthService) Login(req dto.LoginReq, ip string) (*dto.LoginResp, error) {
	if limit := s.loginLimitCount(); limit > 0 && s.log != nil {
		if s.log.CountFailByIPLastDay(ip) >= int64(limit) {
			return nil, ErrLoginLocked
		}
	}
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

// authErr 后台账号业务错误（handler 统一转 code=1 提示）。
type authErr struct{ msg string }

func (e *authErr) Error() string { return e.msg }
func aErr(msg string) error      { return &authErr{msg: msg} }

// ChangePassword 修改当前管理员登录密码（对齐 epay set.php mod=account_n 改密分支）。
// 校验旧密码 → 新密码两次一致 + 长度 → bcrypt 存新哈希。
func (s *AuthService) ChangePassword(adminID uint, oldPwd, newPwd, newPwd2 string) error {
	admin, err := s.repo.FindByID(adminID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(oldPwd)) != nil {
		return aErr("旧密码不正确")
	}
	if len(newPwd) < 6 {
		return aErr("新密码至少 6 位")
	}
	if newPwd != newPwd2 {
		return aErr("两次输入的新密码不一致")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(adminID, string(hash))
}

// Profile 返回当前管理员的账号资料（账号设置弹窗回填）。
func (s *AuthService) Profile(adminID uint) (*dto.AdminProfile, error) {
	admin, err := s.repo.FindByID(adminID)
	if err != nil {
		return nil, err
	}
	return &dto.AdminProfile{
		ID: admin.ID, Username: admin.Username, Nickname: admin.Nickname, Role: admin.Role,
	}, nil
}

// UpdateProfile 修改当前管理员昵称（用户名唯一，改名需查重；对齐 epay 保存 admin_user）。
func (s *AuthService) UpdateProfile(adminID uint, nickname, username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return aErr("用户名不能为空")
	}
	n, err := s.repo.CountByUsername(username, adminID)
	if err != nil {
		return err
	}
	if n > 0 {
		return aErr("该用户名已被占用")
	}
	return s.repo.UpdateFields(adminID, map[string]interface{}{
		"nickname": strings.TrimSpace(nickname),
		"username": username,
	})
}
