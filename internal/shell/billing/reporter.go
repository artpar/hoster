// Package billing provides integration with APIGate for usage metering and billing.
package billing

import (
	"context"
	"log/slog"
	"time"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/shell/store"
)

// =============================================================================
// Background Reporter
// =============================================================================

// Reporter batches and reports usage events to APIGate in the background.
type Reporter struct {
	store     store.Store
	client    Client
	interval  time.Duration
	batchSize int
	logger    *slog.Logger
	stopCh    chan struct{}
	doneCh    chan struct{}
}

// ReporterConfig holds configuration for the background reporter.
type ReporterConfig struct {
	Store     store.Store
	Client    Client
	Interval  time.Duration
	BatchSize int
	Logger    *slog.Logger
}

// NewReporter creates a new background reporter.
func NewReporter(cfg ReporterConfig) *Reporter {
	if cfg.Interval == 0 {
		cfg.Interval = 60 * time.Second
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 100
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &Reporter{
		store:     cfg.Store,
		client:    cfg.Client,
		interval:  cfg.Interval,
		batchSize: cfg.BatchSize,
		logger:    cfg.Logger,
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}
}

// Start begins the background reporting loop.
// It runs until Stop() is called or the context is cancelled.
func (r *Reporter) Start(ctx context.Context) {
	r.logger.Info("starting billing reporter",
		"interval", r.interval,
		"batch_size", r.batchSize,
	)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	defer close(r.doneCh)

	// Report any pending events on startup
	r.reportBatch(ctx)

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("billing reporter stopped due to context cancellation")
			return
		case <-r.stopCh:
			r.logger.Info("billing reporter stopped")
			return
		case <-ticker.C:
			r.reportBatch(ctx)
		}
	}
}

// Stop signals the reporter to stop and waits for it to finish.
func (r *Reporter) Stop() {
	close(r.stopCh)
	<-r.doneCh
}

// reportBatch retrieves unreported events and sends them to APIGate.
func (r *Reporter) reportBatch(ctx context.Context) {
	events, err := r.store.GetUnreportedEvents(ctx, r.batchSize)
	if err != nil {
		r.logger.Error("failed to get unreported events", "error", err)
		return
	}

	if len(events) == 0 {
		return
	}

	r.logger.Debug("reporting usage events", "count", len(events))

	if err := r.client.MeterUsageBatch(ctx, events); err != nil {
		r.logger.Error("failed to report usage events",
			"error", err,
			"count", len(events),
		)
		return
	}

	// Mark events as reported
	ids := make([]string, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}

	if err := r.store.MarkEventsReported(ctx, ids, time.Now()); err != nil {
		r.logger.Error("failed to mark events as reported",
			"error", err,
			"count", len(ids),
		)
		return
	}

	r.logger.Info("reported usage events", "count", len(events))
}

// ReportNow triggers an immediate report cycle (useful for testing).
func (r *Reporter) ReportNow(ctx context.Context) {
	r.reportBatch(ctx)
}

// =============================================================================
// Event Recording Helper
// =============================================================================

// RecordEvent is a convenience function to record a usage event.
// It creates the event and stores it for later batch reporting.
func RecordEvent(ctx context.Context, s store.Store, userID string, eventType domain.EventType, resourceID, resourceType string, metadata map[string]string) error {
	event := domain.NewMeterEvent(
		generateEventID(),
		userID,
		eventType,
		resourceID,
		resourceType,
	)

	if metadata != nil {
		event.Metadata = metadata
	}

	return s.CreateUsageEvent(ctx, &event)
}

// generateEventID generates a unique event ID.
func generateEventID() string {
	return "evt_" + time.Now().Format("20060102150405") + "_" + randomSuffix()
}

// randomSuffix generates a random suffix for event IDs.
func randomSuffix() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
