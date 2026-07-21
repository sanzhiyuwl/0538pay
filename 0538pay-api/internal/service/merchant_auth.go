package service

import (
	"strconv"
	"strings"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
	"github.com/0538pay/api/pkg/jwtauth"
	"golang.org/x/crypto/bcrypt"
)

// MerchantAuthService 处理商户端登录。
// 认证语义对齐 epay user/login.php（密码/密钥双模式 + type 自动纠偏 + 状态守卫），
// 但登录态载体用项目统一的 JWT(scope=merchant)，不移植 epay 的 RC4 cookie。
type MerchantAuthService struct {
	repo *repository.MerchantRepo
	jm   *jwtauth.Manager
}

func NewMerchantAuthService(repo *repository.MerchantRepo, jm *jwtauth.Manager) *MerchantAuthService {
	return &MerchantAuthService{repo: repo, jm: jm}
}

// MerchantAuthError 携带业务提示，handler 统一返回 code=1101。
type MerchantAuthError struct{ Msg string }

func (e *MerchantAuthError) Error() string { return e.Msg }

func maErr(msg string) *MerchantAuthError { return &MerchantAuthError{Msg: msg} }

// Login 校验商户凭据并签发 token。
func (s *MerchantAuthService) Login(req dto.MerchantLoginReq) (*dto.MerchantLoginResp, error) {
	account := strings.TrimSpace(req.Account)
	if account == "" || req.Password == "" {
		return nil, maErr("账号或密码不能为空")
	}

	// type 自动纠偏：密码登录但账号是纯数字且长度≤6，视为商户ID密钥登录（对齐 epay）。
	loginType := req.Type
	if loginType == 1 && isNumeric(account) && len(account) <= 6 {
		loginType = 0
	}

	var m *model.Merchant
	var err error
	if loginType == 1 {
		// 密码登录：按邮箱/手机查，bcrypt 校验密码
		m, err = s.repo.FindByLoginAccount(account)
		if err != nil {
			return nil, err
		}
		if m == nil {
			return nil, maErr("账号或密码不正确")
		}
		if m.Password == "" {
			return nil, maErr("该商户未设置登录密码，请用密钥登录")
		}
		if bcrypt.CompareHashAndPassword([]byte(m.Password), []byte(req.Password)) != nil {
			return nil, maErr("账号或密码不正确")
		}
	} else {
		// 密钥登录：按商户ID查，明文比对通信密钥（对齐 epay key 明文比对）
		uid, ok := parseUID(account)
		if !ok {
			return nil, maErr("商户ID格式不正确")
		}
		m, err = s.repo.FindByUIDSafe(uid)
		if err != nil {
			return nil, err
		}
		if m == nil || m.AppKey == "" || m.AppKey != req.Password {
			return nil, maErr("商户ID或密钥不正确")
		}
	}

	// 状态守卫：封禁不可登录（对齐 epay status=0 封禁；未审核 2 允许登录但功能受限）。
	if m.Status == 0 {
		return nil, maErr("账号已被封禁，请联系平台")
	}

	token, err := s.jm.Generate(m.UID, merchantName(m.UID), "merchant", "merchant")
	if err != nil {
		return nil, err
	}
	return &dto.MerchantLoginResp{Token: token, Info: toMerchantInfo(m)}, nil
}

// CurrentInfo 返回当前登录商户信息（GET /api/merchant/info）。
func (s *MerchantAuthService) CurrentInfo(uid uint) (*dto.MerchantInfo, error) {
	m, err := s.repo.FindByUIDSafe(uid)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, maErr("商户不存在")
	}
	info := toMerchantInfo(m)
	return &info, nil
}

func toMerchantInfo(m *model.Merchant) dto.MerchantInfo {
	return dto.MerchantInfo{
		UID:      m.UID,
		Name:     merchantName(m.UID),
		Money:    m.Money.StringFixed(2),
		Status:   m.Status,
		Pay:      m.Pay,
		Settle:   m.Settle,
		Cert:     m.Cert,
		Email:    m.Email,
		Phone:    m.Phone,
		QQ:       m.QQ,
		GID:      m.GID,
		SettleID: m.SettleID,
		Account:  m.Account,
		Username: m.Username,
		URL:      m.URL,
		Mode:     m.Mode,
	}
}

// merchantName 商户显示名（暂用 uid 派生，接商户名字段后补）。
func merchantName(uid uint) string {
	return "商户" + strconv.FormatUint(uint64(uid), 10)
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func parseUID(s string) (uint, bool) {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil || n == 0 {
		return 0, false
	}
	return uint(n), true
}
