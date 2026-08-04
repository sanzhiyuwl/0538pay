package channel

// Descriptor 是渠道插件的「展示元数据」：中文名 + 品牌族 + 协议 + 形态。
//
// 我方渠道按「支付形态」注册（wxnative/wxjsapi/wxv2micro…），与 epay 按「商户模式」组织的
// 插件名录天然对不齐。后台「已实现插件」页此前靠前端 mock 名录反查 key→中文名，
// 形态级 key（wxv2micro 等）名录里没有，只能显示裸 key，且是双源维护。
//
// 这里把展示元数据收敛为 Go 侧单一数据源：Meta() 据此填充 PluginMeta，
// 前端直接用后端 ShowName 并按 Brand 分组折叠，无需再维护第二份名录。
// 新增渠道时在本表登记一行即可（缺失则回退裸 key，不影响功能）。
type Descriptor struct {
	ShowName string   `json:"showname"` // 完整中文名，如「微信支付 · Native 扫码（APIv3）」
	Brand    string   `json:"brand"`    // 品牌族，用于前端分组，如「微信支付」「支付宝」
	Protocol string   `json:"protocol"` // 协议/版本，如「APIv3」「APIv2」「V1(MD5)」
	Form     string   `json:"form"`     // 支付产品，如「Native 扫码」「JSAPI」「H5」
	Methods  []string `json:"methods"`  // 支持的支付方式（alipay/wxpay/qqpay/bank），对齐 epay pre_plugin.types，前端按选定支付方式过滤插件候选
	// Delegate 标记「已退役的形态包」：仍注册、仅作聚合门面的委托目标，不再直接建通道。
	// 后端 saveChannel 拒绝以该 key 新建通道，前端建通道下拉过滤掉（单一数据源，替代前端硬编码 wxFormKeys）。
	Delegate bool `json:"delegate"`
}

// descriptors 登记所有已实现渠道的展示元数据（key → Descriptor）。
// 与各渠道 init() 注册的 key 一一对应；顺序无关（前端按 Brand+Form 自行排序）。
var descriptors = map[string]Descriptor{
	// 微信支付 APIv3 聚合门面（一个通道承载 Native/JSAPI/H5/APP 全形态，下单按 apptype+设备自动分派，批次一）
	"wxpay": {ShowName: "微信支付（APIv3）", Brand: "微信支付", Protocol: "APIv3", Form: "聚合", Methods: []string{"wxpay"}},
	// 微信支付 APIv3 形态包（直连/服务商由 config sub_mchid 自适应，同一 key 覆盖两种模式）
	// —— 批次一起被 wxpay 门面聚合；批次三退役后仅作委托目标，不再直接建通道。
	"wxnative": {ShowName: "微信支付 · Native 扫码", Brand: "微信支付", Protocol: "APIv3", Form: "Native 扫码", Methods: []string{"wxpay"}, Delegate: true},
	"wxjsapi":  {ShowName: "微信支付 · JSAPI 公众号", Brand: "微信支付", Protocol: "APIv3", Form: "JSAPI", Methods: []string{"wxpay"}, Delegate: true},
	"wxh5":     {ShowName: "微信支付 · H5", Brand: "微信支付", Protocol: "APIv3", Form: "H5", Methods: []string{"wxpay"}, Delegate: true},
	"wxapp":    {ShowName: "微信支付 · APP", Brand: "微信支付", Protocol: "APIv3", Form: "APP", Methods: []string{"wxpay"}, Delegate: true},
	// 微信支付 APIv2 聚合门面（一个通道承载全形态含付款码，下单按 apptype+设备自动分派，批次一）
	"wxpayv2": {ShowName: "微信支付（APIv2）", Brand: "微信支付", Protocol: "APIv2", Form: "聚合", Methods: []string{"wxpay"}},
	// 微信支付 APIv2 形态包 —— 批次一起被 wxpayv2 门面聚合；批次三退役后仅作委托目标。
	"wxv2native": {ShowName: "微信支付 · Native 扫码", Brand: "微信支付", Protocol: "APIv2", Form: "Native 扫码", Methods: []string{"wxpay"}, Delegate: true},
	"wxv2jsapi":  {ShowName: "微信支付 · JSAPI 公众号", Brand: "微信支付", Protocol: "APIv2", Form: "JSAPI", Methods: []string{"wxpay"}, Delegate: true},
	"wxv2h5":     {ShowName: "微信支付 · H5", Brand: "微信支付", Protocol: "APIv2", Form: "H5", Methods: []string{"wxpay"}, Delegate: true},
	"wxv2app":    {ShowName: "微信支付 · APP", Brand: "微信支付", Protocol: "APIv2", Form: "APP", Methods: []string{"wxpay"}, Delegate: true},
	"wxv2micro":  {ShowName: "微信支付 · 付款码", Brand: "微信支付", Protocol: "APIv2", Form: "付款码", Methods: []string{"wxpay"}, Delegate: true},
	// 支付宝
	"alipayf2f":  {ShowName: "支付宝 · 当面付扫码", Brand: "支付宝", Protocol: "开放平台", Form: "当面付扫码", Methods: []string{"alipay"}},
	"alipaypage": {ShowName: "支付宝 · 电脑网站", Brand: "支付宝", Protocol: "开放平台", Form: "电脑网站", Methods: []string{"alipay"}},
	"alipaywap":  {ShowName: "支付宝 · 手机网站", Brand: "支付宝", Protocol: "开放平台", Form: "手机网站 WAP", Methods: []string{"alipay"}},
	// 彩虹易支付（对接上游易支付站点，聚合多方式）
	"epay":  {ShowName: "彩虹易支付 · V1（MD5）", Brand: "彩虹易支付", Protocol: "V1(MD5)", Form: "聚合", Methods: []string{"alipay", "wxpay", "qqpay", "bank"}},
	"epayn": {ShowName: "彩虹易支付 · V2（RSA）", Brand: "彩虹易支付", Protocol: "V2(RSA)", Form: "聚合", Methods: []string{"alipay", "wxpay", "qqpay", "bank"}},
	// 七相聚合支付（对接外部上游 qixiangpay.cn，兼容易支付 V1(MD5) 协议，仅平台自用收款）
	"qixiang": {ShowName: "七相聚合支付", Brand: "七相聚合", Protocol: "V1(MD5)", Form: "聚合", Methods: []string{"alipay", "wxpay"}},
	// 持牌机构 / 个人码
	"fuiou2": {ShowName: "富友支付（合作方）", Brand: "富友支付", Protocol: "合作方版", Form: "扫码", Methods: []string{"alipay", "wxpay", "bank"}},
	"vmq":    {ShowName: "V免签（个人码）", Brand: "V免签", Protocol: "监控回调", Form: "个人收款码", Methods: []string{"alipay", "wxpay", "qqpay"}},
	// 测试桩（前端会过滤，不展示）
	"mock": {ShowName: "模拟渠道（测试桩）", Brand: "测试", Protocol: "mock", Form: "模拟", Methods: []string{"alipay", "wxpay", "qqpay", "bank"}},
}

// Describe 返回某 key 的展示元数据；未登记则回退用 key 兜底（Brand/ShowName 均置 key）。
func Describe(key string) Descriptor {
	if d, ok := descriptors[key]; ok {
		return d
	}
	return Descriptor{ShowName: key, Brand: key, Protocol: "", Form: ""}
}
