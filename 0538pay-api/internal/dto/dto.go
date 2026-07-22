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

// AdminProfile 当前管理员账号资料（账号设置回填）。
type AdminProfile struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
}

// AdminChangePwdReq 管理员改登录密码入参。
type AdminChangePwdReq struct {
	OldPwd  string `json:"oldpwd" binding:"required"`
	NewPwd  string `json:"newpwd" binding:"required"`
	NewPwd2 string `json:"newpwd2" binding:"required"`
}

// AdminChangePayPwdReq 管理员改支付密码入参（对齐 epay set.php mod=paypwd_n）。
type AdminChangePayPwdReq struct {
	OldPwd  string `json:"oldpwd" binding:"required"`
	NewPwd  string `json:"newpwd" binding:"required"`
	NewPwd2 string `json:"newpwd2" binding:"required"`
}

// AdminProfileReq 管理员改账号资料入参（昵称 + 用户名）。
type AdminProfileReq struct {
	Nickname string `json:"nickname"`
	Username string `json:"username" binding:"required"`
}

// AdminView 管理员列表项（Admins.vue 行）。
type AdminView struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	Role      string `json:"role"`
	Status    int8   `json:"status"`
	LastLogin string `json:"last_login"`
	CreatedAt string `json:"created_at"`
}

// AdminSaveReq 新增/编辑管理员入参（编辑时 Password 留空则不改）。
type AdminSaveReq struct {
	Username string `json:"username" binding:"required"`
	Nickname string `json:"nickname"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Status   int8   `json:"status"`
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
	PayType    string `json:"pay_type"`    // qrcode/redirect/html/wap/jsapi/app/scan/urlscheme…
	PayURL     string `json:"pay_url,omitempty"`
	QRCode     string `json:"qrcode,omitempty"`
	RawHTML    string `json:"html,omitempty"`    // 表单自动提交 HTML / JSAPI 参数 JSON（A-2）
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
	SuccessRate string `json:"success_rate"` // 今日成功率%（派生，两位小数）
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
	UID      uint   `json:"uid"`
	MDKey    string `json:"mdkey"`  // 商户通信密钥（MD5 验签，即 AppKey）
	APIURL   string `json:"apiurl"`
	KeyType  int8   `json:"keytype"` // 0=MD5+RSA兼容 1=仅RSA安全
	HasRSA   bool   `json:"has_rsa"` // 是否已配置 RSA 公钥
}

// KeyTypeReq 设置签名模式入参。
type KeyTypeReq struct {
	KeyType int8 `json:"keytype"`
}

// MerchantProfileReq 修改商户资料入参（对齐 epay ajax2.php edit_info）。
type MerchantProfileReq struct {
	SettleID int    `json:"settle_id"` // 结算方式
	Account  string `json:"account"`   // 收款账号
	Username string `json:"username"`  // 真实姓名
	Email    string `json:"email"`
	QQ       string `json:"qq"`
	URL      string `json:"url"`
	Mode     int8   `json:"mode"` // 手续费扣除模式 0/1
	// 以下四项对齐 epay edit_info（用指针区分"未提交"与"提交为0"，nil 则不改该字段）。
	KeyLogin    *int8   `json:"keylogin"`     // 开启密钥登录 0/1
	Refund      *int8   `json:"refund"`       // 订单退款 API 开关 0/1
	Transfer    *int8   `json:"transfer"`     // 代付 API 开关 0/1
	RemainMoney *string `json:"remain_money"` // 预留余额（不参与自动结算）
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
// UID 供后台列表展示商户号（商户端可忽略）。
type RecordView struct {
	ID       uint    `json:"id"`
	UID      uint    `json:"uid"`      // 商户号（后台列表用）
	Action   int8    `json:"action"`   // 1=增加 2=减少
	Money    float64 `json:"money"`    // 变更金额
	OldMoney float64 `json:"oldmoney"` // 变更前余额
	NewMoney float64 `json:"newmoney"` // 变更后余额
	Type     string  `json:"type"`     // 操作类型文案
	TradeNo  string  `json:"trade_no"` // 关联单号
	Date     string  `json:"date"`     // 时间
}

// RecordQuery 资金流水查询入参。
// 商户端：service 强制注入 UID + 按 action/keyword 筛选（越权由注入覆盖保证）。
// 后台：可按 uid/type/column+value/时间范围筛选（对齐 epay record.php + ajax_user.php recordList）。
type RecordQuery struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"pageSize"`
	Action    *int   `form:"action"`    // 1增2减，可空
	Keyword   string `form:"keyword"`   // 类型文案 / 关联单号 模糊（商户端）
	UID       *uint  `form:"uid"`       // 商户号（商户端 service 强制覆盖；后台按 query 筛选）
	Type      string `form:"type"`      // 操作类型文案精确匹配（后台）
	Column    string `form:"column"`    // 后台模糊搜索字段：type/money/trade_no
	Value     string `form:"value"`     // 后台模糊搜索值
	StartTime string `form:"starttime"` // 时间范围起（yyyy-mm-dd）
	EndTime   string `form:"endtime"`   // 时间范围止（yyyy-mm-dd）
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

// RecordStats 资金明细统计（对齐 epay ajax_user.php record_stats：增加/减少/总计金额 + 计数）。
type RecordStats struct {
	IncMoney   float64 `json:"incMoney"`   // 增加金额合计
	DecMoney   float64 `json:"decMoney"`   // 减少金额合计
	TotalMoney float64 `json:"totalMoney"` // 净变更（增 - 减）
	IncCount   int64   `json:"incCount"`   // 入账笔数
	DecCount   int64   `json:"decCount"`   // 出账笔数
	TotalCount int64   `json:"totalCount"` // 总笔数
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
	// edit_info 四项（供资料页回填开关/预留余额）。
	KeyLogin    int8   `json:"keylogin"`     // 开启密钥登录 0/1
	Refund      int8   `json:"refund"`       // 订单退款 API 开关 0/1
	Transfer    int8   `json:"transfer"`     // 代付 API 开关 0/1
	RemainMoney string `json:"remain_money"` // 预留余额
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
	Status   int8   `json:"status"`
	Result   string `json:"result"`
	Password string `json:"password"` // 删除退回(status=4)需管理员密码二次校验(对齐 epay admin_paypwd)
	AdminID  uint   `json:"-"`        // 从鉴权上下文注入，不信任前端
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
	Refund    int8    `json:"refund"`   // 退款API权限 0关1开(A-7)
	Transfer  int8    `json:"transfer"` // 代付API权限 0关1开(A-7)
	UpID      int     `json:"upid"`
	Mode      int8    `json:"mode"`
	Deposit   string  `json:"deposit"`
	// 编辑表单回填（非 PII）：订单名模板/聚合收款开关/预留余额/实名类型
	OrderName   string `json:"ordername"`
	OpenCode    int8   `json:"open_code"`
	RemainMoney string `json:"remain_money"`
	CertType    int8   `json:"certtype"`
}

// MerchantCreateReq 后台添加商户入参（对齐 epay uset.php addUser 表单）。
// 手机号/邮箱不能同时为空且各自唯一；密钥后端随机生成；密码可选（bcrypt）。
type MerchantCreateReq struct {
	GID      int    `json:"gid"`       // 用户组
	SettleID int    `json:"settle_id"` // 结算方式 1支付宝2微信3QQ4银行卡5支付机构
	Account  string `json:"account"`   // 结算账号
	Username string `json:"username"`  // 结算姓名
	URL      string `json:"url"`       // 域名
	Email    string `json:"email"`
	QQ       string `json:"qq"`
	Phone    string `json:"phone"`
	Mode     int8   `json:"mode"`     // 手续费扣除模式 0余额扣费 1订单加费
	Pay      int8   `json:"pay"`      // 支付权限 0关1开2未审核
	Settle   int8   `json:"settle"`   // 结算权限 0/1
	Status   int8   `json:"status"`   // 0封禁1正常2未审核
	Password string `json:"password"` // 登录密码（可选）
}

// MerchantEditReq 后台编辑商户入参（对齐 epay editUser）。money 直接覆盖（管理员强改，不走流水）。
type MerchantEditReq struct {
	GID      int    `json:"gid"`
	UpID     int    `json:"upid"`      // 邀请方
	SettleID int    `json:"settle_id"`
	Account  string `json:"account"`
	Username string `json:"username"`
	Money    string `json:"money"` // 余额（直接覆盖，管理员强改，不产生流水）
	URL      string `json:"url"`
	Email    string `json:"email"`
	QQ       string `json:"qq"`
	Phone    string `json:"phone"`
	Mode     int8   `json:"mode"`
	Pay      int8   `json:"pay"`
	Settle   int8   `json:"settle"`
	Status   int8   `json:"status"`
	Password string `json:"password"` // 非空则改密（bcrypt）
	// 对齐 epay ajax_user.php edit：订单名模板/聚合收款开关/预留余额/保证金/实名信息
	OrderName   string `json:"ordername"`
	OpenCode    int8   `json:"open_code"`
	RemainMoney string `json:"remain_money"`
	Deposit     string `json:"deposit"`
	Cert        int8   `json:"cert"`      // 实名状态 0未认证/审核中 1已认证
	CertType    int8   `json:"certtype"`  // 0个人 1企业
	CertName    string `json:"certname"`  // 真实姓名
	CertNo      string `json:"certno"`    // 证件号
	CertCorp    string `json:"certcorpname"` // 企业名称
	CertCorpNo  string `json:"certcorpno"`   // 企业证件号
}

// MerchantRechargeReq 商户余额充值/扣除入参（走 changeUserMoney 流水）。
type MerchantRechargeReq struct {
	Action int8   `json:"action"` // 0=充值(加款) 1=扣除(扣款)
	Amount string `json:"amount" binding:"required"`
}

// MerchantGroupReq 修改商户用户组 + 有效期入参（对齐 epay setUserGroup）。
type MerchantGroupReq struct {
	GID     int    `json:"gid"`
	EndTime string `json:"endtime"` // 到期时间 yyyy-mm-dd（空=永久）
}

// MerchantSetStatusReq 商户权限/状态切换入参（对齐 epay setUser：type 分派）。
// Field: user(整体状态) / pay(支付权限) / settle(结算权限)。
type MerchantSetStatusReq struct {
	Field  string `json:"field"`  // user/pay/settle
	Status int8   `json:"status"` // 目标值
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

// ===== 代付 / 转账（C3）=====

// TransferView 代付记录对外响应，字段/json tag 对齐前端 mock/transfer.ts 的 TransferRecord。
// 金额格式化为两位小数字符串；后台端展示 uid（0=管理员发起）。
type TransferView struct {
	BizNo      string  `json:"biz_no"`       // 交易号
	PayOrderNo string  `json:"pay_order_no"` // 第三方转账单号
	UID        uint    `json:"uid"`          // 商户号（0=管理员）
	Type       string  `json:"type"`         // 付款方式 alipay/wxpay/qqpay/bank
	Channel    int     `json:"channel"`      // 通道 id
	Account    string  `json:"account"`      // 收款账号
	Username   string  `json:"username"`     // 收款姓名
	Money      string  `json:"money"`        // 到账金额
	CostMoney  string  `json:"costmoney"`    // 商户扣款（含手续费）
	Desc       string  `json:"desc"`         // 备注
	AddTime    string  `json:"addtime"`      // 提交时间
	PayTime    *string `json:"paytime"`      // 付款时间（处理中为 null）
	Status     int8    `json:"status"`       // 0处理中 1成功 2失败
	Result     string  `json:"result"`       // 失败原因
}

// TransferStats 代付概况统计（对齐前端 calcTransferStats）。
type TransferStats struct {
	Total           int64   `json:"total"`
	TotalMoney      float64 `json:"totalMoney"`
	SuccessMoney    float64 `json:"successMoney"`
	SuccessCount    int64   `json:"successCount"`
	ProcessingCount int64   `json:"processingCount"`
	FailCount       int64   `json:"failCount"`
}

// TransferQuery 代付列表查询入参（对齐 transfer.php 的搜索/筛选/分页）。
type TransferQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Keyword  string `form:"keyword"` // 交易号/收款账号/姓名 模糊
	UID      *uint  `form:"uid"`     // 商户号（后台按 query；商户端 service 强制注入）
	Type     string `form:"type"`    // 付款方式
	Status   *int   `form:"status"`  // 状态（-1/空=全部）
}

// Normalize 补默认分页值并做安全上限。
func (q *TransferQuery) Normalize() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
}

// TransferCreateReq 发起代付入参（后台与商户端共用；后台 uid=0 免费，商户端走费率+余额校验）。
type TransferCreateReq struct {
	BizNo    string `json:"biz_no"`   // 交易号（19位数字；空则后端自动生成）
	Type     string `json:"type" binding:"required"` // alipay/wxpay/qqpay/bank
	Channel  int    `json:"channel"`  // 通道 id（可空，取该方式默认通道）
	Account  string `json:"account" binding:"required"` // 收款账号
	Username string `json:"username"` // 收款姓名（选填）
	Money    string `json:"money" binding:"required"`   // 到账金额
	Desc     string `json:"desc"`     // 备注（≤32字）
	Password string `json:"password"` // 身份校验：后台=管理员密码 / 商户=登录密码
}

// TransferBatchReq 后台批量代付入参（C-2，对齐 epay transfer_batch）。一次校验密码，逐条处理。
type TransferBatchReq struct {
	Password string              `json:"password"` // 管理员密码（一次校验）
	Items    []TransferCreateReq `json:"items"`    // 批量代付明细
}

// TransferStatusReq 后台手动改单条代付状态（不动资金）。
// Status: 1成功 2失败；Result 失败原因（可选）。
type TransferStatusReq struct {
	Status int8   `json:"status"`
	Result string `json:"result"`
}

// TransferBizReq 仅带交易号的操作入参（退回/删除/查询）。
type TransferBizReq struct {
	BizNo string `json:"biz_no" binding:"required"`
}

// ===== 分账 profit-sharing（C3）=====

// PsOrderView 分账订单对外响应，字段/json tag 对齐前端 mock/profitsharing.ts 的 PsOrder。
type PsOrderView struct {
	ID          uint   `json:"id"`
	TradeNo     string `json:"trade_no"`     // 系统订单号
	APITradeNo  string `json:"api_trade_no"` // 接口订单号
	RID         uint   `json:"rid"`          // 分账规则 id
	RuleName    string `json:"rulename"`     // 规则名称（派生：比例 + 接收方）
	ChannelID   int    `json:"channelid"`    // 支付通道 id
	ChannelName string `json:"channelname"`  // 通道名称
	Receiver    string `json:"receiver"`     // 接收方
	Money       string `json:"money"`        // 分账金额
	AddTime     string `json:"addtime"`      // 时间
	Status      int8   `json:"status"`       // 0待分账 1已提交 2成功 3失败 4取消
	Result      string `json:"result"`       // 失败原因
}

// PsStats 分账统计概况（对齐前端 calcPsStats）。
type PsStats struct {
	TotalMoney   float64 `json:"totalMoney"`
	SuccessMoney float64 `json:"successMoney"`
	FailMoney    float64 `json:"failMoney"`
	TotalCount   int64   `json:"totalCount"`
	SuccessCount int64   `json:"successCount"`
	FailCount    int64   `json:"failCount"`
	SuccessRate  float64 `json:"successRate"`
}

// PsOrderQuery 分账订单列表查询入参（对齐 ps_order.php 的搜索/筛选/分页）。
type PsOrderQuery struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"pageSize"`
	RID       *uint  `form:"rid"`       // 分账规则 id
	Status    *int   `form:"status"`    // 状态（-1/空=全部）
	Column    string `form:"column"`    // 搜索字段 trade_no/api_trade_no/money
	Value     string `form:"value"`     // 搜索值
	StartTime string `form:"starttime"` // 时间范围起
	EndTime   string `form:"endtime"`   // 时间范围止
}

// Normalize 补默认分页值并做安全上限。
func (q *PsOrderQuery) Normalize() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
}

// PsStatusReq 分账订单状态操作入参（提交/查询/回退/取消/改金额/删除，用 action 区分）。
type PsStatusReq struct {
	Action string `json:"action"` // submit/query/return/cancel/editmoney/delete
	Money  string `json:"money"`  // editmoney 时的新金额
}

// PsReceiverView 分账规则对外响应（C-1，对齐前端规则管理表格）。
type PsReceiverView struct {
	ID          uint   `json:"id"`
	Channel     int    `json:"channel"`      // 支付通道 id
	ChannelName string `json:"channel_name"` // 派生通道名
	SubChannel  int    `json:"subchannel"`   // 子通道 id（0=不限）
	UID         int    `json:"uid"`          // 绑定商户（0=通道级全局）
	Account     string `json:"account"`      // 接收方账号（| 分隔多接收方）
	Name        string `json:"name"`         // 接收方姓名
	Rate        string `json:"rate"`         // 分账比例 %
	MinMoney    string `json:"minmoney"`     // 订单最小金额门槛
	Status      int8   `json:"status"`       // 0关闭 1开启
	AddTime     string `json:"addtime"`
}

// PsReceiverReq 分账规则新增/编辑入参（C-1，对齐 epay add/edit_receiver）。
type PsReceiverReq struct {
	Channel    int    `json:"channel"`    // 支付通道 id（必填）
	SubChannel int    `json:"subchannel"` // 子通道 id（0=不限）
	UID        int    `json:"uid"`        // 绑定商户（0/空=通道级全局）
	Account    string `json:"account"`    // 接收方账号（必填，| 分隔多接收方）
	Name       string `json:"name"`       // 接收方姓名（可空）
	Rate       string `json:"rate"`       // 分账比例 %（空=30，多接收方用 | 分隔）
	MinMoney   string `json:"minmoney"`   // 订单最小金额门槛（空=0 不限）
}

// PsReceiverStatusReq 分账规则开关入参。
type PsReceiverStatusReq struct {
	Status int8 `json:"status"` // 0关 1开
}

// ===== 风控 / 黑名单 / 域名（C4）=====

// RiskView 风控记录对外响应（只读），对齐前端 mock/risk.ts 的 RiskRecord。
type RiskView struct {
	ID      uint   `json:"id"`
	UID     uint   `json:"uid"`
	Type    int8   `json:"type"`    // 0关键词屏蔽 1成功率 2通知失败 3投诉率
	Content string `json:"content"` // 风控内容
	URL     string `json:"url"`     // 风控网址
	Date    string `json:"date"`    // 时间
}

// RiskQuery 风控记录查询入参（对齐 riskList：精确等值搜索 column+value + 类型）。
type RiskQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Column   string `form:"column"` // uid/url/content
	Value    string `form:"value"`  // 精确等值
	Type     *int   `form:"type"`   // -1/空=全部
}

func (q *RiskQuery) Normalize() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
}

// BlacklistView 黑名单对外响应，对齐前端 mock/blacklist.ts 的 BlackItem。
type BlacklistView struct {
	ID      uint    `json:"id"`
	Type    int8    `json:"type"`    // 0支付账号 1IP
	Content string  `json:"content"` // 账号/IP
	AddTime string  `json:"addtime"` // 添加时间
	EndTime *string `json:"endtime"` // 过期时间（null=永久）
	Remark  string  `json:"remark"`  // 备注
}

// BlacklistQuery 黑名单查询入参（对齐 blackList：kw 精确等值 + 类型）。
type BlacklistQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Keyword  string `form:"kw"`   // content 精确等值
	Type     *int   `form:"type"` // -1/空=全部
}

func (q *BlacklistQuery) Normalize() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
}

// BlacklistAddReq 添加黑名单入参（对齐 addBlack）。
type BlacklistAddReq struct {
	Type    int8   `json:"type"`
	Content string `json:"content" binding:"required"`
	Days    int    `json:"days"`   // 有效期天数，0=永久
	Remark  string `json:"remark"`
}

// DomainView 授权域名对外响应，对齐前端 mock/domains.ts 的 DomainItem。
type DomainView struct {
	ID      uint    `json:"id"`
	UID     uint    `json:"uid"`
	Domain  string  `json:"domain"`
	Status  int8    `json:"status"`  // 0待审核 1正常 2拒绝
	AddTime string  `json:"addtime"` // 添加时间
	EndTime *string `json:"endtime"` // 审核时间（null=未审核）
}

// DomainQuery 授权域名查询入参（对齐 domainList：uid + kw 精确等值 + 状态）。
type DomainQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	UID      *uint  `form:"uid"`
	Keyword  string `form:"kw"`      // domain 精确等值
	Status   *int   `form:"dstatus"` // -1/空=全部
}

func (q *DomainQuery) Normalize() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
}

// DomainAddReq 后台添加授权域名入参（后台加免审 status=1）。
type DomainAddReq struct {
	UID    uint   `json:"uid" binding:"required"`
	Domain string `json:"domain" binding:"required"`
}

// DomainStatusReq 域名审核/状态变更入参。
type DomainStatusReq struct {
	Status int8 `json:"status"` // 1通过 2拒绝
}

// ===== 官网 CMS（自研）=====

// SiteConfigReq 保存官网 CMS 文档入参（value 为整份 JSON 字符串）。
type SiteConfigReq struct {
	Value string `json:"value"`
}

// ===== 统计 / 日志 / 邀请码（C5）=====

// StatColumn 统计表列定义（key + 中文名）。
type StatColumn struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// StatRow 统计表一行（一个商户）：各列金额 + 合计。
type StatRow struct {
	UID    uint               `json:"uid"`
	Name   string             `json:"name"`   // 商户标识（结算姓名/域名/派生）
	Values map[string]float64 `json:"values"` // 各列金额
	Total  float64            `json:"total"`  // 行合计
}

// StatResult 商户支付统计结果（交叉透视表 + 列合计）。
type StatResult struct {
	Columns   []StatColumn       `json:"columns"`
	Rows      []StatRow          `json:"rows"`
	Totals    map[string]float64 `json:"totals"` // 列合计
	Grand     float64            `json:"grand"`  // 总计
}

// StatQuery 商户支付统计查询入参（对齐 ustat.php：method + type + 时间范围）。
type StatQuery struct {
	Method    string `form:"method"`    // type=按支付方式 / channel=按支付通道
	Type      int    `form:"type"`      // 0订单金额 1支付金额 2分成金额 3手续费利润 4代付金额
	StartDay  string `form:"startday"`  // yyyy-mm-dd
	EndDay    string `form:"endday"`    // yyyy-mm-dd
}

// BuyerStatQuery 支付用户统计入参（C-3，对齐 epay buyerStat）。
type BuyerStatQuery struct {
	Method   int    `form:"method"`   // 0按付款账号 buyer / 1按IP / 2按手机号 mobile
	Type     int    `form:"type"`     // 支付方式ID（0=全部）
	StartDay string `form:"startday"` // yyyy-mm-dd（必填）
	EndDay   string `form:"endday"`   // yyyy-mm-dd（必填）
}

// BuyerStatRow 支付用户统计一行（C-3）。
type BuyerStatRow struct {
	User    string `json:"user"`     // 付款人标识
	Count   int64  `json:"count"`    // 付款次数
	Amount  string `json:"amount"`   // 累计金额
	IsBlack bool   `json:"is_black"` // 是否黑名单
}

// LogView 登录日志对外响应，对齐前端 mock。
type LogView struct {
	ID   uint   `json:"id"`
	UID  uint   `json:"uid"`
	Type string `json:"type"`
	IP   string `json:"ip"`
	City string `json:"city"`
	Date string `json:"date"`
}

// LogQuery 登录日志查询入参（对齐 logList：column 精确等值 + 分页）。
type LogQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Column   string `form:"column"` // uid/ip
	Value    string `form:"value"`
}

func (q *LogQuery) Normalize() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
}

// InviteView 邀请码对外响应，对齐前端 mock/invitecodes.ts。
type InviteView struct {
	ID      uint    `json:"id"`
	Code    string  `json:"code"`
	Status  int8    `json:"status"`  // 0未使用 1已使用
	AddTime string  `json:"addtime"` // 生成时间
	UseTime *string `json:"usetime"` // 使用时间
	UID     *uint   `json:"uid"`     // 使用者
}

// InviteQuery 邀请码查询入参。
type InviteQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Keyword  string `form:"kw"`     // code 精确等值
	Status   *int   `form:"status"` // -1/空=全部
}

func (q *InviteQuery) Normalize() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
}

// InviteGenReq 批量生成邀请码入参。
type InviteGenReq struct {
	Num int `json:"num"` // 生成个数
}

// ===== 实名认证（D3，第三方认证待凭证）=====

// CertInfo 实名认证页信息。
type CertInfo struct {
	Cert      int8   `json:"cert"`       // 0未认证/审核中 1已认证
	CertType  int8   `json:"certtype"`   // 0个人 1企业
	CertName  string `json:"certname"`   // 脱敏姓名
	CertNo    string `json:"certno"`     // 脱敏证件号
	CertCorp  string `json:"certcorp"`   // 企业名
	CertTime  string `json:"certtime"`   // 认证时间
	CertMoney float64 `json:"certmoney"` // 工本费
	Method    string `json:"method"`     // 认证方式说明
	CorpOpen  bool   `json:"corpopen"`   // 是否开放企业认证
}

// CertSubmitReq 实名认证提交入参。
type CertSubmitReq struct {
	CertType int8   `json:"certtype"` // 0个人 1企业
	CertName string `json:"certname" binding:"required"`
	CertNo   string `json:"certno" binding:"required"`
	CertCorp string `json:"certcorp"` // 企业名（企业认证时）
}

// RechargeReq 余额充值入参（金额 + 渠道插件）。余额本身是充值目标，故无余额支付选项。
type RechargeReq struct {
	Amount string `json:"amount" binding:"required"`
	Plugin string `json:"plugin"` // 渠道插件（mock 可真跑；真实渠道待凭证）
}

// ===== 保证金 / 购买会员（D3 增值）=====

// DepositInfo 保证金页信息（当前保证金/门槛/可用余额）。
type DepositInfo struct {
	Deposit    float64 `json:"deposit"`    // 当前保证金
	DepositMin float64 `json:"depositMin"` // 最低保证金要求
	Money      float64 `json:"money"`      // 可用余额
}

// DepositReq 保证金充值/提取入参（金额）。
type DepositReq struct {
	Amount string `json:"amount" binding:"required"`
	// PayType: balance=余额支付(即时) / 其它=渠道(待凭证)。充值时用。
	PayType string `json:"pay_type"`
}

// GroupPlanView 会员套餐对外响应，对齐前端 mock/merchant/groupbuy.ts 的 GroupPlan。
type GroupPlanView struct {
	ID     int      `json:"id"`     // gid
	Name   string   `json:"name"`
	Price  float64  `json:"price"`  // 单期售价
	Expire int      `json:"expire"` // 有效期月数（0=永久）
	Rates  []GroupRateItem `json:"rates"` // 费率说明
}

// GroupRateItem 费率项（通道名 + 费率）。
type GroupRateItem struct {
	Label string `json:"label"`
	Rate  string `json:"rate"`
}

// GroupCurrentView 当前会员状态。
type GroupCurrentView struct {
	GID    int    `json:"gid"`
	Name   string `json:"name"`
	Expire string `json:"expire"` // 到期时间（"—"=永久/无）
}

// GroupBuyReq 购买会员入参。
type GroupBuyReq struct {
	GID     int    `json:"gid" binding:"required"`
	Num     int    `json:"num"`      // 购买月数（永久组忽略）
	PayType string `json:"pay_type"` // balance=余额支付(即时) / 其它=渠道(待凭证)
}

// ===== 用户组管理（后台，对齐 epay glist/gedit/group.php）=====

// GroupView 用户组对外响应（后台列表/卡片），对齐前端 mock/groups.ts 的 Group。
type GroupView struct {
	GID           int             `json:"gid"`
	Name          string          `json:"name"`
	IsBuy         int8            `json:"isbuy"`         // 是否上架可购买 0/1
	Price         string          `json:"price"`         // 售价（两位小数）
	Expire        int             `json:"expire"`        // 有效期月数（0=永久）
	Sort          int             `json:"sort"`          // 排序
	Visible       string          `json:"visible"`       // 可见范围
	Rates         []GroupRateItem `json:"rates"`         // 通道费率说明（解析自 info）
	Info          string          `json:"info"`          // 费率 JSON 原文（编辑回填）
	Config        string          `json:"config"`        // 功能配置 JSON 原文（编辑回填）
	Settings      string          `json:"settings"`      // 用户变量定义
	MerchantCount int64           `json:"merchantCount"` // 该组下商户数
}

// GroupSaveReq 新增/编辑用户组入参（对齐 epay saveGroup）。
// info/config 为 JSON 字符串；组名唯一；expire=有效期月数。
type GroupSaveReq struct {
	Name     string `json:"name" binding:"required"`
	IsBuy    int8   `json:"isbuy"`   // 是否上架可购买 0/1
	Price    string `json:"price"`   // 售价（可空=0）
	Expire   int    `json:"expire"`  // 有效期月数（0=永久）
	Sort     int    `json:"sort"`    // 排序
	Visible  string `json:"visible"` // 可见范围（GID 列表逗号分隔）
	Info     string `json:"info"`    // 通道费率 JSON（[{label,rate}] 或对象）
	Config   string `json:"config"`  // 功能配置 JSON
	Settings string `json:"settings"`
}

// GroupBuyStatusReq 用户组上/下架入参（对齐 epay changebuy）。
type GroupBuyStatusReq struct {
	IsBuy int8 `json:"isbuy"` // 1=上架 0=下架
}

// ===== 订单写操作（后台，对齐 epay ajax_order.php）=====

// OrderStatusReq 裸改订单状态入参（改未完成 0 / 改已完成 1，对齐 epay setStatus）。
type OrderStatusReq struct {
	Status int8 `json:"status"`
}

// OrderRefundInfo 退款前查询可退金额（对齐 epay getmoney / refund_info）。
type OrderRefundInfo struct {
	TradeNo    string  `json:"trade_no"`
	RealMoney  float64 `json:"realmoney"`   // 订单实付
	Refunded   float64 `json:"refunded"`    // 已退累计
	Refundable float64 `json:"refundable"`  // 本次最多可退
	CanAPI     bool    `json:"can_api"`     // 渠道是否支持 API 原路退款
}

// OrderRefundReq 退款入参（对齐 epay refund/apirefund）。
// API=true 走原路退款(需管理员密码+渠道支持)；false 为手动退款(仅扣商户余额)。
type OrderRefundReq struct {
	TradeNo  string `json:"trade_no" binding:"required"`
	Money    string `json:"money" binding:"required"` // 本次退款金额
	API      bool   `json:"api"`                      // 是否原路退款
	Password string `json:"password"`                 // 管理员密码（API 退款校验）
}

// OrderBatchReq 批量操作入参（对齐 epay operation）。
// Action: 0改未完成 1改已完成 2冻结 3解冻 4删除。
type OrderBatchReq struct {
	Action   int8     `json:"action"`
	TradeNos []string `json:"trade_nos"`
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

// ===== 通道轮询组（后台，对齐 epay pay_roll.php + ajax_pay.php getRoll/saveRoll）=====

// RollChannelItem 轮询组内的一个通道项（通道ID + 权重）。
type RollChannelItem struct {
	Channel     int    `json:"channel"`     // 通道ID
	ChannelName string `json:"channelname"` // 通道名（展示用，保存时忽略）
	Weight      int    `json:"weight"`      // 权重（1-99，仅 kind=1 权重随机有效）
}

// RollView 轮询组对外响应，对齐前端 mock/rolls.ts 的 Roll。
type RollView struct {
	ID           uint              `json:"id"`
	Name         string            `json:"name"`
	Type         int               `json:"type"`
	TypeShowName string            `json:"typeshowname"`
	Kind         int8              `json:"kind"`     // 0=顺序 1=权重随机 2=首个启用
	Channels     []RollChannelItem `json:"channels"` // 组内通道（解析自 info）
	Status       int8              `json:"status"`
}

// RollSaveReq 新增/编辑轮询组入参（name/type/kind + 组内通道列表）。
type RollSaveReq struct {
	Name     string            `json:"name" binding:"required"`
	Type     int               `json:"type"`
	Kind     int8              `json:"kind"`
	Channels []RollChannelItem `json:"channels"`
}

// RollStatusReq 轮询组状态切换入参。
type RollStatusReq struct {
	Status int8 `json:"status"`
}

// ===== 子通道（后台商户维度，对齐 epay pre_subchannel + ajax_user saveSubChannel）=====

// SubChannelView 子通道对外响应。
type SubChannelView struct {
	ID          uint   `json:"id"`
	Channel     int    `json:"channel"`     // 归属主通道ID
	ChannelName string `json:"channelname"` // 主通道名（展示用）
	UID         uint   `json:"uid"`
	Name        string `json:"name"`
	Status      int8   `json:"status"`
	Info        string `json:"info"`    // 自定义参数 JSON 原文（编辑回填）
	UseTime     string `json:"usetime"` // 上次使用时间（"—"=从未使用）
}

// SubChannelSaveReq 新增/编辑子通道入参。
type SubChannelSaveReq struct {
	Channel int    `json:"channel" binding:"required"` // 归属主通道ID
	Name    string `json:"name" binding:"required"`
	Info    string `json:"info"` // 自定义参数 JSON（占位替换用，可空）
}

// SubChannelStatusReq 子通道状态切换入参。
type SubChannelStatusReq struct {
	Status int8 `json:"status"`
}

// ===== 用户组通道分配（后台，对齐 epay pre_group.info 的 {typeid:{type,channel,rate}}）=====

// GroupAssignItem 用户组对某支付方式的分配配置（前端提交/回填用）。
type GroupAssignItem struct {
	Type    int    `json:"type"`    // 支付方式ID
	Kind    string `json:"kind"`    // "channel"|"roll"（正整数目标是通道还是轮询组）
	Channel string `json:"channel"` // "0"关/"-1"随机/"-2"子通道/正整数
	Rate    string `json:"rate"`    // 组级费率覆盖（百分数字符串，空=用通道默认）
}

// GroupAssignSaveReq 保存用户组通道分配入参（整组一次性覆盖）。
type GroupAssignSaveReq struct {
	Assigns []GroupAssignItem `json:"assigns"`
}

// ===== 支付方式 pay_type（后台，对齐 epay pay_type.php）=====

// PayTypeView 支付方式对外响应（对齐前端 mock/paytypes.ts）。
type PayTypeView struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	ShowName string `json:"showname"`
	Device   int8   `json:"device"`
	Today    string `json:"today"`  // 今日收款（派生，暂给 0.00）
	Status   int8   `json:"status"`
}

// PayTypeSaveReq 新增/编辑支付方式入参。
type PayTypeSaveReq struct {
	Name     string `json:"name" binding:"required"`
	ShowName string `json:"showname" binding:"required"`
	Device   int8   `json:"device"`
}

// PayTypeStatusReq 状态切换入参。
type PayTypeStatusReq struct {
	Status int8 `json:"status"`
}

// ===== 微信公众号/小程序 pay_weixin（后台，对齐 epay pay_weixin.php）=====

// WeixinView 公众号/小程序对外响应（appsecret 脱敏）。
type WeixinView struct {
	ID        uint   `json:"id"`
	Type      int8   `json:"type"`
	Name      string `json:"name"`
	AppID     string `json:"appid"`
	AppSecret string `json:"appsecret"` // 脱敏展示
}

// WeixinSaveReq 新增/编辑入参。
type WeixinSaveReq struct {
	Type      int8   `json:"type"`
	Name      string `json:"name" binding:"required"`
	AppID     string `json:"appid" binding:"required"`
	AppSecret string `json:"appsecret"`
}

// ===== 企业微信 pay_wework（后台，对齐 epay pay_wework.php）=====

// WeworkView 企业微信对外响应（appsecret 脱敏，含客服数）。
type WeworkView struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	AppID     string `json:"appid"`
	AppSecret string `json:"appsecret"` // 脱敏展示
	KfNum     int64  `json:"kfnum"`     // 客服账号数（派生）
	Status    int8   `json:"status"`
}

// WeworkSaveReq 新增/编辑入参。
type WeworkSaveReq struct {
	Name      string `json:"name" binding:"required"`
	AppID     string `json:"appid" binding:"required"`
	AppSecret string `json:"appsecret"`
}

// WeworkStatusReq 状态切换入参。
type WeworkStatusReq struct {
	Status int8 `json:"status"`
}

// ===== 商户自助流程（注册/完善资料/找回密码，对齐 epay user/reg|completeinfo|findpwd）=====

// MerchantRegReq 注册入参。verifytype: 0=邮箱 1=手机（account 相应为邮箱/手机）。
// captcha_token/captcha 为自研图形验证码（代替 epay 短信/邮箱 OTP）。
type MerchantRegReq struct {
	VerifyType   int8   `json:"verifytype"`   // 0邮箱 1手机
	Account      string `json:"account" binding:"required"` // 邮箱或手机号
	Password     string `json:"password" binding:"required"`
	Invite       string `json:"invite"`       // 注册授权码（reg_open=2 必填，对应 pay_invitecode）
	Ref          string `json:"ref"`          // 邀请返现推广码（?invite= 解出上级 uid，对齐 epay upid）
	CaptchaToken string `json:"captcha_token"`
	Captcha      string `json:"captcha"`
}

// MerchantRegResp 注册结果。NeedReview=true 表示待审核（pay=2）。
type MerchantRegResp struct {
	UID        uint   `json:"uid"`
	NeedReview bool   `json:"need_review"`
	Msg        string `json:"msg"`
}

// MerchantCompleteReq 完善资料入参（对齐 epay completeinfo）。
type MerchantCompleteReq struct {
	SettleID int    `json:"settle_id"` // 结算方式 1支付宝2微信3QQ4银行卡
	Account  string `json:"account" binding:"required"`
	Username string `json:"username" binding:"required"`
	QQ       string `json:"qq"`
	URL      string `json:"url"`
	Email    string `json:"email"` // 手机注册模式可补邮箱
}

// MerchantFindPwdReq 找回密码入参。Type: email/phone（自选）。
type MerchantFindPwdReq struct {
	Type         string `json:"type"`    // email / phone
	Account      string `json:"account" binding:"required"`
	Password     string `json:"password" binding:"required"`
	CaptchaToken string `json:"captcha_token"`
	Captcha      string `json:"captcha"`
}

// CaptchaResp 图形验证码下发（token + SVG 图）。
type CaptchaResp struct {
	Token string `json:"token"`
	SVG   string `json:"svg"`
}

// ---- 邀请返现（对齐 epay user/invite.php）----

// InviteRewardInfo 返现设置 + 推广链接。
type InviteRewardInfo struct {
	Open      bool   `json:"open"`       // 返现总开关
	Rate      string `json:"rate"`       // 返现比例(%)
	OrderType int    `json:"order_type"` // 0按订单金额/1按手续费/2按利润
	OrderFee  bool   `json:"order_fee"`  // true=允许超过手续费
	Link      string `json:"link"`       // 专属推广链接
	Code      string `json:"code"`       // 邀请码
}

// InviteRewardStat 返现统计。
type InviteRewardStat struct {
	Users           int64  `json:"users"`            // 已邀请人数
	IncomeToday     string `json:"income_today"`     // 今日返现
	IncomeYesterday string `json:"income_yesterday"` // 昨日返现
	IncomeTotal     string `json:"income_total"`     // 累计返现
}

// InvitedUserView 已邀请下级商户。
type InvitedUserView struct {
	UID     uint   `json:"uid"`
	AddTime string `json:"addtime"`
	Status  int8   `json:"status"`
}

// ---- 测试支付（对齐 epay user/test.php）----

// TestPayReq 测试支付入参（金额 + 支付方式）。
type TestPayReq struct {
	Money string `json:"money" binding:"required"` // 支付金额（默认前端给 1）
	Type  string `json:"type" binding:"required"`  // 支付方式 plugin/type 名
}

// TestPayInfoResp 测试支付页信息（开关 + 可选支付方式 + 金额上下限）。
type TestPayInfoResp struct {
	Open     bool              `json:"open"`
	MinMoney string            `json:"min_money"`
	MaxMoney string            `json:"max_money"`
	Types    []PayTypeOption   `json:"types"` // 可选支付方式
}

// PayTypeOption 收银可选支付方式。
type PayTypeOption struct {
	Type     string `json:"type"`
	ShowName string `json:"showname"`
}

// ---- 聚合收款码（对齐 epay user/onecode.php）----

// OnecodeInfo 聚合收款码信息（收款 URL + 收款方名称 + 开关）。
type OnecodeInfo struct {
	Open     bool   `json:"open"`      // 是否可用（全局 onecode 开关 或 商户 open_code）
	PayURL   string `json:"pay_url"`   // 固定收款页 URL
	CodeName string `json:"codename"`  // 收款方名称
}

// OnecodeNameReq 保存收款方名称。
type OnecodeNameReq struct {
	CodeName string `json:"codename"`
}

// PaypageInfo 公开收款页信息（扫码后展示，不含敏感字段）。
type PaypageInfo struct {
	CodeName string          `json:"codename"` // 收款方名称
	SiteName string          `json:"sitename"`
	Types    []PayTypeOption `json:"types"` // 可选支付方式
}

// PaypageSubmitReq 收款页下单（输入金额 + 选支付方式）。
type PaypageSubmitReq struct {
	Merchant string `json:"merchant" binding:"required"` // 加密后的收款商户标识
	Money    string `json:"money" binding:"required"`
	Type     string `json:"type" binding:"required"`
}

// ---- 站内信（我方新增，epay 无此实体）----

// MessageView 站内信对外响应。
type MessageView struct {
	ID      uint   `json:"id"`
	UID     uint   `json:"uid"` // 0=全体广播
	Title   string `json:"title"`
	Content string `json:"content"`
	IsRead  bool   `json:"is_read"`
	Date    string `json:"date"`
}

// MessageSendReq 管理员下发站内信入参。UID=0 表示全体广播。
type MessageSendReq struct {
	UID     uint   `json:"uid"` // 0=全体广播
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// ---- 后台仪表盘（对齐 epay admin/index.php + ajax getcount）----

// AdminDashboard 后台仪表盘聚合。
type AdminDashboard struct {
	Overview    []DashOverviewCard `json:"overview"`     // 概况卡（订单/成功/金额/利润）
	Todo        DashTodo           `json:"todo"`         // 待办计数
	TotalMoney  string             `json:"total_money"`  // 商户余额总额
	SettledSum  string             `json:"settled_sum"`  // 已结算总额
	Merchants   int64              `json:"merchants"`    // 商户总数
	OrdersTotal int64              `json:"orders_total"` // 订单总数
	SuccessRate string             `json:"success_rate"` // 今日成功率(%)
	Trend       DashTrend          `json:"trend"`        // 近7日趋势
	Recent      []DashRecentOrder  `json:"recent"`       // 最近订单
	FeeProfit   DashFeeProfit      `json:"fee_profit"`   // 支付方式手续费利润交叉表（今+近6日）
	Alerts      []string           `json:"alerts"`       // 安全告警（弱密码/默认密码等）
}

// DashFeeProfit 支付方式手续费利润交叉表（对齐 epay index.php profit_paytype，今+近6日 × 各支付方式）。
type DashFeeProfit struct {
	Days     []string           `json:"days"`     // 日期标签（如 07-22）
	PayTypes []string           `json:"paytypes"` // 支付方式显示名（列）
	Income   map[string][]string `json:"income"`  // 支付方式 → 每日收入(实付)数组，与 Days 对齐
	Profit   map[string][]string `json:"profit"`  // 支付方式 → 每日利润数组，与 Days 对齐
}

// DashOverviewCard 概况卡：今日/昨日/累计。
type DashOverviewCard struct {
	Label      string `json:"label"`
	Today      string `json:"today"`
	Yesterday  string `json:"yesterday"`
	TotalLabel string `json:"total_label"`
	Total      string `json:"total"`
}

// DashTodo 待办计数。
type DashTodo struct {
	PendingSettle int64 `json:"pending_settle"` // 待结算
	PendingDomain int64 `json:"pending_domain"` // 待审域名
	PendingProfit int64 `json:"pending_profit"` // 待分账
	UnpaidOrders  int64 `json:"unpaid_orders"`  // 今日未付单
}

// DashTrend 近 7 日趋势（订单量 + 交易额）。
type DashTrend struct {
	Labels  []string `json:"labels"`
	Orders  []int64  `json:"orders"`  // 每日已支付订单数
	Amounts []string `json:"amounts"` // 每日已支付交易额
}

// DashRecentOrder 仪表盘实时订单行。
type DashRecentOrder struct {
	TradeNo  string `json:"trade_no"`
	UID      uint   `json:"uid"`
	TypeShow string `json:"typeshowname"`
	Money    string `json:"money"`
	Status   int8   `json:"status"`
	Time     string `json:"time"`
}

// ---- 网站公告（对齐 epay pre_anounce）----

// ---- 快捷登录 OAuth（对齐 epay connect/wxlogin/oauth）----

// OAuthResult 快捷登录结果：已绑定则返回 token+info；未绑定返回 need_bind + openid。
type OAuthResult struct {
	Token    string        `json:"token,omitempty"`
	Info     *MerchantInfo `json:"info,omitempty"`
	NeedBind bool          `json:"need_bind"`
	Provider string        `json:"provider,omitempty"`
	OpenID   string        `json:"openid,omitempty"` // 未绑定时回传，前端绑定时带回
}

// OAuthBindReq 未绑定用户输入商户账号+密码绑定 openid。
type OAuthBindReq struct {
	Provider string `json:"provider" binding:"required"`
	OpenID   string `json:"openid" binding:"required"`
	Account  string `json:"account" binding:"required"`
	Password string `json:"password"`
	Type     int8   `json:"type"` // 复用 MerchantLoginReq 语义 0密钥/1密码
}

// AdminAnnounceView 公告对外响应（后台管理，含 sort/status/addtime）。
type AdminAnnounceView struct {
	ID      uint   `json:"id"`
	Content string `json:"content"`
	Color   string `json:"color"`
	Sort    int    `json:"sort"`
	Status  int8   `json:"status"`
	AddTime string `json:"addtime"`
}

// AnnounceSaveReq 新增/编辑公告入参。
type AnnounceSaveReq struct {
	Content string `json:"content" binding:"required"`
	Color   string `json:"color"`
	Sort    int    `json:"sort"`
	Status  *int8  `json:"status"`
}
