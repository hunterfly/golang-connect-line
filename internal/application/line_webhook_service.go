package application

import (
	"fmt"
	"strings"

	"golang-template/internal/domain"
	"golang-template/internal/ports/output"

	"github.com/sirupsen/logrus"
)

// LineWebhookService struct - Application service implementing LINE webhook use cases
type LineWebhookService struct {
	lineClient output.LineClient
	todoRepo   output.TodoRepository
}

// NewLineWebhookService func - Creates new LINE webhook service
func NewLineWebhookService(lineClient output.LineClient, todoRepo output.TodoRepository) *LineWebhookService {
	return &LineWebhookService{
		lineClient: lineClient,
		todoRepo:   todoRepo,
	}
}

// HandleWebhook func - Use case: Handle incoming webhook events from LINE
func (s *LineWebhookService) HandleWebhook(request domain.LineWebhookRequest) error {
	// Process each event
	for _, event := range request.Events {
		logrus.Infof("Received LINE event: type=%s, source=%s, userID=%s",
			event.Type, event.Source.Type, event.Source.UserID)

		switch event.Type {
		case domain.LineEventTypeMessage:
			if err := s.handleMessageEvent(event); err != nil {
				logrus.Errorf("Failed to handle message event: %v", err)
				return err
			}

		case domain.LineEventTypeFollow:
			if err := s.handleFollowEvent(event); err != nil {
				logrus.Errorf("Failed to handle follow event: %v", err)
				return err
			}

		case domain.LineEventTypeUnfollow:
			if err := s.handleUnfollowEvent(event); err != nil {
				logrus.Errorf("Failed to handle unfollow event: %v", err)
				return err
			}

		default:
			logrus.Infof("Unhandled event type: %s", event.Type)
		}
	}

	return nil
}

// handleMessageEvent - Business logic for message events
func (s *LineWebhookService) handleMessageEvent(event domain.LineWebhookEvent) error {
	if event.Message == nil {
		return nil
	}

	// Only handle text messages
	if event.Message.Type != domain.LineMessageTypeText {
		logrus.Infof("Ignoring non-text message: type=%s", event.Message.Type)
		return nil
	}

	text := strings.TrimSpace(event.Message.Text)
	var replyMessages []domain.LineOutgoingMessage

	// Command routing - Business logic
	if strings.HasPrefix(text, "/") {
		replyMessages = s.handleCommand(text, event.Source.UserID)
	} else {
		// Echo message back
		replyMessages = []domain.LineOutgoingMessage{
			{
				Type: domain.LineMessageTypeText,
				Text: fmt.Sprintf("You said: %s", text),
			},
		}
	}

	// Send reply via LINE client
	if len(replyMessages) > 0 && event.ReplyToken != "" {
		replyReq := domain.LineReplyMessageRequest{
			ReplyToken: event.ReplyToken,
			Messages:   replyMessages,
		}

		if _, err := s.lineClient.ReplyMessage(replyReq); err != nil {
			return fmt.Errorf("failed to send reply: %w", err)
		}
	}

	return nil
}

// handleCommand - Business logic for command processing
func (s *LineWebhookService) handleCommand(text, userID string) []domain.LineOutgoingMessage {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return nil
	}

	command := strings.ToLower(parts[0])

	switch command {
	case "/help":
		return []domain.LineOutgoingMessage{
			{
				Type: domain.LineMessageTypeText,
				Text: "Available commands:\n/help - Show this message\n/about - About this bot\n/echo <text> - Echo your message",
			},
		}

	case "/about":
		return []domain.LineOutgoingMessage{
			{
				Type: domain.LineMessageTypeText,
				Text: "LINE Bot powered by Go + Fiber\nBuilt with Hexagonal Architecture",
			},
		}

	case "/echo":
		if len(parts) > 1 {
			echoText := strings.Join(parts[1:], " ")
			return []domain.LineOutgoingMessage{
				{
					Type: domain.LineMessageTypeText,
					Text: echoText,
				},
			}
		}
		return []domain.LineOutgoingMessage{
			{
				Type: domain.LineMessageTypeText,
				Text: "Usage: /echo <text>",
			},
		}

	default:
		return []domain.LineOutgoingMessage{
			{
				Type: domain.LineMessageTypeText,
				Text: fmt.Sprintf("Unknown command: %s\nType /help for available commands", command),
			},
		}
	}
}

// handleFollowEvent - Business logic for follow events
func (s *LineWebhookService) handleFollowEvent(event domain.LineWebhookEvent) error {
	logrus.Infof("User followed: userID=%s", event.Source.UserID)

	// Send welcome message
	welcomeMsg := domain.LinePushMessageRequest{
		To: event.Source.UserID,
		Messages: []domain.LineOutgoingMessage{
			{
				Type: domain.LineMessageTypeText,
				Text: "Welcome! Thank you for adding me as a friend!\n\nType /help to see available commands.",
			},
		},
	}

	if _, err := s.lineClient.PushMessage(welcomeMsg); err != nil {
		return fmt.Errorf("failed to send welcome message: %w", err)
	}

	return nil
}

// handleUnfollowEvent - Business logic for unfollow events
func (s *LineWebhookService) handleUnfollowEvent(event domain.LineWebhookEvent) error {
	logrus.Infof("User unfollowed: userID=%s", event.Source.UserID)
	// Could save to database for analytics
	return nil
}
