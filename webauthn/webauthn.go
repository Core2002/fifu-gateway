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
		RPDisplayName: "WebAuthn Demo",
		RPID:          "localhost",
		RPOrigins: []string{
			"http://localhost:5200",
			"http://localhost:5000",
			"https://10.0.2.221:5200",
		},
	}

	var err error
	WA, err = webauthn.New(config)
	if err != nil {
		log.Fatal("failed to initialize WebAuthn:", err)
	}
	log.Println("✅ WebAuthn 初始化完成")
}

// ConvertCredentialCreation 将凭证创建请求转换为前端可用的 JSON 格式
func ConvertCredentialCreation(creation *protocol.CredentialCreation, user UserLike) map[string]interface{} {
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

// ConvertCredentialAssertion 将凭证断言请求转换为前端可用的 JSON 格式
func ConvertCredentialAssertion(assertion *protocol.CredentialAssertion) map[string]interface{} {
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
