package infra

import (
	"context"
	"time"
)

// NoopStore is a Store implementation that silently ignores all operations.
//
// Used when:
//   - REDIS_URL is not set in .env (local development)
//   - Redis is unreachable at startup (system continues in-memory-only mode)
//   - Unit tests that don't specifically test Redis behavior
//
// All methods return nil/zero values — never an error.
// This ensures the app never crashes due to missing Redis in dev.
type NoopStore struct{}

// NewNoopStore returns a Store that does nothing — a safe no-op fallback.
func NewNoopStore() Store {
	return &NoopStore{}
}

func (n *NoopStore) Set(_ context.Context, _, _ string, _ time.Duration) error { return nil }
func (n *NoopStore) Get(_ context.Context, _ string) (string, bool, error)     { return "", false, nil }
func (n *NoopStore) Delete(_ context.Context, _ string) error                  { return nil }

func (n *NoopStore) SavePipelineState(_ context.Context, _ int, _ []string) error { return nil }
func (n *NoopStore) LoadPipelineState(_ context.Context) (int, []string, bool, error) {
	return 0, nil, false, nil
}

func (n *NoopStore) SaveTickPrice(_ context.Context, _ string, _ float64) error { return nil }
func (n *NoopStore) LoadTickPrice(_ context.Context, _ string) (float64, bool, error) {
	return 0, false, nil
}

func (n *NoopStore) SaveOrderBook(_ context.Context, _ interface{}) error         { return nil }
func (n *NoopStore) LoadOrderBook(_ context.Context, _ interface{}) (bool, error) { return false, nil }
func (n *NoopStore) SavePositions(_ context.Context, _ interface{}) error         { return nil }
func (n *NoopStore) LoadPositions(_ context.Context, _ interface{}) (bool, error) { return false, nil }

func (n *NoopStore) SaveSessionPnL(_ context.Context, _ float64) error      { return nil }
func (n *NoopStore) LoadSessionPnL(_ context.Context) (float64, bool, error) { return 0, false, nil }

func (n *NoopStore) SaveStrategyField(_ context.Context, _, _ string) error { return nil }
func (n *NoopStore) LoadStrategyField(_ context.Context, _ string) (string, bool, error) {
	return "", false, nil
}

func (n *NoopStore) Ping(_ context.Context) error                        { return nil }
func (n *NoopStore) DumpAll(_ context.Context) (map[string]string, error) { return map[string]string{}, nil }
func (n *NoopStore) Close() error                                         { return nil }
