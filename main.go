package main

import (
	"fifu.fun/fifu-gateway/database"
	"fifu.fun/fifu-gateway/handlers"
	"fifu.fun/fifu-gateway/router"
	"fifu.fun/fifu-gateway/webauthn"
	"github.com/joho/godotenv"
	"log"
	"os"
	"time"
)

// main 程序入口函数，依次初始化数据库、WebAuthn 和 Web 服务器
func main() {
	envFile := ".env.development"
	if os.Getenv("APP_ENV") == "production" {
		envFile = ".env.production"
	}

	err := godotenv.Load(envFile)
	if err != nil {
		log.Panic(err)
	}

	log.Println("=== WebAuthn 服务启动 ===")
	database.Init()
	webauthn.Init()

	// 启动会话清理定时任务
	go startSessionCleanup()

	router.Init()
}

// startSessionCleanup 启动会话清理定时任务
func startSessionCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		handlers.CleanupExpiredSessions()
	}
}
