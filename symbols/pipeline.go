package symbols

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"context"
	"fyers-trading/config"
	"fyers-trading/infra"
	"fyers-trading/marketdata"
)

// PipelineEvent is a structured event written to pipeline_events.txt
type PipelineEvent struct {
	Timestamp string   `json:"timestamp"` // IST formatted
	Event     string   `json:"event"`
	ATM       int      `json:"atm,omitempty"`
	Symbols   []string `json:"symbols,omitempty"`
	Note      string   `json:"note,omitempty"`
}

// EventLogger writes pipeline events to a file for later analytics.
type EventLogger struct {
	file   *os.File
	istLoc *time.Location
}

func NewEventLogger(path string) *EventLogger {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil
	}
	loc, _ := time.LoadLocation("Asia/Kolkata")
	return &EventLogger{file: f, istLoc: loc}
}

func (el *EventLogger) Close() {
	if el != nil && el.file != nil {
		el.file.Close()
	}
}

func (el *EventLogger) Log(event string, atm int, symbols []string, note string) {
	if el == nil || el.file == nil {
		return
	}
	pev := PipelineEvent{
		Timestamp: time.Now().In(el.istLoc).Format("15:04:05"),
		Event:     event,
		ATM:       atm,
		Symbols:   symbols,
		Note:      note,
	}
	data, _ := json.Marshal(pev)
	el.file.Write(append(data, '\n'))
}

type PendingShift struct {
	ATM          int
	Symbols      []string
	SubscribedAt time.Time
}

// Pipeline manages dynamic symbol subscriptions for options based on Nifty ATM changes.
// It implements the sophisticated "fast-moving market lookahead" logic.
type Pipeline struct {
	logger    *slog.Logger
	client    *marketdata.FyersWSClient
	cfg       *config.TradingConfig
	imf       *IMFManager
	evtLogger *EventLogger
	store     infra.Store // Redis state persistence (NoopStore in dev)

	mu sync.Mutex

	// Current active state
	activeATM          int
	activeSymbols      []string
	activeExpiryOffset int // Automatically increments if Fyers rejects an expiry

	// Track last seen Nifty LTP for post-cycle live ATM check
	lastNiftyLTP float64

	// History of pending shifts during an active countdown
	pendingShifts []PendingShift

	shiftTimer       *time.Timer
	timerTarget      time.Time
	// Absolute deadline: timer can never fire later than this.
	maxTimerDeadline time.Time
}

func NewPipeline(logger *slog.Logger, client *marketdata.FyersWSClient, cfg *config.TradingConfig, imf *IMFManager, store infra.Store) *Pipeline {
	if store == nil {
		store = infra.NewNoopStore()
	}
	p := &Pipeline{
		logger:        logger,
		client:        client,
		cfg:           cfg,
		imf:           imf,
		store:         store,
		evtLogger:     NewEventLogger("pipeline_events.txt"),
		pendingShifts: make([]PendingShift, 0),
	}

	// Wire Fyers websocket rejection feedback into the pipeline
	p.client.OnInvalidSymbol = p.handleInvalidSymbols

	return p
}

// SetInitialState is called on boot. It first tries to restore from Redis.
// If Redis has a saved state, restores it immediately (skip first tick wait).
// If not, falls back to computing ATM from the live Nifty LTP.
func (p *Pipeline) SetInitialState(niftyLTP float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Try to restore last-known state from Redis
	restoredATM, restoredSyms, found, err := p.store.LoadPipelineState(ctx)
	if err != nil {
		p.logger.Warn("Could not load pipeline state from Redis, using live Nifty LTP", "error", err)
	}

	if found && restoredATM > 0 && len(restoredSyms) > 0 {
		p.activeATM = restoredATM
		p.activeSymbols = restoredSyms
		p.client.Subscribe(restoredSyms)
		p.logger.Info("Pipeline state RESTORED from Redis",
			"atm", p.activeATM, "symbols", p.activeSymbols)
		p.evtLogger.Log("RESTORED", p.activeATM, restoredSyms, fmt.Sprintf("Nifty LTP=%.2f (restored from Redis)", niftyLTP))
		return nil
	}

	// No Redis state — compute fresh from live LTP
	p.activeATM = ComputeATMStrike(niftyLTP)
	syms, err := p.calculateSymbols(p.activeATM)
	if err != nil {
		return fmt.Errorf("failed to calculate initial symbols: %w", err)
	}
	p.activeSymbols = syms
	p.client.Subscribe(syms)
	p.logger.Info("Pipeline initialized with Active ATM", "atm", p.activeATM, "symbols", p.activeSymbols)
	p.evtLogger.Log("INIT", p.activeATM, syms, fmt.Sprintf("Nifty LTP=%.2f", niftyLTP))

	// Persist the initialized state to Redis immediately
	go func(atm int, currentSyms []string) {
		saveCtx, cancelSave := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelSave()
		if err := p.store.SavePipelineState(saveCtx, atm, currentSyms); err != nil {
			p.logger.Warn("Failed to persist initial pipeline state to Redis", "error", err)
		}
	}(p.activeATM, syms)

	return nil
}

// ProcessNiftyTick should be called by main via tick store or ring buffer consumer 
// whenever a Nifty index tick arrives.
func (p *Pipeline) ProcessNiftyTick(ltp float64) {
	newATM := ComputeATMStrike(ltp)

	p.mu.Lock()
	defer p.mu.Unlock()

	p.lastNiftyLTP = ltp // always track latest Nifty price

	if p.activeATM == 0 {
		return // Not fully initialized yet
	}

	if len(p.pendingShifts) == 0 {
		if newATM == p.activeATM {
			return // Stable state
		}
		// First shift detected!
		p.startShift(newATM)
	} else {
		// A countdown is currently active
		latestPending := p.pendingShifts[len(p.pendingShifts)-1].ATM
		if newATM == latestPending {
			return // We are already exactly tracking this pending ATM
		}

		if newATM == p.activeATM {
			// Market oscillated back exactly to our current active baseline! 
			// We can completely abort the shift cycle and restore absolute stability.
			p.logger.Info("Market strictly recovered to original ATM! Aborting shift timers silently.", "stable_atm", p.activeATM)
			p.pendingShifts = make([]PendingShift, 0)
			if p.shiftTimer != nil {
				p.shiftTimer.Stop()
				p.shiftTimer = nil
			}
			return
		}

		// Prevent infinite buffering of rapidly bouncing ATM strikes (e.g. 23000 -> 22950 -> 23000 -> 22950)
		for _, pending := range p.pendingShifts {
			if pending.ATM == newATM {
				return // We already subscribed to this during the active countdown window to buffer it. Ignore.
			}
		}

		p.logger.Info("Market moved again during pending countdown! Buffering new strikes.", "old_pending", latestPending, "new_pending", newATM)
		p.startShift(newATM) // Add to history, extend timer
	}
}

// GetActiveSymbols provides the CURRENT actively traded symbols for the strategy engine.
func (p *Pipeline) GetActiveSymbols() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Create a copy to prevent race conditions
	cp := make([]string, len(p.activeSymbols))
	copy(cp, p.activeSymbols)
	return cp
}

func (p *Pipeline) startShift(newATM int) {
	now := time.Now()

	syms, err := p.calculateSymbols(newATM)
	if err != nil {
		p.logger.Error("Failed to calculate symbols for shift", "atm", newATM, "err", err)
		return
	}

	// Rule: Immediately subscribe to start buffering historical data!
	p.logger.Info("====== PIPELINE SHIFT: SUBSCRIBING TO NEW STRIKES ======", "atm", newATM, "symbols", syms)
	p.client.Subscribe(syms)

	p.pendingShifts = append(p.pendingShifts, PendingShift{
		ATM:          newATM,
		Symbols:      syms,
		SubscribedAt: now,
	})

	extraSecs := p.cfg.GetStrikeChangeExtraSeconds()
	if extraSecs <= 0 {
		extraSecs = 60 // fallback safe default
	}
	confirmSecs := p.cfg.GetStrikeChangeConfirmSeconds()
	maxSecs := p.cfg.GetMaxConfirmSeconds()

	if len(p.pendingShifts) == 1 {
		// First shift: start base confirm timer AND set the hard ceiling
		duration := time.Duration(confirmSecs) * time.Second
		p.timerTarget = now.Add(duration)
		p.maxTimerDeadline = now.Add(time.Duration(maxSecs) * time.Second)
		p.logger.Info("Started base confirmation timer for new ATM", "atm", newATM, "duration", duration, "max_deadline", p.maxTimerDeadline.Format("15:04:05"))
		p.evtLogger.Log("TIMER_START", newATM, syms, fmt.Sprintf("confirm=%ds max=%ds", confirmSecs, maxSecs))
		
		p.shiftTimer = time.AfterFunc(duration, p.evaluateShifts)
	} else {
		// Fast-moving market: extend the existing timer by extra time,
		// BUT never push past the absolute max ceiling.
		extraDuration := time.Duration(extraSecs) * time.Second
		newTarget := p.timerTarget.Add(extraDuration)
		if newTarget.After(p.maxTimerDeadline) {
			newTarget = p.maxTimerDeadline
			p.logger.Info("Timer extension capped at max_confirm_seconds ceiling", "atm", newATM, "ceiling", p.maxTimerDeadline.Format("15:04:05"))
		}
		p.timerTarget = newTarget
		
		p.logger.Info("Extended confirmation timer for extra buffering", "atm", newATM, "added_secs", extraSecs)
		p.evtLogger.Log("TIMER_EXTEND", newATM, syms, fmt.Sprintf("added=%ds", extraSecs))

		if p.shiftTimer != nil {
			p.shiftTimer.Stop()
		}
		p.shiftTimer = time.AfterFunc(time.Until(p.timerTarget), p.evaluateShifts)
	}
}

func (p *Pipeline) evaluateShifts() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.pendingShifts) == 0 {
		return
	}

	p.logger.Info("Confirmation timer expired. Evaluating pending shifts buffer...")

	now := time.Now()
	extraDuration := time.Duration(p.cfg.GetStrikeChangeExtraSeconds()) * time.Second

	var selectedShift *PendingShift
	var selectedIndex int

	// Traverse backward from newest shift to oldest shift
	for i := len(p.pendingShifts) - 1; i >= 0; i-- {
		shift := p.pendingShifts[i]

		// The user rule: switch to the chronologically newest strike that has AT LEAST `extra_time` worth of data.
		// (Or if it's the very first shift, it has `confirm_time` data which is always >= `extra_time`)
		if now.Sub(shift.SubscribedAt) >= extraDuration || i == 0 {
			selectedShift = &p.pendingShifts[i]
			selectedIndex = i
			break
		}
	}

	oldActiveSymbols := p.activeSymbols

	// Switch Strategy active trading elements
	p.activeATM = selectedShift.ATM
	p.activeSymbols = selectedShift.Symbols
	p.logger.Info("====== PIPELINE CONFIRMED NEW ACTIVE ATM ======", "atm", p.activeATM, "symbols", p.activeSymbols)
	p.evtLogger.Log("ACTIVE_SWITCH", p.activeATM, p.activeSymbols, "confirmed after timer")

	// Persist to Redis for crash recovery (non-blocking, best-effort)
	go func(atm int, syms []string) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := p.store.SavePipelineState(ctx, atm, syms); err != nil {
			p.logger.Warn("Failed to persist pipeline state to Redis", "error", err)
		}
	}(p.activeATM, append([]string{}, p.activeSymbols...))

	// POST-CYCLE CHECK: Is the live ATM already different from what we just confirmed?
	// This catches cases where the market moved 1-2 strikes during a long oscillation window
	// and the confirmed shift is already stale by the time the timer fired.
	if p.lastNiftyLTP > 0 {
		liveATM := ComputeATMStrike(p.lastNiftyLTP)
		if liveATM != p.activeATM {
			p.logger.Info("Post-cycle check: live ATM differs from confirmed ATM! Starting fresh cycle.",
				"confirmed_atm", p.activeATM, "live_atm", liveATM)
			p.evtLogger.Log("POST_CYCLE_MISMATCH", liveATM, nil,
				fmt.Sprintf("confirmed=%d live=%d starting new cycle", p.activeATM, liveATM))
			p.pendingShifts = make([]PendingShift, 0)
			p.startShift(liveATM)
			return
		}
	}

	// Determine what to unsubscribe and if we need to start a follow-up cycle
	latestShift := p.pendingShifts[len(p.pendingShifts)-1]

	allHeldSymbols := make(map[string]bool)
	for _, sym := range oldActiveSymbols {
		allHeldSymbols[sym] = true
	}
	for _, shift := range p.pendingShifts {
		for _, sym := range shift.Symbols {
			allHeldSymbols[sym] = true
		}
	}

	// Never unsubscribe from the newly confirmed active symbols
	for _, sym := range p.activeSymbols {
		delete(allHeldSymbols, sym)
	}

	if p.activeATM != latestShift.ATM {
		// Market is STILL moving! The selected shift wasn't the absolute latest one.
		p.logger.Info("Market still moving away! Initiating follow-up cycle for remaining strikes", "latest_pending", latestShift.ATM)

		// Keep any shifts NEWER than the one we selected
		newTempPending := p.pendingShifts[selectedIndex+1:]

		// Preserve subscriptions for the remaining pending track
		for _, shift := range newTempPending {
			for _, sym := range shift.Symbols {
				delete(allHeldSymbols, sym)
			}
		}

		p.pendingShifts = newTempPending

		// Start a fresh base cycle since we haven't caught up
		duration := time.Duration(p.cfg.GetStrikeChangeConfirmSeconds()) * time.Second
		p.timerTarget = time.Now().Add(duration)
		p.shiftTimer = time.AfterFunc(duration, p.evaluateShifts)
	} else {
		// We perfectly caught up to the market. Reset state to stable.
		p.pendingShifts = make([]PendingShift, 0)
		p.shiftTimer = nil
		p.logger.Info("Market stability reached. Pending queues cleared.")
	}

	// Unsubscribe from anything obsolete
	unsubs := make([]string, 0, len(allHeldSymbols))
	for sym := range allHeldSymbols {
		unsubs = append(unsubs, sym)
	}

	if len(unsubs) > 0 {
		p.logger.Info("====== PIPELINE CLEANUP: UNSUBSCRIBING FROM OLD STRIKES ======", "symbols", unsubs)
		p.client.Unsubscribe(unsubs)
	}
}

// calculateSymbols builds the CE and PE option symbols for the given ATM strike.
//
// Flow:
//  1. Lock the expiry at expiry_week index from config.
//  2. Get all available CE and PE strikes for that expiry from the IMF.
//  3. Snap the ideal strike to the nearest available one.
//  4. Only advance to the next expiry if the locked expiry has ZERO strikes
//     (i.e. it genuinely doesn't exist in the current IMF data).
//
// Weekly and monthly contracts both use 50-pt intervals (monthly sometimes 25-pt),
// so snapping is always safe — we never need to cross-expiry boundaries for strikes.
func (p *Pipeline) calculateSymbols(atm int) ([]string, error) {
	expiries := p.imf.GetNiftyExpiries()
	if len(expiries) == 0 {
		return nil, fmt.Errorf("no Nifty expiries found in IMF — IMF reload required")
	}

	// Start at config week + any dynamic offset triggered by WS rejections
	weekIdx := p.cfg.GetExpiryWeek() + p.activeExpiryOffset
	if weekIdx < 0 {
		weekIdx = 0
	}

	idealCE := ComputeCEStrike(atm, p.cfg)
	idealPE := ComputePEStrike(atm, p.cfg)

	for i := weekIdx; i < len(expiries); i++ {
		exp := expiries[i]
		expiryStr := p.imf.GetExpiryCode(exp)
		expiryLabel := exp.Format("2006-01-02 (Mon)")

		// Find nearest available CE and PE strikes for this expiry.
		nearCE, ceOK := p.imf.FindNearestStrike(exp, idealCE, "CE")
		nearPE, peOK := p.imf.FindNearestStrike(exp, idealPE, "PE")

		if !ceOK || !peOK {
			// This expiry has no strikes in the IMF at all — move to next.
			if i == weekIdx {
				p.logger.Warn("Configured expiry has no strikes in IMF — trying next",
					"expiry", expiryLabel, "expiry_week", weekIdx)
			}
			continue
		}

		ceSym := BuildOptionSymbol(expiryStr, nearCE, "CE")
		peSym := BuildOptionSymbol(expiryStr, nearPE, "PE")

		if nearCE == idealCE && nearPE == idealPE {
			p.logger.Info("Symbols resolved (exact strike)",
				"expiry", expiryLabel, "ce", ceSym, "pe", peSym)
		} else {
			p.logger.Info("Symbols resolved (snapped to nearest available strike)",
				"expiry", expiryLabel,
				"ideal_ce", idealCE, "used_ce", nearCE,
				"ideal_pe", idealPE, "used_pe", nearPE,
				"ce", ceSym, "pe", peSym)
		}
		if i > weekIdx {
			p.logger.Warn("Fell back to a later expiry",
				"configured_week", weekIdx, "used_week", i, "expiry", expiryLabel)
		}
		return []string{ceSym, peSym}, nil
	}

	// Absolute fallback — no expiry found in IMF at all
	fallbackExp := expiries[weekIdx]
	expiryStr := p.imf.GetExpiryCode(fallbackExp)
	if expiryStr == "" {
		// Just in case we didn't have it mapped somehow
		expiryStr = fallbackExp.Format("06") + "MON" 
	}
	ceSym := BuildOptionSymbol(expiryStr, idealCE, "CE")
	peSym := BuildOptionSymbol(expiryStr, idealPE, "PE")
	p.logger.Error("No expiry with available strikes found in IMF — subscribing with computed symbols",
		"ce", ceSym, "pe", peSym)
	return []string{ceSym, peSym}, nil
}

// handleInvalidSymbols is triggered asynchronously when the Fyers WS client receives an "invalid_symbols" error.
func (p *Pipeline) handleInvalidSymbols(invalid []string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check these are actually our current target symbols
	// (just in case it's a delayed error message for an old subscription)
	isCurrent := false
	for _, inv := range invalid {
		for _, cur := range p.activeSymbols {
			if inv == cur {
				isCurrent = true
				break
			}
		}
	}

	if !isCurrent {
		p.logger.Debug("Received invalid_symbols error for non-active symbols", "symbols", invalid)
		return
	}

	p.logger.Warn("Exchange rejected the requested symbols — walking forward to next expiry",
		"rejected", invalid,
		"old_offset", p.activeExpiryOffset,
		"new_offset", p.activeExpiryOffset+1)

	// Increment offset and trigger a recalculation
	p.activeExpiryOffset++
	
	syms, err := p.calculateSymbols(p.activeATM)
	if err != nil {
		p.logger.Error("Failed to calculate fallback symbols after rejection", "err", err)
		return
	}

	p.activeSymbols = syms
	p.client.Subscribe(syms)
	p.evtLogger.Log("FALLBACK", p.activeATM, syms, fmt.Sprintf("WS Rejected previous expiry (offset %d)", p.activeExpiryOffset))
	
	// Ensure the new valid state makes it to Redis
	go func(atm int, currentSyms []string) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := p.store.SavePipelineState(ctx, atm, currentSyms); err != nil {
			p.logger.Error("Failed to save fallback pipeline state to Redis", "err", err)
		}
	}(p.activeATM, syms)
}
