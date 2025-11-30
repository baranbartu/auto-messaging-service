package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"automessaging/internal/model"
	"automessaging/internal/repository"
)

var _ repository.MessageRepository = (*MessageRepository)(nil)

// MessageRepository provides PostgreSQL backed message operations.
type MessageRepository struct {
	db *sql.DB
}

// NewMessageRepository creates a new repository instance.
func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// FetchNextUnsent retrieves the earliest unsent messages.
func (r *MessageRepository) FetchNextUnsent(ctx context.Context, limit int) ([]model.Message, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT id, "to", content, sent, sent_at, created_at
        FROM messages
        WHERE sent = false
        ORDER BY created_at ASC
        LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []model.Message
	for rows.Next() {
		var msg model.Message
		var sentAt sql.NullTime
		if err := rows.Scan(&msg.ID, &msg.To, &msg.Content, &msg.Sent, &sentAt, &msg.CreatedAt); err != nil {
			return nil, err
		}
		if sentAt.Valid {
			ts := sentAt.Time
			msg.SentAt = &ts
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// MarkAsSent updates a message row with sent details.
func (r *MessageRepository) MarkAsSent(ctx context.Context, id uuid.UUID, sentAt time.Time) error {
	res, err := r.db.ExecContext(ctx, `
        UPDATE messages
        SET sent = true, sent_at = $2
        WHERE id = $1`, id, sentAt)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ListSent lists sent messages with pagination and counts total.
func (r *MessageRepository) ListSent(ctx context.Context, offset, limit int) ([]model.Message, int, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT id, "to", content, sent, sent_at, created_at
        FROM messages
        WHERE sent = true
        ORDER BY sent_at DESC NULLS LAST, created_at DESC
        OFFSET $1 LIMIT $2`, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var messages []model.Message
	for rows.Next() {
		var msg model.Message
		var sentAt sql.NullTime
		if err := rows.Scan(&msg.ID, &msg.To, &msg.Content, &msg.Sent, &sentAt, &msg.CreatedAt); err != nil {
			return nil, 0, err
		}
		if sentAt.Valid {
			ts := sentAt.Time
			msg.SentAt = &ts
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM messages WHERE sent = true`).Scan(&total); err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}
