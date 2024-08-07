package models

import (
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode"
)

// User storage user representation
type User struct {
	Id        string    `json:"id" db:"id"`
	FirstName string    `json:"first_name" db:"first_name"`
	LastName  string    `json:"last_name" db:"last_name"`
	NickName  string    `json:"nickname,omitempty" db:"nickname"`
	Password  string    `json:"password,omitempty" db:"password"`
	Email     string    `json:"email" db:"email"`
	Country   string    `json:"country" db:"country"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// WithId add ID to user
func (u *User) WithId(id string) *User {
	u.Id = id
	return u
}

// WithCreateAt add create_at field for user
func (u *User) WithCreateAt(c time.Time) *User {
	u.CreatedAt = c
	return u
}

// WithPassword add an un-hashed password to user
func (u *User) WithPassword(pwd string) *User {
	u.Password = pwd
	return u
}

// WithEmail add email to user
func (u *User) WithEmail(email string) *User {
	u.Email = email
	return u
}

func (u *User) Validate() error {
	if u.Password == "" {
		return fmt.Errorf("empty password")
	}

	if u.NickName != "" && !validateNickname(u.NickName, 1, 32) {
		return fmt.Errorf("invalid nickname")
	}
	if err := ValidateEmail(u.Email); err != nil {
		return err
	}

	return nil
}

func validateNickname(login string, minL, maxL int) bool {
	l := len(login)
	if l < minL || l > maxL {
		return false
	}
	firstL := unicode.IsLetter(rune(login[0])) || unicode.IsDigit(rune(login[0]))
	lastL := unicode.IsLetter(rune(login[l-1])) || unicode.IsDigit(rune(login[l-1]))

	if !firstL || !lastL {
		return false
	}
	const specialCharSet = "._-"
	for i, r := range login {
		if strings.ContainsRune(specialCharSet, r) {
			// two special chars in a raw is restricted
			if i+1 <= l-1 && strings.ContainsRune(specialCharSet, rune(login[i+1])) {
				return false
			}
		} else if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("empty email")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("email malformed '%s': %w", email, err)
	}
	return nil
}
