package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fyers-trading/config"
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

	// 2. Initialize Market Data Infrastructure
	ringBuffer := marketdata.NewRingBuffer(appCfg.RingBufferSize)

	// Track latest prices
	tickStore := marketdata.NewTickStore(logger)
	tickStore.StartReader(ringBuffer.NewReader())

	// Watch internal latencies (printing every 5 seconds)
	latencyMonitor := marketdata.NewLatencyMonitor(logger, 5*time.Second)
	latencyMonitor.StartReader(ringBuffer.NewReader())

	// completely eliminate Terminal I/O lag from the hot path.
	testReader := ringBuffer.NewReader()
	pipelineReader := ringBuffer.NewReader()

	ticksLog := make([]marketdata.TickEvent, 0, 500000)
	doneChan := make(chan bool)

	// We declare wsClient here so we can inject it into the Pipeline
	wsClient := marketdata.NewFyersWSClient(wsToken, ringBuffer, logger)
	pipeline := symbols.NewPipeline(logger, wsClient, tradingCfg, imfManager)

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
	go func() {
		for {
			tick := testReader.Next()
			if tick == nil {
				close(doneChan)
				return
			}
			
			// Store a struct copy so we don't accidentally maintain pointers to pooled objects
			ticksLog = append(ticksLog, *tick)
		}
	}()

	// 3. Start WS Client connecting
	if err := wsClient.Connect(); err != nil {
		logger.Error("Failed to connect to Fyers WebSocket", "error", err)
		os.Exit(1)
	}

	// Subscribe only to the index initially. The Pipeline takes over the options!
	wsClient.Subscribe([]string{
		"NSE:NIFTY50-INDEX",
	})

	// Wait for termination
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Received shutdown signal, terminating...")
	wsClient.Close()
	// Close ring buffer to signal all reader goroutines to exit gracefully
	ringBuffer.Close()

	<-doneChan

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
