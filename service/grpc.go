package main

import (
	"log"
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "deephealth/build/gen"
)

const (
	port = ":50051"
)

type server struct{}

func (s *server) SubmitReport(ctx context.Context, in *pb.SubmitReportRequest) (*pb.SubmitReportReply, error) {
	return &pb.SubmitReportReply{Result: pb.SubmitReportReply_IGNORED}, nil
}

func (s *server) GetReport(ctx context.Context, in *pb.GetReportRequest) (*pb.GetReportReply, error) {
	var report pb.Report
	return &pb.GetReportReply{Report: &report}, nil
}

func (s *server) ObserveSubject(ctx context.Context, in *pb.ObserveRequest) (*pb.ObserveReply, error) {
	return &pb.ObserveReply{Success: true}, nil
}

func (s *server) StopObservingSubject(ctx context.Context, in *pb.ObserveRequest) (*pb.ObserveReply, error) {
	return &pb.ObserveReply{Success: true}, nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterHealthServiceServer(s, &server{})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
