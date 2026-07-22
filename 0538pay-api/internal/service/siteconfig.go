package service

import (
	"encoding/json"

	"github.com/0538pay/api/internal/repository"
)

// SiteConfigService 官网 CMS 内容读写（自研）。key 白名单校验 + value JSON 合法性校验。
type SiteConfigService struct {
	repo *repository.SiteConfigRepo
}

func NewSiteConfigService(repo *repository.SiteConfigRepo) *SiteConfigService {
	return &SiteConfigService{repo: repo}
}

// SiteConfigError 携带业务提示。
type SiteConfigError struct{ Msg string }

func (e *SiteConfigError) Error() string { return e.Msg }

// allowedKeys 官网 CMS 的三份文档 key（白名单，防越权写任意键）。
var allowedKeys = map[string]bool{
	"content":  true, // 首页营销板块 DIY
	"docs":     true, // 开发者文档
	"settings": true, // 网站设置外壳 + SEO
	"articles": true, // 文章资讯（分类 + 文章，对齐 epay article.php）
}

// Get 读取某份 CMS 文档（官网公开读）。key 非法或无记录返回空串（前端回退默认）。
func (s *SiteConfigService) Get(key string) (string, error) {
	if !allowedKeys[key] {
		return "", &SiteConfigError{Msg: "未知的配置项"}
	}
	return s.repo.Get(key)
}

// Set 保存某份 CMS 文档（后台鉴权写）。校验 key 白名单 + value 为合法 JSON。
func (s *SiteConfigService) Set(key, value string) error {
	if !allowedKeys[key] {
		return &SiteConfigError{Msg: "未知的配置项"}
	}
	if value == "" {
		return &SiteConfigError{Msg: "内容不能为空"}
	}
	// 校验是合法 JSON（前端存整份对象），避免存入脏数据
	var js json.RawMessage
	if err := json.Unmarshal([]byte(value), &js); err != nil {
		return &SiteConfigError{Msg: "内容不是合法的 JSON"}
	}
	return s.repo.Set(key, value)
}
