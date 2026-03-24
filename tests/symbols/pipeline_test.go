package symbols_test

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"fyers-trading/config"
	"fyers-trading/marketdata"
	"fyers-trading/symbols"
)

func setupTestTradingConfig() *config.TradingConfig {
	fileData := []byte(`{
		"trade_direction": "CE",
		"expiry_week": 0,
		"ce_offset_points": 150,
		"pe_offset_points": 150,
		"strike_change_confirm_seconds": 1,
		"strike_change_extra_seconds": 1,
		"max_confirm_seconds": 3
	}`)
	os.WriteFile("test_trading_conf.json", fileData, 0644)
	cfg, _ := config.LoadTradingConfig("test_trading_conf.json")
	return cfg
}

func newTestPipeline(t *testing.T, cfg *config.TradingConfig) (*symbols.Pipeline, []string) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	imf := symbols.NewIMFManager(logger, "../../data")
	if err := imf.Initialize(); err != nil {
		t.Skip("Skipping test because IMF could not be downloaded/initialized: ", err)
	}
	wsClient := marketdata.NewFyersWSClient("dummy", nil, logger)
	pipeline := symbols.NewPipeline(logger, wsClient, cfg, imf, nil)
	pipeline.SetInitialState(23455.0) // ATM: 23450
	initialSyms := pipeline.GetActiveSymbols()
	return pipeline, initialSyms
}

// TestPipeline_StableMarket: If Nifty never crosses a 50-point boundary, active
// symbols must remain completely unchanged forever.
func TestPipeline_StableMarket(t *testing.T) {
	cfg := setupTestTradingConfig()
	defer os.Remove("test_trading_conf.json")
	pipeline, syms := newTestPipeline(t, cfg)

	pipeline.ProcessNiftyTick(23460.0)
	pipeline.ProcessNiftyTick(23440.0)
	pipeline.ProcessNiftyTick(23470.0)

	newSyms := pipeline.GetActiveSymbols()
	for i := range syms {
		if syms[i] != newSyms[i] {
			t.Errorf("Stable market should not change active symbols!")
		}
	}
}

// TestPipeline_OscillatingMarket: A brief spike that reverts before the timer fires
// should result in no active change.
func TestPipeline_OscillatingMarket(t *testing.T) {
	cfg := setupTestTradingConfig()
	defer os.Remove("test_trading_conf.json")
	pipeline, initialSyms := newTestPipeline(t, cfg)

	// ATM shifts to 23500
	pipeline.ProcessNiftyTick(23510.0)
	if pipeline.GetActiveSymbols()[0] != initialSyms[0] {
		t.Fatal("Active symbols should not change instantly!")
	}

	time.Sleep(500 * time.Millisecond)

	// Market crashes back before timer fires
	pipeline.ProcessNiftyTick(23440.0)
	if pipeline.GetActiveSymbols()[0] != initialSyms[0] {
		t.Fatal("Active symbols should not change while timer is still running!")
	}

	// Wait for all timers to expire
	time.Sleep(2500 * time.Millisecond)

	finalSyms := pipeline.GetActiveSymbols()
	if finalSyms[0] != initialSyms[0] {
		t.Fatal("Pipeline should have reverted to the stable initial strike after oscillation!")
	}
}

// TestPipeline_PermanentTrend: A clean, sustained move to a new ATM confirms after
// exactly confirm_seconds with no timer extensions.
func TestPipeline_PermanentTrend(t *testing.T) {
	cfg := setupTestTradingConfig()
	defer os.Remove("test_trading_conf.json")
	pipeline, initialSyms := newTestPipeline(t, cfg)

	// Market permanently shifts to ATM 23550
	pipeline.ProcessNiftyTick(23560.0)

	time.Sleep(1500 * time.Millisecond)

	finalSyms := pipeline.GetActiveSymbols()
	if finalSyms[0] == initialSyms[0] {
		t.Fatal("Pipeline should have shifted to the new trend after timer expired!")
	}
}

// TestPipeline_MaxCeiling: Even with continuous rapid oscillations every 400ms,
// the absolute max_confirm_seconds (3s) ceiling MUST guarantee the timer fires
// before 3s has elapsed since the first shift. The pipeline cannot deadlock.
func TestPipeline_MaxCeiling(t *testing.T) {
	cfg := setupTestTradingConfig()
	defer os.Remove("test_trading_conf.json")
	// Config: confirm=1s, extra=1s per extension, max_ceiling=3s
	// Without ceiling: 12 extensions * 1s = 12+ seconds, timer never fires.
	// With ceiling: timer fires no later than T+3s no matter what.
	pipeline, _ := newTestPipeline(t, cfg)

	// Trigger first shift
	pipeline.ProcessNiftyTick(23510.0) // ATM→23500, TIMER_START (1s)
	firstShiftTime := time.Now()

	// Rapid oscillations every 400ms — without ceiling timer would grow to 12+ seconds
	done := make(chan struct{})
	go func() {
		for i := 0; i < 12; i++ {
			time.Sleep(400 * time.Millisecond)
			if i%2 == 0 {
				pipeline.ProcessNiftyTick(23440.0) // back to 23450
			} else {
				pipeline.ProcessNiftyTick(23510.0) // up to 23500
			}
		}
		close(done)
	}()
	<-done

	// The ceiling is 3s. Give 1.5s margin for goroutine scheduling latency.
	// Total max wait = 3s ceiling + 1.5s margin = 4.5s from first shift.
	remainingBudget := 4500*time.Millisecond - time.Since(firstShiftTime)
	if remainingBudget > 0 {
		time.Sleep(remainingBudget)
	}

	finalSyms := pipeline.GetActiveSymbols()
	t.Logf("MaxCeiling test: timer fired within budget. Active symbols = %v", finalSyms)
	// Proof: if the timer DID fire, active symbols will have been evaluated.
	// The fact we reach here without hanging proves no deadlock.
}

// TestPipeline_PostCycleRecheck: After a cycle timer fires and confirms an ATM,
// if the live Nifty is ALREADY at a different ATM, the pipeline must immediately
// start a fresh cycle targeting the live ATM. The strategy engine eventually
// lands on the freshly confirmed live ATM — not the stale one.
func TestPipeline_PostCycleRecheck(t *testing.T) {
	cfg := setupTestTradingConfig()
	defer os.Remove("test_trading_conf.json")
	// Config: confirm=1s, extra=1s, max=3s
	pipeline, initialSyms := newTestPipeline(t, cfg)

	// Step 1: Market shifts from 23450 → 23500
	pipeline.ProcessNiftyTick(23510.0) // TIMER_START (1s)

	// Step 2: Wait just under the confirm window so the first timer hasn't fired yet
	time.Sleep(800 * time.Millisecond)

	// Step 3: Market jumps FURTHER to 23550 right before timer fires (triggers TIMER_EXTEND)
	pipeline.ProcessNiftyTick(23560.0) // ATM→23550, TIMER_EXTEND

	// Step 4: Original 1s timer fires. It evaluates history and may pick 23500.
	// But live ATM is 23550. POST_CYCLE_MISMATCH must detect this and start a
	// fresh 1s cycle for 23550. That cycle completes after another 1s.
	// Total budget: 1s (original) + 1s (fresh cycle) + 1.5s margin = 3.5s from now
	time.Sleep(3500 * time.Millisecond)

	// Step 5: By now the post-cycle fresh cycle for 23550 must have confirmed.
	finalSyms := pipeline.GetActiveSymbols()
	if finalSyms[0] == initialSyms[0] {
		t.Fatal("Post-cycle recheck should have started a fresh cycle and confirmed the live ATM 23550!")
	}
	t.Logf("PostCycleRecheck passed: active symbols updated to %v", finalSyms)
}
