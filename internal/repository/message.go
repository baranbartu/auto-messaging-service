package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"automessaging/internal/model"
)

// MessageRepository defines the database operations required for messages.
type MessageRepository interface {
	FetchNextUnsent(ctx context.Context, limit int) ([]model.Message, error)
	MarkAsSent(ctx context.Context, id uuid.UUID, sentAt time.Time) error
	ListSent(ctx context.Context, offset, limit int) ([]model.Message, int, error)
}
