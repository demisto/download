package web

import (
	"math/rand"
	"net/http"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/asaskevich/govalidator"
	"github.com/demisto/download/conf"
	"github.com/demisto/download/domain"
	"github.com/demisto/download/repo"
	"github.com/demisto/download/util"
	"github.com/gorilla/context"
	"time"
)

// quizHandler returns the requested number of quiz questions
func (ac *AppContext) quizHandler(w http.ResponseWriter, r *http.Request) {
	countStr := r.FormValue("count")
	count, err := strconv.Atoi(countStr)
	if err != nil {
		WriteError(w, ErrBadRequest)
		return
	}
	// Don't let anyone retrieve all the questions
	if count > 3 || count < 1 {
		WriteError(w, ErrBadRequest)
		return
	}
	q, err := ac.r.Questions()
	if err != nil {
		log.WithError(err).Warn("Unable to serve questions")
		WriteError(w, ErrInternalServer)
		return
	}
	selected := make([]domain.Quiz, 0, count)
	for i := 0; i < count; i++ {
		qnum := rand.Intn(len(q))
		selected = append(selected, q[qnum])
		q = append(q[:qnum], q[qnum+1:]...)
	}
	writeWithFilter(w, selected, domain.QuizFilterFields...)
}

// updateQuizHandler updates the quizes
func (ac *AppContext) updateQuizHandler(w http.ResponseWriter, r *http.Request) {
	q := context.Get(r, "body").(*domain.Quiz)
	log.Infof("Updating quiz: %#v", q)
	err := ac.r.SetQuestion(q)
	if err != nil {
		log.WithError(err).Warn("Unable to save question - %#v", q)
		WriteError(w, ErrInternalServer)
		return
	}
	writeJSON(w, q)
}

type quizResponse struct {
	Token     string        `json:"token"`
	Email     string        `json:"email"`
	Questions []domain.Quiz `json:"questions"`
}

func (ac *AppContext) checkQuiz(w http.ResponseWriter, r *http.Request) {
	q := context.Get(r, "body").(*quizResponse)
	log.Infof("Getting quiz response: %#v", q)
	if q.Token == "" || q.Email == "" || len(q.Questions) != 3 {
		WriteError(w, ErrMissingPartRequest)
		return
	}
	if !govalidator.IsEmail(q.Email) {
		WriteError(w, &Error{ID: "bad_request", Status: 400, Title: "Invalid Email", Detail: "Invalid email provided"})
		return
	}
	token, err := ac.r.Token(q.Token)
	if err == repo.ErrNotFound {
		// Make sure no one is brute forcing us
		ac.preventBruteForce(r.RemoteAddr)
		WriteError(w, ErrBadRequest)
		return
	}
	if err != nil {
		log.WithError(err).Warnf("Error getting token from DB - %#v", q)
		WriteError(w, ErrInternalServer)
		return
	}
	if token.Downloads < 1 {
		WriteError(w, &Error{ID: "bad_request", Status: 400, Title: "Invalid Token", Detail: "Token is fully used and no longer allowed to download"})
		return
	}
	// Valid token, let's clear the brute force
	ac.resetBruteForce(r.RemoteAddr)
	questions, err := ac.r.Questions()
	if err != nil {
		log.WithError(err).Warn("Error getting questions from DB")
		WriteError(w, ErrInternalServer)
		return
	}
	for i := range q.Questions {
		found := false
		for j := range questions {
			if q.Questions[i].Name == questions[j].Name {
				found = true
				if !questions[j].IsCorrect(&q.Questions[i]) {
					writeJSON(w, map[string]bool{"result": false})
					return
				}
			}
		}
		if !found {
			WriteError(w, ErrBadRequest)
			return
		}
	}
	// ok, all answers are good and we have a valid token, let's save the user and create a session
	u := &domain.User{Username: q.Token + "*-*" + q.Email, Email: q.Email, Token: q.Token, Type: domain.UserTypeUser, LastLogin: time.Now()}
	err = ac.r.SetUser(u)
	if err != nil {
		log.WithError(err).Warnf("Error saving token user - %#v", u)
		WriteError(w, ErrInternalServer)
		return
	}

	sess := session{
		User: u.Username,
		When: time.Now().Unix() * 1000,
	}
	secure := conf.Options.SSL.Key != ""
	timeout := conf.Options.Security.Timeout
	val, _ := util.EncryptJSON(&sess, conf.Options.Security.SessionKey)

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

	writeJSON(w, map[string]bool{"result": true})
}
