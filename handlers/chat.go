package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/hot"
	"github.com/tmc/langchaingo/llms"
	chatv1 "go-chat/proto/chat/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	streamTimeout  = 5 * time.Minute
	cacheTTL       = 1 * time.Hour
	cacheCapacity  = 1_000
)

// ChatServer handles gRPC requests.
type ChatServer struct {
	chatv1.UnimplementedChatServiceServer
	llm llms.Model
}

func NewChatServer(llm llms.Model) *ChatServer {
	return &ChatServer{llm: llm}
}

func (s *ChatServer) Generate(ctx context.Context, req *chatv1.GenerateRequest) (*chatv1.GenerateResponse, error) {
	completion, err := llms.GenerateFromSinglePrompt(ctx, s.llm, req.Prompt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "llm error: %v", err)
	}
	return &chatv1.GenerateResponse{Response: completion}, nil
}

// StreamHandler handles HTTP SSE streaming requests.
type StreamHandler struct {
	llm   llms.Model
	cache *hot.HotCache[string, string]
}

func NewStreamHandler(llm llms.Model) *StreamHandler {
	cache := hot.NewHotCache[string, string](hot.WTinyLFU, cacheCapacity).
		WithTTL(cacheTTL).
		WithJitter(0.1, 5*time.Minute).
		WithJanitor().
		Build()
	return &StreamHandler{llm: llm, cache: cache}
}

func (h *StreamHandler) GenerateStream(c *gin.Context) {
	var req chatv1.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Cache hit — replay word by word to preserve streaming UX
	if cached, ok, _ := h.cache.Get(req.Prompt); ok {
		for _, word := range strings.Fields(cached) {
			_ = sseEvent(c.Writer, map[string]string{"chunk": word + " "})
		}
		fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
		c.Writer.Flush()
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), streamTimeout)
	defer cancel()

	var buf strings.Builder

	_, err := llms.GenerateFromSinglePrompt(ctx, h.llm, req.Prompt,
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			buf.Write(chunk)
			return sseEvent(c.Writer, map[string]string{"chunk": string(chunk)})
		}),
	)

	if err != nil {
		_ = sseEvent(c.Writer, map[string]string{"error": err.Error()})
		return
	}

	h.cache.Set(req.Prompt, buf.String())

	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	c.Writer.Flush()
}

func sseEvent(w gin.ResponseWriter, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return err
	}
	w.Flush()
	return nil
}
