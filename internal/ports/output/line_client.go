package output

import "golang-template/internal/domain"

// LineClient interface - Output port
// Defines what the application needs from LINE messaging platform
type LineClient interface {
	// ReplyMessage sends reply messages to LINE user via reply token
	ReplyMessage(request domain.LineReplyMessageRequest) (*domain.LineMessageResponse, error)

	// PushMessage sends push messages to LINE user directly
	PushMessage(request domain.LinePushMessageRequest) (*domain.LineMessageResponse, error)

	// GetProfile gets user profile information
	GetProfile(userID string) (interface{}, error)
}
