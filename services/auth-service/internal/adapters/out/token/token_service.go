package token

import (
	"errors"
	"time"

	"github.com/fiapx/auth-service/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

type TokenService struct {
	secret []byte
	expiry time.Duration
}

func NewTokenService(secret string, expiry time.Duration) *TokenService {
	return &TokenService{secret: []byte(secret), expiry: expiry}
}

func (s *TokenService) Generate(userID, email string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"iat":   now.Unix(),
		"exp":   now.Add(s.expiry).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

func (s *TokenService) Validate(tokenString string) (domain.Claims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil || !token.Valid {
		return domain.Claims{}, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return domain.Claims{}, errors.New("invalid claims")
	}

	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	return domain.Claims{UserID: sub, Email: email}, nil
}
