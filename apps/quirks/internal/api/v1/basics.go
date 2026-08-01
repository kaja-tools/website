package v1

import (
	"context"

	"github.com/twitchtv/twirp"
)

type BasicsService struct{}

func (s *BasicsService) Types(ctx context.Context, req *TypesRequest) (*TypesRequest, error) {
	return req, nil
}

func (s *BasicsService) Map(ctx context.Context, req *MapRequest) (*MapRequest, error) {
	return req, nil
}

func (s *BasicsService) Panic(ctx context.Context, req *Void) (*Message, error) {
	panic("This is broken")
}

func (s *BasicsService) Repeated(ctx context.Context, req *RepeatedRequest) (*RepeatedRequest, error) {
	return req, nil
}

func (s *BasicsService) Unauthorized(ctx context.Context, req *Void) (*Void, error) {
	return nil, twirp.NewError(twirp.PermissionDenied, "unauthorized")
}

func (s *BasicsService) Headers(ctx context.Context, req *Void) (*HeadersResponse, error) {
	headers := make(map[string]string)

	httpHeaders, _ := twirp.HTTPRequestHeaders(ctx)
	for key, values := range httpHeaders {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	return &HeadersResponse{Headers: headers}, nil
}
