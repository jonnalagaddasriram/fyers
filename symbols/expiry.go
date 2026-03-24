package symbols

import (
	"fmt"
	"time"

	"fyers-trading/config"
)

// GetTradingExpiry uses the pre-parsed IMF valid dates to return the correct real expiry date.
// expiryWeek 0 = current nearest expiry, 1 = next valid expiry, etc.
func GetTradingExpiry(imf *IMFManager, cfg *config.TradingConfig) (time.Time, error) {
	expiries := imf.GetNiftyExpiries()
	if len(expiries) == 0 {
		return time.Time{}, fmt.Errorf("no valid expiries found in IMF data")
	}

	weekIdx := cfg.GetExpiryWeek()
	if weekIdx < 0 || weekIdx >= len(expiries) {
		// Fallback to the furthest known expiry if asked for something out of bounds (highly unlikely)
		return expiries[len(expiries)-1], fmt.Errorf("requested expiry week %d out of bounds (max %d)", weekIdx, len(expiries)-1)
	}

	return expiries[weekIdx], nil
}

