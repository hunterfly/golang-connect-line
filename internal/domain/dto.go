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
