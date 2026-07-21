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
	"golang.org/x/crypto/bcrypt"
)

func dec(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

// hashPwd 生成 bcrypt 密码哈希（与商户登录校验一致）。
func hashPwd(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("生成密码哈希失败: %v", err)
	}
	return string(h)
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

	// 测试登录密码统一 merchant888（密码登录用账号=邮箱/手机；密钥登录用商户ID+AppKey）。
	pwd := hashPwd("merchant888")

	merchants := []model.Merchant{
		{
			UID: 1, GID: 2, GroupEnd: &ge, Money: dec("1580.60"), SettleID: 1,
			AppKey: "testkey_uid1_abcdef", KeyType: 0, Password: pwd,
			Account: "13800001111", Username: "张伟", QQ: "10001", Phone: "13800001111",
			Email: "zhang@example.com", URL: "shop-a.com", AddTime: now.AddDate(0, -3, 0),
			Status: 1, Cert: 1, Pay: 1, Settle: 1, UpID: 0, Mode: 0, Deposit: dec("500.00"),
		},
		{
			UID: 2, GID: 1, Money: dec("326.00"), SettleID: 2, Password: pwd,
			AppKey: "testkey_uid2_ghijkl", KeyType: 0,
			Account: "wx_li_ming", Username: "李明", QQ: "10002", Phone: "13900002222",
			Email: "li@example.com", URL: "shop-b.net", AddTime: now.AddDate(0, -1, 0),
			Status: 1, Cert: 0, Pay: 1, Settle: 0, UpID: 1, Mode: 0, Deposit: dec("0.00"),
		},
		{
			UID: 3, GID: 0, Money: dec("0.00"), SettleID: 1, Password: pwd,
			AppKey: "testkey_uid3_mnopqr", KeyType: 0,
			Account: "", Username: "", QQ: "", Phone: "13700003333",
			Email: "new@example.com", URL: "shop-c.cn", AddTime: now.AddDate(0, 0, -2),
			Status: 2, Cert: 0, Pay: 2, Settle: 0, UpID: 0, Mode: 0, Deposit: dec("0.00"),
		},
	}

	created, patched := 0, 0
	for _, m := range merchants {
		var exist model.Merchant
		if db.Where("uid = ?", m.UID).First(&exist).Error == nil {
			// 已存在：回填测试密钥与登录密码（历史数据可能缺失），其余字段保持不动。
			fields := map[string]interface{}{}
			if m.AppKey != "" && exist.AppKey != m.AppKey {
				fields["app_key"] = m.AppKey
				fields["keytype"] = m.KeyType
			}
			if exist.Password == "" {
				fields["password"] = m.Password
			}
			if len(fields) > 0 {
				db.Model(&exist).Updates(fields)
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
	log.Printf("测试商户播种完成，新建 %d 条，回填 %d 条（登录密码统一 merchant888）", created, patched)
}
