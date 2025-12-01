package lmstudio

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang-template/configs"
	"golang-template/internal/domain"

	"github.com/sirupsen/logrus"
)

// LMStudioClientAdapter struct - Output adapter for LM Studio's OpenAI-compatible API
type LMStudioClientAdapter struct {
	httpClient  *http.Client
	baseURL     string
	configModel string
	timeout     time.Duration

	// Model caching
	cachedModel string
	modelMu     sync.RWMutex
}

// NewLMStudioClientAdapter func - Creates new LM Studio client adapter
func NewLMStudioClientAdapter(config configs.LMStudio) (*LMStudioClientAdapter, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:1234"
	}

	// Remove trailing slash if present
	baseURL = strings.TrimSuffix(baseURL, "/")

	timeout := time.Duration(config.Timeout) * time.Second
	if config.Timeout <= 0 {
		timeout = 60 * time.Second
	}

	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	adapter := &LMStudioClientAdapter{
		httpClient:  httpClient,
		baseURL:     baseURL,
		configModel: config.Model,
		timeout:     timeout,
	}

	logrus.Infof("LM Studio client adapter initialized with base URL: %s, timeout: %v", baseURL, timeout)

	return adapter, nil
}

// Retry configuration constants
const (
	maxRetryAttempts  = 30
	initialDelay      = 1 * time.Second
	maxDelay          = 30 * time.Second
	backoffMultiplier = 2
)

// Streaming configuration constants
const (
	streamingChannelBufferSize = 100
)

// retryWithBackoff executes an operation with exponential backoff retry logic
func (a *LMStudioClientAdapter) retryWithBackoff(ctx context.Context, operation func() (*http.Response, error)) (*http.Response, error) {
	var lastErr error
	delay := initialDelay

	for attempt := 1; attempt <= maxRetryAttempts; attempt++ {
		resp, err := operation()

		// Check if we should retry
		if err != nil {
			if !a.isTransientError(err, 0) {
				return nil, err
			}
			lastErr = err
			logrus.Warnf("LM Studio request attempt %d/%d failed with error: %v, retrying in %v", attempt, maxRetryAttempts, err, delay)
		} else if resp != nil {
			// Check status code
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return resp, nil
			}

			// Don't retry on 4xx client errors
			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				return nil, fmt.Errorf("%w: status %d - %s", domain.ErrInvalidRequest, resp.StatusCode, string(body))
			}

			// Retry on 5xx server errors
			if a.isTransientError(nil, resp.StatusCode) {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				lastErr = fmt.Errorf("server error: status %d - %s", resp.StatusCode, string(body))
				logrus.Warnf("LM Studio request attempt %d/%d failed with status %d, retrying in %v", attempt, maxRetryAttempts, resp.StatusCode, delay)
			} else {
				return resp, nil
			}
		}

		// Check context before sleeping
		if attempt < maxRetryAttempts {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
			case <-time.After(delay):
			}

			// Calculate next delay with exponential backoff
			delay = delay * backoffMultiplier
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("%w: %v after %d attempts", domain.ErrLMStudioUnavailable, lastErr, maxRetryAttempts)
	}
	return nil, fmt.Errorf("%w: max retries exceeded", domain.ErrLMStudioUnavailable)
}

// isTransientError determines if an error or status code is transient and should be retried
func (a *LMStudioClientAdapter) isTransientError(err error, statusCode int) bool {
	// Check for transient status codes (5xx server errors)
	if statusCode >= 500 && statusCode < 600 {
		return true
	}

	// 4xx errors are NOT transient
	if statusCode >= 400 && statusCode < 500 {
		return false
	}

	if err == nil {
		return false
	}

	// Check for network-related errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}

	// Check for connection refused or other network issues
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	// Check for DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	// Check for context deadline exceeded (timeout)
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Check error message for common transient patterns
	errMsg := err.Error()
	transientPatterns := []string{
		"connection refused",
		"connection reset",
		"no such host",
		"network is unreachable",
		"i/o timeout",
		"EOF",
	}
	for _, pattern := range transientPatterns {
		if strings.Contains(strings.ToLower(errMsg), strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// ListModels queries the /v1/models endpoint to retrieve available models from LM Studio
func (a *LMStudioClientAdapter) ListModels(ctx context.Context) ([]domain.ModelInfo, error) {
	url := fmt.Sprintf("%s/v1/models", a.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list models request: %w", err)
	}

	resp, err := a.retryWithBackoff(ctx, func() (*http.Response, error) {
		return a.httpClient.Do(req)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var modelsResp modelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to parse models response: %w", err)
	}

	// Convert to domain models
	models := make([]domain.ModelInfo, len(modelsResp.Data))
	for i, m := range modelsResp.Data {
		models[i] = domain.ModelInfo{
			ID:      m.ID,
			Object:  m.Object,
			OwnedBy: m.OwnedBy,
		}
	}

	logrus.Infof("Listed %d models from LM Studio", len(models))

	return models, nil
}

// getModel returns the model to use for requests, with caching
func (a *LMStudioClientAdapter) getModel(ctx context.Context) (string, error) {
	// Fast path: check if model is already cached
	a.modelMu.RLock()
	if a.cachedModel != "" {
		model := a.cachedModel
		a.modelMu.RUnlock()
		return model, nil
	}
	a.modelMu.RUnlock()

	// Slow path: need to determine and cache the model
	a.modelMu.Lock()
	defer a.modelMu.Unlock()

	// Double-check after acquiring write lock
	if a.cachedModel != "" {
		return a.cachedModel, nil
	}

	// Check if model is configured via environment variable
	if a.configModel != "" {
		a.cachedModel = a.configModel
		logrus.Infof("Using configured model from environment: %s", a.cachedModel)
		return a.cachedModel, nil
	}

	// Query available models and select the first one
	models, err := a.ListModels(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get models for selection: %w", err)
	}

	if len(models) == 0 {
		return "", fmt.Errorf("%w: no models available in LM Studio", domain.ErrLMStudioUnavailable)
	}

	a.cachedModel = models[0].ID
	logrus.Infof("Selected first available model: %s", a.cachedModel)

	return a.cachedModel, nil
}

// ChatCompletion sends a non-streaming chat completion request to LM Studio
func (a *LMStudioClientAdapter) ChatCompletion(ctx context.Context, request domain.ChatCompletionRequest) (*domain.ChatCompletionResponse, error) {
	// Get model to use
	model, err := a.getModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	// Override model if specified in request
	if request.Model != nil && *request.Model != "" {
		model = *request.Model
	}

	// Build request body
	reqBody := chatCompletionAPIRequest{
		Model:    model,
		Messages: make([]chatMessageAPI, len(request.Messages)),
		Stream:   false,
	}

	for i, msg := range request.Messages {
		reqBody.Messages[i] = chatMessageAPI{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	if request.Temperature != nil {
		reqBody.Temperature = request.Temperature
	}

	// Marshal request body
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/chat/completions", a.baseURL)

	// Execute request with retry
	resp, err := a.retryWithBackoff(ctx, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return a.httpClient.Do(req)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send chat completion request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var apiResp chatCompletionAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse chat completion response: %w", err)
	}

	// Extract content from choices
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := apiResp.Choices[0].Message.Content

	// Build domain response
	response := &domain.ChatCompletionResponse{
		Content:          content,
		Model:            apiResp.Model,
		PromptTokens:     apiResp.Usage.PromptTokens,
		CompletionTokens: apiResp.Usage.CompletionTokens,
		TotalTokens:      apiResp.Usage.TotalTokens,
	}

	logrus.Infof("Chat completion successful, model: %s, tokens: %d", response.Model, response.TotalTokens)

	return response, nil
}

// ChatCompletionStream sends a streaming chat completion request to LM Studio
// Returns a read-only channel that emits ChatCompletionChunk as they arrive
func (a *LMStudioClientAdapter) ChatCompletionStream(ctx context.Context, request domain.ChatCompletionRequest) (<-chan domain.ChatCompletionChunk, error) {
	// Get model to use
	model, err := a.getModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	// Override model if specified in request
	if request.Model != nil && *request.Model != "" {
		model = *request.Model
	}

	// Build request body with stream=true
	reqBody := chatCompletionAPIRequest{
		Model:    model,
		Messages: make([]chatMessageAPI, len(request.Messages)),
		Stream:   true,
	}

	for i, msg := range request.Messages {
		reqBody.Messages[i] = chatMessageAPI{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	if request.Temperature != nil {
		reqBody.Temperature = request.Temperature
	}

	// Marshal request body
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal streaming request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/chat/completions", a.baseURL)

	// Create request (no retry for streaming - we want immediate feedback)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create streaming request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	// Send request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send streaming request: %w", err)
	}

	// Check response status
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return nil, fmt.Errorf("%w: status %d - %s", domain.ErrInvalidRequest, resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("%w: status %d - %s", domain.ErrLMStudioUnavailable, resp.StatusCode, string(body))
	}

	// Create buffered channel for chunks
	chunkChan := make(chan domain.ChatCompletionChunk, streamingChannelBufferSize)

	// Launch goroutine to parse SSE and emit chunks
	go a.processStreamingResponse(ctx, resp, chunkChan)

	logrus.Infof("Started streaming chat completion with model: %s", model)

	return chunkChan, nil
}

// processStreamingResponse parses SSE from response body and sends chunks to channel
// This runs in a goroutine and is responsible for closing the channel when done
func (a *LMStudioClientAdapter) processStreamingResponse(ctx context.Context, resp *http.Response, chunkChan chan<- domain.ChatCompletionChunk) {
	// Ensure cleanup happens
	defer func() {
		resp.Body.Close()
		close(chunkChan)
		logrus.Debug("Streaming response processing completed, channel closed")
	}()

	scanner := bufio.NewScanner(resp.Body)

	for {
		// Check context cancellation before reading
		select {
		case <-ctx.Done():
			logrus.Debug("Streaming cancelled by context")
			// Send error chunk for cancellation
			a.sendChunk(chunkChan, domain.ChatCompletionChunk{
				Done:  true,
				Error: fmt.Errorf("streaming cancelled: %w", ctx.Err()),
			})
			return
		default:
		}

		// Read next line
		if !scanner.Scan() {
			// Check for scanner error
			if err := scanner.Err(); err != nil {
				logrus.Errorf("Error reading streaming response: %v", err)
				a.sendChunk(chunkChan, domain.ChatCompletionChunk{
					Done:  true,
					Error: fmt.Errorf("failed to read streaming response: %w", err),
				})
			} else {
				// EOF reached without [DONE] - treat as normal completion
				logrus.Debug("Streaming EOF reached")
				a.sendChunk(chunkChan, domain.ChatCompletionChunk{
					Done: true,
				})
			}
			return
		}

		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse SSE line
		chunk, done, err := a.parseSSELine(line)
		if err != nil {
			logrus.Warnf("Error parsing SSE line: %v, line: %s", err, line)
			continue // Skip malformed lines but continue processing
		}

		// Check if stream is done
		if done {
			logrus.Debug("Received [DONE] marker, completing stream")
			a.sendChunk(chunkChan, domain.ChatCompletionChunk{
				Done: true,
			})
			return
		}

		// Send chunk if we got content
		if chunk != nil {
			// Check context before sending
			select {
			case <-ctx.Done():
				logrus.Debug("Streaming cancelled by context during send")
				return
			default:
			}

			a.sendChunk(chunkChan, *chunk)
		}
	}
}

// sendChunk safely sends a chunk to the channel
// It handles the case where the channel might be blocked or closed
func (a *LMStudioClientAdapter) sendChunk(chunkChan chan<- domain.ChatCompletionChunk, chunk domain.ChatCompletionChunk) {
	// Non-blocking send with recovery for closed channel
	defer func() {
		if r := recover(); r != nil {
			logrus.Warnf("Failed to send chunk (channel may be closed): %v", r)
		}
	}()

	chunkChan <- chunk
}

// parseSSELine parses a single SSE line and extracts the chat completion chunk
// Returns (chunk, done, error) where:
//   - chunk: the parsed chunk (nil if line doesn't contain content)
//   - done: true if this is the [DONE] marker
//   - error: parsing error (non-fatal, caller should continue)
func (a *LMStudioClientAdapter) parseSSELine(line string) (*domain.ChatCompletionChunk, bool, error) {
	// SSE lines start with "data: "
	if !strings.HasPrefix(line, "data: ") {
		// Not a data line, skip it (could be event:, id:, retry:, or comment)
		return nil, false, nil
	}

	// Extract the data portion
	data := strings.TrimPrefix(line, "data: ")

	// Check for [DONE] marker
	if data == "[DONE]" {
		return nil, true, nil
	}

	// Parse JSON data
	var sseResp chatCompletionStreamResponse
	if err := json.Unmarshal([]byte(data), &sseResp); err != nil {
		return nil, false, fmt.Errorf("failed to parse SSE JSON: %w", err)
	}

	// Extract content from delta
	if len(sseResp.Choices) == 0 {
		return nil, false, nil
	}

	content := sseResp.Choices[0].Delta.Content

	// If content is empty, still return a valid chunk (some chunks might just have role info)
	chunk := &domain.ChatCompletionChunk{
		Content: content,
		Done:    false,
		Error:   nil,
	}

	return chunk, false, nil
}

// API request/response structures for LM Studio's OpenAI-compatible API

// chatMessageAPI represents a message in the API request
type chatMessageAPI struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatCompletionAPIRequest represents the request body for chat completions
type chatCompletionAPIRequest struct {
	Model       string           `json:"model"`
	Messages    []chatMessageAPI `json:"messages"`
	Stream      bool             `json:"stream"`
	Temperature *float64         `json:"temperature,omitempty"`
}

// chatCompletionAPIResponse represents the response from non-streaming chat completions
type chatCompletionAPIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// chatCompletionStreamResponse represents a single SSE chunk from streaming chat completions
type chatCompletionStreamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason,omitempty"`
	} `json:"choices"`
}

// modelsResponse represents the response from the /v1/models endpoint
type modelsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}
