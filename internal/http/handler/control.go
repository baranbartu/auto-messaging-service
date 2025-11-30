package handler

import (
	"context"
	"net/http"

	"automessaging/internal/scheduler"
)

// SchedulerController abstracts scheduler operations for handlers.
type SchedulerController interface {
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool
}

// ControlHandler handles scheduler start/stop endpoints.
type ControlHandler struct {
	scheduler SchedulerController
}

// NewControlHandler creates a new instance.
func NewControlHandler(s SchedulerController) *ControlHandler {
	return &ControlHandler{scheduler: s}
}

// Start triggers the scheduler loop.
func (h *ControlHandler) Start(w http.ResponseWriter, r *http.Request) {
	if err := h.scheduler.Start(context.Background()); err != nil {
		status := http.StatusInternalServerError
		if err == scheduler.ErrAlreadyRunning {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

// Stop halts the scheduler loop.
func (h *ControlHandler) Stop(w http.ResponseWriter, r *http.Request) {
	if err := h.scheduler.Stop(); err != nil {
		status := http.StatusInternalServerError
		if err == scheduler.ErrNotRunning {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}
