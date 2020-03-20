package auth_test

import (
	"testing"

	"github.com/olafal0/dispatch/auth"
)

func TestPasswordChecking(t *testing.T) {
	hash, err := auth.GetHash("testpassword")
	if err != nil {
		t.Error(err)
	}

	valid := auth.CheckPassword("testpassword", hash)
	if !valid {
		t.Error("Should have been valid")
	}

	valid = auth.CheckPassword("testpasswrd", hash)
	if valid {
		t.Error("Should not have been valid")
	}
}

func TestJWTs(t *testing.T) {
	signer := auth.NewTokenSigner("dispatch", []byte("GcWik@!FN2s@xZK#rXh&FkLM9b^dGLQs"))
	token, err := signer.CreateToken("testuser")
	if err != nil {
		t.Error(err)
	}

	claims, err := signer.ParseToken(token)
	if err != nil {
		t.Error(err)
	}
	if claims.Subject != "testuser" {
		t.Errorf("Incorrect sub: %s\n", claims.Subject)
	}
	if claims.Issuer != "dispatch" {
		t.Errorf("Incorrect issuer: %s\n", claims.Issuer)
	}
}
