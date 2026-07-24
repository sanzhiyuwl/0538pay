// seedprofit 造分账规则(接收方)测试数据，覆盖"绑定商户扣款"与"通道级全局"两类。
// 用于验证下单匹配规则 → 支付成功建分账单 → 提交分账扣商户余额 → 取消/回退解冻 全链路。
// 用法：go run ./cmd/seedprofit -config ./configs
package main

import (
	"flag"
	"log"
	"time"

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

	// 找 mock 通道 id（下单走它）
	var mockCh model.Channel
	if err := db.Where("plugin = ?", "mock").First(&mockCh).Error; err != nil {
		log.Fatalf("未找到 mock 通道，请先跑 seedchannels: %v", err)
	}
	uid1 := uint(1000)

	rules := []model.ProfitReceiver{
		{
			// 绑定商户1 + mock 通道(mode=0) → 分账成功从商户1余额扣款
			Channel: int(mockCh.ID), SubChannel: 0, UID: &uid1,
			Account: "2088001", Name: "推广渠道商A",
			Rate: "10", MinMoney: dec("0"), Status: 1, AddTime: time.Now(),
		},
	}

	created := 0
	for i := range rules {
		var exist model.ProfitReceiver
		q := db.Where("channel = ? AND account = ?", rules[i].Channel, rules[i].Account)
		if rules[i].UID != nil {
			q = q.Where("uid = ?", *rules[i].UID)
		}
		if q.First(&exist).Error == nil {
			continue
		}
		if err := db.Create(&rules[i]).Error; err != nil {
			log.Printf("创建分账规则失败: %v", err)
			continue
		}
		created++
	}
	log.Printf("分账规则播种完成，新建 %d 条（绑定商户1的 mock 通道 10%% 分账规则）", created)
}
