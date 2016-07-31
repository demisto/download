package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/demisto/download/conf"
	"github.com/demisto/download/repo"
	"github.com/demisto/download/util"
	"github.com/gorilla/context"
	"github.com/justinas/alice"
)

type HandlerFixture struct {
	appcontext *AppContext
	handlers   alice.Chain
	router     *Router
	response   *httptest.ResponseRecorder
	r          *repo.Repo
}

func newHandlerFixture(t *testing.T) *HandlerFixture {
	return create(t, "")
}

func create(t *testing.T, confFile string) *HandlerFixture {
	hf := new(HandlerFixture)
	if confFile == "" {
		conf.Default()
	} else {
		conf.Default()
		err := conf.Load(confFile)
		if err != nil {
			t.Fatal(err)
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	_, err = os.Stat(filepath.Join(wd, "static"))
	if err != nil {
		up, _ := filepath.Split(wd)
		_, err = os.Stat(filepath.Join(up, "static"))
		if err != nil {
			t.Fatal(err)
		}
		wd = up
	}
	hf.r, err = repo.New()
	if err != nil {
		t.Fatal(err)
	}
	hf.appcontext = NewContext(hf.r)
	hf.handlers = alice.New(context.ClearHandler, recoverHandler)
	hf.router = New(hf.appcontext, filepath.Join(wd, "static"))
	hf.response = httptest.NewRecorder()
	return hf
}

func (hf *HandlerFixture) Close() {
	hf.r.Close()
}

func (hf *HandlerFixture) sendMultiPartRequest(req *http.Request, isSessionCookie bool, sessionValue, contentTypeHeader string) {
	hf.sendGeneralRequest(req, isSessionCookie, sessionValue, contentTypeHeader)
}

func (hf *HandlerFixture) sendRequest(req *http.Request, isSessionCookie bool, sessionValue string) {
	hf.sendGeneralRequest(req, isSessionCookie, sessionValue, "application/json")
}

func (hf *HandlerFixture) sendGeneralRequest(req *http.Request, isSessionCookie bool, sessionValue, contentTypeHeader string) {
	hf.response = httptest.NewRecorder()

	contentType := "application/json"
	if len(contentTypeHeader) > 0 {
		contentType = contentTypeHeader
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", contentType)
	pass := conf.Options.Security.SessionKey
	val, cErr := util.Encrypt(noXSRFAllowed+time.Now().String(), []byte(pass))
	if cErr == nil {
		req.AddCookie(&http.Cookie{Name: xsrfCookie, Value: val})
		req.Header.Set(xsrfHeader, val)

	}
	if isSessionCookie {
		req.AddCookie(&http.Cookie{Name: sessionCookie, Value: sessionValue})
	}

	hf.router.ServeHTTP(hf.response, req)
}
