package output

import (
	"context"

	"golang-template/internal/domain"
)

// LMStudioClient interface - Output port
// Defines what the application needs from LM Studio's OpenAI-compatible API
// for sending prompts and receiving AI-generated responses.
type LMStudioClient interface {
	// ChatCompletion sends a non-streaming chat completion request to LM Studio.
	// It accepts a ChatCompletionRequest containing messages, model, and optional parameters,
	// and returns a ChatCompletionResponse with the generated content and usage statistics.
	// Returns an error if the request fails or the response cannot be parsed.
	ChatCompletion(ctx context.Context, request domain.ChatCompletionRequest) (*domain.ChatCompletionResponse, error)

	// ChatCompletionStream sends a streaming chat completion request to LM Studio.
	// It returns a read-only channel that emits ChatCompletionChunk objects as the response
	// is generated. The channel is closed when the stream completes or an error occurs.
	// If an error occurs during streaming, a chunk with Error set and Done=true is sent
	// before the channel is closed.
	// Returns an error if the initial request fails before streaming begins.
	ChatCompletionStream(ctx context.Context, request domain.ChatCompletionRequest) (<-chan domain.ChatCompletionChunk, error)

	// ListModels queries the /v1/models endpoint to retrieve available models from LM Studio.
	// Returns a slice of ModelInfo containing model metadata (ID, object type, owner).
	// This can be used for model selection when no model is configured via environment variable.
	ListModels(ctx context.Context) ([]domain.ModelInfo, error)
}
