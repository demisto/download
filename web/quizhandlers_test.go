package web

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/demisto/download/domain"
	"encoding/json"
	"bytes"
)

func TestCheckQuiz(t *testing.T) {
	f := newHandlerFixture(t)
	defer f.Close()
	f.r.SetToken(&domain.Token{Name:"t", Downloads: 10})
	f.r.SetQuestion(&domain.Quiz{Name: "1", Correct:[]int{0}})
	f.r.SetQuestion(&domain.Quiz{Name: "2", Correct:[]int{1}})
	f.r.SetQuestion(&domain.Quiz{Name: "3", Correct:[]int{2, 3}})
	qr := quizResponse{Token:"t", Email:"aaa@bbb.com", Questions: []domain.Quiz{
		{Name:"1", Correct:[]int{0}}, {Name:"2", Correct:[]int{1}}, {Name:"3", Correct:[]int{2, 3}}},
	}
	b, _ := json.Marshal(qr)
	req, err := http.NewRequest("POST", "http://demisto.com/check", bytes.NewBuffer(b))
	if err != nil {
		t.Fatal(err)
	}
	f.sendRequest(req, false, "")
	if f.response.Code != http.StatusOK {
		t.Fatalf("Did not receive the correct status - %v", f.response.Code)
	}
	assert.Contains(t, f.response.Header().Get("Set-Cookie"), "SD=")
}

func TestCheckQuizBruteForce(t *testing.T) {
	f := newHandlerFixture(t)
	defer f.Close()
	qr := quizResponse{Token:"none", Email:"aaa@bbb.com", Questions: []domain.Quiz{{Name:"1"}, {Name:"2"}, {Name:"3"}}}
	b, _ := json.Marshal(qr)
	req, _ := http.NewRequest("POST", "http://demisto.com/check", bytes.NewBuffer(b))
	f.sendRequest(req, false, "")
	assert.EqualValues(t, 1, bruteForceMap.Len(), "brute force map not updated")
}
