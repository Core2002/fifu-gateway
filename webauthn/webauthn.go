package webauthn

import (
	"encoding/base64"
	"io"
	"log"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

var WA *webauthn.WebAuthn

// SessionData WebAuthn 会话数据类型
type SessionData = webauthn.SessionData

// Credential WebAuthn 凭证类型
type Credential = webauthn.Credential

// Init 初始化 WebAuthn 配置
func Init() {
	config := &webauthn.Config{
		RPDisplayName: "FiFu WebAuthn",
		RPID:          "tls.internal",
		RPOrigins: []string{
			"http://localhost:5200",
			"http://localhost:5000",
			"https://tls.internal",
		},
	}

	var err error
	WA, err = webauthn.New(config)
	if err != nil {
		log.Fatal("failed to initialize WebAuthn:", err)
	}
	log.Println("✅ WebAuthn 初始化完成")
}

// transportsToString 转换 transports 为字符串数组
func transportsToString(transports []protocol.AuthenticatorTransport) []string {
	var result []string
	for _, t := range transports {
		result = append(result, string(t))
	}
	return result
}

// credentialDescriptor 转换凭证描述符为 SimpleWebAuthn 兼容格式
func credentialDescriptor(cred protocol.CredentialDescriptor) map[string]interface{} {
	m := map[string]interface{}{
		"id":   base64.RawURLEncoding.EncodeToString(cred.CredentialID),
		"type": cred.Type,
	}
	if len(cred.Transport) > 0 {
		m["transports"] = transportsToString(cred.Transport)
	}
	return m
}

// ConvertCredentialCreation 将凭证创建请求转换为前端可用的 JSON 格式
// 兼容 SimpleWebAuthn 的 PublicKeyCredentialCreationOptions 格式
func ConvertCredentialCreation(creation *protocol.CredentialCreation, user UserLike) map[string]interface{} {
	opts := creation.Response

	challenge := base64.RawURLEncoding.EncodeToString(opts.Challenge)
	userID := base64.RawURLEncoding.EncodeToString(user.WebAuthnID())

	excludeCreds := make([]map[string]interface{}, 0, len(opts.CredentialExcludeList))
	for _, cred := range opts.CredentialExcludeList {
		excludeCreds = append(excludeCreds, credentialDescriptor(cred))
	}

	pubKeyParams := make([]map[string]interface{}, 0, len(opts.Parameters))
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

	// 构建 authenticatorSelection
	authSel := opts.AuthenticatorSelection
	if authSel.AuthenticatorAttachment != "" ||
		authSel.ResidentKey != "" ||
		authSel.UserVerification != "" ||
		authSel.RequireResidentKey != nil {
		sel := make(map[string]interface{})
		if authSel.AuthenticatorAttachment != "" {
			sel["authenticatorAttachment"] = string(authSel.AuthenticatorAttachment)
		}
		if authSel.RequireResidentKey != nil {
			sel["requireResidentKey"] = *authSel.RequireResidentKey
		}
		if authSel.ResidentKey != "" {
			sel["residentKey"] = string(authSel.ResidentKey)
		}
		if authSel.UserVerification != "" {
			sel["userVerification"] = string(authSel.UserVerification)
		}
		result["authenticatorSelection"] = sel
	}

	return result
}

// ConvertCredentialAssertion 将凭证断言请求转换为前端可用的 JSON 格式
// 兼容 SimpleWebAuthn 的 PublicKeyCredentialRequestOptions 格式
func ConvertCredentialAssertion(assertion *protocol.CredentialAssertion) map[string]interface{} {
	opts := assertion.Response

	challenge := base64.RawURLEncoding.EncodeToString(opts.Challenge)

	allowCreds := make([]map[string]interface{}, 0, len(opts.AllowedCredentials))
	for _, cred := range opts.AllowedCredentials {
		allowCreds = append(allowCreds, credentialDescriptor(protocol.CredentialDescriptor{
			CredentialID: cred.CredentialID,
			Type:         cred.Type,
			Transport:    cred.Transport,
		}))
	}

	result := map[string]interface{}{
		"challenge": challenge,
		"timeout":   opts.Timeout,
	}

	if opts.RelyingPartyID != "" {
		result["rpId"] = opts.RelyingPartyID
	}

	if len(allowCreds) > 0 {
		result["allowCredentials"] = allowCreds
	}

	if opts.UserVerification != "" {
		result["userVerification"] = string(opts.UserVerification)
	}

	return result
}

// ParseCredentialCreation 解析凭证创建响应
func ParseCredentialCreation(body io.ReadCloser) (*protocol.ParsedCredentialCreationData, error) {
	return protocol.ParseCredentialCreationResponseBody(body)
}

// ParseCredentialRequest 解析凭证请求响应
func ParseCredentialRequest(body io.ReadCloser) (*protocol.ParsedCredentialAssertionData, error) {
	return protocol.ParseCredentialRequestResponseBody(body)
}

// UserLike 定义一个接口，包含 WebAuthn 用户所需的基本方法
type UserLike interface {
	WebAuthnID() []byte
	WebAuthnName() string
	WebAuthnDisplayName() string
	WebAuthnIcon() string
	WebAuthnCredentials() []webauthn.Credential
}
