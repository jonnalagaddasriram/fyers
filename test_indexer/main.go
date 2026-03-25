package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"fyers-trading/config"

	fyersws "github.com/FyersDev/fyers-go-sdk/websocket"
)

func main() {
	appCfg, err := config.LoadAppConfig()
	if err != nil {
		fmt.Printf("Startup Error Loading App Config: %v\n", err)
		os.Exit(1)
	}

	appID := appCfg.FyersAppID
	token := appCfg.FyersAccessToken
	accessToken := fmt.Sprintf("%s:%s", appID, token)

	indexSymbols := []string{"NSE:NIFTY50-INDEX"}
	optionSymbols := []string{"NSE:NIFTY2632422700PE", "NSE:NIFTY2632422700CE"}

	// dataSocket needs to be accessible in onConnect, so declare it here
	var indexSocket *fyersws.FyersDataSocket
	var optionSocket *fyersws.FyersDataSocket

	onIndexConnect := func() {
		fmt.Printf("[Index Socket] Connected! Subscribing Index to SymbolUpdate...\n")
		indexSocket.Subscribe(indexSymbols, "SymbolUpdate")
	}

	onOptionConnect := func() {
		fmt.Printf("[Option Socket] Connected! Subscribing Options to DepthUpdate...\n")
		optionSocket.Subscribe(optionSymbols, "DepthUpdate")
	}

	type StoredTick struct {
		Msg      fyersws.DataResponse
		RecvTime time.Time
	}

	var indexMu sync.Mutex
	var indexTicks []StoredTick

	var optionMu sync.Mutex
	var optionTicks []StoredTick

	var firstMsgTime, lastMsgTime time.Time

	onIndexMessage := func(message fyersws.DataResponse) {
		now := time.Now()
		indexMu.Lock()
		indexTicks = append(indexTicks, StoredTick{Msg: message, RecvTime: now})
		indexMu.Unlock()
	}

	onOptionMessage := func(message fyersws.DataResponse) {
		now := time.Now()
		optionMu.Lock()
		// Only capture start/end window based strictly on options
		if firstMsgTime.IsZero() {
			firstMsgTime = now
		}
		lastMsgTime = now
		optionTicks = append(optionTicks, StoredTick{Msg: message, RecvTime: now})
		optionMu.Unlock()
	}

	onError := func(message fyersws.DataError) {
		fmt.Printf("Error: %v\n", message)
	}

	onClose := func(message fyersws.DataClose) {
		fmt.Printf("Connection closed: %v\n", message)
	}

	indexSocket = fyersws.NewFyersDataSocket(
		accessToken,
		"",
		false, // full mode
		false,
		true,
		50,
		onIndexConnect,
		onClose,
		onError,
		onIndexMessage,
	)

	optionSocket = fyersws.NewFyersDataSocket(
		accessToken,
		"",
		false,
		false,
		true,
		50,
		onOptionConnect,
		onClose,
		onError,
		onOptionMessage,
	)

	fmt.Println("Attempting to connect Index Socket & Option Socket... Press Ctrl+C to close and view analytics.")
	
	go func() {
		if err := indexSocket.Connect(); err != nil {
			fmt.Printf("failed to connect to Index Socket: %v\n", err)
		}
	}()

	go func() {
		// Small delay to ensure they don't hit the Fyers auth endpoint at the exact same millisecond
		time.Sleep(500 * time.Millisecond)
		if err := optionSocket.Connect(); err != nil {
			fmt.Printf("failed to connect to Option Socket: %v\n", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("\nReceived interrupt signal, closing connections...")
	indexSocket.CloseConnection()
	optionSocket.CloseConnection()
	fmt.Println("Both Data Socket connections closed")

	// ==========================================
	// Analytics processing on close
	// ==========================================
	
	optionMu.Lock()
	totalTicks := len(optionTicks)
	counts := make(map[string]int)
	
	t1 := firstMsgTime
	t2 := lastMsgTime

	var batchSizes []int
	var timeGaps []time.Duration
	currentBatchSize := 0
	var lastBatchTime time.Time

	for _, stored := range optionTicks {
		msg := stored.Msg
		t := stored.RecvTime
		
		if sym, ok := msg["symbol"].(string); ok {
			counts[sym]++
		}
		
		if lastBatchTime.IsZero() {
			lastBatchTime = t
			currentBatchSize = 1
		} else {
			// If ticks arrive within 10ms of each other, consider them the same batch
			if t.Sub(lastBatchTime) < 10*time.Millisecond {
				currentBatchSize++
			} else {
				batchSizes = append(batchSizes, currentBatchSize)
				timeGaps = append(timeGaps, t.Sub(lastBatchTime))
				currentBatchSize = 1
				lastBatchTime = t
			}
		}
	}
	if currentBatchSize > 0 {
		batchSizes = append(batchSizes, currentBatchSize)
	}
	optionMu.Unlock()

	fmt.Println("\n==================================================")
	fmt.Println(" Final Options Tick Analytics (test_indexer)")
	fmt.Println("==================================================")
	fmt.Printf("Total Option Records Processed : %d\n", totalTicks)
	
	if !t1.IsZero() && !t2.IsZero() {
		loc, _ := time.LoadLocation("Asia/Kolkata")
		if loc == nil {
			loc = time.FixedZone("IST", 5*3600+30*60)
		}
		fmt.Printf("Options Exchange Time Window   : %s to %s (IST)\n", t1.In(loc).Format("15:04:05"), t2.In(loc).Format("15:04:05"))
	}
	fmt.Println()
	
	fmt.Println("--- Breakdown by Symbol (Options Only) ---")

	for sym, count := range counts {
		var percentage float64
		if totalTicks > 0 {
			percentage = (float64(count) / float64(totalTicks)) * 100.0
		}
		fmt.Printf("  %s : %d ticks (%.1f%%)\n", sym, count, percentage)
	}
	
	fmt.Println("\n--- Batch & Timing Flow ---")
	if len(batchSizes) > 0 {
		avgBatch := 0
		for _, b := range batchSizes {
			avgBatch += b
		}
		fmt.Printf("  Total Network Batches Received : %d\n", len(batchSizes))
		fmt.Printf("  Average Ticks per Batch        : %.2f ticks\n", float64(avgBatch)/float64(len(batchSizes)))
	}
	if len(timeGaps) > 0 {
		var totalGap time.Duration
		for _, g := range timeGaps {
			totalGap += g
		}
		avgGap := totalGap / time.Duration(len(timeGaps))
		fmt.Printf("  Average Wait Time between Batches: %v\n", avgGap)
	}
	
	fmt.Println("==================================================")
}
