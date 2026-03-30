package server

import (
	"log"
	"net"

	"go-chat/handlers"
	chatv1 "go-chat/proto/chat/v1"

	"github.com/tmc/langchaingo/llms"
	"google.golang.org/grpc"
)

func StartGRPC(addr string, llm llms.Model) *handlers.ChatServer {
	grpcServer := grpc.NewServer()
	chatServer := handlers.NewChatServer(llm)
	chatv1.RegisterChatServiceServer(grpcServer, chatServer)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", addr, err)
	}

	go func() {
		log.Printf("gRPC server listening on %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	return chatServer
}
