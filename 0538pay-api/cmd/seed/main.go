// seed 初始化首个后台管理员，方便登录联调。
// 用法：go run ./cmd/seed -config ./configs -user admin -pass admin888
package main

import (
	"flag"
	"log"

	"github.com/epvia/api/internal/config"
	"github.com/epvia/api/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	configPath := flag.String("config", "./configs", "配置目录路径")
	user := flag.String("user", "admin", "管理员用户名")
	pass := flag.String("pass", "admin888", "管理员密码")
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

	hash, err := bcrypt.GenerateFromPassword([]byte(*pass), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("生成密码哈希失败: %v", err)
	}

	admin := model.Admin{
		Username: *user,
		Password: string(hash),
		Nickname: "超级管理员",
		Role:     "super",
		Status:   1,
	}
	// 存在则更新密码，不存在则创建
	var existing model.Admin
	if db.Where("username = ?", *user).First(&existing).Error == nil {
		existing.Password = string(hash)
		if err := db.Save(&existing).Error; err != nil {
			log.Fatalf("更新管理员失败: %v", err)
		}
		log.Printf("已更新管理员 %s 的密码", *user)
		return
	}
	if err := db.Create(&admin).Error; err != nil {
		log.Fatalf("创建管理员失败: %v", err)
	}
	log.Printf("已创建管理员 %s / %s", *user, *pass)
}
