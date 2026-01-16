package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/lk2023060901/zeus-go/exmaples/grpc/basic/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("192.168.1.239:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := protos.NewChatServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.Chat(ctx)
	if err != nil {
		panic(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			msg, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				return
			}
			fmt.Printf("server: %s\n", msg.Content)
		}
	}()

	for i := 1; i <= 5; i++ {
		msg := &protos.ChatMessage{
			Id:        fmt.Sprintf("msg-%d", i),
			Sender:    "client",
			Content:   fmt.Sprintf("hello %d", i),
			UnixMilli: time.Now().UnixMilli(),
		}
		if err := stream.Send(msg); err != nil {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}

	_ = stream.CloseSend()
	<-done
}
