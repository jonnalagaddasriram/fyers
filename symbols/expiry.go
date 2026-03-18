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

// FormatExpiryForSymbol formats a date to Fyers symbol convention
// YYMDD (YY=2-digit year, M=1-9,O,N,D, DD=2-digit day). Example: "26317"
func FormatExpiryForSymbol(date time.Time) string {
	year := date.Year() % 100
	month := date.Month()
	day := date.Day()

	monthChar := ""
	if month <= 9 {
		monthChar = fmt.Sprintf("%d", month)
	} else if month == 10 {
		monthChar = "O"
	} else if month == 11 {
		monthChar = "N"
	} else if month == 12 {
		monthChar = "D"
	}

	return fmt.Sprintf("%02d%s%02d", year, monthChar, day)
}
