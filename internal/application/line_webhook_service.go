package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang-template/internal/domain"
	"golang-template/internal/ports/output"

	"github.com/sirupsen/logrus"
)

// User-friendly error message constant
const lmStudioErrorMessage = "Sorry, I'm having trouble processing your request right now. Please try again later."

// Maximum user input length before truncation (4000 characters)
const maxUserInputLength = 4000

// AI response splitting constants
const maxLineMessageLength = 5000
const sentenceBoundaryLookback = 200
const maxMessagesPerResponse = 5

// LineWebhookService struct - Application service implementing LINE webhook use cases
type LineWebhookService struct {
	lineClient      output.LineClient
	lmStudioClient  output.LMStudioClient
	sessionStore    output.SessionStore
	systemPrompt    string
	sessionTimeout  time.Duration
	sessionMaxTurns int
}

// NewLineWebhookService func - Creates new LINE webhook service
func NewLineWebhookService(
	lineClient output.LineClient,
	lmStudioClient output.LMStudioClient,
	sessionStore output.SessionStore,
	systemPrompt string,
	sessionTimeout time.Duration,
	sessionMaxTurns int,
) *LineWebhookService {
	return &LineWebhookService{
		lineClient:      lineClient,
		lmStudioClient:  lmStudioClient,
		sessionStore:    sessionStore,
		systemPrompt:    systemPrompt,
		sessionTimeout:  sessionTimeout,
		sessionMaxTurns: sessionMaxTurns,
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

// truncateUserInput - Helper method to truncate user input if it exceeds maxUserInputLength
// Messages under 4000 characters are returned unchanged.
// Messages over 4000 characters are truncated to exactly 4000 characters.
// A warning is logged when truncation occurs.
func (s *LineWebhookService) truncateUserInput(text string) string {
	if len(text) <= maxUserInputLength {
		return text
	}

	originalLen := len(text)
	truncated := text[:maxUserInputLength]
	logrus.Warnf("User input truncated from %d to %d characters", originalLen, maxUserInputLength)
	return truncated
}

// buildChatRequest - Helper method to build ChatCompletionRequest with system prompt, history, and user message
// Format: [system prompt] + [conversation history] + [new user message]
func (s *LineWebhookService) buildChatRequest(userMessage string, history []domain.ChatMessage) domain.ChatCompletionRequest {
	// Build messages array: [system] + [history] + [new user message]
	messages := make([]domain.ChatMessage, 0, len(history)+2)

	// Add system prompt first
	messages = append(messages, domain.ChatMessage{
		Role:    domain.ChatMessageRoleSystem,
		Content: s.systemPrompt,
	})

	// Add conversation history between system prompt and new user message
	messages = append(messages, history...)

	// Add new user message last
	messages = append(messages, domain.ChatMessage{
		Role:    domain.ChatMessageRoleUser,
		Content: userMessage,
	})

	return domain.ChatCompletionRequest{
		Messages: messages,
		Stream:   false,
	}
}

// splitAIResponse - Helper method to split AI response into multiple messages if it exceeds LINE's limit
// If content <= 5000 chars, returns single-element slice
// If content > 5000 chars, finds sentence boundary within last 200 chars
// Looks for `.`, `!`, `?` followed by space or end of string
// Falls back to hard split at 5000 if no boundary found
// Limits output to 5 messages maximum
func (s *LineWebhookService) splitAIResponse(content string) []string {
	if len(content) <= maxLineMessageLength {
		return []string{content}
	}

	var messages []string
	remaining := content

	for len(remaining) > 0 && len(messages) < maxMessagesPerResponse {
		if len(remaining) <= maxLineMessageLength {
			messages = append(messages, remaining)
			break
		}

		// Find the best split point within the lookback window
		splitPoint := s.findSentenceBoundary(remaining, maxLineMessageLength)

		messages = append(messages, remaining[:splitPoint])
		remaining = remaining[splitPoint:]
	}

	return messages
}

// findSentenceBoundary - Helper method to find sentence boundary within lookback window
// Returns the position to split at (inclusive of the boundary character and following space)
func (s *LineWebhookService) findSentenceBoundary(text string, maxLen int) int {
	if len(text) <= maxLen {
		return len(text)
	}

	// Look for sentence boundary within the lookback window
	searchStart := maxLen - sentenceBoundaryLookback
	if searchStart < 0 {
		searchStart = 0
	}

	// Search from the end of the window backwards to find the latest sentence boundary
	bestBoundary := -1
	for i := maxLen - 1; i >= searchStart; i-- {
		if i < len(text) && isSentenceEnd(text, i) {
			// Include the space after the sentence boundary if present
			if i+1 < len(text) && text[i+1] == ' ' {
				bestBoundary = i + 2 // Include the boundary char and the space
			} else {
				bestBoundary = i + 1 // Include just the boundary char
			}
			break
		}
	}

	// If a sentence boundary was found, use it; otherwise, hard split at maxLen
	if bestBoundary > 0 {
		return bestBoundary
	}

	return maxLen
}

// isSentenceEnd - Helper to check if the character at position is a sentence ending
func isSentenceEnd(text string, pos int) bool {
	if pos < 0 || pos >= len(text) {
		return false
	}

	char := text[pos]
	if char != '.' && char != '!' && char != '?' {
		return false
	}

	// Check if followed by space or end of string
	nextPos := pos + 1
	if nextPos >= len(text) {
		return true
	}

	return text[nextPos] == ' '
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
	var aiResponseContent string
	var isError bool

	// Command routing - Business logic
	// Commands starting with "/" are handled by handleCommand method
	if strings.HasPrefix(text, "/") {
		replyMessages := s.handleCommand(text, event.Source.UserID)
		// Send command response via reply message
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

	// Retrieve conversation history from session store
	var history []domain.ChatMessage
	if s.sessionStore != nil {
		session, err := s.sessionStore.GetSession(event.Source.UserID)
		if err != nil {
			logrus.Warnf("Failed to get session for user %s: %v", event.Source.UserID, err)
		}
		if session != nil {
			history = session.GetHistory()
		}
	}

	// AI-powered message processing via LM Studio
	// Truncate user input if it exceeds maximum length
	truncatedText := s.truncateUserInput(text)
	chatRequest := s.buildChatRequest(truncatedText, history)

	// Call LM Studio for AI response
	response, err := s.lmStudioClient.ChatCompletion(context.Background(), chatRequest)
	if err != nil {
		// Error handling - check for specific LM Studio errors
		// Log full error details for debugging
		if errors.Is(err, domain.ErrLMStudioUnavailable) {
			logrus.Errorf("LM Studio error: %v", err)
		} else if errors.Is(err, domain.ErrLMStudioTimeout) {
			logrus.Errorf("LM Studio error: %v", err)
		} else {
			// Generic error - still log with full details
			logrus.Errorf("LM Studio error: %v", err)
		}

		// Return user-friendly message for all LM Studio errors
		// Do not expose technical error details to LINE user
		aiResponseContent = lmStudioErrorMessage
		isError = true
	} else {
		// Extract AI response content
		aiResponseContent = response.Content

		// Store conversation turn in session (only on success)
		if s.sessionStore != nil {
			// Get existing session or create new one
			session, _ := s.sessionStore.GetSession(event.Source.UserID)
			if session == nil {
				session = domain.NewConversationSession(event.Source.UserID, s.sessionTimeout, s.sessionMaxTurns)
			}

			// Create ChatMessage for user and assistant
			userMsg := domain.ChatMessage{
				Role:    domain.ChatMessageRoleUser,
				Content: truncatedText,
			}
			assistantMsg := domain.ChatMessage{
				Role:    domain.ChatMessageRoleAssistant,
				Content: aiResponseContent,
			}

			// Add turn and update session
			session.AddTurn(userMsg, assistantMsg)
			if err := s.sessionStore.UpdateSession(session); err != nil {
				logrus.Warnf("Failed to update session for user %s: %v", event.Source.UserID, err)
			}
		}
	}

	// Split AI response if it exceeds LINE's message length limit
	splitMessages := s.splitAIResponse(aiResponseContent)

	// For error messages, send as single message (won't be long)
	// For AI responses, send first via ReplyMessage, subsequent via PushMessage
	if isError || len(splitMessages) == 1 {
		// Single message - use ReplyMessage
		if event.ReplyToken != "" {
			replyReq := domain.LineReplyMessageRequest{
				ReplyToken: event.ReplyToken,
				Messages: []domain.LineOutgoingMessage{
					{
						Type: domain.LineMessageTypeText,
						Text: splitMessages[0],
					},
				},
			}

			if _, err := s.lineClient.ReplyMessage(replyReq); err != nil {
				return fmt.Errorf("failed to send reply: %w", err)
			}
		}
	} else {
		// Multiple messages - first via ReplyMessage, rest via PushMessage
		// Send first message via ReplyMessage
		if event.ReplyToken != "" {
			replyReq := domain.LineReplyMessageRequest{
				ReplyToken: event.ReplyToken,
				Messages: []domain.LineOutgoingMessage{
					{
						Type: domain.LineMessageTypeText,
						Text: splitMessages[0],
					},
				},
			}

			if _, err := s.lineClient.ReplyMessage(replyReq); err != nil {
				return fmt.Errorf("failed to send reply: %w", err)
			}
		}

		// Send subsequent messages via PushMessage
		for i := 1; i < len(splitMessages); i++ {
			pushReq := domain.LinePushMessageRequest{
				To: event.Source.UserID,
				Messages: []domain.LineOutgoingMessage{
					{
						Type: domain.LineMessageTypeText,
						Text: splitMessages[i],
					},
				},
			}

			if _, err := s.lineClient.PushMessage(pushReq); err != nil {
				logrus.Errorf("Failed to send push message %d: %v", i, err)
				// Continue sending remaining messages even if one fails
			}
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
				Text: "Available commands:\n/help - Show this message\n/about - About this bot\n/echo <text> - Echo your message\n/clear - Clear conversation history",
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

	case "/clear":
		// Clear conversation history by deleting the user's session
		if s.sessionStore != nil {
			_ = s.sessionStore.DeleteSession(userID)
		}
		return []domain.LineOutgoingMessage{
			{
				Type: domain.LineMessageTypeText,
				Text: "Conversation history cleared.",
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
