package main

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/lk2023060901/zeus-go/exmaples/grpc/basic/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

type chatServer struct {
	protos.UnimplementedChatServiceServer
}

func (s *chatServer) Chat(stream protos.ChatService_ChatServer) error {
	ctx := stream.Context()
	p, _ := peer.FromContext(ctx)
	clientAddr := "unknown"
	if p != nil {
		clientAddr = p.Addr.String()
	}

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		reply := &protos.ChatMessage{
			Id:        msg.Id,
			Sender:    "server",
			Content:   fmt.Sprintf("recv from %s: %s", clientAddr, msg.Content),
			UnixMilli: time.Now().UnixMilli(),
		}
		if err := stream.Send(reply); err != nil {
			return err
		}
	}
}

func main() {
	lis, err := net.Listen("tcp", "192.168.1.239:50051")
	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer()
	protos.RegisterChatServiceServer(grpcServer, &chatServer{})

	if err := grpcServer.Serve(lis); err != nil {
		panic(err)
	}
}
