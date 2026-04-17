package router

import (
	"log"
	"net/http"

	"fifu.fun/fifu-gateway/handlers"
	"fifu.fun/fifu-gateway/middleware"
	"fifu.fun/fifu-gateway/utils"
	"github.com/gin-gonic/gin"

	"github.com/go-dev-frame/sponge/pkg/gin/proxy"
)

// Init 初始化 Gin Web 框架，配置 CORS 和路由
func Init() {
	r := gin.Default()
	r.Use(gin.Recovery())

	p := proxy.New(r)

	err := p.Pass("/api", []string{"http://localhost:5100"})
	if err != nil {
		panic(err)
	}

	// 健康检查
	r.GET("/health", func(ctx *gin.Context) { ctx.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	publicKey, privateKey, err := utils.GenerateKeys()
	if err != nil {
		log.Fatal("canot generate keys", err)
	}
	tokenMaker, err := utils.NewPasetoMakerAsymmetric(publicKey, privateKey)
	if err != nil {
		log.Fatal("canot create token maker:", err)
	}

	// 需要认证的路由
	userHandler := handlers.NewUserHandler(tokenMaker)

	// WebAuthn 路由
	r.POST("/webauthn/register/start", handlers.RegisterStart)
	r.POST("/webauthn/register/finish", handlers.RegisterFinish)
	r.POST("/webauthn/login/start", handlers.LoginStart)
	r.POST("/webauthn/login/finish", userHandler.LoginFinish)

	auth := r.Group("/webauthn").Use(middleware.AuthMiddleware(tokenMaker))
	{
		auth.GET("/profile", userHandler.GetProfile)
	}

	log.Println("🚀 服务器启动在 http://localhost:5000")
	r.Run("0.0.0.0:5000")
}
