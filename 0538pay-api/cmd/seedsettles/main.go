// seedsettles 造测试结算数据：批次(pay_batch) + 明细(pay_settle)，覆盖不同方式/状态/自动手动，
// 对齐前端 mock/settle.ts 结构，方便 Settle.vue 切真接口后联调。
// 注意：seed 直接落库，不走扣款逻辑（造历史数据用，不影响商户余额）。
// 用法：go run ./cmd/seedsettles -config ./configs
package main

import (
	"flag"
	"fmt"
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

	now := time.Now()

	// 3 个批次：已完成 / 部分完成 / 处理中
	batches := []model.SettleBatch{
		{Batch: "B20260719020000", AllMoney: dec("0"), Count: 0, Time: now.AddDate(0, 0, -2).Truncate(time.Hour), Status: 1},
		{Batch: "B20260720020000", AllMoney: dec("0"), Count: 0, Time: now.AddDate(0, 0, -1).Truncate(time.Hour), Status: 2},
		{Batch: "B20260721020000", AllMoney: dec("0"), Count: 0, Time: now.Truncate(time.Hour), Status: 0},
	}

	// 测试商户与结算账号（对齐 seedmerchants 的 uid，附几个额外 uid 撑满列表）
	type mrow struct {
		uid      uint
		typ      int8
		account  string
		username string
	}
	mrows := []mrow{
		{1000, 1, "13800001111", "张伟"},
		{1001, 2, "wx_li_ming", "李明"},
		{1005, 4, "6222 **** **** 8888", "王芳"},
		{1008, 1, "13712345678", "刘洋"},
		{1012, 3, "100200300", "陈静"},
		{1003, 5, "ORG0088", "赵磊"},
	}

	// 造 24 条明细，均匀挂到 3 个批次
	records := make([]model.SettleRecord, 0, 24)
	for i := 0; i < 24; i++ {
		m := mrows[i%len(mrows)]
		bi := i % len(batches)
		b := batches[bi]
		// 批次0=已完成→全部 status=1；批次1=部分完成→含失败；批次2=处理中→正在结算
		var status int8
		switch bi {
		case 0:
			status = 1
		case 1:
			status = int8(1)
			if i%4 == 1 {
				status = 3 // 结算失败
			}
		default:
			status = 2
		}
		moneyNum := float64((i*137)%4600) + 88.0
		money := dec(fmt.Sprintf("%.2f", moneyNum))
		fee := money.Mul(dec("0.5")).Div(dec("100")).Round(2)
		if fee.LessThan(dec("0.1")) {
			fee = dec("0.1")
		}
		if fee.GreaterThan(dec("20")) {
			fee = dec("20")
		}
		realmoney := money.Sub(fee)
		auto := int8(1)
		if i%5 == 4 {
			auto = 0 // 部分标手动
		}
		var endTime *time.Time
		if status == 1 {
			t := b.Time.Add(6 * time.Hour)
			endTime = &t
		}
		result := ""
		if status == 3 {
			result = "收款账号有误，转账被退回"
		}
		records = append(records, model.SettleRecord{
			UID: m.uid, Batch: b.Batch, Auto: auto, Type: m.typ,
			Account: m.account, Username: m.username,
			Money: money, RealMoney: realmoney,
			AddTime: b.Time, EndTime: endTime, Status: status, Result: result,
		})
	}

	// 另造 3 条无批次的待结算（模拟手动申请/待收批），供"生成批次"功能测试
	for i := 0; i < 3; i++ {
		m := mrows[i]
		money := dec(fmt.Sprintf("%d.00", 200+i*150))
		fee := money.Mul(dec("0.5")).Div(dec("100")).Round(2)
		if fee.LessThan(dec("0.1")) {
			fee = dec("0.1")
		}
		records = append(records, model.SettleRecord{
			UID: m.uid, Batch: "", Auto: 0, Type: m.typ,
			Account: m.account, Username: m.username,
			Money: money, RealMoney: money.Sub(fee),
			AddTime: now, Status: 0,
		})
	}

	// 幂等：批次已存在则整体跳过（seed 可重复执行）
	var existBatch model.SettleBatch
	if db.Where("batch = ?", batches[0].Batch).First(&existBatch).Error == nil {
		log.Printf("结算测试数据已存在（批次 %s），跳过播种", batches[0].Batch)
		return
	}

	// 回填批次总额/条数
	for bi := range batches {
		sum := decimal.Zero
		cnt := 0
		for _, r := range records {
			if r.Batch == batches[bi].Batch {
				sum = sum.Add(r.RealMoney)
				cnt++
			}
		}
		batches[bi].AllMoney = sum
		batches[bi].Count = cnt
	}

	for _, b := range batches {
		if err := db.Create(&b).Error; err != nil {
			log.Printf("创建批次 %s 失败: %v", b.Batch, err)
		}
	}
	created := 0
	for i := range records {
		if err := db.Create(&records[i]).Error; err != nil {
			log.Printf("创建结算明细失败: %v", err)
			continue
		}
		created++
	}
	log.Printf("结算测试数据播种完成，新建 %d 个批次、%d 条明细", len(batches), created)
}
