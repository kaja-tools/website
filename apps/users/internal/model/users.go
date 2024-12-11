package model

import (
	"encoding/json"
	"fmt"

	"github.com/cockroachdb/pebble"
)

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type UserResult struct {
	User *User
	Found bool
}

type Users struct {
	db *pebble.DB
}

func NewUsers(db *pebble.DB) *Users {
	return &Users{db: db}
}

func (u *Users) Set(user *User) error {
	value, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}
	return u.db.Set([]byte(user.ID), value, pebble.Sync)
}

func (u *Users) Get(id string) (*UserResult, error) {
	value, closer, err := u.db.Get([]byte(id))
	if err != nil {
		if err == pebble.ErrNotFound {
			return &UserResult{Found: false}, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	defer closer.Close()

	var user User
	if err := json.Unmarshal(value, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}
	return &UserResult{User: &user, Found: true}, nil
}

func (u *Users) Delete(id string) error {
	return u.db.Delete([]byte(id), pebble.Sync)
}
