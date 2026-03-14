package utils

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/o1egl/paseto"
	"golang.org/x/crypto/ed25519"
)

type TokenPayload struct {
	UserID    uint      `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	IssuedAt  time.Time `json:"iat"`
	ExpiredAt time.Time `json:"exp"`
}

type PasetoMaker struct {
	passeto       *paseto.V2
	symmetricKey  []byte
	privateKey    ed25519.PrivateKey
	publicKey     ed25519.PublicKey
	useAsymmetric bool
}

// NewPasetoMaker 创建堆成加密模式的 Maker（V2.local）
func NewPasetoMaker(symmetricKey string) (*PasetoMaker, error) {
	if len(symmetricKey) != 32 {
		return nil, fmt.Errorf("invalid key size: must be 32 bytes, got %d", len(symmetricKey))
	}

	maker := &PasetoMaker{
		passeto:       paseto.NewV2(),
		symmetricKey:  []byte(symmetricKey),
		useAsymmetric: false,
	}
	return maker, nil
}

// NewPasetoMakerAsymmetric 创建非堆成加密模式的 Maker（v2.public）
func NewPasetoMakerAsymmetric(publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey) (*PasetoMaker, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid key size: must be 32 bytes, got %d", len(privateKey))
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid key size: must be 32 bytes, got %d", len(publicKey))
	}
	maker := &PasetoMaker{
		passeto:       paseto.NewV2(),
		privateKey:    privateKey,
		publicKey:     publicKey,
		useAsymmetric: true,
	}
	return maker, nil
}

// GenerateKeys 生成 Ed25519 密钥对（用于非堆成模式）
func GenerateKeys() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(nil)
}

func (maker *PasetoMaker) CreateToken(userID uint, username, role string, duration time.Duration) (string, error) {
	payload := TokenPayload{
		UserID:    userID,
		Username:  username,
		Role:      role,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(duration),
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	var token string
	if maker.useAsymmetric {
		// 使用私钥签名（v2.public）
		token, err = maker.passeto.Sign(maker.privateKey, payloadBytes, nil)
	} else {
		// 使用堆成密钥加密（v2.local）
		token, err = maker.passeto.Encrypt(maker.publicKey, payloadBytes, nil)
	}
	return token, err
}

// VerifyToken 验证并解析令牌
func (maker *PasetoMaker) VerifyToken(token string) (*TokenPayload, error) {
	var payloadBytes []byte
	var err error
	if maker.useAsymmetric {
		//使用公钥验证（v2.public）
		err = maker.passeto.Verify(token, maker.publicKey, payloadBytes, nil)
	} else {
		// 使用堆成密钥解密）—v2.local—
		err = maker.passeto.Decrypt(token, maker.symmetricKey, payloadBytes, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	var payload TokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	if time.Now().After(payload.ExpiredAt) {
		return nil, fmt.Errorf("token has expired")
	}

	return &payload, nil
}
