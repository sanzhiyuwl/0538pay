package service

import (
	"strings"
	"sync"

	"github.com/0538pay/api/internal/repository"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

// ConfigService 系统配置读写（对齐 epay set.php + pre_config）。
// 键名对齐 epay set.php 表单 name。值一律按字符串存，读时按需转型。
// 内存缓存全量配置，写入后 reload 并通知订阅者（各业务服务刷新其缓存常量）。
type ConfigService struct {
	repo    *repository.ConfigRepo
	mu      sync.RWMutex
	cache   map[string]string
	subs    []func() // 配置变更后的回调（各服务重载常量）
}

func NewConfigService(repo *repository.ConfigRepo) *ConfigService {
	return &ConfigService{repo: repo, cache: map[string]string{}}
}

// ConfigError 携带业务提示。
type ConfigError struct{ Msg string }

func (e *ConfigError) Error() string { return e.Msg }

// configDefaults 各配置项默认值（对齐 epay install.sql seed / set.php 缺省语义）。
// 缓存/DB 无该键时回退到此。仅列出我们当前用到 + 系统设置页需要的键。
var configDefaults = map[string]string{
	// 结算 settle
	"settle_open":     "3",   // 0关/1仅自动/2仅手动/3自动+手动
	"settle_type":     "1",   // 0=D+0全部余额 / 1=D+1前1天
	"settle_money":    "30",  // 最低结算金额
	"settle_rate":     "0.5", // 结算手续费率(%)
	"settle_fee_min":  "0.1", // 手续费封底(元)
	"settle_fee_max":  "20",  // 手续费封顶(元)
	"settle_maxlimit": "5",   // 每日手动申请次数上限(0不限)
	// 代付 transfer
	"transfer_rate":     "",      // 代付手续费率(%)，空则复用 settle_rate
	"transfer_minmoney": "1",     // 单笔最小
	"transfer_maxmoney": "20000", // 单笔最大
	"transfer_maxlimit": "10",    // 同账号每日代付次数上限(0不限)
	// 退款 refund
	"refund_fee_type": "0", // 0平台承担手续费(退回商户扣getmoney) / 1商户承担(全额退扣realmoney)
	// 保证金 / 实名
	"user_deposit_min": "1000", // 保证金最低充值金额
	"cert_money":       "0",    // 实名工本费(0免费)
	"user_deposit":     "0",    // 是否启用保证金门槛(1开0关；对齐 epay user_deposit)
	"user_deposit_day": "0",    // 保证金提取冻结天数(最近N天有成功订单禁提取,0不限；对齐 epay user_deposit_day)
	// 支付 pay
	"pay_maxmoney": "50000",                // 最大支付金额
	"pay_minmoney": "0.01",                 // 最小支付金额
	"blockname":    "博彩|赌博|违禁|毒品|枪支", // 商品屏蔽关键词(|分隔)
	"blockalert":   "温馨提醒该商品禁止出售",     // 屏蔽提示
	"forceqq":      "0", // 强制填联系QQ才能收款(1开0关；对齐 epay forceqq)
	"pay_iplimit":  "0", // 同IP当日成功单数上限(0不限；对齐 epay pay_iplimit)
	"pay_userlimit": "0", // 单买家(openid/buyer)当日下单上限(0不限；对齐 epay pay_userlimit)
	"payfee_lessthan": "0", // 最低手续费触发阈值(手续费低于此值时启用兜底,0关闭；对齐 epay payfee_lessthan)
	"payfee_mincost":  "0", // 最低手续费金额(对齐 epay payfee_mincost)
	"pay_payaddstart":  "0", // 随机增减金额触发起始金额(realmoney≥此值才微调,0关闭；对齐 epay pay_payaddstart)
	"pay_payaddmin":    "0", // 随机增减最小值(元；对齐 epay pay_payaddmin)
	"pay_payaddmax":    "0", // 随机增减最大值(元；对齐 epay pay_payaddmax)
	"notifyordername":  "0", // 异步通知商品名强制为 product (1开/0关；对齐 epay notifyordername)
	"pay_domain_forbid": "0", // 域名白名单全局开关(1开启白名单校验/0不校验；对齐 epay pay_domain_forbid)
	"pageordername":     "0", // 收银台商品名强制为 onlinepay (1开/0关；对齐 epay pageordername)
	"ordername":         "",  // 全局订单名模板([name]/[order]/[outorder]/[qq]/[phone]占位；对齐 epay ordername)
	// 站点 site
	"sitename": "0538pay 聚合支付平台",
	"kfqq":     "",
	// 注册登录 reg
	"reg_open":         "1", // 1开/0关/2仅邀请
	"user_review":      "0", // 注册审核
	"reg_input_settle": "0", // 注册后可不填结算账户
	"reg_pay":          "0", // 注册付费
	"reg_pay_price":    "0", // 注册付费金额
	"captcha_open_login": "0", // 登录验证码
	"captcha_version":  "1", // 极验版本 0=V3/1=V4
	"captcha_id":       "",
	"captcha_key":      "",
	// 计划任务 cron
	"cronkey": "",
	// 管理员支付密码（对齐 epay admin_paypwd，用于转账/结算/API 退款二次校验，默认 123456）。
	// 我方以 bcrypt 哈希存储（比 epay 明文安全）；空值语义=未设置，回退默认 123456。
	"admin_paypwd": "",
	// 系统内部签名密钥（对齐 epay syskey，用于 V1 api.php order 内部签名查单 md5(SYS_KEY.trade_no.SYS_KEY)）
	"syskey": "",
	// 结算/代付打款备注（对齐 epay transfer_desc，用于银行结算导出模板的"用途/备注"列）
	"transfer_desc": "货款结算",
	// IP 获取 / 代理 other
	"ip_type":      "2", // 0 XFF/1 XRealIP/2 RemoteAddr
	"proxy":        "0",
	"proxy_server": "",
	"proxy_port":   "",
	"proxy_user":   "",
	"proxy_pwd":    "",
	"proxy_type":   "http",
	// 快捷登录 oauth（枚举开关默认 "0" 关，避免前端下拉空选）
	"login_qq":     "0",
	"login_alipay": "0",
	"login_wx":     "0",
	// 消息提醒 notice（开关默认 "0"）
	"wxnotice":           "0",
	"mailnotice":         "0",
	"msgconfig_regaudit": "0",
	"msgconfig_apply":    "0",
	"msgconfig_domain":   "0",
	"msgconfig_order":    "0",
	"msgconfig_settle":   "0",
	"msgconfig_balance":  "0",
	// 实名认证 cert（开关默认 "0"）
	"cert_open":     "0",
	"cert_channel":  "0",
	"cert_corpopen": "0",
	"cert_force":    "0",
	// 邮箱短信 mail（枚举默认）
	"mail_cloud": "0",
	"sms_api":    "0",
	// 邀请返现 invite（对齐 epay set.php?mod=user 的邀请设置）
	"invite_open":       "0", // 邀请返现总开关 0关/1开
	"invite_rate":       "",  // 返现比例(%)，空=不返现
	"invite_order_type": "0", // 0按订单金额 / 1按订单手续费 / 2按平台利润
	"invite_order_fee":  "0", // 0返现不超过手续费(封顶) / 1允许超过
	// 测试支付 test（对齐 epay test.php）
	"test_open":    "0", // 测试支付页开关 0关/1开
	"test_pay_uid": "0", // 测试收款商户 uid（0=下到当前商户）
	// 聚合收款码 onecode（对齐 epay onecode.php）
	"onecode": "0", // 聚合收款码全局开关 0关/1开
	// 使用说明（我方做成后台可编辑；epay help.php 是硬编码静态页）
	"help_content": "", // 商户使用说明正文（HTML/富文本），空则前端用内置默认文案
	// 风控自动关停 risk（对齐 epay set.php?mod=risk + cron do=check）
	"auto_check_notify":  "0",  // 连续通知失败自动关停商户支付 0关/1开
	"check_notify_count": "10", // 连续通知失败达该次数则关停(写 pay_risk type=2)
	"auto_check_sucrate": "0",  // 商户成功率自动关停 0关/1开
	"check_sucrate_second": "600", // 统计窗口(秒)
	"check_sucrate_count":  "20",  // 窗口内最少订单数(达到才判定，避免小样本误伤)
	"check_sucrate_value":  "30",  // 成功率低于该值(%)则关停(写 pay_risk type=1)
	"auto_check_channel":    "0",   // 通道/子通道自动关停 0关/1开(对齐 epay auto_check_channel)
	"check_channel_second":  "600", // 通道统计窗口(秒；对齐 epay check_channel_second)
	"check_channel_failcount": "0", // 窗口内连续未支付订单数达此值则关停(0=未开启；对齐 epay check_channel_failcount)
	"check_channel_ids":     "",    // 限定检查的通道ID(逗号分隔,留空=全部；对齐 epay check_channel_ids)
}

// configGroups 各系统设置分组包含的键（白名单，前端按 group 读写）。
var configGroups = map[string][]string{
	"settle": {
		"settle_open", "settle_type", "settle_money", "settle_rate",
		"settle_fee_min", "settle_fee_max", "settle_maxlimit",
	},
	"transfer": {
		"transfer_rate", "transfer_minmoney", "transfer_maxmoney", "transfer_maxlimit",
	},
	"pay": {
		"pay_maxmoney", "pay_minmoney", "blockname", "blockalert", "refund_fee_type",
		"payfee_lessthan", "payfee_mincost",
		"pay_payaddstart", "pay_payaddmin", "pay_payaddmax", "notifyordername",
		"pay_domain_forbid", "pageordername", "ordername",
		"cert_force", "forceqq", "pay_iplimit", "pay_userlimit",
	},
	"deposit": {"user_deposit_min", "cert_money", "user_deposit", "user_deposit_day"},
	"site":    {"sitename", "kfqq", "reg_open"},
	"reg": {
		"reg_open", "user_review", "reg_input_settle", "reg_pay", "reg_pay_price",
		"captcha_open_login", "captcha_version", "captcha_id", "captcha_key",
	},
	"cron":  {"cronkey"},
	"other": {"ip_type", "proxy", "proxy_server", "proxy_port", "proxy_user", "proxy_pwd", "proxy_type"},
	"oauth": {
		"login_qq", "login_qq_appid", "login_qq_appkey",
		"login_alipay", "login_wx", "login_apiurl", "login_appid", "login_appkey",
	},
	"notice": {
		"wxnotice", "wxnotice_tpl_order", "wxnotice_tpl_settle", "wxnotice_tpl_login", "wxnotice_tpl_balance",
		"msgconfig_regaudit", "msgconfig_apply", "msgconfig_domain",
		"mailnotice", "msgconfig_order", "msgconfig_settle", "msgconfig_balance",
	},
	"cert": {
		"cert_open", "cert_channel", "cert_appcode", "cert_qcloudid", "cert_qcloudkey",
		"cert_aliyunid", "cert_aliyunkey", "cert_aliyunsceneid", "cert_corpopen", "cert_appcode2", "cert_force", "cert_money",
	},
	"mail": {
		"mail_cloud", "mail_smtp", "mail_port", "mail_name", "mail_pwd",
		"mail_apiuser", "mail_apikey", "mail_name2", "mail_recv",
		"sms_api", "sms_appid", "sms_appkey", "sms_sign",
		"sms_tpl_reg", "sms_tpl_find", "sms_tpl_edit", "sms_tpl_balance",
	},
	"invite": {
		"invite_open", "invite_rate", "invite_order_type", "invite_order_fee",
	},
	"test": {
		"test_open", "test_pay_uid",
	},
	"onecode": {"onecode"},
	"help":    {"help_content"},
	"risk": {
		"auto_check_notify", "check_notify_count",
		"auto_check_sucrate", "check_sucrate_second", "check_sucrate_count", "check_sucrate_value",
		"auto_check_channel", "check_channel_second", "check_channel_failcount", "check_channel_ids",
	},
}

// Load 从 DB 全量加载配置进内存缓存（启动时调一次）。缺省键补默认值。
func (s *ConfigService) Load() error {
	m, err := s.repo.All()
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.cache = m
	s.mu.Unlock()
	return nil
}

// OnChange 注册配置变更回调（各业务服务把"重载常量"函数挂进来）。
func (s *ConfigService) OnChange(fn func()) { s.subs = append(s.subs, fn) }

// notify 触发所有订阅者重载。
func (s *ConfigService) notify() {
	for _, fn := range s.subs {
		fn()
	}
}

// raw 取原始字符串值：缓存命中返回；否则回退默认；再无返回空串。
func (s *ConfigService) raw(key string) (string, bool) {
	s.mu.RLock()
	v, ok := s.cache[key]
	s.mu.RUnlock()
	if ok {
		return v, true
	}
	if d, ok := configDefaults[key]; ok {
		return d, true
	}
	return "", false
}

// Str 取字符串配置。
func (s *ConfigService) Str(key string) string {
	v, _ := s.raw(key)
	return v
}

// Dec 取金额/费率配置为 decimal。空或非法返回 def。
func (s *ConfigService) Dec(key string, def decimal.Decimal) decimal.Decimal {
	v, _ := s.raw(key)
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	d, err := decimal.NewFromString(v)
	if err != nil {
		return def
	}
	return d
}

// Int 取整数配置。空或非法返回 def。
func (s *ConfigService) Int(key string, def int) int {
	v, _ := s.raw(key)
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	d, err := decimal.NewFromString(v)
	if err != nil {
		return def
	}
	return int(d.IntPart())
}

// Bool 取布尔配置（"1"/"true" 为真）。
func (s *ConfigService) Bool(key string) bool {
	v, _ := s.raw(key)
	v = strings.TrimSpace(v)
	return v == "1" || v == "true"
}

// 平台 RSA 密钥的配置键（对齐 epay $conf['private_key']/$conf['public_key']，用于 V2 mapi 回包签名）。
const (
	keySysRSAPrivate = "sys_rsa_private"
	keySysRSAPublic  = "sys_rsa_public"
)

// PlatformPrivateKey 返回平台 RSA 私钥（V2 mapi 回包签名用）。缓存无则返回空串，
// 由 EnsurePlatformKeys 负责首次生成。
func (s *ConfigService) PlatformPrivateKey() string { return s.Str(keySysRSAPrivate) }

// PlatformPublicKey 返回平台 RSA 公钥（下发给商户验回包签名）。
func (s *ConfigService) PlatformPublicKey() string { return s.Str(keySysRSAPublic) }

// EnsurePlatformKeys 保证平台 RSA 密钥对存在：缓存无则用 gen 生成并持久化。
// gen 返回 (私钥 base64, 公钥 base64)，由调用方传入 sign.GenerateRSAKeyPair 避免服务层反向依赖 pkg/sign。
// 启动时调一次，供 V2 mapi 回包签名与商户获取平台公钥。
func (s *ConfigService) EnsurePlatformKeys(gen func() (string, string, error)) error {
	if strings.TrimSpace(s.Str(keySysRSAPrivate)) != "" && strings.TrimSpace(s.Str(keySysRSAPublic)) != "" {
		return nil
	}
	priv, pub, err := gen()
	if err != nil {
		return err
	}
	if err := s.repo.SetMany(map[string]string{
		keySysRSAPrivate: priv,
		keySysRSAPublic:  pub,
	}); err != nil {
		return err
	}
	return s.Load()
}

// GetGroup 读取某分组的配置项（前端设置页回填）。返回 key→当前值(含默认)。
func (s *ConfigService) GetGroup(group string) (map[string]string, error) {
	keys, ok := configGroups[group]
	if !ok {
		return nil, &ConfigError{Msg: "未知的配置分组"}
	}
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		out[k], _ = s.raw(k)
	}
	return out, nil
}

// SaveGroup 保存某分组的配置（仅接受该分组白名单内的键，防越权写任意键）。
func (s *ConfigService) SaveGroup(group string, kv map[string]string) error {
	keys, ok := configGroups[group]
	if !ok {
		return &ConfigError{Msg: "未知的配置分组"}
	}
	allow := make(map[string]bool, len(keys))
	for _, k := range keys {
		allow[k] = true
	}
	filtered := make(map[string]string)
	for k, v := range kv {
		if allow[k] {
			filtered[k] = strings.TrimSpace(v)
		}
	}
	if len(filtered) == 0 {
		return &ConfigError{Msg: "没有可保存的配置项"}
	}
	if err := s.repo.SetMany(filtered); err != nil {
		return err
	}
	// 重载缓存 + 通知业务服务刷新常量
	if err := s.Load(); err != nil {
		return err
	}
	s.notify()
	return nil
}

// ===== 管理员支付密码（对齐 epay admin_paypwd）=====
//
// 转账/结算/API 退款等资金操作的二次校验用「支付密码」，与登录密码相互独立。
// epay 用 pre_config.admin_paypwd 明文存、默认 123456；我方沿用 config 存储位置，
// 但以 bcrypt 哈希存放，且保留「未设置时回退默认 123456」的兼容语义，避免老数据锁死。
const (
	keyAdminPayPwd    = "admin_paypwd"
	defaultAdminPayPwd = "123456" // 对齐 epay 出厂默认支付密码
)

// PayPwdIsDefault 返回当前支付密码是否仍为出厂默认值 123456（仪表盘安全告警用）。
func (s *ConfigService) PayPwdIsDefault() bool {
	return s.VerifyPayPwd(defaultAdminPayPwd) == nil
}

// VerifyPayPwd 校验支付密码。未设置（空值）时回退比对默认 123456，
// 保证「补齐该功能前就存在的部署」在管理员改密前仍能用默认值通过校验。
// 校验通过返回 nil，否则返回业务错误。
func (s *ConfigService) VerifyPayPwd(pwd string) error {
	stored := strings.TrimSpace(s.Str(keyAdminPayPwd))
	if stored == "" {
		// 尚未设置：回退默认值明文比对
		if pwd == defaultAdminPayPwd {
			return nil
		}
		return &ConfigError{Msg: "支付密码不正确"}
	}
	if bcrypt.CompareHashAndPassword([]byte(stored), []byte(pwd)) != nil {
		return &ConfigError{Msg: "支付密码不正确"}
	}
	return nil
}

// ChangePayPwd 修改管理员支付密码：先校验旧支付密码（未设置时旧密码填默认 123456），
// 新密码两次一致 + 长度≥6，bcrypt 存新哈希。对齐 epay set.php mod=paypwd_n。
func (s *ConfigService) ChangePayPwd(oldPwd, newPwd, newPwd2 string) error {
	if err := s.VerifyPayPwd(oldPwd); err != nil {
		return &ConfigError{Msg: "原支付密码不正确"}
	}
	if len(newPwd) < 6 {
		return &ConfigError{Msg: "新支付密码至少 6 位"}
	}
	if newPwd != newPwd2 {
		return &ConfigError{Msg: "两次输入的新支付密码不一致"}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.SetMany(map[string]string{keyAdminPayPwd: string(hash)}); err != nil {
		return err
	}
	if err := s.Load(); err != nil {
		return err
	}
	s.notify()
	return nil
}
