package service

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
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
	notice  *NoticeService // 新注册待审核管理员通知（可空；SetNoticeService 注入）
	pay     *PayService    // 付费注册下单/回调建号（可空；SetPayService 注入。B1-51）
}

func NewMerchantRegService(repo *repository.MerchantRepo, cfg *ConfigService, invite *InviteService, captcha *CaptchaService) *MerchantRegService {
	return &MerchantRegService{repo: repo, cfg: cfg, invite: invite, captcha: captcha}
}

// SetNoticeService 注入对外通知中枢（K-1）。注册需审核时发 regaudit 场景管理员通知。
func (s *MerchantRegService) SetNoticeService(n *NoticeService) { s.notice = n }

// SetPayService 注入收单服务（B1-51 付费注册：下 tid=1 注册费订单 + 回调建号）。
func (s *MerchantRegService) SetPayService(p *PayService) { s.pay = p }

// regPayInfo 付费注册待建号信息，序列化进 tid=1 订单 param，回调成功后 FinalizeRegPay 读回建号
// （对齐 epay CACHE reg_<trade_no> 承载的 verifytype/email/phone/pwd/upid/invitecodeid）。
type regPayInfo struct {
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Pwd        string `json:"pwd"` // 明文，回调建号时再 bcrypt（对齐 epay 缓存明文 pwd + getMd5Pwd）
	UpID       int    `json:"upid"`
	InviteID   uint   `json:"invitecodeid,omitempty"`
	RegPayDone bool   `json:"-"`
}

// registerPay 付费注册下单（reg_pay=1，对齐 epay user/ajax.php:237-256）：
// 校验收款商户 reg_pay_uid 存在 → 下 tid=1 注册费订单（金额 reg_pay_price，param 挂注册信息）→
// 返回待付订单信息（NeedPay=true）。用户支付成功后回调走 FinalizeRegPay 建号。
func (s *MerchantRegService) registerPay(req dto.MerchantRegReq, email, phone string, upid int, inviteID uint) (*dto.MerchantRegResp, error) {
	if s.pay == nil {
		return nil, maErr("付费注册未启用")
	}
	payUID := uint(s.cfg.Int("reg_pay_uid", 0))
	if payUID == 0 {
		return nil, maErr("注册收款商户ID未配置")
	}
	if up, e := s.repo.FindByUID(payUID); e != nil || up == nil {
		return nil, maErr("注册收款商户ID不存在")
	}
	price := s.cfg.Dec("reg_pay_price", decimal.Zero)
	if price.LessThanOrEqual(decimal.Zero) {
		return nil, maErr("注册付费金额未配置")
	}
	plugin := strings.TrimSpace(req.Plugin)
	if plugin == "" {
		return nil, maErr("请选择支付方式")
	}
	info := regPayInfo{Email: email, Phone: phone, Pwd: req.Password, UpID: upid, InviteID: inviteID}
	raw, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}
	// 下 tid=1 订单，收款方 param.uid=reg_pay_uid（内部业务方），param 同时挂注册信息。
	resp, err := s.pay.CreateInternalOrder(context.Background(), payUID, 1, "商户申请", price, plugin, string(raw))
	if err != nil {
		return nil, err
	}
	return &dto.MerchantRegResp{NeedPay: true, Pay: resp, Msg: "订单创建成功，请完成支付以完成注册"}, nil
}

// FinalizeRegPay 付费注册订单支付成功后的建号钩子（对齐 epay processOrder tid==1）。
// 由收单回调在 tid==1 入账后调用：读订单 param 里的注册信息 → 建号（bcrypt/密钥/审核态/核销邀请码/通知）。
// 幂等由回调侧保证（订单只翻转一次）；param 缺失或已建号则静默跳过。
func (s *MerchantRegService) FinalizeRegPay(param string) error {
	var info regPayInfo
	if strings.TrimSpace(param) == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(param), &info); err != nil {
		return nil // param 非注册信息（非付费注册订单），跳过
	}
	if info.Email == "" && info.Phone == "" {
		return nil
	}
	_, _, err := s.createMerchantAccount(info.Email, info.Phone, info.Pwd, info.UpID, info.InviteID)
	return err
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

	// 邀请返现关系：推广码 ref 解出上级 uid（对齐 epay reg 从 ?invite= 解出 upid）。
	// 上级须真实存在且非自身，否则 upid=0（无邀请关系）。
	upid := 0
	if ref := strings.TrimSpace(req.Ref); ref != "" {
		if id := decodeInviteUID(ref); id > 0 {
			if up, e := s.repo.FindByUID(id); e == nil && up != nil {
				upid = int(id)
			}
		}
	}

	// 5. 付费注册（reg_pay=1）：不立即建号，先落 tid=1 订单，把注册信息挂到订单 param，
	//    回调成功后由 FinalizeRegPay 建号（对齐 epay user/ajax.php reg_pay 分支 + processOrder tid==1）。
	if s.cfg.Int("reg_pay", 0) == 1 {
		return s.registerPay(req, email, phone, upid, inviteID)
	}

	// 6. 直接建号（免费注册）：审核态(user_review→pay) + bcrypt 密码 + 32 位密钥 + 核销邀请码 + 通知。
	m, needReview, err := s.createMerchantAccount(email, phone, req.Password, upid, inviteID)
	if err != nil {
		return nil, err
	}

	msg := "注册成功，请登录"
	if needReview {
		msg = "注册成功，请等待管理员审核后开通支付"
	}
	return &dto.MerchantRegResp{UID: m.UID, NeedReview: needReview, Msg: msg}, nil
}

// createMerchantAccount 落一条商户账号（免费注册 & 付费注册回调建号共用，对齐 epay INSERT pre_user）。
// 生成 32 位密钥 + bcrypt 密码，审核态 user_review→pay(1正常/2待审)，status 恒 1，核销邀请码，
// 审核态发 regaudit 管理员通知。返回创建的商户与是否待审核。
func (s *MerchantRegService) createMerchantAccount(email, phone, plainPwd string, upid int, inviteID uint) (*model.Merchant, bool, error) {
	key, err := randomHex(32)
	if err != nil {
		return nil, false, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPwd), bcrypt.DefaultCost)
	if err != nil {
		return nil, false, err
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
		UpID:     upid,
		AddTime:  time.Now(),
	}
	if err := s.repo.Create(m); err != nil {
		return nil, false, err
	}
	if inviteID > 0 {
		_ = s.invite.MarkUsed(inviteID, m.UID)
	}
	if needReview && s.notice != nil {
		account := email
		if account == "" {
			account = phone
		}
		go s.notice.Send("regaudit", 0, map[string]string{
			"uid":     strconv.FormatUint(uint64(m.UID), 10),
			"account": account,
		})
	}
	return m, needReview, nil
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
