package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bytes"
	"strings"

	"github.com/stretchr/testify/assert"
)

func TestRouter(t *testing.T) {
	f := newHandlerFixture(t)
	defer f.Close()

	req, err := http.NewRequest("GET", "http://demisto.com/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Did not receive the correct status: %v - %v", rec.Code, rec.Body)
	}
}

func TestClickjacking(t *testing.T) {
	f := newHandlerFixture(t)
	defer f.Close()

	req, err := http.NewRequest("GET", "http://demisto.com/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Did not receive the correct status: %v - %v", rec.Code, rec.Body)
	}

	assert.EqualValues(t, rec.Header().Get(xFrameOptionsHeader), "DENY",
		"X-FRAME-OPTIONS header must be set to DENY to protect from ClickJacking")

}

func loginWithUserAndPassword(t *testing.T, f *HandlerFixture, username, pswd string, failOnStatus bool) string {
	loginReq, err := http.NewRequest("POST", "http://demisto.com/login", bytes.NewBufferString(`{"user":"`+username+`","password":"`+pswd+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	f.sendRequest(loginReq, false, "")

	if f.response.Code != http.StatusOK && failOnStatus {
		t.Fatalf("Could not login - %v %v", f.response.Code, f.response.Body)
	}
	res := ""
	if f.response.Code == http.StatusOK {
		session := f.response.Header()["Set-Cookie"][0]
		res = strings.SplitN(strings.Split(session, ";")[0], "=", 2)[1]
	}
	f.response = httptest.NewRecorder()
	return res
}
