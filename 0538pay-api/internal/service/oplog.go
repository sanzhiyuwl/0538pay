package service

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
)

// 操作日志 scope：商户端 / 管理端 / 代理端（同一张 pay_oplog 表按 scope 区分，一表多用）。
const (
	ScopeMerchant = "merchant"
	ScopeAdmin    = "admin"
	ScopeAgent    = "agent" // 独立代理端 /agent 的写操作（进件/退款/邀请/结算等）
)

// 操作级别。
const (
	OpLevelNormal  = "normal"
	OpLevelWarning = "warning"
	OpLevelDanger  = "danger"
)

// 操作分类（商户端 + 管理端共用一套 key）。
const (
	OpCatAccount  = "account"  // 账户资料/密钥/密码
	OpCatFund     = "fund"     // 资金：提现/退款/代付/充值/结算
	OpCatAuth     = "auth"     // 认证/绑定
	OpCatConfig   = "config"   // 配置/系统设置
	OpCatOrder    = "order"    // 订单管理（管理端）
	OpCatMerchant = "merchant" // 商户管理（管理端）
	OpCatChannel  = "channel"  // 通道/支付接口（管理端）
	OpCatRisk     = "risk"     // 风控/黑名单（管理端）
	OpCatContent  = "content"  // 内容运营（管理端）
	OpCatSystem   = "system"   // 系统/权限/数据（管理端）
	OpCatEnroll   = "enroll"   // 进件/邀请/名额/结算（代理端）
)

// OpActionMeta 动作元数据：中文名 + 分类 + 级别 + 默认对象模板。
type OpActionMeta struct {
	CN       string // 中文动作名
	Category string
	Level    string
	Target   string // 默认对象摘要（无 detail 时用）
}

// exportLimit 导出上限保护（对齐订单导出量级）。
const opExportLimit = 100000

// OpLogService 操作日志：写入（动作映射 + 异步）+ 查询 + 导出。
type OpLogService struct {
	repo *repository.OpLogRepo
}

func NewOpLogService(repo *repository.OpLogRepo) *OpLogService {
	return &OpLogService{repo: repo}
}

// opActions 动作映射表：key = "METHOD 路由模板"（路由注册路径，不含 /api/merchant 前缀）。
// 覆盖商户端全部写操作；新增接口只需在此加一行。中间件据此自动埋点。
// 关键资金操作的 Target 会被 service 层附加的 detail 覆盖为精确摘要。
var opActions = map[string]OpActionMeta{
	// —— 资金类（danger/warning）——
	"POST /order/refund":      {"发起退款", OpCatFund, OpLevelDanger, "发起订单退款"},
	"POST /transfer":          {"发起代付", OpCatFund, OpLevelDanger, "发起代付转账"},
	"POST /deposit/withdraw":  {"保证金提现", OpCatFund, OpLevelWarning, "保证金提现"},
	"POST /deposit/recharge":  {"保证金充值", OpCatFund, OpLevelWarning, "保证金充值"},
	"POST /recharge":          {"余额充值", OpCatFund, OpLevelWarning, "余额充值"},
	"POST /groups/buy":        {"购买会员", OpCatFund, OpLevelWarning, "购买用户组套餐"},
	// —— 账户/密钥/密码（danger）——
	"PUT /password":           {"修改登录密码", OpCatAccount, OpLevelDanger, "修改登录密码"},
	"POST /apikey/reset":      {"重置API密钥", OpCatAccount, OpLevelDanger, "重置通信密钥"},
	"POST /apikey/rsa":        {"生成RSA密钥", OpCatAccount, OpLevelDanger, "生成RSA密钥对"},
	"PUT /apikey/keytype":     {"修改签名模式", OpCatAccount, OpLevelWarning, "修改签名模式"},
	"POST /rebind":            {"换绑手机/邮箱", OpCatAuth, OpLevelDanger, "换绑手机/邮箱"},
	"POST /oauth/unbind":      {"解绑第三方账号", OpCatAuth, OpLevelDanger, "解绑第三方登录"},
	// —— 认证（warning）——
	"POST /cert":              {"提交实名认证", OpCatAuth, OpLevelWarning, "提交实名认证"},
	// —— 资料/配置（normal）——
	"PUT /profile":            {"修改资料", OpCatAccount, OpLevelNormal, "修改商户资料"},
	"POST /complete":          {"完善资料", OpCatAccount, OpLevelNormal, "完善商户资料"},
	"POST /apply":             {"提交申请", OpCatAccount, OpLevelNormal, "提交商户申请"},
	"PUT /msgconfig":          {"消息提醒配置", OpCatConfig, OpLevelNormal, "修改消息提醒配置"},
	"PUT /channelinfo":        {"自定义接口信息", OpCatConfig, OpLevelNormal, "修改自定义接口信息"},
	"POST /onecode/name":      {"收款码命名", OpCatConfig, OpLevelNormal, "修改聚合收款码名称"},
	"POST /domains":           {"新增授权域名", OpCatConfig, OpLevelNormal, "新增授权域名"},
	"DELETE /domains/:id":     {"删除授权域名", OpCatConfig, OpLevelNormal, "删除授权域名"},
	"POST /order/notify":      {"重发通知", OpCatConfig, OpLevelNormal, "重发订单通知"},
	"POST /test":              {"测试支付", OpCatConfig, OpLevelNormal, "发起测试支付"},
}

// LookupAction 按 "METHOD 路由模板" 查商户端动作元数据，未登记返回 (zero,false)。
func LookupAction(key string) (OpActionMeta, bool) {
	m, ok := opActions[key]
	return m, ok
}

// ===== 管理端操作日志（scope=admin，复用同表）=====
// 管理端写操作近百个、且新增频繁，逐条硬编码维护成本高、易漏。改用「资源段 + 动作」派生：
// 路由 /api/admin/<resource>/... + METHOD → 资源中文名/分类 + 动作动词，自动覆盖全部写操作。

// adminResource 资源段元数据：中文名 + 分类。
type adminResource struct {
	CN  string
	Cat string
}

// adminResources 管理端一级资源段 → 中文名 + 分类。新增资源加一行即可。
var adminResources = map[string]adminResource{
	"orders":            {"订单", OpCatOrder},
	"merchants":         {"商户", OpCatMerchant},
	"admins":            {"管理员", OpCatSystem},
	"roles":             {"角色", OpCatSystem},
	"groups":            {"用户组", OpCatMerchant},
	"channels":          {"支付通道", OpCatChannel},
	"subchannels":       {"子通道", OpCatChannel},
	"rolls":             {"通道轮询", OpCatChannel},
	"paytypes":          {"支付方式", OpCatChannel},
	"weixins":           {"微信配置", OpCatChannel},
	"weworks":           {"企业微信", OpCatChannel},
	"plugins":           {"支付插件", OpCatChannel},
	"settles":           {"结算", OpCatFund},
	"settle":            {"结算", OpCatFund},
	"records":           {"资金明细", OpCatFund},
	"transfers":         {"代付", OpCatFund},
	"ps":                {"分账", OpCatFund},
	"blacklist":         {"黑名单", OpCatRisk},
	"domains":           {"授权域名", OpCatMerchant},
	"invitecodes":       {"邀请码", OpCatMerchant},
	"messages":          {"站内信", OpCatContent},
	"announces":         {"网站公告", OpCatContent},
	"articles":          {"文章", OpCatContent},
	"article-categories": {"文章分类", OpCatContent},
	"site":              {"官网内容", OpCatContent},
	"config":            {"系统设置", OpCatConfig},
	"paypwd":            {"支付密码", OpCatConfig},
	"password":          {"登录密码", OpCatConfig},
	"profile":           {"个人资料", OpCatConfig},
	"clean":             {"数据清理", OpCatSystem},
	"upload":            {"上传", OpCatContent},
}

// adminVerb 动作动词派生：按 HTTP 方法 + 末段关键词。返回 (动词, 级别)。
func adminVerb(method, lastSeg string) (string, string) {
	switch {
	case method == "DELETE":
		return "删除", OpLevelDanger
	case lastSeg == "status":
		return "改状态", OpLevelWarning
	case lastSeg == "refund":
		return "退款", OpLevelDanger
	case lastSeg == "recharge":
		return "充值/扣款", OpLevelDanger
	case lastSeg == "resetkey":
		return "重置密钥", OpLevelDanger
	case lastSeg == "freeze":
		return "冻结", OpLevelWarning
	case lastSeg == "unfreeze":
		return "解冻", OpLevelWarning
	case lastSeg == "clear":
		return "清空", OpLevelDanger
	case lastSeg == "batch-delete":
		return "批量删除", OpLevelDanger
	case lastSeg == "generate":
		return "生成", OpLevelNormal
	case lastSeg == "notify":
		return "重发通知", OpLevelNormal
	case lastSeg == "fill":
		return "补单", OpLevelWarning
	case method == "POST":
		return "新增", OpLevelNormal
	case method == "PUT":
		return "修改", OpLevelNormal
	default:
		return "操作", OpLevelNormal
	}
}

// LookupAdminAction 派生管理端动作元数据。key = "METHOD 路由模板"（不含 /api/admin 前缀），
// 如 "DELETE /orders/:trade_no"。未识别资源返回 (zero,false)，不记日志（避免噪音）。
func LookupAdminAction(key string) (OpActionMeta, bool) {
	sp := strings.SplitN(key, " ", 2)
	if len(sp) != 2 {
		return OpActionMeta{}, false
	}
	method, path := sp[0], sp[1]
	segs := strings.Split(strings.Trim(path, "/"), "/")
	if len(segs) == 0 || segs[0] == "" {
		return OpActionMeta{}, false
	}
	res, ok := adminResources[segs[0]]
	if !ok {
		return OpActionMeta{}, false
	}
	verb, level := adminVerb(method, segs[len(segs)-1])
	return OpActionMeta{
		CN:       verb + res.CN,
		Category: res.Cat,
		Level:    level,
		Target:   verb + res.CN,
	}, true
}

// ===== 代理端操作日志（scope=agent，复用同表）=====
// 代理端 /agent 写操作路由固定（进件/邀请两类），条目少，用显式映射即可，语义比派生更准。
// key = "METHOD 路由模板"（不含 /api/agent 前缀）。

var agentActions = map[string]OpActionMeta{
	// —— 进件（建单收开户费/退款为资金敏感，标 danger/warning）——
	"POST /enrolls":               {"建进件单", OpCatEnroll, OpLevelWarning, "创建特约商户进件单"},
	"POST /enrolls/:id/submit":    {"提交微信审核", OpCatEnroll, OpLevelNormal, "提交进件资料至微信"},
	"POST /enrolls/:id/sync":      {"同步微信状态", OpCatEnroll, OpLevelNormal, "查询微信进件状态"},
	"POST /enrolls/:id/refund":    {"进件退款", OpCatFund, OpLevelDanger, "原路退还开户费"},
	"POST /enrolls/:id/material":  {"填写进件资料", OpCatEnroll, OpLevelNormal, "填写/更新进件资料"},
	"POST /enrolls/:id/media":     {"上传进件图片", OpCatEnroll, OpLevelNormal, "上传进件媒体文件"},
	// —— 邀请链接 ——
	"POST /enroll-invites":             {"生成邀请链接", OpCatEnroll, OpLevelNormal, "生成进件邀请码"},
	"PUT /enroll-invites/:id/status":   {"启停邀请链接", OpCatEnroll, OpLevelNormal, "启用/停用邀请链接"},
	"DELETE /enroll-invites/:id":       {"删除邀请链接", OpCatEnroll, OpLevelWarning, "删除邀请链接"},
}

// LookupAgentAction 按 "METHOD 路由模板" 查代理端动作元数据，未登记返回 (zero,false)。
func LookupAgentAction(key string) (OpActionMeta, bool) {
	m, ok := agentActions[key]
	return m, ok
}

// AgentOpActionOptions 代理端动作下拉选项（前端筛选用，按登记表去重）。
func (s *OpLogService) AgentOpActionOptions() []map[string]string {
	seen := map[string]bool{}
	out := make([]map[string]string, 0, len(agentActions))
	for _, m := range agentActions {
		if seen[m.CN] {
			continue
		}
		seen[m.CN] = true
		out = append(out, map[string]string{"value": m.CN, "label": m.CN, "category": m.Category, "level": m.Level})
	}
	return out
}

// AgentList / AgentExportRows 代理端操作日志（平台在控制台查看所有代理的操作）。
func (s *OpLogService) AgentList(q dto.OpLogQuery) ([]dto.OpLogView, int64, error) {
	return s.listScope(ScopeAgent, q)
}
func (s *OpLogService) AgentExportRows(q dto.OpLogQuery) ([]dto.OpLogView, error) {
	return s.exportScope(ScopeAgent, q)
}

// OpActionOptions 商户端动作下拉选项（前端筛选用，按分类聚合）。
func (s *OpLogService) OpActionOptions() []map[string]string {
	seen := map[string]bool{}
	out := make([]map[string]string, 0, len(opActions))
	for _, m := range opActions {
		if seen[m.CN] {
			continue
		}
		seen[m.CN] = true
		out = append(out, map[string]string{"value": m.CN, "label": m.CN, "category": m.Category, "level": m.Level})
	}
	return out
}

// AdminOpActionOptions 管理端动作下拉选项。由资源段 × 动词派生的组合可能很多，
// 这里取「已落库的去重动作」——查库里 scope=admin 实际出现过的 action，避免列一堆从没发生的组合。
func (s *OpLogService) AdminOpActionOptions() []map[string]string {
	actions, _ := s.repo.DistinctActions(ScopeAdmin)
	out := make([]map[string]string, 0, len(actions))
	for _, a := range actions {
		out = append(out, map[string]string{"value": a, "label": a})
	}
	return out
}

// OpContext 一次操作的上下文（中间件采集）。
type OpContext struct {
	Scope    string
	UID      uint
	Operator string
	IP       string
	Meta     OpActionMeta
	Result   string // ok/fail
	Target   string // 精确对象摘要（service 层附加，空则用 Meta.Target）
	Detail   string // JSON 明细（可空）
}

// Record 异步写入一条操作日志（fire-and-forget，吞错不阻断主流程）。
// 照搬 LogService.Record 的 goroutine + _= 吞错写法。
func (s *OpLogService) Record(ctx OpContext) {
	target := ctx.Target
	if target == "" {
		target = ctx.Meta.Target
	}
	result := ctx.Result
	if result == "" {
		result = "ok"
	}
	rec := &model.OperationLog{
		Scope:     ctx.Scope,
		UID:       ctx.UID,
		Operator:  ctx.Operator,
		Action:    ctx.Meta.CN,
		Category:  ctx.Meta.Category,
		Level:     ctx.Meta.Level,
		Target:    target,
		Detail:    ctx.Detail,
		Result:    result,
		IP:        ctx.IP,
		CreatedAt: time.Now(),
	}
	go func() { _ = s.repo.Create(rec) }()
}

// buildFilter 把查询入参转为 repo 过滤条件（含时间范围解析）。scope 区分商户/管理端。
func (s *OpLogService) buildFilter(scope string, q dto.OpLogQuery) repository.OpLogFilter {
	f := repository.OpLogFilter{
		Scope:    scope,
		UID:      q.UID,
		Action:   strings.TrimSpace(q.Action),
		Category: strings.TrimSpace(q.Category),
		Level:    strings.TrimSpace(q.Level),
		Result:   strings.TrimSpace(q.Result),
		Keyword:  strings.TrimSpace(q.Keyword),
	}
	loc := time.Local
	if t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(q.StartTime), loc); err == nil {
		f.Start = &t
	}
	if t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(q.EndTime), loc); err == nil {
		end := t.AddDate(0, 0, 1).Add(-time.Second) // 含结束日 23:59:59
		f.End = &end
	}
	return f
}

// toView 单条 model → DTO。
func toOpLogView(l *model.OperationLog) dto.OpLogView {
	return dto.OpLogView{
		ID: l.ID, Scope: l.Scope, UID: l.UID, Operator: l.Operator,
		Action: l.Action, ActionCN: l.Action,
		Category: l.Category, Level: l.Level, Target: l.Target,
		Detail: l.Detail, Result: l.Result, IP: l.IP,
		Date: l.CreatedAt.Format(timeLayout),
	}
}

// listScope 分页查询指定 scope 的操作日志。
func (s *OpLogService) listScope(scope string, q dto.OpLogQuery) ([]dto.OpLogView, int64, error) {
	q.Normalize()
	list, total, err := s.repo.List(s.buildFilter(scope, q), q.Page, q.PageSize)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.OpLogView, 0, len(list))
	for i := range list {
		views = append(views, toOpLogView(&list[i]))
	}
	return views, total, nil
}

// exportScope 取指定 scope 全量（供流式 CSV 导出，不分页）。
func (s *OpLogService) exportScope(scope string, q dto.OpLogQuery) ([]dto.OpLogView, error) {
	list, err := s.repo.ExportAll(s.buildFilter(scope, q), opExportLimit)
	if err != nil {
		return nil, err
	}
	views := make([]dto.OpLogView, 0, len(list))
	for i := range list {
		views = append(views, toOpLogView(&list[i]))
	}
	return views, nil
}

// List / ExportRows 商户端操作日志（保持原签名，供 merchant 侧 handler 调用）。
func (s *OpLogService) List(q dto.OpLogQuery) ([]dto.OpLogView, int64, error) {
	return s.listScope(ScopeMerchant, q)
}
func (s *OpLogService) ExportRows(q dto.OpLogQuery) ([]dto.OpLogView, error) {
	return s.exportScope(ScopeMerchant, q)
}

// AdminList / AdminExportRows 管理端操作日志。
func (s *OpLogService) AdminList(q dto.OpLogQuery) ([]dto.OpLogView, int64, error) {
	return s.listScope(ScopeAdmin, q)
}
func (s *OpLogService) AdminExportRows(q dto.OpLogQuery) ([]dto.OpLogView, error) {
	return s.exportScope(ScopeAdmin, q)
}

// DecodeDetail 把 detail JSON 解为 map（前端展开用；非法/空返回 nil）。
func DecodeDetail(detail string) map[string]interface{} {
	if strings.TrimSpace(detail) == "" {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(detail), &m); err != nil {
		return nil
	}
	return m
}
