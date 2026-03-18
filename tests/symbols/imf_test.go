package symbols_test

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"fyers-trading/symbols"
)

func TestIMFManager(t *testing.T) {
	// Note: We skip the live download test on generic CI to save time/bandwidth
	// Uncomment the check locally if you want to verify the download explicitly.

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

	dates := imf.GetNiftyExpiries()
	if len(dates) == 0 {
		t.Fatalf("Expected to find valid Nifty expiries, got 0")
	}

	t.Logf("--- NEXT UPCOMING NIFTY EXPIRIES ---")
	for i := 0; i < len(dates) && i < 2; i++ {
		t.Logf("Expiry %d: %s", i+1, dates[i].Format("2006-01-02"))
	}
	t.Logf("--------------------------------------")

	// Verify dates are sorted
	for i := 1; i < len(dates); i++ {
		if dates[i-1].After(dates[i]) {
			t.Errorf("Dates are not sorted: %v is after %v", dates[i-1], dates[i])
		}
	}

	// Make sure we have dates in the future (or very recent past if testing after hours)
	// We check against yesterday to avoid timezone edge cases.
	yesterday := time.Now().AddDate(0, 0, -1)
	if len(dates) > 0 && dates[0].Before(yesterday) {
		t.Errorf("The nearest expiry %v appears to be too old", dates[0])
	}
}
