package domain

import (
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/demisto/download/util"
)

type Quiz struct {
	Name     string   `json:"name"`
	Question string   `json:"question"`
	Answers  []string `json:"answers"`
	Correct  []int    `json:"correct"`
}

// QuizFilterFields is the list of fields we should filter when sending to clients
var QuizFilterFields = []string{"right"}

const qseq = "*|*"

func AnswersFromString(answers string) []string {
	return strings.Split(answers, qseq)
}

func AnswersToString(answers []string) string {
	return strings.Join(answers, qseq)
}

func CorrectFromString(correct string) []int {
	var c []int
	cs := strings.Split(correct, qseq)
	for _, csi := range cs {
		i, err := strconv.Atoi(csi)
		if err == nil {
			c = append(c, i)
		} else {
			log.WithError(err).Warnf("Unable to parse correct answer - %s", csi)
		}
	}
	return c
}

func CorrectToString(correct []int) string {
	cs := make([]string, len(correct))
	for i, c := range correct {
		cs[i] = strconv.Itoa(c)
	}
	return strings.Join(cs, qseq)
}

func (q *Quiz) IsCorrect(q1 *Quiz) bool {
	for _, correct := range q.Correct {
		if !util.In(q1.Correct, correct) {
			return false
		}
	}
	return len(q.Correct) == len(q1.Correct)
}
