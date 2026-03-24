package marketdata

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	fyersgosdk "github.com/FyersDev/fyers-go-sdk"
	fyersws "github.com/FyersDev/fyers-go-sdk/websocket"
)

type FyersWSClient struct {
	socket          *fyersws.FyersDataSocket
	ringBuffer      *RingBuffer
	logger          *slog.Logger
	symbols         []string
	OnInvalidSymbol func(invalidSymbols []string) // Callback for pipeline fallback
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
		
		// If Fyers rejects our symbols, parse out the list and trigger fallback
		if client.OnInvalidSymbol != nil {
			if invalidRaw, ok := message["invalid_symbols"]; ok {
				if invalidList, ok := invalidRaw.([]interface{}); ok {
					var invalidStrs []string
					for _, v := range invalidList {
						if str, isStr := v.(string); isStr {
							invalidStrs = append(invalidStrs, str)
						}
					}
					if len(invalidStrs) > 0 {
						client.OnInvalidSymbol(invalidStrs)
					}
				}
			}
		}
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
		if bidPrice, ok := message["bid_price"]; ok {
			tick.BidPrice = convertToFloat64(bidPrice)
		}
		if askPrice, ok := message["ask_price"]; ok {
			tick.AskPrice = convertToFloat64(askPrice)
		}
		if bidSize, ok := message["bid_size"]; ok {
			tick.BidSize = convertToInt64(bidSize)
		}
		if askSize, ok := message["ask_size"]; ok {
			tick.AskSize = convertToInt64(askSize)
		}
		if openVal, ok := message["open_price"]; ok {
			tick.Open = convertToFloat64(openVal)
		}
		if highVal, ok := message["high_price"]; ok {
			tick.High = convertToFloat64(highVal)
		}
		if lowVal, ok := message["low_price"]; ok {
			tick.Low = convertToFloat64(lowVal)
		}
		if closeVal, ok := message["prev_close_price"]; ok {
			tick.Close = convertToFloat64(closeVal)
		}
		if chVal, ok := message["ch"]; ok {
			tick.Change = convertToFloat64(chVal)
		}
		if chpVal, ok := message["chp"]; ok {
			tick.ChangePercent = convertToFloat64(chpVal)
		}

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
	c.symbols = append(c.symbols, symbols...)
	if c.socket != nil {
		c.socket.Subscribe(symbols, "SymbolUpdate")
		c.logger.Info("Subscribed to symbols", "symbols", symbols)
	}
}

func (c *FyersWSClient) Unsubscribe(symbols []string) {
	// Not perfect symbol tracking here, but Fyers SDK tracks it internally anyway for Unsubscribe
	if c.socket != nil {
		c.socket.Unsubscribe(symbols, "SymbolUpdate")
		c.logger.Info("Unsubscribed from symbols", "symbols", symbols)
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

// VerifyToken calls the Fyers Profile API to ensure the access token is valid
// before attempting to open the WebSocket connection.
func VerifyToken(appID, accessToken string) error {
	client := fyersgosdk.NewFyersModel(appID, accessToken)

	profStr, err := client.GetProfile()
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}

	var profile fyersgosdk.Profile
	if err := json.Unmarshal([]byte(profStr), &profile); err != nil {
		return fmt.Errorf("failed to parse profile response: %w", err)
	}

	if profile.S != "ok" {
		return fmt.Errorf("invalid token (code: %d, message: %s)", profile.Code, profile.Message)
	}

	return nil
}
