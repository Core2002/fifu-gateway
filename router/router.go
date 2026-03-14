package router

import (
	"log"
	"net/http"

	"fifu.fun/test/handlers"
	"fifu.fun/test/middleware"
	"fifu.fun/test/utils"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Init 初始化 Gin Web 框架，配置 CORS 和路由
func Init() {
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:5000",
			"http://127.0.0.1:5000",
		},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	r.GET("/", func(ctx *gin.Context) {
		ctx.Redirect(302, "/app/")
	})

	r.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

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

	auth := r.Group("/").Use(middleware.AuthMiddleware(tokenMaker))
	{
		auth.GET("/profile", userHandler.GetProfile)
	}

	r.Static("/app", "./public")

	log.Println("🚀 服务器启动在 http://127.0.0.1:5000")
	log.Println("📁 前端访问：http://127.0.0.1:5000/app/")
	r.Run("0.0.0.0:5000")
}
