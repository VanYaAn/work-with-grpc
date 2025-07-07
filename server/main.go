package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	pb "mod_ex/proto"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus" // Импортируем prometheus
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"  // Для получения кодов ошибок gRPC
	"google.golang.org/grpc/status" // Для получения статуса из ошибки gRPC
)

// Определение собственной гистограммы для измерения задержки запросов
var (
	// requestDuration_custom - это вектор гистограмм.
	// Каждая гистограмма будет иметь метки для метода gRPC, сервиса, кода ответа и типа запроса.
	requestDuration_custom = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_server_handling_seconds_custom",                                                           // Имя метрики
			Help:    "Histogram of response latency (seconds) of gRPC that had been handled by the server (custom).", // Описание
			Buckets: prometheus.DefBuckets,                                                                           // Используем стандартные диапазоны бакетов Prometheus
		},
		[]string{"grpc_method", "grpc_service", "grpc_code", "grpc_type"},
	)
)

// init() функция вызывается при запуске приложения, здесь регистрируем нашу метрику
func init() {
	prometheus.MustRegister(requestDuration_custom)
	// Optionally, unregister default grpc_prometheus metrics if they conflict
	// prometheus.Unregister(grpc_prometheus.Default  // There is no Default for metrics, this is not needed.
	// You may choose to unregister specific grpc_prometheus metrics if they are problematic
	// but generally, it's fine to have both.
}

type Server struct {
	pb.UnimplementedEXSServer
}

func (s *Server) GetSingleInfo(ctx context.Context, req *pb.SimpleRequest) (*pb.SimpleResponse, error) {
	// Добавим небольшую задержку, чтобы метрика имела хоть какие-то значения, отличные от нуля
	return &pb.SimpleResponse{Name: "OKAY"}, nil
}

func (s *Server) GetStreamInfo(stream grpc.BidiStreamingServer[pb.SimpleRequest, pb.SimpleResponse]) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			// В случае ошибки, завершаем поток и возвращаем ошибку
			return err
		}
		// Добавим небольшую задержку для стриминга тоже
		time.Sleep(5 * time.Millisecond)
		if err := stream.Send(&pb.SimpleResponse{Name: in.Name}); err != nil {
			return err
		}
	}
}

// timeInterceptor теперь будет записывать данные в нашу кастомную гистограмму
func timeInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	start := time.Now()
	resp, err = handler(ctx, req) // Вызываем основной обработчик RPC

	// Определяем статус код gRPC для метрики
	statusCode := codes.OK // По умолчанию OK
	if err != nil {
		if s, ok := status.FromError(err); ok {
			statusCode = s.Code() // Если это ошибка gRPC, берем ее код
		} else {
			statusCode = codes.Unknown // Иначе, неизвестная ошибка
		}
		log.Printf("Request %s failed: %v", info.FullMethod, err)
	}

	duration := time.Since(start).Seconds() // Вычисляем длительность в секундах
	log.Printf("Request %s took %f seconds with status %s", info.FullMethod, duration, statusCode.String())

	// Записываем измерение в нашу кастомную гистограмму
	// Метки: grpc_method, grpc_service, grpc_code, grpc_type
	requestDuration_custom.WithLabelValues(
		info.FullMethod,     // Полное имя метода (например, /ex.EXS/GetSingleInfo)
		"ex.EXS",            // Имя сервиса (из вашего proto-файла)
		statusCode.String(), // Текстовое представление gRPC кода
		"unary",             // Тип RPC (для GetSingleInfo это 'unary')
	).Observe(duration)

	return resp, err
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	fmt.Println("Listening on", lis.Addr())
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		// Теперь используем наш timeInterceptor для юнарных запросов,
		// чтобы он записывал метрики времени выполнения.
		grpc.UnaryInterceptor(timeInterceptor),
		// grpc_prometheus.StreamServerInterceptor по-прежнему полезен для других метрик стриминга
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
	)
	pb.RegisterEXSServer(grpcServer, &Server{})

	// Все еще регистрируем стандартные метрики grpc_prometheus,
	// так как они предоставляют полезную информацию (например, количество запросов, сообщений).
	grpc_prometheus.Register(grpcServer)

	log.Println("Server is running on port :50051")
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC server: %v", err)
		}
	}()

	// Prometheus HTTP server для экспорта метрик
	http.Handle("/metrics", promhttp.Handler())
	log.Println("Prometheus metrics server is running on port :9092")
	if err := http.ListenAndServe(":9092", nil); err != nil {
		log.Fatalf("failed to start Prometheus HTTP server: %v", err)
	}
}
