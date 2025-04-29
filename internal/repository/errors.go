package repository

import (
	"fmt"
)

var (
	// NoUsersFoundError causes when DB could not find user with criteria
	NoUsersFoundError = fmt.Errorf("no users found")
	// DuplicateKeyError causes when Create or Update performed on already created items
	DuplicateKeyError = fmt.Errorf("duplicate key value violates unique constraint")
	// HashingPwdError causes when user password could not been hashed
	GeneratePwdError = fmt.Errorf("could not generate hashed user password")
)
