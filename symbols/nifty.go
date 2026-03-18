package symbols

import (
	"fmt"
	"math"

	"fyers-trading/config"
)

// NiftyIndexSymbol returns the standard Fyers symbol for Nifty 50.
func NiftyIndexSymbol() string {
	return "NSE:NIFTY50-INDEX"
}

// BuildOptionSymbol constructs the Fyers option symbol format (e.g., NSE:NIFTY2631723250CE)
func BuildOptionSymbol(expiryStr string, strike int, optType string) string {
	return fmt.Sprintf("NSE:NIFTY%s%d%s", expiryStr, strike, optType)
}

// RoundToStrike rounds the given price to the nearest multiple of step size.
func RoundToStrike(price float64, step int) int {
	return int(math.Round(price/float64(step))) * step
}

// ComputeATMStrike returns the Nifty 50 ATM strike (step size 50).
func ComputeATMStrike(niftyLTP float64) int {
	return RoundToStrike(niftyLTP, 50)
}

// ComputeCEStrike calculates the CE strike dynamically based on the TradingConfig offset.
func ComputeCEStrike(atm int, cfg *config.TradingConfig) int {
	offset := cfg.GetCEOffsetPoints()
	return atm - offset // CE lower than ATM (ITM)
}

// ComputePEStrike calculates the PE strike dynamically based on the TradingConfig offset.
func ComputePEStrike(atm int, cfg *config.TradingConfig) int {
	offset := cfg.GetPEOffsetPoints()
	return atm + offset // PE higher than ATM (ITM)
}
