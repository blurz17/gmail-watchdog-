package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Task represents a function to be executed periodically.
type Task struct {
	Name     string
	Interval time.Duration
	Fn       func(ctx context.Context) error
}

// Scheduler manages periodic task execution.
type Scheduler struct {
	tasks  []Task
	logger *slog.Logger
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// New creates a new Scheduler.
func New(logger *slog.Logger) *Scheduler {
	return &Scheduler{
		logger: logger,
	}
}

// Add registers a task for periodic execution.
func (s *Scheduler) Add(task Task) {
	s.tasks = append(s.tasks, task)
}

// Start begins executing all registered tasks.
func (s *Scheduler) Start(ctx context.Context) {
	ctx, s.cancel = context.WithCancel(ctx)

	for _, task := range s.tasks {
		s.wg.Add(1)
		go s.runTask(ctx, task)
	}

	s.logger.Info("scheduler started", "tasks", len(s.tasks))
}

// Stop gracefully stops all tasks and waits for them to complete.
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	s.logger.Info("scheduler stopped")
}

func (s *Scheduler) runTask(ctx context.Context, task Task) {
	defer s.wg.Done()

	s.logger.Info("task registered",
		"task", task.Name,
		"interval", task.Interval,
	)

	// Run immediately on start, then on interval.
	s.executeTask(ctx, task)

	ticker := time.NewTicker(task.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("task stopping", "task", task.Name)
			return
		case <-ticker.C:
			s.executeTask(ctx, task)
		}
	}
}

func (s *Scheduler) executeTask(ctx context.Context, task Task) {
	start := time.Now()

	s.logger.Debug("task executing", "task", task.Name)

	if err := task.Fn(ctx); err != nil {
		s.logger.Error("task failed",
			"task", task.Name,
			"error", err,
			"duration", time.Since(start),
		)
	} else {
		s.logger.Debug("task completed",
			"task", task.Name,
			"duration", time.Since(start),
		)
	}
}
