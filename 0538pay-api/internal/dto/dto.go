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
