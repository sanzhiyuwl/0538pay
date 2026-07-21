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

// Config 是单个渠道实例的配置（来自通道表 config JSON），各渠道自行解释所需字段。
// 通用字段覆盖大多数渠道；渠道特有的键放 Extra（如微信 mch_id/serial_no/api_v3_key）。
type Config struct {
	AppID      string            // 公众账号/应用 ID
	Key        string            // MD5/APIv3 等对称密钥
	PrivateKey string            // 商户私钥（PEM）
	PublicKey  string            // 平台/渠道公钥（PEM）
	MchID      string            // 商户号（微信等）
	SerialNo   string            // 商户证书序列号（微信 APIv3）
	NotifyURL  string            // 该渠道的回调地址（留空则用系统默认）
	Extra      map[string]string // 渠道特有的其它键值
}

// ExtraOr 返回 Extra 中 key 的值，缺失时返回 def。
func (c Config) ExtraOr(key, def string) string {
	if c.Extra != nil {
		if v, ok := c.Extra[key]; ok && v != "" {
			return v
		}
	}
	return def
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

// 原始回调注入键：handler 把 JSON 型回调（如微信 APIv3）的原始报文体与验签头
// 以下面这些保留键塞进 Notify 的 raw map，渠道层据此验签+解密。表单型回调用不到。
const (
	RawBody      = "_raw_body"      // 回调原始报文体
	RawSignature = "_raw_signature" // 验签签名头
	RawTimestamp = "_raw_timestamp" // 验签时间戳头
	RawNonce     = "_raw_nonce"     // 验签随机串头
	RawSerial    = "_raw_serial"    // 证书/公钥序列号头
)

// Keys 返回已注册的所有渠道 key，便于后台展示可用插件。
func Keys() []string {
	ks := make([]string, 0, len(registry))
	for k := range registry {
		ks = append(ks, k)
	}
	return ks
}
