package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// Replicator periodically snapshots in-memory state to Redis.
// It runs as a dedicated goroutine and respects context cancellation
// for graceful shutdown — it flushes one final snapshot before exiting.
//
// Usage:
//
//	replicator := infra.NewReplicator(store, logger, 5*time.Second)
//	go replicator.Run(ctx, snapshotFn)
type Replicator struct {
	store    Store
	logger   *slog.Logger
	interval time.Duration
}

// SnapshotFn is called on every replication tick to gather the current state.
// Return an error to skip that snapshot (it will be retried next interval).
type SnapshotFn func(ctx context.Context, store Store) error

// NewReplicator creates a Replicator that syncs every interval.
func NewReplicator(store Store, logger *slog.Logger, interval time.Duration) *Replicator {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &Replicator{store: store, logger: logger, interval: interval}
}

// Run starts the replication loop. Blocks until ctx is cancelled.
// On cancellation, one final snapshot is attempted before returning.
func (r *Replicator) Run(ctx context.Context, snapshot SnapshotFn) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	r.logger.Info("Redis replicator started", "interval", r.interval)

	for {
		select {
		case <-ticker.C:
			if err := snapshot(ctx, r.store); err != nil {
				r.logger.Warn("Redis snapshot failed", "error", err)
			} else {
				r.logger.Debug("Redis snapshot complete")
			}

		case <-ctx.Done():
			// Final flush before exit — use a fresh background context since
			// the parent context is already cancelled.
			r.logger.Info("Replicator shutting down, flushing final snapshot...")
			flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := snapshot(flushCtx, r.store); err != nil {
				r.logger.Error("Final Redis snapshot failed", "error", err)
			} else {
				r.logger.Info("Final Redis snapshot complete")
			}
			return
		}
	}
}

// ---------------------------------------------------------------------------
// ConnectStore is the main.go entry point for Redis setup.
// It returns a ready-to-use Store, always — either RedisStore or NoopStore.
// Callers should check the returned bool to know which one they got.
// ---------------------------------------------------------------------------

// ConnectStore tries to connect to Redis at the given URL.
// If the URL is empty or the connection fails, it gracefully falls back to
// a NoopStore and logs a warning — the app continues without persistence.
//
// Returns:
//   - store: always non-nil (either RedisStore or NoopStore)
//   - usingRedis: true if real Redis is connected
func ConnectStore(ctx context.Context, redisURL string, logger *slog.Logger) (store Store, usingRedis bool) {
	if redisURL == "" {
		logger.Warn("REDIS_URL not set — running without state persistence (NoopStore). " +
			"Set REDIS_URL in .env for crash recovery.")
		return NewNoopStore(), false
	}

	rs, err := NewRedisStore(ctx, redisURL)
	if err != nil {
		logger.Warn("Redis unavailable — falling back to NoopStore (in-memory only mode)",
			"url", redisURL, "error", err)
		return NewNoopStore(), false
	}

	logger.Info("Redis connected", "url", redisURL)
	return rs, true
}

// ---------------------------------------------------------------------------
// EODExporter writes the full Redis state snapshot to a JSON byte slice.
// Call this at end-of-day before the S3 upload.
// ---------------------------------------------------------------------------

// ExportStateJSON returns the entire fyers:* keyspace as a pretty-printed
// JSON byte slice, ready to write to a file or upload to S3.
func ExportStateJSON(ctx context.Context, store Store) ([]byte, error) {
	data, err := store.DumpAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("DumpAll: %w", err)
	}
	return json.MarshalIndent(data, "", "  ")
}
