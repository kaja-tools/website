package v1

import (
	"context"
	"fmt"
	"io"
	"strings"

	"google.golang.org/grpc"
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

// GenerateNumbers is a server streaming RPC that sends numbers 1 through req.Count.
func (s *QuirksService) GenerateNumbers(req *GenerateRequest, stream grpc.ServerStreamingServer[GenerateResponse]) error {
	for i := int32(1); i <= req.Count; i++ {
		if err := stream.Send(&GenerateResponse{Number: i}); err != nil {
			return err
		}
	}
	return nil
}

// AccumulateSum is a client streaming RPC that sums all streamed numbers.
func (s *QuirksService) AccumulateSum(stream grpc.ClientStreamingServer[AccumulateRequest, AccumulateResponse]) error {
	var total int32
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&AccumulateResponse{Total: total})
		}
		if err != nil {
			return err
		}
		total += req.Number
	}
}

// Echo is a bidirectional streaming RPC that echoes each message back with a prefix.
func (s *QuirksService) Echo(stream grpc.BidiStreamingServer[EchoMessage, EchoMessage]) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := stream.Send(&EchoMessage{Content: fmt.Sprintf("echo: %s", req.Content)}); err != nil {
			return err
		}
	}
}

// QuirksTwirpService implements the Twirp Quirks interface (unary-only).
// Twirp does not support streaming, so protoc-gen-twirp generates unary
// signatures for the streaming RPCs. This struct provides those.
type QuirksTwirpService struct{}

func (s *QuirksTwirpService) MethodWithAReallyLongNameGmthggupcbmnphflnnvu(ctx context.Context, req *Void) (*Message, error) {
	return &Message{
		Name: strings.Repeat("Ha ", 1000),
	}, nil
}

func (s *QuirksTwirpService) Sum(ctx context.Context, req *SumStringsRequest) (*SumStringsResponse, error) {
	return &SumStringsResponse{
		Result: req.A + req.B,
	}, nil
}

func (s *QuirksTwirpService) GenerateNumbers(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	return &GenerateResponse{Number: req.Count}, nil
}

func (s *QuirksTwirpService) AccumulateSum(ctx context.Context, req *AccumulateRequest) (*AccumulateResponse, error) {
	return &AccumulateResponse{Total: req.Number}, nil
}

func (s *QuirksTwirpService) Echo(ctx context.Context, req *EchoMessage) (*EchoMessage, error) {
	return &EchoMessage{Content: fmt.Sprintf("echo: %s", req.Content)}, nil
}

type Quirks_2Service struct {
	UnimplementedQuirks_2Server
}

func (s *Quirks_2Service) CamelCaseMethod(ctx context.Context, req *Void) (*Void, error) {
	return &Void{}, nil
}
