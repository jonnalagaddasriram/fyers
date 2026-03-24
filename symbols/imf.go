package symbols

import (
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	fyersIMFURL = "https://public.fyers.in/sym_details/NSE_FO.csv"
	cacheFileName = "NSE_FO.csv"
)

// IMFManager is responsible for downloading and parsing the Fyers Instrument Master File.
type IMFManager struct {
	logger *slog.Logger
	mu     sync.RWMutex

	// Sorted list of valid weekly/monthly expiry dates for Nifty
	niftyExpiryDates []time.Time
	// Full set of valid Nifty option symbols from the IMF — used for quick validation.
	niftySymbols map[string]struct{}
	// Strikes available per expiry + option type.
	niftyStrikesByExpiry map[string][]int
	// Exact 5-char expiry code used by Fyers for this date (handles YYMdd vs YYMMM)
	niftyExpiryCodes map[string]string
	cacheDirPath     string
}

// NewIMFManager creates a new instance.
func NewIMFManager(logger *slog.Logger, cacheDirPath string) *IMFManager {
	return &IMFManager{
		logger:               logger,
		niftyExpiryDates:     make([]time.Time, 0),
		niftySymbols:         make(map[string]struct{}),
		niftyStrikesByExpiry: make(map[string][]int),
		niftyExpiryCodes:     make(map[string]string),
		cacheDirPath:         cacheDirPath,
	}
}

// Initialize downloads the IMF if necessary and parses the valid Nifty expiry dates.
func (m *IMFManager) Initialize() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	err := m.ensureIMFCached()
	if err != nil {
		return fmt.Errorf("failed to ensure IMF is cached: %w", err)
	}

	err = m.parseNiftyExpiries()
	if err != nil {
		return fmt.Errorf("failed to parse Nifty expiries from IMF: %w", err)
	}

	return nil
}

// GetNiftyExpiries returns a thread-safe copy of the sorted valid Nifty expiry dates.
func (m *IMFManager) GetNiftyExpiries() []time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cp := make([]time.Time, len(m.niftyExpiryDates))
	copy(cp, m.niftyExpiryDates)
	return cp
}

// SymbolExists reports whether the given symbol string exists in the current IMF data.
// Returns false for any symbol not published by Fyers (wrong strike, wrong expiry, etc).
func (m *IMFManager) SymbolExists(symbol string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.niftySymbols[symbol]
	return ok
}

// FindNearestStrike returns the closest available strike price to targetStrike for
// the given expiry date and option type ("CE" or "PE").
func (m *IMFManager) FindNearestStrike(expiry time.Time, targetStrike int, optType string) (int, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := expiry.Format("2006-01-02") + ":" + optType
	strikes, ok := m.niftyStrikesByExpiry[key]
	if !ok || len(strikes) == 0 {
		return 0, false
	}

	// Binary search for insertion point, then compare neighbours
	lo, hi := 0, len(strikes)-1
	for lo < hi {
		mid := (lo + hi) / 2
		if strikes[mid] < targetStrike {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	// lo is the first element >= targetStrike
	best := strikes[lo]
	if lo > 0 {
		below := strikes[lo-1]
		if abs(below-targetStrike) < abs(best-targetStrike) {
			best = below
		}
	}
	return best, true
}

// GetExpiryCode returns the exact 5-character string Fyers uses for this date
// (e.g., "26324" for weekly or "26MAR" for monthly).
func (m *IMFManager) GetExpiryCode(expiry time.Time) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.niftyExpiryCodes[expiry.Format("2006-01-02")]
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func istLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		// Fallback: UTC+5:30
		return time.FixedZone("IST", 5*3600+30*60)
	}
	return loc
}

func (m *IMFManager) ensureIMFCached() error {
	path := filepath.Join(m.cacheDirPath, cacheFileName)
	ist := istLocation()

	// Refresh if file doesn't exist OR if it was written on a previous IST calendar day.
	// This guarantees fresh data every trading morning regardless of the machine's timezone
	// (EC2 runs in UTC — a 12h rolling check would break across midnight IST).
	info, err := os.Stat(path)
	if err == nil {
		nowIST := time.Now().In(ist)
		fileIST := info.ModTime().In(ist)
		sameDay := nowIST.Year() == fileIST.Year() &&
			nowIST.Month() == fileIST.Month() &&
			nowIST.Day() == fileIST.Day()
		if sameDay {
			m.logger.Info("Using cached IMF file (same IST trading day)",
				"path", path, "age", time.Since(info.ModTime()).Round(time.Minute).String())
			return nil
		}
		m.logger.Info("Cached IMF file is from a previous IST trading day — refreshing", "path", path)
	} else if !os.IsNotExist(err) {
		return err
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(m.cacheDirPath, 0755); err != nil {
		return err
	}

	m.logger.Info("Downloading Fyers IMF...", "url", fyersIMFURL)
	resp, err := http.Get(fyersIMFURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	size, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	m.logger.Info("IMF downloaded successfully", "bytes", size)
	return nil
}

// parseNiftyExpiries reads the CSV and extracts unique, future expiry dates for Nifty options.
// Example CSV Line: (Tokens vary by segment)
// 1011141300898,NSE:NIFTY24APR22000CE,14,0,22000.0,0,10,21703,11-Apr-2024,1,...
// Fyers format: FyersToken, SymbolDetails, InstrumentType, MinimumLotSize, TickSize, ISIN, TradingSession, LastUpdate... (The schema is slightly varying so we rely on Expiry seconds if possible, or string parsing)
// Per Fyers docs, column 13 (0-indexed) or similar contains the Expiry Date as Unix Timestamp.
// However, the most robust way across their schema changes is to parse the `SymbolDetails` string.
// Example: NSE:NIFTY24O1722000CE -> Year 24, Month O (Oct), Day 17.
func (m *IMFManager) parseNiftyExpiries() error {
	path := filepath.Join(m.cacheDirPath, cacheFileName)
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	ist := istLocation()
	uniqueDates := make(map[string]time.Time)
	nowIST := time.Now().In(ist)
	// Keep today's expiry (for expiry-day morning trading) and all future ones.
	todayStartIST := time.Date(nowIST.Year(), nowIST.Month(), nowIST.Day(), 0, 0, 0, 0, ist)

	lineCount := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			m.logger.Warn("IMF CSV parse error on a line, skipping", "err", err)
			continue
		}
		lineCount++

		if len(record) > 10 {
			symbol := record[9]
			// We only care about Nifty Options (CE or PE)
			if strings.HasPrefix(symbol, "NSE:NIFTY") && (strings.HasSuffix(symbol, "CE") || strings.HasSuffix(symbol, "PE")) {
				// Prevent matching NSE:NIFTYNXT50, NSE:NIFTYMID50, etc.
				// Genuine NIFTY 50 options are immediately followed by the 2-digit year (e.g., '2' for 2024/2025/2026)
				if len(symbol) > 9 && symbol[9] >= '0' && symbol[9] <= '9' {
					// Column 8 = Unix expiry timestamp (always UTC from Fyers)
					expiryUnix, err := strconv.ParseInt(record[8], 10, 64)
					if err != nil {
						continue
					}

				// Convert to IST for correct date comparison regardless of machine timezone
					expiryIST := time.Unix(expiryUnix, 0).In(ist)
					expiryDayIST := time.Date(expiryIST.Year(), expiryIST.Month(), expiryIST.Day(), 0, 0, 0, 0, ist)

					if !expiryDayIST.Before(todayStartIST) {
						dateKey := expiryDayIST.Format("2006-01-02")
						uniqueDates[dateKey] = expiryDayIST
						// Cache symbol for O(1) lookup
						m.niftySymbols[symbol] = struct{}{}

						// Extract strike from symbol and add to per-expiry strike index.
						// Symbol format: NSE:NIFTY{YYMDD}{STRIKE}{CE|PE}
						// Strip prefix "NSE:NIFTY" and suffix "CE"/"PE" then expiry code (5 chars)
						var optType string
						if strings.HasSuffix(symbol, "CE") {
							optType = "CE"
						} else {
							optType = "PE"
						}
						inner := strings.TrimPrefix(symbol, "NSE:NIFTY")
						inner = strings.TrimSuffix(strings.TrimSuffix(inner, "CE"), "PE")
						// inner = expiryCode (5 chars) + strike; expiry code is always 5 chars (either YYMdd or YYMMM)
						if len(inner) > 5 {
							expiryCode := inner[:5]
							m.niftyExpiryCodes[dateKey] = expiryCode
							
							strikeStr := inner[5:]
							if strike, err := strconv.Atoi(strikeStr); err == nil {
								strikesKey := dateKey + ":" + optType
								m.niftyStrikesByExpiry[strikesKey] = append(m.niftyStrikesByExpiry[strikesKey], strike)
							}
						}
					}
				}
			}
		}
	}

	// Sort each expiry's strike list for binary-search in FindNearestStrike
	for key := range m.niftyStrikesByExpiry {
		sort.Ints(m.niftyStrikesByExpiry[key])
	}

	m.logger.Info("Parsed IMF lines", "total_lines", lineCount)

	m.niftyExpiryDates = make([]time.Time, 0, len(uniqueDates))
	for _, d := range uniqueDates {
		m.niftyExpiryDates = append(m.niftyExpiryDates, d)
	}

	// Sort nearest → furthest
	sort.Slice(m.niftyExpiryDates, func(i, j int) bool {
		return m.niftyExpiryDates[i].Before(m.niftyExpiryDates[j])
	})

	m.logger.Info("Extracted Nifty expiries", "count", len(m.niftyExpiryDates))
	if len(m.niftyExpiryDates) > 0 {
		m.logger.Info("Next upcoming expiry", "date", m.niftyExpiryDates[0].Format("2006-01-02 (Monday)"))
	}

	return nil
}
