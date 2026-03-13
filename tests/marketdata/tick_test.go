package marketdata_test

import (
	"testing"
	"fyers-trading/marketdata"
)

func TestTickEventPool_CleanGet(t *testing.T) {
	// Get a tick from pool
	tick1 := marketdata.GetTickEvent()
	
    if tick1.Symbol != "" || tick1.LTP != 0.0 || tick1.Volume != 0 {
		t.Errorf("Expected clean TickEvent to be zeroed initially")
	}
    
    tick1.Symbol = "NIFTY"
	tick1.LTP = 22000.0
    tick1.Volume = 500

	// Put it back
	marketdata.PutTickEvent(tick1)

	// Get a tick again, should be cleanly zeroed if the pool re-used it
	tick2 := marketdata.GetTickEvent()
	
    if tick2.Symbol != "" || tick2.LTP != 0.0 || tick2.Volume != 0 {
		t.Errorf("Expected recycled TickEvent to be zeroed, got Symbol='%s', LTP=%f, Vol=%d", tick2.Symbol, tick2.LTP, tick2.Volume)
	}
}

func TestTickEventPool_NilPut(t *testing.T) {
    // Should not panic or crash
    marketdata.PutTickEvent(nil)
}
