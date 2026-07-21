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

// ===== 商户中心工作台（D2）=====

// MerchantDashboard 商户工作台聚合数据，对齐前端 mock/merchant/dashboard.ts。
type MerchantDashboard struct {
	Info     MerchantDashInfo    `json:"merchantInfo"`
	Alerts   MerchantAlerts      `json:"alerts"`
	Channels []MerchantChannel   `json:"channels"`
	Announces []AnnounceView     `json:"announces"`
	Trend    SettleTrendView     `json:"trend"`
}

// MerchantDashInfo 工作台商户信息（status 用字符串枚举，对齐前端 mock 消费方式）。
type MerchantDashInfo struct {
	UID             uint    `json:"uid"`
	Name            string  `json:"name"`
	QQ              string  `json:"qq"`
	Status          string  `json:"status"` // normal/banned/payoff/settleoff/auditing/uncert
	GroupName       string  `json:"groupName"`
	Money           float64 `json:"money"`
	SettleMoney     float64 `json:"settleMoney"`
	TodayIncome     float64 `json:"todayIncome"`
	YesterdayIncome float64 `json:"yesterdayIncome"`
	Orders          int64   `json:"orders"`
	OrdersToday     int64   `json:"ordersToday"`
}

// MerchantAlerts 工作台顶部提醒横幅开关。
type MerchantAlerts struct {
	NeedCert   bool `json:"needCert"`
	NoSecurity bool `json:"noSecurity"`
	NoLoginPwd bool `json:"noLoginPwd"`
}

// MerchantChannel 通道费率/收入统计行。
type MerchantChannel struct {
	TypeName    string  `json:"typename"`
	ShowName    string  `json:"showname"`
	Today       float64 `json:"today"`
	Yesterday   float64 `json:"yesterday"`
	SuccessRate float64 `json:"successRate"`
	Rate        string  `json:"rate"`
}

// AnnounceView 公告。
type AnnounceView struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
	Color   string `json:"color"`
	Time    string `json:"time"`
}

// SettleTrendView 结算金额趋势图。
type SettleTrendView struct {
	Labels []string  `json:"labels"`
	Data   []float64 `json:"data"`
}

// MerchantSettleView 商户端结算记录，对齐前端 mock/merchant/settle.ts 的 SettleRecord
// （金额用 number、auto 用 boolean、失败原因 failReason，区别于 admin 端 SettleView）。
type MerchantSettleView struct {
	ID         uint    `json:"id"`
	Type       int8    `json:"type"`       // 1支付宝2微信3QQ4银行卡5支付机构
	Auto       bool    `json:"auto"`       // 是否自动
	Account    string  `json:"account"`    // 结算账号
	Money      float64 `json:"money"`      // 结算金额
	RealMoney  float64 `json:"realmoney"`  // 实际到账
	AddTime    string  `json:"addtime"`    // 结算时间
	Status     int8    `json:"status"`     // 0待结算1已完成2正在结算3失败
	FailReason string  `json:"failReason"` // 失败原因
}

// ApplyInfo 申请提现页信息，对齐前端 mock/merchant/settle.ts 的 applyInfo。
type ApplyInfo struct {
	SettleName     string  `json:"settleName"`     // 提现方式（只读）
	Account        string  `json:"account"`        // 提现账号
	Username       string  `json:"username"`       // 真实姓名
	Money          float64 `json:"money"`          // 当前余额
	EnableMoney    float64 `json:"enableMoney"`    // 可提现余额
	SettleMin      float64 `json:"settleMin"`      // 最低提现额
	SettleMaxLimit int     `json:"settleMaxLimit"` // 每日次数上限
	SettleRate     float64 `json:"settleRate"`     // 手续费率 %
	SettleFeeMin   float64 `json:"settleFeeMin"`   // 最低手续费
	SettleFeeMax   float64 `json:"settleFeeMax"`   // 最高手续费
	SettleType     int     `json:"settleType"`     // 1=D+0 2=D+1
	TodayCount     int     `json:"todayCount"`     // 今日已提现次数
}

// ApplyReq 申请提现入参。
type ApplyReq struct {
	Amount string `json:"amount" binding:"required"` // 提现金额
}

// MerchantApiInfo 商户 API 信息（对齐前端 MerchantApi.vue，V1 MD5 部分）。
// RSA/keytype 属 V2 协议，V2 未实现前不下发（避免造悬空数据）。
type MerchantApiInfo struct {
	UID    uint   `json:"uid"`
	MDKey  string `json:"mdkey"` // 商户通信密钥（MD5 验签，即 AppKey）
	APIURL string `json:"apiurl"`
}

// MerchantProfileReq 修改商户资料入参（仅模型已有字段：收款账号 + 联系方式 + 扣费模式）。
type MerchantProfileReq struct {
	SettleID int    `json:"settle_id"` // 结算方式
	Account  string `json:"account"`   // 收款账号
	Username string `json:"username"`  // 真实姓名
	Email    string `json:"email"`
	QQ       string `json:"qq"`
	URL      string `json:"url"`
	Mode     int8   `json:"mode"` // 手续费扣除模式 0/1
}

// MerchantPwdReq 修改登录密码入参。
type MerchantPwdReq struct {
	OldPwd string `json:"oldpwd"` // 旧密码（已设密码时必填）
	NewPwd string `json:"newpwd" binding:"required"`
}

// RefundReq 订单退款入参（商户端）。
type RefundReq struct {
	TradeNo string `json:"trade_no" binding:"required"`
	// 全额退款，暂不支持部分退款金额与支付密码（支付密码校验待商户资料域）
}

// RecordView 资金流水对外响应，对齐前端 mock/merchant/records.ts（金额用 number）+ epay pre_record。
type RecordView struct {
	ID       uint    `json:"id"`
	Action   int8    `json:"action"`   // 1=增加 2=减少
	Money    float64 `json:"money"`    // 变更金额
	OldMoney float64 `json:"oldmoney"` // 变更前余额
	NewMoney float64 `json:"newmoney"` // 变更后余额
	Type     string  `json:"type"`     // 操作类型文案
	TradeNo  string  `json:"trade_no"` // 关联单号
	Date     string  `json:"date"`     // 时间
}

// RecordQuery 资金流水查询入参（商户端：按类型/关键词筛选 + 分页）。
type RecordQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Action   *int   `form:"action"`  // 1增2减，可空
	Keyword  string `form:"keyword"` // 类型文案 / 关联单号 模糊
	UID      *uint  `form:"-"`       // 商户号（商户端强制注入）
}

// Normalize 补默认分页值并做安全上限。
func (q *RecordQuery) Normalize() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
}

// MerchantLoginReq 商户登录入参（对齐 epay user/login.php 双模式）。
// Type=1 密码登录(Account=邮箱/手机, Password=登录密码)；Type=0 密钥登录(Account=商户ID, Password=通信密钥)。
type MerchantLoginReq struct {
	Type     int8   `json:"type"`               // 1=密码登录 0=密钥登录
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// MerchantLoginResp 商户登录成功返回。
type MerchantLoginResp struct {
	Token string       `json:"token"`
	Info  MerchantInfo `json:"info"`
}

// MerchantInfo 当前登录商户的基础信息 + 资料字段（工作台/顶栏/资料页共用）。
type MerchantInfo struct {
	UID      uint   `json:"uid"`
	Name     string `json:"name"`     // 商户名（暂用 uid 派生，接商户名字段后补）
	Money    string `json:"money"`    // 余额
	Status   int8   `json:"status"`   // 0封禁1正常2未审核
	Pay      int8   `json:"pay"`      // 支付权限
	Settle   int8   `json:"settle"`   // 结算权限
	Cert     int8   `json:"cert"`     // 实名
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	QQ       string `json:"qq"`
	GID      int    `json:"gid"`
	SettleID int    `json:"settle_id"` // 结算方式
	Account  string `json:"account"`   // 收款账号
	Username string `json:"username"`  // 真实姓名
	URL      string `json:"url"`       // 网站域名
	Mode     int8   `json:"mode"`      // 手续费扣除模式
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
	UID      *uint  `form:"-"`       // 商户号过滤（商户端强制注入当前商户，不从 query 绑定）
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
