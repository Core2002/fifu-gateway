package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/gorm"
)

var (
	db           *gorm.DB
	wa           *webauthn.WebAuthn
	sessions     = map[string]*webauthn.SessionData{}
	sessionMutex sync.RWMutex
)

// 用户结构体，实现 webauthn.Interface 接口
type User struct {
	ID          uint                  `gorm:"primarykey"`
	Username    string                `gorm:"unique"`
	Credentials []webauthn.Credential `gorm:"serializer:json"`
}

// WebAuthnID 返回用户的 ID 字节数组（使用用户名的字节作为唯一标识）
func (u *User) WebAuthnID() []byte {
	return []byte(u.Username)
}

// WebAuthnName 返回用户名
func (u *User) WebAuthnName() string { return u.Username }

// WebAuthnDisplayName 返回用户显示名称
func (u *User) WebAuthnDisplayName() string { return u.Username }

// WebAuthnIcon 返回用户图标 URL（未使用）
func (u *User) WebAuthnIcon() string { return "" }

// WebAuthnCredentials 返回用户的凭证列表
func (u *User) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

// AddCredential 添加新的凭证到用户账户
func (u *User) AddCredential(cred webauthn.Credential) {
	u.Credentials = append(u.Credentials, cred)
}

// 注册请求结构体
type StartRequest struct {
	Username string `json:"username"`
}

// initDb 初始化 SQLite 数据库并自动迁移表结构
func initDb() {
	var err error
	db, err = gorm.Open(sqlite.Open("webauthn.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	err = db.AutoMigrate(&User{})
	if err != nil {
		log.Fatal("failed to migrate database:", err)
	}

	log.Println("✅ 数据库初始化完成")
}

// initWebAuthn 初始化 WebAuthn 配置
func initWebAuthn() {
	config := &webauthn.Config{
		RPDisplayName: "WebAuthn Demo",
		RPID:          "localhost",
		RPOrigins: []string{
			"http://localhost:5000",
			"http://127.0.0.1:5000",
			"http://localhost:8080",
			"http://127.0.0.1:8080",
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		},
	}

	var err error
	wa, err = webauthn.New(config)
	if err != nil {
		log.Fatal("failed to initialize WebAuthn:", err)
	}
	log.Println("✅ WebAuthn 初始化完成")
}

// convertCredentialCreation 将凭证创建请求转换为前端可用的 JSON 格式
func convertCredentialCreation(creation *protocol.CredentialCreation, user User) map[string]interface{} {
	opts := creation.Response

	challenge := base64.RawURLEncoding.EncodeToString(opts.Challenge)

	userID := base64.RawURLEncoding.EncodeToString(user.WebAuthnID())

	var excludeCreds []map[string]interface{}
	for _, cred := range opts.CredentialExcludeList {
		excludeCreds = append(excludeCreds, map[string]interface{}{
			"id":         base64.RawURLEncoding.EncodeToString(cred.CredentialID),
			"type":       cred.Type,
			"transports": cred.Transport,
		})
	}

	var pubKeyParams []map[string]interface{}
	for _, param := range opts.Parameters {
		pubKeyParams = append(pubKeyParams, map[string]interface{}{
			"type": param.Type,
			"alg":  param.Algorithm,
		})
	}

	result := map[string]interface{}{
		"rp": map[string]interface{}{
			"name": opts.RelyingParty.Name,
			"id":   opts.RelyingParty.ID,
		},
		"user": map[string]interface{}{
			"id":          userID,
			"name":        opts.User.Name,
			"displayName": opts.User.DisplayName,
		},
		"challenge":        challenge,
		"pubKeyCredParams": pubKeyParams,
		"timeout":          opts.Timeout,
		"attestation":      string(opts.Attestation),
	}

	if len(excludeCreds) > 0 {
		result["excludeCredentials"] = excludeCreds
	}

	if opts.AuthenticatorSelection.AuthenticatorAttachment != "" ||
		opts.AuthenticatorSelection.ResidentKey != "" ||
		opts.AuthenticatorSelection.UserVerification != "" {
		result["authenticatorSelection"] = map[string]interface{}{
			"authenticatorAttachment": opts.AuthenticatorSelection.AuthenticatorAttachment,
			"residentKey":             opts.AuthenticatorSelection.ResidentKey,
			"userVerification":        opts.AuthenticatorSelection.UserVerification,
		}
	}

	return result
}

// convertCredentialAssertion 将凭证断言请求转换为前端可用的 JSON 格式
func convertCredentialAssertion(assertion *protocol.CredentialAssertion) map[string]interface{} {
	opts := assertion.Response

	challenge := base64.RawURLEncoding.EncodeToString(opts.Challenge)

	var allowCreds []map[string]interface{}
	for _, cred := range opts.AllowedCredentials {
		allowCreds = append(allowCreds, map[string]interface{}{
			"id":         base64.RawURLEncoding.EncodeToString(cred.CredentialID),
			"type":       cred.Type,
			"transports": cred.Transport,
		})
	}

	result := map[string]interface{}{
		"challenge":        challenge,
		"rpId":             opts.RelyingPartyID,
		"timeout":          opts.Timeout,
		"userVerification": string(opts.UserVerification),
	}

	if len(allowCreds) > 0 {
		result["allowCredentials"] = allowCreds
	}

	return result
}

// hRegisterStart 处理注册请求的开始阶段，生成注册挑战
func hRegisterStart(ctx *gin.Context) {
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
	var existingUser User
	result := db.Where("username = ?", req.Username).First(&existingUser)
	if result.Error == nil {
		ctx.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}

	// 使用用户名创建临时用户对象用于生成挑战
	user := User{
		Username: req.Username,
	}

	creation, session, err := wa.BeginRegistration(&user)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "生成注册挑战失败：" + err.Error()})
		return
	}

	// 保存会话和用户信息
	sessionMutex.Lock()
	sessions[req.Username] = session
	sessionMutex.Unlock()

	response := convertCredentialCreation(creation, user)

	ctx.JSON(http.StatusOK, response)
}

// hRegisterFinish 处理注册完成阶段，验证并保存新生成的凭证
func hRegisterFinish(ctx *gin.Context) {
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
	tempUser := &User{
		Username: req.Username,
	}

	credentialParsed, err := protocol.ParseCredentialCreationResponseBody(
		io.NopCloser(bytes.NewReader(bodyBytes)),
	)
	if err != nil {
		log.Printf("❌ 解析凭证创建响应失败：%v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "解析凭证创建响应失败：" + err.Error()})
		return
	}

	// 使用临时用户进行验证
	cred, err := wa.CreateCredential(tempUser, *session, credentialParsed)
	if err != nil {
		log.Printf("❌ 注册验证失败：%+v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "注册验证失败：" + err.Error()})
		return
	}

	// 验证通过后，创建真实用户并保存到数据库
	user := User{
		Username:    req.Username,
		Credentials: []webauthn.Credential{*cred},
	}
	if err := db.Create(&user).Error; err != nil {
		log.Printf("❌ 保存用户失败：%v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "保存用户失败：" + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "registered"})
}

// hLoginStart 处理登录请求的开始阶段，生成登录挑战
func hLoginStart(ctx *gin.Context) {
	var req StartRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	if req.Username == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "用户名不能为空"})
		return
	}

	var user User
	result := db.First(&user, "username = ?", req.Username)
	if result.Error != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "用户未注册"})
		return
	}

	if len(user.Credentials) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "用户未设置通行密钥"})
		return
	}

	assertion, session, err := wa.BeginLogin(&user)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "生成登录挑战失败：" + err.Error()})
		return
	}

	sessionMutex.Lock()
	sessions[req.Username] = session
	sessionMutex.Unlock()

	response := convertCredentialAssertion(assertion)

	ctx.JSON(http.StatusOK, response)
}

// hLoginFinish 处理登录完成阶段，验证用户凭证并完成登录
func hLoginFinish(ctx *gin.Context) {
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

	var user User
	result := db.First(&user, "username = ?", req.Username)
	if result.Error != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	assertionParsed, err := protocol.ParseCredentialRequestResponseBody(
		io.NopCloser(bytes.NewReader(bodyBytes)),
	)
	if err != nil {
		log.Printf("❌ 解析认证响应失败：%v", err)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "登录验证失败：" + err.Error()})
		return
	}

	cred, err := wa.ValidateLogin(&user, *session, assertionParsed)
	if err != nil {
		log.Printf("❌ 登录验证失败：%+v", err)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "登录验证失败：" + err.Error()})
		return
	}

	for i, c := range user.Credentials {
		if string(c.ID) == string(cred.ID) {
			user.Credentials[i] = *cred
			db.Save(&user)
			break
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "login ok"})
}

// initGin 初始化 Gin Web 框架，配置 CORS 和路由
func initGin() {
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

	r.POST("/webauthn/register/start", hRegisterStart)
	r.POST("/webauthn/register/finish", hRegisterFinish)
	r.POST("/webauthn/login/start", hLoginStart)
	r.POST("/webauthn/login/finish", hLoginFinish)

	r.Static("/app", "./public")

	log.Println("🚀 服务器启动在 http://127.0.0.1:5000")
	log.Println("📁 前端访问：http://127.0.0.1:5000/app/")
	r.Run("0.0.0.0:5000")
}

// main 程序入口函数，依次初始化数据库、WebAuthn 和 Web 服务器
func main() {
	log.Println("=== WebAuthn 服务启动 ===")
	initDb()
	initWebAuthn()
	initGin()
}
