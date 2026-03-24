package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"fyers-trading/config"
	"fyers-trading/infra"
	"fyers-trading/marketdata"
	"fyers-trading/symbols"

	fyersws "github.com/FyersDev/fyers-go-sdk/websocket"
)

func setLogger(level string, format string) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	if level == "debug" {
		opts.Level = slog.LevelDebug
	}

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func main() {
	// 1. Load configuration
	appCfg, err := config.LoadAppConfig()
	if err != nil {
		fmt.Printf("Startup Error Loading App Config: %v\n", err)
		os.Exit(1)
	}

	logger := setLogger(appCfg.LogLevel, appCfg.LogFormat)
	logger.Info("Booting Fyers Trading System", "env", appCfg.Env, "dry_run", appCfg.DryRun)

	// Since we are just testing market data connection,
	// Make sure FYERS_APP_ID and FYERS_ACCESS_TOKEN exist.
	if appCfg.FyersAppID == "" || appCfg.FyersAccessToken == "" {
		logger.Error("FYERS_APP_ID or FYERS_ACCESS_TOKEN not set in .env")
		os.Exit(1)
	}

	// Format specific to Fyers WS: "app_id:access_token"
	wsToken := fmt.Sprintf("%s:%s", appCfg.FyersAppID, appCfg.FyersAccessToken)

	// Verify Access Token via Profile API
	logger.Info("Verifying Fyers access token via Profile API...")
	if err := marketdata.VerifyToken(appCfg.FyersAppID, appCfg.FyersAccessToken); err != nil {
		logger.Error("Access token verification failed", "error", err)
		os.Exit(1)
	}
	logger.Info("Access token verified successfully.")

	// 1.5 Load Trading Config and start Hot-Reloader
	tradingCfg, err := config.LoadTradingConfig("trading.json")
	if err != nil {
		logger.Error("Failed to load trading.json", "error", err)
		os.Exit(1)
	}

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := tradingCfg.Reload("trading.json"); err != nil {
				logger.Warn("Failed to hot-reload trading.json", "error", err)
			}
		}
	}()

	// 1.6 Initialize IMF Manager (Master File)
	imfManager := symbols.NewIMFManager(logger, "data")
	if err := imfManager.Initialize(); err != nil {
		logger.Error("Failed to initialize Fyers Instrument Master File", "error", err)
		os.Exit(1)
	}

	// 1.7 Connect Redis (graceful: falls back to NoopStore if unavailable)
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	store, usingRedis := infra.ConnectStore(ctx, appCfg.RedisURL, logger)
	defer store.Close()
	if !usingRedis {
		logger.Warn("Running without Redis — crash recovery disabled")
	}

	// 2. Initialize Market Data Infrastructure
	// We use two completely independent lock-free pure SPMC RingBuffers!
	optionsRingBuffer := marketdata.NewRingBuffer(appCfg.RingBufferSize)
	indexRingBuffer := marketdata.NewRingBuffer(1024)

	// Track latest prices
	tickStore := marketdata.NewTickStore(logger)
	tickStore.StartReader(optionsRingBuffer.NewReader())
	tickStore.StartReader(indexRingBuffer.NewReader())

	// Watch internal latencies (printing every 5 seconds)
	latencyMonitor := marketdata.NewLatencyMonitor(logger, 5*time.Second)
	latencyMonitor.StartReader(optionsRingBuffer.NewReader())
	latencyMonitor.StartReader(indexRingBuffer.NewReader())

	// completely eliminate Terminal I/O lag from the hot path.
	testOptionReader := optionsRingBuffer.NewReader()
	testIndexReader := indexRingBuffer.NewReader()
	pipelineReader := indexRingBuffer.NewReader() // Pipeline strictly reads Index to calculate ATM strikes

	var ticksLogMu sync.Mutex
	ticksLog := make([]marketdata.TickEvent, 0, 500000)
	doneChan := make(chan bool)

	// We declare wsClient here so we can inject it into the Pipeline
	wsClient := marketdata.NewFyersWSClient(wsToken, optionsRingBuffer, logger)
	pipeline := symbols.NewPipeline(logger, wsClient, tradingCfg, imfManager, store)

	// Thread 2: The Pipeline (Takes one read cursor, computes symbol rules dynamically)
	go func() {
		initialSetupDone := false
		for {
			tick := pipelineReader.Next()
			if tick == nil {
				return
			}
			
			// Process Nifty Ticks through the Pipeline
			if tick.Symbol == "NSE:NIFTY50-INDEX" {
				if !initialSetupDone && tick.LTP > 0 {
					if err := pipeline.SetInitialState(tick.LTP); err != nil {
						logger.Error("Failed to initialize pipeline state", "error", err)
					} else {
						initialSetupDone = true
					}
				} else if initialSetupDone {
					pipeline.ProcessNiftyTick(tick.LTP)
				}
			}
		}
	}()

	// Thread 3a: Options Logger
	go func() {
		for {
			tick := testOptionReader.Next()
			if tick == nil {
				close(doneChan)
				return
			}
			
			// Thread-safe append to unified file log
			ticksLogMu.Lock()
			ticksLog = append(ticksLog, *tick)
			ticksLogMu.Unlock()
		}
	}()

	// Thread 3b: Index Logger
	go func() {
		for {
			tick := testIndexReader.Next()
			if tick == nil {
				return
			}
			
			ticksLogMu.Lock()
			ticksLog = append(ticksLog, *tick)
			ticksLogMu.Unlock()
		}
	}()



	// 3. Start the Hybrid WebSocket Connections
	// 3a. Custom TBT Protobuf Socket (For High-Frequency Options Depth)
	if err := wsClient.Connect(); err != nil {
		logger.Error("Failed to connect to Fyers TBT WebSocket", "error", err)
		os.Exit(1)
	}

	// 3b. Official Fyers SDK Socket (For the Nifty Index ONLY, as it lacks depth)
	indexSymbols := []string{"NSE:NIFTY50-INDEX"}
	
	onIndexConnect := func() {
		logger.Info("Index SDK Socket Connected. Subscribing to Nifty50 Index.")
		// We use an external variable scope so we can access indexSocket here, but we can't easily do it exactly 
		// inside the func literal unless indexSocket is declared beforehand.
	}

	onIndexMessage := func(message fyersws.DataResponse) {
		tick := marketdata.GetTickEvent()
		tick.RecvTimestamp = time.Now().UnixNano()

		if sym, ok := message["symbol"].(string); ok {
			tick.Symbol = sym
		} else {
			marketdata.PutTickEvent(tick)
			return
		}

		if ltpVal, ok := message["ltp"]; ok {
			switch v := ltpVal.(type) {
			case float64:
				tick.LTP = v
			case int:
				tick.LTP = float64(v)
			case int32:
				tick.LTP = float64(v)
			case int64:
				tick.LTP = float64(v)
			case fyersws.FloatSDK:
				tick.LTP = float64(v)
			}
		}

		if exchTimeVal, ok := message["exch_feed_time"]; ok {
			switch v := exchTimeVal.(type) {
			case int64:
				tick.ExchTimestamp = v
			case int32:
				tick.ExchTimestamp = int64(v)
			case int:
				tick.ExchTimestamp = int64(v)
			case float64:
				tick.ExchTimestamp = int64(v)
			case fyersws.FloatSDK:
				tick.ExchTimestamp = int64(v)
			}
		}
		
		if tick.LTP > 0 {
			indexRingBuffer.Write(tick)
		} else {
			marketdata.PutTickEvent(tick)
		}
	}

	onError := func(message fyersws.DataError) {
		logger.Error("Index SDK Socket Error", "error", message)
	}

	onClose := func(message fyersws.DataClose) {
		logger.Warn("Index SDK Socket Closed", "msg", message)
	}

	indexSocket := fyersws.NewFyersDataSocket(
		wsToken,
		"",    // log path
		false, // lite mode
		false, // Write to file
		true,  // auto reconnect
		1,     // 1ms queue processing interval (ultra-low latency)
		onIndexConnect,
		onClose,
		onError,
		onIndexMessage,
	)

	// Since we couldn't properly inject indexSocket into the onIndexConnect literal (because 
	// it's a value-copy closure at config-time in the SDK), we simply subscribe post-connection.
	go func() {
		logger.Info("Attempting SDK Index WebSocket connection...")
		if err := indexSocket.Connect(); err != nil {
			logger.Error("Failed to connect to Index SDK Socket", "error", err)
		}
	}()

	// Give it a tiny delay to connect before manually calling Subscribe, 
	// since the SDK Connect() call blocks in its own loop in the background!
	go func() {
		time.Sleep(2 * time.Second)
		indexSocket.Subscribe(indexSymbols, "SymbolUpdate")
		logger.Info("Subscribed Index to SymbolUpdate via SDK", "symbols", indexSymbols)
	}()

	// Wait for termination (OS signal or Market Close)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Info("Received OS shutdown signal, terminating...")

	// Cancel context — signals the Replicator to flush one final snapshot
	cancelCtx()
	wsClient.Close()
	indexSocket.CloseConnection()
	// Close ring buffer to signal all reader goroutines to exit gracefully
	optionsRingBuffer.Close()
	indexRingBuffer.Close()

	<-doneChan

	if usingRedis {
		logger.Info("Exporting Redis state to local file...")
		// Use a fresh context since the parent ctx is already canceled during shutdown
		exportCtx, cancelExport := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelExport()

		data, err := infra.ExportStateJSON(exportCtx, store)
		if err == nil {
			err = os.MkdirAll("logs", 0755)
			if err == nil {
				err = os.WriteFile("logs/redis_export.json", data, 0644)
				if err == nil {
					logger.Info("Successfully exported Redis state to logs/redis_export.json")
				} else {
					logger.Error("Failed to write redis_export.json", "error", err)
				}
			} else {
				logger.Error("Failed to create logs directory", "error", err)
			}
		} else {
			logger.Error("Failed gracefully dumping Redis state", "error", err)
		}
	}

	logger.Info("Writing captured ticks to ticks.txt...", "total_ticks", len(ticksLog))
	file, err := os.Create("ticks.txt")
	if err == nil {
		for _, t := range ticksLog {
			b, _ := json.Marshal(t)
			file.Write(b)
			file.WriteString("\n")
		}
		file.Close()
		logger.Info("Successfully saved ticks to ticks.txt")
	} else {
		logger.Error("Failed to write to file", "error", err)
	}

	logger.Info("Goodbye!")
}
