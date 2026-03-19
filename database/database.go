package database

import (
	"log"

	"fifu.fun/fifu-gateway/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Init 初始化 SQLite 数据库并自动迁移表结构
func Init() {
	var err error
	DB, err = gorm.Open(sqlite.Open("webauthn.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	err = DB.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatal("failed to migrate database:", err)
	}

	log.Println("✅ 数据库初始化完成")
}
