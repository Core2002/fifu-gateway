package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"fifu.fun/test/database"
	"fifu.fun/test/models"
	wa "fifu.fun/test/webauthn"
	"github.com/gin-gonic/gin"
)

var (
	sessions     = map[string]*wa.SessionData{}
	sessionMutex sync.RWMutex
)

// StartRequest 注册/登录请求结构体
type StartRequest struct {
	Username string `json:"username"`
}

// RegisterStart 处理注册请求的开始阶段，生成注册挑战
func RegisterStart(ctx *gin.Context) {
	var req StartRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	if req.Username == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "用户名不能为空"})
		return
	}

	log.Printf("📝 注册开始 - 用户名：%s", req.Username)

	// 检查用户是否已存在
	var existingUser models.User
	result := database.DB.Where("username = ?", req.Username).First(&existingUser)
	if result.Error == nil {
		ctx.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}

	// 使用用户名创建临时用户对象用于生成挑战
	user := models.User{
		Username: req.Username,
	}

	creation, session, err := wa.WA.BeginRegistration(&user)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "生成注册挑战失败：" + err.Error()})
		return
	}

	// 保存会话和用户信息
	sessionMutex.Lock()
	sessions[req.Username] = session
	sessionMutex.Unlock()

	response := wa.ConvertCredentialCreation(creation, &user)

	ctx.JSON(http.StatusOK, response)
}

// RegisterFinish 处理注册完成阶段，验证并保存新生成的凭证
func RegisterFinish(ctx *gin.Context) {
	var req struct {
		Username string `json:"username"`
	}
	bodyBytes, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败：" + err.Error()})
		return
	}

	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据：" + err.Error()})
		return
	}

	if req.Username == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "用户名不能为空"})
		return
	}

	log.Printf("📝 注册完成 - 用户名：%s", req.Username)

	sessionMutex.Lock()
	session, exists := sessions[req.Username]
	delete(sessions, req.Username)
	sessionMutex.Unlock()

	if !exists {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "注册会话已过期"})
		return
	}

	// 使用用户名创建临时用户对象进行验证
	tempUser := &models.User{
		Username: req.Username,
	}

	credentialParsed, err := wa.ParseCredentialCreation(io.NopCloser(bytes.NewReader(bodyBytes)))
	if err != nil {
		log.Printf("❌ 解析凭证创建响应失败：%v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "解析凭证创建响应失败：" + err.Error()})
		return
	}

	// 使用临时用户进行验证
	cred, err := wa.WA.CreateCredential(tempUser, *session, credentialParsed)
	if err != nil {
		log.Printf("❌ 注册验证失败：%+v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "注册验证失败：" + err.Error()})
		return
	}

	// 验证通过后，创建真实用户并保存到数据库
	user := models.User{
		Username:    req.Username,
		Credentials: []wa.Credential{*cred},
	}
	if err := database.DB.Create(&user).Error; err != nil {
		log.Printf("❌ 保存用户失败：%v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "保存用户失败：" + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "registered"})
}

// LoginStart 处理登录请求的开始阶段，生成登录挑战
func LoginStart(ctx *gin.Context) {
	var req StartRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	if req.Username == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "用户名不能为空"})
		return
	}

	var user models.User
	result := database.DB.First(&user, "username = ?", req.Username)
	if result.Error != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "用户未注册"})
		return
	}

	if len(user.Credentials) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "用户未设置通行密钥"})
		return
	}

	assertion, session, err := wa.WA.BeginLogin(&user)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "生成登录挑战失败：" + err.Error()})
		return
	}

	sessionMutex.Lock()
	sessions[req.Username] = session
	sessionMutex.Unlock()

	response := wa.ConvertCredentialAssertion(assertion)

	ctx.JSON(http.StatusOK, response)
}

// LoginFinish 处理登录完成阶段，验证用户凭证并完成登录
func LoginFinish(ctx *gin.Context) {
	bodyBytes, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败：" + err.Error()})
		return
	}

	var req struct {
		Username string `json:"username"`
	}
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据：" + err.Error()})
		return
	}

	if req.Username == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "用户名不能为空"})
		return
	}

	log.Printf("🔐 登录完成 - 用户名：%s", req.Username)

	sessionMutex.Lock()
	session, exists := sessions[req.Username]
	delete(sessions, req.Username)
	sessionMutex.Unlock()

	if !exists {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "登录会话已过期"})
		return
	}

	var user models.User
	result := database.DB.First(&user, "username = ?", req.Username)
	if result.Error != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	assertionParsed, err := wa.ParseCredentialRequest(io.NopCloser(bytes.NewReader(bodyBytes)))
	if err != nil {
		log.Printf("❌ 解析认证响应失败：%v", err)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "登录验证失败：" + err.Error()})
		return
	}

	cred, err := wa.WA.ValidateLogin(&user, *session, assertionParsed)
	if err != nil {
		log.Printf("❌ 登录验证失败：%+v", err)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "登录验证失败：" + err.Error()})
		return
	}

	for i, c := range user.Credentials {
		if string(c.ID) == string(cred.ID) {
			user.Credentials[i] = *cred
			database.DB.Save(&user)
			break
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "login ok"})
}
