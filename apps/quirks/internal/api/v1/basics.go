package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type BasicsService struct {
	UnimplementedBasicsServer
}

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
	return nil, status.Error(codes.PermissionDenied, "unauthorized")
}

func (s *BasicsService) Headers(ctx context.Context, req *Void) (*HeadersResponse, error) {
	headers := make(map[string]string)
	md, _ := metadata.FromIncomingContext(ctx)
	for key, values := range md {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	return &HeadersResponse{Headers: headers}, nil
}
