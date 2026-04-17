package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// PKCEHelper PKCE 辅助工具
type PKCEHelper struct{}

// GeneratePKCE 生成 PKCE verifier 和 challenge
func (h *PKCEHelper) GeneratePKCE() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	verifier = base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b)

	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])

	return verifier, challenge, nil
}

// GenerateState 生成随机 state
func (h *PKCEHelper) GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), nil
}

// GenerateUUID 生成随机 UUID
func (h *PKCEHelper) GenerateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
