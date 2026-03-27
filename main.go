package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fyers-trading/config"
	"fyers-trading/infra"
	"fyers-trading/marketdata"
	"fyers-trading/symbols"
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
	ringBuffer := marketdata.NewRingBuffer(8192)

	// Track latest prices
	tickStore := marketdata.NewTickStore(logger)
	tickStore.StartReader(ringBuffer.NewReader())

	// Watch internal latencies (printing every 5 seconds)
	latencyMonitor := marketdata.NewLatencyMonitor(logger, 5*time.Second)
	latencyMonitor.StartReader(ringBuffer.NewReader())

	// completely eliminate Terminal I/O lag from the hot path.
	testReader := ringBuffer.NewReader()
	pipelineReader := ringBuffer.NewReader()

	// optionTicksLog captures only CE/PE option ticks (not the index) for post-run analytics.
	optionTicksLog := make([]marketdata.TickEvent, 0, 500000)
	doneChan := make(chan bool)

	// We declare wsClient here so we can inject it into the Pipeline
	wsClient := marketdata.NewFyersWSClient(wsToken, ringBuffer, logger)
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

	// Thread 3: The Logger / Final Memory array (Takes a second read cursor)
	// Only records CE/PE option ticks — index ticks are excluded from disk analysis.
	go func() {
		for {
			tick := testReader.Next()
			if tick == nil {
				close(doneChan)
				return
			}

			// We intentionally record the Index here too so the analytics tool can calculate relative throughput.

			// Store a struct copy so we don't accidentally maintain pointers to pooled objects
			optionTicksLog = append(optionTicksLog, *tick)
		}
	}()



	// 3. Start WS Client connecting
	if err := wsClient.Connect(); err != nil {
		logger.Error("Failed to connect to Fyers WebSocket", "error", err)
		os.Exit(1)
	}

	// Subscribe to the index via SymbolUpdate (lightweight LTP feed — separate from CE/PE depth).
	wsClient.SubscribeIndex([]string{
		"NSE:NIFTY50-INDEX",
	})

	// Wait for termination (OS signal or Market Close)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Info("Received OS shutdown signal, terminating...")

	// Cancel context — signals the Replicator to flush one final snapshot
	cancelCtx()
	wsClient.Close()
	// Close ring buffer to signal all reader goroutines to exit gracefully
	ringBuffer.Close()

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

	logger.Info("Writing captured CE/PE option ticks to ticks.txt...", "total_option_ticks", len(optionTicksLog))
	file, err := os.Create("ticks.txt")
	if err == nil {
		for _, t := range optionTicksLog {
			b, _ := json.Marshal(t)
			file.Write(b)
			file.WriteString("\n")
		}
		file.Close()
		logger.Info("Successfully saved CE/PE option ticks to ticks.txt")
	} else {
		logger.Error("Failed to write to file", "error", err)
	}

	logger.Info("Goodbye!")
}
