package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/0538pay/api/internal/repository"
)

// NoticeService 统一对外通知触达中枢（对齐 epay includes/lib/MsgNotice.php send）。
//
// send(scene, uid, param)：
//   - uid==0：管理员场景（regaudit/apply/domain/complain_all/mchrisk_all），按 msgconfig_<scene>
//     开关决定是否发管理员邮件（mail_recv 优先，回退 mail_name）。
//   - uid>0：商户场景，读该商户 msgconfig[scene] 选择通道：1=微信模板消息 / 2=邮件 / 3=短信。
//     order 场景可设 order_money 阈值（低于则不发）；balance 场景带余额阈值。
//
// 三通道分别委托 WxmpService(K-2)/MailService(K-3)/SmsService。任一通道未配置/失败，
// 静默返回 false（通知是附属能力，绝不阻塞或回滚支付/结算主流程）。
type NoticeService struct {
	merchants *repository.MerchantRepo
	cfg       *ConfigService
	mail      *MailService
	wxmp      *WxmpService
	sms       *SmsService
}

func NewNoticeService(m *repository.MerchantRepo, cfg *ConfigService, mail *MailService, wxmp *WxmpService, sms *SmsService) *NoticeService {
	return &NoticeService{merchants: m, cfg: cfg, mail: mail, wxmp: wxmp, sms: sms}
}

// 管理员群发场景（对齐 epay $scene_all）：complain/mchrisk 用 <scene>_all 开关。
var noticeAdminAllScenes = map[string]bool{"complain": true, "mchrisk": true}

// Send 统一发送入口。返回是否成功送出（失败/未配置返回 false，不返回 error 以免调用方误处理）。
func (s *NoticeService) Send(scene string, uid uint, param map[string]string) bool {
	if s == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	if uid == 0 {
		return s.sendAdmin(ctx, scene, param)
	}
	return s.sendMerchant(ctx, scene, uid, param)
}

// sendAdmin uid==0 管理员场景：按 msgconfig 开关发管理员邮件。
func (s *NoticeService) sendAdmin(ctx context.Context, scene string, param map[string]string) bool {
	switchKey := scene
	if noticeAdminAllScenes[scene] {
		switchKey = scene + "_all"
	}
	if s.cfg.Int("msgconfig_"+switchKey, 0) != 1 {
		return false
	}
	return s.sendMailScene(ctx, scene, s.adminReceiver(), param)
}

// sendMerchant uid>0 商户场景：读 msgconfig 选通道。
func (s *NoticeService) sendMerchant(ctx context.Context, scene string, uid uint, param map[string]string) bool {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil || m == nil {
		return false
	}
	mc := parseMsgConfig(m.MsgConfig)

	// order 场景金额阈值：低于商户设置的 order_money 不发（对齐 epay）。
	if scene == "order" {
		if threshold := msgConfFloat(mc, "order_money"); threshold > 0 {
			if money := parseFloat(param["money"]); money < threshold {
				return false
			}
		}
	}
	// balance 场景带余额阈值供文案使用。
	if scene == "balance" {
		if param == nil {
			param = map[string]string{}
		}
		param["msgmoney"] = msgConfStr(mc, "balance_threshold")
	}

	channel := msgConfInt(mc, scene)
	switch channel {
	case 1: // 微信模板消息
		if m.WxUID != "" {
			return s.sendWechatTpl(ctx, scene, m.WxUID, param)
		}
	case 2: // 邮件（需站点该场景邮件开关开启）
		if m.Email != "" && s.cfg.Int("msgconfig_"+scene, 0) == 1 {
			return s.sendMailScene(ctx, scene, m.Email, param)
		}
	case 3: // 短信（仅 balance/complain 有模板，对齐 epay）
		if m.Phone != "" {
			return s.sendSmsScene(ctx, scene, m.Phone, param)
		}
	}
	return false
}

// adminReceiver 管理员收件邮箱：mail_recv 优先，回退 mail_name。
func (s *NoticeService) adminReceiver() string {
	if r := s.cfg.Str("mail_recv"); r != "" {
		return r
	}
	return s.cfg.Str("mail_name")
}

// sendMailScene 组装场景邮件模板并发送（对齐 MsgNotice::get_msg_tpl + send_mail_msg）。
func (s *NoticeService) sendMailScene(ctx context.Context, scene, to string, param map[string]string) bool {
	if to == "" || s.mail == nil {
		return false
	}
	title, content := s.mailTemplate(scene, param)
	if content == "" {
		return false
	}
	return s.mail.Send(ctx, to, title, content) == nil
}

// sendSmsScene balance/complain 短信（对齐 epay MsgNotice 短信分支）。其它场景无短信模板。
func (s *NoticeService) sendSmsScene(ctx context.Context, scene, phone string, param map[string]string) bool {
	if s.sms == nil {
		return false
	}
	var tpl string
	var tplParam map[string]string
	switch scene {
	case "balance":
		tpl = s.cfg.Str("sms_tpl_balance")
		tplParam = map[string]string{"code": param["msgmoney"]}
	case "complain":
		tpl = s.cfg.Str("sms_tpl_complain")
		tplParam = map[string]string{"code": param["trade_no"]}
	default:
		return false
	}
	if tpl == "" {
		return false
	}
	return s.sms.SendCommon(ctx, phone, tpl, tplParam) == nil
}

// sendWechatTpl 组装场景模板消息并发送（对齐 MsgNotice::send_wechat_tplmsg）。
// login_wx 指定作为通知主体的公众号（wid）；未配 template_id 或 wid 则跳过。
func (s *NoticeService) sendWechatTpl(ctx context.Context, scene, openid string, param map[string]string) bool {
	if s.wxmp == nil {
		return false
	}
	wid := s.cfg.Int("login_wx", 0)
	if wid == 0 {
		return false
	}
	templateID, data, jump := s.wechatTemplate(scene, param)
	if templateID == "" {
		return false
	}
	return s.wxmp.SendTemplateMessage(ctx, uint(wid), openid, templateID, jump, data) == nil
}

// ---- 模板组装 ----

// mailTemplate 生成场景邮件标题+正文（对齐 MsgNotice::get_msg_tpl）。
func (s *NoticeService) mailTemplate(scene string, p map[string]string) (title, content string) {
	site := s.cfg.Str("sitename")
	now := time.Now().Format("2006-01-02 15:04:05")
	switch scene {
	case "regaudit":
		return "新注册商户待审核提醒",
			fmt.Sprintf("尊敬的%s管理员，网站有新注册的商户待审核，请及时前往用户列表审核处理。<br/>商户ID：%s<br/>注册账号：%s<br/>注册时间：%s",
				site, p["uid"], p["account"], now)
	case "apply":
		return "新的提现待处理提醒",
			fmt.Sprintf("尊敬的%s管理员，商户%s发起了手动提现申请，请及时处理。<br/>商户ID：%s<br/>提现方式：%s<br/>提现金额：%s<br/>提交时间：%s",
				site, p["uid"], p["uid"], p["type"], p["realmoney"], now)
	case "domain":
		return "新的授权支付域名待审核提醒",
			fmt.Sprintf("尊敬的%s管理员，商户%s提交了新的授权支付域名，请及时审核处理。<br/>商户ID：%s<br/>授权域名：%s<br/>提交时间：%s",
				site, p["uid"], p["uid"], p["domain"], now)
	case "order":
		return "新订单通知 - " + site,
			fmt.Sprintf("尊敬的商户，您有一条新订单通知。<br/>商品名称：%s<br/>订单金额：¥%s<br/>支付方式：%s<br/>商户订单号：%s<br/>系统订单号：%s<br/>支付完成时间：%s",
				p["name"], p["money"], p["type"], p["out_trade_no"], p["trade_no"], p["time"])
	case "settle":
		return "结算完成通知 - " + site,
			fmt.Sprintf("尊敬的商户，今日结算已完成，请查收。<br/>结算金额：¥%s<br/>实际到账：¥%s<br/>结算账号：%s<br/>结算完成时间：%s",
				p["money"], p["realmoney"], p["account"], p["time"])
	case "login":
		return "账号登录通知 - " + site,
			fmt.Sprintf("尊敬的商户，您的账号<b>%s</b>已于%s成功登录到商户平台。<br/>登录IP：%s<br/>登录时间：%s",
				p["user"], p["time"], p["clientip"], p["time"])
	case "balance":
		return "商户余额不足提醒 - " + site,
			fmt.Sprintf("尊敬的商户，您的手续费余额不足%s元，为避免造成订单失败请及时充值。<br/>当前余额：%s元",
				p["msgmoney"], p["money"])
	case "complain":
		return "支付交易投诉通知 - " + site,
			fmt.Sprintf("尊敬的商户，%s！<br/>系统订单号：%s<br/>投诉原因：%s<br/>投诉详情：%s<br/>商品名称：%s<br/>订单金额：¥%s<br/>投诉时间：%s",
				p["type"], p["trade_no"], p["title"], p["content"], p["ordername"], p["money"], p["time"])
	case "mchrisk":
		return "渠道商户违规处置通知 - " + site,
			fmt.Sprintf("尊敬的商户，您有新的渠道商户违规处置记录！<br/>渠道子商户号：%s<br/>商户名称：%s<br/>风险类型：%s<br/>处罚方案：%s（%s）<br/>记录时间：%s",
				p["mchid"], p["mchname"], p["risk_desc"], p["punish_type"], p["punish_desc"], p["punish_time"])
	}
	return "", ""
}

// wechatTemplate 生成场景模板消息 data + 跳转 URL（对齐 MsgNotice::send_wechat_tplmsg，
// 字段映射由后台 wxnotice_tpl_<scene>_<field> 配置项决定，未配置的字段跳过）。
func (s *NoticeService) wechatTemplate(scene string, p map[string]string) (templateID string, data TplData, jump string) {
	site := s.cfg.Str("sitename")
	data = TplData{}
	put := func(fieldKey, value string) {
		if k := s.cfg.Str(fieldKey); k != "" {
			data[k] = struct {
				Value string `json:"value"`
			}{Value: value}
		}
	}
	switch scene {
	case "order":
		templateID = s.cfg.Str("wxnotice_tpl_order")
		put("wxnotice_tpl_order_no", p["trade_no"])
		put("wxnotice_tpl_order_name", truncRune(p["name"], 20))
		put("wxnotice_tpl_order_money", "¥"+p["money"])
		put("wxnotice_tpl_order_time", p["time"])
		put("wxnotice_tpl_order_outno", truncByte(p["out_trade_no"], 32))
		jump = "/m/orders"
	case "settle":
		templateID = s.cfg.Str("wxnotice_tpl_settle")
		put("wxnotice_tpl_settle_type", "结算成功")
		put("wxnotice_tpl_settle_account", p["account"])
		put("wxnotice_tpl_settle_money", "¥"+p["money"])
		put("wxnotice_tpl_settle_realmoney", "¥"+p["realmoney"])
		put("wxnotice_tpl_settle_time", p["time"])
		jump = "/m/settle"
	case "login":
		templateID = s.cfg.Str("wxnotice_tpl_login")
		put("wxnotice_tpl_login_user", p["user"])
		put("wxnotice_tpl_login_time", p["time"])
		put("wxnotice_tpl_login_name", site)
		put("wxnotice_tpl_login_ip", p["clientip"])
		put("wxnotice_tpl_login_iploc", p["ipinfo"])
		jump = "/m"
	case "balance":
		templateID = s.cfg.Str("wxnotice_tpl_balance")
		put("wxnotice_tpl_balance_user", p["user"])
		put("wxnotice_tpl_balance_time", p["time"])
		put("wxnotice_tpl_balance_money", p["money"])
		put("wxnotice_tpl_balance_msg", "为避免造成订单失败，请及时充值")
		jump = "/m"
	}
	return templateID, data, jump
}

// ---- msgconfig 解析工具 ----

// parseMsgConfig 解析商户 msgconfig JSON（值可能是数字或字符串，统一为 map）。
func parseMsgConfig(raw string) map[string]interface{} {
	m := map[string]interface{}{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return m
	}
	_ = json.Unmarshal([]byte(raw), &m)
	return m
}

func msgConfInt(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case string:
		return int(parseFloat(v))
	}
	return 0
}

func msgConfFloat(m map[string]interface{}, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case string:
		return parseFloat(v)
	}
	return 0
}

func msgConfStr(m map[string]interface{}, key string) string {
	switch v := m[key].(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%g", v)
	}
	return ""
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	var f float64
	_, _ = fmt.Sscanf(s, "%g", &f)
	return f
}

// truncRune 按字符截断（中文按字，对齐 epay mb_substr）。
func truncRune(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// truncByte 按字节截断（对齐 epay substr）。
func truncByte(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
