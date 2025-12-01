package lmstudio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang-template/configs"
	"golang-template/internal/domain"
)

// TestNewLMStudioClientAdapterWithConfig tests adapter construction with valid config
func TestNewLMStudioClientAdapterWithConfig(t *testing.T) {
	config := configs.LMStudio{
		BaseURL: "http://localhost:5678",
		Model:   "test-model",
		Timeout: 30,
	}

	adapter, err := NewLMStudioClientAdapter(config)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if adapter == nil {
		t.Fatal("expected adapter to be non-nil")
	}

	if adapter.baseURL != "http://localhost:5678" {
		t.Errorf("expected baseURL to be http://localhost:5678, got: %s", adapter.baseURL)
	}

	if adapter.configModel != "test-model" {
		t.Errorf("expected configModel to be test-model, got: %s", adapter.configModel)
	}

	if adapter.timeout != 30*time.Second {
		t.Errorf("expected timeout to be 30s, got: %v", adapter.timeout)
	}
}

// TestNewLMStudioClientAdapterWithDefaultValues tests adapter construction with default values
func TestNewLMStudioClientAdapterWithDefaultValues(t *testing.T) {
	config := configs.LMStudio{
		BaseURL: "", // Empty - should default to http://localhost:1234
		Model:   "",
		Timeout: 0, // Zero - should default to 60 seconds
	}

	adapter, err := NewLMStudioClientAdapter(config)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if adapter == nil {
		t.Fatal("expected adapter to be non-nil")
	}

	if adapter.baseURL != "http://localhost:1234" {
		t.Errorf("expected default baseURL to be http://localhost:1234, got: %s", adapter.baseURL)
	}

	if adapter.timeout != 60*time.Second {
		t.Errorf("expected default timeout to be 60s, got: %v", adapter.timeout)
	}
}

// TestChatCompletionSuccess tests non-streaming chat completion with mock HTTP server
func TestChatCompletionSuccess(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request path
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("expected path /v1/chat/completions, got: %s", r.URL.Path)
		}

		// Verify method
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got: %s", r.Method)
		}

		// Verify content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got: %s", r.Header.Get("Content-Type"))
		}

		// Decode request body to verify structure
		var reqBody chatCompletionAPIRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Verify request body
		if reqBody.Stream != false {
			t.Errorf("expected stream=false for non-streaming, got: %v", reqBody.Stream)
		}

		if len(reqBody.Messages) != 1 {
			t.Errorf("expected 1 message, got: %d", len(reqBody.Messages))
		}

		// Return mock response
		response := chatCompletionAPIResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "test-model",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "Hello! How can I help you today?",
					},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     10,
				CompletionTokens: 8,
				TotalTokens:      18,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create adapter with mock server URL
	config := configs.LMStudio{
		BaseURL: server.URL,
		Model:   "test-model",
		Timeout: 30,
	}

	adapter, err := NewLMStudioClientAdapter(config)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	// Send chat completion request
	ctx := context.Background()
	request := domain.ChatCompletionRequest{
		Messages: []domain.ChatMessage{
			{
				Role:    domain.ChatMessageRoleUser,
				Content: "Hello!",
			},
		},
		Stream: false,
	}

	response, err := adapter.ChatCompletion(ctx, request)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify response
	if response == nil {
		t.Fatal("expected response to be non-nil")
	}

	if response.Content != "Hello! How can I help you today?" {
		t.Errorf("expected content 'Hello! How can I help you today?', got: %s", response.Content)
	}

	if response.Model != "test-model" {
		t.Errorf("expected model 'test-model', got: %s", response.Model)
	}

	if response.TotalTokens != 18 {
		t.Errorf("expected total tokens 18, got: %d", response.TotalTokens)
	}
}

// TestChatCompletionStreamChannelBehavior tests streaming chat completion channel behavior
func TestChatCompletionStreamChannelBehavior(t *testing.T) {
	// Create mock server for streaming
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request path
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("expected path /v1/chat/completions, got: %s", r.URL.Path)
		}

		// Decode request body
		var reqBody chatCompletionAPIRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Verify stream=true
		if reqBody.Stream != true {
			t.Errorf("expected stream=true for streaming, got: %v", reqBody.Stream)
		}

		// Send SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("expected ResponseWriter to implement Flusher")
			return
		}

		// Send chunks
		chunks := []string{"Hello", ", ", "World", "!"}
		for i, chunk := range chunks {
			sseData := fmt.Sprintf(`{"id":"chatcmpl-%d","object":"chat.completion.chunk","created":%d,"model":"test-model","choices":[{"index":0,"delta":{"content":"%s"},"finish_reason":null}]}`,
				i, time.Now().Unix(), chunk)
			fmt.Fprintf(w, "data: %s\n\n", sseData)
			flusher.Flush()
		}

		// Send [DONE] marker
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	// Create adapter with mock server URL
	config := configs.LMStudio{
		BaseURL: server.URL,
		Model:   "test-model",
		Timeout: 30,
	}

	adapter, err := NewLMStudioClientAdapter(config)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	// Send streaming request
	ctx := context.Background()
	request := domain.ChatCompletionRequest{
		Messages: []domain.ChatMessage{
			{
				Role:    domain.ChatMessageRoleUser,
				Content: "Say hello world",
			},
		},
		Stream: true,
	}

	chunkChan, err := adapter.ChatCompletionStream(ctx, request)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Collect all chunks
	var receivedContent strings.Builder
	var chunkCount int
	var finalChunk domain.ChatCompletionChunk

	for chunk := range chunkChan {
		chunkCount++
		if chunk.Error != nil && !chunk.Done {
			t.Errorf("unexpected error in chunk: %v", chunk.Error)
		}
		if chunk.Done {
			finalChunk = chunk
		} else {
			receivedContent.WriteString(chunk.Content)
		}
	}

	// Verify we received all chunks plus done marker
	if chunkCount < 4 {
		t.Errorf("expected at least 4 chunks, got: %d", chunkCount)
	}

	// Verify final chunk is done marker
	if !finalChunk.Done {
		t.Error("expected final chunk to have Done=true")
	}

	// Verify received content
	expectedContent := "Hello, World!"
	if receivedContent.String() != expectedContent {
		t.Errorf("expected content '%s', got: '%s'", expectedContent, receivedContent.String())
	}

	// Verify channel is closed (should not block)
	select {
	case _, ok := <-chunkChan:
		if ok {
			t.Error("expected channel to be closed")
		}
	default:
		// Channel is closed, this is expected
	}
}

// TestListModelsResponseParsing tests ListModels response parsing
func TestListModelsResponseParsing(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request path
		if r.URL.Path != "/v1/models" {
			t.Errorf("expected path /v1/models, got: %s", r.URL.Path)
		}

		// Verify method
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got: %s", r.Method)
		}

		// Return mock models response
		response := modelsResponse{
			Object: "list",
			Data: []struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				OwnedBy string `json:"owned_by"`
			}{
				{
					ID:      "llama-3.2-1b-instruct",
					Object:  "model",
					OwnedBy: "lmstudio",
				},
				{
					ID:      "qwen2.5-coder-7b-instruct",
					Object:  "model",
					OwnedBy: "lmstudio",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create adapter with mock server URL
	config := configs.LMStudio{
		BaseURL: server.URL,
		Model:   "",
		Timeout: 30,
	}

	adapter, err := NewLMStudioClientAdapter(config)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	// List models
	ctx := context.Background()
	models, err := adapter.ListModels(ctx)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify response
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got: %d", len(models))
	}

	if models[0].ID != "llama-3.2-1b-instruct" {
		t.Errorf("expected first model ID 'llama-3.2-1b-instruct', got: %s", models[0].ID)
	}

	if models[1].ID != "qwen2.5-coder-7b-instruct" {
		t.Errorf("expected second model ID 'qwen2.5-coder-7b-instruct', got: %s", models[1].ID)
	}

	if models[0].OwnedBy != "lmstudio" {
		t.Errorf("expected OwnedBy 'lmstudio', got: %s", models[0].OwnedBy)
	}
}

// TestRetryLogicFor5xxErrors tests retry behavior for 5xx server errors
func TestRetryLogicFor5xxErrors(t *testing.T) {
	var requestCount int32

	// Create mock server that returns 503 twice then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)

		if count <= 2 {
			// First two requests: return 503 Service Unavailable
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Service temporarily unavailable"))
			return
		}

		// Third request: return success
		response := modelsResponse{
			Object: "list",
			Data: []struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				OwnedBy string `json:"owned_by"`
			}{
				{
					ID:      "success-model",
					Object:  "model",
					OwnedBy: "lmstudio",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create adapter with mock server URL
	config := configs.LMStudio{
		BaseURL: server.URL,
		Model:   "",
		Timeout: 30,
	}

	adapter, err := NewLMStudioClientAdapter(config)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	// List models - should retry on 503 and eventually succeed
	ctx := context.Background()
	models, err := adapter.ListModels(ctx)
	if err != nil {
		t.Fatalf("expected no error after retry, got: %v", err)
	}

	// Verify we got the response from the third (successful) request
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got: %d", len(models))
	}

	if models[0].ID != "success-model" {
		t.Errorf("expected model ID 'success-model', got: %s", models[0].ID)
	}

	// Verify we made 3 requests (2 retries + 1 success)
	finalCount := atomic.LoadInt32(&requestCount)
	if finalCount != 3 {
		t.Errorf("expected 3 requests (2 failures + 1 success), got: %d", finalCount)
	}
}

// TestNoRetryFor4xxErrors tests that 4xx errors are not retried
func TestNoRetryFor4xxErrors(t *testing.T) {
	var requestCount int32

	// Create mock server that always returns 400 Bad Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)

		// Always return 400 Bad Request
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid request"}`))
	}))
	defer server.Close()

	// Create adapter with mock server URL
	config := configs.LMStudio{
		BaseURL: server.URL,
		Model:   "",
		Timeout: 30,
	}

	adapter, err := NewLMStudioClientAdapter(config)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	// List models - should fail immediately without retry
	ctx := context.Background()
	_, err = adapter.ListModels(ctx)

	// Verify we got an error
	if err == nil {
		t.Fatal("expected error for 4xx response, got nil")
	}

	// Verify we only made 1 request (no retry for 4xx)
	finalCount := atomic.LoadInt32(&requestCount)
	if finalCount != 1 {
		t.Errorf("expected exactly 1 request (no retry for 4xx), got: %d", finalCount)
	}

	// Verify error wraps ErrInvalidRequest
	if !strings.Contains(err.Error(), "invalid request") {
		t.Errorf("expected error to contain 'invalid request', got: %v", err)
	}
}
