package users

import "context"

type UsersServer struct{}

func (s *UsersServer) GetUser(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	return &GetUserResponse{Id: req.Id, Name: "John Doe"}, nil
}
