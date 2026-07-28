package service

import (
	"strings"

	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
	"github.com/epvia/api/pkg/jwtauth"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

// —— 代理权限体系（2026-07-28，可扩展权限点）——
//
// 核心诉求：代理能用哪些功能由平台逐项开通的权限决定——权限开通啥代理就有啥。
// 新增功能只需在 AgentPermissionCatalog 追加一行权限点 key，平台勾选即生效，代理端骨架不改。
// 代理持有的权限存 pay_agent.permissions（逗号分隔 key 串）；权限（能不能用）与
// agent_id 数据隔离（只看自己名下）正交，两者都要有，不可互相替代。

// 权限点 key 常量。
const (
	PermEnroll     = "enroll"     // 进件代理：发起/管理特约商户进件（原 can_enroll）
	PermAcquire    = "acquire"    // 收单代理：收单分润（占位，暂不启用；原 can_acquire）
	PermQuota      = "quota"      // 名额钱包：买名额/用名额
	PermInvite     = "invite"     // 邀请链接：生成/管理进件邀请码与二维码
	PermRefund     = "refund"     // 手动退款：代理自助对自己名下单发起退款
	PermSettlement = "settlement" // 佣金结算：查看佣金/提现
)

// AgentPermission 权限点元数据（供平台勾选、前端数据驱动渲染）。
type AgentPermission struct {
	Key   string `json:"key"`   // 权限点 key
	Name  string `json:"name"`  // 中文名
	Group string `json:"group"` // 分组
	Desc  string `json:"desc"`  // 说明
}

// AgentPermissionCatalog 权限点清单（单一数据源、可扩展）。
// ★ 将来加新功能 = 在此清单追加一行；GET /api/console/agent-permissions 返回它供平台勾选。
var AgentPermissionCatalog = []AgentPermission{
	{Key: PermEnroll, Name: "进件代理", Group: "进件", Desc: "发起/管理特约商户进件"},
	{Key: PermQuota, Name: "名额钱包", Group: "进件", Desc: "购买名额 / 消耗名额"},
	{Key: PermInvite, Name: "邀请链接", Group: "进件", Desc: "生成/管理进件邀请码与二维码"},
	{Key: PermRefund, Name: "手动退款", Group: "进件", Desc: "对自己名下单发起原路退款"},
	{Key: PermSettlement, Name: "佣金结算", Group: "进件", Desc: "查看佣金 / 提现"},
	{Key: PermAcquire, Name: "收单代理", Group: "收单", Desc: "收单分润（占位，暂不启用）"},
}

// validPermKeys 权限点合法 key 集合（校验用）。
var validPermKeys = func() map[string]bool {
	m := make(map[string]bool, len(AgentPermissionCatalog))
	for _, p := range AgentPermissionCatalog {
		m[p.Key] = true
	}
	return m
}()

// NormalizePermissions 清洗权限点集合：去空白、去重、剔除非法 key，输出稳定的逗号分隔串。
func NormalizePermissions(keys []string) string {
	seen := make(map[string]bool)
	var out []string
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" || seen[k] || !validPermKeys[k] {
			continue
		}
		seen[k] = true
		out = append(out, k)
	}
	return strings.Join(out, ",")
}

// ParsePermissions 把 permissions 串解析为 key 集合。
func ParsePermissions(s string) map[string]bool {
	m := make(map[string]bool)
	for _, k := range strings.Split(s, ",") {
		k = strings.TrimSpace(k)
		if k != "" {
			m[k] = true
		}
	}
	return m
}

// HasPermission 判断权限串是否含某权限点。
func HasPermission(permissions, key string) bool {
	return ParsePermissions(permissions)[key]
}

// AgentService 代理业务编排：代理 CRUD、权限、名额钱包/流水、代理端登录。
// 平台端 /console 管所有代理，代理端 /agent 只碰自己——共用本 service，入参强制带 agent_id 隔离。
type AgentService struct {
	repo       *repository.AgentRepo
	enrollRepo *repository.EnrollRepo // 删代理守卫：查名下进件单/邀请足迹，未注入前守卫仅看名额侧
	jm         *jwtauth.Manager       // 代理端登录签发 JWT（scope=agent），未注入前登录不可用
}

func NewAgentService(repo *repository.AgentRepo) *AgentService {
	return &AgentService{repo: repo}
}

// SetEnrollRepo 注入进件仓储（删代理守卫据此判断代理是否有进件/邀请业务足迹）。
func (s *AgentService) SetEnrollRepo(er *repository.EnrollRepo) { s.enrollRepo = er }

// SetJWT 注入 JWT 管理器（代理端 /agent 登录签发 scope=agent 的 token）。
func (s *AgentService) SetJWT(jm *jwtauth.Manager) { s.jm = jm }

// AgentLoginResult 代理端登录结果。
type AgentLoginResult struct {
	Token       string `json:"token"`
	Name        string `json:"name"`
	Account     string `json:"account"`
	Permissions string `json:"permissions"`
}

// Login 代理端登录：校验账号密码（bcrypt）+ 启用状态，签发 scope=agent 的 JWT。
// token 里的 UID 即 agent_id，代理端所有接口据此强制数据隔离（只看/只碰自己名下）。
func (s *AgentService) Login(account, password string) (*AgentLoginResult, error) {
	if s.jm == nil {
		return nil, agErr("代理端登录未就绪")
	}
	account = strings.TrimSpace(account)
	if account == "" || password == "" {
		return nil, agErr("请输入登录账号和密码")
	}
	a, err := s.repo.FindByAccount(account)
	if err != nil {
		return nil, agErr("账号或密码错误")
	}
	if bcrypt.CompareHashAndPassword([]byte(a.Password), []byte(password)) != nil {
		return nil, agErr("账号或密码错误")
	}
	if a.Status != 1 {
		return nil, agErr("该代理账号已停用")
	}
	// role 存权限串，scope=agent；中间件按 scope 拦截，权限门控在 handler 逐项判断。
	token, err := s.jm.Generate(a.ID, a.Name, a.Permissions, "agent")
	if err != nil {
		return nil, err
	}
	return &AgentLoginResult{
		Token: token, Name: a.Name, Account: a.Account, Permissions: a.Permissions,
	}, nil
}

// Profile 取代理自身资料（代理端顶栏/权限渲染用）。
func (s *AgentService) Profile(agentID uint) (*model.Agent, error) {
	a, err := s.repo.FindByID(agentID)
	if err != nil {
		return nil, agErr("代理不存在")
	}
	return a, nil
}

// Repo 暴露底层 repo，供同域其它 service（如登录/进件）复用查询。
func (s *AgentService) Repo() *repository.AgentRepo { return s.repo }

// Permissions 返回权限点清单（供平台勾选）。
func (s *AgentService) Permissions() []AgentPermission { return AgentPermissionCatalog }

// HasPermissionLive 按 agent_id 实时查库判断是否拥有某权限点。
// ★ 用于接口门控：权限存 JWT 是登录时快照，平台改权限后旧 token 不会更新，
//   故门控实时读库，保证平台一开通、代理下次操作即刻生效，无需重登。
func (s *AgentService) HasPermissionLive(agentID uint, key string) bool {
	a, err := s.repo.FindByID(agentID)
	if err != nil {
		return false
	}
	// 停用的代理一律无权（与登录态兜底一致）。
	if a.Status != 1 {
		return false
	}
	return HasPermission(a.Permissions, key)
}

// AgentError 携带业务提示，handler 统一返回错误码。
type AgentError struct{ Msg string }

func (e *AgentError) Error() string { return e.Msg }

func agErr(msg string) *AgentError { return &AgentError{Msg: msg} }

// List 分页查询代理。
func (s *AgentService) List(keyword string, status *int8, page, pageSize int) ([]model.Agent, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return s.repo.List(keyword, status, page, pageSize)
}

// Get 取单个代理。
func (s *AgentService) Get(id uint) (*model.Agent, error) { return s.repo.FindByID(id) }

// Create 新建代理。account 唯一、密码 bcrypt、权限点清洗后落库。
func (s *AgentService) Create(name, account, password, contact, remark string, permKeys []string) (*model.Agent, error) {
	name = strings.TrimSpace(name)
	account = strings.TrimSpace(account)
	if name == "" || account == "" {
		return nil, agErr("代理名称和登录账号不能为空")
	}
	if password == "" {
		return nil, agErr("请设置登录密码")
	}
	if _, err := s.repo.FindByAccount(account); err == nil {
		return nil, agErr("登录账号已存在")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	a := &model.Agent{
		Name: name, Account: account, Password: string(hash),
		Contact: contact, Status: 1, Permissions: NormalizePermissions(permKeys), Remark: remark,
	}
	if err := s.repo.Create(a); err != nil {
		return nil, err
	}
	return a, nil
}

// Update 更新代理资料/权限/状态。password 非空则一并改密。
func (s *AgentService) Update(id uint, name, contact, remark string, status *int8, permKeys []string, password string) error {
	if _, err := s.repo.FindByID(id); err != nil {
		return agErr("代理不存在")
	}
	fields := map[string]any{
		"name":        strings.TrimSpace(name),
		"contact":     contact,
		"remark":      remark,
		"permissions": NormalizePermissions(permKeys),
	}
	if status != nil {
		fields["status"] = *status
	}
	if err := s.repo.Update(id, fields); err != nil {
		return err
	}
	if password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		return s.repo.UpdatePassword(id, string(hash))
	}
	return nil
}

// SetStatus 启用/停用代理。
func (s *AgentService) SetStatus(id uint, status int8) error {
	return s.repo.Update(id, map[string]any{"status": status})
}

// SetPermissions 只更新代理的权限点集合（权限分配独立页用，不动名称/账号/备注）。
// 权限清洗后落库；实时门控 HasPermissionLive 读库即生效，代理无需重登。
func (s *AgentService) SetPermissions(id uint, permKeys []string) error {
	if _, err := s.repo.FindByID(id); err != nil {
		return agErr("代理不存在")
	}
	return s.repo.Update(id, map[string]any{"permissions": NormalizePermissions(permKeys)})
}

// Delete 删除代理（守卫式物理删）。
// 涉及资金/业务留痕的代理禁止物理删——只要有名额流水（买过/用过名额）、进件单或邀请链接，
// 一律拒绝并提示改用「停用」，避免删掉 pay_agent 行后留下钱包/流水/进件单/邀请/结算流水孤儿记录。
// 只有纯净代理（从未售过名额、无任何进件/邀请）才允许物理删，并同事务把随建的空钱包行一并清掉。
func (s *AgentService) Delete(id uint) error {
	if _, err := s.repo.FindByID(id); err != nil {
		return agErr("代理不存在")
	}
	// ① 名额足迹：有流水即买过/用过名额，涉及资金。
	logs, err := s.repo.QuotaLogCount(id)
	if err != nil {
		return err
	}
	if logs > 0 {
		return agErr("该代理有名额资金记录，不能删除；如需停用请改用「停用」")
	}
	// ② 进件/邀请足迹（进件仓储已注入时才查；未注入则仅凭名额侧判断）。
	if s.enrollRepo != nil {
		enrolls, err := s.enrollRepo.CountByAgent(id)
		if err != nil {
			return err
		}
		if enrolls > 0 {
			return agErr("该代理名下有进件单，不能删除；如需停用请改用「停用」")
		}
		invites, err := s.enrollRepo.CountInvitesByAgent(id)
		if err != nil {
			return err
		}
		if invites > 0 {
			return agErr("该代理名下有邀请链接，不能删除；如需停用请改用「停用」")
		}
	}
	// 纯净代理：同事务删代理 + 随建的空钱包行。
	return s.repo.DeleteWithWallet(id)
}

// Wallet 取代理名额钱包。
func (s *AgentService) Wallet(agentID uint) (*model.AgentQuotaWallet, error) {
	return s.repo.Wallet(agentID)
}

// AdjustQuota 平台侧手动调整代理名额（售卖名额/纠错），走流水。
// change>0 增发，change<0 扣减；amount 为对应金额（售卖批发款）。
func (s *AgentService) AdjustQuota(agentID uint, change int, amount decimal.Decimal, remark string) error {
	if change == 0 {
		return agErr("变动数量不能为 0")
	}
	if _, err := s.repo.FindByID(agentID); err != nil {
		return agErr("代理不存在")
	}
	typ := "purchase"
	if change < 0 {
		typ = "consume"
	}
	return s.repo.ChangeQuota(agentID, typ, change, amount, "", remark)
}

// QuotaLogs 名额流水（agentID 为空看全部）。
func (s *AgentService) QuotaLogs(agentID *uint, page, pageSize int) ([]model.AgentQuotaLog, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return s.repo.QuotaLogs(agentID, page, pageSize)
}
