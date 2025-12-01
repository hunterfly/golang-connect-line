package output

import "golang-template/internal/domain"

// SessionStore interface - Output port
// Defines what the application needs for managing conversation sessions.
// Sessions store conversation history per LINE user to enable multi-turn dialogue
// with the LLM. Implementations must be thread-safe for concurrent access.
type SessionStore interface {
	// GetSession retrieves a conversation session by LINE user ID.
	// Returns the session if found and not expired, or nil if the session
	// does not exist or has expired. Implementations should perform lazy cleanup
	// of expired sessions and update LastAccessTime for valid sessions.
	// Returns an error only if there is a storage access failure.
	GetSession(userID string) (*domain.ConversationSession, error)

	// UpdateSession creates or updates a conversation session.
	// The session's LastAccessTime should be updated to the current time.
	// If a session with the same UserID already exists, it will be overwritten.
	// Returns an error if the session cannot be stored.
	UpdateSession(session *domain.ConversationSession) error

	// DeleteSession removes a conversation session by LINE user ID.
	// This operation is idempotent - deleting a non-existent session
	// should not return an error.
	// Returns an error only if there is a storage access failure.
	DeleteSession(userID string) error
}
