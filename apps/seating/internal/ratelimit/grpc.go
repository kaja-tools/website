package ratelimit

import (
	"context"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// The budget is refused as RESOURCE_EXHAUSTED, which is the gRPC status for a
// rate limit and what a client reads as one — the same distinction HTTP 429 is.
const refusal = "rate limit exceeded"

// UnaryInterceptor spends one call of the caller's budget and reports the
// budget on the response either way. The headers ride on a refusal too, which
// is when they are worth the most: they say how long to wait.
func UnaryInterceptor(l *Limiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		result := l.Take(ClientOf(ctx))
		grpc.SetHeader(ctx, metadata.New(result.Headers()))
		if !result.Allowed {
			return nil, status.Error(codes.ResourceExhausted, refusal)
		}
		return handler(ctx, req)
	}
}

// StreamInterceptor is the same for a stream, which costs one call however long
// it stays open. Reflection is exempt: it is how a client learns what the
// service is, and a spent budget must not leave a caller unable to ask.
func StreamInterceptor(l *Limiter) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if strings.HasPrefix(info.FullMethod, "/grpc.reflection.") {
			return handler(srv, ss)
		}
		result := l.Take(ClientOf(ss.Context()))
		ss.SetHeader(metadata.New(result.Headers()))
		if !result.Allowed {
			return status.Error(codes.ResourceExhausted, refusal)
		}
		return handler(srv, ss)
	}
}

// ClientOf names who a call came from. Behind the Fly edge that is the header
// the proxy adds; a call that reached the process directly — another service on
// the private network, or a local run — is named by its own address. A caller
// nothing can name shares one budget, which is the safe way round.
//
// It names the machine, not the person: calls made through a hosted Kaja arrive
// from that server, so a shared demo shares a budget.
func ClientOf(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	if ip := first(md, "fly-client-ip"); ip != "" {
		return ip
	}
	// X-Forwarded-For is a chain the nearest proxy appends to, so the client is
	// the first entry.
	if forwarded := first(md, "x-forwarded-for"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		if host, _, err := net.SplitHostPort(p.Addr.String()); err == nil {
			return host
		}
		return p.Addr.String()
	}
	return "unknown"
}

func first(md metadata.MD, name string) string {
	values := md.Get(name)
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}
