package models

import "github.com/go-webauthn/webauthn/webauthn"

// User 用户结构体，实现 webauthn.Interface 接口
type User struct {
	ID          uint                  `gorm:"primarykey"`
	Username    string                `gorm:"unique"`
	Role        string                `gorm:"unique"`
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
