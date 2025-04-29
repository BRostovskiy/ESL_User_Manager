package service

import (
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode"
)

const (
	UpdateField = ""
)

// User storage user representation
type User struct {
	ID        string
	FirstName string
	LastName  string
	NickName  string
	Password  string
	Email     string
	Country   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// WithID add ID to user
func (u *User) WithID(id string) *User {
	u.ID = id
	return u
}

// WithFirstName add/change first name to user
func (u *User) WithFirstName(firstName string) *User {
	u.FirstName = firstName
	return u
}

// WithLastName add last name to user
func (u *User) WithLastName(lastName string) *User {
	u.LastName = lastName
	return u
}

// WithNickName add/change user nickname
func (u *User) WithNickName(nickName string) *User {
	u.NickName = nickName
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

func (u *User) WithCountry(c string) *User {
	u.Country = c
	return u
}

func (u *User) WithCreateAt(c time.Time) *User {
	u.CreatedAt = c
	return u
}

func (u *User) Validate(passwordOkEmpty bool) error {
	if u.FirstName == "" {
		return fmt.Errorf("empty first name")
	}
	if u.LastName == "" {
		return fmt.Errorf("empty last name")
	}
	if u.Country == "" {
		return fmt.Errorf("empty country")
	}
	if u.FirstName == "" {
		return fmt.Errorf("empty first name")
	}
	if u.Password == "" && !passwordOkEmpty {
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
