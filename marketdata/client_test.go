package marketdata

import (
	"testing"
	"time"

	"fyers-trading/marketdata/fyersproto"
	"log/slog"
	"os"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

func createTestClient() *FyersWSClient {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rb := NewRingBuffer(10)
	
	// Create a fully decoupled test client
	c := NewFyersWSClient("dummy_token", rb, logger)
	return c
}

func TestParseAndDispatch_Snapshot(t *testing.T) {
	c := createTestClient()
	reader := c.ringBuffer.NewReader()
	
	// Build a mock Snapshot feed
	fytoken := "101126033051714"
	sym := "NSE:NIFTY2640723300CE"
	
	// Set up the exact initial structure mimicking Fyers Proto response
	feed := &fyersproto.MarketFeed{
		Ticker:   sym,
		Snapshot: true,
		Depth: &fyersproto.Depth{
			Bids: []*fyersproto.MarketLevel{
				{Num: wrapperspb.UInt32(0), Price: wrapperspb.Int64(10050), Qty: wrapperspb.UInt32(100), Nord: wrapperspb.UInt32(1)},
				{Num: wrapperspb.UInt32(1), Price: wrapperspb.Int64(10040), Qty: wrapperspb.UInt32(200), Nord: wrapperspb.UInt32(2)},
			},
			Asks: []*fyersproto.MarketLevel{
				{Num: wrapperspb.UInt32(0), Price: wrapperspb.Int64(10060), Qty: wrapperspb.UInt32(50), Nord: wrapperspb.UInt32(1)},
			},
		},
	}

	c.parseAndDispatch(fytoken, feed, time.Now().UnixNano())

	// Validate it hit the RingBuffer!
	tick := reader.Next()
	if tick == nil {
		t.Fatalf("Expected a tick to be pushed to RingBuffer")
	}

	if tick.Symbol != sym {
		t.Errorf("Expected Symbol %s, got %s", sym, tick.Symbol)
	}
	
	// Check the actual state cache map directly to ensure persistence
	c.mu.Lock()
	state, exists := c.orderbooks[sym]
	c.mu.Unlock()
	
	if !exists {
		t.Fatalf("Orderbook state was not cached for %s", sym)
	}

	if state.Bids[0].Price != 100.50 {
		t.Errorf("Expected Bid[0] Price 100.50, got %v", state.Bids[0].Price)
	}
	if state.Bids[1].Price != 100.40 {
		t.Errorf("Expected Bid[1] Price 100.40, got %v", state.Bids[1].Price)
	}
	if state.Asks[0].Price != 100.60 {
		t.Errorf("Expected Ask[0] Price 100.60, got %v", state.Asks[0].Price)
	}
	// Level 2 should be empty
	if state.Asks[1].Price != 0 {
		t.Errorf("Expected Ask[1] to be completely empty")
	}
}

func TestParseAndDispatch_DiffPacket(t *testing.T) {
	c := createTestClient()
	reader := c.ringBuffer.NewReader()
	
	fytoken := "1011111111"
	sym := "NSE:SBIN-EQ"
	
	// 1. Send an initial Snapshot
	feed1 := &fyersproto.MarketFeed{
		Ticker:   sym,
		Snapshot: true,
		Depth: &fyersproto.Depth{
			Bids: []*fyersproto.MarketLevel{
				{Num: wrapperspb.UInt32(0), Price: wrapperspb.Int64(50000), Qty: wrapperspb.UInt32(100)},
				{Num: wrapperspb.UInt32(1), Price: wrapperspb.Int64(49950), Qty: wrapperspb.UInt32(200)},
				{Num: wrapperspb.UInt32(2), Price: wrapperspb.Int64(49900), Qty: wrapperspb.UInt32(300)},
			},
		},
	}
	c.parseAndDispatch(fytoken, feed1, time.Now().UnixNano())

	_ = reader.Next() // Clear the first snapshot
	
	// 2. Send a DIFF packet updating strictly Level 1 (index 1) and Level 3 (index 3)
	feed2 := &fyersproto.MarketFeed{
		Snapshot: false, // Diff!
		Depth: &fyersproto.Depth{
			Bids: []*fyersproto.MarketLevel{
				// Level 1 qty changes from 200 to 500
				{Num: wrapperspb.UInt32(1), Price: wrapperspb.Int64(49950), Qty: wrapperspb.UInt32(500)},
				// New Level 3 appears
				{Num: wrapperspb.UInt32(3), Price: wrapperspb.Int64(49800), Qty: wrapperspb.UInt32(1000)},
			},
		},
	}
	// Don't send Ticker explicitly — should fall back intelligently based on fytoken
	c.parseAndDispatch(fytoken, feed2, time.Now().UnixNano())

	tick := reader.Next()
	if tick == nil {
		t.Fatalf("Expected a second tick to be pushed to RingBuffer")
	}
	
	// Validate the patch applied successfully
	c.mu.Lock()
	state := c.orderbooks[sym]
	c.mu.Unlock()

	// Level 0 should remain UNCHANGED from snapshot
	if state.Bids[0].Price != 500.00 || state.Bids[0].Qty != 100 {
		t.Errorf("Diff packet erroneously wiped Level 0! Got Price: %f, Qty: %d", state.Bids[0].Price, state.Bids[0].Qty)
	}

	// Level 1 should be UPDATED
	if state.Bids[1].Price != 499.50 || state.Bids[1].Qty != 500 {
		t.Errorf("Diff packet failed to update Level 1! Got Qty: %d", state.Bids[1].Qty)
	}

	// Level 2 should remain UNCHANGED from snapshot
	if state.Bids[2].Price != 499.00 || state.Bids[2].Qty != 300 {
		t.Errorf("Diff packet erroneously wiped Level 2!")
	}

	// Level 3 should be NEWLY APPLIED
	if state.Bids[3].Price != 498.00 || state.Bids[3].Qty != 1000 {
		t.Errorf("Diff packet failed to insert Level 3!")
	}
}

// TestParseAndDispatch_InvalidIndex verifies out-of-bounds market levels don't panic
func TestParseAndDispatch_InvalidIndex(t *testing.T) {
	c := createTestClient()
	
	fytoken := "invalid_idx_test"
	sym := "NSE:TEST"
	
	feed := &fyersproto.MarketFeed{
		Ticker:   sym,
		Snapshot: true,
		Depth: &fyersproto.Depth{
			Bids: []*fyersproto.MarketLevel{
				{Num: wrapperspb.UInt32(5), Price: wrapperspb.Int64(10000), Qty: wrapperspb.UInt32(1)}, // Valid
				{Num: wrapperspb.UInt32(200), Price: wrapperspb.Int64(20000), Qty: wrapperspb.UInt32(1)}, // Invalid! Max is 49
			},
		},
	}
	
	// Execute — Should not panic.
	c.parseAndDispatch(fytoken, feed, time.Now().UnixNano())
	
	c.mu.Lock()
	state := c.orderbooks[sym]
	c.mu.Unlock()
	
	if state.Bids[5].Price != 100.00 {
		t.Errorf("Valid deep level failed to parse")
	}
}
