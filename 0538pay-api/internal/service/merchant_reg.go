package service

import (
	"regexp"
	"strings"
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

// emailRe 邮箱格式（对齐 epay 正则）。
var emailRe = regexp.MustCompile(`^[A-Za-z0-9._-]+@[A-Za-z0-9._-]+\.[A-Za-z0-9._-]+$`)

// MerchantRegService 商户自助流程：注册 / 完善资料 / 找回密码（对齐 epay user/reg|completeinfo|findpwd）。
// 验证码用自研图形验证码(captcha)代替 epay 短信/邮箱 OTP；密码用 bcrypt(与现有登录一致，不移植 epay md5)。
type MerchantRegService struct {
	repo    *repository.MerchantRepo
	cfg     *ConfigService
	invite  *InviteService
	captcha *CaptchaService
}

func NewMerchantRegService(repo *repository.MerchantRepo, cfg *ConfigService, invite *InviteService, captcha *CaptchaService) *MerchantRegService {
	return &MerchantRegService{repo: repo, cfg: cfg, invite: invite, captcha: captcha}
}

// checkPassword 密码规则（对齐 epay：长度≥6、不等于账号、非纯数字）。
func checkPassword(pwd, account string) error {
	if len(pwd) < 6 {
		return maErr("密码长度不能少于 6 位")
	}
	if pwd == account {
		return maErr("密码不能与账号相同")
	}
	if isNumeric(pwd) {
		return maErr("密码不能是纯数字")
	}
	return nil
}

// Register 商户注册。对齐 epay user/reg：reg_open 分支 + 图形验证码 + 邀请码校验核销 +
// user_review→pay 字段 + bcrypt 密码 + 32 位密钥。返回是否待审核。
func (s *MerchantRegService) Register(req dto.MerchantRegReq) (*dto.MerchantRegResp, error) {
	// 1. reg_open：0关/1开/2仅邀请
	regOpen := s.cfg.Int("reg_open", 1)
	if regOpen == 0 {
		return nil, maErr("未开放商户申请")
	}

	// 2. 图形验证码
	if !s.captcha.Verify(req.CaptchaToken, req.Captcha) {
		return nil, maErr("验证码错误或已过期")
	}

	account := strings.TrimSpace(req.Account)
	if account == "" {
		return nil, maErr("账号不能为空")
	}
	if err := checkPassword(req.Password, account); err != nil {
		return nil, err
	}

	// 3. 账号格式 + 重复校验（verifytype: 0邮箱 1手机）
	var email, phone string
	if req.VerifyType == 1 {
		if !isNumeric(account) || len(account) != 11 {
			return nil, maErr("手机号格式不正确")
		}
		phone = account
		n, err := s.repo.CountByPhone(phone, 0)
		if err != nil {
			return nil, err
		}
		if n > 0 {
			return nil, maErr("该手机号已注册")
		}
	} else {
		if !emailRe.MatchString(account) {
			return nil, maErr("邮箱格式不正确")
		}
		email = account
		n, err := s.repo.CountByEmail(email, 0)
		if err != nil {
			return nil, err
		}
		if n > 0 {
			return nil, maErr("该邮箱已注册")
		}
	}

	// 4. 邀请码校验（reg_open==2 强制），先校验不核销
	var inviteID uint
	if regOpen == 2 {
		code := strings.TrimSpace(req.Invite)
		if code == "" {
			return nil, maErr("当前为仅邀请注册，请填写邀请码")
		}
		id, ok := s.invite.Verify(code)
		if !ok {
			return nil, maErr("邀请码不存在或已被使用")
		}
		inviteID = id
	}

	// 5. 密钥 + bcrypt 密码 + 审核态(user_review→pay)
	key, err := randomHex(32)
	if err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	pay := int8(1)
	needReview := s.cfg.Int("user_review", 0) == 1
	if needReview {
		pay = 2 // 待审核（对齐 epay：审核态落 pay 字段，status 恒 1）
	}
	m := &model.Merchant{
		GID:      0,
		AppKey:   key,
		Password: string(hash),
		Email:    email,
		Phone:    phone,
		Money:    decimal.Zero,
		SettleID: 1,
		Status:   1, // 恒为 1（对齐 epay）
		Pay:      pay,
		Settle:   1,
		AddTime:  time.Now(),
	}
	if err := s.repo.Create(m); err != nil {
		return nil, err
	}

	// 6. 核销邀请码
	if inviteID > 0 {
		_ = s.invite.MarkUsed(inviteID, m.UID)
	}

	msg := "注册成功，请登录"
	if needReview {
		msg = "注册成功，请等待管理员审核后开通支付"
	}
	return &dto.MerchantRegResp{UID: m.UID, NeedReview: needReview, Msg: msg}, nil
}

// Complete 完善资料。对齐 epay completeinfo：结算方式/账号/姓名/QQ/域名校验 + 入库。
// 已完善(account+username 都非空)则拒绝重复。
func (s *MerchantRegService) Complete(uid uint, req dto.MerchantCompleteReq) error {
	m, err := s.repo.FindByUIDSafe(uid)
	if err != nil {
		return err
	}
	if m == nil {
		return maErr("商户不存在")
	}
	if m.Account != "" && m.Username != "" {
		return maErr("你已完善相关信息")
	}
	account := strings.TrimSpace(req.Account)
	username := strings.TrimSpace(req.Username)
	if account == "" || username == "" {
		return maErr("收款账号和真实姓名不能为空")
	}
	stype := req.SettleID
	if stype < 1 || stype > 5 {
		stype = 1
	}
	// 分类型校验收款账号（对齐 epay）
	switch stype {
	case 1: // 支付宝：手机号(11) 或 邮箱
		if len(account) != 11 && !strings.Contains(account, "@") {
			return maErr("支付宝账号应为手机号或邮箱")
		}
	case 2: // 微信
		if len(account) < 3 {
			return maErr("微信账号格式不正确")
		}
	case 3: // QQ钱包：5~10 位数字
		if !isNumeric(account) || len(account) < 5 || len(account) > 10 {
			return maErr("QQ 账号格式不正确")
		}
	}
	fields := map[string]interface{}{
		"settle_id": stype,
		"account":   account,
		"username":  username,
	}
	if qq := strings.TrimSpace(req.QQ); qq != "" {
		if !isNumeric(qq) || len(qq) < 5 || len(qq) > 10 {
			return maErr("QQ 号格式不正确")
		}
		fields["qq"] = qq
	}
	if url := normalizeDomain(req.URL); url != "" {
		if len(url) < 4 || !strings.Contains(url, ".") {
			return maErr("网站域名格式不正确")
		}
		fields["url"] = url
	}
	// 手机注册模式可补邮箱（查重）
	if m.Phone != "" && strings.TrimSpace(req.Email) != "" {
		email := strings.TrimSpace(req.Email)
		if !emailRe.MatchString(email) {
			return maErr("邮箱格式不正确")
		}
		n, err := s.repo.CountByEmail(email, uid)
		if err != nil {
			return err
		}
		if n > 0 {
			return maErr("该邮箱已被其他商户绑定")
		}
		fields["email"] = email
	}
	return s.repo.UpdateFields(uid, fields)
}

// FindPwd 找回密码。对齐 epay findpwd：账号存在校验 + 图形验证码 + bcrypt 重置。
// 短信/邮箱 OTP 待凭证，本实现用图形验证码守卫（防刷）。
func (s *MerchantRegService) FindPwd(req dto.MerchantFindPwdReq) error {
	if !s.captcha.Verify(req.CaptchaToken, req.Captcha) {
		return maErr("验证码错误或已过期")
	}
	account := strings.TrimSpace(req.Account)
	if account == "" {
		return maErr("账号不能为空")
	}
	if err := checkPassword(req.Password, account); err != nil {
		return err
	}
	// 账号存在校验（找回与注册相反：必须已注册）
	m, err := s.repo.FindByLoginAccount(account)
	if err != nil {
		return err
	}
	if m == nil {
		return maErr("该账号未找到注册商户")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdateFields(m.UID, map[string]interface{}{"password": string(hash)})
}

// normalizeDomain 去 http(s):// 前缀与尾部斜杠（对齐 epay 前端处理）。
func normalizeDomain(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimRight(s, "/")
	return s
}
