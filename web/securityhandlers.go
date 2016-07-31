package web

import (
	"encoding/base64"
	"net"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/demisto/download/conf"
	"github.com/demisto/download/domain"
	"github.com/demisto/download/util"
	"github.com/gorilla/context"
	lru "github.com/hashicorp/golang-lru"
	"golang.org/x/crypto/bcrypt"
)

var bruteForceMap *lru.Cache

type credentials struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type userPreferences struct {
	ID       string   `json:"id"`
	Homepage string   `json:"homepage"`
	Notify   []string `json:"notify"`
}

func initBruteForceMap(sleep bool) {
	tmpBruteForceMap, err := lru.New(100)
	if err != nil {
		log.WithError(err).Error("Failed creating brute force lru map sleep:", sleep)
		if sleep {
			time.Sleep(time.Second * 10)
		}
	} else {
		bruteForceMap = tmpBruteForceMap
	}
}

func (ac *AppContext) preventBruteForce(key string) {
	var count int
	//This is just for safety if for some reason we fail to create the map in the initialization phase
	if bruteForceMap == nil {
		initBruteForceMap(true)
	}
	countInter, exists := bruteForceMap.Get(key)
	if exists {
		count, _ = countInter.(int)
		count++
	}
	if count <= 0 {
		count = 1
	}
	if count > 5 {
		time.Sleep(time.Second * 60 * time.Duration(count-5))
	} else if count > 2 {
		time.Sleep(time.Second * 10)
	}
	bruteForceMap.Add(key, count)
}

func (ac *AppContext) resetBruteForce(key string) {
	bruteForceMap.Remove(key)
}

// r.RemoteAddr is in format ip:port, and might contain ipv6 data in format a:a:a:a:a:a:port
func (ac *AppContext) getBruteforceKey(r *http.Request, username string) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return host + username
}

func (ac *AppContext) handleLoginError(r *http.Request, w http.ResponseWriter, user string) {
	ac.preventBruteForce(ac.getBruteforceKey(r, user))
	WriteError(w, ErrCredentials)
}

func (ac *AppContext) doLogin(w http.ResponseWriter, r *http.Request, username, password string) (u *domain.User) {
	body := &credentials{User: username, Password: password}
	if body.User == "" || body.Password == "" {
		ac.handleLoginError(r, w, body.User)
		return nil
	}

	u, err := ac.r.User(body.User)
	if err == nil {
		hash, err := base64.StdEncoding.DecodeString(u.Hash)
		if err != nil {
			ac.handleLoginError(r, w, body.User)
			return nil
		}
		if bcrypt.CompareHashAndPassword(hash, []byte(body.Password)) != nil {
			ac.handleLoginError(r, w, body.User)
			return nil
		}
	} else {
		ac.handleLoginError(r, w, body.User)
		return nil
	}
	// successful login need to reset login cookie
	ac.resetBruteForce(ac.getBruteforceKey(r, u.Username))
	return
}

func (ac *AppContext) loginResponse(w http.ResponseWriter, r *http.Request, u *domain.User) {
	log.Infof("User %s logged in\n", u.Username)
	loginTime := time.Now()

	sess := session{
		User: u.Username,
		When: loginTime.Unix() * 1000,
	}
	secure := conf.Options.SSL.Key != ""
	timeout := conf.Options.Security.Timeout
	val, _ := util.EncryptJSON(&sess, conf.Options.Security.SessionKey)

	u.LastLogin = loginTime
	ac.r.SetUser(u)

	// Set the cookie for the user
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    val,
		Path:     "/",
		Expires:  time.Now().Add(time.Duration(timeout) * time.Minute),
		MaxAge:   timeout * 60,
		Secure:   secure,
		HttpOnly: true,
	})
	writeWithFilter(w, u, domain.UserFilterFields...)
}
func (ac *AppContext) loginHandler(w http.ResponseWriter, r *http.Request) {
	body := context.Get(r, "body").(*credentials)
	u := ac.doLogin(w, r, body.User, body.Password)
	if u != nil {
		ac.loginResponse(w, r, u)
	}
}

func (ac *AppContext) logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", Expires: time.Now(), MaxAge: -1, Secure: conf.Options.SSL.Key != "", HttpOnly: true})
	w.WriteHeader(http.StatusNoContent)
	w.Write([]byte("\n"))
}

type userDetails struct {
	Username string          `json:"username"`
	Password string          `json:"password"`
	Email    string          `json:"email"`
	Name     string          `json:"name"`
	Type     domain.UserType `json:"type"`
	Token    string          `json:"token"`
}

// handleUserUpdate creates or updates any user in the system. Permissions are checked by middleware.
func (ac *AppContext) handleUserUpdate(w http.ResponseWriter, r *http.Request) {
	details := context.Get(r, "body").(*userDetails)
	// Skip validation checks for now
	u := &domain.User{
		Username:   details.Username,
		Hash:       domain.GetHashFromPassword(details.Password),
		Email:      details.Email,
		Name:       details.Name,
		Type:       details.Type,
		Token:      details.Token,
		ModifyDate: time.Now(),
	}
	ac.r.SetUser(u)
	writeWithFilter(w, u, domain.UserFilterFields...)
}

func (ac *AppContext) userCurrHandler(w http.ResponseWriter, r *http.Request) {
	u := context.Get(r, "user").(*domain.User)
	writeWithFilter(w, u, domain.UserFilterFields...)
}
