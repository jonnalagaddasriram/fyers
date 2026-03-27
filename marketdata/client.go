package marketdata

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	fyersgosdk "github.com/FyersDev/fyers-go-sdk"
	fyersws "github.com/FyersDev/fyers-go-sdk/websocket"
)

type FyersWSClient struct {
	socket          *fyersws.FyersDataSocket
	ringBuffer      *RingBuffer
	logger          *slog.Logger
	symbols         []string // CE/PE option symbols (DepthUpdate)
	indexSymbols    []string // Index symbols (SymbolUpdate)
	subscribed      map[string]bool
	orderbooks      map[string]*TickEvent
	mu              sync.Mutex
	OnInvalidSymbol func(invalidSymbols []string) // Callback for pipeline fallback
}

func NewFyersWSClient(accessToken string, rb *RingBuffer, logger *slog.Logger) *FyersWSClient {
	client := &FyersWSClient{
		ringBuffer:   rb,
		logger:       logger,
		symbols:      make([]string, 0),
		indexSymbols: make([]string, 0),
		subscribed:   make(map[string]bool),
		orderbooks:   make(map[string]*TickEvent),
	}

	onConnect := func() {
		client.logger.Info("WebSocket Connected")
		if len(client.indexSymbols) > 0 {
			client.socket.Subscribe(client.indexSymbols, "SymbolUpdate")
		}
		if len(client.symbols) > 0 {
			client.socket.Subscribe(client.symbols, "DepthUpdate")
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

		var sym string
		if s, ok := message["symbol"].(string); ok {
			sym = s
		} else {
			return // Ignore malformed messages entirely
		}

		client.mu.Lock()
		state, exists := client.orderbooks[sym]
		if !exists {
			state = GetTickEvent()
			state.Symbol = sym
			client.orderbooks[sym] = state
		}

		state.RecvTimestamp = recvTime

		// Common OHLC / LTP fields (present in SymbolUpdate; may be absent in DepthUpdate)
		if ltpVal, ok := message["ltp"]; ok {
			state.LTP = convertToFloat64(ltpVal)
		}
		if volVal, ok := message["vol_traded_today"]; ok {
			state.Volume = convertToInt64(volVal)
		}
		if volVal, ok := message["volume"]; ok && state.Volume == 0 {
			state.Volume = convertToInt64(volVal)
		}
		if exchTimeVal, ok := message["exch_feed_time"]; ok {
			state.ExchTimestamp = convertToInt64(exchTimeVal)
		}
		if openVal, ok := message["open_price"]; ok {
			state.Open = convertToFloat64(openVal)
		}
		if highVal, ok := message["high_price"]; ok {
			state.High = convertToFloat64(highVal)
		}
		if lowVal, ok := message["low_price"]; ok {
			state.Low = convertToFloat64(lowVal)
		}
		if closeVal, ok := message["prev_close_price"]; ok {
			state.Close = convertToFloat64(closeVal)
		}
		if chVal, ok := message["ch"]; ok {
			state.Change = convertToFloat64(chVal)
		}
		if chpVal, ok := message["chp"]; ok {
			state.ChangePercent = convertToFloat64(chpVal)
		}

		// 5-level market depth — only present in DepthUpdate ticks (type="dp").
		// Keys are 1-indexed: bid_price1…bid_price5, ask_price1…ask_price5, etc.
		if message["type"] == "dp" {
			for i := 0; i < 5; i++ {
				n := i + 1
				if v, ok := message[fmt.Sprintf("bid_price%d", n)]; ok {
					state.Bid[i].Price = convertToFloat64(v)
				}
				if v, ok := message[fmt.Sprintf("bid_size%d", n)]; ok {
					state.Bid[i].Size = convertToInt64(v)
				}
				if v, ok := message[fmt.Sprintf("bid_order%d", n)]; ok {
					state.Bid[i].Orders = convertToInt64(v)
				}
				if v, ok := message[fmt.Sprintf("ask_price%d", n)]; ok {
					state.Ask[i].Price = convertToFloat64(v)
				}
				if v, ok := message[fmt.Sprintf("ask_size%d", n)]; ok {
					state.Ask[i].Size = convertToInt64(v)
				}
				if v, ok := message[fmt.Sprintf("ask_order%d", n)]; ok {
					state.Ask[i].Orders = convertToInt64(v)
				}
			}
		}

		// Secure strategy API compatibility safely overriding the root Level 0 elements
		state.AskPrice = state.Ask[0].Price
		state.AskSize = state.Ask[0].Size
		state.BidPrice = state.Bid[0].Price
		state.BidSize = state.Bid[0].Size

		// Create a strictly deep-copied wrapper clone out of the mutex mapping
		tickClone := GetTickEvent()
		*tickClone = *state
		client.mu.Unlock()

		// Write to ring buffer strictly lock-free
		client.ringBuffer.Write(tickClone)
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

// Subscribe subscribes to CE/PE option symbols using DepthUpdate (full order book).
func (c *FyersWSClient) Subscribe(symbols []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	var newLocals []string
	for _, sym := range symbols {
		if !c.subscribed[sym] {
			c.subscribed[sym] = true
			newLocals = append(newLocals, sym)
		}
	}
	
	c.symbols = symbols
	
	if len(newLocals) > 0 && c.socket != nil {
		c.socket.Subscribe(newLocals, "DepthUpdate")
		c.logger.Info("Subscribed to CE/PE symbols (DepthUpdate)", "symbols", newLocals)
	}
}

// Unsubscribe removes CE/PE option symbols from the DepthUpdate feed.
func (c *FyersWSClient) Unsubscribe(symbols []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	for _, sym := range symbols {
		delete(c.subscribed, sym)
		if oldState, ok := c.orderbooks[sym]; ok {
			PutTickEvent(oldState)
			delete(c.orderbooks, sym) // Purge Memory Leak natively
		}
	}
	
	if len(symbols) > 0 && c.socket != nil {
		c.socket.Unsubscribe(symbols, "DepthUpdate")
		// c.logger.Info("Unsubscribed from CE/PE symbols (DepthUpdate)", "symbols", symbols)
	}
}

// SubscribeIndex subscribes to index symbols using SymbolUpdate (LTP-only, lighter feed).
func (c *FyersWSClient) SubscribeIndex(symbols []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var newLocals []string
	for _, sym := range symbols {
		if !c.subscribed[sym] {
			c.subscribed[sym] = true
			newLocals = append(newLocals, sym)
		}
	}
	
	c.indexSymbols = symbols
	
	if len(newLocals) > 0 && c.socket != nil {
		c.socket.Subscribe(newLocals, "SymbolUpdate")
		// c.logger.Info("Subscribed to index symbols (SymbolUpdate)", "symbols", newLocals)
	}
}

// UnsubscribeIndex removes index symbols from the SymbolUpdate feed.
func (c *FyersWSClient) UnsubscribeIndex(symbols []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	for _, sym := range symbols {
		delete(c.subscribed, sym)
		if oldState, ok := c.orderbooks[sym]; ok {
			PutTickEvent(oldState)
			delete(c.orderbooks, sym)
		}
	}
	
	if len(symbols) > 0 && c.socket != nil {
		c.socket.Unsubscribe(symbols, "SymbolUpdate")
		// c.logger.Info("Unsubscribed from index symbols (SymbolUpdate)", "symbols", symbols)
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
