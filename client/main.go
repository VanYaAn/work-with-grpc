package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "mod_ex/proto"
)

func main() {
	// Инициализация соединения вне цикла, чтобы не создавать новое соединение при каждой итерации.
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer conn.Close()

	client := pb.NewEXSClient(conn)

	for i := 0; i < 1000000; i++ {
		// Создавайте новый контекст для каждого запроса с таймаутом.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, err := client.GetSingleInfo(ctx, &pb.SimpleRequest{Name: "test"})
		cancel() // Отменяйте контекст сразу после получения ответа (или ошибки).

		if err != nil {
			log.Fatalf("could not greet: %v", err) // Корректно обрабатывайте ошибки.
		}

	}
}
