package marketdata

import (
	"log/slog"
	"time"
)

type LatencyMonitor struct {
	logger *slog.Logger
	intv   time.Duration
}

func NewLatencyMonitor(logger *slog.Logger, interval time.Duration) *LatencyMonitor {
	return &LatencyMonitor{
		logger: logger,
		intv:   interval,
	}
}

// StartReader loops over the ring buffer and calculates end-to-end processing delays.
func (lm *LatencyMonitor) StartReader(reader *Reader) {
	go func() {
		lm.logger.Info("Latency monitor started")
		ticker := time.NewTicker(lm.intv)
		defer ticker.Stop()

		var maxLatency int64
		var totalLatency int64
		var count int64

		for {
			select {
			case <-ticker.C:
				if count > 0 {
					avg := time.Duration(totalLatency / count)
					max := time.Duration(maxLatency)
					lm.logger.Info("Latency Report",
						"ticks_processed", count,
						"avg_latency_internal", avg.String(),
						"max_latency_internal", max.String(),
					)
					// Reset
					maxLatency = 0
					totalLatency = 0
					count = 0
				}
			default:
				tick := reader.Next()
				if tick == nil {
					lm.logger.Info("Latency monitor stopped")
					return
				}
				now := time.Now().UnixNano()
				latency := now - tick.RecvTimestamp

				if latency > maxLatency {
					maxLatency = latency
				}
				totalLatency += latency
				count++
			}
		}
	}()
}
