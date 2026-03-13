package marketdata

import (
	"log/slog"
	"time"

	fyersws "github.com/FyersDev/fyers-go-sdk/websocket"
)

type FyersWSClient struct {
	socket     *fyersws.FyersDataSocket
	ringBuffer *RingBuffer
	logger     *slog.Logger
	symbols    []string
}

func NewFyersWSClient(accessToken string, rb *RingBuffer, logger *slog.Logger) *FyersWSClient {
	client := &FyersWSClient{
		ringBuffer: rb,
		logger:     logger,
		symbols:    make([]string, 0),
	}

	onConnect := func() {
		client.logger.Info("WebSocket Connected")
		if len(client.symbols) > 0 {
			client.socket.Subscribe(client.symbols, "SymbolUpdate")
		}
	}

	onClose := func(message fyersws.DataClose) {
		client.logger.Warn("WebSocket Closed", "message", message)
	}

	onError := func(message fyersws.DataError) {
		client.logger.Error("WebSocket Error", "error", message)
	}

	onMessage := func(message fyersws.DataResponse) {
		recvTime := time.Now().UnixNano()

		// Get an empty tick from pool
		tick := GetTickEvent()
		tick.RecvTimestamp = recvTime

		// Parse the map[string]interface{}
		if sym, ok := message["symbol"].(string); ok {
			tick.Symbol = sym
		} else {
			PutTickEvent(tick)
			return // Ignore weird messages
		}

		if ltpVal, ok := message["ltp"]; ok {
			tick.LTP = convertToFloat64(ltpVal)
		}
		if volVal, ok := message["vol_traded_today"]; ok {
			tick.Volume = convertToInt64(volVal) // Volume might be 'vol_traded_today' or 'volume'
		}
		if volVal, ok := message["volume"]; ok && tick.Volume == 0 {
			tick.Volume = convertToInt64(volVal)
		}
		if exchTimeVal, ok := message["exch_feed_time"]; ok {
			tick.ExchTimestamp = convertToInt64(exchTimeVal)
		}
		// Try bid/ask which might be in "bids" / "asks" mapped arrays,
		// but for Nifty options getting LTP is prioritized on the hotpath.
		// Detailed fields can be added here.

		// Write to ring buffer
		client.ringBuffer.Write(tick)
	}

	// 50 max retries, auto-reconnect true
	client.socket = fyersws.NewFyersDataSocket(
		accessToken,
		"",    // log path
		false, // lite mode disabled
		false, // output to file
		true,  // auto reconnect
		50,    // max retries
		onConnect,
		onClose,
		onError,
		onMessage,
	)

	return client
}

func (c *FyersWSClient) Connect() error {
	c.logger.Info("Attempting WebSocket connection...")
	return c.socket.Connect()
}

func (c *FyersWSClient) Close() {
	c.logger.Info("Closing WebSocket connection")
	c.socket.CloseConnection()
}

func (c *FyersWSClient) Subscribe(symbols []string) {
	c.symbols = symbols
	if c.socket != nil { // will actually send if connected
		c.socket.Subscribe(symbols, "SymbolUpdate")
		c.logger.Info("Subscribed to symbols", "symbols", symbols)
	}
}

// Helper to handle Fyers SDK custom float types (fyersws.FloatSDK)
func convertToFloat64(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case fyersws.FloatSDK:
		return float64(v)
	default:
		return 0
	}
}

// Helper to handle Fyers SDK integer types
func convertToInt64(val interface{}) int64 {
	switch v := val.(type) {
	case int64:
		return v
	case int32:
		return int64(v)
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case fyersws.FloatSDK:
		return int64(v)
	default:
		return 0
	}
}
