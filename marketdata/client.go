package marketdata

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	fyersproto "fyers-trading/marketdata/fyersproto"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

// FyersWSClient connects directly to the Fyers v3 Tbtws (Tick-By-Tick) WebSocket.
type FyersWSClient struct {
	accessToken string
	ringBuffer  *RingBuffer
	logger      *slog.Logger

	conn    *websocket.Conn
	symbols []string
	mu      sync.Mutex // Protects sub/unsub sending
	done    chan struct{}
}

func NewFyersWSClient(accessToken string, rb *RingBuffer, logger *slog.Logger) *FyersWSClient {
	return &FyersWSClient{
		accessToken: accessToken,
		ringBuffer:  rb,
		logger:      logger,
		symbols:     make([]string, 0),
		done:        make(chan struct{}),
	}
}

// Connect establishes the Gorilla WebSocket to the Fyers API v3 docsv3 endpoint.
func (c *FyersWSClient) Connect() error {
	// The TBT socket url is wss://rtsocket-api.fyers.in/versova
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

	// Start reading pump
	go c.readPump()

	return nil
}

// Subscribe requests SymbolUpdate (Tick-by-Tick) for the given NSE symbols.
func (c *FyersWSClient) Subscribe(symbols []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.symbols = symbols
	if c.conn != nil {
		c.sendSubscription(symbols)
	}
}

func (c *FyersWSClient) sendSubscription(symbols []string) {
	// Fyers v3 Tbtws docsv3 Subscription JSON format
	subData := map[string]interface{}{
		"subs":    1,       // 1 for subscribe
		"symbols": symbols,
		"mode":    "depth", // mode can be depth/info
		"channel": "1",     // assigning all sub to channel 1 for now
	}

	subMsg := map[string]interface{}{
		"type": 1, // 1 = Subscription message type
		"data": subData,
	}

	err := c.conn.WriteJSON(subMsg)
	if err != nil {
		c.logger.Error("Failed to send subscription", "err", err)
		return
	}

	// Important: We must also send a Switch Channel message to "resume" channel 1
	// { "type": 2, "data": { "resumeChannels": ["1"], "pauseChannels": [] } }
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
		c.logger.Error("Failed to unmarshal protobuf Tbtws message", "err", err)
		return
	}

	// Type 0 = Ping => We must respond with pong (Gorilla usually does this itself, but doc says we might need to send text "ping"?)
	if socketMsg.Type == fyersproto.MessageType_ping {
		c.logger.Debug("Received Tbtws Ping")
		// The custom Pong response is usually handled by the websocket protocol natively (Gorilla pong handler).
		// However, the docs mention sending "ping" string. We will let standard ping/pong handle it unless disconnections happen.
		return
	}

	if socketMsg.Error {
		c.logger.Error("Tbtws returned error in message", "msg", socketMsg.Msg)
		return
	}

	// We got feeds
	for ticker, feed := range socketMsg.Feeds {
		if feed != nil {
			c.parseAndDispatch(ticker, feed, recvTime)
		}
	}
}

func (c *FyersWSClient) parseAndDispatch(ticker string, feed *fyersproto.MarketFeed, recvTime int64) {
	tick := GetTickEvent()
	tick.RecvTimestamp = recvTime
	tick.Symbol = ticker

	if feed.FeedTime != nil {
		tick.ExchTimestamp = int64(feed.FeedTime.Value)
	}

	// The feeds will have Quote or Depth based on mode. We requested "depth".
	// "quote" has standard LTP / Volume
	if feed.Quote != nil {
		if feed.Quote.Ltp != nil {
			tick.LTP = float64(feed.Quote.Ltp.Value) / 100.0 // Usually, prices in Fyers protobuf are * 100
		}
		if feed.Quote.Vtt != nil {
			tick.Volume = int64(feed.Quote.Vtt.Value)
		}
	} else if feed.Depth != nil {
		// Sometimes LTP is not in depth directly, but we might receive OHLCV or Quote along with it 
		// depending if it's a Snapshot or diff packet.
		// If LTP is not present but Depth is, we can infer top of book if necessary, but Fyers usually sends Quote for trades.
		// Actually, Fyers Tbtws usually populates Quote.LTP on every trade tick.
	}

	// We only want to dispatch if we actually received a price update
	if tick.LTP > 0 {
		c.ringBuffer.Write(tick)
	} else {
		PutTickEvent(tick) // recycle if just a naked depth update without LTP change
	}
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
