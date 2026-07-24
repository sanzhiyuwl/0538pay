// seedchannels 造几条测试支付通道，覆盖不同支付方式/模式/状态，方便前端联调。
// 含一条 plugin=mock 的通道，让阶段A收单链路走真实通道记录(带费率)。
// 用法：go run ./cmd/seedchannels -config ./configs
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

	channels := []model.Channel{
		{
			Name: "模拟支付通道", Type: 0, TypeName: "mock", TypeShow: "模拟支付",
			Plugin: "mock", Mode: 0, Rate: dec("2.00"), CostRate: dec("1.50"),
			DayTop: 0, Status: 1,
		},
		{
			Name: "支付宝官方直连", Type: 1, TypeName: "alipay", TypeShow: "支付宝",
			Plugin: "alipay", Mode: 0, Rate: dec("0.38"), CostRate: dec("0.23"),
			DayTop: 0, PayMin: "0.01", PayMax: "50000.00", Status: 1,
		},
		{
			Name: "微信服务商A", Type: 2, TypeName: "wxpay", TypeShow: "微信支付",
			Plugin: "wxpay", Mode: 0, Rate: dec("0.60"), CostRate: dec("0.38"),
			DayTop: 100000, Status: 1,
		},
		{
			Name: "QQ钱包官方", Type: 3, TypeName: "qqpay", TypeShow: "QQ钱包",
			Plugin: "qqpay", Mode: 1, Rate: dec("1.00"), CostRate: dec("0.60"),
			DayTop: 0, Status: 0,
		},
	}

	created := 0
	for _, c := range channels {
		var exist model.Channel
		if db.Where("plugin = ? AND name = ?", c.Plugin, c.Name).First(&exist).Error == nil {
			continue // 已存在跳过，可重复执行
		}
		if err := db.Create(&c).Error; err != nil {
			log.Printf("创建通道 %s 失败: %v", c.Name, err)
			continue
		}
		created++
	}
	log.Printf("测试通道播种完成，新建 %d 条（含 mock/支付宝/微信/QQ）", created)
}
