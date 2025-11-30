package handler

import (
	"context"
	"net/http"
	"strconv"

	"automessaging/internal/service"
)

// MessageHandler provides HTTP endpoints for messages.
type MessageHandler struct {
	svc interface {
		ListSentMessages(ctx context.Context, page, limit int) (service.SentMessagesResult, error)
	}
}

// NewMessageHandler builds a MessageHandler.
func NewMessageHandler(svc interface {
	ListSentMessages(ctx context.Context, page, limit int) (service.SentMessagesResult, error)
}) *MessageHandler {
	return &MessageHandler{svc: svc}
}

// ListSent handles GET /messages/sent.
func (h *MessageHandler) ListSent(w http.ResponseWriter, r *http.Request) {
	page := parseIntDefault(r.URL.Query().Get("page"), 1)
	limit := parseIntDefault(r.URL.Query().Get("limit"), 20)

	result, err := h.svc.ListSentMessages(r.Context(), page, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func parseIntDefault(value string, def int) int {
	if value == "" {
		return def
	}
	if v, err := strconv.Atoi(value); err == nil {
		return v
	}
	return def
}
