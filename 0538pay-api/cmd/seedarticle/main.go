// seedarticle 造官网文章分类 + 文章默认数据（从前端 mock/articles.ts 移植）。
// 首次运行灌入，已有分类则跳过（幂等）。用法：go run ./cmd/seedarticle -config ./configs
package main

import (
	"flag"
	"log"
	"time"

	"github.com/0538pay/api/internal/config"
	"github.com/0538pay/api/internal/model"
)

func d(s string) time.Time {
	t, _ := time.ParseInLocation("2006-01-02", s, time.Local)
	return t
}

func main() {
	configPath := flag.String("config", "./configs", "配置目录路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	db, err := model.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		log.Fatalf("建表失败: %v", err)
	}

	var catCount int64
	db.Model(&model.ArticleCategory{}).Count(&catCount)
	if catCount > 0 {
		log.Printf("已有 %d 个分类，跳过 seed（幂等）", catCount)
		return
	}

	cats := []model.ArticleCategory{
		{Name: "产品动态", EnName: "Function Update", Cover: "/assets/news-product.jpg", Sort: 1, AddTime: time.Now()},
		{Name: "公司新闻", EnName: "Company Dynamics", Cover: "/assets/news-company.jpg", Sort: 2, AddTime: time.Now()},
		{Name: "行业新闻", EnName: "Industry News", Cover: "/assets/news-industry.jpg", Sort: 3, AddTime: time.Now()},
	}
	for i := range cats {
		if err := db.Create(&cats[i]).Error; err != nil {
			log.Fatalf("创建分类失败: %v", err)
		}
	}
	c1, c2, c3 := cats[0].ID, cats[1].ID, cats[2].ID

	arts := []model.Article{
		{CategoryID: c1, Title: "0538Pay 标准版系统 v3.0 正式发布上线", Summary: "0538Pay 标准版系统 v3.0 正式发布上线！全面重构支付内核，支持多渠道聚合、实时到账、开放 API，让收款更简单高效。", Content: "<h2>v3.0 版本重磅升级</h2><p>本次 v3.0 版本对支付内核进行了全面重构，带来更稳定、更高效的收款体验：</p><ul><li>多渠道聚合：一次对接支持支付宝、微信、QQ钱包、云闪付</li><li>实时到账：买家付款秒级入账，T+1 自动结算</li><li>开放 API：RESTful 接口 + MD5/RSA 双签名</li></ul><p>欢迎各位商户升级体验！</p>", Tags: "标准版系统,聚合支付,开放API,实时到账", IsNew: true, Top: true, Active: 1, Sort: 1, Count: 1280, AddTime: d("2026-07-10")},
		{CategoryID: c1, Title: "陀螺匠系统 v2.4 正式发布，快来升级新版本", Summary: "陀螺匠系统 v2.4 正式发布，新增多项实用功能，优化系统性能，欢迎升级体验。", Content: "<p>陀螺匠系统 v2.4 正式发布，本次更新包含多项功能优化与体验提升。</p>", Tags: "陀螺匠系统,系统升级,性能优化", Active: 1, Sort: 2, Count: 860, AddTime: d("2026-07-10")},
		{CategoryID: c1, Title: "0538Pay Pro 私域会员电商系统 v4.1 正式发布", Summary: "私域会员电商系统 v4.1 正式发布，助力商户打造专属会员运营体系。", Content: "<p>私域会员电商系统 v4.1 正式发布，新增会员分层运营、积分商城等功能。</p>", Tags: "私域电商,会员运营,积分商城", Active: 1, Sort: 3, Count: 720, AddTime: d("2026-06-30")},
		{CategoryID: c2, Title: "0538Pay 主题广场｜以创意赋能，让装修更简单", Summary: "为了帮大家解决装修痛点，让每一位 0538Pay 用户都能轻松拥有专业级商城装修，主题广场正式上线。", Content: "<h2>主题广场正式上线</h2><p>为了帮大家解决装修痛点，让每一位用户都能轻松拥有专业级商城装修，我们推出了主题广场。</p>", Tags: "主题广场,店铺装修,产品发布", IsNew: true, Top: true, Active: 1, Sort: 1, Count: 960, AddTime: d("2026-03-05")},
		{CategoryID: c2, Title: "0538Pay 2026 年春节放假通知", Summary: "2026 年春节放假安排通知，请各位商户提前做好业务安排。", Content: "<p>2026 年春节放假安排通知，具体时间见正文。</p>", Tags: "公司公告,放假通知", Active: 1, Sort: 2, Count: 420, AddTime: d("2026-02-13")},
		{CategoryID: c3, Title: "2026 商业相关性报告：对话式商业成新趋势", Summary: "未来商业竞争正在从流量获取迈向决策效率竞争。从研究结果可以看到，消费者真正需要的是帮助理解与决策的服务。", Content: "<h2>对话式商业成新趋势</h2><p>未来商业竞争正在从流量获取迈向决策效率竞争，对话式商业成为新的增长点。</p>", Tags: "行业观察,对话式商业,趋势报告", IsNew: true, Top: true, Active: 1, Sort: 1, Count: 1120, AddTime: d("2026-07-14")},
		{CategoryID: c3, Title: "新电商法即将发布：对平台加重处罚", Summary: "新电商法即将发布，对平台责任提出更高要求，加重违规处罚力度。", Content: "<p>新电商法即将发布，平台需提前做好合规准备。</p>", Tags: "行业观察,电商法,平台合规", Active: 1, Sort: 2, Count: 780, AddTime: d("2026-07-14")},
	}
	created := 0
	for i := range arts {
		if err := db.Create(&arts[i]).Error; err != nil {
			log.Printf("创建文章失败: %v", err)
			continue
		}
		created++
	}
	log.Printf("seed 完成：%d 个分类，%d 篇文章", len(cats), created)
}
