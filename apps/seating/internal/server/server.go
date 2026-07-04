package server

import (
	"context"
	"sync"

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
	m, err := s.store.SeatMap(req.PerformanceId)
	if err != nil {
		return nil, err
	}
	return &api.GetSeatMapResponse{SeatMap: m}, nil
}

func (s *Server) PollSeatChanges(ctx context.Context, req *api.PollSeatChangesRequest) (*api.PollSeatChangesResponse, error) {
	changes, next, reset, err := s.store.Changes(req.PerformanceId, req.Cursor)
	if err != nil {
		return nil, err
	}
	return &api.PollSeatChangesResponse{Changes: changes, Cursor: next, Reset_: reset}, nil
}

func (s *Server) HoldSeats(ctx context.Context, req *api.HoldSeatsRequest) (*api.HoldSeatsResponse, error) {
	h, err := s.store.Hold(req.PerformanceId, req.SeatIds)
	if err != nil {
		return nil, err
	}
	return &api.HoldSeatsResponse{Hold: h}, nil
}

func (s *Server) ConfirmSeats(ctx context.Context, req *api.ConfirmSeatsRequest) (*api.ConfirmSeatsResponse, error) {
	seats, err := s.store.Confirm(req.HoldId)
	if err != nil {
		return nil, err
	}
	return &api.ConfirmSeatsResponse{Seats: seats}, nil
}

func (s *Server) ReleaseSeats(ctx context.Context, req *api.ReleaseSeatsRequest) (*api.ReleaseSeatsResponse, error) {
	if err := s.store.Release(req.HoldId); err != nil {
		return nil, err
	}
	return &api.ReleaseSeatsResponse{}, nil
}

func (s *Server) WatchSeats(req *api.WatchSeatsRequest, stream api.Seating_WatchSeatsServer) error {
	snapshot, ch, cancel, err := s.store.Subscribe(req.PerformanceId)
	if err != nil {
		return err
	}
	defer cancel()

	if !req.ChangesOnly {
		if err := stream.Send(&api.SeatUpdate{Update: &api.SeatUpdate_Snapshot{Snapshot: snapshot}}); err != nil {
			return err
		}
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

// PickSeats is an interactive session: the first command must be watch,
// after which hold/release/confirm commands can be interleaved with the
// live seat changes flowing back. Failed commands become Rejection frames
// instead of stream errors, so one lost race doesn't end the session.
func (s *Server) PickSeats(stream api.Seating_PickSeatsServer) error {
	var sendMu sync.Mutex
	send := func(u *api.SeatUpdate) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		return stream.Send(u)
	}

	watching := false
	perfID := ""
	done := make(chan error, 1)
	var cancelWatch func()
	defer func() {
		if cancelWatch != nil {
			cancelWatch()
		}
	}()

	for {
		cmd, err := stream.Recv()
		if err != nil {
			return nil // client closed the session
		}

		switch c := cmd.Command.(type) {
		case *api.PickSeatsCommand_Watch:
			if watching {
				if err := send(rejection("already watching " + perfID)); err != nil {
					return err
				}
				continue
			}
			snapshot, ch, cancel, err := s.store.Subscribe(c.Watch.PerformanceId)
			if err != nil {
				if err := send(rejection(status.Convert(err).Message())); err != nil {
					return err
				}
				continue
			}
			watching = true
			perfID = c.Watch.PerformanceId
			cancelWatch = cancel
			if err := send(&api.SeatUpdate{Update: &api.SeatUpdate_Snapshot{Snapshot: snapshot}}); err != nil {
				return err
			}
			go func() {
				for update := range ch {
					if err := send(update); err != nil {
						done <- err
						return
					}
				}
			}()

		case *api.PickSeatsCommand_Hold:
			if !watching {
				if err := send(rejection("send a watch command first")); err != nil {
					return err
				}
				continue
			}
			h, err := s.store.Hold(perfID, c.Hold.SeatIds)
			if err != nil {
				if err := send(rejectionSeats(status.Convert(err).Message(), c.Hold.SeatIds)); err != nil {
					return err
				}
				continue
			}
			if err := send(&api.SeatUpdate{Update: &api.SeatUpdate_Hold{Hold: h}}); err != nil {
				return err
			}

		case *api.PickSeatsCommand_Release:
			if err := s.store.Release(c.Release.HoldId); err != nil {
				if err := send(rejection(status.Convert(err).Message())); err != nil {
					return err
				}
			}

		case *api.PickSeatsCommand_Confirm:
			if _, err := s.store.Confirm(c.Confirm.HoldId); err != nil {
				if err := send(rejection(status.Convert(err).Message())); err != nil {
					return err
				}
			}

		default:
			if err := send(rejection("empty command")); err != nil {
				return err
			}
		}

		select {
		case err := <-done:
			return err
		default:
		}
	}
}

func rejection(reason string) *api.SeatUpdate {
	return &api.SeatUpdate{Update: &api.SeatUpdate_Rejection{Rejection: &api.Rejection{Reason: reason}}}
}

func rejectionSeats(reason string, seatIDs []string) *api.SeatUpdate {
	return &api.SeatUpdate{Update: &api.SeatUpdate_Rejection{Rejection: &api.Rejection{Reason: reason, SeatIds: seatIDs}}}
}
