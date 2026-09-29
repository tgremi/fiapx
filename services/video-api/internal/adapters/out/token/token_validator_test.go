package token

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestValidate(t *testing.T) {
	const secret = "segredo"
	v := NewValidator(secret)

	claims := jwt.MapClaims{"sub": "u-1", "email": "a@b.com", "exp": time.Now().Add(time.Hour).Unix()}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	got, err := v.Validate(tok)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if got.UserID != "u-1" || got.Email != "a@b.com" {
		t.Fatalf("claims: %+v", got)
	}
}

func TestValidateInvalidToken(t *testing.T) {
	v := NewValidator("segredo")
	if _, err := v.Validate("token-invalido"); err == nil {
		t.Fatal("esperado erro para token inválido")
	}
}

func TestValidateWrongSecret(t *testing.T) {
	v := NewValidator("segredo")

	claims := jwt.MapClaims{"sub": "u-1", "email": "a@b.com"}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("outro"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := v.Validate(tok); err == nil {
		t.Fatal("esperado erro para secret diferente")
	}
}
