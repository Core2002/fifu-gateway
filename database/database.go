package database

import (
	"log"
	"os"

	"fifu.fun/fifu-gateway/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Init 初始化 SQLite 数据库并自动迁移表结构
func Init() {
	// 确保 data 目录存在
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatal("failed to create data directory:", err)
	}

	var err error
	DB, err = gorm.Open(sqlite.Open("data/webauthn.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	err = DB.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatal("failed to migrate database:", err)
	}

	log.Println("✅ 数据库初始化完成")
}
