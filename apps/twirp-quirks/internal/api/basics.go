package api

import (
	"context"
)

type BasicsHandler struct{}

func NewBasicsHandler() *BasicsHandler {
	return &BasicsHandler{}
}

func (h *BasicsHandler) Types(ctx context.Context, req *TypesRequest) (*TypesRequest, error) {
	return req, nil
}

func (h *BasicsHandler) Map(ctx context.Context, req *MapRequest) (*MapRequest, error) {
	return req, nil
}

func (h *BasicsHandler) Panic(ctx context.Context, req *Void) (*Message, error) {
	panic("This is broken")
}

func (h *BasicsHandler) Repeated(ctx context.Context, req *RepeatedRequest) (*RepeatedRequest, error) {
	return req, nil
}
