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
	// 支付场景参数（A-2，对齐 epay device/method/sub_openid/auth_code）。渠道按需消费；不支持则忽略。
	Method   string // web/jump/jsapi/scan（下单方式）
	Device   string // pc/mobile/qq/wechat/alipay/app（设备/来源）
	SubOpenID string // JSAPI 场景买家 openid
	SubAppID  string // 微信 JSAPI 场景公众号/小程序 appid
	AuthCode  string // 付款码支付的 auth_code（付款码前缀判定支付方式）
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

// RefundReq 统一退款入参。
type RefundReq struct {
	TradeNo     string          // 系统订单号（下单时的 out_trade_no）
	ChannelNo   string          // 渠道支付流水号（支付宝 trade_no / 微信 transaction_id，可空则用 TradeNo）
	OutRefundNo string          // 商户退款单号（幂等键）
	Money       decimal.Decimal // 本次退款金额
	TotalMoney  decimal.Decimal // 原订单总额（微信 APIv3 退款必填）
	Reason      string          // 退款原因
}

// RefundResp 统一退款出参。
type RefundResp struct {
	RefundNo string          // 渠道退款单号
	Money    decimal.Decimal // 实际退款金额
	Success  bool            // 渠道是否受理成功
}

// Refunder 可选能力：支持原路退款的渠道额外实现此接口。
// 收单主链退款时若渠道实现了它则调真实退款，否则仅走余额层（对齐 epay 有渠道退款/无则人工）。
type Refunder interface {
	Refund(ctx context.Context, cfg Config, req RefundReq) (RefundResp, error)
}

// TransferReq 统一代付（打款到账户）入参（对齐 epay transfer/transfer_query/transfer_proof）。
type TransferReq struct {
	OutBizNo string          // 商户代付单号（幂等键）
	Account  string          // 收款账号（支付宝账号/微信 openid/银行卡号等）
	Name     string          // 收款人姓名（实名校验用）
	Money    decimal.Decimal // 打款金额
	Remark   string          // 打款备注/说明
	Extra    map[string]string
}

// TransferResp 统一代付出参。
type TransferResp struct {
	TransferNo string // 渠道打款流水号
	Status     int8   // 0=处理中 1=成功 2=失败（对齐 epay transfer status）
	Message    string // 渠道返回的说明（失败原因等）
	ProofURL   string // 打款凭证下载/查看地址（渠道支持则回填）
}

// Transferer 可选能力：支持渠道打款（代付到账户）的渠道额外实现此接口。
// 代付主链发起打款时若渠道实现了它则调真实打款 API，否则仅走本地扣余额记账
// （对齐 epay Transfer.php：有渠道打款则调 API，无则人工/余额层）。
// 打款实现依赖各渠道真实凭证，本接口先定契约，具体渠道实现待凭证。
type Transferer interface {
	Transfer(ctx context.Context, cfg Config, req TransferReq) (TransferResp, error)
	// TransferQuery 查询打款状态（对齐 epay transfer_query）。渠道可选实现，不支持返回 error。
	TransferQuery(ctx context.Context, cfg Config, outBizNo string) (TransferResp, error)
}

// FieldInput 描述一个渠道配置项的元数据（供后台按插件动态渲染密钥表单，
// 对齐 epay 插件 $info['inputs'] 的字段声明）。避免每接一个新渠道都要前后端手写表单。
type FieldInput struct {
	Name    string   `json:"name"`    // 配置键名（写入 Config.Extra 或通用字段）
	Label   string   `json:"label"`   // 表单显示名
	Type    string   `json:"type"`    // 控件类型：text/password/textarea/select
	Options []string `json:"options"` // type=select 时的可选项
	Require bool     `json:"require"` // 是否必填
	Tip     string   `json:"tip"`     // 输入提示/说明
}

// ProductType 描述渠道支持的一种支付产品/形态（对齐 epay $info['transtypes']，
// 如 支付宝当面付/网页/APP，微信 Native/JSAPI/H5）。供后台选择通道产品。
type ProductType struct {
	Code string `json:"code"` // 产品编码
	Name string `json:"name"` // 产品显示名
}

// Configurable 可选能力：渠道声明自己的配置字段与支持产品（元数据驱动后台表单）。
// 实现此接口的渠道，后台"配置密钥"抽屉据 Inputs() 动态渲染，无需为每个渠道手写表单；
// Products() 供后台选择通道支持的支付产品形态。未实现则后台退回通用 JSON 文本框。
type Configurable interface {
	Inputs() []FieldInput
	Products() []ProductType
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

// PluginMeta 渠道插件的能力与配置元数据（供后台动态渲染密钥表单/展示能力）。
type PluginMeta struct {
	Key           string        `json:"key"`
	Inputs        []FieldInput  `json:"inputs"`   // 配置字段（实现 Configurable 才有）
	Products      []ProductType `json:"products"` // 支持产品（实现 Configurable 才有）
	CanRefund     bool          `json:"can_refund"`     // 是否实现 Refunder
	CanTransfer   bool          `json:"can_transfer"`   // 是否实现 Transferer
	Configurable  bool          `json:"configurable"`   // 是否实现 Configurable
}

// Meta 返回某渠道的能力元数据（key 未注册返回 ok=false）。
func Meta(key string) (PluginMeta, bool) {
	c, ok := registry[key]
	if !ok {
		return PluginMeta{}, false
	}
	m := PluginMeta{Key: key}
	if _, ok := c.(Refunder); ok {
		m.CanRefund = true
	}
	if _, ok := c.(Transferer); ok {
		m.CanTransfer = true
	}
	if cfg, ok := c.(Configurable); ok {
		m.Configurable = true
		m.Inputs = cfg.Inputs()
		m.Products = cfg.Products()
	}
	return m, true
}

// AllMeta 返回所有已注册渠道的能力元数据。
func AllMeta() []PluginMeta {
	out := make([]PluginMeta, 0, len(registry))
	for k := range registry {
		if m, ok := Meta(k); ok {
			out = append(out, m)
		}
	}
	return out
}
