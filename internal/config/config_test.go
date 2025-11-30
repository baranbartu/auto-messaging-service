package config

import (
	"testing"
	"time"
)

func TestSchedulerIntervalClamp(t *testing.T) {
	t.Setenv("SCHEDULER_INTERVAL", "30s")
	t.Setenv("SCHEDULER_FETCH_LIMIT", "2")
	t.Setenv("SERVER_SHUTDOWN_TIMEOUT", "10s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Scheduler.Interval != 2*time.Minute {
		t.Fatalf("expected interval to be clamped to 2m, got %v", cfg.Scheduler.Interval)
	}
}
