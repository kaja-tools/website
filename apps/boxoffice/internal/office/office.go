// Package office implements the BoxOffice Twirp service. Reservations and
// orders live in memory; the seats themselves are held and sold through the
// seating gRPC service, the same way the crowd and kaja users do it.
package office

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kaja-tools/website/v2/internal/api"
	"github.com/kaja-tools/website/v2/internal/theatre"
	"github.com/twitchtv/twirp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	maxCustomerName = 40
	// Expired and canceled reservations are kept around briefly so that a
	// late Purchase gets a helpful error instead of "not found".
	reservationGrace = 30 * time.Minute
)

type reservation struct {
	id         string
	holdID     string
	perfID     string
	eventID    string
	eventTitle string
	customer   string
	seatIDs    []string
	status     api.ReservationStatus
	total      int32
	expires    time.Time
}

type Office struct {
	seating          api.SeatingClient
	theatre          *theatre.Client
	publicTheatreURL string

	mu           sync.Mutex
	reservations map[string]*reservation
	orders       map[string]*api.Order
}

func New(seating api.SeatingClient, theatreClient *theatre.Client, publicTheatreURL string) *Office {
	return &Office{
		seating:          seating,
		theatre:          theatreClient,
		publicTheatreURL: publicTheatreURL,
		reservations:     map[string]*reservation{},
		orders:           map[string]*api.Order{},
	}
}

func (o *Office) Reserve(ctx context.Context, req *api.ReserveRequest) (*api.ReserveResponse, error) {
	customer := strings.TrimSpace(req.CustomerName)
	if customer == "" {
		return nil, twirp.RequiredArgumentError("customer_name")
	}
	if len(customer) > maxCustomerName {
		return nil, twirp.InvalidArgumentError("customer_name", fmt.Sprintf("at most %d characters", maxCustomerName))
	}

	perf, err := o.theatre.Performance(req.PerformanceId)
	if err != nil {
		return nil, err
	}

	holdResp, err := o.seating.HoldSeats(ctx, &api.HoldSeatsRequest{
		PerformanceId: req.PerformanceId,
		SeatIds:       req.SeatIds,
	})
	if err != nil {
		return nil, fromGRPC(err)
	}
	hold := holdResp.Hold

	r := &reservation{
		id:         "res_" + randomID(4),
		holdID:     hold.Id,
		perfID:     req.PerformanceId,
		eventID:    perf.EventID,
		eventTitle: perf.EventTitle,
		customer:   customer,
		seatIDs:    hold.SeatIds,
		status:     api.ReservationStatus_RESERVATION_STATUS_PENDING,
		total:      hold.TotalPriceCents,
		expires:    hold.ExpiresAt.AsTime(),
	}

	o.mu.Lock()
	o.reservations[r.id] = r
	o.mu.Unlock()

	time.AfterFunc(time.Until(r.expires), func() { o.markExpired(r.id) })
	time.AfterFunc(time.Until(r.expires)+reservationGrace, func() { o.forget(r.id) })

	return &api.ReserveResponse{Reservation: reservationProto(r)}, nil
}

func (o *Office) Purchase(ctx context.Context, req *api.PurchaseRequest) (*api.PurchaseResponse, error) {
	o.mu.Lock()
	r, ok := o.reservations[req.ReservationId]
	o.mu.Unlock()
	if !ok {
		return nil, twirp.NotFoundError(fmt.Sprintf("reservation %q not found", req.ReservationId))
	}

	switch r.status {
	case api.ReservationStatus_RESERVATION_STATUS_PURCHASED:
		return nil, twirp.NewError(twirp.FailedPrecondition, "this reservation was already purchased")
	case api.ReservationStatus_RESERVATION_STATUS_CANCELED:
		return nil, twirp.NewError(twirp.FailedPrecondition, "this reservation was canceled")
	case api.ReservationStatus_RESERVATION_STATUS_EXPIRED:
		return nil, twirp.NewError(twirp.FailedPrecondition, "this reservation expired — the seats went back on sale")
	}

	if err := checkPayment(req); err != nil {
		return nil, err
	}

	confirmResp, err := o.seating.ConfirmSeats(ctx, &api.ConfirmSeatsRequest{HoldId: r.holdID})
	if err != nil {
		return nil, fromGRPC(err)
	}

	order := &api.Order{
		ConfirmationCode: "KAJA-" + strings.ToUpper(randomID(2)),
		PerformanceId:    r.perfID,
		EventId:          r.eventID,
		EventTitle:       r.eventTitle,
		EventUrl:         fmt.Sprintf("%s/events/%s", o.publicTheatreURL, r.eventID),
		CustomerName:     r.customer,
		TotalCents:       r.total,
		PurchasedAt:      timestamppb.Now(),
	}
	for _, seat := range confirmResp.Seats {
		order.Tickets = append(order.Tickets, &api.Ticket{
			SeatId:     seat.Id,
			Section:    strings.TrimPrefix(seat.Section.String(), "SECTION_"),
			PriceCents: seat.PriceCents,
		})
	}

	o.mu.Lock()
	r.status = api.ReservationStatus_RESERVATION_STATUS_PURCHASED
	o.orders[order.ConfirmationCode] = order
	o.mu.Unlock()

	return &api.PurchaseResponse{Order: order}, nil
}

func (o *Office) CancelReservation(ctx context.Context, req *api.CancelReservationRequest) (*api.CancelReservationResponse, error) {
	o.mu.Lock()
	r, ok := o.reservations[req.ReservationId]
	o.mu.Unlock()
	if !ok {
		return nil, twirp.NotFoundError(fmt.Sprintf("reservation %q not found", req.ReservationId))
	}
	if r.status != api.ReservationStatus_RESERVATION_STATUS_PENDING {
		return nil, twirp.NewError(twirp.FailedPrecondition, "only pending reservations can be canceled")
	}

	if _, err := o.seating.ReleaseSeats(ctx, &api.ReleaseSeatsRequest{HoldId: r.holdID}); err != nil {
		if status.Code(err) != codes.NotFound { // already expired server-side is fine
			return nil, fromGRPC(err)
		}
	}

	o.mu.Lock()
	r.status = api.ReservationStatus_RESERVATION_STATUS_CANCELED
	o.mu.Unlock()

	return &api.CancelReservationResponse{}, nil
}

func (o *Office) GetOrder(ctx context.Context, req *api.GetOrderRequest) (*api.GetOrderResponse, error) {
	o.mu.Lock()
	order, ok := o.orders[strings.ToUpper(strings.TrimSpace(req.ConfirmationCode))]
	o.mu.Unlock()
	if !ok {
		return nil, twirp.NotFoundError(fmt.Sprintf("no order %q", req.ConfirmationCode))
	}
	return &api.GetOrderResponse{Order: order}, nil
}

func checkPayment(req *api.PurchaseRequest) error {
	switch payment := req.Payment.(type) {
	case *api.PurchaseRequest_Card:
		number := strings.ReplaceAll(payment.Card.Number, " ", "")
		if number != "4242424242424242" {
			return twirp.NewError(twirp.InvalidArgument,
				"card declined (this is a demo — 4242 4242 4242 4242 always works)")
		}
		expiry, err := time.Parse("01/06", payment.Card.Expiry)
		if err != nil {
			return twirp.InvalidArgumentError("expiry", "must be formatted MM/YY")
		}
		// Cards are valid through the end of their expiry month.
		if expiry.AddDate(0, 1, 0).Before(time.Now()) {
			return twirp.NewError(twirp.InvalidArgument, "card expired")
		}
	case *api.PurchaseRequest_GiftCode:
		if !strings.EqualFold(payment.GiftCode.Code, "KAJA") {
			return twirp.NewError(twirp.InvalidArgument,
				`unknown gift code (this is a demo — try "KAJA")`)
		}
	default:
		return twirp.NewError(twirp.InvalidArgument, "payment required: pay by card or gift_code")
	}
	return nil
}

func (o *Office) markExpired(id string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if r, ok := o.reservations[id]; ok && r.status == api.ReservationStatus_RESERVATION_STATUS_PENDING {
		r.status = api.ReservationStatus_RESERVATION_STATUS_EXPIRED
	}
}

func (o *Office) forget(id string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.reservations, id)
}

func reservationProto(r *reservation) *api.Reservation {
	return &api.Reservation{
		Id:            r.id,
		PerformanceId: r.perfID,
		EventTitle:    r.eventTitle,
		SeatIds:       r.seatIDs,
		CustomerName:  r.customer,
		Status:        r.status,
		TotalCents:    r.total,
		ExpiresAt:     timestamppb.New(r.expires),
	}
}

// fromGRPC translates a seating service error into the equivalent Twirp
// error so callers see one consistent error model.
func fromGRPC(err error) error {
	st := status.Convert(err)
	code := twirp.Internal
	switch st.Code() {
	case codes.AlreadyExists:
		code = twirp.AlreadyExists
	case codes.NotFound:
		code = twirp.NotFound
	case codes.InvalidArgument:
		code = twirp.InvalidArgument
	case codes.FailedPrecondition:
		code = twirp.FailedPrecondition
	case codes.Unavailable:
		code = twirp.Unavailable
	}
	return twirp.NewError(code, st.Message())
}

func randomID(bytes int) string {
	b := make([]byte, bytes)
	rand.Read(b)
	return hex.EncodeToString(b)
}
