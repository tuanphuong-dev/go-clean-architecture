package common

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"go-clean-arch/domain"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JwtProviderConfig interface {
	AccessTokenExpiresIn() time.Duration
	AccessTokenSecret() string
	RefreshTokenExpiresIn() time.Duration
	RefreshTokenSecret() string
	TokenIssuer() string
}

type JWTProvider struct {
	cfg JwtProviderConfig
}

func NewJWTProvider(cfg JwtProviderConfig) *JWTProvider {
	return &JWTProvider{cfg: cfg}
}

func (j *JWTProvider) Generate(tokenType domain.TokenType, userID, sessionID string) (string, error) {
	switch tokenType {
	case domain.TokenTypeAccess:
		return j.generateAccessToken(userID, sessionID)
	case domain.TokenTypeRefresh:
		return j.generateRefreshToken()
	default:
		return "", errors.New("invalid token type")
	}
}

func (j *JWTProvider) generateAccessToken(userID, sessionID string) (string, error) {
	claims := domain.JwtClaims{
		Sub: userID,
		Sid: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.cfg.TokenIssuer(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.cfg.AccessTokenExpiresIn())),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.cfg.AccessTokenSecret()))
}

func (j *JWTProvider) generateRefreshToken() (string, error) {
	// Generate 32 random bytes and encode as hex string
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (j *JWTProvider) Verify(tokenType domain.TokenType, tokenStr string) (*domain.JwtClaims, error) {
	// Only access tokens are JWT tokens that can be verified
	if tokenType != domain.TokenTypeAccess {
		return nil, errors.New("only access tokens can be verified with JWT")
	}

	token, err := jwt.ParseWithClaims(tokenStr, &domain.JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.cfg.AccessTokenSecret()), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*domain.JwtClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
