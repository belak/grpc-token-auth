package main

import (
	"context"

	"github.com/belak/grpc-token-auth/pb"
)

func init() {
	// Ensure this server actually implements the EchoServiceServer interface.
	// This is particularly useful if we're exporting it out of this package.
	var s *Server = nil
	var _ pb.EchoServiceServer = s
}

func (s *Server) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	return &pb.EchoResponse{
		Message: req.GetMessage(),
	}, nil
}

func (s *Server) StreamingEcho(stream pb.EchoService_StreamingEchoServer) error {
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
