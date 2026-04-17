package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"go-chat/config"
	"go-chat/handlers"
	chatv1 "go-chat/proto/chat/v1"

	"github.com/tmc/langchaingo/llms"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func StartHTTP(addr, grpcAddr string, cfg *config.Config, llm llms.Model) {
	gwMux := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			return invoker(ctx, method, req, reply, cc, opts...)
		}),
	}
	if err := chatv1.RegisterChatServiceHandlerFromEndpoint(context.Background(), gwMux, grpcAddr, opts); err != nil {
		log.Fatalf("failed to register gateway: %v", err)
	}

	r := gin.New()
	r.Use(gin.Recovery(), handlers.Logger())
	r.SetTrustedProxies([]string{"127.0.0.1"})
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8080", "http://localhost:3030"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.POST("/v1/auth/login", handlers.Login)

	streamHandler := handlers.NewStreamHandler(llm)

	api := r.Group("/", handlers.RequireInternalToken(cfg.InternalSecret))
	api.Any("/generate", gin.WrapH(gwMux))
	api.POST("/generate/stream", streamHandler.GenerateStream)

	r.Static("/static", "./static")
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/static/index.html")
	})

	log.Printf("HTTP server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}
