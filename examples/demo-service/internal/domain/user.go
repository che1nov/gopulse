package domain

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalidUser = errors.New("invalid user")

type User struct {
	ID        string
	Name      string
	Email     string
	CreatedAt time.Time
}

func NewUser(id, name, email string, now time.Time) (User, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(strings.ToLower(email))
	if id == "" || name == "" || !strings.Contains(email, "@") {
		return User{}, ErrInvalidUser
	}

	return User{
		ID:        id,
		Name:      name,
		Email:     email,
		CreatedAt: now.UTC(),
	}, nil
}
