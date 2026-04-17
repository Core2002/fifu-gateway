package router

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"fifu.fun/fifu-gateway/handlers"
	"fifu.fun/fifu-gateway/middleware"
	"fifu.fun/fifu-gateway/utils"
	"github.com/gin-gonic/gin"
)

// Init 初始化 Gin Web 框架，配置 CORS 和路由
func Init() {
	r := gin.Default()
	r.Use(gin.Recovery())

	// 生成密钥和 tokenMaker
	publicKey, privateKey, err := utils.GenerateKeys()
	if err != nil {
		log.Fatal("cannot generate keys", err)
	}
	tokenMaker, err := utils.NewPasetoMakerAsymmetric(publicKey, privateKey)
	if err != nil {
		log.Fatal("cannot create token maker:", err)
	}

	// 创建需要认证的路由组
	authGroup := r.Group("/api").Use(middleware.AuthMiddleware(tokenMaker))

	// 创建反向代理
	backendURL, _ := url.Parse("http://localhost:5100")
	proxy := httputil.NewSingleHostReverseProxy(backendURL)

	// 确保认证 header 被转发到后端
	proxy.Director = func(req *http.Request) {
		// 去掉 /api 前缀，传递给后端
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/api")
		req.URL.Host = backendURL.Host
		req.URL.Scheme = backendURL.Scheme
		req.Host = backendURL.Host
		req.Header.Add("X-Forwarded-For", req.RemoteAddr)
	}

	authGroup.Any("/*action", func(c *gin.Context) {
		// 手动传递认证 header
		if payload, exists := c.Get(middleware.AuthorizationPayloadKey); exists {
			if p, ok := payload.(*utils.TokenPayload); ok {
				c.Request.Header.Set("X-User-ID", fmt.Sprintf("%d", p.UserID))
				c.Request.Header.Set("X-Username", p.Username)
				c.Request.Header.Set("X-User-Role", p.Role)
			}
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	})

	// 健康检查
	r.GET("/health", func(ctx *gin.Context) { ctx.JSON(http.StatusOK, gin.H{"status": "ok"}) })

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
