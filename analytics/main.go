package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"time"
)

// ─── Data types ──────────────────────────────────────────────────────────────

func roundToATM(price float64) int {
	return int(math.Round(price/50.0)) * 50
}

type TickEvent struct {
	Symbol        string  `json:"Symbol"`
	LTP           float64 `json:"LTP"`
	ExchTimestamp int64   `json:"ExchTimestamp"`
	RecvTimestamp int64   `json:"RecvTimestamp"`
}

type PipelineEvent struct {
	Timestamp string   `json:"timestamp"` // "HH:MM:SS" IST
	Event     string   `json:"event"`
	ATM       int      `json:"atm"`
	Symbols   []string `json:"symbols"`
	Note      string   `json:"note"`
}

// ATMPhase represents one stable active period: from when an ATM became active
// until the next switch (or end of data).
type ATMPhase struct {
	ATM       int
	Symbols   []string
	StartTime time.Time
	EndTime   time.Time // zero = still active / use data end
	Event     string    // INIT or ACTIVE_SWITCH
}

// BucketStats holds per-15-min bucket metrics.
type BucketStats struct {
	TotalRecords      int64
	SymbolCounts      map[string]int64
	InterArrivalCount int64
	TotalInterArrival time.Duration
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// parseIST parses an "HH:MM:SS" timestamp using today's date in IST.
func parseIST(ts string, istLoc *time.Location) (time.Time, error) {
	now := time.Now().In(istLoc)
	full := fmt.Sprintf("%04d-%02d-%02d %s", now.Year(), now.Month(), now.Day(), ts)
	return time.ParseInLocation("2006-01-02 15:04:05", full, istLoc)
}

// ─── Main ────────────────────────────────────────────────────────────────────

func main() {
	chunkMinutes := 15.0

	istLoc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		fmt.Printf("Warning: Could not load Asia/Kolkata timezone, using local time: %v\n", err)
		istLoc = time.Local
	}

	// ── 1a. Quick first-pass of ticks.txt to find the data window start ──────
	// pipeline_events.txt is append-only; we use the tick data window to discard
	// events from previous runs before building phases.
	var dataStartTime, dataEndTime time.Time
	if f, err := os.Open("ticks.txt"); err == nil {
		fsc := bufio.NewScanner(f)
		fbuf := make([]byte, 0, 1*1024*1024)
		fsc.Buffer(fbuf, 4*1024*1024)
		for fsc.Scan() {
			line := fsc.Text()
			if line == "" {
				continue
			}
			var t TickEvent
			if json.Unmarshal([]byte(line), &t) == nil && t.RecvTimestamp > 0 {
				rt := time.Unix(0, t.RecvTimestamp)
				if dataStartTime.IsZero() || rt.Before(dataStartTime) {
					dataStartTime = rt
				}
				if rt.After(dataEndTime) {
					dataEndTime = rt
				}
			}
		}
		f.Close()
	}

	// ── 1b. Parse pipeline_events.txt — filter to current run only ────────────
	var pipelineEvents []PipelineEvent
	eventsFile, evErr := os.Open("pipeline_events.txt")
	if evErr == nil {
		sc := bufio.NewScanner(eventsFile)
		// Allow events up to 10 min before the first tick (covers the INIT that
		// fired before any CE/PE tick arrived) but reject everything older.
		runCutoff := dataStartTime.Add(-10 * time.Minute)
		for sc.Scan() {
			line := sc.Text()
			if line == "" {
				continue
			}
			var ev PipelineEvent
			if json.Unmarshal([]byte(line), &ev) != nil {
				continue
			}
			if !dataStartTime.IsZero() {
				t, err := parseIST(ev.Timestamp, istLoc)
				if err != nil || t.Before(runCutoff) {
					continue // belongs to a previous run
				}
			}
			pipelineEvents = append(pipelineEvents, ev)
		}
		eventsFile.Close()
	}

	// Build ordered ATM phases from INIT + ACTIVE_SWITCH events only.
	// These mark when the pipeline committed to a new active set of symbols.
	var phases []ATMPhase
	for _, ev := range pipelineEvents {
		if ev.Event != "INIT" && ev.Event != "ACTIVE_SWITCH" && ev.Event != "RESTORED" {
			continue
		}
		t, err := parseIST(ev.Timestamp, istLoc)
		if err != nil {
			continue
		}
		phases = append(phases, ATMPhase{
			ATM:       ev.ATM,
			Symbols:   ev.Symbols,
			StartTime: t,
			Event:     ev.Event,
		})
	}
	// Sort by start time
	sort.Slice(phases, func(i, j int) bool {
		return phases[i].StartTime.Before(phases[j].StartTime)
	})
	// Fill EndTime for each phase = next phase's StartTime
	for i := 0; i < len(phases)-1; i++ {
		phases[i].EndTime = phases[i+1].StartTime
	}

	// Count TIMER_EXTENDs and TIMER_STARTs per ATM for volatility summary
	timerExtendCount := make(map[int]int)
	timerStartCount := make(map[int]int)
	for _, ev := range pipelineEvents {
		if ev.Event == "TIMER_EXTEND" {
			timerExtendCount[ev.ATM]++
		}
		if ev.Event == "TIMER_START" {
			timerStartCount[ev.ATM]++
		}
	}

	// ── 2. Parse ticks.txt ───────────────────────────────────────────────────
	ticksFile, err := os.Open("ticks.txt")
	if err != nil {
		fmt.Printf("Error opening ticks.txt: %v\n", err)
		os.Exit(1)
	}
	defer ticksFile.Close()

	var totalRecords int64
	var skippedRecords int64
	symbolCounts := make(map[string]int64)
	lastRecvTime := make(map[string]time.Time)

	var totalInterArrivalTime time.Duration
	var minInterArrivalTime time.Duration = time.Hour
	var maxInterArrivalTime time.Duration = -1
	var interArrivalCount int64

	// dataStartTime / dataEndTime already declared above in the first-pass scan
	buckets := make(map[int64]*BucketStats)

	// Per-phase tick tracking: phase index -> per-second tick counts
	// We'll accumulate (recvNano, symbol) per phase and compute stats at the end.
	type tickSample struct {
		recvTime time.Time
		symbol   string
	}
	phaseTicks := make([][]tickSample, len(phases))
	for i := range phaseTicks {
		phaseTicks[i] = make([]tickSample, 0)
	}

	sc := bufio.NewScanner(ticksFile)
	buf := make([]byte, 0, 4*1024*1024)
	sc.Buffer(buf, 4*1024*1024)

	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		var tick TickEvent
		if err := json.Unmarshal([]byte(line), &tick); err != nil {
			skippedRecords++
			continue
		}
		if tick.RecvTimestamp == 0 {
			continue
		}

		totalRecords++
		symbolCounts[tick.Symbol]++
		recvTime := time.Unix(0, tick.RecvTimestamp)

		// Track overall window
		if dataStartTime.IsZero() || recvTime.Before(dataStartTime) {
			dataStartTime = recvTime
		}
		if recvTime.After(dataEndTime) {
			dataEndTime = recvTime
		}

		// Use exchange timestamp when available; fall back to recv time for depth ticks
		var tickTime time.Time
		if tick.ExchTimestamp > 0 {
			tickTime = time.Unix(tick.ExchTimestamp, 0)
		} else {
			tickTime = recvTime
		}

		// 15-min bucket
		bucketKey := tickTime.Truncate(time.Duration(chunkMinutes) * time.Minute).Unix()
		b, ok := buckets[bucketKey]
		if !ok {
			b = &BucketStats{SymbolCounts: make(map[string]int64)}
			buckets[bucketKey] = b
		}
		b.TotalRecords++
		b.SymbolCounts[tick.Symbol]++

		// Inter-arrival (per symbol, using recv clock)
		if lastTime, exists := lastRecvTime[tick.Symbol]; exists {
			ia := recvTime.Sub(lastTime)
			if ia > 0 {
				totalInterArrivalTime += ia
				interArrivalCount++
				b.TotalInterArrival += ia
				b.InterArrivalCount++
				if ia < minInterArrivalTime {
					minInterArrivalTime = ia
				}
				if ia > maxInterArrivalTime {
					maxInterArrivalTime = ia
				}
			}
		}
		lastRecvTime[tick.Symbol] = recvTime

		// Assign tick to its pipeline phase
		for i, ph := range phases {
			inPhase := !recvTime.Before(ph.StartTime) &&
				(ph.EndTime.IsZero() || recvTime.Before(ph.EndTime))
			if inPhase {
				phaseTicks[i] = append(phaseTicks[i], tickSample{recvTime: recvTime, symbol: tick.Symbol})
				break
			}
		}
	}

	// ── 3. Print report ──────────────────────────────────────────────────────

	fmt.Println("==================================================")
	fmt.Println("     Tick Data Analytics Report")
	fmt.Println("==================================================")
	fmt.Printf("Total CE/PE Ticks Recorded : %d\n", totalRecords)
	if skippedRecords > 0 {
		fmt.Printf("Skipped (parse errors)     : %d\n", skippedRecords)
	}
	if !dataStartTime.IsZero() {
		fmt.Printf("Data Window (recv clock)   : %s → %s IST  (%.0fs)\n",
			dataStartTime.In(istLoc).Format("15:04:05"),
			dataEndTime.In(istLoc).Format("15:04:05"),
			dataEndTime.Sub(dataStartTime).Seconds())
	}

	// Per-symbol breakdown
	fmt.Println("\n--- Symbol Breakdown ---")
	for sym, count := range symbolCounts {
		fmt.Printf("  %-35s : %d ticks\n", sym, count)
	}

	// Global frequency
	fmt.Println("\n--- Global Frequency / Throughput ---")
	if interArrivalCount > 0 {
		avg := time.Duration(int64(totalInterArrivalTime) / interArrivalCount)
		tps := 0.0
		if avg.Seconds() > 0 {
			tps = 1.0 / avg.Seconds()
		}
		fmt.Printf("  Avg inter-arrival : %v  (%.2f ticks/sec per symbol)\n", avg, tps)
		if minInterArrivalTime < time.Hour {
			fmt.Printf("  Min inter-arrival : %v\n", minInterArrivalTime)
			fmt.Printf("  Max inter-arrival : %v\n", maxInterArrivalTime)
		}
	} else {
		fmt.Println("  (not enough data)")
	}

	// ── 4. Pipeline Events section ───────────────────────────────────────────
	if len(pipelineEvents) > 0 {
		fmt.Println("\n==================================================")
		fmt.Println("     Pipeline Events Summary")
		fmt.Println("==================================================")

		// Count events by type
		eventTypeCounts := make(map[string]int)
		for _, ev := range pipelineEvents {
			eventTypeCounts[ev.Event]++
		}
		fmt.Printf("Total pipeline events : %d\n", len(pipelineEvents))
		for _, evType := range []string{"INIT", "ACTIVE_SWITCH", "TIMER_START", "TIMER_EXTEND", "RESTORED", "FALLBACK", "POST_CYCLE_MISMATCH"} {
			if c, ok := eventTypeCounts[evType]; ok {
				fmt.Printf("  %-22s : %d\n", evType, c)
			}
		}

		// ATM phase timeline
		if len(phases) > 0 {
			fmt.Println("\n--- ATM Phase Timeline ---")
			for i, ph := range phases {
				var duration string
				if !ph.EndTime.IsZero() {
					d := ph.EndTime.Sub(ph.StartTime)
					duration = fmt.Sprintf("%.0fs", d.Seconds())
				} else {
					if !dataEndTime.IsZero() {
						d := dataEndTime.Sub(ph.StartTime)
						duration = fmt.Sprintf("~%.0fs (until data end)", d.Seconds())
					} else {
						duration = "ongoing"
					}
				}

				extendCount := timerExtendCount[ph.ATM]
				extendNote := ""
				if extendCount > 0 {
					extendNote = fmt.Sprintf("  [%d extensions — volatile entry]", extendCount)
				}

				phaseTickCount := len(phaseTicks[i])
				tpsNote := ""
				if phaseTickCount > 0 && !ph.StartTime.IsZero() {
					endT := ph.EndTime
					if endT.IsZero() {
						endT = dataEndTime
					}
					phDuration := endT.Sub(ph.StartTime).Seconds()
					if phDuration > 0 {
						tpsNote = fmt.Sprintf("  → %.2f ticks/sec", float64(phaseTickCount)/phDuration)
					}
				}

				fmt.Printf("  [%s IST] ATM %-6d  via %-13s  duration: %-18s  ticks: %-5d%s%s\n",
					ph.StartTime.In(istLoc).Format("15:04:05"),
					ph.ATM,
					ph.Event,
					duration,
					phaseTickCount,
					tpsNote,
					extendNote,
				)
			}
		}

		// Per-phase throughput detail
		if len(phases) > 0 {
			fmt.Println("\n--- Per-Phase Tick Throughput ---")
			for i, ph := range phases {
				ticks := phaseTicks[i]
				if len(ticks) == 0 {
					continue
				}

				endT := ph.EndTime
				if endT.IsZero() {
					endT = dataEndTime
				}
				phDurSec := endT.Sub(ph.StartTime).Seconds()

				// bucket ticks into 1-second windows to get min/avg/max ticks/sec
				secBuckets := make(map[int64]int)
				for _, ts := range ticks {
					key := ts.recvTime.Unix()
					secBuckets[key]++
				}

				var minTPS, maxTPS, sumTPS float64
				minTPS = math.MaxFloat64
				for _, cnt := range secBuckets {
					f := float64(cnt)
					sumTPS += f
					if f < minTPS {
						minTPS = f
					}
					if f > maxTPS {
						maxTPS = f
					}
				}
				avgTPS := float64(len(ticks)) / phDurSec

				fmt.Printf("  ATM %-6d [%s IST]  %d ticks / %.0fs\n",
					ph.ATM,
					ph.StartTime.In(istLoc).Format("15:04:05"),
					len(ticks),
					phDurSec,
				)
				fmt.Printf("    avg: %.2f  min: %.0f  max: %.0f  ticks/sec\n",
					avgTPS, minTPS, maxTPS)
			}
		}
	}

	// ── 5. 15-Minute Bucket Breakdown ────────────────────────────────────────
	fmt.Println("\n==================================================")
	fmt.Printf("     Time-Interval Breakdown (%.0f-Minute Buckets)\n", chunkMinutes)
	fmt.Println("==================================================")

	var keys []int64
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	for _, k := range keys {
		b := buckets[k]
		windowStart := time.Unix(k, 0).In(istLoc)
		windowEnd := windowStart.Add(time.Duration(chunkMinutes) * time.Minute)
		fmt.Printf("\n[%s - %s IST]  total: %d ticks\n",
			windowStart.Format("15:04"), windowEnd.Format("15:04"), b.TotalRecords)

		// Per-symbol in bucket
		for sym, cnt := range b.SymbolCounts {
			fmt.Printf("  %-35s : %d\n", sym, cnt)
		}

		if b.InterArrivalCount > 0 {
			avgIA := time.Duration(int64(b.TotalInterArrival) / b.InterArrivalCount)
			tps := 0.0
			if avgIA.Seconds() > 0 {
				tps = 1.0 / avgIA.Seconds()
			}
			fmt.Printf("  Throughput : %.2f ticks/sec per symbol  (avg inter-arrival: %v)\n", tps, avgIA)
		}
	}

	fmt.Println("\n==================================================")
}
