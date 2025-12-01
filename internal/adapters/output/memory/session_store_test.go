package memory

import (
	"testing"
	"time"

	"golang-template/internal/domain"
)

// Default test configuration values
const (
	testTimeout  = 30 * time.Minute
	testMaxTurns = 10
)

// ============================================================================
// Task 3.1: Focused tests for memory session store with config
// ============================================================================

// TestNewMemorySessionStoreAcceptsTimeoutAndMaxTurnsParameters tests that
// NewMemorySessionStore accepts timeout and maxTurns parameters and stores them correctly.
func TestNewMemorySessionStoreAcceptsTimeoutAndMaxTurnsParameters(t *testing.T) {
	timeout := 45 * time.Minute
	maxTurns := 15

	store := NewMemorySessionStore(timeout, maxTurns)

	if store == nil {
		t.Fatal("expected NewMemorySessionStore to return non-nil store")
	}

	if store.GetTimeout() != timeout {
		t.Errorf("expected timeout %v, got %v", timeout, store.GetTimeout())
	}

	if store.GetMaxTurns() != maxTurns {
		t.Errorf("expected maxTurns %d, got %d", maxTurns, store.GetMaxTurns())
	}
}

// TestGetSessionReturnsNilForExpiredSessionWithConfiguredTimeout tests that
// GetSession returns nil when a session has expired based on configured timeout,
// demonstrating that the session store correctly handles sessions created with config values.
func TestGetSessionReturnsNilForExpiredSessionWithConfiguredTimeout(t *testing.T) {
	// Use a short timeout for testing
	timeout := 5 * time.Minute
	maxTurns := 10

	store := NewMemorySessionStore(timeout, maxTurns)

	// Create a session using the store's configured values
	session := domain.NewConversationSession("U1234567890abcdef", store.GetTimeout(), store.GetMaxTurns())

	// Manually set LastAccessTime to simulate an expired session
	session.LastAccessTime = time.Now().Add(-6 * time.Minute)

	// Store directly in sync.Map to bypass UpdateSession's LastAccessTime update
	store.sessions.Store("U1234567890abcdef", session)

	// Try to retrieve the expired session
	retrieved, err := store.GetSession("U1234567890abcdef")
	if err != nil {
		t.Errorf("expected no error on GetSession, got %v", err)
	}

	// Should return nil for expired session
	if retrieved != nil {
		t.Error("expected nil for expired session, got non-nil")
	}
}

// TestSessionStorePassesConfigToNewConversationSession tests that
// the store's configuration values can be used to create new ConversationSession instances.
func TestSessionStorePassesConfigToNewConversationSession(t *testing.T) {
	timeout := 60 * time.Minute
	maxTurns := 20

	store := NewMemorySessionStore(timeout, maxTurns)

	// Create a session using the store's config values (simulating what application layer would do)
	session := domain.NewConversationSession("U1234567890abcdef", store.GetTimeout(), store.GetMaxTurns())

	if session == nil {
		t.Fatal("expected session to be created, got nil")
	}

	// Verify session is not expired (within timeout)
	if session.IsExpired() {
		t.Error("expected fresh session to not be expired")
	}

	// Store and retrieve the session
	err := store.UpdateSession(session)
	if err != nil {
		t.Errorf("expected no error on UpdateSession, got %v", err)
	}

	retrieved, err := store.GetSession("U1234567890abcdef")
	if err != nil {
		t.Errorf("expected no error on GetSession, got %v", err)
	}

	if retrieved == nil {
		t.Fatal("expected session to be retrieved, got nil")
	}

	if retrieved.UserID != "U1234567890abcdef" {
		t.Errorf("expected UserID 'U1234567890abcdef', got %s", retrieved.UserID)
	}
}

// ============================================================================
// Existing tests (updated to use new signatures)
// ============================================================================

// TestGetSessionReturnsNilForNonExistentUser tests that GetSession returns nil for a user that does not exist
func TestGetSessionReturnsNilForNonExistentUser(t *testing.T) {
	store := NewMemorySessionStore(testTimeout, testMaxTurns)

	session, err := store.GetSession("non-existent-user")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if session != nil {
		t.Error("expected nil session for non-existent user, got non-nil")
	}
}

// TestUpdateSessionCreatesNewSession tests that UpdateSession creates a new session correctly
func TestUpdateSessionCreatesNewSession(t *testing.T) {
	store := NewMemorySessionStore(testTimeout, testMaxTurns)

	session := domain.NewConversationSession("U1234567890abcdef", testTimeout, testMaxTurns)
	session.AddTurn(
		domain.ChatMessage{Role: domain.ChatMessageRoleUser, Content: "Hello"},
		domain.ChatMessage{Role: domain.ChatMessageRoleAssistant, Content: "Hi there!"},
	)

	err := store.UpdateSession(session)
	if err != nil {
		t.Errorf("expected no error on UpdateSession, got %v", err)
	}

	// Retrieve the session
	retrieved, err := store.GetSession("U1234567890abcdef")
	if err != nil {
		t.Errorf("expected no error on GetSession, got %v", err)
	}

	if retrieved == nil {
		t.Fatal("expected session to be retrieved, got nil")
	}

	if retrieved.UserID != "U1234567890abcdef" {
		t.Errorf("expected UserID 'U1234567890abcdef', got %s", retrieved.UserID)
	}

	history := retrieved.GetHistory()
	if len(history) != 2 {
		t.Errorf("expected 2 messages in history, got %d", len(history))
	}
}

// TestGetSessionRetrievesExistingSessionWithUpdatedLastAccessTime tests that GetSession retrieves existing session and updates LastAccessTime
func TestGetSessionRetrievesExistingSessionWithUpdatedLastAccessTime(t *testing.T) {
	store := NewMemorySessionStore(testTimeout, testMaxTurns)

	// Create and store a session
	session := domain.NewConversationSession("U1234567890abcdef", testTimeout, testMaxTurns)
	originalAccessTime := time.Now().Add(-10 * time.Minute)
	session.LastAccessTime = originalAccessTime

	err := store.UpdateSession(session)
	if err != nil {
		t.Fatalf("expected no error on UpdateSession, got %v", err)
	}

	// Wait a small amount to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Retrieve the session
	retrieved, err := store.GetSession("U1234567890abcdef")
	if err != nil {
		t.Errorf("expected no error on GetSession, got %v", err)
	}

	if retrieved == nil {
		t.Fatal("expected session to be retrieved, got nil")
	}

	// LastAccessTime should be updated to a more recent time
	if !retrieved.LastAccessTime.After(originalAccessTime) {
		t.Errorf("expected LastAccessTime to be updated after retrieval, original: %v, retrieved: %v",
			originalAccessTime, retrieved.LastAccessTime)
	}
}

// TestExpiredSessionIsTreatedAsNew tests that expired sessions are removed and nil is returned (lazy cleanup)
func TestExpiredSessionIsTreatedAsNew(t *testing.T) {
	store := NewMemorySessionStore(testTimeout, testMaxTurns)

	// Create and store an expired session
	session := domain.NewConversationSession("U1234567890abcdef", testTimeout, testMaxTurns)
	session.LastAccessTime = time.Now().Add(-31 * time.Minute) // 31 minutes ago (expired)
	session.AddTurn(
		domain.ChatMessage{Role: domain.ChatMessageRoleUser, Content: "Old message"},
		domain.ChatMessage{Role: domain.ChatMessageRoleAssistant, Content: "Old response"},
	)

	// Store directly in the sync.Map to bypass UpdateSession's LastAccessTime update
	store.sessions.Store("U1234567890abcdef", session)

	// Try to retrieve the expired session
	retrieved, err := store.GetSession("U1234567890abcdef")
	if err != nil {
		t.Errorf("expected no error on GetSession, got %v", err)
	}

	// Should return nil for expired session
	if retrieved != nil {
		t.Error("expected nil for expired session, got non-nil")
	}

	// Verify the session was deleted (lazy cleanup)
	_, exists := store.sessions.Load("U1234567890abcdef")
	if exists {
		t.Error("expected expired session to be deleted from store")
	}
}

// TestDeleteSessionRemovesSession tests that DeleteSession removes a session
func TestDeleteSessionRemovesSession(t *testing.T) {
	store := NewMemorySessionStore(testTimeout, testMaxTurns)

	// Create and store a session
	session := domain.NewConversationSession("U1234567890abcdef", testTimeout, testMaxTurns)
	err := store.UpdateSession(session)
	if err != nil {
		t.Fatalf("expected no error on UpdateSession, got %v", err)
	}

	// Verify session exists
	retrieved, _ := store.GetSession("U1234567890abcdef")
	if retrieved == nil {
		t.Fatal("expected session to exist before deletion")
	}

	// Delete the session
	err = store.DeleteSession("U1234567890abcdef")
	if err != nil {
		t.Errorf("expected no error on DeleteSession, got %v", err)
	}

	// Verify session is deleted
	retrieved, err = store.GetSession("U1234567890abcdef")
	if err != nil {
		t.Errorf("expected no error on GetSession after deletion, got %v", err)
	}
	if retrieved != nil {
		t.Error("expected nil session after deletion, got non-nil")
	}
}

// TestDeleteSessionIsIdempotent tests that deleting a non-existent session does not return an error
func TestDeleteSessionIsIdempotent(t *testing.T) {
	store := NewMemorySessionStore(testTimeout, testMaxTurns)

	// Delete a session that doesn't exist
	err := store.DeleteSession("non-existent-user")
	if err != nil {
		t.Errorf("expected no error when deleting non-existent session, got %v", err)
	}
}
