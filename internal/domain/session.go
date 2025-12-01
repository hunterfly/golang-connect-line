package domain

import "time"

// ConversationSession represents a conversation session for a LINE user
type ConversationSession struct {
	UserID         string        // LINE user identifier
	Messages       []ChatMessage // Conversation history
	LastAccessTime time.Time     // For session expiration checking
	timeout        time.Duration // Configurable session timeout
	maxTurns       int           // Configurable maximum conversation turns
}

// NewConversationSession creates a new conversation session for a user
// with configurable timeout and maxTurns parameters
func NewConversationSession(userID string, timeout time.Duration, maxTurns int) *ConversationSession {
	return &ConversationSession{
		UserID:         userID,
		Messages:       make([]ChatMessage, 0),
		LastAccessTime: time.Now(),
		timeout:        timeout,
		maxTurns:       maxTurns,
	}
}

// IsExpired checks if the session has exceeded the configured timeout
func (s *ConversationSession) IsExpired() bool {
	return time.Since(s.LastAccessTime) > s.timeout
}

// AddTurn adds a user message and assistant response to the conversation
// If the history limit is reached, the oldest turn (2 messages) is removed
func (s *ConversationSession) AddTurn(userMsg, assistantMsg ChatMessage) {
	// Check if at maximum turns and remove oldest turn if needed
	if len(s.Messages) >= s.maxTurns*2 {
		// Remove the oldest turn (first 2 messages)
		s.Messages = s.Messages[2:]
	}

	// Add the new turn
	s.Messages = append(s.Messages, userMsg, assistantMsg)
}

// GetHistory returns a copy of the conversation history
func (s *ConversationSession) GetHistory() []ChatMessage {
	if len(s.Messages) == 0 {
		return []ChatMessage{}
	}

	// Return a copy to prevent external modification
	history := make([]ChatMessage, len(s.Messages))
	copy(history, s.Messages)
	return history
}
