package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// WxComplaintClient 微信支付「消费者投诉2.0」服务商版 REST 子客户端（自研扩展）。
//
// 复用 SubMerchantService 的签名 REST 通道（doRequest：BuildAuthorization + Wechatpay-Serial 头 + 应答验签）
// 与媒体上传（uploadMultipart）——两者同包，直接调其未导出方法，零重复实现。
// 服务商语义：用 sp_mchid 签名调用（doRequest 内已用服务商商户号），业务参数带 complainted_mchid（被诉子商户号）。
//
// 覆盖官方 partner 版 14 接口：回调地址 CRUD(4)+回调(1，在 service 层收) / 查询列表·详情·历史(3) /
// 回复·结单·退款审批·即时服务(4) / 图片上传·下载(2)。
type WxComplaintClient struct {
	submch *SubMerchantService
}

// NewWxComplaintClient 基于服务商进件微信引擎构造投诉子客户端（共用同一份服务商凭证）。
func NewWxComplaintClient(submch *SubMerchantService) *WxComplaintClient {
	return &WxComplaintClient{submch: submch}
}

const complaintBasePath = "/v3/merchant-service/complaints-v2"
const complaintNotifyPath = "/v3/merchant-service/complaint-notifications"
const complaintImageUploadPath = "/v3/merchant-service/images/upload"

// —— 组1：投诉通知回调地址 CRUD ——

// NotifyURLResp 回调地址应答（创建/查询/更新共用）。
type NotifyURLResp struct {
	MchID string `json:"mchid"`
	URL   string `json:"url"`
}

// CreateNotifyURL 创建投诉通知回调地址（4012458106）。一个商户号只能建一个，重复报 268435484（数据已存在）。
func (c *WxComplaintClient) CreateNotifyURL(ctx context.Context, notifyURL string) (*NotifyURLResp, []byte, error) {
	body, _ := json.Marshal(map[string]string{"url": notifyURL})
	return c.notifyURLReq(ctx, http.MethodPost, string(body))
}

// GetNotifyURL 查询投诉通知回调地址（4012459065）。未设置报 268435456（数据不存在）。
func (c *WxComplaintClient) GetNotifyURL(ctx context.Context) (*NotifyURLResp, []byte, error) {
	return c.notifyURLReq(ctx, http.MethodGet, "")
}

// UpdateNotifyURL 更新投诉通知回调地址（4012459287）。
func (c *WxComplaintClient) UpdateNotifyURL(ctx context.Context, notifyURL string) (*NotifyURLResp, []byte, error) {
	body, _ := json.Marshal(map[string]string{"url": notifyURL})
	return c.notifyURLReq(ctx, http.MethodPut, string(body))
}

// DeleteNotifyURL 删除投诉通知回调地址（4012460474）。成功返回 204 无体。
func (c *WxComplaintClient) DeleteNotifyURL(ctx context.Context) ([]byte, error) {
	raw, code, err := c.submch.doRequest(ctx, http.MethodDelete, complaintNotifyPath, "")
	if err != nil {
		return raw, err
	}
	if code < 200 || code >= 300 {
		return raw, smErr("删除投诉回调地址失败: " + wxErrMsg(raw))
	}
	return raw, nil
}

func (c *WxComplaintClient) notifyURLReq(ctx context.Context, method, body string) (*NotifyURLResp, []byte, error) {
	raw, code, err := c.submch.doRequest(ctx, method, complaintNotifyPath, body)
	if err != nil {
		return nil, raw, err
	}
	if code < 200 || code >= 300 {
		return nil, raw, smErr("投诉回调地址操作失败: " + wxErrMsg(raw))
	}
	var r NotifyURLResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, raw, fmt.Errorf("解析回调地址应答失败: %w", err)
	}
	return &r, raw, nil
}

// —— 组2：主动查询 ——

// ComplaintOrder 投诉关联订单单项。
type ComplaintOrder struct {
	TransactionID string `json:"transaction_id"`
	OutTradeNo    string `json:"out_trade_no"`
	Amount        int    `json:"amount"` // 订单金额（分）
	State         string `json:"state"`  // 订单状态
}

// ComplaintDetail 投诉单详情（4012691648，官方字段全集）。
type ComplaintDetail struct {
	ComplaintID           string           `json:"complaint_id"`
	ComplaintTime         string           `json:"complaint_time"`
	ComplaintDetail       string           `json:"complaint_detail"`
	ComplaintState        string           `json:"complaint_state"` // PENDING/PROCESSING/PROCESSED
	ComplaintedMchID      string           `json:"complainted_mchid"`
	PayerPhone            string           `json:"payer_phone"` // ★密文（平台证书加密）
	PayerOpenID           string           `json:"payer_openid"`
	ComplaintOrderInfo    []ComplaintOrder `json:"complaint_order_info"`
	ComplaintFullRefunded bool             `json:"complaint_full_refunded"`
	IncomingUserResponse  bool             `json:"incoming_user_response"`
	UserComplaintTimes    int              `json:"user_complaint_times"`
	ComplaintMediaList    []any            `json:"complaint_media_list"`
	ProblemDescription    string           `json:"problem_description"`
	ProblemType           string           `json:"problem_type"` // REFUND/SERVICE_NOT_WORK/OTHERS
	ApplyRefundAmount     int              `json:"apply_refund_amount"`
	UserTagList           []string         `json:"user_tag_list"`
	InPlatformService     bool             `json:"in_platform_service"`
	NeedImmediateService  bool             `json:"need_immediate_service"`
	IsAgentMode           bool             `json:"is_agent_mode"`
}

// ComplaintListResp 投诉单列表应答（4012691285）。
type ComplaintListResp struct {
	Data       []ComplaintDetail `json:"data"`
	Limit      int               `json:"limit"`
	Offset     int               `json:"offset"`
	TotalCount int               `json:"total_count"`
}

// ListComplaintsParams 列表查询入参。日期跨度≤30天，limit[1,50]。
type ListComplaintsParams struct {
	Limit            int    // [1,50]
	Offset           int    // 分页偏移
	BeginDate        string // yyyy-MM-DD
	EndDate          string // yyyy-MM-DD
	ComplaintedMchID string // 按子商户过滤（可选）
}

// ListComplaints 查询投诉单列表（4012691285，轮询兜底用）。
func (c *WxComplaintClient) ListComplaints(ctx context.Context, p ListComplaintsParams) (*ComplaintListResp, []byte, error) {
	q := url.Values{}
	limit := p.Limit
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(p.Offset))
	if p.BeginDate != "" {
		q.Set("begin_date", p.BeginDate)
	}
	if p.EndDate != "" {
		q.Set("end_date", p.EndDate)
	}
	if p.ComplaintedMchID != "" {
		q.Set("complainted_mchid", p.ComplaintedMchID)
	}
	raw, code, err := c.submch.doRequest(ctx, http.MethodGet, complaintBasePath+"?"+q.Encode(), "")
	if err != nil {
		return nil, raw, err
	}
	if code < 200 || code >= 300 {
		return nil, raw, smErr("查询投诉单列表失败: " + wxErrMsg(raw))
	}
	var r ComplaintListResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, raw, fmt.Errorf("解析投诉单列表应答失败: %w", err)
	}
	return &r, raw, nil
}

// GetComplaint 查询投诉单详情（4012691648）。complaintID 必填。
func (c *WxComplaintClient) GetComplaint(ctx context.Context, complaintID string) (*ComplaintDetail, []byte, error) {
	if strings.TrimSpace(complaintID) == "" {
		return nil, nil, smErr("投诉单号为空")
	}
	raw, code, err := c.submch.doRequest(ctx, http.MethodGet, complaintBasePath+"/"+url.PathEscape(complaintID), "")
	if err != nil {
		return nil, raw, err
	}
	if code < 200 || code >= 300 {
		return nil, raw, smErr("查询投诉单详情失败: " + wxErrMsg(raw))
	}
	var r ComplaintDetail
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, raw, fmt.Errorf("解析投诉单详情应答失败: %w", err)
	}
	return &r, raw, nil
}

// NegotiationHistory 协商历史单项（4012691802）。
type NegotiationHistory struct {
	LogID              string   `json:"log_id"`
	Operator           string   `json:"operator"`     // 操作人（USER/MERCHANT/PLATFORM）
	OperateTime        string   `json:"operate_time"` // rfc3339
	OperateType        string   `json:"operate_type"` // 操作类型
	OperateDetails     string   `json:"operate_details"`
	ImageList          []string `json:"image_list"`
	ComplaintMediaList []any    `json:"complaint_media_list"`
}

// NegotiationHistoryResp 协商历史应答。
type NegotiationHistoryResp struct {
	Data       []NegotiationHistory `json:"data"`
	Limit      int                  `json:"limit"`
	Offset     int                  `json:"offset"`
	TotalCount int                  `json:"total_count"`
}

// GetNegotiationHistory 查询投诉单协商历史（4012691802）。limit[1,300]。
func (c *WxComplaintClient) GetNegotiationHistory(ctx context.Context, complaintID string, limit, offset int) (*NegotiationHistoryResp, []byte, error) {
	if strings.TrimSpace(complaintID) == "" {
		return nil, nil, smErr("投诉单号为空")
	}
	if limit <= 0 || limit > 300 {
		limit = 50
	}
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	path := complaintBasePath + "/" + url.PathEscape(complaintID) + "/negotiation-historys?" + q.Encode()
	raw, code, err := c.submch.doRequest(ctx, http.MethodGet, path, "")
	if err != nil {
		return nil, raw, err
	}
	if code < 200 || code >= 300 {
		return nil, raw, smErr("查询协商历史失败: " + wxErrMsg(raw))
	}
	var r NegotiationHistoryResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, raw, fmt.Errorf("解析协商历史应答失败: %w", err)
	}
	return &r, raw, nil
}

// —— 组3：处理投诉 ——

// ResponseReq 回复用户入参（4012467213）。
type ResponseReq struct {
	ComplaintedMchID string   `json:"complainted_mchid"`
	ResponseContent  string   `json:"response_content"`           // ≤200 字符（按开发指引更严口径）
	ResponseImages   []string `json:"response_images,omitempty"`  // 图片 media_id，≤4
	JumpURL          string   `json:"jump_url,omitempty"`
	JumpURLText      string   `json:"jump_url_text,omitempty"`
}

// Response 回复用户（4012467213）。POST .../{complaint_id}/response。成功 204 无体。
func (c *WxComplaintClient) Response(ctx context.Context, complaintID string, req ResponseReq) ([]byte, error) {
	body, _ := json.Marshal(req)
	return c.opPost(ctx, complaintID, "/response", string(body), "回复用户")
}

// Complete 反馈处理完成（4012467217）。POST .../{complaint_id}/complete。body 仅 complainted_mchid。
func (c *WxComplaintClient) Complete(ctx context.Context, complaintID, complaintedMchID string) ([]byte, error) {
	body, _ := json.Marshal(map[string]string{"complainted_mchid": complaintedMchID})
	return c.opPost(ctx, complaintID, "/complete", string(body), "反馈处理完成")
}

// UpdateRefundReq 更新退款审批结果入参（4012467218）。
type UpdateRefundReq struct {
	ComplaintedMchID string   `json:"complainted_mchid"`
	Action           string   `json:"action"`                       // APPROVE 同意 / REJECT 驳回
	LaunchRefundDay  int      `json:"launch_refund_day,omitempty"`  // 承诺退款天数（APPROVE）
	RejectReason     string   `json:"reject_reason,omitempty"`      // 驳回原因（REJECT）
	RejectMediaList  []string `json:"reject_media_list,omitempty"`  // 驳回凭证图片
	Remark           string   `json:"remark,omitempty"`
}

// UpdateRefundProgress 更新退款审批结果（4012467218）。POST .../{complaint_id}/update-refund-progress。
func (c *WxComplaintClient) UpdateRefundProgress(ctx context.Context, complaintID string, req UpdateRefundReq) ([]byte, error) {
	body, _ := json.Marshal(req)
	return c.opPost(ctx, complaintID, "/update-refund-progress", string(body), "更新退款审批")
}

// ImmediateResponseReq 回复即时服务投诉单入参（4017151726）。
type ImmediateResponseReq struct {
	ComplaintedMchID string   `json:"complainted_mchid"`
	ResponseContent  string   `json:"response_content"`
	ResponseImages   []string `json:"response_images,omitempty"`
}

// ResponseImmediate 回复需即时服务的投诉单（4017151726）。POST .../{complaint_id}/immediate-response。
func (c *WxComplaintClient) ResponseImmediate(ctx context.Context, complaintID string, req ImmediateResponseReq) ([]byte, error) {
	body, _ := json.Marshal(req)
	return c.opPost(ctx, complaintID, "/immediate-response", string(body), "回复即时服务投诉")
}

// opPost 处理类接口统一 POST（成功 200/204 均视为成功）。
func (c *WxComplaintClient) opPost(ctx context.Context, complaintID, suffix, body, action string) ([]byte, error) {
	if strings.TrimSpace(complaintID) == "" {
		return nil, smErr("投诉单号为空")
	}
	path := complaintBasePath + "/" + url.PathEscape(complaintID) + suffix
	raw, code, err := c.submch.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return raw, err
	}
	if code < 200 || code >= 300 {
		return raw, smErr(action + "失败: " + wxErrMsg(raw))
	}
	return raw, nil
}

// —— 组4：图片 ——

// UploadImage 上传商户反馈图片（4012467222），返回 media_id 供回复引用（≤4）。
// ★ 注意路径 /v3/merchant-service/images/upload 与进件媒体上传 /v3/merchant/media/upload 不同，
//   故不用 submch.UploadMedia，直接调 uploadMultipart 传投诉图片专用路径。
func (c *WxComplaintClient) UploadImage(ctx context.Context, filename string, data []byte) (string, error) {
	return c.submch.uploadMultipart(ctx, complaintImageUploadPath, filename, data, "投诉图片")
}

// DownloadImage 下载/查看投诉图片（4012467223）。mediaURL 为详情/历史里返回的图片链接（含 query）。
// 返回图片二进制。
func (c *WxComplaintClient) DownloadImage(ctx context.Context, mediaPath string) ([]byte, error) {
	if strings.TrimSpace(mediaPath) == "" {
		return nil, smErr("图片地址为空")
	}
	raw, code, err := c.submch.doRequest(ctx, http.MethodGet, mediaPath, "")
	if err != nil {
		return raw, err
	}
	if code < 200 || code >= 300 {
		return nil, smErr("下载投诉图片失败: " + wxErrMsg(raw))
	}
	return raw, nil
}
