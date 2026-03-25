// Tests for the infra package using miniredis (in-process Redis, no Docker needed).
// All tests run in < 1 second and have no external dependencies.
package infra_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"fyers-trading/infra"
)

// newTestStore spins up an in-process miniredis and returns a connected RedisStore.
// Callers must call mr.Close() when done.
func newTestStore(t *testing.T) (infra.Store, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	store, err := infra.NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	return store, mr
}

// ---------------------------------------------------------------------------
// Core primitive tests
// ---------------------------------------------------------------------------

func TestStore_SetGet(t *testing.T) {
	store, mr := newTestStore(t)
	defer mr.Close()
	defer store.Close()

	ctx := context.Background()

	if err := store.Set(ctx, "hello", "world", time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	val, found, err := store.Get(ctx, "hello")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found")
	}
	if val != "world" {
		t.Fatalf("expected 'world', got %q", val)
	}
}

func TestStore_GetMissingKey(t *testing.T) {
	store, mr := newTestStore(t)
	defer mr.Close()
	defer store.Close()

	_, found, err := store.Get(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Fatal("expected key NOT to be found")
	}
}

func TestStore_Delete(t *testing.T) {
	store, mr := newTestStore(t)
	defer mr.Close()
	defer store.Close()

	ctx := context.Background()
	_ = store.Set(ctx, "key", "value", time.Minute)
	if err := store.Delete(ctx, "key"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, found, _ := store.Get(ctx, "key")
	if found {
		t.Fatal("expected key to be deleted")
	}
}

func TestStore_TTLExpiry(t *testing.T) {
	store, mr := newTestStore(t)
	defer mr.Close()
	defer store.Close()

	ctx := context.Background()
	_ = store.Set(ctx, "expiring", "soon", 1*time.Second)

	// Fast-forward miniredis clock past the TTL
	mr.FastForward(2 * time.Second)

	_, found, err := store.Get(ctx, "expiring")
	if err != nil {
		t.Fatalf("Get after expiry: %v", err)
	}
	if found {
		t.Fatal("expected key to have expired")
	}
}

// ---------------------------------------------------------------------------
// Pipeline state persistence tests
// ---------------------------------------------------------------------------

func TestStore_PipelineStatePersistence(t *testing.T) {
	store, mr := newTestStore(t)
	defer mr.Close()
	defer store.Close()

	ctx := context.Background()
	atm := 23550
	symbols := []string{"NSE:NIFTY2631723400CE", "NSE:NIFTY2631723700PE"}

	if err := store.SavePipelineState(ctx, atm, symbols); err != nil {
		t.Fatalf("SavePipelineState: %v", err)
	}

	restoredATM, restoredSyms, found, err := store.LoadPipelineState(ctx)
	if err != nil {
		t.Fatalf("LoadPipelineState: %v", err)
	}
	if !found {
		t.Fatal("expected pipeline state to be found")
	}
	if restoredATM != atm {
		t.Fatalf("ATM mismatch: want %d, got %d", atm, restoredATM)
	}
	if len(restoredSyms) != len(symbols) {
		t.Fatalf("symbols length mismatch: want %d, got %d", len(symbols), len(restoredSyms))
	}
	for i := range symbols {
		if restoredSyms[i] != symbols[i] {
			t.Fatalf("symbol[%d] mismatch: want %q, got %q", i, symbols[i], restoredSyms[i])
		}
	}
}

func TestStore_PipelineStateNotFound(t *testing.T) {
	store, mr := newTestStore(t)
	defer mr.Close()
	defer store.Close()

	_, _, found, err := store.LoadPipelineState(context.Background())
	if err != nil {
		t.Fatalf("LoadPipelineState on empty store: %v", err)
	}
	if found {
		t.Fatal("expected no pipeline state in empty store")
	}
}

// ---------------------------------------------------------------------------
// Tick price warmup
// ---------------------------------------------------------------------------

func TestStore_TickPriceRoundTrip(t *testing.T) {
	store, mr := newTestStore(t)
	defer mr.Close()
	defer store.Close()

	ctx := context.Background()
	if err := store.SaveTickPrice(ctx, "NSE:NIFTY50-INDEX", 23501.75); err != nil {
		t.Fatalf("SaveTickPrice: %v", err)
	}
	ltp, found, err := store.LoadTickPrice(ctx, "NSE:NIFTY50-INDEX")
	if err != nil {
		t.Fatalf("LoadTickPrice: %v", err)
	}
	if !found {
		t.Fatal("expected tick price to be found")
	}
	if ltp < 23501.00 || ltp > 23502.00 {
		t.Fatalf("LTP precision error: got %f", ltp)
	}
}

// ---------------------------------------------------------------------------
// Session PnL
// ---------------------------------------------------------------------------

func TestStore_SessionPnLRoundTrip(t *testing.T) {
	store, mr := newTestStore(t)
	defer mr.Close()
	defer store.Close()

	ctx := context.Background()
	_ = store.SaveSessionPnL(ctx, -1250.50)
	pnl, found, err := store.LoadSessionPnL(ctx)
	if err != nil {
		t.Fatalf("LoadSessionPnL: %v", err)
	}
	if !found || pnl > -1250.00 || pnl < -1251.00 {
		t.Fatalf("PnL round-trip failed: found=%v pnl=%f", found, pnl)
	}
}

// ---------------------------------------------------------------------------
// Strategy field persistence
// ---------------------------------------------------------------------------

func TestStore_StrategyFieldRoundTrip(t *testing.T) {
	store, mr := newTestStore(t)
	defer mr.Close()
	defer store.Close()

	ctx := context.Background()
	_ = store.SaveStrategyField(ctx, "trailing_high", "23650.75")
	val, found, err := store.LoadStrategyField(ctx, "trailing_high")
	if err != nil {
		t.Fatalf("LoadStrategyField: %v", err)
	}
	if !found || val != "23650.75" {
		t.Fatalf("strategy field round-trip failed: found=%v val=%q", found, val)
	}
}

// ---------------------------------------------------------------------------
// Healthcheck
// ---------------------------------------------------------------------------

func TestStore_PingHealthy(t *testing.T) {
	store, mr := newTestStore(t)
	defer mr.Close()
	defer store.Close()

	if err := store.Ping(context.Background()); err != nil {
		t.Fatalf("Ping on live Redis: %v", err)
	}
}

// ---------------------------------------------------------------------------
// DumpAll / EOD export
// ---------------------------------------------------------------------------

func TestStore_DumpAll(t *testing.T) {
	store, mr := newTestStore(t)
	defer mr.Close()
	defer store.Close()

	ctx := context.Background()
	_ = store.SavePipelineState(ctx, 23500, []string{"NSE:NIFTY2631723350CE"})
	_ = store.SaveSessionPnL(ctx, 500.0)

	dump, err := store.DumpAll(ctx)
	if err != nil {
		t.Fatalf("DumpAll: %v", err)
	}
	if _, ok := dump["pipeline:active_atm"]; !ok {
		t.Fatal("DumpAll missing pipeline:active_atm")
	}
	if _, ok := dump["session:pnl"]; !ok {
		t.Fatal("DumpAll missing session:pnl")
	}
}

// ---------------------------------------------------------------------------
// NoopStore tests — must never error and never hang
// ---------------------------------------------------------------------------

func TestNoopStore_NeverErrors(t *testing.T) {
	store := infra.NewNoopStore()
	ctx := context.Background()

	if err := store.Set(ctx, "k", "v", time.Minute); err != nil {
		t.Fatalf("Noop Set: %v", err)
	}
	if _, _, err := store.Get(ctx, "k"); err != nil {
		t.Fatalf("Noop Get: %v", err)
	}
	if err := store.Delete(ctx, "k"); err != nil {
		t.Fatalf("Noop Delete: %v", err)
	}
	if err := store.SavePipelineState(ctx, 23500, nil); err != nil {
		t.Fatalf("Noop SavePipelineState: %v", err)
	}
	_, _, found, err := store.LoadPipelineState(ctx)
	if err != nil || found {
		t.Fatalf("Noop LoadPipelineState: found=%v err=%v", found, err)
	}
	if err := store.Ping(ctx); err != nil {
		t.Fatalf("Noop Ping: %v", err)
	}
	dump, err := store.DumpAll(ctx)
	if err != nil || len(dump) != 0 {
		t.Fatalf("Noop DumpAll: len=%d err=%v", len(dump), err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Noop Close: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Replicator test
// ---------------------------------------------------------------------------

func TestReplicator_CallsSnapshotAndFlushesOnCancel(t *testing.T) {
	store, mr := newTestStore(t)
	defer mr.Close()
	defer store.Close()

	logger := noopLogger()
	replicator := infra.NewReplicator(store, logger, 100*time.Millisecond)

	callCount := 0
	snapshotFn := func(ctx context.Context, s infra.Store) error {
		callCount++
		return s.Set(ctx, "fyers:replicated", "yes", time.Minute)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		replicator.Run(ctx, snapshotFn)
		close(done)
	}()

	// Let it tick at least twice
	time.Sleep(350 * time.Millisecond)
	cancel()
	<-done // Ensure final flush ran

	if callCount < 2 {
		t.Fatalf("expected at least 2 snapshot calls, got %d", callCount)
	}

	// Verify final flush actually wrote to Redis
	val, found, _ := store.Get(context.Background(), "fyers:replicated")
	if !found || val != "yes" {
		t.Fatalf("final snapshot data not found in Redis: found=%v val=%q", found, val)
	}
}
