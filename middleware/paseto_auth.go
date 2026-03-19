package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"fifu.fun/fifu-gateway/utils"
	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeader     = "authorization"
	AuthorizationTypeBearer = "bearer"
	AuthorizationPayloadKey = "authorization_payload"
)

// AuthMiddleware 认证中间件
func AuthMiddleware(tokenMaker *utils.PasetoMaker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizationHeader := ctx.Request.Header.Get(AuthorizationHeader)
		if len(authorizationHeader) == 0 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header is not provided",
			})
			return
		}
		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
			})
			return
		}
		authorizationType := strings.ToLower(fields[0])
		if authorizationType != AuthorizationTypeBearer {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": fmt.Sprintf("unsupported authorization type %s", authorizationType),
			})
			return
		}
		accessToken := fields[1]
		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}
		// 将用户信息存入上下文
		ctx.Set(AuthorizationPayloadKey, payload)
		ctx.Next()
	}
}

// RoleMiddleware 角色权限中间件
func RoleMiddleware(roles ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		payload, exists := ctx.Get(AuthorizationPayloadKey)
		if !exists {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization payload not found",
			})
			return
		}
		tokenPayload, ok := payload.(*utils.TokenPayload)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization payload",
			})
			return
		}
		// 检查角色权限
		hasPermission := false
		for _, role := range roles {
			if tokenPayload.Role == role {
				hasPermission = true
				break
			}
		}
		if !hasPermission {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "permission denied",
			})
		}
		ctx.Next()
	}
}
