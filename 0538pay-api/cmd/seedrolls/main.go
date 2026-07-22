// seedrolls 造通道体系增强的测试数据：轮询组 + 子通道 + 用户组通道分配。
// 依赖 seedchannels（需先有 mock/支付宝/微信通道）与 seedmerchants（商户ID 1）。
// 用法：go run ./cmd/seedrolls -config ./configs
package main

import (
	"encoding/json"
	"flag"
	"log"
	"strconv"
	"time"

	"github.com/0538pay/api/internal/config"
	"github.com/0538pay/api/internal/model"
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

	// 需要至少两个 mock 通道用于轮询/随机验证。取 plugin=mock 的通道；不足则再建一个。
	var mocks []model.Channel
	db.Where("plugin = ?", "mock").Order("id ASC").Find(&mocks)
	if len(mocks) < 2 {
		extra := model.Channel{
			Name: "模拟支付通道B", Type: 0, TypeName: "mock", TypeShow: "模拟支付",
			Plugin: "mock", Mode: 0, Status: 1,
		}
		if err := db.Create(&extra).Error; err != nil {
			log.Fatalf("补建 mock 通道失败: %v", err)
		}
		mocks = append(mocks, extra)
		log.Printf("补建 mock 通道 B (id=%d)", extra.ID)
	}
	c1, c2 := int(mocks[0].ID), int(mocks[1].ID)

	// 1. 轮询组：顺序轮询组（type=0 mock），组内两个 mock 通道。
	var roll model.Roll
	if db.Where("name = ?", "mock顺序轮询组").First(&roll).Error != nil {
		roll = model.Roll{
			Name: "mock顺序轮询组", Type: 0, Kind: 0, Status: 1,
			Info: itoa(c1) + "," + itoa(c2),
		}
		if err := db.Create(&roll).Error; err != nil {
			log.Fatalf("创建轮询组失败: %v", err)
		}
		log.Printf("创建顺序轮询组 id=%d，成员通道 %d,%d", roll.ID, c1, c2)
	}

	// 2. 子通道：商户ID 1 在 mock 主通道下两个子通道（顺序调度验证）。
	now := time.Now()
	for i, name := range []string{"子通道甲", "子通道乙"} {
		var exist model.SubChannel
		if db.Where("uid = ? AND name = ?", 1, name).First(&exist).Error == nil {
			continue
		}
		ut := now.Add(time.Duration(i) * time.Minute) // 错开 usetime，保证顺序稳定
		sc := model.SubChannel{
			Channel: c1, UID: 1, Name: name, Status: 1,
			Info: `{"remark":"` + name + `"}`, AddTime: now, UseTime: &ut,
		}
		if err := db.Create(&sc).Error; err != nil {
			log.Printf("创建子通道 %s 失败: %v", name, err)
			continue
		}
		log.Printf("创建子通道 %s id=%d", name, sc.ID)
	}

	// 3. 用户组通道分配：默认组 gid=0（epay 格式 {typeid:{type,channel,rate}}）。
	//    支付方式 0(mock) 分配为轮询组(roll)，指向上面的顺序轮询组，费率覆盖 2.00%。
	assign := map[string]map[string]string{
		"0": {"type": "roll", "channel": itoa(int(roll.ID)), "rate": "2.00"},
	}
	b, _ := json.Marshal(assign)
	var g0 model.Group
	if db.Where("gid = ?", 0).First(&g0).Error != nil {
		// GORM Create 把零值主键当自增忽略；MySQL 默认也把插入 0 当作取下一个自增值。
		// 开 NO_AUTO_VALUE_ON_ZERO 会话模式后原生插入，强制写 gid=0（对齐 epay 默认组 gid=0）。
		db.Exec("SET SESSION sql_mode=CONCAT(@@sql_mode, ',NO_AUTO_VALUE_ON_ZERO')")
		if err := db.Exec("INSERT INTO pay_group (gid, name, info) VALUES (0, ?, ?)", "默认用户组", string(b)).Error; err != nil {
			log.Fatalf("创建默认组失败: %v", err)
		}
		log.Printf("创建默认组 gid=0 并写入通道分配")
	} else {
		if err := db.Model(&model.Group{}).Where("gid = ?", 0).Update("info", string(b)).Error; err != nil {
			log.Fatalf("更新默认组分配失败: %v", err)
		}
		log.Printf("更新默认组 gid=0 通道分配：mock→轮询组%d 费率2.00%%", roll.ID)
	}

	log.Printf("通道体系测试数据播种完成")
}

func itoa(n int) string { return strconv.Itoa(n) }
