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
	streamTimeout = 5 * time.Minute
	cacheTTL      = 1 * time.Hour
	cacheCapacity = 1_000

	maxInputTokens  = 2_048
	maxOutputTokens = 1_024
	charsPerToken   = 4
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
	if err := validateMessages(req.Messages); err != nil {
		return nil, err
	}
	messages := protoToLangchain(req.Messages)
	resp, err := s.llm.GenerateContent(ctx, messages, llms.WithMaxTokens(maxOutputTokens))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "llm error: %v", err)
	}
	return &chatv1.GenerateResponse{Response: resp.Choices[0].Content}, nil
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

	if err := validateMessages(req.Messages); err != nil {
		st, _ := status.FromError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": st.Message()})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	cacheKey := messageCacheKey(req.Messages)

	// Cache hit — replay word by word to preserve streaming UX
	if cached, ok, _ := h.cache.Get(cacheKey); ok {
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
	messages := protoToLangchain(req.Messages)

	_, err := h.llm.GenerateContent(ctx, messages,
		llms.WithMaxTokens(maxOutputTokens),
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

	h.cache.Set(cacheKey, buf.String())

	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	c.Writer.Flush()
}

func protoToLangchain(msgs []*chatv1.Message) []llms.MessageContent {
	out := make([]llms.MessageContent, len(msgs))
	for i, m := range msgs {
		var role llms.ChatMessageType
		switch m.Role {
		case chatv1.Role_ROLE_SYSTEM:
			role = llms.ChatMessageTypeSystem
		case chatv1.Role_ROLE_ASSISTANT:
			role = llms.ChatMessageTypeAI
		default:
			role = llms.ChatMessageTypeHuman
		}
		out[i] = llms.MessageContent{
			Role:  role,
			Parts: []llms.ContentPart{llms.TextPart(m.Content)},
		}
	}
	return out
}

func validateMessages(msgs []*chatv1.Message) error {
	if len(msgs) == 0 {
		return status.Error(codes.InvalidArgument, "messages must not be empty")
	}
	total := 0
	for _, m := range msgs {
		total += len(m.Content) / charsPerToken
	}
	if total > maxInputTokens {
		return status.Errorf(codes.InvalidArgument, "messages too long: estimated %d tokens, max %d", total, maxInputTokens)
	}
	return nil
}

func messageCacheKey(msgs []*chatv1.Message) string {
	b, _ := json.Marshal(msgs)
	return string(b)
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
