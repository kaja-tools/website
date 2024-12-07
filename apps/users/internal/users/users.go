package users

import (
	"context"

	"github.com/google/uuid"
)

var users = make(map[string]string, 100)

type UsersServer struct{}

func (s *UsersServer) GetUser(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	name := users[req.Id]

	return &GetUserResponse{Id: req.Id, Name: name}, nil
}

func (s *UsersServer) CreateUser(ctx context.Context, req *CreateUserRequest) (*CreateUserResponse, error) {
	id := uuid.New().String()
	users[id] = req.Name

	return &CreateUserResponse{Id: id, Name: users[id]}, nil
}
