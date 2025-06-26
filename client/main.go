package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "mod_ex/proto"
)

var requestMetris = promauto.NewSummaryVec(prometheus.SummaryOpts{
	Namespace:  "ads",
	Subsystem:  "http",
	Name:       "request",
	Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
}, []string{"status"})

func main() {
	// Инициализация соединения вне цикла, чтобы не создавать новое соединение при каждой итерации.
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer conn.Close()

	client := pb.NewEXSClient(conn)

	counter := 1 // Инициализируйте счетчик до цикла.
	defer func() {
		fmt.Println("Total requests:", counter) // Выводите значение счетчика при завершении.
	}()

	for {
		// Создавайте новый контекст для каждого запроса с таймаутом.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, err := client.GetSingleInfo(ctx, &pb.SimpleRequest{Name: "test"})
		cancel() // Отменяйте контекст сразу после получения ответа (или ошибки).

		if err != nil {
			fmt.Println("Request number:", counter) // Указывайте номер запроса при ошибке.
			log.Fatalf("could not greet: %v", err)  // Корректно обрабатывайте ошибки.
		}
		counter++
	}
}
