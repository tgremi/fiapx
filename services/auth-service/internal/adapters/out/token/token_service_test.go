package token

import (
	"testing"
	"time"
)

func TestGenerateAndValidate(t *testing.T) {
	s := NewTokenService("segredo", time.Hour)

	tok, err := s.Generate("u-123", "a@b.com")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	claims, err := s.Validate(tok)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.UserID != "u-123" || claims.Email != "a@b.com" {
		t.Fatalf("claims: %+v", claims)
	}
}

func TestValidateInvalidToken(t *testing.T) {
	s := NewTokenService("segredo", time.Hour)
	if _, err := s.Validate("token-invalido"); err == nil {
		t.Fatal("esperado erro para token inválido")
	}
}

func TestValidateWrongSecret(t *testing.T) {
	a := NewTokenService("a", time.Hour)
	b := NewTokenService("b", time.Hour)

	tok, err := a.Generate("u", "e@e.com")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if _, err := b.Validate(tok); err == nil {
		t.Fatal("esperado erro para secret diferente")
	}
}
