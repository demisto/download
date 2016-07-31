package web

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/demisto/download/conf"
	"github.com/demisto/download/domain"
	"github.com/demisto/download/util"
	"github.com/go-errors/errors"
	"github.com/gorilla/context"
)

func recoverHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.WithField("error", err).Warn("Recovered from error")
				log.Error(errors.Wrap(err, 2).ErrorStack())
				WriteError(w, ErrInternalServer)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (l *loggingResponseWriter) WriteHeader(status int) {
	l.status = status
	l.ResponseWriter.WriteHeader(status)
}

func loggingHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		lw := &loggingResponseWriter{w, 200}
		t1 := time.Now()
		next.ServeHTTP(lw, r)
		t2 := time.Now()
		log.Infof("[%s] %q %v %v", r.Method, r.URL.String(), lw.status, t2.Sub(t1))
	}

	return http.HandlerFunc(fn)
}

func acceptHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept"), "application/json") {
			log.Warn("Request without accept header received")
			WriteError(w, ErrNotAcceptable)
			return
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func contentTypeHandler(next http.Handler, contentType string) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if !strings.Contains(ct, contentType) {
			log.Warnf("Request without proper content type received. Got: %s, Expected: %s", ct, contentType)
			WriteError(w, ErrUnsupportedMediaType)
			return
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func jsonContentTypeHandler(next http.Handler) http.Handler {
	return contentTypeHandler(next, "application/json")
}

func multipartContentTypeHandler(next http.Handler) http.Handler {
	return contentTypeHandler(next, "multipart/form-data")
}

func bodyHandler(v interface{}) func(http.Handler) http.Handler {
	t := reflect.TypeOf(v)

	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			val := reflect.New(t).Interface()
			err := json.NewDecoder(r.Body).Decode(val)

			if err != nil {
				log.WithFields(log.Fields{"body": r.Body, "err": err}).Warn("Error handling body")
				WriteError(w, ErrBadRequest)
				return
			}

			if next != nil {
				context.Set(r, "body", val)
				next.ServeHTTP(w, r)
			}
		}

		return http.HandlerFunc(fn)
	}

	return m
}

const (
	// xsrfCookie is the name of the XSRF cookie
	xsrfCookie = `XSRF-TOKEN`
	// xsrfHeader is the name of the expected header
	xsrfHeader = `X-XSRF-TOKEN`
	// noXsrfAllowed is the error message
	noXSRFAllowed = `No XSRF Allowed`
	// xFrameOptionsHeader is the name of the x frame header
	xFrameOptionsHeader = `X-Frame-Options`
)

// Handle Clickjacking protection
func clickjackingHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(xFrameOptionsHeader, "DENY")
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func decryptSession(request *http.Request) (*session, error) {
	cookie, err := request.Cookie(sessionCookie)
	// No session, bye bye
	if err != nil {
		return nil, err
	}
	var sess session
	err = util.DecryptJSON(cookie.Value, conf.Options.Security.SessionKey, &sess)
	if err != nil {
		log.WithFields(log.Fields{"cookie": cookie.Value, "error": err}).Warn("Unable to decrypt encrypted session")
		return nil, err
	}
	// If the session is no longer valid
	if sess.When+int64(conf.Options.Security.Timeout)*60*1000 < time.Now().Unix()*1000 {
		log.Debug("Session timeout")
		return nil, ErrAuth
	}
	return &sess, nil
}

// Handle CSRF protection
func csrfHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		csrf, err := r.Cookie(xsrfCookie)
		csrfHeader := r.Header.Get(xsrfHeader)
		ok := false
		secure := conf.Options.SSL.Key != ""
		pass := conf.Options.Security.SessionKey

		// If it is an idempotent method, set the cookie
		if r.Method == "GET" || r.Method == "HEAD" || r.RequestURI == "/saml" {
			// But set it only if there is no XSRF or the session is not valid
			shouldCreate := err != nil
			if !shouldCreate {
				// Check session only if XSRF token exists
				_, err := decryptSession(r)
				shouldCreate = err != nil
			}
			if shouldCreate {
				val, cErr := util.Encrypt(noXSRFAllowed+time.Now().String(), []byte(pass))
				if cErr == nil {
					http.SetCookie(w, &http.Cookie{Name: xsrfCookie, Value: val, Path: "/", Expires: time.Now().Add(365 * 24 * time.Hour), MaxAge: 365 * 24 * 60 * 60, Secure: secure, HttpOnly: false})
				} else {
					log.WithField("error", cErr).Error("Unable to generate CSRF")
				}
			}
			ok = true
		} else if err == nil && csrf.Value == csrfHeader {
			val, cErr := util.Decrypt(csrfHeader, []byte(pass))
			if cErr == nil && strings.HasPrefix(val, noXSRFAllowed) {
				ok = true
			} else if cErr != nil {
				log.WithError(cErr).Errorf("Failed to execute %s method because of csrf", r.Method)
			}
		} else {
			log.WithError(err).Warnf("Csrf issue for method : %s", r.Method)
		}
		if ok {
			next.ServeHTTP(w, r)
		} else {
			WriteError(w, ErrCSRF)
		}
	}
	return http.HandlerFunc(fn)
}

const (
	sessionCookie = `SD`
)

func (ac *AppContext) authHandler(next http.Handler) http.Handler {
	fn := func(writer http.ResponseWriter, request *http.Request) {
		session, err := decryptSession(request)
		if err != nil {
			WriteError(writer, ErrAuth)
			return
		}
		context.Set(request, "session", &session)
		log.Debugf("User %v in request", session.User)
		u, err := ac.r.User(session.User)
		if err != nil {
			log.WithFields(log.Fields{"username": session.User, "id": session.User, "error": err}).Warn("Unable to load user from repository")
			panic(err)
		}

		context.Set(request, "user", u)

		// Set the new cookie for the user with the new timeout
		session.When = time.Now().Unix() * 1000
		secure := conf.Options.SSL.Key != ""
		timeout := conf.Options.Security.Timeout
		val, _ := util.EncryptJSON(&session, conf.Options.Security.SessionKey)
		http.SetCookie(writer, &http.Cookie{
			Name:     sessionCookie,
			Value:    val,
			Path:     "/",
			Expires:  time.Now().Add(time.Duration(timeout) * time.Minute),
			MaxAge:   timeout * 60,
			Secure:   secure,
			HttpOnly: true,
		})
		next.ServeHTTP(writer, request)
	}
	return http.HandlerFunc(fn)
}

func (ac *AppContext) permissionsHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		currentUser := context.Get(r, "user").(*domain.User)
		requires := context.Get(r, "requires")
		if requires != nil && len(requires.([]domain.UserType)) > 0 {
			if !util.In(requires, currentUser.Type) {
				WriteError(w, ErrPermission)
				return
			}
		}
		next.ServeHTTP(w, r)
		return
	}
	return http.HandlerFunc(fn)
}
