package ratelimit

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/kaja-tools/website/v2/internal/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestTakeSpendsTheWindowAndRefusesTheRest(t *testing.T) {
	now := time.Now()
	l := New(3, time.Minute)
	l.now = func() time.Time { return now }

	for i := 1; i <= 3; i++ {
		result := l.Take("a")
		if !result.Allowed {
			t.Fatalf("call %d refused, want allowed", i)
		}
		if result.Remaining != 3-i {
			t.Fatalf("call %d: remaining %d, want %d", i, result.Remaining, 3-i)
		}
	}

	refused := l.Take("a")
	if refused.Allowed || refused.Remaining != 0 {
		t.Fatalf("fourth call: %+v, want refused with nothing remaining", refused)
	}
	if fresh := l.Take("b"); !fresh.Allowed {
		t.Fatal("another client was refused from the first client's window")
	}

	now = now.Add(time.Minute)
	if turned := l.Take("a"); !turned.Allowed || turned.Remaining != 2 {
		t.Fatalf("after the window turned over: %+v, want a full budget", turned)
	}
}

func TestRefusalIsNotCountedAgainstTheWindow(t *testing.T) {
	now := time.Now()
	l := New(1, time.Minute)
	l.now = func() time.Time { return now }

	l.Take("a")
	for i := 0; i < 5; i++ {
		l.Take("a")
	}

	now = now.Add(time.Minute)
	if turned := l.Take("a"); !turned.Allowed {
		t.Fatal("refused calls extended the window they were refused by")
	}
}

func TestExpiredClientsAreSwept(t *testing.T) {
	now := time.Now()
	l := New(1, time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < sweepAt; i++ {
		l.Take(string(rune(i)))
	}
	now = now.Add(2 * time.Minute)
	l.Take("last")

	if len(l.clients) != 1 {
		t.Fatalf("kept %d clients after the sweep, want 1", len(l.clients))
	}
}

func TestHeadersSayTheBudgetBothWays(t *testing.T) {
	headers := Result{Allowed: true, Limit: 120, Remaining: 57, ResetAfter: 11500 * time.Millisecond, Window: 30 * time.Second}.Headers()

	want := map[string]string{
		"ratelimit-policy":    `"default";q=120;w=30`,
		"ratelimit":           `"default";r=57;t=12`,
		"ratelimit-limit":     "120",
		"ratelimit-remaining": "57",
		"ratelimit-reset":     "12",
	}
	for name, value := range want {
		if headers[name] != value {
			t.Errorf("%s = %q, want %q", name, headers[name], value)
		}
	}
	if _, ok := headers["retry-after"]; ok {
		t.Error("an allowed call carries retry-after")
	}

	refused := Result{Limit: 120, ResetAfter: 9 * time.Second, Window: 30 * time.Second}.Headers()
	if refused["retry-after"] != "9" {
		t.Errorf("retry-after = %q, want %q", refused["retry-after"], "9")
	}
}

func TestClientOfPrefersTheProxysHeader(t *testing.T) {
	md := metadata.Pairs("fly-client-ip", "203.0.113.7", "x-forwarded-for", "198.51.100.1, 203.0.113.7")
	if got := ClientOf(metadata.NewIncomingContext(context.Background(), md)); got != "203.0.113.7" {
		t.Errorf("client = %q, want the fly-client-ip", got)
	}

	forwarded := metadata.Pairs("x-forwarded-for", "198.51.100.1, 203.0.113.7")
	if got := ClientOf(metadata.NewIncomingContext(context.Background(), forwarded)); got != "198.51.100.1" {
		t.Errorf("client = %q, want the first entry of the chain", got)
	}
}

// The headers only matter if a client gets them, and a refused gRPC call is
// exactly where that is easy to lose: the status is sent without a handler ever
// running.
func TestARefusedCallStillCarriesTheHeaders(t *testing.T) {
	listener := bufconn.Listen(1 << 20)
	limiter := New(1, time.Minute)
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(UnaryInterceptor(limiter)))
	api.RegisterSeatingServer(server, unimplemented{})
	go server.Serve(listener)
	defer server.Stop()

	conn, err := grpc.NewClient("passthrough:bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := api.NewSeatingClient(conn)
	client.GetSeatMap(context.Background(), &api.GetSeatMapRequest{})

	var header, trailer metadata.MD
	_, err = client.GetSeatMap(context.Background(), &api.GetSeatMapRequest{}, grpc.Header(&header), grpc.Trailer(&trailer))
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("second call: %v, want RESOURCE_EXHAUSTED", err)
	}

	got := func(name string) string {
		if values := header.Get(name); len(values) > 0 {
			return values[0]
		}
		if values := trailer.Get(name); len(values) > 0 {
			return values[0]
		}
		return ""
	}
	if got("ratelimit-remaining") != "0" {
		t.Errorf("ratelimit-remaining = %q, want %q", got("ratelimit-remaining"), "0")
	}
	if !strings.HasPrefix(got("ratelimit"), `"default";r=0;`) {
		t.Errorf("ratelimit = %q, want the structured field with nothing remaining", got("ratelimit"))
	}
	if got("retry-after") == "" {
		t.Error("a refusal said nothing about how long to wait")
	}
}

type unimplemented struct {
	api.UnimplementedSeatingServer
}
