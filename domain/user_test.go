package domain

import "testing"

func TestGetHashFromPassword(t *testing.T) {
	if len(GetHashFromPassword("1234")) != 80 {
		t.Fatal("Incorrect hash")
	}
}

func TestSetPassword(t *testing.T) {
	user := User{
		Username: "test",
		Hash:     "hash",
		Email:    "test@acme.com",
		Name:     "Tester",
	}
	user.SetPassword("1234")
	if len(user.Hash) != 80 {
		t.Fatal("Incorrect hash")
	}
}
