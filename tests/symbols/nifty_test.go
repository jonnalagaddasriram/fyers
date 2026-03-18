package symbols_test

import (
	"testing"

	"fyers-trading/config"
	"fyers-trading/symbols"
)

func TestNiftyIndexSymbol(t *testing.T) {
	expected := "NSE:NIFTY50-INDEX"
	if got := symbols.NiftyIndexSymbol(); got != expected {
		t.Errorf("Expected %s, got %s", expected, got)
	}
}

func TestBuildOptionSymbol(t *testing.T) {
	expected := "NSE:NIFTY2631723250CE"
	got := symbols.BuildOptionSymbol("26317", 23250, "CE")
	if got != expected {
		t.Errorf("Expected %s, got %s", expected, got)
	}
}

func TestComputeStrikesWithConfig(t *testing.T) {
	// Create a mock config
	cfg := &config.TradingConfig{}
	cfg.CEOffsetPoints = 150
	cfg.PEOffsetPoints = 100 // Different offset to test mapping

	atm := symbols.ComputeATMStrike(23240.1)
	if atm != 23250 {
		t.Errorf("Expected ATM 23250, got %d", atm)
	}
	
	ce := symbols.ComputeCEStrike(atm, cfg)
	if ce != 23100 { // 23250 - 150 = 23100 (ITM CE)
		t.Errorf("Expected CE 23100, got %d", ce)
	}
	
	pe := symbols.ComputePEStrike(atm, cfg)
	if pe != 23350 { // 23250 + 100 = 23350 (ITM PE)
		t.Errorf("Expected PE 23350, got %d", pe)
	}
}
