package handlers

import (
	"net/http"

	"fifu.fun/fifu-gateway/middleware"
	"fifu.fun/fifu-gateway/utils"
	"github.com/gin-gonic/gin"
)

func (h *UserHandler) GetProfile(ctx *gin.Context) {
	payload, exists := ctx.Get(middleware.AuthorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	tokenPayload := payload.(*utils.TokenPayload)
	ctx.JSON(http.StatusOK, gin.H{
		"user": UserResponse{
			ID:       tokenPayload.UserID,
			Username: tokenPayload.Username,
			Role:     tokenPayload.Role,
		},
		"issued_at":  tokenPayload.IssuedAt,
		"expired_at": tokenPayload.ExpiredAt,
	})
}
