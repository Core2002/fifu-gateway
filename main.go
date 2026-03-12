package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
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

type User struct {
	ID          uint                  `gorm:"primarykey"`
	Username    string                `gorm:"unique"`
	Credentials []webauthn.Credential `gorm:"serializer:json"`
}

func (u *User) WebAuthnID() []byte {
	idBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(idBytes, uint64(u.ID))
	return idBytes
}

func (u *User) WebAuthnName() string        { return u.Username }
func (u *User) WebAuthnDisplayName() string { return u.Username }
func (u *User) WebAuthnIcon() string        { return "" }
func (u *User) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

func (u *User) AddCredential(cred webauthn.Credential) {
	u.Credentials = append(u.Credentials, cred)
}

type StartRequest struct {
	Username string `json:"username"`
}

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

func initWebAuthn() {
	config := &webauthn.Config{
		RPDisplayName: "WebAuthn Demo",
		RPID:          "localhost", // 或者改为空字符串 "" 允许所有
		RPOrigins: []string{
			"http://localhost:5000",
			"http://127.0.0.1:5000",
			"http://localhost:8080", // 如果你用其他端口
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

// 将 protocol.CredentialCreation 转换为前端可用的 JSON 格式
func convertCredentialCreation(creation *protocol.CredentialCreation, user User) map[string]interface{} {
	opts := creation.Response

	// 转换 challenge
	challenge := base64.RawURLEncoding.EncodeToString(opts.Challenge)

	// ✅ 修复：直接使用 user.WebAuthnID() 获取用户 ID
	userID := base64.RawURLEncoding.EncodeToString(user.WebAuthnID())

	// 转换 excludeCredentials
	var excludeCreds []map[string]interface{}
	for _, cred := range opts.CredentialExcludeList {
		excludeCreds = append(excludeCreds, map[string]interface{}{
			"id":         base64.RawURLEncoding.EncodeToString(cred.CredentialID),
			"type":       cred.Type,
			"transports": cred.Transport,
		})
	}

	// 转换 pubKeyCredParams
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

	// 转换 authenticatorSelection
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

// 将 protocol.CredentialAssertion 转换为前端可用的 JSON 格式
func convertCredentialAssertion(assertion *protocol.CredentialAssertion) map[string]interface{} {
	opts := assertion.Response

	// 转换 challenge
	challenge := base64.RawURLEncoding.EncodeToString(opts.Challenge)

	// 转换 allowCredentials
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

	log.Printf("📝 注册开始 - 用户名: %s", req.Username)

	var user User
	result := db.FirstOrCreate(&user, User{Username: req.Username})
	if result.Error != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "数据库操作失败"})
		return
	}

	// 使用 protocol 包创建注册选项
	creation, session, err := wa.BeginRegistration(&user)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "生成注册挑战失败: " + err.Error()})
		return
	}

	// 保存会话
	sessionMutex.Lock()
	sessions[req.Username] = session
	sessionMutex.Unlock()

	// 转换为前端可用的格式 - 传入 user 对象以获取正确的 ID
	response := convertCredentialCreation(creation, user)
	log.Printf("返回的注册选项: %+v", response)

	ctx.JSON(http.StatusOK, response)
}

func hRegisterFinish(ctx *gin.Context) {
	// 首先需要获取用户名，用于查找 session
	// 由于 ParseCredentialCreationResponseBody 会消费请求体，我们需要先从中提取用户名
	var req struct {
		Username string `json:"username"`
	}
	// 重新读取请求体以解析 username
	bodyBytes, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败：" + err.Error()})
		return
	}

	// 解析 JSON 获取 username
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

	var user User
	result := db.First(&user, "username = ?", req.Username)
	if result.Error != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 使用协议包解析凭证创建响应（从原始请求体）
	credentialParsed, err := protocol.ParseCredentialCreationResponseBody(
		io.NopCloser(bytes.NewReader(bodyBytes)),
	)
	if err != nil {
		log.Printf("❌ 解析凭证创建响应失败：%v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "解析凭证创建响应失败：" + err.Error()})
		return
	}

	// 使用验证后的数据完成注册
	cred, err := wa.CreateCredential(&user, *session, credentialParsed)
	if err != nil {
		log.Printf("❌ 注册验证失败：%+v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "注册验证失败：" + err.Error()})
		return
	}

	user.AddCredential(*cred)
	db.Save(&user)

	ctx.JSON(http.StatusOK, gin.H{"status": "registered"})
}

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
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "生成登录挑战失败: " + err.Error()})
		return
	}

	sessionMutex.Lock()
	sessions[req.Username] = session
	sessionMutex.Unlock()

	// 转换为前端可用的格式
	response := convertCredentialAssertion(assertion)
	log.Printf("返回的登录选项: %+v", response)

	ctx.JSON(http.StatusOK, response)
}

func hLoginFinish(ctx *gin.Context) {
	// 首先需要获取用户名，用于查找 session
	// 读取请求体以解析 username
	bodyBytes, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败：" + err.Error()})
		return
	}

	// 解析 JSON 获取 username
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

	// 使用协议包解析认证响应（从原始请求体）
	assertionParsed, err := protocol.ParseCredentialRequestResponseBody(
		io.NopCloser(bytes.NewReader(bodyBytes)),
	)
	if err != nil {
		log.Printf("❌ 解析认证响应失败：%v", err)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "登录验证失败：" + err.Error()})
		return
	}

	// 使用验证后的数据完成登录
	cred, err := wa.ValidateLogin(&user, *session, assertionParsed)
	if err != nil {
		log.Printf("❌ 登录验证失败：%+v", err)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "登录验证失败：" + err.Error()})
		return
	}

	// 更新凭证计数器
	for i, c := range user.Credentials {
		if string(c.ID) == string(cred.ID) {
			user.Credentials[i] = *cred
			db.Save(&user)
			break
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "login ok"})
}

func initGin() {
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:5000",
			"http://127.0.0.1:5000",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
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
	log.Println("📁 前端访问: http://127.0.0.1:5000/app/")
	r.Run("0.0.0.0:5000")
}

func main() {
	log.Println("=== WebAuthn 服务启动 ===")
	initDb()
	initWebAuthn()
	initGin()
}
