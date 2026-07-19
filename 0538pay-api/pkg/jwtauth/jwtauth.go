// Package jwtauth 封装 JWT 签发与解析。
package jwtauth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 是 token 内携带的身份信息。
type Claims struct {
	UID   uint   `json:"uid"`
	Name  string `json:"name"`
	Role  string `json:"role"` // 角色标识，用于 RBAC
	Scope string `json:"scope"` // admin / merchant / console
	jwt.RegisteredClaims
}

// Manager 持有签名密钥与有效期。
type Manager struct {
	secret      []byte
	expireHours int
}

func New(secret string, expireHours int) *Manager {
	if expireHours <= 0 {
		expireHours = 72
	}
	return &Manager{secret: []byte(secret), expireHours: expireHours}
}

// Generate 签发一个 token。
func (m *Manager) Generate(uid uint, name, role, scope string) (string, error) {
	now := time.Now()
	claims := Claims{
		UID:   uid,
		Name:  name,
		Role:  role,
		Scope: scope,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(m.expireHours) * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Parse 校验并解析 token。
func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("签名方法非法")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("token 无效")
	}
	return claims, nil
}
