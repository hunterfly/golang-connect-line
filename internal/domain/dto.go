package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DTOs (Data Transfer Objects) - Domain layer request/response structures

type (
	// TodoRequest struct - Domain request DTO
	TodoRequest struct {
		ID          *uuid.UUID  `json:"id"`
		Title       *string     `json:"title"`
		Description *string     `json:"description"`
		Date        *string     `json:"date"`
		Image       *string     `json:"image"`
		Status      *TodoStatus `json:"status"`
	}

	// QueryTodoRequest struct - Domain query request DTO
	QueryTodoRequest struct {
		ID          *uuid.UUID
		Title       *string
		Description *string
		Status      *string

		Limit      *int
		Page       *int
		OrderBy    *string
		Asc        *bool
		Pagination *Pagination
		SortMethod *SortMethod
	}

	// Pagination struct
	Pagination struct {
		Limit  int
		Offset int
	}

	// SortMethod struct
	SortMethod struct {
		Asc     bool
		OrderBy string
	}

	// TodoResponse struct - Domain response DTO
	TodoResponse struct {
		ID          *uuid.UUID      `json:"id,omitempty"`
		Title       *string         `json:"title,omitempty"`
		Description *string         `json:"description,omitempty"`
		Date        *string         `json:"date,omitempty"`
		Image       *string         `json:"image,omitempty"`
		Status      *TodoStatus     `json:"status,omitempty"`
		CreatedAt   *time.Time      `json:"created_at,omitempty"`
		UpdatedAt   *time.Time      `json:"updated_at,omitempty"`
		DeletedAt   *gorm.DeletedAt `json:"deleted_at,omitempty"`
	}

	// TodoListResponse struct - Domain list response DTO
	TodoListResponse struct {
		Todos       []TodoResponse
		CurrentPage *int
		PerPage     *int
		TotalItem   *int64
	}

	// LineWebhookRequest struct - Domain LINE webhook request DTO
	LineWebhookRequest struct {
		Events []LineWebhookEvent
	}

	// LineReplyMessageRequest struct - Domain LINE reply message request DTO
	LineReplyMessageRequest struct {
		ReplyToken string
		Messages   []LineOutgoingMessage
	}

	// LinePushMessageRequest struct - Domain LINE push message request DTO
	LinePushMessageRequest struct {
		To       string
		Messages []LineOutgoingMessage
	}

	// LineOutgoingMessage struct - Domain LINE outgoing message DTO
	LineOutgoingMessage struct {
		Type      LineMessageType
		Text      string
		PackageID string // For sticker
		StickerID string // For sticker
	}

	// LineMessageResponse struct - Domain LINE API response DTO
	LineMessageResponse struct {
		Status  string
		Message string
	}
)

// LM Studio DTOs - Chat completion request/response structures

// ChatMessageRole represents the role in a chat message
type ChatMessageRole string

const (
	// ChatMessageRoleSystem - System message role
	ChatMessageRoleSystem ChatMessageRole = "system"
	// ChatMessageRoleUser - User message role
	ChatMessageRoleUser ChatMessageRole = "user"
	// ChatMessageRoleAssistant - Assistant message role
	ChatMessageRoleAssistant ChatMessageRole = "assistant"
)

type (
	// ChatMessage struct - Domain chat message DTO for LM Studio
	ChatMessage struct {
		Role    ChatMessageRole `json:"role"`
		Content string          `json:"content"`
	}

	// ChatCompletionRequest struct - Domain chat completion request DTO
	ChatCompletionRequest struct {
		Messages    []ChatMessage `json:"messages"`
		Model       *string       `json:"model,omitempty"`
		Stream      bool          `json:"stream"`
		Temperature *float64      `json:"temperature,omitempty"`
	}

	// ChatCompletionResponse struct - Domain chat completion response DTO
	ChatCompletionResponse struct {
		Content          string `json:"content"`
		Model            string `json:"model"`
		PromptTokens     int    `json:"prompt_tokens"`
		CompletionTokens int    `json:"completion_tokens"`
		TotalTokens      int    `json:"total_tokens"`
	}

	// ChatCompletionChunk struct - Domain chat completion chunk DTO for streaming
	ChatCompletionChunk struct {
		Content string `json:"content"`
		Done    bool   `json:"done"`
		Error   error  `json:"-"`
	}

	// ModelInfo struct - Domain model information DTO
	ModelInfo struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		OwnedBy string `json:"owned_by"`
	}
)
