// seedorders 造几条测试订单，覆盖不同支付状态，方便前端联调看数据。
// 用法：go run ./cmd/seedorders -config ./configs
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

func decp(s string) *decimal.Decimal {
	d := dec(s)
	return &d
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
	end1 := now.Add(-30 * time.Minute)
	end3 := now.Add(-2 * time.Hour)

	orders := []model.Order{
		{
			TradeNo: "20260717" + "0001", OutTradeNo: "M1001", APITradeNo: "2026071722001",
			UID: 1, Domain: "shop-a.com", Name: "VIP会员月卡",
			Money: dec("30.00"), RealMoney: decp("30.00"), GetMoney: dec("29.40"), ProfitMoney: dec("0.60"),
			Type: 1, TypeName: "alipay", TypeShow: "支付宝", Channel: 1, Plugin: "alipay",
			IP: "112.10.20.30", Buyer: "158****6688",
			AddTime: now.Add(-35 * time.Minute), EndTime: &end1, Status: 1, Settle: 0, Combine: 0,
		},
		{
			TradeNo: "20260717" + "0002", OutTradeNo: "M1002", APITradeNo: "",
			UID: 1, Domain: "shop-a.com", Name: "游戏充值100元",
			Money: dec("100.00"), RealMoney: nil, GetMoney: dec("0"), ProfitMoney: dec("0"),
			Type: 2, TypeName: "wxpay", TypeShow: "微信支付", Channel: 2, Plugin: "wxpay",
			IP: "112.10.20.31", Buyer: "",
			AddTime: now.Add(-10 * time.Minute), EndTime: nil, Status: 0, Settle: 0, Combine: 0,
		},
		{
			TradeNo: "20260717" + "0003", OutTradeNo: "M1003", APITradeNo: "2026071722003",
			UID: 2, Domain: "shop-b.net", Name: "实物商品-蓝牙耳机",
			Money: dec("199.00"), RealMoney: decp("199.00"), GetMoney: dec("195.02"),
			RefundMoney: dec("199.00"), ProfitMoney: dec("3.98"),
			Type: 1, TypeName: "alipay", TypeShow: "支付宝", Channel: 1, Plugin: "alipay",
			IP: "112.10.20.32", Buyer: "139****2233",
			AddTime: now.Add(-3 * time.Hour), EndTime: &end3, Status: 2, Settle: 0, Combine: 0,
		},
	}

	created := 0
	for _, o := range orders {
		var exist model.Order
		if db.Where("trade_no = ?", o.TradeNo).First(&exist).Error == nil {
			continue // 已存在跳过，可重复执行
		}
		if err := db.Create(&o).Error; err != nil {
			log.Printf("创建订单 %s 失败: %v", o.TradeNo, err)
			continue
		}
		created++
	}
	log.Printf("测试订单播种完成，新建 %d 条（已支付/未支付/已退款各一）", created)
}
