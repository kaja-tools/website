package v1

import (
	"context"
	"strings"
)

type QuirksService struct {
	UnimplementedQuirksServer
}

func (s *QuirksService) MethodWithAReallyLongNameGmthggupcbmnphflnnvu(ctx context.Context, req *Void) (*Message, error) {
	return &Message{
		Name: strings.Repeat("Ha ", 1000),
	}, nil
}

func (s *QuirksService) Sum(ctx context.Context, req *SumStringsRequest) (*SumStringsResponse, error) {
	return &SumStringsResponse{
		Result: req.A + req.B,
	}, nil
}

type Quirks_2Service struct {
	UnimplementedQuirks_2Server
}

func (s *Quirks_2Service) CamelCaseMethod(ctx context.Context, req *Void) (*Void, error) {
	return &Void{}, nil
}
