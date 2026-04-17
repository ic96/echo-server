package main

import (
	"go-chat/config"
	"go-chat/server"

	"log"

	"github.com/tmc/langchaingo/llms/openai"
)

const (
	grpcAddr = "localhost:9090"
	httpAddr = ":8080"
)

func main() {
	cfg := config.Load()

	llm, err := openai.New(
		openai.WithToken(cfg.OpenRouterAPIKey),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("openai/gpt-oss-20b:free"),
	)
	if err != nil {
		log.Fatal(err)
	}

	server.StartGRPC(grpcAddr, llm)
	server.StartHTTP(httpAddr, grpcAddr, cfg, llm)
}
