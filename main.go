package main

import (
	"log"

	"fifu.fun/test/database"
	"fifu.fun/test/router"
	"fifu.fun/test/webauthn"
)

// main 程序入口函数，依次初始化数据库、WebAuthn 和 Web 服务器
func main() {
	log.Println("=== WebAuthn 服务启动 ===")
	database.Init()
	webauthn.Init()
	router.Init()
}
