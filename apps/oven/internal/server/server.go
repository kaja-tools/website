package server

import (
	"context"

	"github.com/kaja-tools/website/v2/internal/api"
	"github.com/kaja-tools/website/v2/internal/store"
)

// Whoever is calling the API is "you" on the rack. The bakery's own staff
// bake under their own names.
const visitor = "you"

type Server struct {
	api.UnimplementedOvenServer
	store *store.Store
}

func New(s *store.Store) *Server {
	return &Server{store: s}
}

func (s *Server) GetOven(ctx context.Context, req *api.GetOvenRequest) (*api.GetOvenResponse, error) {
	return &api.GetOvenResponse{Oven: s.store.Oven()}, nil
}

func (s *Server) ClaimRack(ctx context.Context, req *api.ClaimRackRequest) (*api.ClaimRackResponse, error) {
	claim, err := s.store.Claim(req.CookieId, req.Trays, visitor)
	if err != nil {
		return nil, err
	}
	return &api.ClaimRackResponse{Claim: claim}, nil
}

func (s *Server) StartBake(ctx context.Context, req *api.StartBakeRequest) (*api.StartBakeResponse, error) {
	batch, err := s.store.Start(req.ClaimId)
	if err != nil {
		return nil, err
	}
	return &api.StartBakeResponse{Batch: batch}, nil
}

func (s *Server) WatchOven(req *api.WatchOvenRequest, stream api.Oven_WatchOvenServer) error {
	snapshot, ch, cancel := s.store.Subscribe()
	defer cancel()

	if err := stream.Send(&api.OvenUpdate{Update: &api.OvenUpdate_Snapshot{Snapshot: snapshot}}); err != nil {
		return err
	}

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case update, ok := <-ch:
			if !ok {
				return nil // dropped for falling behind
			}
			if err := stream.Send(update); err != nil {
				return err
			}
		}
	}
}
