package router

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"fifu.fun/fifu-gateway/handlers"
	"fifu.fun/fifu-gateway/middleware"
	"fifu.fun/fifu-gateway/utils"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

//go:embed public/*
var staticFiles embed.FS

// Init 初始化 Gin Web 框架，配置 CORS 和路由
func Init() {
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:5200",
			"https://cat.fifu.fun:5200",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	r.GET("/", func(ctx *gin.Context) {
		ctx.Redirect(302, "/app/")
	})

	// 静态文件服务 - 使用嵌入的文件系统
	staticSubFS, _ := fs.Sub(staticFiles, "public")
	// 注册路由，传递嵌入的静态文件
	r.StaticFS("/app", http.FS(staticSubFS))

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

	admin := r.Group("/").
		Use(middleware.AuthMiddleware(tokenMaker)).
		Use(middleware.RoleMiddleware("admin"))
	{
		admin.GET("/admin", func(context *gin.Context) {
			context.JSON(http.StatusOK, gin.H{"message": "admin"})
		})
	}

	// 业务服务代理路由
	targetURL, _ := url.Parse("http://localhost:5100")
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware(tokenMaker)).
		Use(middleware.RoleMiddleware("admin"))
	api.Any("/*path", func(ctx *gin.Context) {
		// 从上下文获取用户信息并注入请求头
		if payload, exists := ctx.Get("authorization_payload"); exists {
			if p, ok := payload.(*utils.TokenPayload); ok {
				ctx.Request.Header.Set("X-User-ID", fmt.Sprintf("%d", p.UserID))
				ctx.Request.Header.Set("X-Username", p.Username)
				ctx.Request.Header.Set("X-User-Role", p.Role)
			}
		}

		// 修改请求路径
		ctx.Request.URL.Path = strings.TrimPrefix(ctx.Request.URL.Path, "/api")
		if ctx.Request.URL.Path == "" {
			ctx.Request.URL.Path = "/"
		}

		proxy.ServeHTTP(ctx.Writer, ctx.Request)
	})

	log.Println("🚀 服务器启动在 http://localhost:5000")
	r.Run("0.0.0.0:5000")
}
