package line

import (
	"fmt"

	"golang-template/internal/domain"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/sirupsen/logrus"
)

// LineClientAdapter struct - Output adapter for LINE messaging platform
type LineClientAdapter struct {
	client *messaging_api.MessagingApiAPI
}

// NewLineClientAdapter func - Creates new LINE client adapter
func NewLineClientAdapter(channelToken string) (*LineClientAdapter, error) {
	client, err := messaging_api.NewMessagingApiAPI(channelToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create LINE messaging API client: %w", err)
	}

	return &LineClientAdapter{
		client: client,
	}, nil
}

// ReplyMessage - Sends reply messages to LINE user via reply token
func (a *LineClientAdapter) ReplyMessage(request domain.LineReplyMessageRequest) (*domain.LineMessageResponse, error) {
	// Convert domain messages to LINE SDK messages
	messages := make([]messaging_api.MessageInterface, 0, len(request.Messages))

	for _, msg := range request.Messages {
		lineMsg, err := a.convertToLineMessage(msg)
		if err != nil {
			logrus.Errorf("Failed to convert message: %v", err)
			continue
		}
		messages = append(messages, lineMsg)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no valid messages to send")
	}

	// Send reply via LINE SDK
	req := &messaging_api.ReplyMessageRequest{
		ReplyToken: request.ReplyToken,
		Messages:   messages,
	}

	_, err := a.client.ReplyMessage(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send reply message: %w", err)
	}

	logrus.Infof("Successfully sent reply message with token: %s", request.ReplyToken)

	return &domain.LineMessageResponse{
		Status:  "success",
		Message: "Reply message sent successfully",
	}, nil
}

// PushMessage - Sends push messages to LINE user directly
func (a *LineClientAdapter) PushMessage(request domain.LinePushMessageRequest) (*domain.LineMessageResponse, error) {
	// Convert domain messages to LINE SDK messages
	messages := make([]messaging_api.MessageInterface, 0, len(request.Messages))

	for _, msg := range request.Messages {
		lineMsg, err := a.convertToLineMessage(msg)
		if err != nil {
			logrus.Errorf("Failed to convert message: %v", err)
			continue
		}
		messages = append(messages, lineMsg)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no valid messages to send")
	}

	// Send push message via LINE SDK
	req := &messaging_api.PushMessageRequest{
		To:       request.To,
		Messages: messages,
	}

	_, err := a.client.PushMessage(req, "")
	if err != nil {
		return nil, fmt.Errorf("failed to send push message: %w", err)
	}

	logrus.Infof("Successfully sent push message to: %s", request.To)

	return &domain.LineMessageResponse{
		Status:  "success",
		Message: "Push message sent successfully",
	}, nil
}

// GetProfile - Gets user profile information
func (a *LineClientAdapter) GetProfile(userID string) (interface{}, error) {
	profile, err := a.client.GetProfile(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	return profile, nil
}

// convertToLineMessage - Helper function to convert domain message to LINE SDK message
func (a *LineClientAdapter) convertToLineMessage(msg domain.LineOutgoingMessage) (messaging_api.MessageInterface, error) {
	switch msg.Type {
	case domain.LineMessageTypeText:
		return &messaging_api.TextMessage{
			Text: msg.Text,
		}, nil

	case domain.LineMessageTypeSticker:
		return &messaging_api.StickerMessage{
			PackageId: msg.PackageID,
			StickerId: msg.StickerID,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported message type: %s", msg.Type)
	}
}
