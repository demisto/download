package domain

import (
	"encoding/base64"
	"time"

	"fmt"
	"golang.org/x/crypto/bcrypt"
)

type UserType int

const (
	// UserTypeAdmin - can upload content, generate tokens and basically do anything
	UserTypeAdmin = iota
	// Customer that can download content with relevant token or with credentials
	UserTypeUser
)

// Stringer implementation
func (s UserType) String() string {
	switch s {
	case UserTypeAdmin:
		return "Admin"
	case UserTypeUser:
		return "Customer"
	default:
		return "Unknown"
	}
}

// User holds information about a user within the system.
// A user has a role for each project.
type User struct {
	Username   string    `json:"username"`
	Hash       string    `json:"hash"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	Type       UserType  `json:"type"`
	LastLogin  time.Time `json:"lastLogin" db:"last_login"`
	Token      string    `json:"token"`
	ModifyDate time.Time `json:"modifyDate" db:"modify_date"`
}

// GetHashFromPassword returns the hash based on bcrypt
func GetHashFromPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return base64.StdEncoding.EncodeToString(hash)
}

// SetPassword sets the password on the user with bcrypt
func (u *User) SetPassword(password string) {
	u.Hash = GetHashFromPassword(password)
}

func (u *User) UsernameForToken() string {
	return fmt.Sprintf("%d-%s", u.Token, u.Email)
}

// UserFilterFields is the list of fields we should filter when sending to clients
var UserFilterFields = []string{"hash"}
