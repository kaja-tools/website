package server

import (
	"context"

	"github.com/kaja-tools/website/v2/internal/api"
	"github.com/kaja-tools/website/v2/internal/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	api.UnimplementedSeatingServer
	store *store.Store
}

func New(s *store.Store) *Server {
	return &Server{store: s}
}

func (s *Server) GetSeatMap(ctx context.Context, req *api.GetSeatMapRequest) (*api.GetSeatMapResponse, error) {
	m, err := s.store.SeatMap(req.ShowId)
	if err != nil {
		return nil, err
	}
	return &api.GetSeatMapResponse{SeatMap: m}, nil
}

func (s *Server) BookSeats(ctx context.Context, req *api.BookSeatsRequest) (*api.BookSeatsResponse, error) {
	return s.store.Book(req.ShowId, req.SeatIds)
}

func (s *Server) WatchSeats(req *api.WatchSeatsRequest, stream api.Seating_WatchSeatsServer) error {
	snapshot, ch, cancel, err := s.store.Subscribe(req.ShowId)
	if err != nil {
		return err
	}
	defer cancel()

	if err := stream.Send(&api.SeatUpdate{Update: &api.SeatUpdate_Snapshot{Snapshot: snapshot}}); err != nil {
		return err
	}

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case update, ok := <-ch:
			if !ok {
				return status.Error(codes.ResourceExhausted, "stream fell too far behind")
			}
			if err := stream.Send(update); err != nil {
				return err
			}
		}
	}
}
