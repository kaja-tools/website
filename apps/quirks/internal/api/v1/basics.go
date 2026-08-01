package v1

import (
	"context"

	"github.com/twitchtv/twirp"
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
	// Check if this is a gRPC request by looking for gRPC metadata
	if _, ok := metadata.FromIncomingContext(ctx); ok {
		return nil, status.Error(codes.PermissionDenied, "unauthorized")
	}
	// Twirp request
	return nil, twirp.NewError(twirp.PermissionDenied, "unauthorized")
}

func (s *BasicsService) Headers(ctx context.Context, req *Void) (*HeadersResponse, error) {
	headers := make(map[string]string)

	// Check if this is a gRPC request by looking for gRPC metadata
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		// gRPC request - extract metadata
		for key, values := range md {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
	} else {
		// Twirp request - extract HTTP headers
		httpHeaders, _ := twirp.HTTPRequestHeaders(ctx)
		for key, values := range httpHeaders {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
	}

	return &HeadersResponse{Headers: headers}, nil
}
