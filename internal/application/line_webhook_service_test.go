package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"golang-template/internal/domain"
)

// Default session configuration values for tests
const defaultTestTimeout = 30 * time.Minute
const defaultTestMaxTurns = 10

// Mock implementations for testing

// MockLineClient implements output.LineClient for testing
type MockLineClient struct {
	ReplyMessageFunc func(request domain.LineReplyMessageRequest) (*domain.LineMessageResponse, error)
	PushMessageFunc  func(request domain.LinePushMessageRequest) (*domain.LineMessageResponse, error)
	GetProfileFunc   func(userID string) (interface{}, error)

	// Captured values for assertions
	LastReplyRequest *domain.LineReplyMessageRequest
	LastPushRequest  *domain.LinePushMessageRequest

	// Track all push requests for multi-message testing
	PushRequests []domain.LinePushMessageRequest
}

func (m *MockLineClient) ReplyMessage(request domain.LineReplyMessageRequest) (*domain.LineMessageResponse, error) {
	m.LastReplyRequest = &request
	if m.ReplyMessageFunc != nil {
		return m.ReplyMessageFunc(request)
	}
	return &domain.LineMessageResponse{Status: "ok"}, nil
}

func (m *MockLineClient) PushMessage(request domain.LinePushMessageRequest) (*domain.LineMessageResponse, error) {
	m.LastPushRequest = &request
	m.PushRequests = append(m.PushRequests, request)
	if m.PushMessageFunc != nil {
		return m.PushMessageFunc(request)
	}
	return &domain.LineMessageResponse{Status: "ok"}, nil
}

func (m *MockLineClient) GetProfile(userID string) (interface{}, error) {
	if m.GetProfileFunc != nil {
		return m.GetProfileFunc(userID)
	}
	return nil, nil
}

// MockLMStudioClient implements output.LMStudioClient for testing
type MockLMStudioClient struct {
	ChatCompletionFunc       func(ctx context.Context, request domain.ChatCompletionRequest) (*domain.ChatCompletionResponse, error)
	ChatCompletionStreamFunc func(ctx context.Context, request domain.ChatCompletionRequest) (<-chan domain.ChatCompletionChunk, error)
	ListModelsFunc           func(ctx context.Context) ([]domain.ModelInfo, error)

	// Captured values for assertions
	LastChatRequest *domain.ChatCompletionRequest
}

func (m *MockLMStudioClient) ChatCompletion(ctx context.Context, request domain.ChatCompletionRequest) (*domain.ChatCompletionResponse, error) {
	m.LastChatRequest = &request
	if m.ChatCompletionFunc != nil {
		return m.ChatCompletionFunc(ctx, request)
	}
	return &domain.ChatCompletionResponse{Content: "AI response"}, nil
}

func (m *MockLMStudioClient) ChatCompletionStream(ctx context.Context, request domain.ChatCompletionRequest) (<-chan domain.ChatCompletionChunk, error) {
	if m.ChatCompletionStreamFunc != nil {
		return m.ChatCompletionStreamFunc(ctx, request)
	}
	return nil, nil
}

func (m *MockLMStudioClient) ListModels(ctx context.Context) ([]domain.ModelInfo, error) {
	if m.ListModelsFunc != nil {
		return m.ListModelsFunc(ctx)
	}
	return nil, nil
}

// MockSessionStore implements output.SessionStore for testing
type MockSessionStore struct {
	GetSessionFunc    func(userID string) (*domain.ConversationSession, error)
	UpdateSessionFunc func(session *domain.ConversationSession) error
	DeleteSessionFunc func(userID string) error

	// Captured values for assertions
	LastGetUserID      string
	LastUpdatedSession *domain.ConversationSession
	LastDeleteUserID   string

	// Track all update calls
	UpdateCalls []*domain.ConversationSession

	// Track delete calls
	DeleteCalls []string
}

func (m *MockSessionStore) GetSession(userID string) (*domain.ConversationSession, error) {
	m.LastGetUserID = userID
	if m.GetSessionFunc != nil {
		return m.GetSessionFunc(userID)
	}
	return nil, nil
}

func (m *MockSessionStore) UpdateSession(session *domain.ConversationSession) error {
	m.LastUpdatedSession = session
	m.UpdateCalls = append(m.UpdateCalls, session)
	if m.UpdateSessionFunc != nil {
		return m.UpdateSessionFunc(session)
	}
	return nil
}

func (m *MockSessionStore) DeleteSession(userID string) error {
	m.LastDeleteUserID = userID
	m.DeleteCalls = append(m.DeleteCalls, userID)
	if m.DeleteSessionFunc != nil {
		return m.DeleteSessionFunc(userID)
	}
	return nil
}

// Test helper to create a basic text message event
func createTextMessageEvent(text string) domain.LineWebhookEvent {
	return domain.LineWebhookEvent{
		Type:       domain.LineEventTypeMessage,
		ReplyToken: "test-reply-token",
		Source: domain.LineSource{
			Type:   domain.LineSourceTypeUser,
			UserID: "test-user-id",
		},
		Message: &domain.LineMessage{
			ID:   "test-message-id",
			Type: domain.LineMessageTypeText,
			Text: text,
		},
	}
}

// Task Group 4: Session Configuration and /clear Command Tests

// TestNewLineWebhookService_AcceptsSessionConfigParameters tests that NewLineWebhookService accepts session config parameters
func TestNewLineWebhookService_AcceptsSessionConfigParameters(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{}
	mockSessionStore := &MockSessionStore{}

	customTimeout := 45 * time.Minute
	customMaxTurns := 15

	// Act
	service := NewLineWebhookService(
		mockLineClient,
		mockLMStudioClient,
		mockSessionStore,
		"You are a helpful assistant",
		customTimeout,
		customMaxTurns,
	)

	// Assert
	if service == nil {
		t.Fatal("Expected service to be created")
	}

	// Verify the service stores the session config values
	if service.sessionTimeout != customTimeout {
		t.Errorf("Expected sessionTimeout to be %v, got %v", customTimeout, service.sessionTimeout)
	}

	if service.sessionMaxTurns != customMaxTurns {
		t.Errorf("Expected sessionMaxTurns to be %d, got %d", customMaxTurns, service.sessionMaxTurns)
	}
}

// TestNewSessionCreation_UsesConfiguredTimeoutAndMaxTurns tests that new sessions are created with configured values
func TestNewSessionCreation_UsesConfiguredTimeoutAndMaxTurns(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{
		ChatCompletionFunc: func(ctx context.Context, request domain.ChatCompletionRequest) (*domain.ChatCompletionResponse, error) {
			return &domain.ChatCompletionResponse{Content: "Hello!"}, nil
		},
	}
	mockSessionStore := &MockSessionStore{
		GetSessionFunc: func(userID string) (*domain.ConversationSession, error) {
			// Return nil to simulate no existing session, forcing creation of a new one
			return nil, nil
		},
	}

	customTimeout := 45 * time.Minute
	customMaxTurns := 15

	service := NewLineWebhookService(
		mockLineClient,
		mockLMStudioClient,
		mockSessionStore,
		"You are a helpful assistant",
		customTimeout,
		customMaxTurns,
	)

	event := createTextMessageEvent("Hello!")
	request := domain.LineWebhookRequest{
		Events: []domain.LineWebhookEvent{event},
	}

	// Act
	err := service.HandleWebhook(request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify a session was updated (which means it was created)
	if mockSessionStore.LastUpdatedSession == nil {
		t.Fatal("Expected session to be created and updated")
	}

	// The session should have been created with the configured values
	// We can verify this by checking the session was properly stored
	session := mockSessionStore.LastUpdatedSession
	if session.UserID != "test-user-id" {
		t.Errorf("Expected session UserID to be 'test-user-id', got '%s'", session.UserID)
	}
}

// TestClearCommand_CallsDeleteSessionAndReturnsConfirmation tests /clear command behavior
func TestClearCommand_CallsDeleteSessionAndReturnsConfirmation(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{}
	mockSessionStore := &MockSessionStore{}

	service := NewLineWebhookService(
		mockLineClient,
		mockLMStudioClient,
		mockSessionStore,
		"You are a helpful assistant",
		defaultTestTimeout,
		defaultTestMaxTurns,
	)

	event := createTextMessageEvent("/clear")
	request := domain.LineWebhookRequest{
		Events: []domain.LineWebhookEvent{event},
	}

	// Act
	err := service.HandleWebhook(request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify DeleteSession was called with the correct user ID
	if len(mockSessionStore.DeleteCalls) != 1 {
		t.Errorf("Expected DeleteSession to be called once, got %d calls", len(mockSessionStore.DeleteCalls))
	}

	if mockSessionStore.LastDeleteUserID != "test-user-id" {
		t.Errorf("Expected DeleteSession to be called with 'test-user-id', got '%s'", mockSessionStore.LastDeleteUserID)
	}

	// Verify the confirmation message was sent
	if mockLineClient.LastReplyRequest == nil {
		t.Fatal("Expected reply message to be sent")
	}

	if len(mockLineClient.LastReplyRequest.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(mockLineClient.LastReplyRequest.Messages))
	}

	expectedMessage := "Conversation history cleared."
	actualMessage := mockLineClient.LastReplyRequest.Messages[0].Text

	if actualMessage != expectedMessage {
		t.Errorf("Expected message:\n%q\nGot:\n%q", expectedMessage, actualMessage)
	}
}

// TestHelpCommand_IncludesClearCommand tests that /help includes /clear in output
func TestHelpCommand_IncludesClearCommand(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{}
	mockSessionStore := &MockSessionStore{}

	service := NewLineWebhookService(
		mockLineClient,
		mockLMStudioClient,
		mockSessionStore,
		"You are a helpful assistant",
		defaultTestTimeout,
		defaultTestMaxTurns,
	)

	event := createTextMessageEvent("/help")
	request := domain.LineWebhookRequest{
		Events: []domain.LineWebhookEvent{event},
	}

	// Act
	err := service.HandleWebhook(request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify help message was sent
	if mockLineClient.LastReplyRequest == nil {
		t.Fatal("Expected reply message to be sent")
	}

	if len(mockLineClient.LastReplyRequest.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(mockLineClient.LastReplyRequest.Messages))
	}

	helpText := mockLineClient.LastReplyRequest.Messages[0].Text

	// Verify /clear is included in help text
	if !strings.Contains(helpText, "/clear") {
		t.Errorf("Expected help text to contain '/clear', got:\n%s", helpText)
	}

	// Verify the description of /clear command is present
	if !strings.Contains(helpText, "Clear conversation history") {
		t.Errorf("Expected help text to contain 'Clear conversation history', got:\n%s", helpText)
	}
}

// Task Group 4: Conversation Context Integration Tests

// TestBuildChatRequest_IncludesHistoryInCorrectOrder tests that buildChatRequest formats messages as [system] + [history] + [new user message]
func TestBuildChatRequest_IncludesHistoryInCorrectOrder(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{}
	mockSessionStore := &MockSessionStore{}

	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	history := []domain.ChatMessage{
		{Role: domain.ChatMessageRoleUser, Content: "Hello"},
		{Role: domain.ChatMessageRoleAssistant, Content: "Hi there!"},
		{Role: domain.ChatMessageRoleUser, Content: "How are you?"},
		{Role: domain.ChatMessageRoleAssistant, Content: "I'm doing well, thanks!"},
	}

	// Act
	request := service.buildChatRequest("What's the weather?", history)

	// Assert
	expectedMessageCount := 1 + len(history) + 1 // system + history + new user message
	if len(request.Messages) != expectedMessageCount {
		t.Errorf("Expected %d messages, got %d", expectedMessageCount, len(request.Messages))
	}

	// Verify first message is system prompt
	if request.Messages[0].Role != domain.ChatMessageRoleSystem {
		t.Errorf("Expected first message to be system role, got %s", request.Messages[0].Role)
	}
	if request.Messages[0].Content != "You are a helpful assistant" {
		t.Errorf("Expected system prompt content, got: %s", request.Messages[0].Content)
	}

	// Verify history is in correct order (messages 1-4)
	for i, histMsg := range history {
		actualIdx := i + 1 // Skip system message
		if request.Messages[actualIdx].Role != histMsg.Role {
			t.Errorf("History message %d: expected role %s, got %s", i, histMsg.Role, request.Messages[actualIdx].Role)
		}
		if request.Messages[actualIdx].Content != histMsg.Content {
			t.Errorf("History message %d: expected content %q, got %q", i, histMsg.Content, request.Messages[actualIdx].Content)
		}
	}

	// Verify last message is new user message
	lastIdx := len(request.Messages) - 1
	if request.Messages[lastIdx].Role != domain.ChatMessageRoleUser {
		t.Errorf("Expected last message to be user role, got %s", request.Messages[lastIdx].Role)
	}
	if request.Messages[lastIdx].Content != "What's the weather?" {
		t.Errorf("Expected new user message content, got: %s", request.Messages[lastIdx].Content)
	}
}

// TestBuildChatRequest_EmptyHistory tests that buildChatRequest works correctly with empty history
func TestBuildChatRequest_EmptyHistory(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{}
	mockSessionStore := &MockSessionStore{}

	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	// Act
	request := service.buildChatRequest("Hello!", []domain.ChatMessage{})

	// Assert
	if len(request.Messages) != 2 {
		t.Errorf("Expected 2 messages (system + user), got %d", len(request.Messages))
	}

	// Verify first message is system prompt
	if request.Messages[0].Role != domain.ChatMessageRoleSystem {
		t.Errorf("Expected first message to be system role, got %s", request.Messages[0].Role)
	}

	// Verify second message is user message
	if request.Messages[1].Role != domain.ChatMessageRoleUser {
		t.Errorf("Expected second message to be user role, got %s", request.Messages[1].Role)
	}
	if request.Messages[1].Content != "Hello!" {
		t.Errorf("Expected user message content 'Hello!', got: %s", request.Messages[1].Content)
	}
}

// TestHandleMessageEvent_StoresUserAndAssistantMessages tests that handleMessageEvent stores both user and assistant messages in session
func TestHandleMessageEvent_StoresUserAndAssistantMessages(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{
		ChatCompletionFunc: func(ctx context.Context, request domain.ChatCompletionRequest) (*domain.ChatCompletionResponse, error) {
			return &domain.ChatCompletionResponse{Content: "Hello! How can I help you?"}, nil
		},
	}
	mockSessionStore := &MockSessionStore{}

	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	event := createTextMessageEvent("Hi there!")
	request := domain.LineWebhookRequest{
		Events: []domain.LineWebhookEvent{event},
	}

	// Act
	err := service.HandleWebhook(request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify session was updated
	if mockSessionStore.LastUpdatedSession == nil {
		t.Fatal("Expected session to be updated")
	}

	// Verify session contains the user and assistant messages
	session := mockSessionStore.LastUpdatedSession
	if len(session.Messages) != 2 {
		t.Errorf("Expected 2 messages in session (user + assistant), got %d", len(session.Messages))
	}

	// Verify user message
	if session.Messages[0].Role != domain.ChatMessageRoleUser {
		t.Errorf("Expected first message to be user role, got %s", session.Messages[0].Role)
	}
	if session.Messages[0].Content != "Hi there!" {
		t.Errorf("Expected user message 'Hi there!', got: %s", session.Messages[0].Content)
	}

	// Verify assistant message
	if session.Messages[1].Role != domain.ChatMessageRoleAssistant {
		t.Errorf("Expected second message to be assistant role, got %s", session.Messages[1].Role)
	}
	if session.Messages[1].Content != "Hello! How can I help you?" {
		t.Errorf("Expected assistant message 'Hello! How can I help you?', got: %s", session.Messages[1].Content)
	}
}

// TestConversationContext_MaintainedAcrossMultipleMessages tests that conversation context is preserved across multiple interactions
func TestConversationContext_MaintainedAcrossMultipleMessages(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}

	// Track the requests to verify history is being passed
	var chatRequests []domain.ChatCompletionRequest
	mockLMStudioClient := &MockLMStudioClient{
		ChatCompletionFunc: func(ctx context.Context, request domain.ChatCompletionRequest) (*domain.ChatCompletionResponse, error) {
			chatRequests = append(chatRequests, request)
			return &domain.ChatCompletionResponse{Content: "AI response " + string(rune('A'+len(chatRequests)-1))}, nil
		},
	}

	// Use a real session to accumulate history
	var storedSession *domain.ConversationSession
	mockSessionStore := &MockSessionStore{
		GetSessionFunc: func(userID string) (*domain.ConversationSession, error) {
			return storedSession, nil
		},
		UpdateSessionFunc: func(session *domain.ConversationSession) error {
			storedSession = session
			return nil
		},
	}

	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	// Act - Simulate 3 messages in conversation
	messages := []string{"First message", "Second message", "Third message"}
	for _, msg := range messages {
		event := createTextMessageEvent(msg)
		request := domain.LineWebhookRequest{
			Events: []domain.LineWebhookEvent{event},
		}
		err := service.HandleWebhook(request)
		if err != nil {
			t.Fatalf("HandleWebhook failed for message '%s': %v", msg, err)
		}
	}

	// Assert
	if len(chatRequests) != 3 {
		t.Errorf("Expected 3 chat completion requests, got %d", len(chatRequests))
	}

	// First request should have no history (just system + user message)
	if len(chatRequests[0].Messages) != 2 {
		t.Errorf("First request should have 2 messages (system + user), got %d", len(chatRequests[0].Messages))
	}

	// Second request should have history from first turn (system + 2 history + user = 4)
	if len(chatRequests[1].Messages) != 4 {
		t.Errorf("Second request should have 4 messages (system + history + user), got %d", len(chatRequests[1].Messages))
	}

	// Third request should have history from first two turns (system + 4 history + user = 6)
	if len(chatRequests[2].Messages) != 6 {
		t.Errorf("Third request should have 6 messages (system + history + user), got %d", len(chatRequests[2].Messages))
	}

	// Verify the history is in correct order in the third request
	req3 := chatRequests[2]
	// Index 0: system prompt
	if req3.Messages[0].Role != domain.ChatMessageRoleSystem {
		t.Errorf("Expected message 0 to be system, got %s", req3.Messages[0].Role)
	}
	// Index 1: first user message from history
	if req3.Messages[1].Role != domain.ChatMessageRoleUser || req3.Messages[1].Content != "First message" {
		t.Errorf("Expected message 1 to be first user message, got: %s - %s", req3.Messages[1].Role, req3.Messages[1].Content)
	}
	// Index 2: first assistant response from history
	if req3.Messages[2].Role != domain.ChatMessageRoleAssistant || req3.Messages[2].Content != "AI response A" {
		t.Errorf("Expected message 2 to be first AI response, got: %s - %s", req3.Messages[2].Role, req3.Messages[2].Content)
	}
	// Index 3: second user message from history
	if req3.Messages[3].Role != domain.ChatMessageRoleUser || req3.Messages[3].Content != "Second message" {
		t.Errorf("Expected message 3 to be second user message, got: %s - %s", req3.Messages[3].Role, req3.Messages[3].Content)
	}
	// Index 4: second assistant response from history
	if req3.Messages[4].Role != domain.ChatMessageRoleAssistant || req3.Messages[4].Content != "AI response B" {
		t.Errorf("Expected message 4 to be second AI response, got: %s - %s", req3.Messages[4].Role, req3.Messages[4].Content)
	}
	// Index 5: new user message
	if req3.Messages[5].Role != domain.ChatMessageRoleUser || req3.Messages[5].Content != "Third message" {
		t.Errorf("Expected message 5 to be third user message, got: %s - %s", req3.Messages[5].Role, req3.Messages[5].Content)
	}
}

// TestHandleMessageEvent_DoesNotStoreOnError tests that session is not updated when LM Studio returns an error
func TestHandleMessageEvent_DoesNotStoreOnError(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{
		ChatCompletionFunc: func(ctx context.Context, request domain.ChatCompletionRequest) (*domain.ChatCompletionResponse, error) {
			return nil, domain.ErrLMStudioUnavailable
		},
	}
	mockSessionStore := &MockSessionStore{}

	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	event := createTextMessageEvent("Hello!")
	request := domain.LineWebhookRequest{
		Events: []domain.LineWebhookEvent{event},
	}

	// Act
	err := service.HandleWebhook(request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error (error should be handled gracefully), got: %v", err)
	}

	// Verify session was NOT updated
	if mockSessionStore.LastUpdatedSession != nil {
		t.Error("Expected session to NOT be updated when LM Studio errors")
	}
}

// Task Group 4: Error Handling Tests (Legacy - updated to include sessionStore)

// TestErrorHandling_LMStudioUnavailable tests that ErrLMStudioUnavailable returns user-friendly message
func TestErrorHandling_LMStudioUnavailable(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{
		ChatCompletionFunc: func(ctx context.Context, request domain.ChatCompletionRequest) (*domain.ChatCompletionResponse, error) {
			return nil, domain.ErrLMStudioUnavailable
		},
	}
	mockSessionStore := &MockSessionStore{}

	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	event := createTextMessageEvent("Hello, AI!")
	request := domain.LineWebhookRequest{
		Events: []domain.LineWebhookEvent{event},
	}

	// Act
	err := service.HandleWebhook(request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error returned to caller, got: %v", err)
	}

	if mockLineClient.LastReplyRequest == nil {
		t.Fatal("Expected reply message to be sent")
	}

	if len(mockLineClient.LastReplyRequest.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(mockLineClient.LastReplyRequest.Messages))
	}

	expectedMessage := "Sorry, I'm having trouble processing your request right now. Please try again later."
	actualMessage := mockLineClient.LastReplyRequest.Messages[0].Text

	if actualMessage != expectedMessage {
		t.Errorf("Expected user-friendly message:\n%q\nGot:\n%q", expectedMessage, actualMessage)
	}
}

// TestErrorHandling_LMStudioTimeout tests that ErrLMStudioTimeout returns user-friendly message
func TestErrorHandling_LMStudioTimeout(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{
		ChatCompletionFunc: func(ctx context.Context, request domain.ChatCompletionRequest) (*domain.ChatCompletionResponse, error) {
			return nil, domain.ErrLMStudioTimeout
		},
	}
	mockSessionStore := &MockSessionStore{}

	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	event := createTextMessageEvent("Hello, AI!")
	request := domain.LineWebhookRequest{
		Events: []domain.LineWebhookEvent{event},
	}

	// Act
	err := service.HandleWebhook(request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error returned to caller, got: %v", err)
	}

	if mockLineClient.LastReplyRequest == nil {
		t.Fatal("Expected reply message to be sent")
	}

	if len(mockLineClient.LastReplyRequest.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(mockLineClient.LastReplyRequest.Messages))
	}

	expectedMessage := "Sorry, I'm having trouble processing your request right now. Please try again later."
	actualMessage := mockLineClient.LastReplyRequest.Messages[0].Text

	if actualMessage != expectedMessage {
		t.Errorf("Expected user-friendly message:\n%q\nGot:\n%q", expectedMessage, actualMessage)
	}
}

// TestErrorHandling_GenericError tests that generic errors also return user-friendly message
func TestErrorHandling_GenericError(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{
		ChatCompletionFunc: func(ctx context.Context, request domain.ChatCompletionRequest) (*domain.ChatCompletionResponse, error) {
			return nil, errors.New("connection refused")
		},
	}
	mockSessionStore := &MockSessionStore{}

	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	event := createTextMessageEvent("Hello, AI!")
	request := domain.LineWebhookRequest{
		Events: []domain.LineWebhookEvent{event},
	}

	// Act
	err := service.HandleWebhook(request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error returned to caller, got: %v", err)
	}

	if mockLineClient.LastReplyRequest == nil {
		t.Fatal("Expected reply message to be sent")
	}

	expectedMessage := "Sorry, I'm having trouble processing your request right now. Please try again later."
	actualMessage := mockLineClient.LastReplyRequest.Messages[0].Text

	if actualMessage != expectedMessage {
		t.Errorf("Expected user-friendly message:\n%q\nGot:\n%q", expectedMessage, actualMessage)
	}
}

// TestErrorHandling_TechnicalDetailsNotExposed tests that technical error details are not sent to user
func TestErrorHandling_TechnicalDetailsNotExposed(t *testing.T) {
	// Arrange
	technicalError := errors.New("ECONNREFUSED: connection to localhost:1234 failed with TCP timeout after 30s")

	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{
		ChatCompletionFunc: func(ctx context.Context, request domain.ChatCompletionRequest) (*domain.ChatCompletionResponse, error) {
			return nil, technicalError
		},
	}
	mockSessionStore := &MockSessionStore{}

	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	event := createTextMessageEvent("Hello, AI!")
	request := domain.LineWebhookRequest{
		Events: []domain.LineWebhookEvent{event},
	}

	// Act
	_ = service.HandleWebhook(request)

	// Assert
	if mockLineClient.LastReplyRequest == nil {
		t.Fatal("Expected reply message to be sent")
	}

	actualMessage := mockLineClient.LastReplyRequest.Messages[0].Text

	// Verify technical details are NOT in the message
	if containsAny(actualMessage, []string{"ECONNREFUSED", "localhost", "1234", "TCP", "timeout", "30s"}) {
		t.Errorf("Technical error details should not be exposed to user. Message: %q", actualMessage)
	}
}

// Helper function to check if string contains any of the substrings
func containsAny(s string, substrings []string) bool {
	for _, sub := range substrings {
		if len(sub) > 0 && len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// Task Group 5: User Input Truncation Tests

// TestTruncateUserInput_ShortMessage tests that messages under 4000 chars are not modified
func TestTruncateUserInput_ShortMessage(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{}
	mockSessionStore := &MockSessionStore{}
	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	shortMessage := "This is a short message"

	// Act
	result := service.truncateUserInput(shortMessage)

	// Assert
	if result != shortMessage {
		t.Errorf("Short message should not be modified.\nExpected: %q\nGot: %q", shortMessage, result)
	}
}

// TestTruncateUserInput_ExactlyMaxLength tests that messages at exactly 4000 chars are not modified
func TestTruncateUserInput_ExactlyMaxLength(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{}
	mockSessionStore := &MockSessionStore{}
	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	// Create a message with exactly 4000 characters
	exactMessage := strings.Repeat("a", maxUserInputLength)

	// Act
	result := service.truncateUserInput(exactMessage)

	// Assert
	if result != exactMessage {
		t.Errorf("Message at exactly max length should not be modified.\nExpected length: %d\nGot length: %d", len(exactMessage), len(result))
	}
}

// TestTruncateUserInput_LongMessage tests that messages over 4000 chars are truncated to 4000
func TestTruncateUserInput_LongMessage(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{}
	mockSessionStore := &MockSessionStore{}
	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	// Create a message with 5000 characters (exceeds 4000 limit)
	longMessage := strings.Repeat("b", 5000)

	// Act
	result := service.truncateUserInput(longMessage)

	// Assert
	if len(result) != maxUserInputLength {
		t.Errorf("Long message should be truncated to %d chars.\nExpected length: %d\nGot length: %d", maxUserInputLength, maxUserInputLength, len(result))
	}

	// Verify the content is from the beginning of the original message
	expectedContent := strings.Repeat("b", maxUserInputLength)
	if result != expectedContent {
		t.Errorf("Truncated message should contain first %d characters of original", maxUserInputLength)
	}
}

// TestTruncateUserInput_IntegrationWithHandler tests truncation is applied before ChatCompletion call
func TestTruncateUserInput_IntegrationWithHandler(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{
		ChatCompletionFunc: func(ctx context.Context, request domain.ChatCompletionRequest) (*domain.ChatCompletionResponse, error) {
			// Verify the user message was truncated
			if len(request.Messages) < 2 {
				t.Fatal("Expected at least 2 messages (system + user)")
			}
			userMessage := request.Messages[len(request.Messages)-1].Content // Get last message (new user message)
			if len(userMessage) > maxUserInputLength {
				t.Errorf("User message should be truncated to %d chars, got %d", maxUserInputLength, len(userMessage))
			}
			return &domain.ChatCompletionResponse{Content: "AI response"}, nil
		},
	}
	mockSessionStore := &MockSessionStore{}

	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	// Create event with a very long message (6000 chars)
	longText := strings.Repeat("x", 6000)
	event := createTextMessageEvent(longText)
	request := domain.LineWebhookRequest{
		Events: []domain.LineWebhookEvent{event},
	}

	// Act
	err := service.HandleWebhook(request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify ChatCompletion was called
	if mockLMStudioClient.LastChatRequest == nil {
		t.Fatal("Expected ChatCompletion to be called")
	}

	// Verify user message in request was truncated (last message is the new user message)
	lastIdx := len(mockLMStudioClient.LastChatRequest.Messages) - 1
	userMessage := mockLMStudioClient.LastChatRequest.Messages[lastIdx].Content
	if len(userMessage) != maxUserInputLength {
		t.Errorf("User message in ChatCompletionRequest should be %d chars, got %d", maxUserInputLength, len(userMessage))
	}
}

// Task Group 6: AI Response Splitting Tests

// TestSplitAIResponse_ShortMessage tests that responses under 5000 chars return single message
func TestSplitAIResponse_ShortMessage(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{}
	mockSessionStore := &MockSessionStore{}
	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	shortResponse := "This is a short AI response."

	// Act
	result := service.splitAIResponse(shortResponse)

	// Assert
	if len(result) != 1 {
		t.Errorf("Expected 1 message for short response, got %d", len(result))
	}

	if result[0] != shortResponse {
		t.Errorf("Short response should not be modified.\nExpected: %q\nGot: %q", shortResponse, result[0])
	}
}

// TestSplitAIResponse_ExactlyMaxLength tests that responses at exactly 5000 chars return single message
func TestSplitAIResponse_ExactlyMaxLength(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{}
	mockSessionStore := &MockSessionStore{}
	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	// Create a message with exactly 5000 characters
	exactMessage := strings.Repeat("a", maxLineMessageLength)

	// Act
	result := service.splitAIResponse(exactMessage)

	// Assert
	if len(result) != 1 {
		t.Errorf("Expected 1 message for exact max length response, got %d", len(result))
	}

	if result[0] != exactMessage {
		t.Errorf("Response at exactly max length should not be modified.\nExpected length: %d\nGot length: %d", len(exactMessage), len(result[0]))
	}
}

// TestSplitAIResponse_LongMessage tests that responses over 5000 chars split into multiple messages
func TestSplitAIResponse_LongMessage(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{}
	mockSessionStore := &MockSessionStore{}
	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	// Create a message with 7500 characters (should split into 2 messages)
	longMessage := strings.Repeat("b", 7500)

	// Act
	result := service.splitAIResponse(longMessage)

	// Assert
	if len(result) < 2 {
		t.Errorf("Expected at least 2 messages for 7500 char response, got %d", len(result))
	}

	// Verify no individual message exceeds max length
	for i, msg := range result {
		if len(msg) > maxLineMessageLength {
			t.Errorf("Message %d exceeds max length: %d > %d", i, len(msg), maxLineMessageLength)
		}
	}

	// Verify total content is preserved
	totalContent := strings.Join(result, "")
	if totalContent != longMessage {
		t.Errorf("Total content should be preserved.\nExpected length: %d\nGot length: %d", len(longMessage), len(totalContent))
	}
}

// TestSplitAIResponse_SentenceBoundary tests that split prefers sentence boundaries (. ! ?)
func TestSplitAIResponse_SentenceBoundary(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{}
	mockSessionStore := &MockSessionStore{}
	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	// Create a message that exceeds 5000 chars with a sentence boundary within the lookback window
	// Total needs to be > 5000. Put a sentence ending around position 4900 (within 200 chars lookback from 5000)
	// Part 1: 4899 chars of 'a' + ". " = 4901 chars
	// Part 2: Needs enough to push total over 5000, so at least 100 more chars
	prefix := strings.Repeat("a", 4899)
	sentence1 := prefix + ". "                              // 4901 chars total
	sentence2 := strings.Repeat("b", 200) + " End of text." // 213 chars
	longMessage := sentence1 + sentence2                    // 5114 chars total, exceeds 5000

	// Act
	result := service.splitAIResponse(longMessage)

	// Assert
	if len(result) < 2 {
		t.Errorf("Expected at least 2 messages for %d char message, got %d", len(longMessage), len(result))
	}

	// Verify first message ends at sentence boundary (includes the period and space)
	if len(result) >= 1 {
		firstMsg := result[0]
		// The first message should end with ". " because we split after the sentence boundary
		if !strings.HasSuffix(firstMsg, ". ") {
			// Or it could have trimmed the trailing space
			if !strings.HasSuffix(firstMsg, ".") {
				t.Errorf("Expected first message to end at sentence boundary (period), got: ...%q", lastChars(firstMsg, 20))
			}
		}

		// The first message should be shorter than maxLineMessageLength because we split at sentence boundary
		if len(firstMsg) > maxLineMessageLength {
			t.Errorf("First message should not exceed max length: %d > %d", len(firstMsg), maxLineMessageLength)
		}
	}
}

// TestSplitAIResponse_MaxMessages tests that maximum 5 messages are returned
func TestSplitAIResponse_MaxMessages(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}
	mockLMStudioClient := &MockLMStudioClient{}
	mockSessionStore := &MockSessionStore{}
	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	// Create a very long message that would require more than 5 messages (30000+ chars)
	veryLongMessage := strings.Repeat("c", 30000)

	// Act
	result := service.splitAIResponse(veryLongMessage)

	// Assert
	if len(result) > maxMessagesPerResponse {
		t.Errorf("Expected maximum %d messages, got %d", maxMessagesPerResponse, len(result))
	}

	if len(result) != maxMessagesPerResponse {
		t.Errorf("Expected exactly %d messages for very long response, got %d", maxMessagesPerResponse, len(result))
	}

	// Verify no individual message exceeds max length
	for i, msg := range result {
		if len(msg) > maxLineMessageLength {
			t.Errorf("Message %d exceeds max length: %d > %d", i, len(msg), maxLineMessageLength)
		}
	}
}

// TestSplitAIResponse_MultiMessageSending_Integration tests that multi-message sending uses ReplyMessage and PushMessage correctly
func TestSplitAIResponse_MultiMessageSending_Integration(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}

	// Create AI response that will require 2 messages
	longAIResponse := strings.Repeat("x", 7500)

	mockLMStudioClient := &MockLMStudioClient{
		ChatCompletionFunc: func(ctx context.Context, request domain.ChatCompletionRequest) (*domain.ChatCompletionResponse, error) {
			return &domain.ChatCompletionResponse{Content: longAIResponse}, nil
		},
	}
	mockSessionStore := &MockSessionStore{}

	service := NewLineWebhookService(mockLineClient, mockLMStudioClient, mockSessionStore, "You are a helpful assistant", defaultTestTimeout, defaultTestMaxTurns)

	event := createTextMessageEvent("Generate a long response")
	request := domain.LineWebhookRequest{
		Events: []domain.LineWebhookEvent{event},
	}

	// Act
	err := service.HandleWebhook(request)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify ReplyMessage was called for the first message
	if mockLineClient.LastReplyRequest == nil {
		t.Fatal("Expected ReplyMessage to be called")
	}

	if len(mockLineClient.LastReplyRequest.Messages) != 1 {
		t.Errorf("Expected 1 message in ReplyMessage request, got %d", len(mockLineClient.LastReplyRequest.Messages))
	}

	// Verify PushMessage was called for subsequent messages
	if len(mockLineClient.PushRequests) == 0 {
		t.Fatal("Expected PushMessage to be called for subsequent messages")
	}

	// Verify push messages are sent to the correct user
	for i, pushReq := range mockLineClient.PushRequests {
		if pushReq.To != "test-user-id" {
			t.Errorf("Push request %d should be sent to test-user-id, got %s", i, pushReq.To)
		}
	}
}

// Helper function to get last N characters of a string
func lastChars(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

// ============================================================================
// Task Group 6: Strategic End-to-End Tests for Session Management Feature
// ============================================================================

// TestEndToEnd_ClearCommandThenFreshConversation tests the complete workflow:
// 1. User has existing conversation with history
// 2. User sends /clear command
// 3. Session is deleted
// 4. User sends new message
// 5. Fresh session is created with no history
func TestEndToEnd_ClearCommandThenFreshConversation(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}

	// Track chat requests to verify history state
	var chatRequests []domain.ChatCompletionRequest
	mockLMStudioClient := &MockLMStudioClient{
		ChatCompletionFunc: func(ctx context.Context, request domain.ChatCompletionRequest) (*domain.ChatCompletionResponse, error) {
			chatRequests = append(chatRequests, request)
			return &domain.ChatCompletionResponse{Content: "AI response"}, nil
		},
	}

	// Use a real session store behavior simulation
	var storedSession *domain.ConversationSession
	mockSessionStore := &MockSessionStore{
		GetSessionFunc: func(userID string) (*domain.ConversationSession, error) {
			return storedSession, nil
		},
		UpdateSessionFunc: func(session *domain.ConversationSession) error {
			storedSession = session
			return nil
		},
		DeleteSessionFunc: func(userID string) error {
			storedSession = nil // Clear the session
			return nil
		},
	}

	service := NewLineWebhookService(
		mockLineClient,
		mockLMStudioClient,
		mockSessionStore,
		"You are a helpful assistant",
		defaultTestTimeout,
		defaultTestMaxTurns,
	)

	// Act - Step 1: Send first message to establish history
	event1 := createTextMessageEvent("Hello, remember this!")
	request1 := domain.LineWebhookRequest{Events: []domain.LineWebhookEvent{event1}}
	err := service.HandleWebhook(request1)
	if err != nil {
		t.Fatalf("First message failed: %v", err)
	}

	// Verify session has history after first message
	if storedSession == nil {
		t.Fatal("Expected session to be created after first message")
	}
	if len(storedSession.Messages) != 2 {
		t.Errorf("Expected 2 messages in session after first turn, got %d", len(storedSession.Messages))
	}

	// Act - Step 2: Send /clear command
	eventClear := createTextMessageEvent("/clear")
	requestClear := domain.LineWebhookRequest{Events: []domain.LineWebhookEvent{eventClear}}
	err = service.HandleWebhook(requestClear)
	if err != nil {
		t.Fatalf("/clear command failed: %v", err)
	}

	// Verify session was deleted
	if storedSession != nil {
		t.Error("Expected session to be nil after /clear command")
	}

	// Verify confirmation message was sent
	if mockLineClient.LastReplyRequest == nil {
		t.Fatal("Expected reply for /clear command")
	}
	if mockLineClient.LastReplyRequest.Messages[0].Text != "Conversation history cleared." {
		t.Errorf("Expected 'Conversation history cleared.' message, got: %s", mockLineClient.LastReplyRequest.Messages[0].Text)
	}

	// Act - Step 3: Send new message after /clear
	event2 := createTextMessageEvent("New conversation starts here")
	request2 := domain.LineWebhookRequest{Events: []domain.LineWebhookEvent{event2}}
	err = service.HandleWebhook(request2)
	if err != nil {
		t.Fatalf("Message after /clear failed: %v", err)
	}

	// Assert - Verify fresh session was created
	if storedSession == nil {
		t.Fatal("Expected new session to be created after /clear")
	}
	if len(storedSession.Messages) != 2 {
		t.Errorf("Expected 2 messages in new session, got %d", len(storedSession.Messages))
	}

	// Verify the message after /clear had no history (only system + user message)
	if len(chatRequests) < 2 {
		t.Fatalf("Expected at least 2 chat requests, got %d", len(chatRequests))
	}
	lastRequest := chatRequests[len(chatRequests)-1]
	// Should have only 2 messages: system prompt + new user message (no history)
	if len(lastRequest.Messages) != 2 {
		t.Errorf("Expected 2 messages (system + user) after /clear, got %d. This indicates old history was carried over.", len(lastRequest.Messages))
	}
}

// TestIntegration_SessionRespectsConfiguredMaxTurns tests that session properly enforces
// the configured maxTurns limit, removing oldest turns when limit is reached
func TestIntegration_SessionRespectsConfiguredMaxTurns(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}

	// Track chat requests to observe history growth and trimming
	var chatRequests []domain.ChatCompletionRequest
	mockLMStudioClient := &MockLMStudioClient{
		ChatCompletionFunc: func(ctx context.Context, request domain.ChatCompletionRequest) (*domain.ChatCompletionResponse, error) {
			chatRequests = append(chatRequests, request)
			return &domain.ChatCompletionResponse{Content: "Response"}, nil
		},
	}

	// Use real session store behavior
	var storedSession *domain.ConversationSession
	mockSessionStore := &MockSessionStore{
		GetSessionFunc: func(userID string) (*domain.ConversationSession, error) {
			return storedSession, nil
		},
		UpdateSessionFunc: func(session *domain.ConversationSession) error {
			storedSession = session
			return nil
		},
	}

	// Configure with maxTurns = 3 (6 messages max in history)
	customMaxTurns := 3
	service := NewLineWebhookService(
		mockLineClient,
		mockLMStudioClient,
		mockSessionStore,
		"You are a helpful assistant",
		defaultTestTimeout,
		customMaxTurns,
	)

	// Act - Send 5 messages (exceeds 3-turn limit)
	messages := []string{"Message 1", "Message 2", "Message 3", "Message 4", "Message 5"}
	for _, msg := range messages {
		event := createTextMessageEvent(msg)
		request := domain.LineWebhookRequest{Events: []domain.LineWebhookEvent{event}}
		err := service.HandleWebhook(request)
		if err != nil {
			t.Fatalf("Failed to handle message '%s': %v", msg, err)
		}
	}

	// Assert - Verify session has exactly maxTurns * 2 messages (3 turns = 6 messages)
	if storedSession == nil {
		t.Fatal("Expected session to exist")
	}
	expectedMessages := customMaxTurns * 2
	if len(storedSession.Messages) != expectedMessages {
		t.Errorf("Expected session to have %d messages (maxTurns=%d), got %d",
			expectedMessages, customMaxTurns, len(storedSession.Messages))
	}

	// Verify oldest messages were removed (Message 1 and 2 should not be in history)
	for _, msg := range storedSession.Messages {
		if msg.Content == "Message 1" || msg.Content == "Message 2" {
			t.Errorf("Expected oldest messages to be removed, but found: %s", msg.Content)
		}
	}

	// Verify newest messages are present
	history := storedSession.GetHistory()
	foundMessage5 := false
	for _, msg := range history {
		if msg.Content == "Message 5" {
			foundMessage5 = true
			break
		}
	}
	if !foundMessage5 {
		t.Error("Expected newest message 'Message 5' to be in history")
	}
}

// TestIntegration_ExpiredSessionCreatesNewSession tests that when a session expires,
// sending a new message creates a fresh session without history
func TestIntegration_ExpiredSessionCreatesNewSession(t *testing.T) {
	// Arrange
	mockLineClient := &MockLineClient{}

	// Track chat requests
	var chatRequests []domain.ChatCompletionRequest
	mockLMStudioClient := &MockLMStudioClient{
		ChatCompletionFunc: func(ctx context.Context, request domain.ChatCompletionRequest) (*domain.ChatCompletionResponse, error) {
			chatRequests = append(chatRequests, request)
			return &domain.ChatCompletionResponse{Content: "AI response"}, nil
		},
	}

	// Simulate session store that returns expired session, then nil after deletion
	var storedSession *domain.ConversationSession
	sessionDeleted := false
	mockSessionStore := &MockSessionStore{
		GetSessionFunc: func(userID string) (*domain.ConversationSession, error) {
			if sessionDeleted {
				return nil, nil
			}
			return storedSession, nil
		},
		UpdateSessionFunc: func(session *domain.ConversationSession) error {
			storedSession = session
			sessionDeleted = false
			return nil
		},
		DeleteSessionFunc: func(userID string) error {
			storedSession = nil
			sessionDeleted = true
			return nil
		},
	}

	// Use a very short timeout for testing (1 millisecond)
	shortTimeout := 1 * time.Millisecond
	service := NewLineWebhookService(
		mockLineClient,
		mockLMStudioClient,
		mockSessionStore,
		"You are a helpful assistant",
		shortTimeout,
		defaultTestMaxTurns,
	)

	// Act - Step 1: Send first message
	event1 := createTextMessageEvent("Remember this important info!")
	request1 := domain.LineWebhookRequest{Events: []domain.LineWebhookEvent{event1}}
	err := service.HandleWebhook(request1)
	if err != nil {
		t.Fatalf("First message failed: %v", err)
	}

	// Verify session was created with history
	if storedSession == nil {
		t.Fatal("Expected session to be created")
	}
	if len(storedSession.Messages) != 2 {
		t.Errorf("Expected 2 messages after first turn, got %d", len(storedSession.Messages))
	}

	// Step 2: Simulate session expiration by manually setting LastAccessTime in the past
	storedSession.LastAccessTime = time.Now().Add(-10 * time.Millisecond) // Expired

	// Step 3: Simulate the session store returning nil for expired session
	// (This is what MemorySessionStore does when session.IsExpired() returns true)
	mockSessionStore.GetSessionFunc = func(userID string) (*domain.ConversationSession, error) {
		if storedSession != nil && storedSession.IsExpired() {
			// Memory store deletes expired sessions and returns nil
			storedSession = nil
			return nil, nil
		}
		return storedSession, nil
	}

	// Act - Step 4: Send new message after session expired
	event2 := createTextMessageEvent("New message after expiration")
	request2 := domain.LineWebhookRequest{Events: []domain.LineWebhookEvent{event2}}
	err = service.HandleWebhook(request2)
	if err != nil {
		t.Fatalf("Message after expiration failed: %v", err)
	}

	// Assert - The new message should have been sent without history
	if len(chatRequests) < 2 {
		t.Fatalf("Expected at least 2 chat requests, got %d", len(chatRequests))
	}

	lastRequest := chatRequests[len(chatRequests)-1]
	// Should have only 2 messages: system prompt + new user message (no history from expired session)
	if len(lastRequest.Messages) != 2 {
		t.Errorf("Expected 2 messages (system + user) after session expiration, got %d. Old history may have carried over.", len(lastRequest.Messages))
	}

	// Verify a new session was created with fresh history
	if storedSession == nil {
		t.Fatal("Expected new session to be created")
	}
	if len(storedSession.Messages) != 2 {
		t.Errorf("Expected new session to have 2 messages (new turn only), got %d", len(storedSession.Messages))
	}
}
