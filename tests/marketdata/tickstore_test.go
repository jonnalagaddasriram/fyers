package marketdata_test

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"fyers-trading/marketdata"
)

func TestTickStore_StoreAndGet(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := marketdata.NewTickStore(logger)

	tick1 := marketdata.GetTickEvent()
	tick1.Symbol = "NSE:SBIN-EQ"
	tick1.LTP = 500.0
	tick1.Volume = 1000
	tick1.RecvTimestamp = time.Now().UnixNano()

	store.Store(tick1)

	retrieved, found := store.Get("NSE:SBIN-EQ")
	if !found {
		t.Fatal("Expected to find NSE:SBIN-EQ in store")
	}

	if retrieved.LTP != 500.0 {
		t.Errorf("Expected LTP 500.0, got %f", retrieved.LTP)
	}
	if retrieved.Volume != 1000 {
		t.Errorf("Expected Volume 1000, got %d", retrieved.Volume)
	}

	// Make sure a missing symbol returns false
	_, foundMissing := store.Get("NSE:NIFTY")
	if foundMissing {
		t.Fatal("Expected not to find missing symbol")
	}
}

func TestTickStore_Overwrite(t *testing.T) {
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := marketdata.NewTickStore(logger)

    tick1 := marketdata.GetTickEvent()
    tick1.Symbol = "NIFTY"
    tick1.LTP = 22000.0
    store.Store(tick1)
    
    tick2 := marketdata.GetTickEvent()
    tick2.Symbol = "NIFTY"
    tick2.LTP = 22100.5
    store.Store(tick2)

    ret, _ := store.Get("NIFTY")
    if ret.LTP != 22100.5 {
        t.Errorf("Expected store to overwrite to latest LTP 22100.5, got %f", ret.LTP)
    }
}

func TestTickStore_IndependentCopies(t *testing.T) {
    // Ensuring that modifications to the pooled tick after Store don't affect TickStore.
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := marketdata.NewTickStore(logger)

    tick := marketdata.GetTickEvent()
    tick.Symbol = "RELIANCE"
    tick.LTP = 2800.0
    store.Store(tick)
    
    // Modify the pointer
    tick.LTP = 9999.0
    
    ret, _ := store.Get("RELIANCE")
    if ret.LTP != 2800.0 {
        t.Errorf("TickStore should store value copies, but it was affected by pointer modification. Expected 2800.0, got %f", ret.LTP)
    }
}
