package hasher

import "testing"

func TestHashAndCompare(t *testing.T) {
	h := NewPasswordHasher()

	hash, err := h.Hash("senha-secreta")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "" || hash == "senha-secreta" {
		t.Fatalf("hash inválido: %q", hash)
	}

	if !h.Compare(hash, "senha-secreta") {
		t.Fatal("Compare deveria retornar true para senha correta")
	}
	if h.Compare(hash, "senha-errada") {
		t.Fatal("Compare deveria retornar false para senha errada")
	}
	if h.Compare("hash-invalido", "senha") {
		t.Fatal("Compare deveria retornar false para hash inválido")
	}
}
