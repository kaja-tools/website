package v2

import (
	"context"
)

type QuirksService struct {
	UnimplementedQuirksServer
}

func (s *QuirksService) Sum(ctx context.Context, req *SumIntsRequest) (*SumIntsResponse, error) {
	return &SumIntsResponse{
		Result: req.A + req.B,
	}, nil
}
