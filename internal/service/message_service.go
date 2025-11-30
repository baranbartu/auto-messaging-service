package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"automessaging/internal/model"
	"automessaging/internal/repository"
)

// MessageService orchestrates message processing.
type MessageService struct {
	deps           dependencies
	client         *http.Client
	webhookURL     string
	webhookAuthKey string
	fetchLimit     int
	logger         *log.Logger
}

type dependencies struct {
	repo  repository.MessageRepository
	redis redis.Cmdable
}

// MessageServiceOptions configures MessageService.
type MessageServiceOptions struct {
	FetchLimit     int
	WebhookURL     string
	WebhookAuthKey string
	HTTPTimeout    time.Duration
	Logger         *log.Logger
}

// SentMessagesResult captures paginated sent messages.
type SentMessagesResult struct {
	Messages []model.Message `json:"messages"`
	Total    int             `json:"total"`
	Page     int             `json:"page"`
	Limit    int             `json:"limit"`
}

// Dependencies groups constructor requirements for MessageService.
type Dependencies struct {
	Repo  repository.MessageRepository
	Redis redis.Cmdable
}

// NewMessageService builds a MessageService.
func NewMessageService(deps Dependencies, opts MessageServiceOptions) *MessageService {
	timeout := opts.HTTPTimeout
	if timeout == 0 {
		timeout = 15 * time.Second
	}

	fetchLimit := opts.FetchLimit
	if fetchLimit <= 0 {
		fetchLimit = 2
	}

	logger := opts.Logger
	if logger == nil {
		logger = log.New(os.Stdout, "message-service ", log.LstdFlags)
	}

	return &MessageService{
		deps: dependencies{
			repo:  deps.Repo,
			redis: deps.Redis,
		},
		client:         &http.Client{Timeout: timeout},
		webhookURL:     opts.WebhookURL,
		webhookAuthKey: opts.WebhookAuthKey,
		fetchLimit:     fetchLimit,
		logger:         logger,
	}
}

// ProcessPendingMessages fetches unsent messages and sends them to the webhook.
func (s *MessageService) ProcessPendingMessages(ctx context.Context) error {
	if s.webhookURL == "" {
		return errors.New("webhook URL is not configured")
	}

	messages, err := s.deps.repo.FetchNextUnsent(ctx, s.fetchLimit)
	if err != nil {
		return err
	}

	if len(messages) == 0 {
		return nil
	}

	for _, msg := range messages {
		if err := s.sendMessage(ctx, msg); err != nil {
			s.logger.Printf("failed to send message %s: %v", msg.ID, err)
		}
	}

	return nil
}

// ListSentMessages returns paginated sent messages.
func (s *MessageService) ListSentMessages(ctx context.Context, page, limit int) (SentMessagesResult, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit
	items, total, err := s.deps.repo.ListSent(ctx, offset, limit)
	if err != nil {
		return SentMessagesResult{}, err
	}

	return SentMessagesResult{
		Messages: items,
		Total:    total,
		Page:     page,
		Limit:    limit,
	}, nil
}

func (s *MessageService) sendMessage(ctx context.Context, msg model.Message) error {
	payload := map[string]string{
		"to":      msg.To,
		"content": msg.Content,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.webhookAuthKey != "" {
		req.Header.Set("x-ins-auth-key", s.webhookAuthKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	var webhookResp webhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&webhookResp); err != nil {
		return fmt.Errorf("decode webhook response: %w", err)
	}

	if webhookResp.Message != "Accepted" || webhookResp.MessageID == "" {
		return fmt.Errorf("webhook rejected message %s", msg.ID)
	}

	sentAt := time.Now().UTC()
	if err := s.deps.repo.MarkAsSent(ctx, msg.ID, sentAt); err != nil {
		return err
	}

	if err := s.storeSentMetadata(ctx, msg.ID, webhookResp.MessageID, sentAt); err != nil {
		s.logger.Printf("failed to store metadata in redis for %s: %v", msg.ID, err)
	}

	return nil
}

func (s *MessageService) storeSentMetadata(ctx context.Context, messageID uuid.UUID, remoteID string, sentAt time.Time) error {
	key := fmt.Sprintf("sent_message:%s", remoteID)
	values := map[string]interface{}{
		"message_id": remoteID,
		"local_id":   messageID.String(),
		"sent_at":    sentAt.Format(time.RFC3339Nano),
	}

	if err := s.deps.redis.HSet(ctx, key, values).Err(); err != nil {
		return err
	}
	return nil
}

type webhookResponse struct {
	Message   string `json:"message"`
	MessageID string `json:"messageId"`
}
