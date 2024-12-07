package users

import (
	"context"

	"github.com/google/uuid"
)

type UsersServer struct{}

func (s *UsersServer) GetUser(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	return &GetUserResponse{Id: req.Id, Name: "John Doe"}, nil
}

func (s *UsersServer) CreateUser(ctx context.Context, req *CreateUserRequest) (*CreateUserResponse, error) {
	return &CreateUserResponse{Id: uuid.New().String(), Name: req.Name}, nil
}
