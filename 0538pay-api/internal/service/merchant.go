package service

import (
	"strings"
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
	"github.com/0538pay/api/pkg/money"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

// groupNameCache 用户组名缓存：List 时按需从用户组表刷新，派生 groupname 列。
// 默认组 gid=0 固定为「默认用户组」。
var groupNameCache = map[int]string{0: "默认用户组"}

func groupName(gid int) string {
	if n, ok := groupNameCache[gid]; ok {
		return n
	}
	return "默认用户组"
}

// MerchantService 商户业务逻辑。
type MerchantService struct {
	repo        *repository.MerchantRepo
	accounts    *repository.AccountRepo
	groups      *repository.GroupRepo
	subchannels *repository.SubChannelRepo // 删商户级联删子通道（可空，SetSubChannelRepo 注入）
}

func NewMerchantService(repo *repository.MerchantRepo, accounts *repository.AccountRepo, groups *repository.GroupRepo) *MerchantService {
	return &MerchantService{repo: repo, accounts: accounts, groups: groups}
}

// SetSubChannelRepo 注入子通道 repo，删商户时级联删其子通道（对齐 epay delUser 级联）。
func (s *MerchantService) SetSubChannelRepo(r *repository.SubChannelRepo) { s.subchannels = r }

// MerchantError 携带业务错误码与提示，handler 据此返回 code+msg。
type MerchantError struct {
	Code int
	Msg  string
}

func (e *MerchantError) Error() string { return e.Msg }

func meErr(msg string) *MerchantError { return &MerchantError{Code: 1003, Msg: msg} }

// refreshGroupNames 从用户组表刷新组名缓存（List 前调用，派生 groupname）。
func (s *MerchantService) refreshGroupNames() {
	if s.groups == nil {
		return
	}
	list, err := s.groups.All()
	if err != nil {
		return
	}
	for i := range list {
		groupNameCache[list[i].GID] = list[i].Name
	}
}

// List 返回分页商户（转对外 View：金额补两位小数、时间格式化、派生组名）。
func (s *MerchantService) List(q dto.MerchantQuery) ([]dto.MerchantView, int64, error) {
	q.Normalize()
	s.refreshGroupNames()
	list, total, err := s.repo.List(q)
	if err != nil {
		return nil, 0, err
	}
	// 用户组到期惰性降级：endtime 过期 → 该商户回落默认组（对齐 epay userList）。
	now := time.Now()
	for i := range list {
		m := &list[i]
		if m.GID != 0 && m.GroupEnd != nil && m.GroupEnd.Before(now) {
			if err := s.repo.UpdateFields(m.UID, map[string]interface{}{"gid": 0, "group_end": nil}); err == nil {
				m.GID = 0
				m.GroupEnd = nil
			}
		}
	}
	views := make([]dto.MerchantView, 0, len(list))
	for i := range list {
		views = append(views, toMerchantView(&list[i]))
	}
	return views, total, nil
}

func toMerchantView(m *model.Merchant) dto.MerchantView {
	v := dto.MerchantView{
		UID:       m.UID,
		GID:       m.GID,
		GroupName: groupName(m.GID),
		Money:     money.String(m.Money),
		SettleID:  m.SettleID,
		Account:   m.Account,
		Username:  m.Username,
		QQ:        m.QQ,
		Phone:     m.Phone,
		Email:     m.Email,
		URL:       m.URL,
		AddTime:   m.AddTime.Format(timeLayout),
		Status:    m.Status,
		Cert:      m.Cert,
		Pay:       m.Pay,
		Settle:    m.Settle,
		UpID:      m.UpID,
		Mode:      m.Mode,
		Deposit:   money.String(m.Deposit),
	}
	if m.GroupEnd != nil {
		s := m.GroupEnd.Format(timeLayout)
		v.EndTime = &s
	}
	return v
}

// ===== 写操作（对齐 epay ajax_user.php）=====

// Create 后台添加商户（对齐 epay addUser）。
// 校验：手机号/邮箱不能同时为空、各自唯一；密钥后端随机 32 位；密码可选 bcrypt。
// 返回新建 uid 与通信密钥。
func (s *MerchantService) Create(req dto.MerchantCreateReq) (uint, string, error) {
	phone := strings.TrimSpace(req.Phone)
	email := strings.TrimSpace(req.Email)
	if phone == "" && email == "" {
		return 0, "", meErr("手机号和邮箱不能都为空")
	}
	if phone != "" {
		n, err := s.repo.CountByPhone(phone, 0)
		if err != nil {
			return 0, "", err
		}
		if n > 0 {
			return 0, "", meErr("手机号已存在")
		}
	}
	if email != "" {
		n, err := s.repo.CountByEmail(email, 0)
		if err != nil {
			return 0, "", err
		}
		if n > 0 {
			return 0, "", meErr("邮箱已存在")
		}
	}
	if req.SettleID < 1 || req.SettleID > 5 {
		req.SettleID = 1
	}
	key, err := randomHex(32)
	if err != nil {
		return 0, "", err
	}
	m := &model.Merchant{
		GID:      req.GID,
		AppKey:   key,
		SettleID: req.SettleID,
		Account:  strings.TrimSpace(req.Account),
		Username: strings.TrimSpace(req.Username),
		Money:    decimal.Zero,
		URL:      strings.TrimSpace(req.URL),
		Email:    email,
		QQ:       strings.TrimSpace(req.QQ),
		Phone:    phone,
		Mode:     req.Mode,
		Cert:     0,
		Pay:      req.Pay,
		Settle:   req.Settle,
		Status:   req.Status,
		AddTime:  time.Now(),
	}
	if pwd := strings.TrimSpace(req.Password); pwd != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
		if err != nil {
			return 0, "", err
		}
		m.Password = string(hash)
	}
	if err := s.repo.Create(m); err != nil {
		return 0, "", err
	}
	return m.UID, key, nil
}

// Update 后台编辑商户（对齐 epay editUser）。money 直接覆盖（管理员强改，不产生流水）。
// 密码非空则改密。编辑不做手机/邮箱查重（对齐 epay editUser）。
func (s *MerchantService) Update(uid uint, req dto.MerchantEditReq) error {
	m, err := s.repo.FindByUIDSafe(uid)
	if err != nil {
		return err
	}
	if m == nil {
		return meErr("商户不存在")
	}
	if req.SettleID < 1 || req.SettleID > 5 {
		return meErr("结算方式不合法")
	}
	fields := map[string]interface{}{
		"gid":       req.GID,
		"upid":      req.UpID,
		"settle_id": req.SettleID,
		"account":   strings.TrimSpace(req.Account),
		"username":  strings.TrimSpace(req.Username),
		"url":       strings.TrimSpace(req.URL),
		"email":     strings.TrimSpace(req.Email),
		"qq":        strings.TrimSpace(req.QQ),
		"phone":     strings.TrimSpace(req.Phone),
		"mode":      req.Mode,
		"pay":       req.Pay,
		"settle":    req.Settle,
		"status":    req.Status,
	}
	// money 直接覆盖（管理员强改旁路，不走 changeUserMoney 流水，对齐 epay editUser）
	if strings.TrimSpace(req.Money) != "" {
		mv, err := decimal.NewFromString(strings.TrimSpace(req.Money))
		if err != nil || mv.IsNegative() {
			return meErr("余额格式不正确")
		}
		fields["money"] = mv
	}
	if pwd := strings.TrimSpace(req.Password); pwd != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		fields["password"] = string(hash)
	}
	return s.repo.UpdateFields(uid, fields)
}

// Recharge 后台余额充值/扣除（对齐 epay recharge）。走 changeUserMoney：事务+行锁+流水。
// action=0 充值(加款,'后台加款')；action=1 扣除(扣款,'后台扣款')，扣款额截断到不超过当前余额。
func (s *MerchantService) Recharge(uid uint, req dto.MerchantRechargeReq) error {
	m, err := s.repo.FindByUIDSafe(uid)
	if err != nil {
		return err
	}
	if m == nil {
		return meErr("商户不存在")
	}
	amount, err := decimal.NewFromString(strings.TrimSpace(req.Amount))
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		return meErr("请输入有效的金额")
	}
	if req.Action == 1 {
		// 扣款不能超过余额，自动截断（对齐 epay recharge do=1）
		if amount.GreaterThan(m.Money) {
			amount = m.Money
		}
		return s.accounts.ChangeUserMoney(uid, amount, false, "后台扣款", "")
	}
	return s.accounts.ChangeUserMoney(uid, amount, true, "后台加款", "")
}

// SetGroup 修改商户用户组 + 有效期（对齐 epay setUserGroup / changeUserGroup）。
// endtime 空 → 永久(NULL)。
func (s *MerchantService) SetGroup(uid uint, req dto.MerchantGroupReq) error {
	m, err := s.repo.FindByUIDSafe(uid)
	if err != nil {
		return err
	}
	if m == nil {
		return meErr("商户不存在")
	}
	fields := map[string]interface{}{"gid": req.GID}
	if strings.TrimSpace(req.EndTime) == "" {
		fields["group_end"] = nil
	} else {
		t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(req.EndTime), time.Local)
		if err != nil {
			return meErr("到期时间格式不正确")
		}
		fields["group_end"] = t
	}
	return s.repo.UpdateFields(uid, fields)
}

// SetStatus 商户权限/状态切换（对齐 epay setUser：type 分派）。
// field=user→status / pay→pay / settle→settle。
func (s *MerchantService) SetStatus(uid uint, req dto.MerchantSetStatusReq) error {
	m, err := s.repo.FindByUIDSafe(uid)
	if err != nil {
		return err
	}
	if m == nil {
		return meErr("商户不存在")
	}
	col := "status"
	switch req.Field {
	case "pay":
		col = "pay"
	case "settle":
		col = "settle"
	case "user", "", "status":
		col = "status"
	default:
		return meErr("未知的状态类型")
	}
	return s.repo.UpdateFields(uid, map[string]interface{}{col: req.Status})
}

// ResetKey 重置商户 MD5 通信密钥（对齐 epay resetKey）。随机 32 位，原密钥立即失效。
func (s *MerchantService) ResetKey(uid uint) (string, error) {
	m, err := s.repo.FindByUIDSafe(uid)
	if err != nil {
		return "", err
	}
	if m == nil {
		return "", meErr("商户不存在")
	}
	key, err := randomHex(32)
	if err != nil {
		return "", err
	}
	if err := s.repo.UpdateFields(uid, map[string]interface{}{"app_key": key}); err != nil {
		return "", err
	}
	return key, nil
}

// Delete 删除商户（对齐 epay delUser）。
func (s *MerchantService) Delete(uid uint) error {
	m, err := s.repo.FindByUIDSafe(uid)
	if err != nil {
		return err
	}
	if m == nil {
		return meErr("商户不存在")
	}
	if err := s.repo.Delete(uid); err != nil {
		return err
	}
	// 级联删该商户的子通道（对齐 epay delUser 级联）。失败不回滚商户删除，仅记录。
	if s.subchannels != nil {
		_ = s.subchannels.DeleteByMerchant(uid)
	}
	return nil
}
