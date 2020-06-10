package main

import (
	"context"

	"github.com/belak/grpc-token-auth/pb"
)

// Ensure this server actually implements the EchoServiceServer interface. This
// is particularly useful if we're exporting it out of this package.
var _ pb.EchoServiceServer = (*Server)(nil)

func (s *Server) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	return &pb.EchoResponse{
		Message: req.GetMessage(),
	}, nil
}

func (s *Server) StreamingEcho(stream pb.EchoService_StreamingEchoServer) error {
	// Flush the headers. This is not strictly required, but it makes it easier
	// to write clients for some languages because you can start the stream (and
	// start writing to it) without waiting for the server to send messages.
	err := stream.SendHeader(nil)
	if err != nil {
		return err
	}

	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		err = stream.Send(&pb.EchoResponse{Message: req.GetMessage()})
		if err != nil {
			return err
		}
	}
}
