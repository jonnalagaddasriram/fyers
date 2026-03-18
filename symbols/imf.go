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
	cacheDirPath     string
}

// NewIMFManager creates a new instance.
func NewIMFManager(logger *slog.Logger, cacheDirPath string) *IMFManager {
	return &IMFManager{
		logger:           logger,
		niftyExpiryDates: make([]time.Time, 0),
		cacheDirPath:     cacheDirPath,
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

	// Return a copy to prevent external modification
	cp := make([]time.Time, len(m.niftyExpiryDates))
	copy(cp, m.niftyExpiryDates)
	return cp
}

func (m *IMFManager) ensureIMFCached() error {
	path := filepath.Join(m.cacheDirPath, cacheFileName)

	// Check if file exists and is less than 12 hours old
	info, err := os.Stat(path)
	if err == nil {
		if time.Since(info.ModTime()) < 12*time.Hour {
			m.logger.Info("Using cached IMF file", "path", path, "age", time.Since(info.ModTime()).String())
			return nil
		}
		m.logger.Info("Cached IMF file is older than 12 hours, replacing", "path", path)
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
	// We might have different number of fields depending on updates
	reader.FieldsPerRecord = -1 

	uniqueDates := make(map[string]time.Time)
	now := time.Now()
	// To avoid issues with intraday expiry processing, we'll keep today's expiry as well.
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

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

		// Example schema line:
		// 101126033051701,BANKNIFTY 30 Mar 26 FUT,11,30,0.2,,0915-1530|1815-1915:,2026-03-30,1774864800,NSE:BANKNIFTY26MARFUT,10,11,51701,BANKNIFTY,26009,-1.0,XX,101000000026009,None,0,0.0
		// Column 8 (index 8) -> 1774864800 (Expiry Timestamp Unix)
		// Column 9 (index 9) -> NSE:BANKNIFTY26MARFUT (Symbol string)

		if len(record) > 10 {
			symbol := record[9]
			// We only care about Nifty Options
			if strings.HasPrefix(symbol, "NSE:NIFTY") && (strings.HasSuffix(symbol, "CE") || strings.HasSuffix(symbol, "PE")) {
				
				// Expiry timestamp
				expiryTsStr := record[8]
				expiryUnix, err := strconv.ParseInt(expiryTsStr, 10, 64)
				if err != nil {
					continue
				}

				expiryTime := time.Unix(expiryUnix, 0).Local()
				// Truncate to day start for unique-ing
				expiryDay := time.Date(expiryTime.Year(), expiryTime.Month(), expiryTime.Day(), 0, 0, 0, 0, time.Local)
				
				if !expiryDay.Before(todayStart) {
					dateKey := expiryDay.Format("2006-01-02")
					uniqueDates[dateKey] = expiryDay
				}
			}
		}
	}

	m.logger.Info("Parsed IMF lines", "total_lines", lineCount)

	m.niftyExpiryDates = make([]time.Time, 0, len(uniqueDates))
	for _, d := range uniqueDates {
		m.niftyExpiryDates = append(m.niftyExpiryDates, d)
	}

	// Sort dates from nearest to furthest
	sort.Slice(m.niftyExpiryDates, func(i, j int) bool {
		return m.niftyExpiryDates[i].Before(m.niftyExpiryDates[j])
	})

	m.logger.Info("Extracted Nifty expiries", "count", len(m.niftyExpiryDates))
	if len(m.niftyExpiryDates) > 0 {
		m.logger.Info("Next upcoming expiry", "date", m.niftyExpiryDates[0].Format("2006-01-02"))
	}

	return nil
}
