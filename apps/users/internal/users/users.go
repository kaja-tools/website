package users

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/kaja-tools/website/v2/internal/model"
	"github.com/twitchtv/twirp"
)

type UsersHandler struct {
	model *model.Users
}

func NewUsersHandler(model *model.Users) *UsersHandler {
	return &UsersHandler{model: model}
}

func (h *UsersHandler) CreateUser(ctx context.Context, req *CreateUserRequest) (*CreateUserResponse, error) {
	id := uuid.New().String()
	user := model.User{
		ID:   id,
		Name: req.Name,
	}

	if err := h.model.Set(&user); err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return &CreateUserResponse{User: modelUserToApiUser(&user)}, nil
}

func (h *UsersHandler) GetUser(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	user, err := h.model.Get(req.Id)

	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return &GetUserResponse{
		User: modelUserToApiUser(user),
	}, nil
}

func (u *UsersHandler) UpdateUser(ctx context.Context, req *UpdateUserRequest) (*UpdateUserResponse, error) {
	user, err := u.model.Get(req.Id)

	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	user.Name = req.Name

	if err := u.model.Set(user); err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return &UpdateUserResponse{User: modelUserToApiUser(user)}, nil
}

func (u *UsersHandler) DeleteUser(ctx context.Context, req *DeleteUserRequest) (*DeleteUserResponse, error) {
	if err := u.model.Delete(req.Id); err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return &DeleteUserResponse{}, nil
}

func NewLoggingServerHooks() *twirp.ServerHooks {
	return &twirp.ServerHooks{
		RequestRouted: func(ctx context.Context) (context.Context, error) {
			method, _ := twirp.MethodName(ctx)
			log.Println("Method: " + method)
			return ctx, nil
		},
		Error: func(ctx context.Context, twerr twirp.Error) context.Context {
			log.Println("Error: " + string(twerr.Code()))
			return ctx
		},
		ResponseSent: func(ctx context.Context) {
			log.Println("Response Sent (error or success)")
		},
	}
}

func modelUserToApiUser(user *model.User) *User {
	return &User{
		Id:   user.ID,
		Name: user.Name,
	}
}
