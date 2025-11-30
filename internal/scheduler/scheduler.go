package scheduler

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

// Processor defines the behavior required by Scheduler.
type Processor interface {
	ProcessPendingMessages(ctx context.Context) error
}

// Scheduler drives periodic message processing without cron.
type Scheduler struct {
	processor Processor
	interval  time.Duration
	logger    *log.Logger

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

// ErrAlreadyRunning is emitted when start is called twice.
var ErrAlreadyRunning = errors.New("scheduler already running")

// ErrNotRunning is emitted when trying to stop an idle scheduler.
var ErrNotRunning = errors.New("scheduler not running")

// New builds a scheduler.
func New(processor Processor, interval time.Duration, logger *log.Logger) *Scheduler {
	if interval <= 0 {
		interval = 2 * time.Minute
	}
	if logger == nil {
		logger = log.New(log.Writer(), "scheduler ", log.LstdFlags)
	}
	return &Scheduler{processor: processor, interval: interval, logger: logger}
}

// Start begins the background loop.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrAlreadyRunning
	}

	if ctx == nil {
		ctx = context.Background()
	}

	loopCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.running = true

	go s.run(loopCtx)
	s.logger.Println("scheduler started")

	return nil
}

// Stop cancels the loop.
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return ErrNotRunning
	}

	s.cancel()
	s.running = false
	s.logger.Println("scheduler stopped")
	return nil
}

// IsRunning reports the scheduler state.
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *Scheduler) run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.execute(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.execute(ctx)
		}
	}
}

func (s *Scheduler) execute(ctx context.Context) {
	if err := s.processor.ProcessPendingMessages(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		s.logger.Printf("scheduler iteration failed: %v", err)
	}
}
