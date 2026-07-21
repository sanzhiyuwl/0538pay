// seedmerchants 造几条测试商户，覆盖不同状态，方便前端联调。
// 用法：go run ./cmd/seedmerchants -config ./configs
package main

import (
	"flag"
	"log"
	"time"

	"github.com/0538pay/api/internal/config"
	"github.com/0538pay/api/internal/model"
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

	now := time.Now()
	ge := now.AddDate(1, 0, 0) // VIP 组一年后到期

	merchants := []model.Merchant{
		{
			UID: 1, GID: 2, GroupEnd: &ge, Money: dec("1580.60"), SettleID: 1,
			AppKey: "testkey_uid1_abcdef", KeyType: 0,
			Account: "13800001111", Username: "张伟", QQ: "10001", Phone: "138****1111",
			Email: "zhang@example.com", URL: "shop-a.com", AddTime: now.AddDate(0, -3, 0),
			Status: 1, Cert: 1, Pay: 1, Settle: 1, UpID: 0, Mode: 0, Deposit: dec("500.00"),
		},
		{
			UID: 2, GID: 1, Money: dec("326.00"), SettleID: 2,
			Account: "wx_li_ming", Username: "李明", QQ: "10002", Phone: "139****2222",
			Email: "li@example.com", URL: "shop-b.net", AddTime: now.AddDate(0, -1, 0),
			Status: 1, Cert: 0, Pay: 1, Settle: 0, UpID: 1, Mode: 0, Deposit: dec("0.00"),
		},
		{
			UID: 3, GID: 0, Money: dec("0.00"), SettleID: 1,
			Account: "", Username: "", QQ: "", Phone: "137****3333",
			Email: "new@example.com", URL: "shop-c.cn", AddTime: now.AddDate(0, 0, -2),
			Status: 2, Cert: 0, Pay: 2, Settle: 0, UpID: 0, Mode: 0, Deposit: dec("0.00"),
		},
	}

	created, patched := 0, 0
	for _, m := range merchants {
		var exist model.Merchant
		if db.Where("uid = ?", m.UID).First(&exist).Error == nil {
			// 已存在：回填测试密钥（历史数据可能没有 app_key），其余字段保持不动。
			if m.AppKey != "" && exist.AppKey != m.AppKey {
				db.Model(&exist).Updates(map[string]interface{}{"app_key": m.AppKey, "keytype": m.KeyType})
				patched++
			}
			continue
		}
		if err := db.Create(&m).Error; err != nil {
			log.Printf("创建商户 %d 失败: %v", m.UID, err)
			continue
		}
		created++
	}
	log.Printf("测试商户播种完成，新建 %d 条，回填密钥 %d 条", created, patched)
}
