// +build integration

package repo

import (
	"testing"

	"github.com/demisto/download/conf"
	"github.com/demisto/download/domain"
)

func getTestDB(t *testing.T) *Repo {
	conf.Default()
	r, err := New()
	if err != nil {
		t.Fatalf("%v", err)
	}
	r.db.Exec("DELETE FROM users")
	r.db.Exec("DELETE FROM questions")
	return r
}

func TestNew(t *testing.T) {
	r := getTestDB(t)
	r.Close()
}

func TestUser(t *testing.T) {
	r := getTestDB(t)
	u := &domain.User{Username: "test", Email: "kuku@kiki"}
	u.SetPassword("zzz")
	err := r.SetUser(u)
	if err != nil {
		t.Fatalf("Unable to create user - %v", err)
	}
	u1, err := r.User("test")
	if err != nil {
		t.Fatalf("Unable to load user - %v", err)
	}
	if u1.Username != u.Username {
		t.Error("User name is not retrieved")
	}
	u.Email = "aaa@bbb"
	r.SetUser(u)
	u1, err = r.User("test")

	if u.Email != u1.Email {
		t.Fatal("Email not updated")
	}
	r.Close()
}

func TestToken(t *testing.T) {
	r := getTestDB(t)
	token := &domain.Token{Name: "t", Downloads: 10}
	err := r.SetToken(token)
	if err != nil {
		t.Fatalf("Unable to create token - %v", err)
	}
	tokens, err := r.Tokens()
	if err != nil {
		t.Fatalf("Unable to retrieve tokens - %v", err)
	}
	if len(tokens) != 1 {
		t.Errorf("Expecting a single token - %v", tokens)
	}
	tokens, err = r.OpenTokens()
	if err != nil {
		t.Fatalf("Unable to retrieve open tokens - %v", err)
	}
	if len(tokens) != 1 {
		t.Errorf("Expecting a single open token - %v", tokens)
	}
	token.Downloads = 0
	err = r.SetToken(token)
	if err != nil {
		t.Fatalf("Unable to update token - %v", err)
	}
	tokens, err = r.OpenTokens()
	if err != nil {
		t.Fatalf("Unable to retrieve open tokens - %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("Expecting no open tokens - %v", tokens)
	}
}
