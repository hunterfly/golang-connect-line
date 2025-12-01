package domain

import "time"

// LineEventType represents the type of webhook event from LINE
type LineEventType string

const (
	// LineEventTypeMessage - Message event
	LineEventTypeMessage LineEventType = "message"
	// LineEventTypeFollow - Follow event
	LineEventTypeFollow LineEventType = "follow"
	// LineEventTypeUnfollow - Unfollow event
	LineEventTypeUnfollow LineEventType = "unfollow"
	// LineEventTypeJoin - Join event
	LineEventTypeJoin LineEventType = "join"
	// LineEventTypeLeave - Leave event
	LineEventTypeLeave LineEventType = "leave"
	// LineEventTypePostback - Postback event
	LineEventTypePostback LineEventType = "postback"
)

// LineMessageType represents the type of message
type LineMessageType string

const (
	// LineMessageTypeText - Text message
	LineMessageTypeText LineMessageType = "text"
	// LineMessageTypeImage - Image message
	LineMessageTypeImage LineMessageType = "image"
	// LineMessageTypeVideo - Video message
	LineMessageTypeVideo LineMessageType = "video"
	// LineMessageTypeAudio - Audio message
	LineMessageTypeAudio LineMessageType = "audio"
	// LineMessageTypeFile - File message
	LineMessageTypeFile LineMessageType = "file"
	// LineMessageTypeLocation - Location message
	LineMessageTypeLocation LineMessageType = "location"
	// LineMessageTypeSticker - Sticker message
	LineMessageTypeSticker LineMessageType = "sticker"
)

// LineSourceType represents the source type of the event
type LineSourceType string

const (
	// LineSourceTypeUser - User source
	LineSourceTypeUser LineSourceType = "user"
	// LineSourceTypeGroup - Group source
	LineSourceTypeGroup LineSourceType = "group"
	// LineSourceTypeRoom - Room source
	LineSourceTypeRoom LineSourceType = "room"
)

// LineWebhookEvent represents a LINE webhook event (domain entity)
type LineWebhookEvent struct {
	ID         string
	Type       LineEventType
	Timestamp  time.Time
	Source     LineSource
	ReplyToken string
	Message    *LineMessage
}

// LineSource represents the source of the event
type LineSource struct {
	Type    LineSourceType
	UserID  string
	GroupID string
	RoomID  string
}

// LineMessage represents a message from LINE
type LineMessage struct {
	ID        string
	Type      LineMessageType
	Text      string
	PackageID string // For sticker
	StickerID string // For sticker
}
