package marketdata

import (
	"log/slog"
	"sync"
)

// TickStore provides O(1) concurrent read access to the latest tick per symbol
type TickStore struct {
	store  sync.Map
	logger *slog.Logger
}

func NewTickStore(logger *slog.Logger) *TickStore {
	return &TickStore{
		logger: logger,
	}
}

// Store saves a tick. The caller passes a copied or safe tick because
// if the tick was from a ring buffer, it might be overwritten later.
// To avoid allocations, we store a copy in sync.Map or store fields flat.
func (ts *TickStore) Store(tick *TickEvent) {
	// make a tiny copy to keep in map since pool ticks get recycled
	// Alternatively, just store a struct copy instead of pointer.
	t := *tick
	ts.store.Store(tick.Symbol, t)
}

// Get returns the latest tick and bool if found
func (ts *TickStore) Get(symbol string) (TickEvent, bool) {
	if val, ok := ts.store.Load(symbol); ok {
		return val.(TickEvent), true
	}
	return TickEvent{}, false
}

// StartReader starts a goroutine to continuously consume the RingBuffer and update TickStore.
// It skips to latest if falling behind.
func (ts *TickStore) StartReader(reader *Reader) {
	go func() {
		ts.logger.Info("TickStore reader started")
		for {
			tick := reader.Next()
			if tick == nil {
				ts.logger.Info("TickStore reader stopped")
				return
			}
			ts.Store(tick)
		}
	}()
}
