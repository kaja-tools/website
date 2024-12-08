package users

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Internal domain type
type UserDB struct {
	ID   string
	Name string
	// Add other relevant fields
}

var users = make(map[string]UserDB, 100)

type UsersServer struct{}

func (s *UsersServer) CreateUser(ctx context.Context, req *CreateUserRequest) (*CreateUserResponse, error) {
	id := uuid.New().String()

	// Convert proto type to domain type
	user := UserDB{
		ID:   id,
		Name: req.User.Name,
		// Map other fields
	}
	users[id] = user

	return &CreateUserResponse{Id: id}, nil
}

func (s *UsersServer) GetUser(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	user, exists := users[req.Id]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	u := User{
		Name: user.Name,
	}

	// Convert domain type to proto type
	return &GetUserResponse{
		User: &u,
	}, nil
}

func (s *UsersServer) UpdateUser(ctx context.Context, req *UpdateUserRequest) (*UpdateUserResponse, error) {
	user, exists := users[req.Id]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	user.Name = req.User.Name
	users[req.Id] = user

	return &UpdateUserResponse{}, nil
}

func (s *UsersServer) DeleteUser(ctx context.Context, req *DeleteUserRequest) (*DeleteUserResponse, error) {
	_, exists := users[req.Id]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	delete(users, req.Id)
	return &DeleteUserResponse{}, nil
}
