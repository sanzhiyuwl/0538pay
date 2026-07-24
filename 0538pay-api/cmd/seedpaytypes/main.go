// seedpaytypes 造系统自带支付方式(对齐 epay pre_type 初始数据 1-6)。
// 用法：go run ./cmd/seedpaytypes -config ./configs
package main

import (
	"flag"
	"log"

	"github.com/epvia/api/internal/config"
	"github.com/epvia/api/internal/model"
)

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

	types := []model.PayType{
		{Name: "alipay", ShowName: "支付宝", Device: 0, Status: 1},
		{Name: "wxpay", ShowName: "微信支付", Device: 0, Status: 1},
		{Name: "qqpay", ShowName: "QQ钱包", Device: 0, Status: 1},
		{Name: "bank", ShowName: "云闪付", Device: 0, Status: 0},
		{Name: "jdpay", ShowName: "京东支付", Device: 2, Status: 0},
		{Name: "usdt", ShowName: "USDT", Device: 0, Status: 0},
	}
	created := 0
	for i := range types {
		var exist model.PayType
		if db.Where("name = ? AND device = ?", types[i].Name, types[i].Device).First(&exist).Error == nil {
			continue
		}
		if err := db.Create(&types[i]).Error; err != nil {
			log.Printf("创建支付方式 %s 失败: %v", types[i].Name, err)
			continue
		}
		created++
	}
	log.Printf("支付方式播种完成，新建 %d 个", created)
}
