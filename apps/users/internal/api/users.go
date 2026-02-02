package api

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
	if req.Name == "" {
		return nil, twirp.InvalidArgumentError("name", "cannot be empty")
	}

	id := uuid.New().String()
	user := model.User{
		ID:   id,
		Name: req.Name,
	}

	if err := h.model.Set(&user); err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return &CreateUserResponse{Message: "User created. Next, you can retrieve it with GetUser().", User: modelUserToApiUser(&user)}, nil
}

func (h *UsersHandler) GetUser(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	if req.Id == "" {
		return nil, twirp.InvalidArgumentError("id", "cannot be empty")
	}

	userResult, err := h.model.Get(req.Id)

	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	if !userResult.Found {
		return nil, twirp.NotFoundError("user not found")
	}

	return &GetUserResponse{
		User: modelUserToApiUser(userResult.User),
	}, nil
}

func (u *UsersHandler) UpdateUser(ctx context.Context, req *UpdateUserRequest) (*UpdateUserResponse, error) {
	if req.Id == "" {
		return nil, twirp.InvalidArgumentError("id", "cannot be empty")
	}
	if req.Name == "" {
		return nil, twirp.InvalidArgumentError("name", "cannot be empty")
	}

	userResult, err := u.model.Get(req.Id)

	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	if !userResult.Found {
		return nil, twirp.NotFoundError("user not found")
	}

	userResult.User.Name = req.Name

	if err := u.model.Set(userResult.User); err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return &UpdateUserResponse{User: modelUserToApiUser(userResult.User)}, nil
}

func (u *UsersHandler) DeleteUser(ctx context.Context, req *DeleteUserRequest) (*DeleteUserResponse, error) {
	if req.Id == "" {
		return nil, twirp.InvalidArgumentError("id", "cannot be empty")
	}

	userResult, err := u.model.Get(req.Id)

	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	if !userResult.Found {
		return nil, twirp.NotFoundError("user not found")
	}

	if err := u.model.Delete(req.Id); err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return &DeleteUserResponse{}, nil
}

func (u *UsersHandler) GetAllUsers(ctx context.Context, req *GetAllUsersRequest) (*GetAllUsersResponse, error) {
	users, err := u.model.GetAll()

	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	apiUsers := make([]*User, len(users))
	for i, user := range users {
		apiUsers[i] = modelUserToApiUser(user)
	}

	return &GetAllUsersResponse{Users: apiUsers}, nil
}

func (u *UsersHandler) DeleteAllUsers(ctx context.Context, req *DeleteAllUsersRequest) (*DeleteAllUsersResponse, error) {
	if err := u.model.DeleteAll(); err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return &DeleteAllUsersResponse{Message: "All users deleted. You can create new user with CreateUser()."}, nil
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
