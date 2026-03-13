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

	// Pre-allocate a large slice so `append` never blocks for reallocation.
	ticksLog := make([]marketdata.TickEvent, 0, 500000)
	doneChan := make(chan bool)

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

	// 3. Start WS Client
	wsClient := marketdata.NewFyersWSClient(wsToken, ringBuffer, logger)
	if err := wsClient.Connect(); err != nil {
		logger.Error("Failed to connect to Fyers WebSocket", "error", err)
		os.Exit(1)
	}

	// For testing: Subscribe to Nifty50 Index and ATM Strikes (23242 -> 23250)
	// Example expiry: March 17 2026 => 26317
	wsClient.Subscribe([]string{
		"NSE:NIFTY50-INDEX",
		"NSE:NIFTY2631723250CE",
		"NSE:NIFTY2631723250PE",
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
