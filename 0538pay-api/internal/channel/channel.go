// Package channel 定义支付渠道插件的统一抽象与注册表。
//
// 设计要点（抄袭红线）：这是自研的 Go interface + registry 方案，
// 不复刻 epay 的 xxx_plugin.php 钩子命名与目录约定。
// 各渠道在自己的子包 init() 里调用 Register 自注册，payment 层按 key 取用。
package channel

import (
	"context"

	"github.com/shopspring/decimal"
)

// PayType 前端渲染方式。
type PayType string

const (
	PayTypeQRCode   PayType = "qrcode"   // 二维码
	PayTypeRedirect PayType = "redirect" // 跳转 URL
	PayTypeHTML     PayType = "html"     // 表单自动提交
	PayTypeWap      PayType = "wap"      // 手机网页
)

// Config 是单个渠道实例的配置（来自通道表），各渠道自行解释 Extra。
type Config struct {
	AppID     string
	Key       string
	PrivateKey string
	PublicKey  string
	Extra     map[string]string
}

// CreateReq 统一下单入参。
type CreateReq struct {
	TradeNo   string
	Money     decimal.Decimal
	Subject   string
	NotifyURL string
	ReturnURL string
	ClientIP  string
	Extra     map[string]string
}

// CreateResp 统一下单出参。
type CreateResp struct {
	PayType PayType
	PayURL  string
	QRCode  string
	RawHTML string
}

// NotifyResult 回调解析结果。
type NotifyResult struct {
	TradeNo    string          // 商户/系统订单号
	ChannelNo  string          // 渠道流水号
	Money      decimal.Decimal // 实付金额
	Success    bool            // 是否支付成功
	AckContent string          // 需回给渠道的应答内容（如 "success"）
}

// PaymentChannel 是所有支付渠道插件必须实现的接口。
type PaymentChannel interface {
	Key() string
	Create(ctx context.Context, cfg Config, req CreateReq) (CreateResp, error)
	Query(ctx context.Context, cfg Config, tradeNo string) (bool, error)
	Notify(ctx context.Context, cfg Config, raw map[string]string) (NotifyResult, error)
}

// registry 全局渠道注册表。
var registry = map[string]PaymentChannel{}

// Register 注册一个渠道实现，通常在渠道包的 init() 中调用。
func Register(c PaymentChannel) { registry[c.Key()] = c }

// Get 按 key 取渠道实现。
func Get(key string) (PaymentChannel, bool) {
	c, ok := registry[key]
	return c, ok
}

// Keys 返回已注册的所有渠道 key，便于后台展示可用插件。
func Keys() []string {
	ks := make([]string, 0, len(registry))
	for k := range registry {
		ks = append(ks, k)
	}
	return ks
}
