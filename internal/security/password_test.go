package security

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	const plain = "super-secret-password"

	hash, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == plain {
		t.Fatalf("hash must not equal plain password")
	}

	if err := CheckPassword(hash, plain); err != nil {
		t.Fatalf("CheckPassword() should succeed: %v", err)
	}

	if err := CheckPassword(hash, "wrong-password"); err == nil {
		t.Fatalf("CheckPassword() should fail for wrong password")
	}
}
