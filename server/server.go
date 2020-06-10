package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/belak/grpc-token-auth/pb"
)

type ServerConfig struct {
	BindHost  string `json:"bind_host"`
	EnableWeb bool   `json:"enable_web"`
}

type Server struct {
	grpcServer *grpc.Server
	config     ServerConfig
}

func NewServer(config ServerConfig) *Server {
	s := &Server{
		config: config,
	}

	// Build a new grpc server and register all our service implementations.
	s.grpcServer = grpc.NewServer(
		grpc.ChainUnaryInterceptor(s.unaryAuthInterceptor),
		grpc.ChainStreamInterceptor(s.streamAuthInterceptor),
	)
	pb.RegisterEchoServiceServer(s.grpcServer, s)

	return s
}

func (s *Server) authenticate(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing request metadata")
	}

	authTokens := md.Get("authorization")
	if len(authTokens) != 1 {
		return status.Errorf(codes.Unauthenticated, "wrong number of auth tokens: got %d, expected 1", len(authTokens))
	}

	authToken := authTokens[0]
	if !strings.HasPrefix(authToken, "Bearer ") {
		return status.Error(codes.Unauthenticated, "invalid token format")
	}
	authToken = strings.TrimPrefix(authToken, "Bearer ")

	if authToken != "helloworld" {
		return status.Error(codes.Unauthenticated, "invalid auth token")
	}

	return nil
}

func (s *Server) unaryAuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	err := s.authenticate(ctx)
	if err != nil {
		return nil, err
	}

	fmt.Println(info.FullMethod)

	return handler(ctx, req)
}

func (s *Server) streamAuthInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	err := s.authenticate(stream.Context())
	if err != nil {
		return err
	}

	fmt.Println(info.FullMethod)

	return handler(srv, stream)
}

func (s *Server) Run() error {
	wrapped := grpcweb.WrapServer(
		s.grpcServer,
		grpcweb.WithWebsockets(true),
		grpcweb.WithWebsocketPingInterval(30*time.Second),

		// We allow all origins because there's other auth
		grpcweb.WithOriginFunc(func(origin string) bool { return true }),
		grpcweb.WithWebsocketOriginFunc(func(req *http.Request) bool { return true }),
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If web is enabled and this is a valid grpc-web request, send it to
		// the grpc-web handler.
		if s.config.EnableWeb && (wrapped.IsGrpcWebRequest(r) || wrapped.IsGrpcWebSocketRequest(r) || wrapped.IsAcceptableGrpcCorsRequest(r)) {
			wrapped.ServeHTTP(w, r)
			return
		}

		// We use this to determine if the gRPC handler should be called. This
		// is the recommended example from gRPC's ServeHTTP.
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			s.grpcServer.ServeHTTP(w, r)
			return
		}

		// Anything can be put after this point - you could even serve a full
		// website from here. We use StatusNotFound because this is just gRPC.
		w.WriteHeader(http.StatusNotFound)
	})

	// Wrap the handler with h2c so we can use TLS termination with a remote
	// proxy.
	return http.ListenAndServe(s.config.BindHost, h2c.NewHandler(handler, &http2.Server{}))
}
