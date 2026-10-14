package api

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/xXMateus97Xx/email-pixel-tracker/auth"
	"github.com/xXMateus97Xx/email-pixel-tracker/domain"
)

const (
	tokenIssuer     = "email-pixel-tracker"
	minSecretLength = 32
)

var ErrInvalidToken = errors.New("invalid token")

type tokenClaims struct {
	TenantId             string      `json:"tid,omitempty"`
	ImpersonatedTenantId string      `json:"imp,omitempty"`
	Role                 domain.Role `json:"role"`
	jwt.RegisteredClaims
}

// TokenManager issues and validates HS256 JWTs carrying the principal.
// The token only identifies the user: the services reload the user (active
// flag and role) on every request.
type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenManager(secret string, ttl time.Duration) (*TokenManager, error) {
	if len(secret) < minSecretLength {
		return nil, errors.New("jwt secret must have at least 32 bytes")
	}
	if ttl <= 0 {
		return nil, errors.New("jwt ttl must be positive")
	}
	return &TokenManager{secret: []byte(secret), ttl: ttl}, nil
}

func (m *TokenManager) Issue(p auth.Principal) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.ttl)
	claims := tokenClaims{
		TenantId:             p.TenantId,
		ImpersonatedTenantId: p.ImpersonatedTenantId,
		Role:                 p.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tokenIssuer,
			Subject:   p.UserId,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func (m *TokenManager) Parse(token string) (auth.Principal, error) {
	var claims tokenClaims
	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return auth.Principal{}, ErrInvalidToken
	}

	p := auth.Principal{
		UserId:               claims.Subject,
		TenantId:             claims.TenantId,
		Role:                 claims.Role,
		ImpersonatedTenantId: claims.ImpersonatedTenantId,
	}
	if p.Validate() != nil {
		return auth.Principal{}, ErrInvalidToken
	}
	return p, nil
}
