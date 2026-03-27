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

	conn       *websocket.Conn
	symbols    []string
	tokenMap   map[string]string // Maps fytoken back to human-readable symbol
	subscribed map[string]bool   // Deduplication tracker
	orderbooks map[string]*TickEvent
	mu         sync.Mutex        // Protects sub/unsub sending and state caches
	done       chan struct{}
	
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
		subscribed:  make(map[string]bool),
		orderbooks:  make(map[string]*TickEvent),
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

	var newLocals []string
	for _, sym := range symbols {
		if !c.subscribed[sym] {
			c.subscribed[sym] = true
			newLocals = append(newLocals, sym)
		}
	}

	// Store full active list so Connect() can seamlessly recreate the feed if the socket drops
	c.symbols = symbols

	if len(newLocals) == 0 {
		c.logger.Debug("Ignoring duplicate TBT subscription request", "symbols", symbols)
		return
	}

	if c.conn != nil {
		c.sendSubscription(newLocals)
	}
}

// Unsubscribe attempts to stop data for symbols we no longer need.
func (c *FyersWSClient) Unsubscribe(symbols []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.conn == nil || len(symbols) == 0 {
		return
	}

	// Purge from deduplication cache so rapid re-subs aren't infinitely blocked
	for _, sym := range symbols {
		delete(c.subscribed, sym)
	}
	
	unsubMsg := map[string]interface{}{
		"type": 1,
		"data": map[string]interface{}{
			"subs":    -1,
			"symbols": symbols,
			"mode":    "depth",
			"channel": "1",
		},
	}

	if err := c.conn.WriteJSON(unsubMsg); err != nil {
		c.logger.Error("Failed to send unsubscription", "err", err)
	} else {
		c.logger.Info("Unsubscribed from old symbols via TBT", "count", len(symbols))
		
		// Senior-Audit: Eliminate memory leak natively!
		for _, sym := range symbols {
			if oldState, ok := c.orderbooks[sym]; ok {
				PutTickEvent(oldState) // Safely recycle the old pointer to the Pool
				delete(c.orderbooks, sym) // Purge the Map
			}
			// Reverse map search to un-alias fytoken to prevent tokenMap bloating
			for fyToken, targetSym := range c.tokenMap {
				if targetSym == sym {
					delete(c.tokenMap, fyToken)
				}
			}
		}
	}
}

func (c *FyersWSClient) sendSubscription(symbols []string) {
	if len(symbols) == 0 {
		return
	}

	// Fyers v3 Tbtws JSON payload format: type: 1
	subMsg := map[string]interface{}{
		"type": 1,
		"data": map[string]interface{}{
			"subs":    1,
			"symbols": symbols,
			"mode":    "depth",
			"channel": "1",
		},
	}

	err := c.conn.WriteJSON(subMsg)
	if err != nil {
		c.logger.Error("Failed to send subscription", "err", err)
		return
	}
	
	// Important: We must also send a Switch Channel message to "resume" channel 1
	resumeMsg := map[string]interface{}{
		"type": 2,
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
	
	// Fyers Tbtws uses application-level text "ping"s instead of standard RFC-6455 Ping/Pong frames.
	// Therefore, setting a strict Gorilla ReadDeadline causes arbitrary 60-second `i/o timeout` disconnects 
	// because the underlying handler never intercepts a protocol-level Pong to refresh the deadline clock.
	// We rely on our `pingPump` and OS TCP KeepAlives to detect genuine dead connections natively.

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
			
			// Deep inspection of the Close frame!
			if closeErr, ok := err.(*websocket.CloseError); ok {
				c.logger.Error("Tbtws explicitly closed connection!", "code", closeErr.Code, "reason_text", closeErr.Text)
			} else {
				c.logger.Warn("Tbtws connection lost (not a structured CloseError).", "err", err.Error())
			}

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
	c.mu.Lock()
	if feed.Ticker != "" {
		c.tokenMap[fytoken] = feed.Ticker
	}
	sym, ok := c.tokenMap[fytoken]
	if !ok || sym == "" {
		sym = fytoken
	}

	state, exists := c.orderbooks[sym]
	if !exists {
		state = GetTickEvent()
		state.Symbol = sym
		c.orderbooks[sym] = state
	} else if feed.Snapshot {
		// Senior-Audit: Overwriting pointers destroys sync.Pool's zero-allocation bounds! 
		// Instead of dropping it to GC, natively zero-initialize exactly what Fyers resets.
		state.Bids = [50]DepthLevel{}
		state.Asks = [50]DepthLevel{}
	}

	state.RecvTimestamp = recvTime

	if feed.FeedTime != nil {
		state.ExchTimestamp = int64(feed.FeedTime.Value)
	}

	if feed.Quote != nil {
		if feed.Quote.Ltp != nil {
			state.LTP = float64(feed.Quote.Ltp.Value) / 100.0
		}
		if feed.Quote.Vtt != nil {
			state.Volume = int64(feed.Quote.Vtt.Value)
		}
	}
	
	if feed.Depth != nil {
		for _, ask := range feed.Depth.Asks {
			if ask != nil && ask.Num != nil && ask.Num.Value < 50 {
				idx := ask.Num.Value
				if ask.Price != nil {
					state.Asks[idx].Price = float64(ask.Price.Value) / 100.0
				}
				if ask.Qty != nil {
					state.Asks[idx].Qty = int64(ask.Qty.Value)
				}
				if ask.Nord != nil {
					state.Asks[idx].Orders = int64(ask.Nord.Value)
				}
			}
		}
		for _, bid := range feed.Depth.Bids {
			if bid != nil && bid.Num != nil && bid.Num.Value < 50 {
				idx := bid.Num.Value
				if bid.Price != nil {
					state.Bids[idx].Price = float64(bid.Price.Value) / 100.0
				}
				if bid.Qty != nil {
					state.Bids[idx].Qty = int64(bid.Qty.Value)
				}
				if bid.Nord != nil {
					state.Bids[idx].Orders = int64(bid.Nord.Value)
				}
			}
		}
	}

	// Always sync the top-of-book natively to the root level struct for strategy convenience
	// Senior-Audit: If Fyers sends a Diff actively destroying L0 liquidity (Level collapsed), 
	// blindly checking > 0 incorrectly shields the struct and eternally stalls `AskPrice` at a stale quote.
	state.AskPrice = state.Asks[0].Price
	state.AskSize = state.Asks[0].Qty
	state.BidPrice = state.Bids[0].Price
	state.BidSize = state.Bids[0].Qty

	// Create a copied clone to push into the lock-free ring buffer
	tickClone := GetTickEvent()
	*tickClone = *state
	c.mu.Unlock()

	c.ringBuffer.Write(tickClone)
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