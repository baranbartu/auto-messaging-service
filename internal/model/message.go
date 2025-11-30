package model

import (
	"time"

	"github.com/google/uuid"
)

// Message represents the data stored in PostgreSQL about messages to be sent.
type Message struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	To        string     `db:"to" json:"to"`
	Content   string     `db:"content" json:"content"`
	Sent      bool       `db:"sent" json:"sent"`
	SentAt    *time.Time `db:"sent_at" json:"sent_at,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}
