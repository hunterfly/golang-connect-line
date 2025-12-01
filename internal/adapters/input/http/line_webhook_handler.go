package http

import (
	"bytes"
	"net/http"

	"golang-template/internal/domain"
	"golang-template/internal/ports/input"

	"github.com/gofiber/fiber/v2"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
	"github.com/sirupsen/logrus"
)

// LineWebhookHandler struct - Primary/Driving adapter for LINE webhook
type LineWebhookHandler struct {
	service       input.LineWebhookService
	channelSecret string
}

// NewLineWebhookHandler func - Creates new LINE webhook handler
func NewLineWebhookHandler(service input.LineWebhookService, channelSecret string) *LineWebhookHandler {
	return &LineWebhookHandler{
		service:       service,
		channelSecret: channelSecret,
	}
}

// HandleWebhook func - Handles incoming LINE webhook requests
// @Summary LINE Webhook
// @Description Handles webhook events from LINE Messaging API
// @Tags LINE
// @Accept application/json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /webhook/line [post]
func (h *LineWebhookHandler) HandleWebhook(c *fiber.Ctx) error {
	// Convert Fiber request to http.Request for LINE SDK
	body := c.Body()
	httpReq, err := http.NewRequest("POST", "/webhook/line", bytes.NewReader(body))
	if err != nil {
		logrus.Errorf("Failed to create http request: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Internal error",
		})
	}

	// Copy headers
	c.Request().Header.VisitAll(func(key, value []byte) {
		httpReq.Header.Set(string(key), string(value))
	})

	// Parse and validate webhook request
	cb, err := webhook.ParseRequest(h.channelSecret, httpReq)
	// logrus.Printf("HTTP Request: %+v\n", httpReq)
	// logrus.Printf("channelSecret: %v",h.channelSecret)
	if err != nil {
		logrus.Errorf("Failed to parse webhook request: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid signature or request",
		})
	}

	// Convert LINE SDK events to domain events
	domainEvents := make([]domain.LineWebhookEvent, 0, len(cb.Events))

	for _, event := range cb.Events {
		domainEvent := h.convertToDomainEvent(event)
		if domainEvent != nil {
			domainEvents = append(domainEvents, *domainEvent)
		}
	}

	// Create domain webhook request
	webhookReq := domain.LineWebhookRequest{
		Events: domainEvents,
	}

	// Call application service
	if err := h.service.HandleWebhook(webhookReq); err != nil {
		logrus.Errorf("Failed to handle webhook: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to process webhook",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
	})
}

// convertToDomainEvent - Converts LINE SDK event to domain event
func (h *LineWebhookHandler) convertToDomainEvent(event webhook.EventInterface) *domain.LineWebhookEvent {
	switch e := event.(type) {
	case webhook.MessageEvent:
		return h.convertMessageEvent(e)
	case webhook.FollowEvent:
		return h.convertFollowEvent(e)
	case webhook.UnfollowEvent:
		return h.convertUnfollowEvent(e)
	default:
		logrus.Warnf("Unsupported event type: %T", event)
		return nil
	}
}

// convertMessageEvent - Converts message event
func (h *LineWebhookHandler) convertMessageEvent(event webhook.MessageEvent) *domain.LineWebhookEvent {
	domainEvent := &domain.LineWebhookEvent{
		Type:       domain.LineEventTypeMessage,
		ReplyToken: event.ReplyToken,
		Source:     h.convertSource(event.Source),
	}

	// Convert message based on type
	switch msg := event.Message.(type) {
	case webhook.TextMessageContent:
		domainEvent.Message = &domain.LineMessage{
			ID:   msg.Id,
			Type: domain.LineMessageTypeText,
			Text: msg.Text,
		}
	case webhook.StickerMessageContent:
		domainEvent.Message = &domain.LineMessage{
			ID:        msg.Id,
			Type:      domain.LineMessageTypeSticker,
			PackageID: msg.PackageId,
			StickerID: msg.StickerId,
		}
	case webhook.ImageMessageContent:
		domainEvent.Message = &domain.LineMessage{
			ID:   msg.Id,
			Type: domain.LineMessageTypeImage,
		}
	default:
		logrus.Warnf("Unsupported message type: %T", msg)
		return nil
	}

	return domainEvent
}

// convertFollowEvent - Converts follow event
func (h *LineWebhookHandler) convertFollowEvent(event webhook.FollowEvent) *domain.LineWebhookEvent {
	return &domain.LineWebhookEvent{
		Type:       domain.LineEventTypeFollow,
		ReplyToken: event.ReplyToken,
		Source:     h.convertSource(event.Source),
	}
}

// convertUnfollowEvent - Converts unfollow event
func (h *LineWebhookHandler) convertUnfollowEvent(event webhook.UnfollowEvent) *domain.LineWebhookEvent {
	return &domain.LineWebhookEvent{
		Type:   domain.LineEventTypeUnfollow,
		Source: h.convertSource(event.Source),
	}
}

// convertSource - Converts event source
func (h *LineWebhookHandler) convertSource(source webhook.SourceInterface) domain.LineSource {
	switch s := source.(type) {
	case webhook.UserSource:
		return domain.LineSource{
			Type:   domain.LineSourceTypeUser,
			UserID: s.UserId,
		}
	case webhook.GroupSource:
		return domain.LineSource{
			Type:    domain.LineSourceTypeGroup,
			UserID:  s.UserId,
			GroupID: s.GroupId,
		}
	case webhook.RoomSource:
		return domain.LineSource{
			Type:   domain.LineSourceTypeRoom,
			UserID: s.UserId,
			RoomID: s.RoomId,
		}
	default:
		return domain.LineSource{}
	}
}
