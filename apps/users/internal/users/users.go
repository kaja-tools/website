package users

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cockroachdb/pebble"
	"github.com/google/uuid"
)

// Internal domain type
type UserDB struct {
	ID   string
	Name string
	// Add other relevant fields
}

type UsersServer struct {
	db *pebble.DB
}

// New constructor function
func NewUsersServerPebble(dbPath string) (*UsersServer, error) {
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to open pebble db: %w", err)
	}
	return &UsersServer{db: db}, nil
}

func (s *UsersServer) CreateUser(ctx context.Context, req *CreateUserRequest) (*CreateUserResponse, error) {
	id := uuid.New().String()
	user := UserDB{
		ID:   id,
		Name: req.User.Name,
	}

	// Serialize user to JSON
	userData, err := json.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user: %w", err)
	}

	// Store in Pebble DB
	if err := s.db.Set([]byte(id), userData, pebble.Sync); err != nil {
		return nil, fmt.Errorf("failed to store user: %w", err)
	}

	return &CreateUserResponse{Id: id}, nil
}

func (s *UsersServer) GetUser(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	// Get from Pebble DB
	value, closer, err := s.db.Get([]byte(req.Id))
	if err == pebble.ErrNotFound {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	defer closer.Close()

	var user UserDB
	if err := json.Unmarshal(value, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return &GetUserResponse{
		User: &User{Name: user.Name},
	}, nil
}

func (s *UsersServer) UpdateUser(ctx context.Context, req *UpdateUserRequest) (*UpdateUserResponse, error) {
	// Get existing user
	value, closer, err := s.db.Get([]byte(req.Id))
	if err == pebble.ErrNotFound {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	closer.Close()

	var user UserDB
	if err := json.Unmarshal(value, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	// Update user
	user.Name = req.User.Name
	userData, err := json.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user: %w", err)
	}

	if err := s.db.Set([]byte(req.Id), userData, pebble.Sync); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &UpdateUserResponse{}, nil
}

func (s *UsersServer) DeleteUser(ctx context.Context, req *DeleteUserRequest) (*DeleteUserResponse, error) {
	if err := s.db.Delete([]byte(req.Id), pebble.Sync); err != nil {
		return nil, fmt.Errorf("failed to delete user: %w", err)
	}
	return &DeleteUserResponse{}, nil
}
