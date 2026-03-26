package marketdata

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	fyersgosdk "github.com/FyersDev/fyers-go-sdk"
	fyersproto "fyers-trading/marketdata/fyersproto"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

// FyersWSClient connects directly to the Fyers v3 Tbtws (Tick-By-Tick) WebSocket.
type FyersWSClient struct {
	accessToken string
	ringBuffer  *RingBuffer
	logger      *slog.Logger

	conn     *websocket.Conn
	symbols  []string
	tokenMap map[string]string // Maps fytoken back to human-readable symbol
	mu       sync.Mutex        // Protects sub/unsub sending and tokenMap
	done     chan struct{}
	
	// OnInvalidSymbol is a callback hook triggered if the broker rejects specific symbols.
	OnInvalidSymbol func(invalidSymbols []string)
}

func NewFyersWSClient(accessToken string, rb *RingBuffer, logger *slog.Logger) *FyersWSClient {
	return &FyersWSClient{
		accessToken: accessToken,
		ringBuffer:  rb,
		logger:      logger,
		symbols:     make([]string, 0),
		tokenMap:    make(map[string]string),
		done:        make(chan struct{}),
	}
}

// Connect establishes the Gorilla WebSocket to the Fyers API v3 docsv3 endpoint.
func (c *FyersWSClient) Connect() error {
	url := "wss://rtsocket-api.fyers.in/versova"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	// Auth is required in Authorization header format < appId:accessToken >
	req.Header.Set("Authorization", c.accessToken)

	c.logger.Info("Connecting to custom Fyers v3 Tbtws...")

	dialer := websocket.DefaultDialer
	conn, resp, err := dialer.Dial(req.URL.String(), req.Header)
	if err != nil {
		if resp != nil {
			c.logger.Error("WS Handshake failed", "status", resp.StatusCode)
		}
		return err
	}

	c.conn = conn
	c.logger.Info("Connected to Fyers v3 Tbtws successfully.")

	// Resubscribe if we had active symbols (e.g., during a reconnect)
	if len(c.symbols) > 0 {
		c.sendSubscription(c.symbols)
	}

	// Start reading pump and ping pump
	go c.readPump()
	go c.pingPump()

	return nil
}

// isSameSymbols checks if two string slices contain the exact same elements (order-independent)
func isSameSymbols(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int)
	for _, s := range a {
		counts[s]++
	}
	for _, s := range b {
		counts[s]--
		if counts[s] < 0 {
			return false
		}
	}
	return true
}

// Subscribe requests SymbolUpdate (Tick-by-Tick) for the given NSE symbols.
func (c *FyersWSClient) Subscribe(symbols []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Dedup check: if we are already subscribed to these exact symbols, skip
	if isSameSymbols(c.symbols, symbols) {
		c.logger.Info("ATM maintained, already subscribed to symbols", "symbols", symbols)
		return
	}

	c.symbols = symbols
	if c.conn != nil {
		c.sendSubscription(symbols)
	}
}

// Unsubscribe attempts to stop data for symbols we no longer need.
func (c *FyersWSClient) Unsubscribe(symbols []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.conn == nil || len(symbols) == 0 {
		return
	}
	
	unsubData := map[string]interface{}{
		"subs":    -1, // -1 for unsubscribe
		"symbols": symbols,
		"mode":    "depth",
		"channel": "1",
	}

	unsubMsg := map[string]interface{}{
		"type": 1, // 1 = Subscription message type
		"data": unsubData,
	}

	if err := c.conn.WriteJSON(unsubMsg); err != nil {
		c.logger.Error("Failed to send unsubscription", "err", err)
	} else {
		c.logger.Info("Unsubscribed from old symbols via TBT", "count", len(symbols))
	}
}

func (c *FyersWSClient) sendSubscription(symbols []string) {
	if len(symbols) == 0 {
		return
	}

	// Fyers v3 Tbtws docsv3 Subscription JSON format
	subData := map[string]interface{}{
		"subs":    1,       // 1 for subscribe
		"symbols": symbols,
		"mode":    "depth", // Unthrottled 50-level market depth
		"channel": "1",     // assigning all sub to channel 1 for now
	}

	subMsg := map[string]interface{}{
		"type": 1, 
		"data": subData,
	}

	err := c.conn.WriteJSON(subMsg)
	if err != nil {
		c.logger.Error("Failed to send subscription", "err", err)
		return
	}

	// Important: We must also send a Switch Channel message to "resume" channel 1
	resumeMsg := map[string]interface{}{
		"type": 2, // 2 = switch channel
		"data": map[string]interface{}{
			"resumeChannels": []string{"1"},
			"pauseChannels":  []string{},
		},
	}
	err = c.conn.WriteJSON(resumeMsg)
	if err != nil {
		c.logger.Error("Failed to send channel resume", "err", err)
	} else {
		c.logger.Info("Subscribed and resumed TBT stream successfully", "count", len(symbols))
	}
}

// pingPump continuously sends the exact string "ping" application-level heartbeat every 15 seconds.
func (c *FyersWSClient) pingPump() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.mu.Lock()
			if c.conn != nil {
				// The documentation specifically says "Data Format: "ping""
				// So we send a literal text frame containing "ping"
				err := c.conn.WriteMessage(websocket.TextMessage, []byte("ping"))
				if err != nil {
					c.logger.Warn("Failed to send Tbtws Keep-Alive ping", "error", err)
				} else {
					c.logger.Debug("Sent Tbtws Keep-Alive ping")
				}
			}
			c.mu.Unlock()
		}
	}
}

// readPump listens for incoming Tbtws messages and funnels them directly to the lock-free Ring Buffer.
func (c *FyersWSClient) readPump() {
	defer func() {
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
	}()

	c.conn.SetReadLimit(81920) // Allow large packets if Fyers decides to batch during spikes
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		select {
		case <-c.done:
			return
		default:
		}

		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("Tbtws Disconnected unexpectedly", "err", err.Error())
			}
			c.logger.Warn("Tbtws connection lost.", "err", err.Error())
			// Simple auto-reconnect logic
			time.Sleep(2 * time.Second)
			_ = c.Connect()
			return
		}

		c.processMessage(message)
	}
}

// processMessage parses the protobuf payload from Fyers API v3 Tbtws.
func (c *FyersWSClient) processMessage(msg []byte) {
	recvTime := time.Now().UnixNano()

	var socketMsg fyersproto.SocketMessage
	err := proto.Unmarshal(msg, &socketMsg)
	if err != nil {
		// Could be a JSON error payload if not binary?
		c.logger.Error("Failed to unmarshal protobuf Tbtws message", "err", err)
		return
	}

	// Type 0 = Ping => We must respond with pong (Gorilla usually does this itself via WS protocol)
	if socketMsg.Type == fyersproto.MessageType_ping {
		c.logger.Debug("Received Tbtws Ping")
		return
	}

	if socketMsg.Error {
		c.logger.Error("Tbtws returned error in message", "msg", socketMsg.Msg)
		
		// If the error message indicates invalid symbols, notify the pipeline
		if c.OnInvalidSymbol != nil {
			c.OnInvalidSymbol([]string{}) // Just a generic fallback trigger for now
		}
		return
	}

	// We got feeds
	for ticker, feed := range socketMsg.Feeds {
		if feed != nil {
			c.parseAndDispatch(ticker, feed, recvTime)
		}
	}
}

func (c *FyersWSClient) parseAndDispatch(fytoken string, feed *fyersproto.MarketFeed, recvTime int64) {
	tick := GetTickEvent()
	tick.RecvTimestamp = recvTime

	// Map numeric fytoken back to normal string symbol
	c.mu.Lock()
	if feed.Ticker != "" {
		c.tokenMap[fytoken] = feed.Ticker
	}
	sym, ok := c.tokenMap[fytoken]
	c.mu.Unlock()

	if ok && sym != "" {
		tick.Symbol = sym
	} else {
		tick.Symbol = fytoken // fallback to raw
	}

	if feed.FeedTime != nil {
		tick.ExchTimestamp = int64(feed.FeedTime.Value)
	}

	if feed.Quote != nil {
		if feed.Quote.Ltp != nil {
			tick.LTP = float64(feed.Quote.Ltp.Value) / 100.0 // Usually, prices in Fyers protobuf are * 100
		}
		if feed.Quote.Vtt != nil {
			tick.Volume = int64(feed.Quote.Vtt.Value)
		}
	}
	
	if feed.Depth != nil {
		for _, ask := range feed.Depth.Asks {
			if ask != nil && ask.Num != nil && ask.Num.Value == 0 {
				if ask.Price != nil {
					tick.AskPrice = float64(ask.Price.Value) / 100.0
				}
			}
		}
		for _, bid := range feed.Depth.Bids {
			if bid != nil && bid.Num != nil && bid.Num.Value == 0 {
				if bid.Price != nil {
					tick.BidPrice = float64(bid.Price.Value) / 100.0
				}
			}
		}
	}

	// Unconditionally dispatch the tick! If Fyers sent a diff packet (even if it's just 
	// Level 3 orderbook quantity changing), it counts as a valid network event.
	c.ringBuffer.Write(tick)
}

// Close gracefully shuts down the WebSocket to flush pending updates.
func (c *FyersWSClient) Close() {
	c.logger.Info("Closing custom Fyers Tbtws connection...")
	close(c.done)
	if c.conn != nil {
		c.mu.Lock()
		defer c.mu.Unlock()

		err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			c.logger.Warn("Error writing close message", "err", err)
		}
		time.Sleep(100 * time.Millisecond) // Give it time to flush
		c.conn.Close()
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