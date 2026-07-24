// seedgroups 造可购买会员套餐（对齐前端 mock/merchant/groupbuy.ts）。
// 用法：go run ./cmd/seedgroups -config ./configs
package main

import (
	"flag"
	"log"

	"github.com/epvia/api/internal/config"
	"github.com/epvia/api/internal/model"
	"github.com/shopspring/decimal"
)

func dec(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
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

	groups := []model.Group{
		{GID: 1, Name: "普通商户", IsBuy: 0, Price: dec("0"), Expire: 0, Sort: 0, Info: ""},
		{GID: 2, Name: "白银会员", IsBuy: 1, Price: dec("30"), Expire: 1, Sort: 1,
			Info: `[{"label":"支付宝","rate":"0.35"},{"label":"微信","rate":"0.35"},{"label":"QQ钱包","rate":"0.55"}]`},
		{GID: 3, Name: "黄金会员", IsBuy: 1, Price: dec("88"), Expire: 1, Sort: 2,
			Info: `[{"label":"支付宝","rate":"0.30"},{"label":"微信","rate":"0.30"},{"label":"QQ钱包","rate":"0.50"}]`},
		{GID: 4, Name: "钻石会员", IsBuy: 1, Price: dec("1288"), Expire: 0, Sort: 3,
			Info: `[{"label":"支付宝","rate":"0.28"},{"label":"微信","rate":"0.28"},{"label":"QQ钱包","rate":"0.45"}]`},
	}

	created := 0
	for i := range groups {
		var exist model.Group
		if db.Where("gid = ?", groups[i].GID).First(&exist).Error == nil {
			continue
		}
		if err := db.Create(&groups[i]).Error; err != nil {
			log.Printf("创建用户组 %s 失败: %v", groups[i].Name, err)
			continue
		}
		created++
	}
	log.Printf("会员套餐播种完成，新建 %d 个（白银/黄金/钻石可购买）", created)
}
