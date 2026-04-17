package models

import (
	"crypto/sha256"
	"encoding/base64"

	"github.com/go-webauthn/webauthn/webauthn"
)

// User 用户结构体，实现 webauthn.Interface 接口
type User struct {
	ID          uint                  `gorm:"primarykey"`
	Username    string                `gorm:"unique"`
	Role        string                `gorm:"default:'member'"`
	Credentials []webauthn.Credential `gorm:"serializer:json"`
}

// WebAuthnID 返回用户的 WebAuthn ID（使用 username 的稳定哈希）
// 注意：必须与注册和登录时一致，使用 username 的哈希确保稳定性
func (u *User) WebAuthnID() []byte {
	// 使用 username 的 SHA256 哈希作为 WebAuthn ID
	// 这样在注册时（ID=0）和登录时（ID已分配）都能得到相同的 ID
	hash := sha256.Sum256([]byte(u.Username))
	return hash[:]
}

// WebAuthnIDBase64 返回 base64 编码的 WebAuthn ID（用于调试）
func (u *User) WebAuthnIDBase64() string {
	return base64.RawURLEncoding.EncodeToString(u.WebAuthnID())
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
