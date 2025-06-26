package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	pb "mod_ex/proto"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

// func recordMetrics() {
// 	go func() {
// 		for {
// 			opsProcessed.Inc()
// 			time.Sleep(2 * time.Second)
// 		}
// 	}()
// }

// var (
// 	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
// 		Name: "myapp_processed_ops_total",
// 		Help: "The total number of processed events",
// 	})
// )

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

func main() {
	lis, err := net.Listen("tcp", ":50051")
	fmt.Println(lis)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	)
	pb.RegisterEXSServer(grpcServer, &Server{})
	log.Println("Server is running on port :50051")
	go grpcServer.Serve(lis)
	func() {
		// recordMetrics()
		log.Println("Prometheus metrics server is running on port :9092")
		PrometeusHttpServer := http.Server{

			Addr:    ":9092",
			Handler: promhttp.Handler(),
		}
		http.Handle("/metrics", promhttp.Handler())

		if err := PrometeusHttpServer.ListenAndServe(); err != nil {
			log.Fatalf("failed to start Prometheus HTTP server: %v", err)
		}
	}()

}
