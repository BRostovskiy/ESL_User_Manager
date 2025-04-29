package pg

import (
	"time"
)

// User storage user representation
type User struct {
	Id        string
	FirstName string
	LastName  string
	NickName  string
	Password  string
	Email     string
	Country   string
	CreatedAt time.Time
	UpdatedAt time.Time
}
