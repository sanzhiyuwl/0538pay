package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
	"github.com/epvia/api/pkg/wxpayv3"
)

// ChannelControlNotifyService 子商户管控/处置订阅回调（风控第三段）。
//
// 承接微信主动推送的风控状态变化，落 pay_channel_control_flow 流水，并异步用第二段查询接口
// 核实明细刷新快照（★官方明确「不能仅依赖回调，需结合查询接口」——回调 + 查询 + 兜底轮询三者并用）。
//
// 两套独立机制（配置入口/event_type/报文结构都不同，分别建模、分别接）：
//   (A) violation       —— 商户平台处置通知（service_notify_url，4012079216）
//                          event_type VIOLATION.PUNISH/.INTERCEPT/.APPEAL
//   (B) merchant_notify —— 合作伙伴订阅·商户新增管控流水通知（topic 20000，4016022266）
//                          event_type MERCHANT_NOTIFY.NOTIFY
//
// 安全（官方硬要求）：
//   · 验签 WECHATPAY2-SHA256-RSA2048（平台证书/微信支付公钥）；对 "WECHATPAY/SIGNTEST/" 签名探测
//     流量一律走验签、失败即拒绝（天然合规，不误当成正常回调处理）。
//   · 解密 AEAD_AES_256_GCM（APIv3 密钥）。
//   · 幂等：NotifyID（回调唯一 id）唯一索引去重，重复回调直接应答成功（微信按退避重试最长 48h）。
type ChannelControlNotifyService struct {
	flows   *repository.ChannelControlFlowRepo
	enrolls *repository.ChannelEnrollRepo
	submch  *SubMerchantService    // 复用服务商凭证（平台公钥验签 + APIv3 密钥解密）
	control *ChannelControlService // 回调后异步触发第二段查询核实并刷新快照（可空）
	notice  *NoticeService         // 对外通知中枢（可空；SetNoticeService 注入。处置流水商户通知，scene=mchrisk）
}

func NewChannelControlNotifyService(
	flows *repository.ChannelControlFlowRepo,
	enrolls *repository.ChannelEnrollRepo,
	submch *SubMerchantService,
	control *ChannelControlService,
) *ChannelControlNotifyService {
	return &ChannelControlNotifyService{flows: flows, enrolls: enrolls, submch: submch, control: control}
}

// SetNoticeService 注入通知中枢（对齐 epay MsgNotice::send('mchrisk', uid, param) 语义）。
func (s *ChannelControlNotifyService) SetNoticeService(n *NoticeService) { s.notice = n }

// ChannelNotifyError 携带业务提示（handler 据此返回失败应答，触发微信重推）。
type ChannelNotifyError struct{ Msg string }

func (e *ChannelNotifyError) Error() string { return e.Msg }

func cnErr(msg string) *ChannelNotifyError { return &ChannelNotifyError{Msg: msg} }

// NotifyHeaders 回调 HTTP 头中的验签四要素（Wechatpay-*）。
type NotifyHeaders struct {
	Signature string // Wechatpay-Signature
	Timestamp string // Wechatpay-Timestamp
	Nonce     string // Wechatpay-Nonce
	Serial    string // Wechatpay-Serial（平台证书序列号/公钥ID）
}

// notifyEnvelope 回调外层信封（A/B 通用；顶层 id 用作幂等键）。
type notifyControlEnvelope struct {
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

// violationResource (A) 解密后字段（4012079216）。
type violationResource struct {
	SubMchID          string `json:"sub_mchid"`
	CompanyName       string `json:"company_name"`
	RecordID          string `json:"record_id"`
	PunishPlan        string `json:"punish_plan"`
	PunishTime        string `json:"punish_time"`
	PunishDescription string `json:"punish_description"`
	RiskType          string `json:"risk_type"`
	RiskDescription   string `json:"risk_description"`
}

// merchantNotifyResource (B) 解密后字段（4016022266）。
type merchantNotifyResource struct {
	MessageContent struct {
		MerchantCode        string `json:"merchant_code"`
		MerchantCompanyName string `json:"merchant_company_name"`
		BusinessTime        string `json:"business_time"`
		BusinessCode        string `json:"business_code"`
		BusinessState       string `json:"business_state"`
	} `json:"message_content"`
}

// signTestPrefix 微信签名探测流量前缀，见 4013059017「如何应对签名探测流量」。
const signTestPrefix = "WECHATPAY/SIGNTEST/"

// verifyAndDecrypt 验签 + 解密，返回信封与解密后明文。任一环节失败即返回错误（拒绝回调）。
func (s *ChannelControlNotifyService) verifyAndDecrypt(h NotifyHeaders, body []byte) (*notifyControlEnvelope, []byte, error) {
	c := s.submch.creds()
	if c.PublicKey == "" {
		return nil, nil, cnErr("微信服务商平台公钥未配置，无法验证回调签名，拒绝处理")
	}
	if len(c.APIv3Key) != 32 {
		return nil, nil, cnErr("微信服务商 APIv3 密钥未配置或长度不正确，无法解密回调")
	}
	// 签名探测流量：以 WECHATPAY/SIGNTEST/ 开头的错误签名必然验签失败，走验签即天然拒绝；
	// 显式识别只为日志清晰、不误进业务处理（应答非 2xx 即"正确应对探测"）。
	if strings.HasPrefix(h.Signature, signTestPrefix) {
		return nil, nil, cnErr("签名探测流量（SIGNTEST），已按验签失败拒绝")
	}
	pub, err := wxpayv3.ParsePublicKey(c.PublicKey)
	if err != nil {
		return nil, nil, cnErr("解析平台公钥失败: " + err.Error())
	}
	if err := wxpayv3.VerifySignature(pub, h.Timestamp, h.Nonce, string(body), h.Signature); err != nil {
		return nil, nil, cnErr("回调验签不通过: " + err.Error())
	}
	var env notifyControlEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, nil, cnErr("回调报文解析失败: " + err.Error())
	}
	if strings.TrimSpace(env.ID) == "" {
		return nil, nil, cnErr("回调缺少通知 id，无法幂等去重")
	}
	if env.Resource.Ciphertext == "" {
		return nil, nil, cnErr("回调缺少密文")
	}
	plain, err := wxpayv3.DecryptAESGCM(c.APIv3Key, env.Resource.Nonce, env.Resource.AssociatedData, env.Resource.Ciphertext)
	if err != nil {
		return nil, nil, cnErr("回调报文解密失败: " + err.Error())
	}
	return &env, plain, nil
}

// HandleViolation 处理 (A) 商户平台处置通知（service_notify_url）。
// 成功/重复回调返回 nil（应答 200）；验签/解密/落库失败返回错误（应答非 2xx 触发重推）。
func (s *ChannelControlNotifyService) HandleViolation(ctx context.Context, h NotifyHeaders, body []byte) error {
	env, plain, err := s.verifyAndDecrypt(h, body)
	if err != nil {
		return err
	}
	var res violationResource
	if err := json.Unmarshal(plain, &res); err != nil {
		return cnErr("处置通知业务对象解析失败: " + err.Error())
	}
	flow := &model.ChannelControlFlow{
		NotifyID:          env.ID,
		Mechanism:         model.ChannelControlMechViolation,
		EventType:         env.EventType,
		Summary:           env.Summary,
		SubMchID:          strings.TrimSpace(res.SubMchID),
		RecordID:          res.RecordID,
		CompanyName:       res.CompanyName,
		PunishPlan:        res.PunishPlan,
		PunishTime:        res.PunishTime,
		PunishDescription: res.PunishDescription,
		RiskType:          res.RiskType,
		RiskDescription:   res.RiskDescription,
		RawJSON:           string(plain),
	}
	return s.persistAndRefresh(ctx, flow)
}

// HandleMerchantNotify 处理 (B) 合作伙伴订阅·商户新增管控流水通知（topic 20000）。
func (s *ChannelControlNotifyService) HandleMerchantNotify(ctx context.Context, h NotifyHeaders, body []byte) error {
	env, plain, err := s.verifyAndDecrypt(h, body)
	if err != nil {
		return err
	}
	var res merchantNotifyResource
	if err := json.Unmarshal(plain, &res); err != nil {
		return cnErr("管控流水业务对象解析失败: " + err.Error())
	}
	mc := res.MessageContent
	flow := &model.ChannelControlFlow{
		NotifyID:      env.ID,
		Mechanism:     model.ChannelControlMechMerchant,
		EventType:     env.EventType,
		Summary:       env.Summary,
		SubMchID:      strings.TrimSpace(mc.MerchantCode), // 管控流水 merchant_code 即子商户号
		CompanyName:   mc.MerchantCompanyName,
		BusinessCode:  mc.BusinessCode,  // ↔ 第二段 limitation_case_id
		BusinessState: mc.BusinessState, // PUNISHMENT/RECOVERY/DELAYPUNISHMENT/PUNISHMENTCANCELLED
		BusinessTime:  mc.BusinessTime,
		RawJSON:       string(plain),
	}
	return s.persistAndRefresh(ctx, flow)
}

// persistAndRefresh 归属反查 + 幂等落库 + 异步触发第二段查询核实刷新快照。
func (s *ChannelControlNotifyService) persistAndRefresh(ctx context.Context, flow *model.ChannelControlFlow) error {
	// 按 sub_mchid 反查本服务商名下已开通进件单，回填 enroll_id/uid（未匹配到也照落流水，仅无从刷新快照）。
	var enroll *model.ChannelEnroll
	if flow.SubMchID != "" {
		if e, err := s.enrolls.FindApprovedBySubMchID(flow.SubMchID); err == nil && e != nil {
			enroll = e
			flow.EnrollID = e.ID
			flow.UID = e.UID
		}
	}
	if err := s.flows.Create(flow); err != nil {
		if errors.Is(err, repository.ErrFlowDuplicate) {
			return nil // 幂等命中：已处理过，直接应答成功（不重复刷新）
		}
		return cnErr("管控流水落库失败: " + err.Error())
	}
	// 异步核实：回调仅通知"有变化"，明细以第二段查询接口为准（用 background 上下文，不受请求应答生命周期约束）。
	if enroll != nil && s.control != nil {
		enrollID := enroll.ID
		go func() {
			if _, err := s.control.RefreshOne(context.Background(), enrollID); err != nil {
				log.Printf("[channel-control-notify] 回调后刷新快照失败 enroll=%d sub_mchid=%s: %v", enrollID, flow.SubMchID, err)
			}
		}()
	}
	if enroll != nil {
		s.notifyMerchant(enroll, flow)
	}
	return nil
}

// notifyMerchant 处置流水落库后站内信提醒商户（对齐 epay MsgNotice::send('mchrisk', uid, param)，
// 见 includes/lib/MsgNotice.php mchrisk 分支：mchid=渠道子商户号/mchname=商户名称/risk_desc=风险类型/
// punish_type=处罚方案/punish_desc=处罚描述/punish_time=记录时间）。两套机制（A处置通知/B管控流水）
// 字段来源不同，各自取有值的字段拼给同一模板，取不到的用摘要/落库时间兜底。
func (s *ChannelControlNotifyService) notifyMerchant(enroll *model.ChannelEnroll, flow *model.ChannelControlFlow) {
	if s.notice == nil || enroll.UID == 0 {
		return
	}
	riskDesc := flow.RiskType
	if riskDesc == "" {
		riskDesc = flow.RiskDescription
	}
	punishType := flow.PunishPlan
	if punishType == "" {
		punishType = enumText(channelFlowStateText, flow.BusinessState)
	}
	punishDesc := flow.PunishDescription
	if punishDesc == "" {
		punishDesc = flow.Summary
	}
	punishTime := flow.PunishTime
	if punishTime == "" {
		punishTime = flow.BusinessTime
	}
	if punishTime == "" {
		punishTime = flow.CreatedAt.Format(timeLayout)
	}
	go s.notice.Send("mchrisk", enroll.UID, map[string]string{
		"mchid":       flow.SubMchID,
		"mchname":     enroll.MerchantName,
		"risk_desc":   riskDesc,
		"punish_type": punishType,
		"punish_desc": punishDesc,
		"punish_time": punishTime,
	})
}
