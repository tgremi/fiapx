package token

import (
	"errors"

	"github.com/fiapx/video-api/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

type Validator struct {
	secret []byte
}

func NewValidator(secret string) *Validator {
	return &Validator{secret: []byte(secret)}
}

func (v *Validator) Validate(tokenString string) (domain.Claims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return v.secret, nil
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
