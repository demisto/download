package web

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoginAndOut(t *testing.T) {
	f := newHandlerFixture(t)
	defer f.Close()

	sessionValue := loginWithUserAndPassword(t, f, "slavik", "password", true)
	req, err := http.NewRequest("POST", "http://demisto.com/logout", nil)
	if err != nil {
		t.Fatal(err)
	}
	f.sendRequest(req, true, sessionValue)
	if f.response.Code != http.StatusNoContent {
		t.Fatalf("Did not receive the correct status - %v", f.response.Code)
	}
}

func TestBruteForce(t *testing.T) {
	f := newHandlerFixture(t)
	defer f.Close()
	loginWithUserAndPassword(t, f, "slavik", "wrong", false)
	assert.EqualValues(t, 1, bruteForceMap.Len(), "brute force map not updated")
	loginWithUserAndPassword(t, f, "slavik", "password", true)
	assert.EqualValues(t, 0, bruteForceMap.Len(), "brute force map not updated")
}
