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
			// Rate = 给商户的到账/分成比例(对齐 epay pay_channel.rate，calcFee 按 getmoney=money*rate/100)。
			// 98.00 表示商户到账 98%、平台抽 2%。CostRate 才是成本费率%(profit 用 realmoney*costrate/100)。
			Name: "模拟支付通道", Type: 0, TypeName: "mock", TypeShow: "模拟支付",
			Plugin: "mock", Mode: 0, Rate: dec("98.00"), CostRate: dec("1.50"),
			DayTop: 0, Status: 1,
		},
		{
			// Plugin 必须是渠道注册表键(channel.Register 的 Key)，否则 channel.Get 命中不了、
			// 收银台可选支付方式恒空。支付宝当面付扫码 = alipayf2f。
			Name: "支付宝官方直连", Type: 1, TypeName: "alipay", TypeShow: "支付宝",
			Plugin: "alipayf2f", Mode: 0, Rate: dec("99.62"), CostRate: dec("0.23"),
			DayTop: 0, PayMin: "0.01", PayMax: "50000.00", Status: 1,
		},
		{
			// 微信 Native 扫码 = wxnative（注册表键）。
			Name: "微信服务商A", Type: 2, TypeName: "wxpay", TypeShow: "微信支付",
			Plugin: "wxnative", Mode: 0, Rate: dec("99.40"), CostRate: dec("0.38"),
			DayTop: 100000, Status: 1,
		},
		{
			// QQ钱包无独立插件，示例走易支付聚合(epay)通道，保持禁用。
			Name: "QQ钱包官方", Type: 3, TypeName: "qqpay", TypeShow: "QQ钱包",
			Plugin: "epay", Mode: 1, Rate: dec("99.00"), CostRate: dec("0.60"),
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
