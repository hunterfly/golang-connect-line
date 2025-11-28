package input

import "golang-template/internal/domain"

// LineWebhookService interface - Input port (use case)
// Defines what the application can do with LINE webhook events
type LineWebhookService interface {
	// HandleWebhook processes incoming webhook events from LINE
	HandleWebhook(request domain.LineWebhookRequest) error
}
