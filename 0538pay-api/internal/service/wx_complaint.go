package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
	"github.com/epvia/api/pkg/wxpayv3"
)

// WxComplaintService 消费者投诉2.0 业务编排（自研扩展，挂服务商进件线）。
//
// 后端只有一套：微信回调只打服务商这一个回调地址 → 平台先收下 → 按 complainted_mchid 反查本地商户 → 落库分发。
// admin 全量消费（一期），/m 商户端按 sub_mchid 过滤复用（二期）。
//
// 安全（官方硬要求 + 越权拦截）：
//   · 回调验签 WECHATPAY2-SHA256-RSA2048（平台公钥），对 SIGNTEST 探测流量天然拒绝；AEAD_AES_256_GCM 解密。
//   · 幂等：NotifyID 唯一索引去重，重复回调应答成功。
//   · 越权：处理类接口的 complainted_mchid 必须是本服务商名下、已进件成功的子商户，否则拒绝（270701078 语义）。
//   · 敏感：payer_phone 微信用平台证书加密下发，落库存密文 + 脱敏，不回原文。
//   · 回调不能作唯一数据源：收到 {complaint_id} 后一律以「查询详情」补全；另有 scheduler 轮询兜底。
type WxComplaintService struct {
	repo    *repository.WxComplaintRepo
	notifs  *repository.WxComplaintNotifyRepo
	enrolls *repository.ChannelEnrollRepo
	submch  *SubMerchantService // 复用服务商凭证（验签公钥 + APIv3 解密 + 私钥解敏感字段）
	client  *WxComplaintClient  // 14 接口 REST 子客户端
	cfg     *ConfigService      // 回调地址自管理状态存 wx_complaint 分组
}

func NewWxComplaintService(
	repo *repository.WxComplaintRepo,
	notifs *repository.WxComplaintNotifyRepo,
	enrolls *repository.ChannelEnrollRepo,
	submch *SubMerchantService,
	cfg *ConfigService,
) *WxComplaintService {
	return &WxComplaintService{
		repo:    repo,
		notifs:  notifs,
		enrolls: enrolls,
		submch:  submch,
		client:  NewWxComplaintClient(submch),
		cfg:     cfg,
	}
}

// WxComplaintError 携带业务提示（handler 据此返回错误码 / 回调据此决定应答成败）。
type WxComplaintError struct{ Msg string }

func (e *WxComplaintError) Error() string { return e.Msg }

func wcErr(msg string) *WxComplaintError { return &WxComplaintError{Msg: msg} }

// complaintNotifyEnvelope 投诉回调外层信封（顶层 id 作幂等键）。
type complaintNotifyEnvelope struct {
	ID           string `json:"id"`
	CreateTime   string `json:"create_time"`
	EventType    string `json:"event_type"`
	ResourceType string `json:"resource_type"`
	Summary      string `json:"summary"`
	Resource     struct {
		Algorithm      string `json:"algorithm"`
		Ciphertext     string `json:"ciphertext"`
		OriginalType   string `json:"original_type"`
		AssociatedData string `json:"associated_data"`
		Nonce          string `json:"nonce"`
	} `json:"resource"`
}

// complaintNotifyResource 解密后业务对象（投诉回调只给 complaint_id + action_type）。
type complaintNotifyResource struct {
	ComplaintID string `json:"complaint_id"`
	ActionType  string `json:"action_type"`
	raw         string `json:"-"` // 解密后原始明文（落流水 RawJSON 用，非微信字段）
}

// signTest 探测流量前缀（4013059017），复用 channel_control_notify 的 signTestPrefix 常量。

// HandleNotify 处理投诉回调：验签 → 解密 → 幂等落流水 → 查详情补全 Upsert 主表。
// 成功/重复回调返回 nil（应答 200）；验签/解密/落库失败返回错误（应答非 2xx 触发微信重推）。
func (s *WxComplaintService) HandleNotify(ctx context.Context, h NotifyHeaders, body []byte) error {
	env, res, err := s.verifyAndDecrypt(h, body)
	if err != nil {
		return err
	}
	// 幂等落回调流水（NotifyID 唯一索引；重复通知直接应答成功）。
	notify := &model.WxComplaintNotify{
		NotifyID:    env.ID,
		ComplaintID: res.ComplaintID,
		EventType:   env.EventType,
		ActionType:  res.ActionType,
		Summary:     env.Summary,
		RawJSON:     res.raw,
	}
	if err := s.notifs.Create(notify); err != nil {
		if errors.Is(err, repository.ErrComplaintNotifyDuplicate) {
			return nil // 幂等命中：已处理过，应答成功
		}
		return wcErr("投诉回调流水落库失败: " + err.Error())
	}
	// ★ 回调不能作唯一数据源：拉「查询详情」补全落库。查详情失败不影响应答成功（流水已落，靠轮询兜底）。
	if _, err := s.RefreshOne(ctx, res.ComplaintID, res.ActionType, env.EventType); err != nil {
		// 记录但不使回调失败——微信重推无益（同样查不到），交给 scheduler 轮询兜底对账。
		return nil
	}
	return nil
}

// verifyAndDecrypt 校验回调签名并解密业务对象（不落库、不刷新，便于单测端到端覆盖密码学路径）。
// 顺序：平台公钥/APIv3 密钥齐备 → SIGNTEST 探测拒绝 → WECHATPAY2-SHA256-RSA2048 验签 → 解析信封
// → AEAD_AES_256_GCM 解密 → 解析 complaint_id/action_type。任一失败返回 *WxComplaintError。
func (s *WxComplaintService) verifyAndDecrypt(h NotifyHeaders, body []byte) (*complaintNotifyEnvelope, *complaintNotifyResource, error) {
	c := s.submch.creds()
	if c.PublicKey == "" {
		return nil, nil, wcErr("微信服务商平台公钥未配置，无法验证回调签名，拒绝处理")
	}
	if len(c.APIv3Key) != 32 {
		return nil, nil, wcErr("微信服务商 APIv3 密钥未配置或长度不正确，无法解密回调")
	}
	if strings.HasPrefix(h.Signature, signTestPrefix) {
		return nil, nil, wcErr("签名探测流量（SIGNTEST），已按验签失败拒绝")
	}
	pub, err := wxpayv3.ParsePublicKey(c.PublicKey)
	if err != nil {
		return nil, nil, wcErr("解析平台公钥失败: " + err.Error())
	}
	if err := wxpayv3.VerifySignature(pub, h.Timestamp, h.Nonce, string(body), h.Signature); err != nil {
		return nil, nil, wcErr("回调验签不通过: " + err.Error())
	}
	var env complaintNotifyEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, nil, wcErr("回调报文解析失败: " + err.Error())
	}
	if strings.TrimSpace(env.ID) == "" {
		return nil, nil, wcErr("回调缺少通知 id，无法幂等去重")
	}
	if env.Resource.Ciphertext == "" {
		return nil, nil, wcErr("回调缺少密文")
	}
	plain, err := wxpayv3.DecryptAESGCM(c.APIv3Key, env.Resource.Nonce, env.Resource.AssociatedData, env.Resource.Ciphertext)
	if err != nil {
		return nil, nil, wcErr("回调报文解密失败: " + err.Error())
	}
	var res complaintNotifyResource
	if err := json.Unmarshal(plain, &res); err != nil {
		return nil, nil, wcErr("投诉回调业务对象解析失败: " + err.Error())
	}
	if strings.TrimSpace(res.ComplaintID) == "" {
		return nil, nil, wcErr("回调解密后缺少 complaint_id")
	}
	res.raw = string(plain)
	return &env, &res, nil
}

// RefreshOne 按 complaint_id 查微信详情并 Upsert 主表（回调 + admin 手动刷新 + 轮询兜底共用）。
// actionType/eventType 为回调触发时的最近动作（无回调场景传空）。
func (s *WxComplaintService) RefreshOne(ctx context.Context, complaintID, actionType, eventType string) (*model.WxComplaint, error) {
	detail, raw, err := s.client.GetComplaint(ctx, complaintID)
	if err != nil {
		return nil, err
	}
	c := s.fromDetail(detail, string(raw))
	if actionType != "" {
		c.LastActionType = actionType
	}
	if eventType != "" {
		c.LastEventType = eventType
	}
	if err := s.repo.Upsert(c); err != nil {
		return nil, wcErr("投诉单落库失败: " + err.Error())
	}
	return c, nil
}

// fromDetail 把微信投诉详情映射为本地主表模型（反查归属 + 敏感字段加密落库/脱敏 + JSON 序列化）。
func (s *WxComplaintService) fromDetail(d *ComplaintDetail, raw string) *model.WxComplaint {
	c := &model.WxComplaint{
		ComplaintID:           d.ComplaintID,
		ComplaintedMchID:      strings.TrimSpace(d.ComplaintedMchID),
		ComplaintState:        d.ComplaintState,
		ComplaintTime:         d.ComplaintTime,
		ComplaintDetail:       d.ComplaintDetail,
		ProblemType:           d.ProblemType,
		ProblemDesc:           d.ProblemDescription,
		ApplyRefundAmount:     d.ApplyRefundAmount,
		ComplaintFullRefunded: d.ComplaintFullRefunded,
		PayerOpenID:           d.PayerOpenID,
		UserComplaintTimes:    d.UserComplaintTimes,
		IncomingUserResponse:  d.IncomingUserResponse,
		InPlatformService:     d.InPlatformService,
		NeedImmediateService:  d.NeedImmediateService,
		RawJSON:               raw,
	}
	if len(d.ComplaintOrderInfo) > 0 {
		if bs, err := json.Marshal(d.ComplaintOrderInfo); err == nil {
			c.ComplaintOrderInfo = string(bs)
		}
	}
	if len(d.ComplaintMediaList) > 0 {
		if bs, err := json.Marshal(d.ComplaintMediaList); err == nil {
			c.ComplaintMediaList = string(bs)
		}
	}
	if len(d.UserTagList) > 0 {
		if bs, err := json.Marshal(d.UserTagList); err == nil {
			c.UserTagList = string(bs)
		}
	}
	// 敏感字段：payer_phone 微信用平台证书加密下发，存密文 + 脱敏明文（能解则解，解不了留密文不阻塞）。
	if d.PayerPhone != "" {
		c.PayerPhoneEnc = d.PayerPhone
		if plain, err := wxpayv3.DecryptOAEP(d.PayerPhone, s.submch.creds().PrivateKey); err == nil {
			c.PayerPhoneMask = maskPhone(plain)
		}
	}
	// 归属反查：按 complainted_mchid 定位本服务商名下已开通进件单，回填本地商户/进件单/商户名。
	if c.ComplaintedMchID != "" {
		if e, err := s.enrolls.FindApprovedBySubMchID(c.ComplaintedMchID); err == nil && e != nil {
			c.MerchantID = e.UID
			c.EnrollID = e.ID
			c.MerchantName = e.MerchantName
		}
	}
	return c
}

// maskPhone 手机号脱敏：保留前 3 后 4，中间 ****；非常规长度整体打码。
func maskPhone(p string) string {
	p = strings.TrimSpace(p)
	if len(p) >= 7 {
		return p[:3] + "****" + p[len(p)-4:]
	}
	if p == "" {
		return ""
	}
	return "****"
}

// —— 查询（后台读，list/detail/history）——

// ComplaintListItem 列表行 DTO（含状态中文名，order 摘要）。
type ComplaintListItem struct {
	model.WxComplaint
	StateText string `json:"state_text"`
}

// ComplaintListResult 列表结果。
type ComplaintListResult struct {
	List  []ComplaintListItem `json:"list"`
	Total int64               `json:"total"`
	Stats map[string]int64    `json:"stats"` // 各状态计数（PENDING/PROCESSING/PROCESSED）
}

// List 后台投诉单列表（admin 全量；商户端二期传 complaintedMchID 隔离复用）。
func (s *WxComplaintService) List(q repository.WxComplaintQuery) (*ComplaintListResult, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
	rows, total, err := s.repo.List(q)
	if err != nil {
		return nil, wcErr("查询投诉单列表失败: " + err.Error())
	}
	items := make([]ComplaintListItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, ComplaintListItem{WxComplaint: r, StateText: model.WxComplaintStateText(r.ComplaintState)})
	}
	stats, _ := s.repo.Stats(q.ComplaintedMchID, q.MerchantID)
	return &ComplaintListResult{List: items, Total: total, Stats: stats}, nil
}

// ComplaintDetailResult 详情 DTO（主表 + 回调时间线）。
type ComplaintDetailResult struct {
	Complaint model.WxComplaint          `json:"complaint"`
	StateText string                     `json:"state_text"`
	Notifies  []model.WxComplaintNotify  `json:"notifies"` // 回调动作时间线（新在上）
	Orders    []ComplaintOrder           `json:"orders"`   // 关联订单（从 order_info JSON 解析）
}

// Detail 后台投诉单详情（读本地主表 + 回调流水时间线；不现查微信，如需最新用 RefreshOne）。
func (s *WxComplaintService) Detail(id uint) (*ComplaintDetailResult, error) {
	c, err := s.repo.FindByID(id)
	if err != nil {
		return nil, wcErr("投诉单不存在")
	}
	notifies, _ := s.notifs.ListByComplaintID(c.ComplaintID)
	var orders []ComplaintOrder
	if c.ComplaintOrderInfo != "" {
		_ = json.Unmarshal([]byte(c.ComplaintOrderInfo), &orders)
	}
	return &ComplaintDetailResult{
		Complaint: *c,
		StateText: model.WxComplaintStateText(c.ComplaintState),
		Notifies:  notifies,
		Orders:    orders,
	}, nil
}

// SyncDetail 后台手动刷新单个投诉单（现查微信详情覆盖本地快照）。
func (s *WxComplaintService) SyncDetail(ctx context.Context, id uint) (*ComplaintDetailResult, error) {
	c, err := s.repo.FindByID(id)
	if err != nil {
		return nil, wcErr("投诉单不存在")
	}
	if _, err := s.RefreshOne(ctx, c.ComplaintID, "", ""); err != nil {
		return nil, err
	}
	return s.Detail(id)
}

// History 查询投诉单协商历史（现查微信，不落库）。
func (s *WxComplaintService) History(ctx context.Context, id, limit, offset int) (*NegotiationHistoryResp, error) {
	c, err := s.repo.FindByID(uint(id))
	if err != nil {
		return nil, wcErr("投诉单不存在")
	}
	r, _, err := s.client.GetNegotiationHistory(ctx, c.ComplaintID, limit, offset)
	if err != nil {
		return nil, wcErr(err.Error())
	}
	return r, nil
}

// —— 处理（后台代处理，均带越权拦截）——

// mustOwn 取本地投诉单并校验其被诉子商户在本服务商名下已进件成功（越权拦截，270701078 语义）。
func (s *WxComplaintService) mustOwn(id uint) (*model.WxComplaint, error) {
	c, err := s.repo.FindByID(id)
	if err != nil {
		return nil, wcErr("投诉单不存在")
	}
	if strings.TrimSpace(c.ComplaintedMchID) == "" {
		return nil, wcErr("投诉单缺少被诉子商户号，无法处理")
	}
	e, err := s.enrolls.FindApprovedBySubMchID(c.ComplaintedMchID)
	if err != nil {
		return nil, wcErr("校验子商户归属失败: " + err.Error())
	}
	if e == nil {
		return nil, wcErr("该投诉的被诉商户不在本平台名下（或未进件成功），无权处理")
	}
	return c, nil
}

// Reply 回复用户（≤200字符）。
func (s *WxComplaintService) Reply(ctx context.Context, id uint, content string, images []string, jumpURL, jumpText string) error {
	c, err := s.mustOwn(id)
	if err != nil {
		return err
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return wcErr("回复内容不能为空")
	}
	if len([]rune(content)) > 200 {
		return wcErr("回复内容不能超过 200 字符")
	}
	if len(images) > 4 {
		return wcErr("回复图片最多 4 张")
	}
	req := ResponseReq{
		ComplaintedMchID: c.ComplaintedMchID,
		ResponseContent:  content,
		ResponseImages:   images,
		JumpURL:          strings.TrimSpace(jumpURL),
		JumpURLText:      strings.TrimSpace(jumpText),
	}
	if _, err := s.client.Response(ctx, c.ComplaintID, req); err != nil {
		return wcErr(err.Error())
	}
	// 回复后刷新一次详情（状态可能变 PROCESSING），失败不阻塞。
	_, _ = s.RefreshOne(ctx, c.ComplaintID, model.WxComplaintActionMerchantResponse, "")
	return nil
}

// Complete 反馈处理完成（PROCESSED 后禁 complete，本地已终态则幂等当成功）。
func (s *WxComplaintService) Complete(ctx context.Context, id uint) error {
	c, err := s.mustOwn(id)
	if err != nil {
		return err
	}
	if c.ComplaintState == model.WxComplaintStateProcessed {
		return nil // 已终态：幂等当成功，不再调微信（对齐 268440133 规避）
	}
	if _, err := s.client.Complete(ctx, c.ComplaintID, c.ComplaintedMchID); err != nil {
		return wcErr(err.Error())
	}
	_, _ = s.RefreshOne(ctx, c.ComplaintID, model.WxComplaintActionMerchantConfirm, "")
	return nil
}

// UpdateRefund 更新退款审批结果（APPROVE / REJECT）。
func (s *WxComplaintService) UpdateRefund(ctx context.Context, id uint, req UpdateRefundReq) error {
	c, err := s.mustOwn(id)
	if err != nil {
		return err
	}
	action := strings.ToUpper(strings.TrimSpace(req.Action))
	if action != "APPROVE" && action != "REJECT" {
		return wcErr("审批动作须为 APPROVE 或 REJECT")
	}
	if action == "REJECT" && strings.TrimSpace(req.RejectReason) == "" {
		return wcErr("驳回退款须填写驳回原因")
	}
	req.ComplaintedMchID = c.ComplaintedMchID
	req.Action = action
	if _, err := s.client.UpdateRefundProgress(ctx, c.ComplaintID, req); err != nil {
		return wcErr(err.Error())
	}
	act := model.WxComplaintActionMerchantApproveRefund
	if action == "REJECT" {
		act = model.WxComplaintActionMerchantRejectRefund
	}
	_, _ = s.RefreshOne(ctx, c.ComplaintID, act, "")
	return nil
}

// ReplyImmediate 回复需即时服务的投诉单。
func (s *WxComplaintService) ReplyImmediate(ctx context.Context, id uint, content string, images []string) error {
	c, err := s.mustOwn(id)
	if err != nil {
		return err
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return wcErr("回复内容不能为空")
	}
	if len(images) > 4 {
		return wcErr("回复图片最多 4 张")
	}
	req := ImmediateResponseReq{ComplaintedMchID: c.ComplaintedMchID, ResponseContent: content, ResponseImages: images}
	if _, err := s.client.ResponseImmediate(ctx, c.ComplaintID, req); err != nil {
		return wcErr(err.Error())
	}
	_, _ = s.RefreshOne(ctx, c.ComplaintID, model.WxComplaintActionMerchantResponse, "")
	return nil
}

// UploadImage 上传商户反馈图片，返回 media_id（供回复引用）。
func (s *WxComplaintService) UploadImage(ctx context.Context, filename string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", wcErr("图片内容为空")
	}
	if len(data) > 2*1024*1024 {
		return "", wcErr("图片不能超过 2M")
	}
	id, err := s.client.UploadImage(ctx, filename, data)
	if err != nil {
		return "", wcErr(err.Error())
	}
	return id, nil
}

// —— 回调地址自管理（组1）——

// NotifyURLState 回调地址当前状态（后台设置页读）。
type NotifyURLState struct {
	Registered string `json:"registered"` // 微信侧已注册的回调地址（现查微信）
	Local      string `json:"local"`      // 本地记录的期望回调地址（config wx_complaint.notify_url）
}

// GetNotifyURL 查询微信侧已注册的投诉回调地址 + 本地期望值。
func (s *WxComplaintService) GetNotifyURL(ctx context.Context) (*NotifyURLState, error) {
	st := &NotifyURLState{Local: strings.TrimSpace(s.cfg.Str("wx_complaint_notify_url"))}
	r, _, err := s.client.GetNotifyURL(ctx)
	if err == nil && r != nil {
		st.Registered = r.URL
	}
	return st, nil
}

// SetNotifyURL 设置投诉回调地址（幂等）：先查微信，未注册则创建，已注册则更新；本地存期望值。
func (s *WxComplaintService) SetNotifyURL(ctx context.Context, notifyURL string) (*NotifyURLState, error) {
	notifyURL = strings.TrimSpace(notifyURL)
	if !strings.HasPrefix(notifyURL, "https://") {
		return nil, wcErr("回调地址必须是 https:// 开头的公网地址")
	}
	// 一个商户号只能建一个回调地址：先查，有则更新，无则创建（错误码 268435484 数据已存在 → 转更新兜底）。
	existing, _, qerr := s.client.GetNotifyURL(ctx)
	var opErr error
	if qerr == nil && existing != nil && existing.URL != "" {
		_, _, opErr = s.client.UpdateNotifyURL(ctx, notifyURL)
	} else {
		_, _, opErr = s.client.CreateNotifyURL(ctx, notifyURL)
		if opErr != nil {
			// 创建撞「数据已存在」时改走更新兜底。
			if strings.Contains(opErr.Error(), "268435484") || strings.Contains(opErr.Error(), "数据已存在") {
				_, _, opErr = s.client.UpdateNotifyURL(ctx, notifyURL)
			}
		}
	}
	if opErr != nil {
		return nil, wcErr(opErr.Error())
	}
	if err := s.cfg.SaveGroup("wx_complaint", map[string]string{"wx_complaint_notify_url": notifyURL}); err != nil {
		return nil, wcErr("保存本地回调地址失败: " + err.Error())
	}
	return s.GetNotifyURL(ctx)
}

// DeleteNotifyURL 删除微信侧投诉回调地址。
func (s *WxComplaintService) DeleteNotifyURL(ctx context.Context) error {
	if _, err := s.client.DeleteNotifyURL(ctx); err != nil {
		return wcErr(err.Error())
	}
	_ = s.cfg.SaveGroup("wx_complaint", map[string]string{"wx_complaint_notify_url": ""})
	return nil
}

// —— 轮询兜底对账（scheduler）——

// Reconcile 轮询兜底：查最近 beginDate~endDate 的投诉单列表并逐条 Upsert（补回调遗漏）。
// ★ 官方铁律：回调不能作唯一数据源。日期跨度≤30天，limit≤50，超频 270924289 由上层限速。
// 返回处理条数。
func (s *WxComplaintService) Reconcile(ctx context.Context, beginDate, endDate string) (int, error) {
	if !s.submch.Configured() {
		return 0, nil // 服务商凭证未配齐：静默跳过，不空跑微信网关
	}
	n := 0
	offset := 0
	for {
		resp, _, err := s.client.ListComplaints(ctx, ListComplaintsParams{
			Limit: 50, Offset: offset, BeginDate: beginDate, EndDate: endDate,
		})
		if err != nil {
			return n, err
		}
		for i := range resp.Data {
			c := s.fromDetail(&resp.Data[i], "")
			if err := s.repo.Upsert(c); err == nil {
				n++
			}
		}
		offset += len(resp.Data)
		if len(resp.Data) == 0 || offset >= resp.TotalCount {
			break
		}
	}
	return n, nil
}

// —— 商户端 /m（在 admin 越权拦截之上再叠一层「按登录商户 uid 归属」隔离）——
//
// admin 端 mustOwn 只校验「被诉子商户在本平台名下」；商户端还必须校验「该投诉的
// merchant_id == 当前登录商户 uid」，商户只能看/处理自己名下的投诉。列表强制按 uid
// 过滤，处理动作前置 AssertMerchantOwn，双保险防越权。

// AssertMerchantOwn 商户端归属校验：投诉的 merchant_id 必须等于当前登录商户 uid。
// /m 每个详情/处理动作前置调用；校验通过返回投诉单，失败返回越权错误。
func (s *WxComplaintService) AssertMerchantOwn(id, merchantID uint) (*model.WxComplaint, error) {
	c, err := s.repo.FindByID(id)
	if err != nil {
		return nil, wcErr("投诉单不存在")
	}
	if merchantID == 0 || c.MerchantID != merchantID {
		return nil, wcErr("无权处理该投诉（不在你名下）")
	}
	return c, nil
}

// ListForMerchant 商户端列表：强制按登录商户 uid 隔离（覆盖入参里的商户维度，防越权）。
func (s *WxComplaintService) ListForMerchant(merchantID uint, q repository.WxComplaintQuery) (*ComplaintListResult, error) {
	q.MerchantID = merchantID // 强制锁定当前商户，忽略外部传入
	q.ComplaintedMchID = ""   // 商户端不开放按任意子商户号筛选
	return s.List(q)
}

// DetailForMerchant 商户端详情（先归属校验再取本地快照 + 回调时间线）。
func (s *WxComplaintService) DetailForMerchant(id, merchantID uint) (*ComplaintDetailResult, error) {
	if _, err := s.AssertMerchantOwn(id, merchantID); err != nil {
		return nil, err
	}
	return s.Detail(id)
}
