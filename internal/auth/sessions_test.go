package auth

import (
	"strings"
	"testing"
)

func TestNewTokenReturnsURLSafeRandomTokens(t *testing.T) {
	first, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if first == "" || second == "" {
		t.Fatal("expected non-empty tokens")
	}
	if first == second {
		t.Fatal("expected unique tokens")
	}
	if strings.ContainsAny(first+second, "+/=") {
		t.Fatalf("expected raw URL-safe tokens, got %q and %q", first, second)
	}
}

func TestTokenHashIsStableSecretBoundAndDoesNotStoreToken(t *testing.T) {
	token := "session-token"
	hash := TokenHash("secret-a", token)

	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == token {
		t.Fatal("token hash must not equal the raw token")
	}
	if hash != TokenHash("secret-a", token) {
		t.Fatal("expected stable hash for same secret and token")
	}
	if hash == TokenHash("secret-b", token) {
		t.Fatal("expected different hash for different secret")
	}
	if hash == TokenHash("secret-a", "other-token") {
		t.Fatal("expected different hash for different token")
	}
}

func TestConstantTimeEqual(t *testing.T) {
	if !ConstantTimeEqual("same", "same") {
		t.Fatal("expected equal strings to match")
	}
	if ConstantTimeEqual("same", "different") {
		t.Fatal("expected different strings not to match")
	}
}
