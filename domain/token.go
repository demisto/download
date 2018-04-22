package domain

import "github.com/demisto/download/util"

type Token struct {
	Name      string `json:"name"`
	Downloads int    `json:"downloads"`
}

// NewToken with the given number of downloads
func NewToken(downloads int) *Token {
	return &Token{Name: util.SecureRandomString(12, true), Downloads: downloads}
}