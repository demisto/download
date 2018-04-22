package web

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/demisto/download/domain"
	"github.com/gorilla/context"
	"github.com/asaskevich/govalidator"
	"time"
)

// tokenHandler handles get retrieve tokens requests
func (ac *AppContext) tokenHandler(w http.ResponseWriter, r *http.Request) {
	tokens, err := ac.r.OpenTokens()
	if err != nil {
		log.WithError(err).Warn("Unable to retrieve tokens")
	}
	writeJSON(w, tokens)
}

type newTokens struct {
	Count     int `json:"count"`
	Downloads int `json:"downloads"`
}

// createTokensHandler handles creation of new tokens
func (ac *AppContext) createTokensHandler(w http.ResponseWriter, r *http.Request) {
	nt := context.Get(r, "body").(*newTokens)
	log.Infof("Generating tokens: %#v", nt)
	if nt.Count > 50 || nt.Count < 1 {
		WriteError(w, ErrBadRequest)
		return
	}
	tokens := make([]domain.Token, 0, nt.Count)
	for i := 0; i < nt.Count; i++ {
		token := domain.NewToken(nt.Downloads)
		err := ac.r.SetToken(token)
		if err != nil {
			log.WithError(err).Warnf("Unable to generate token - %#v", token)
			WriteError(w, ErrBadRequest)
			return
		}
		tokens = append(tokens, *token)
	}
	writeJSON(w, tokens)
}

// updateToken updates a single token
func (ac *AppContext) updateToken(w http.ResponseWriter, r *http.Request) {
	t := context.Get(r, "body").(*domain.Token)
	log.Infof("Updating token: %#v", t)
	err := ac.r.SetToken(t)
	if err != nil {
		log.WithError(err).Warnf("Unable to save token - %#v", t)
		WriteError(w, ErrBadRequest)
		return
	}
	writeJSON(w, t)
}

type newEmailToken struct {
	Email     string `json:"email"`
	Downloads int `json:"downloads"`
}

// createEmailTokenHandler handles creation of new tokens
func (ac *AppContext) createEmailTokenHandler(w http.ResponseWriter, r *http.Request) {
	nt := context.Get(r, "body").(*newEmailToken)
	if !govalidator.IsEmail(nt.Email) {
		WriteError(w, &Error{ID: "bad_request", Status: 400, Title: "Invalid Email", Detail: "Invalid email provided"})
		return
	}
	log.Infof("Generating token for : %s with %d downloads", nt.Email, nt.Downloads)
	token := domain.NewToken(nt.Downloads)
	err := ac.r.SetToken(token)
	if err != nil {
		log.WithError(err).Warnf("Unable to generate token - %#v", token)
		WriteError(w, ErrBadRequest)
		return
	}
	u := &domain.User{Username: token.Name + "*-*" + nt.Email, Email: nt.Email, Token: token.Name, Type: domain.UserTypeUser, LastLogin: time.Now()}
	err = ac.r.SetUser(u)
	if err != nil {
		log.WithError(err).Warnf("Error saving token user - %#v", u)
		WriteError(w, ErrInternalServer)
		return
	}
	writeJSON(w, token)
}
