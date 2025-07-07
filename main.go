package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	pb "mod_ex/proto"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
)

var Data []string
var counter int

type Server struct {
	pb.UnimplementedEXSServer
}

func (s *Server) GetSingleInfo(ctx context.Context, req *pb.SimpleRequest) (*pb.SimpleResponse, error) {

	return &pb.SimpleResponse{Name: "OKAY"}, nil
}
func (s *Server) GetStreamInfo(stream grpc.BidiStreamingServer[pb.SimpleRequest, pb.SimpleResponse]) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err

		}
		if err := stream.Send(&pb.SimpleResponse{Name: in.Name}); err != nil {
			return err
		}

	}

}
func timeInterseptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	start := time.Now()
	resp, err = handler(ctx, req)
	if err != nil {
		log.Printf("Request %s failed: %v", info.FullMethod, err)
		return nil, err
	}
	Data = append(Data, fmt.Sprintf("%d\n", time.Since(start).Nanoseconds()))
	counter++
	// log.Printf("Request %s took %s", info.FullMethod, time.Since(start))
	return resp, err
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	fmt.Println(lis)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		// grpc.UnaryInterceptor(grpc_promethesus.UnaryServerInterceptor),
		grpc.UnaryInterceptor(timeInterseptor),
	)
	pb.RegisterEXSServer(grpcServer, &Server{})

	log.Println("Server is running on port :50051")
	go func() {
		for {
			if counter == 1000000 {

				fileName := "Data2parallel.txt"
				file, err := os.Create(fileName)
				if err != nil {
					fmt.Println("Unable to create file:", err)
				}
				defer file.Close()
				for _, val := range Data {
					file.WriteString(val)
				}

				fmt.Println("Done.")
			}
		}
	}()
	grpcServer.Serve(lis)

}
