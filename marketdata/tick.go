package marketdata

import "sync"

// DepthLevel holds one level of 5-level market depth (from DepthUpdate feed).
type DepthLevel struct {
	Price  float64
	Size   int64
	Orders int64
}

// TickEvent represents a normalized tick from the exchange.
// Depth fields are only populated for CE/PE option ticks (DepthUpdate feed).
// Index ticks (SymbolUpdate feed) only populate the top-level OHLC/LTP fields.
type TickEvent struct {
	Symbol        string
	LTP           float64
	Open          float64
	High          float64
	Low           float64
	Close         float64
	Volume        int64
	Change        float64
	ChangePercent float64
	ExchTimestamp int64 // timestamp from exchange (seconds); absent in DepthUpdate
	RecvTimestamp int64 // time.Now().UnixNano() at receive

	BidPrice float64
	AskPrice float64
	BidSize  int64
	AskSize  int64

	// 5-level market depth (DepthUpdate only)
	Bid [5]DepthLevel
	Ask [5]DepthLevel
}

// pool allows zero-allocation hot-path for TickEvents
var tickEventPool = sync.Pool{
	New: func() interface{} {
		return new(TickEvent)
	},
}

// GetTickEvent fetches a cleanly initialized TickEvent from the pool
func GetTickEvent() *TickEvent {
	tick := tickEventPool.Get().(*TickEvent)
	// Clear all fields before returning
	*tick = TickEvent{}
	return tick
}

// PutTickEvent returns a TickEvent to the pool
func PutTickEvent(tick *TickEvent) {
	if tick != nil {
		tickEventPool.Put(tick)
	}
}
