package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
	"github.com/shopspring/decimal"
)

// enrollPayTid 进件开户费内部订单 tid（付费前置，回调走 FinalizeEnrollPay 放行）。
// 沿用收单内部订单 tid 分派机制（tid=1 付费注册 / 2 充值 / 3 聚合 / 4 购组 / 5 保证金），
// 进件占用 tid=6。
const enrollPayTid = 6

// EnrollService 进件业务编排：进件单管理、邀请链接、结算查询。
// 平台端 /console 管所有代理的进件，代理端 /agent 只碰自己——共用本 service，
// 入参强制带 agentID（代理端传自己的 id，平台端传 nil 看全部）实现数据隔离。
type EnrollService struct {
	repo      *repository.EnrollRepo
	cfg       *ConfigService
	submch    *SubMerchantService   // 微信进件接口（提交/查状态），未接入前为 nil
	agentRepo *repository.AgentRepo // 名额钱包变动（路径一进件成功扣名额），未接入前为 nil
	pay       *PayService           // 收单下单（付费前置拉起开户费收款），未接入前为 nil
}

func NewEnrollService(repo *repository.EnrollRepo, cfg *ConfigService) *EnrollService {
	return &EnrollService{repo: repo, cfg: cfg}
}

// SetSubMerchant 注入微信进件 service（提交/查状态）。
func (s *EnrollService) SetSubMerchant(sm *SubMerchantService) { s.submch = sm }

// SetAgentRepo 注入代理名额仓储（路径一进件成功扣名额、写结算流水）。
func (s *EnrollService) SetAgentRepo(ar *repository.AgentRepo) { s.agentRepo = ar }

// SetPayService 注入收单服务（付费前置：创建进件单即下开户费收款单）。
func (s *EnrollService) SetPayService(p *PayService) { s.pay = p }

// EnrollError 携带业务提示。
type EnrollError struct{ Msg string }

func (e *EnrollError) Error() string { return e.Msg }

func enErr(msg string) *EnrollError { return &EnrollError{Msg: msg} }

// —— 进件单 ——

// ListEnrolls 分页查询进件单。
func (s *EnrollService) ListEnrolls(q repository.EnrollQuery) ([]model.SubMerchantEnroll, int64, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	return s.repo.ListEnrolls(q)
}

// GetEnroll 取进件单详情。agentID 非空时校验归属（代理端不能看别人的单）。
func (s *EnrollService) GetEnroll(id uint, agentID *uint) (*model.SubMerchantEnroll, error) {
	e, err := s.repo.FindEnroll(id)
	if err != nil {
		return nil, enErr("进件单不存在")
	}
	if agentID != nil && e.AgentID != *agentID {
		return nil, enErr("该进件单不属于您")
	}
	return e, nil
}

// —— 创建进件单（付费前置）——

// CreateEnrollReq 创建进件单入参（第一步只收最基础信息，付款成功后再填全套资料）。
type CreateEnrollReq struct {
	AgentID      uint   // 归属代理（0=平台自己）
	MerchantName string // 商户名称/主体名称
	ContactPhone string // 联系手机（明文，进度查询匹配用）
	Path         int    // 资金路径：1 预购名额 / 2 商户自付
	Source       int    // 发起方式：1平台代填 2代理代填 3客户自助
	InviteCode   string // 自助进件来源邀请码（source=3 回填）
	Plugin       string // 支付方式（收开户费的渠道 plugin，如 alipay/wxpay/mock）
}

// CreateEnroll 付费前置建单（对齐 docs-代理进件/01 第四节）：
// 建 pay_submch_enroll(status=pending_pay，金额=开户零售价) → 下开户费内部收款单(tid=6，
// param 挂进件单号) → 返回收银台信息给前端拉起支付。付款回调走 FinalizeEnrollPay 翻 paid。
// 路径一且后台配置"名额已含开户费、客户免付"(enroll_path1_charge=0) 时跳过收款直接 paid。
func (s *EnrollService) CreateEnroll(ctx context.Context, req CreateEnrollReq) (*model.SubMerchantEnroll, *dto.SubmitResp, error) {
	name := strings.TrimSpace(req.MerchantName)
	if name == "" {
		return nil, nil, enErr("请填写商户名称")
	}
	if req.Path != model.EnrollPathQuota && req.Path != model.EnrollPathSelf {
		req.Path = model.EnrollPathSelf
	}
	if req.Source == 0 {
		req.Source = model.EnrollSourcePlatform
	}

	retail := s.cfg.Dec("enroll_retail_price", decimal.Zero)

	// 路径一且配置客户免付开户费：跳过收款，建单即 paid 放行填料。
	freePath1 := req.Path == model.EnrollPathQuota && s.cfg.Str("enroll_path1_charge") == "0"

	e := &model.SubMerchantEnroll{
		EnrollNo:     genEnrollNo(),
		AgentID:      req.AgentID,
		MerchantName: name,
		ContactPhone: strings.TrimSpace(req.ContactPhone),
		Path:         req.Path,
		RetailAmount: retail,
		Status:       model.EnrollStatusPendingPay,
		Source:       req.Source,
		InviteCode:   strings.TrimSpace(req.InviteCode),
	}
	if freePath1 || retail.LessThanOrEqual(decimal.Zero) {
		e.Status = model.EnrollStatusPaid // 无需收费，直接放行填料
		if err := s.repo.CreateEnroll(e); err != nil {
			return nil, nil, err
		}
		return e, nil, nil
	}

	if s.pay == nil {
		return nil, nil, enErr("收款服务未就绪，无法收取开户费")
	}
	payUID := uint(s.cfg.Int("enroll_pay_uid", 0))
	if payUID == 0 {
		return nil, nil, enErr("进件开户费收款商户未配置（系统设置·进件设置 enroll_pay_uid）")
	}
	plugin := strings.TrimSpace(req.Plugin)
	if plugin == "" {
		return nil, nil, enErr("请选择支付方式")
	}

	if err := s.repo.CreateEnroll(e); err != nil {
		return nil, nil, err
	}

	// 下 tid=6 开户费收款单，param 挂进件单号；回调成功后 FinalizeEnrollPay 翻 paid。
	param, _ := json.Marshal(map[string]string{"enroll_no": e.EnrollNo})
	resp, err := s.pay.CreateInternalOrder(ctx, payUID, enrollPayTid, "特约商户进件开户费", retail, plugin, string(param))
	if err != nil {
		// 下单失败：回滚进件单，避免留一堆无法支付的脏 pending_pay 单。
		_ = s.repo.UpdateEnroll(e.ID, map[string]any{"status": model.EnrollStatusClosed})
		return nil, nil, enErr("拉起开户费收款失败: " + err.Error())
	}
	// 回填收款单号，便于对账/退款追溯。
	if resp != nil && resp.TradeNo != "" {
		_ = s.repo.UpdateEnroll(e.ID, map[string]any{"pay_order_no": resp.TradeNo})
		e.PayOrderNo = resp.TradeNo
	}
	return e, resp, nil
}

// —— 客户自助公开页（source=3，免登录，靠邀请 code）——

// PublicInviteInfo 公开页落地时返回的邀请信息（不含敏感字段，仅够渲染页面）。
type PublicInviteInfo struct {
	Code       string `json:"code"`        // 邀请码（回显）
	AgentName  string `json:"agent_name"`  // 归属代理名（页面展示"由 XX 提供进件服务"）
	RetailNote string `json:"retail_note"` // 开户价说明文案（读 config）
}

// ResolveInvite 公开页落地：按 code 校验邀请链接可用（启用+未失效），打点打开数，返回展示信息。
// 校验失败（不存在/停用/已失效）返回业务错误，前端据此显示"链接已失效"。agentName 供页面展示。
func (s *EnrollService) ResolveInvite(code string, agentName func(uint) string) (*PublicInviteInfo, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, enErr("邀请链接无效")
	}
	v, err := s.repo.FindInviteByCode(code)
	if err != nil {
		return nil, enErr("邀请链接不存在或已被删除")
	}
	if v.Status == model.InviteStatusExpired {
		return nil, enErr("邀请链接已失效，请联系为您服务的代理重新获取")
	}
	if v.Status != model.InviteStatusEnabled {
		return nil, enErr("邀请链接已停用，请联系为您服务的代理")
	}
	// 打点：打开数 +1，首次打开记 first_access_at。
	_ = s.repo.IncInviteOpen(v.ID, v.FirstAccessAt == nil)

	name := ""
	if agentName != nil {
		name = agentName(v.AgentID)
	}
	return &PublicInviteInfo{
		Code:       v.Code,
		AgentName:  name,
		RetailNote: "开户零售价以下单页展示为准",
	}, nil
}

// PublicCreateEnrollReq 客户自助建单入参（source=3）。资料从公开页表单收，敏感字段前端不加密、
// 由后端 SubMerchantService.EncryptSensitive 加密后落 material_json。第一步只收基础信息 + 付费。
type PublicCreateEnrollReq struct {
	Code         string // 邀请码（定位归属代理）
	MerchantName string // 商户名称
	ContactPhone string // 联系手机（明文，进度查询匹配）
	Plugin       string // 支付方式
}

// PublicCreateEnroll 客户自助建进件单：校验邀请可用 → 归属绑定该邀请的代理 → source=3 走 CreateEnroll
// （付费前置，路径固定为商户自付 EnrollPathSelf——自助客户没有代理名额）→ 打点提交数。
// 返回进件单 + 收银台信息（前端拉起支付）。
func (s *EnrollService) PublicCreateEnroll(ctx context.Context, req PublicCreateEnrollReq) (*model.SubMerchantEnroll, *dto.SubmitResp, error) {
	code := strings.TrimSpace(req.Code)
	v, err := s.repo.FindInviteByCode(code)
	if err != nil {
		return nil, nil, enErr("邀请链接不存在")
	}
	if v.Status != model.InviteStatusEnabled {
		return nil, nil, enErr("邀请链接已失效或停用，无法提交")
	}

	e, pay, err := s.CreateEnroll(ctx, CreateEnrollReq{
		AgentID:      v.AgentID,
		MerchantName: req.MerchantName,
		ContactPhone: req.ContactPhone,
		Path:         model.EnrollPathSelf, // 自助客户走商户自付
		Source:       model.EnrollSourceSelf,
		InviteCode:   code,
		Plugin:       req.Plugin,
	})
	if err != nil {
		return nil, nil, err
	}
	// 打点：提交数 +1。
	_ = s.repo.IncInviteSubmit(v.ID)
	return e, pay, nil
}

// FinalizeEnrollPay 进件开户费收款成功后的放行钩子（收单回调 tid=6 入账后调用）。
// 读订单 param 里的进件单号 → 把该单 pending_pay 翻 paid（放行第2步填全套资料/提交微信）。
// 幂等：仅 pending_pay→paid 翻一次；param 非进件订单或单不存在则静默跳过（不阻断收款入账）。
func (s *EnrollService) FinalizeEnrollPay(param string) error {
	if strings.TrimSpace(param) == "" {
		return nil
	}
	var p struct {
		EnrollNo string `json:"enroll_no"`
	}
	if err := json.Unmarshal([]byte(param), &p); err != nil || p.EnrollNo == "" {
		return nil // 非进件订单
	}
	e, err := s.repo.FindEnrollByNo(p.EnrollNo)
	if err != nil {
		return nil // 单不存在，跳过（不阻断收款）
	}
	if e.Status != model.EnrollStatusPendingPay {
		return nil // 已翻过或已终态，幂等跳过
	}
	return s.repo.UpdateEnroll(e.ID, map[string]any{"status": model.EnrollStatusPaid})
}

// CloseTimeoutPending 定时任务：把创建超过 enroll_pay_timeout 分钟仍未支付的进件单关单。
// 关单=终态事件，若来源为自助邀请链接则同时起算链接 24h 有效期。返回关单数。
func (s *EnrollService) CloseTimeoutPending() (int64, error) {
	timeout := s.cfg.Int("enroll_pay_timeout", 30)
	if timeout <= 0 {
		timeout = 30
	}
	before := time.Now().Add(-time.Duration(timeout) * time.Minute)
	list, err := s.repo.ListPendingBefore(before)
	if err != nil {
		return 0, err
	}
	var n int64
	for i := range list {
		e := &list[i]
		if err := s.repo.UpdateEnroll(e.ID, map[string]any{"status": model.EnrollStatusClosed}); err != nil {
			continue
		}
		s.anchorInviteExpire(e.InviteCode) // 终态事件锚定链接 24h
		n++
	}
	return n, nil
}

// anchorInviteExpire 终态事件（关单/驳回/退款完成）后给来源邀请链接锚定 24h 有效期。
// 仅当链接仍启用且 expire_at 尚未设置时才写（避免重复延后）。code 为空或链接不存在则跳过。
func (s *EnrollService) anchorInviteExpire(code string) {
	if code == "" {
		return
	}
	v, err := s.repo.FindInviteByCode(code)
	if err != nil || v.Status != model.InviteStatusEnabled || v.ExpireAt != nil {
		return
	}
	hours := s.cfg.Int("enroll_link_expire", 24)
	if hours <= 0 {
		hours = 24
	}
	exp := time.Now().Add(time.Duration(hours) * time.Hour)
	_ = s.repo.UpdateInvite(v.ID, map[string]any{"expire_at": exp})
}

// —— 邀请链接 ——

// ListInvites 分页查询邀请链接。
func (s *EnrollService) ListInvites(agentID *uint, page, pageSize int) ([]model.EnrollInvite, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return s.repo.ListInvites(agentID, page, pageSize)
}

// CreateInvite 生成邀请链接。agentID=0 表示平台自己发。返回含唯一 code 的记录。
func (s *EnrollService) CreateInvite(agentID uint, name string) (*model.EnrollInvite, error) {
	code, err := genInviteCode()
	if err != nil {
		return nil, err
	}
	v := &model.EnrollInvite{
		Code: code, AgentID: agentID, Name: strings.TrimSpace(name),
		Status: model.InviteStatusEnabled,
	}
	if err := s.repo.CreateInvite(v); err != nil {
		return nil, err
	}
	return v, nil
}

// SetInviteStatus 停启用邀请链接。agentID 非空时校验归属。
func (s *EnrollService) SetInviteStatus(id uint, status int8, agentID *uint) error {
	v, err := s.repo.FindInvite(id)
	if err != nil {
		return enErr("邀请链接不存在")
	}
	if agentID != nil && v.AgentID != *agentID {
		return enErr("该邀请链接不属于您")
	}
	return s.repo.UpdateInvite(id, map[string]any{"status": status})
}

// DeleteInvite 删除邀请链接。agentID 非空时校验归属。
func (s *EnrollService) DeleteInvite(id uint, agentID *uint) error {
	v, err := s.repo.FindInvite(id)
	if err != nil {
		return enErr("邀请链接不存在")
	}
	if agentID != nil && v.AgentID != *agentID {
		return enErr("该邀请链接不属于您")
	}
	return s.repo.DeleteInvite(id)
}

// —— 结算流水 ——

// ListSettleLogs 分页查询进件结算流水。
func (s *EnrollService) ListSettleLogs(agentID *uint, page, pageSize int) ([]model.EnrollSettleLog, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return s.repo.ListSettleLogs(agentID, page, pageSize)
}

// ExpireDueInvites 定时任务入口：把 expire_at 已到的邀请链接置 expired。
func (s *EnrollService) ExpireDueInvites() (int64, error) {
	return s.repo.ExpireDueInvites(time.Now())
}

// —— 提交微信 / 查状态（本地状态机对齐 docs-代理进件/01 第五节）——

// SubmitToWx 把已支付待完善的进件单提交给微信（POST /v3/applyment4sub/applyment/）。
// 前置：status 必须为 paid（付费前置，先收钱才放行提交）；material_json 为完整进件请求体
// （敏感字段调用方在落库前已经过 SubMerchantService.EncryptSensitive 加密）。
// 成功后回填 wx_applyment_id、business_code，status→submitted，记 submit_time。
// agentID 非空时校验归属（代理端不能替别人提交）。
func (s *EnrollService) SubmitToWx(ctx context.Context, id uint, agentID *uint) (*model.SubMerchantEnroll, error) {
	if s.submch == nil {
		return nil, enErr("微信进件服务未就绪")
	}
	if !s.submch.Configured() {
		return nil, enErr("微信服务商凭证未配置，请先在系统设置填写")
	}
	e, err := s.repo.FindEnroll(id)
	if err != nil {
		return nil, enErr("进件单不存在")
	}
	if agentID != nil && e.AgentID != *agentID {
		return nil, enErr("该进件单不属于您")
	}
	if e.Status != model.EnrollStatusPaid && e.Status != model.EnrollStatusRejected {
		return nil, enErr("当前状态不可提交微信（需已支付待完善或被驳回后重提）")
	}
	if strings.TrimSpace(e.MaterialJSON) == "" {
		return nil, enErr("进件资料为空，请先填写完整资料")
	}

	// business_code：驳回重提复用原编号覆盖，首次提交按进件单号生成。
	businessCode := e.BusinessCode
	if businessCode == "" {
		businessCode = "EN" + e.EnrollNo
	}
	// 把 business_code 注入请求体（资料 JSON 里不含时补上）。
	body, err := injectBusinessCode(e.MaterialJSON, businessCode)
	if err != nil {
		return nil, enErr("进件资料格式错误: " + err.Error())
	}

	r, _, err := s.submch.SubmitApplyment(ctx, body)
	if err != nil {
		return nil, enErr(err.Error())
	}

	now := time.Now()
	fields := map[string]any{
		"business_code": businessCode,
		"status":        model.EnrollStatusSubmitted,
		"wx_state":      "APPLYMENT_STATE_AUDITING",
		"reject_reason": "",
		"submit_time":   &now,
	}
	if r.ApplymentID > 0 {
		fields["wx_applyment_id"] = decimalToApplyment(r.ApplymentID)
	}
	if err := s.repo.UpdateEnroll(id, fields); err != nil {
		return nil, err
	}
	return s.repo.FindEnroll(id)
}

// SyncWxState 主动拉取微信申请单最新状态并落库，驱动本地状态机推进。
// finished（拿到 sub_mchid）→ 触发结算/扣名额；rejected → 记驳回原因（退款由退款流程处理）。
// agentID 非空时校验归属。返回同步后的进件单。
func (s *EnrollService) SyncWxState(ctx context.Context, id uint, agentID *uint) (*model.SubMerchantEnroll, error) {
	if s.submch == nil {
		return nil, enErr("微信进件服务未就绪")
	}
	e, err := s.repo.FindEnroll(id)
	if err != nil {
		return nil, enErr("进件单不存在")
	}
	if agentID != nil && e.AgentID != *agentID {
		return nil, enErr("该进件单不属于您")
	}
	if e.Status != model.EnrollStatusSubmitted {
		return nil, enErr("仅审核中的进件单可查询微信状态")
	}
	if e.BusinessCode == "" {
		return nil, enErr("该进件单缺少业务申请编号，无法查询")
	}

	st, _, err := s.submch.QueryApplymentByBusinessCode(ctx, e.BusinessCode)
	if err != nil {
		return nil, enErr(err.Error())
	}
	return s.applyWxState(e, st.ApplymentState, st.ApplymentStateMsg, st.SubMchID)
}

// applyWxState 把微信申请单状态映射到本地状态机并落库；命中终态触发结算。
func (s *EnrollService) applyWxState(e *model.SubMerchantEnroll, wxState, wxMsg, subMchID string) (*model.SubMerchantEnroll, error) {
	fields := map[string]any{"wx_state": wxState}

	switch wxState {
	case "APPLYMENT_STATE_FINISHED":
		if subMchID == "" {
			// 状态说完成但未返回 sub_mchid，视为尚未真正开通，保持 submitted 等下次查。
			if err := s.repo.UpdateEnroll(e.ID, fields); err != nil {
				return nil, err
			}
			return s.repo.FindEnroll(e.ID)
		}
		now := time.Now()
		fields["wx_sub_mchid"] = subMchID
		fields["status"] = model.EnrollStatusFinished
		fields["finish_time"] = &now
		if err := s.repo.UpdateEnroll(e.ID, fields); err != nil {
			return nil, err
		}
		// 幂等：仅当此前未完成时才结算（避免重复查询重复结算）。
		if e.Status != model.EnrollStatusFinished {
			e.WxSubMchID = subMchID
			if err := s.settleOnFinish(e); err != nil {
				return nil, err
			}
		}
	case "APPLYMENT_STATE_REJECTED":
		fields["status"] = model.EnrollStatusRejected
		fields["reject_reason"] = wxMsg
		if err := s.repo.UpdateEnroll(e.ID, fields); err != nil {
			return nil, err
		}
	default:
		// 审核中 / 待账户验证 / 待签约 / 开通权限中：保持 submitted，仅刷新 wx_state 原值。
		if err := s.repo.UpdateEnroll(e.ID, fields); err != nil {
			return nil, err
		}
	}
	return s.repo.FindEnroll(e.ID)
}

// settleOnFinish 进件成功（拿到 sub_mchid）后的资金结算，写 pay_enroll_settle_log。
//   路径一（预购名额）：扣代理 1 名额（consume），客户付的全额过账给代理（agent_amount=全额、platform_amount=0）。
//   路径二（商户自付）：按后台可配比例分账（平台一份 / 代理一份）。
// 幂等由调用方（applyWxState 的状态跃迁判断）保证。
func (s *EnrollService) settleOnFinish(e *model.SubMerchantEnroll) error {
	log := &model.EnrollSettleLog{
		EnrollID:   e.ID,
		AgentID:    e.AgentID,
		Path:       e.Path,
		PayOrderNo: e.PayOrderNo,
		SettleTime: time.Now(),
	}

	switch e.Path {
	case model.EnrollPathQuota:
		// 扣代理 1 名额（余额不足也不阻断进件已成的事实，仅记流水；名额兜底见待优化）。
		if s.agentRepo != nil && e.AgentID > 0 {
			_ = s.agentRepo.ChangeQuota(e.AgentID, "consume", -1, decimal.Zero, e.EnrollNo, "进件成功扣名额")
		}
		log.AgentAmount = e.RetailAmount // 客户付的全额分给代理
		log.PlatformAmount = decimal.Zero
	default: // EnrollPathSelf
		platformShare := s.cfg.Dec("enroll_platform_share", decimal.Zero)
		agentShare := s.cfg.Dec("enroll_agent_share", decimal.Zero)
		log.PlatformAmount = platformShare
		log.AgentAmount = agentShare
	}
	return s.repo.CreateSettleLog(log)
}

// injectBusinessCode 确保进件请求体 JSON 含 business_code 字段（缺则补，已有则以我方编号为准覆盖）。
func injectBusinessCode(materialJSON, businessCode string) (string, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(materialJSON), &m); err != nil {
		return "", err
	}
	m["business_code"] = businessCode
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// decimalToApplyment 微信 applyment_id 是数字，落库字段是字符串，转一下。
func decimalToApplyment(id int64) string {
	return decimal.NewFromInt(id).String()
}

// genInviteCode 生成 16 位十六进制唯一短码（URL 里用）。
func genInviteCode() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// genEnrollNo 生成进件单号：EN + yyyyMMddHHmmss + 6 位随机十六进制（业务唯一键）。
func genEnrollNo() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return fmt.Sprintf("EN%s%s", time.Now().Format("20060102150405"), hex.EncodeToString(b))
}
