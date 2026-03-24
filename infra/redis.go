// Package infra provides infrastructure-level clients for external services.
// This file implements the Redis state persistence layer for crash recovery.
//
// Architecture:
//   - Store interface: all app code depends on this, never on concrete Redis.
//   - RedisStore: production implementation backed by go-redis/v9.
//   - NoopStore (noop.go): silent no-op for local dev when REDIS_URL is unset.
//
// Key design decisions:
//   - All operations accept a context.Context so they respect graceful shutdown.
//   - All state is serialized as JSON so it can be trivially dumped to S3 at EOD.
//   - Default TTL is 26 hours — survives overnight but expires before next session.
//   - The Connect() function never fatally exits; it returns an error so main.go
//     can fall back to NoopStore gracefully.
package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// DefaultTTL: 26h ensures state survives a nightly restart but auto-clears
	// before the next trading session even if the EOD cleanup job misses.
	DefaultTTL = 26 * time.Hour

	// Key prefixes — all keys are namespaced under "fyers:" for easy inspection.
	keyPrefix              = "fyers:"
	keyPipelineATM         = keyPrefix + "pipeline:active_atm"
	keyPipelineSymbols     = keyPrefix + "pipeline:active_symbols"
	keyPipelineTradingDate = keyPrefix + "pipeline:trading_date" // IST date: "2006-01-02"
	keyTickPrice           = keyPrefix + "tick:"                 // + symbol
	keyOrderBook           = keyPrefix + "orders"                // JSON hash
	keyPositions           = keyPrefix + "positions"             // JSON hash
	keySessionPnL          = keyPrefix + "session:pnl"
	keyStrategyState       = keyPrefix + "strategy:" // + field name
)

// ---------------------------------------------------------------------------
// Store interface — the only thing the rest of the application sees.
// ---------------------------------------------------------------------------

// Store defines the contract for state persistence.
// All methods accept a context so they respect cancellation and timeouts.
type Store interface {
	// --- Primitive operations ---
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Get(ctx context.Context, key string) (value string, found bool, err error)
	Delete(ctx context.Context, key string) error

	// --- Pipeline state ---
	SavePipelineState(ctx context.Context, atm int, symbols []string) error
	LoadPipelineState(ctx context.Context) (atm int, symbols []string, found bool, err error)

	// --- TickStore warmup ---
	SaveTickPrice(ctx context.Context, symbol string, ltp float64) error
	LoadTickPrice(ctx context.Context, symbol string) (ltp float64, found bool, err error)

	// --- Order book & positions (used by OMS when built) ---
	SaveOrderBook(ctx context.Context, data interface{}) error
	LoadOrderBook(ctx context.Context, dest interface{}) (found bool, err error)
	SavePositions(ctx context.Context, data interface{}) error
	LoadPositions(ctx context.Context, dest interface{}) (found bool, err error)

	// --- Session P&L ---
	SaveSessionPnL(ctx context.Context, pnl float64) error
	LoadSessionPnL(ctx context.Context) (pnl float64, found bool, err error)

	// --- Strategy state (trailing highs/lows, no-trade zone, etc.) ---
	SaveStrategyField(ctx context.Context, field, value string) error
	LoadStrategyField(ctx context.Context, field string) (value string, found bool, err error)

	// --- Health & lifecycle ---
	Ping(ctx context.Context) error
	// DumpAll returns every fyers:* key as a JSON map — used for EOD S3 export.
	DumpAll(ctx context.Context) (map[string]string, error)
	Close() error
}

// ---------------------------------------------------------------------------
// RedisStore — production implementation.
// ---------------------------------------------------------------------------

// RedisStore implements Store using go-redis/v9.
type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisStore connects to Redis at the given URL.
// URL format: "redis://[:password@]host:port/db" or "redis://localhost:6379/0"
//
// Returns an error if the connection cannot be established — callers should
// fall back to NewNoopStore() and log a warning rather than crashing.
func NewRedisStore(ctx context.Context, redisURL string) (*RedisStore, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Redis URL %q: %w", redisURL, err)
	}

	client := redis.NewClient(opts)

	// Validate the connection immediately at startup
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("Redis ping failed: %w", err)
	}

	return &RedisStore{client: client, ttl: DefaultTTL}, nil
}

// ---------------------------------------------------------------------------
// Primitive operations
// ---------------------------------------------------------------------------

func (r *RedisStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisStore) Get(ctx context.Context, key string) (string, bool, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}

func (r *RedisStore) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// ---------------------------------------------------------------------------
// Pipeline state
// ---------------------------------------------------------------------------

func istDate() string {
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		ist = time.FixedZone("IST", 5*3600+30*60)
	}
	return time.Now().In(ist).Format("2006-01-02")
}

func (r *RedisStore) SavePipelineState(ctx context.Context, atm int, symbols []string) error {
	symJSON, err := json.Marshal(symbols)
	if err != nil {
		return fmt.Errorf("marshal symbols: %w", err)
	}
	pipe := r.client.Pipeline()
	pipe.Set(ctx, keyPipelineATM, fmt.Sprintf("%d", atm), r.ttl)
	pipe.Set(ctx, keyPipelineSymbols, string(symJSON), r.ttl)
	pipe.Set(ctx, keyPipelineTradingDate, istDate(), r.ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *RedisStore) LoadPipelineState(ctx context.Context) (int, []string, bool, error) {
	// Check trading date first — discard if it's from a previous IST day
	savedDate, found, err := r.Get(ctx, keyPipelineTradingDate)
	if err != nil || !found {
		return 0, nil, false, err
	}
	if savedDate != istDate() {
		// State is from a different trading day — do not restore it
		return 0, nil, false, nil
	}

	atmStr, found, err := r.Get(ctx, keyPipelineATM)
	if err != nil || !found {
		return 0, nil, false, err
	}
	var atm int
	if _, err := fmt.Sscanf(atmStr, "%d", &atm); err != nil {
		return 0, nil, false, fmt.Errorf("parse atm %q: %w", atmStr, err)
	}

	symStr, found, err := r.Get(ctx, keyPipelineSymbols)
	if err != nil || !found {
		return 0, nil, false, err
	}
	var symbols []string
	if err := json.Unmarshal([]byte(symStr), &symbols); err != nil {
		return 0, nil, false, fmt.Errorf("unmarshal symbols: %w", err)
	}
	return atm, symbols, true, nil
}

// ---------------------------------------------------------------------------
// Tick price warmup
// ---------------------------------------------------------------------------

func (r *RedisStore) SaveTickPrice(ctx context.Context, symbol string, ltp float64) error {
	return r.client.Set(ctx, keyTickPrice+symbol, fmt.Sprintf("%f", ltp), r.ttl).Err()
}

func (r *RedisStore) LoadTickPrice(ctx context.Context, symbol string) (float64, bool, error) {
	val, found, err := r.Get(ctx, keyTickPrice+symbol)
	if err != nil || !found {
		return 0, false, err
	}
	var ltp float64
	if _, err := fmt.Sscanf(val, "%f", &ltp); err != nil {
		return 0, false, fmt.Errorf("parse ltp %q: %w", val, err)
	}
	return ltp, true, nil
}

// ---------------------------------------------------------------------------
// Order book & positions (data is caller-serialized as arbitrary struct)
// ---------------------------------------------------------------------------

func (r *RedisStore) SaveOrderBook(ctx context.Context, data interface{}) error {
	return r.setJSON(ctx, keyOrderBook, data)
}

func (r *RedisStore) LoadOrderBook(ctx context.Context, dest interface{}) (bool, error) {
	return r.getJSON(ctx, keyOrderBook, dest)
}

func (r *RedisStore) SavePositions(ctx context.Context, data interface{}) error {
	return r.setJSON(ctx, keyPositions, data)
}

func (r *RedisStore) LoadPositions(ctx context.Context, dest interface{}) (bool, error) {
	return r.getJSON(ctx, keyPositions, dest)
}

// ---------------------------------------------------------------------------
// Session P&L
// ---------------------------------------------------------------------------

func (r *RedisStore) SaveSessionPnL(ctx context.Context, pnl float64) error {
	return r.client.Set(ctx, keySessionPnL, fmt.Sprintf("%f", pnl), r.ttl).Err()
}

func (r *RedisStore) LoadSessionPnL(ctx context.Context) (float64, bool, error) {
	val, found, err := r.Get(ctx, keySessionPnL)
	if err != nil || !found {
		return 0, false, err
	}
	var pnl float64
	if _, err := fmt.Sscanf(val, "%f", &pnl); err != nil {
		return 0, false, err
	}
	return pnl, true, nil
}

// ---------------------------------------------------------------------------
// Strategy state (arbitrary key-value for trailing high/low, no-trade zone, etc.)
// ---------------------------------------------------------------------------

func (r *RedisStore) SaveStrategyField(ctx context.Context, field, value string) error {
	return r.client.Set(ctx, keyStrategyState+field, value, r.ttl).Err()
}

func (r *RedisStore) LoadStrategyField(ctx context.Context, field string) (string, bool, error) {
	return r.Get(ctx, keyStrategyState+field)
}

// ---------------------------------------------------------------------------
// Health & lifecycle
// ---------------------------------------------------------------------------

func (r *RedisStore) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// DumpAll returns every fyers:* key with its current value.
// Used at end-of-day to produce the full state snapshot for S3 export.
func (r *RedisStore) DumpAll(ctx context.Context) (map[string]string, error) {
	keys, err := r.client.Keys(ctx, keyPrefix+"*").Result()
	if err != nil {
		return nil, fmt.Errorf("keys scan: %w", err)
	}
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		val, err := r.client.Get(ctx, key).Result()
		if err != nil && err != redis.Nil {
			return nil, fmt.Errorf("get %q: %w", key, err)
		}
		// Strip the namespace prefix for cleaner EOD export keys
		result[strings.TrimPrefix(key, keyPrefix)] = val
	}
	return result, nil
}

func (r *RedisStore) Close() error {
	return r.client.Close()
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (r *RedisStore) setJSON(ctx context.Context, key string, data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal %q: %w", key, err)
	}
	return r.client.Set(ctx, key, string(b), r.ttl).Err()
}

func (r *RedisStore) getJSON(ctx context.Context, key string, dest interface{}) (bool, error) {
	val, found, err := r.Get(ctx, key)
	if err != nil || !found {
		return false, err
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false, fmt.Errorf("unmarshal %q: %w", key, err)
	}
	return true, nil
}
