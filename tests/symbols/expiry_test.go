package symbols_test

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"fyers-trading/config"
	"fyers-trading/symbols"
)

func TestGetTradingExpiry(t *testing.T) {
	// Skip if NO_NETWORK is set
	if os.Getenv("NO_NETWORK") == "true" {
		t.Skip("Skipping IMF network test")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	imf := symbols.NewIMFManager(logger, "../../data")

	// Since cacheDir is mapped to "data", ensure we can create it
	err := imf.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize IMF Manager: %v", err)
	}

	cfg := &config.TradingConfig{}
	
	// Test current week
	cfg.ExpiryWeek = 0
	expiry1, err := symbols.GetTradingExpiry(imf, cfg)
	if err != nil {
		t.Errorf("Failed to get expiry 0: %v", err)
	}
	
	// Test next week
	cfg.ExpiryWeek = 1
	expiry2, err := symbols.GetTradingExpiry(imf, cfg)
	if err != nil {
		t.Errorf("Failed to get expiry 1: %v", err)
	}

	if !expiry1.Before(expiry2) {
		t.Errorf("Expected first expiry %v to be before second expiry %v", expiry1, expiry2)
	}
}

func TestFormatExpiryForSymbol(t *testing.T) {
	date1 := time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC)
	if got := symbols.FormatExpiryForSymbol(date1); got != "26317" {
		t.Errorf("Expected 26317, got %s", got)
	}

	date2 := time.Date(2026, time.October, 27, 0, 0, 0, 0, time.UTC)
	if got := symbols.FormatExpiryForSymbol(date2); got != "26O27" {
		t.Errorf("Expected 26O27, got %s", got)
	}
}
