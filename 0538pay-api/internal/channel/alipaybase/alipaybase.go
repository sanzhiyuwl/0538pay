// Package alipaybase 提供支付宝各支付方式（当面付/网页/H5/JSAPI/APP/退款）共用的
// 网关调用、系统参数组装、RSA2 签名与应答解析逻辑。
//
// 所有方式同源：同一 openapi.alipay.com/gateway.do 网关、同一 pkg/alipay RSA2 签名，
// 仅 method / biz_content / 应答处理不同。抽出公共部分，各变体渠道只需组 biz_content + 解析各自应答字段。
package alipaybase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/0538pay/api/internal/channel"
	"github.com/0538pay/api/pkg/alipay"
	"github.com/shopspring/decimal"
)

const (
	GatewayURL  = "https://openapi.alipay.com/gateway.do"
	httpTimeout = 15 * time.Second
	TimeLayout  = "2006-01-02 15:04:05"
)

// BuildSysParams 组装支付宝系统参数（含 biz_content），未签名。
// method 如 alipay.trade.page.pay；notifyURL/returnURL 空则不带。
func BuildSysParams(cfg channel.Config, method, bizContent, notifyURL, returnURL string) map[string]string {
	p := map[string]string{
		"app_id":      cfg.AppID,
		"method":      method,
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format(TimeLayout),
		"version":     "1.0",
		"biz_content": bizContent,
	}
	if n := strings.TrimSpace(notifyURL); n != "" {
		p["notify_url"] = n
	}
	if r := strings.TrimSpace(returnURL); r != "" {
		p["return_url"] = r
	}
	return p
}

// SignedParams 用商户私钥对系统参数 RSA2 签名，返回含 sign 的完整参数表。
func SignedParams(cfg channel.Config, params map[string]string) (map[string]string, error) {
	priv, err := alipay.ParsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("解析支付宝应用私钥失败: %w", err)
	}
	sign, err := alipay.SignParams(priv, params)
	if err != nil {
		return nil, err
	}
	params["sign"] = sign
	return params, nil
}

// PostGateway 以 form-urlencoded POST 网关，返回应答 body（用于 JSON 应答类：precreate/create/query/refund）。
func PostGateway(ctx context.Context, params map[string]string) ([]byte, error) {
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", GatewayURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求支付宝网关失败: %w", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// BuildRedirectURL 组装 GET 跳转 URL（用于 page.pay/wap.pay 页面类，浏览器直接跳转即渲染收银台）。
// 支付宝允许把已签名的系统参数作为 query 拼到网关地址上，用户浏览器 GET 打开。
func BuildRedirectURL(params map[string]string) string {
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	return GatewayURL + "?" + form.Encode()
}

// ParseResp 通用应答解析：取出指定 responseKey 下的 code/sub_code/msg/sub_msg + 原始 json.RawMessage 交调用方细解。
// code=10000 视为成功。
func ParseResp(body []byte, responseKey string) (json.RawMessage, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("支付宝应答解析失败: %w", err)
	}
	raw, ok := envelope[responseKey]
	if !ok {
		return nil, fmt.Errorf("支付宝应答缺少 %s", responseKey)
	}
	var head struct {
		Code    string `json:"code"`
		Msg     string `json:"msg"`
		SubCode string `json:"sub_code"`
		SubMsg  string `json:"sub_msg"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, err
	}
	if head.Code != "10000" {
		return nil, fmt.Errorf("支付宝接口失败[%s]: %s %s", head.Code, head.Msg, head.SubMsg)
	}
	return raw, nil
}

// ParseNotify 表单型异步回调通用解析：剔除框架保留键 → 支付宝公钥验签 → 读 trade_status 判成功。
// 供各支付宝支付方式渠道复用（回调格式统一）。
func ParseNotify(cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	params := map[string]string{}
	for k, v := range raw {
		if strings.HasPrefix(k, "_raw_") {
			continue
		}
		params[k] = v
	}
	if params["out_trade_no"] == "" {
		return channel.NotifyResult{}, fmt.Errorf("回调缺少 out_trade_no")
	}
	if cfg.PublicKey != "" {
		pub, err := alipay.ParsePublicKey(cfg.PublicKey)
		if err != nil {
			return channel.NotifyResult{}, fmt.Errorf("解析支付宝公钥失败: %w", err)
		}
		if !alipay.Verify(pub, params) {
			return channel.NotifyResult{}, fmt.Errorf("支付宝回调验签失败")
		}
	}
	status := params["trade_status"]
	success := status == "TRADE_SUCCESS" || status == "TRADE_FINISHED"
	money, _ := decimal.NewFromString(params["total_amount"])
	return channel.NotifyResult{
		TradeNo:    params["out_trade_no"],
		ChannelNo:  params["trade_no"],
		Money:      money,
		Success:    success,
		AckContent: "success",
	}, nil
}

// QueryPaid 主动查单 alipay.trade.query，判是否已支付。各支付宝方式通用。
func QueryPaid(ctx context.Context, cfg channel.Config, tradeNo string) (bool, error) {
	if cfg.AppID == "" {
		return false, fmt.Errorf("支付宝通道缺少 appid 配置")
	}
	biz, _ := json.Marshal(map[string]string{"out_trade_no": tradeNo})
	params, err := SignedParams(cfg, BuildSysParams(cfg, "alipay.trade.query", string(biz), "", ""))
	if err != nil {
		return false, err
	}
	body, err := PostGateway(ctx, params)
	if err != nil {
		return false, err
	}
	raw, err := ParseResp(body, "alipay_trade_query_response")
	if err != nil {
		if strings.Contains(err.Error(), "ACQ.TRADE_NOT_EXIST") {
			return false, nil
		}
		return false, err
	}
	var r struct {
		TradeStatus string `json:"trade_status"`
	}
	_ = json.Unmarshal(raw, &r)
	return r.TradeStatus == "TRADE_SUCCESS" || r.TradeStatus == "TRADE_FINISHED", nil
}

// Refund 通用退款 alipay.trade.refund。各支付宝方式通用（同网关+签名）。
func Refund(ctx context.Context, cfg channel.Config, req channel.RefundReq) (channel.RefundResp, error) {
	if cfg.AppID == "" {
		return channel.RefundResp{}, fmt.Errorf("支付宝通道缺少 appid 配置")
	}
	biz := map[string]string{
		"out_trade_no":  req.TradeNo,
		"refund_amount": req.Money.StringFixed(2),
	}
	if req.ChannelNo != "" {
		biz["trade_no"] = req.ChannelNo
	}
	if req.OutRefundNo != "" {
		biz["out_request_no"] = req.OutRefundNo
	}
	if req.Reason != "" {
		biz["refund_reason"] = req.Reason
	}
	bizJSON, _ := json.Marshal(biz)
	params, err := SignedParams(cfg, BuildSysParams(cfg, "alipay.trade.refund", string(bizJSON), "", ""))
	if err != nil {
		return channel.RefundResp{}, err
	}
	body, err := PostGateway(ctx, params)
	if err != nil {
		return channel.RefundResp{}, err
	}
	raw, err := ParseResp(body, "alipay_trade_refund_response")
	if err != nil {
		return channel.RefundResp{}, err
	}
	var r struct {
		TradeNo   string `json:"trade_no"`
		RefundFee string `json:"refund_fee"`
		FundChange string `json:"fund_change"`
	}
	_ = json.Unmarshal(raw, &r)
	money, _ := decimal.NewFromString(r.RefundFee)
	return channel.RefundResp{
		RefundNo: r.TradeNo,
		Money:    money,
		Success:  true, // code=10000 即受理成功（fund_change=Y 表示本次实际退款）
	}, nil
}
