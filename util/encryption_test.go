package util

import (
	"regexp"
	"testing"
)

func TestEncrypt(t *testing.T) {
	plain := `
	the quick brown fox jumped over the white fence
	the quick brown fox jumped over the white fence
	the quick brown fox jumped over the white fence
	the quick brown fox jumped over the white fence
	the quick brown fox jumped over the white fence
	the quick brown fox jumped over the white fence
	the quick brown fox jumped over the white fence
	the quick brown fox jumped over the white fence
	the quick brown fox jumped over the white fence
	the quick brown fox jumped over the white fence
	`
	pass := []byte(`12345678901234567890123456789012`)
	// Test with 32 byte key
	ciph, err := Encrypt(plain, pass)
	if err != nil {
		t.Fatal(err)
	}
	plain1, err := Decrypt(ciph, pass)
	if err != nil {
		t.Fatal(err)
	}
	if plain != plain1 {
		t.Fatalf("Expected '%v' but got '%v'", plain, plain1)
	}
}

func TestJson(t *testing.T) {
	type plainType struct {
		t1 string
	}
	plain := plainType{"t1"}
	pass := `12345678901234567890123456789012`
	// Test with 32 byte key
	ciph, err := EncryptJSON(plain, pass)
	if err != nil {
		t.Fatal(err)
	}
	err = DecryptJSON(ciph, pass, &plain)
	if err != nil {
		t.Fatal(err)
	}
	if plain.t1 != "t1" {
		t.Fatalf("Expected '%v' but got '%v'", "t1", plain.t1)
	}
}

func TestSecureRandomStringAlphaOnly(t *testing.T) {
	str := SecureRandomString(50, true)
	if len(str) != 50 {
		t.Fatalf("Expected %v length but got %v", 50, len(str))
	}

	alphaOnly := regexp.MustCompile(`^[A-z]+$`)
	if !alphaOnly.MatchString(str) {
		t.Fatal("Expected random string to be alpha only")
	}
}

func TestSecureRandomStringAlphaNumeric(t *testing.T) {
	str := SecureRandomString(50, false)
	if len(str) != 50 {
		t.Fatalf("Expected %v length but got %v", 50, len(str))
	}

	alphaOnly := regexp.MustCompile(`^[A-z0-9]+$`)
	if !alphaOnly.MatchString(str) {
		t.Fatal("Expected random string to contains both alpha and numeric charcters")
	}
}
