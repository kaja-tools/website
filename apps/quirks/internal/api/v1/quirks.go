package v1

import (
	"context"
	"fmt"
	"strings"
)

// QuirksService implements the Twirp Quirks interface (unary-only). Twirp does
// not support streaming, so protoc-gen-twirp generates unary signatures for the
// streaming RPCs and this struct provides those.
type QuirksService struct{}

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

func (s *QuirksService) GenerateNumbers(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	return &GenerateResponse{Number: req.Count}, nil
}

func (s *QuirksService) AccumulateSum(ctx context.Context, req *AccumulateRequest) (*AccumulateResponse, error) {
	return &AccumulateResponse{Total: req.Number}, nil
}

func (s *QuirksService) Echo(ctx context.Context, req *EchoMessage) (*EchoMessage, error) {
	return &EchoMessage{Content: fmt.Sprintf("echo: %s", req.Content)}, nil
}

type Quirks_2Service struct{}

func (s *Quirks_2Service) CamelCaseMethod(ctx context.Context, req *Void) (*Void, error) {
	return &Void{}, nil
}
