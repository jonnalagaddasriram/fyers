package marketdata

import "sync"

// TickEvent represents a normalized tick from the exchange
type TickEvent struct {
	Symbol        string
	LTP           float64
	Open          float64
	High          float64
	Low           float64
	Close         float64
	Volume        int64
	BidPrice      float64
	AskPrice      float64
	Change        float64
	ChangePercent float64
	ExchTimestamp int64 // timestamp from exchange (seconds)
	RecvTimestamp int64 // time.Now().UnixNano() at receive
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
