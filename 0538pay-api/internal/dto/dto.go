package dto

// LoginReq 后台登录入参。
type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResp 登录成功返回。
type LoginResp struct {
	Token    string `json:"token"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
}

// OrderView 订单对外响应结构。金额统一格式化为两位小数字符串，
// 字段/json tag 严格对齐前端 mock/orders.ts 的 Order interface。
type OrderView struct {
	TradeNo     string  `json:"trade_no"`
	OutTradeNo  string  `json:"out_trade_no"`
	APITradeNo  string  `json:"api_trade_no"`
	UID         uint    `json:"uid"`
	Domain      string  `json:"domain"`
	Name        string  `json:"name"`
	Money       string  `json:"money"`
	RealMoney   *string `json:"realmoney"`
	GetMoney    string  `json:"getmoney"`
	RefundMoney string  `json:"refundmoney"`
	ProfitMoney string  `json:"profitmoney"`
	Type        int     `json:"type"`
	TypeName    string  `json:"typename"`
	TypeShow    string  `json:"typeshowname"`
	Channel     int     `json:"channel"`
	Plugin      string  `json:"plugin"`
	IP          string  `json:"ip"`
	Buyer       string  `json:"buyer"`
	AddTime     string  `json:"addtime"`
	EndTime     *string `json:"endtime"`
	Status      int8    `json:"status"`
	Settle      int8    `json:"settle"`
	Combine     int8    `json:"combine"`
}

// SubmitResp 下单成功返回（对齐 epay mapi 风格：type + 支付信息）。
// PayType=qrcode 时用 QRCode/PayURL；redirect 时用 PayURL。
type SubmitResp struct {
	TradeNo    string `json:"trade_no"`    // 系统订单号
	OutTradeNo string `json:"out_trade_no"` // 商户订单号
	PayType    string `json:"pay_type"`    // qrcode/redirect/html/wap
	PayURL     string `json:"pay_url,omitempty"`
	QRCode     string `json:"qrcode,omitempty"`
	Money      string `json:"money"` // 订单金额，两位小数
}

// ChannelView 通道对外响应，字段/json tag 对齐前端 mock/channels.ts 的 Channel。
// today/yesterday 为派生统计（非表字段），阶段C1先给0，后续接订单统计。
type ChannelView struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Type      int    `json:"type"`
	TypeName  string `json:"typename"`
	TypeShow  string `json:"typeshowname"`
	Plugin    string `json:"plugin"`
	Mode      int8   `json:"mode"`
	Rate      string `json:"rate"`      // 百分比，两位小数
	CostRate  string `json:"costrate"`
	DayTop    int    `json:"daytop"`
	PayMin    string `json:"paymin"`
	PayMax    string `json:"paymax"`
	Today     string `json:"today"`     // 今日收款（派生）
	Yesterday string `json:"yesterday"` // 昨日收款（派生）
	Status    int8   `json:"status"`
}

// ChannelSaveReq 新增/编辑通道的表单入参（对齐 epay pay_channel 编辑表单：
// name/rate 必填、plugin 必选、type 支付方式；rate/costrate 走字符串再解析为 decimal）。
type ChannelSaveReq struct {
	Name     string `json:"name" binding:"required"`
	Type     int    `json:"type"`
	Plugin   string `json:"plugin" binding:"required"`
	Mode     int8   `json:"mode"`     // 0=平台代收 1=商户直清
	Rate     string `json:"rate" binding:"required"`
	CostRate string `json:"costrate"` // 可空，默认 0
	DayTop   int    `json:"daytop"`   // 单日限额（mode=1 时置 0）
	PayMin   string `json:"paymin"`
	PayMax   string `json:"paymax"`
}

// ChannelStatusReq 状态切换入参。
type ChannelStatusReq struct {
	Status int8 `json:"status"` // 0=关闭 1=开启
}

// ChannelConfigReq 密钥/参数配置入参（config 为 JSON 字符串）。
type ChannelConfigReq struct {
	Config string `json:"config"`
}

// ChannelConfigView 读取通道密钥配置（供配置抽屉回填，含 config 原文 JSON）。
type ChannelConfigView struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Plugin string `json:"plugin"`
	Config string `json:"config"`
}

// ChannelQuery 通道列表查询入参（对齐前端 Channels.vue 的筛选）。
type ChannelQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Keyword  string `form:"keyword"` // 通道 ID / 名称
	Plugin   string `form:"plugin"`  // 插件标识模糊
	Type     *int   `form:"type"`    // 支付方式
	Status   *int   `form:"status"`  // 状态
}

// Normalize 补默认分页值并做安全上限。
func (q *ChannelQuery) Normalize() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
}

// SettleView 结算明细对外响应，字段/json tag 对齐前端 mock/settle.ts 的 SettleRecord。
// 金额统一格式化为两位小数字符串；merchant 为商户名（暂用 uid 派生，接商户名字段后补）。
type SettleView struct {
	ID        uint    `json:"id"`
	Batch     string  `json:"batch"`     // 所属批次号
	UID       uint    `json:"uid"`       // 商户号
	Merchant  string  `json:"merchant"`  // 商户名称（派生）
	Type      int8    `json:"type"`      // 结算方式
	Auto      int8    `json:"auto"`      // 1=自动 0=手动
	Account   string  `json:"account"`   // 结算账号
	Username  string  `json:"username"`  // 结算姓名
	Money     string  `json:"money"`     // 结算金额
	RealMoney string  `json:"realmoney"` // 实际到账
	AddTime   string  `json:"addtime"`   // 创建时间
	EndTime   *string `json:"endtime"`   // 完成时间
	Status    int8    `json:"status"`    // 0待结算1已完成2正在结算3结算失败
	Result    string  `json:"result"`    // 失败原因
}

// SettleBatchView 结算批次对外响应，对齐前端 mock/settle.ts 的 SettleBatch。
type SettleBatchView struct {
	Batch    string `json:"batch"`
	AllMoney string `json:"allmoney"` // 批次总金额
	Count    int    `json:"count"`    // 批次总数量
	Time     string `json:"time"`     // 生成时间
	Status   int8   `json:"status"`   // 0处理中1已完成2部分完成
}

// SettleQuery 结算明细列表查询入参（对齐 admin/slist.php 的筛选/分页）。
type SettleQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Keyword  string `form:"keyword"` // 结算账号/姓名 模糊
	UID      *uint  `form:"uid"`     // 商户号
	Type     *int   `form:"type"`    // 结算方式（0/空=全部）
	Status   *int   `form:"status"`  // 状态（-1/空=全部）
	Batch    string `form:"batch"`   // 所属批次
}

// Normalize 补默认分页值并做安全上限。
func (q *SettleQuery) Normalize() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
}

// SettleStatusReq 单条结算记录状态变更入参。
// Status: 0待结算 1已完成 2正在结算 3结算失败 4删除（删除时余额退回商户）。
// Result: status=3 时可携带失败原因。
type SettleStatusReq struct {
	Status int8   `json:"status"`
	Result string `json:"result"`
}

// CashierView 收银台中间页所需的公开订单信息（无敏感字段，无需鉴权）。
type CashierView struct {
	TradeNo    string `json:"trade_no"`
	OutTradeNo string `json:"out_trade_no"`
	Name       string `json:"name"`
	Money      string `json:"money"`
	Plugin     string `json:"plugin"`      // 渠道标识（mock 时前端展示模拟支付）
	PayType    string `json:"pay_type"`    // 收银台渲染方式 qrcode/redirect/html（真实渠道）
	QRCode     string `json:"qrcode"`      // 二维码内容/支付链接（真实渠道；mock 为空，前端走模拟支付）
	Status     int8   `json:"status"`      // 0未付1已付…（已付则前端提示勿重复支付）
	AddTime    string `json:"addtime"`
	ReturnURL  string `json:"return_url"`  // 支付完成跳回商户
}

// MerchantView 商户对外响应，字段/json tag 对齐前端 mock/merchants.ts 的 Merchant。
type MerchantView struct {
	UID       uint    `json:"uid"`
	GID       int     `json:"gid"`
	GroupName string  `json:"groupname"`
	EndTime   *string `json:"endtime"`
	Money     string  `json:"money"`
	SettleID  int     `json:"settle_id"`
	Account   string  `json:"account"`
	Username  string  `json:"username"`
	QQ        string  `json:"qq"`
	Phone     string  `json:"phone"`
	Email     string  `json:"email"`
	URL       string  `json:"url"`
	AddTime   string  `json:"addtime"`
	Status    int8    `json:"status"`
	Cert      int8    `json:"cert"`
	Pay       int8    `json:"pay"`
	Settle    int8    `json:"settle"`
	UpID      int     `json:"upid"`
	Mode      int8    `json:"mode"`
	Deposit   string  `json:"deposit"`
}

// MerchantQuery 商户列表查询入参（对齐 admin/user.php 的搜索/筛选/分页）。
type MerchantQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Column   string `form:"column"`  // 搜索字段（uid/account/username/url/qq/phone/email）
	Keyword  string `form:"keyword"` // 搜索关键词
	GID      *int   `form:"gid"`     // 用户组筛选
	Status   *int   `form:"status"`  // 用户状态
	Pay      *int   `form:"pay"`     // 支付权限
	SettleSt *int   `form:"settle"`  // 结算权限
}

// Normalize 补默认分页值并做安全上限。
func (q *MerchantQuery) Normalize() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
}

// OrderQuery 订单列表查询入参（对齐 order.php 的搜索/分页）。
type OrderQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Column   string `form:"column"`  // 搜索字段名（trade_no/out_trade_no/...）
	Keyword  string `form:"keyword"` // 搜索关键词
	Status   *int   `form:"status"`  // 状态筛选（可空）
}

// Normalize 补默认分页值并做安全上限。
func (q *OrderQuery) Normalize() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
}
