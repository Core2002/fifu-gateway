package main

import (
	"log"

	"fifu.fun/fifu-gateway/database"
	"fifu.fun/fifu-gateway/router"
	"fifu.fun/fifu-gateway/webauthn"
)

// main 程序入口函数，依次初始化数据库、WebAuthn 和 Web 服务器
func main() {
	log.Println("=== WebAuthn 服务启动 ===")
	database.Init()
	webauthn.Init()
	router.Init()
}
