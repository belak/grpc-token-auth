package main

import (
	"context"
	"crypto/x509"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/belak/grpc-token-auth/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func Env(key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("failed to look up variable: %s", key)
	}
	return val
}

func EnvDefault(key string, def string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return def
	}

	return val
}

func main() {
	insecureRaw := EnvDefault("INSECURE", "false")
	insecure, err := strconv.ParseBool(insecureRaw)
	if err != nil {
		log.Fatalf("failed to parse INSECURE value: %s", insecureRaw)
	}

	client, err := NewClient(ClientConfig{
		DialTimeout: 5 * time.Second,
		Token:       Env("TOKEN"),
		Addr:        EnvDefault("SERVER_ADDR", "localhost:8000"),
		Insecure:    insecure,
	})
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}

	ctx := context.Background()

	resp, err := client.Echo(ctx, &pb.EchoRequest{Message: "hello world"})
	if err != nil {
		log.Fatalf("failed send echo request: %s", err)
	}

	log.Printf("got echo response: %+v", resp)

	stream, err := client.StreamingEcho(ctx)
	if err != nil {
		log.Fatalf("failed start echo stream: %s", err)
	}

	for i := 0; i < 5; i++ {
		time.Sleep(1)

		err := stream.Send(&pb.EchoRequest{Message: "hello world " + strconv.Itoa(i)})
		if err != nil {
			log.Fatalf("failed to send stream request %d: %s", i, err)
		}

		resp, err := stream.Recv()
		if err != nil {
			log.Fatalf("failed to receive stream response %d: %s", i, err)
		}
		log.Printf("got stream response %d: %+v", i, resp)
	}
}

type ClientConfig struct {
	Addr        string
	DialTimeout time.Duration
	Token       string
	Insecure    bool
}

func NewClient(config ClientConfig) (pb.EchoServiceClient, error) {
	ctx := context.Background()
	if config.DialTimeout > 0 {
		newCtx, cancel := context.WithTimeout(ctx, config.DialTimeout)
		defer cancel()
		ctx = newCtx
	}

	var opt grpc.DialOption
	if config.Insecure {
		opt = grpc.WithInsecure()
	} else {
		certPool, err := x509.SystemCertPool()
		if err != nil {
			return nil, err
		}

		opt = grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(certPool, ""))
	}

	conn, err := grpc.DialContext(ctx, config.Addr,
		opt,
		grpc.WithPerRPCCredentials(TokenAuth{
			Token:    config.Token,
			Insecure: config.Insecure,
		}),
		grpc.WithBlock())

	return pb.NewEchoServiceClient(conn), err
}

type TokenAuth struct {
	Token    string
	Insecure bool
}

func (a TokenAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"auth_token": a.Token,
	}, nil
}

func (a TokenAuth) RequireTransportSecurity() bool {
	return !a.Insecure
}
