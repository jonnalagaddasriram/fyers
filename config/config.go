package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	FyersAppID        string
	FyersAppSecret    string
	FyersAccessToken  string
	FyersRedirectURL  string
	DryRun            bool
	LogLevel          string
	LogFormat         string
	RingBufferSize    int
	LatencyReportIntv time.Duration
	RedisURL          string
	RedisStateSync    time.Duration
	Env               string
}

type TradingConfig struct {
	mu                         sync.RWMutex
	TradeDirection             string `json:"trade_direction"`
	ExpiryWeek                 int    `json:"expiry_week"` // 0 = current, 1 = next
	CEOffsetPoints             int    `json:"ce_offset_points"`
	PEOffsetPoints             int    `json:"pe_offset_points"`
	LotSize                    int    `json:"lot_size"`
	NumLots                    int    `json:"num_lots"`
	ProductType                string `json:"product_type"`
	OrderType                  string `json:"order_type"`
	StrikeChangeConfirmSeconds int    `json:"strike_change_confirm_seconds"`
	StrikeChangeExtraSeconds   int    `json:"strike_change_extra_seconds"`
	MaxConfirmSeconds          int    `json:"max_confirm_seconds"`
	MaxPositionValue           int    `json:"max_position_value"`
	MaxDailyLoss               int    `json:"max_daily_loss"`
	TradingStartTime           string `json:"trading_start_time"`
	TradingEndTime             string `json:"trading_end_time"`
}

func LoadAppConfig() (*AppConfig, error) {
	// Not an error if .env doesn't exist, we might be in Docker with env vars already set
	_ = godotenv.Load()

	cfg := &AppConfig{
		FyersAppID:       os.Getenv("FYERS_APP_ID"),
		FyersAppSecret:   os.Getenv("FYERS_APP_SECRET"),
		FyersAccessToken: os.Getenv("FYERS_ACCESS_TOKEN"),
		FyersRedirectURL: os.Getenv("FYERS_REDIRECT_URL"),
		LogLevel:         getEnvOrDefault("LOG_LEVEL", "info"),
		LogFormat:        getEnvOrDefault("LOG_FORMAT", "json"),
		Env:              getEnvOrDefault("ENV", "development"),
		RedisURL:         getEnvOrDefault("REDIS_URL", "redis://localhost:6379/0"),
	}

	dryRunStr := getEnvOrDefault("DRY_RUN", "true")
	cfg.DryRun = (dryRunStr == "true" || dryRunStr == "1" || dryRunStr == "yes")

	rbSizeStr := getEnvOrDefault("RING_BUFFER_SIZE", "65536")
	if size, err := strconv.Atoi(rbSizeStr); err == nil && size > 0 {
		// round to next power of 2 if needed (for simplicity just expecting power of 2)
		cfg.RingBufferSize = size
	} else {
		cfg.RingBufferSize = 65536
	}

	latIntvStr := getEnvOrDefault("LATENCY_REPORT_INTERVAL", "30s")
	if d, err := time.ParseDuration(latIntvStr); err == nil {
		cfg.LatencyReportIntv = d
	} else {
		cfg.LatencyReportIntv = 30 * time.Second
	}

	redisSyncStr := getEnvOrDefault("REDIS_STATE_SYNC_INTERVAL", "5s")
	if d, err := time.ParseDuration(redisSyncStr); err == nil {
		cfg.RedisStateSync = d
	} else {
		cfg.RedisStateSync = 5 * time.Second
	}

	if cfg.FyersAppID == "" && cfg.Env != "test" {
		return nil, fmt.Errorf("FYERS_APP_ID environment variable is required")
	}

	return cfg, nil
}

func LoadTradingConfig(filePath string) (*TradingConfig, error) {
	data, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to read trading config file: %w", err)
	}

	cfg := &TradingConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse trading config file: %w", err)
	}

	if cfg.TradeDirection != "CE" && cfg.TradeDirection != "PE" {
		return nil, fmt.Errorf("invalid trade_direction: %s, must be CE or PE", cfg.TradeDirection)
	}

	return cfg, nil
}

func (tc *TradingConfig) Reload(filePath string) error {
	newCfg, err := LoadTradingConfig(filePath)
	if err != nil {
		return err
	}

	tc.mu.Lock()
	tc.TradeDirection = newCfg.TradeDirection
	tc.ExpiryWeek = newCfg.ExpiryWeek
	tc.CEOffsetPoints = newCfg.CEOffsetPoints
	tc.PEOffsetPoints = newCfg.PEOffsetPoints
	tc.LotSize = newCfg.LotSize
	tc.NumLots = newCfg.NumLots
	tc.ProductType = newCfg.ProductType
	tc.OrderType = newCfg.OrderType
	tc.StrikeChangeConfirmSeconds = newCfg.StrikeChangeConfirmSeconds
	tc.StrikeChangeExtraSeconds = newCfg.StrikeChangeExtraSeconds
	tc.MaxPositionValue = newCfg.MaxPositionValue
	tc.MaxDailyLoss = newCfg.MaxDailyLoss
	tc.TradingStartTime = newCfg.TradingStartTime
	tc.TradingEndTime = newCfg.TradingEndTime
	tc.mu.Unlock()

	return nil
}

// Thread-safe getters
func (tc *TradingConfig) GetTradeDirection() string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.TradeDirection
}

func (tc *TradingConfig) GetExpiryWeek() int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.ExpiryWeek
}

func (tc *TradingConfig) GetCEOffsetPoints() int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.CEOffsetPoints
}

func (tc *TradingConfig) GetPEOffsetPoints() int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.PEOffsetPoints
}

func (tc *TradingConfig) GetLotSize() int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.LotSize
}

func (tc *TradingConfig) GetNumLots() int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.NumLots
}

func (tc *TradingConfig) GetProductType() string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.ProductType
}

func (tc *TradingConfig) GetOrderType() string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.OrderType
}

func (tc *TradingConfig) GetStrikeChangeConfirmSeconds() int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.StrikeChangeConfirmSeconds
}

func (tc *TradingConfig) GetStrikeChangeExtraSeconds() int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.StrikeChangeExtraSeconds
}

func (tc *TradingConfig) GetMaxConfirmSeconds() int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	if tc.MaxConfirmSeconds <= 0 {
		return 360 // safe default: 6 minutes absolute max
	}
	return tc.MaxConfirmSeconds
}

func (tc *TradingConfig) GetMaxPositionValue() int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.MaxPositionValue
}

func (tc *TradingConfig) GetMaxDailyLoss() int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.MaxDailyLoss
}

func (tc *TradingConfig) GetTradingStartTime() string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.TradingStartTime
}

func (tc *TradingConfig) GetTradingEndTime() string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.TradingEndTime
}

func getEnvOrDefault(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}
