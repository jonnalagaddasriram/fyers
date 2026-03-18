package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"
)

// roundToATM rounds the price to the nearest 50, matching ComputeATMStrike in symbols/nifty.go
func roundToATM(price float64) int {
	return int(math.Round(price/50.0)) * 50
}

// TickEvent mirrors the structure found in the ticks.txt
type TickEvent struct {
	Symbol        string  `json:"Symbol"`
	LTP           float64 `json:"LTP"`
	ExchTimestamp int64   `json:"ExchTimestamp"` // in seconds
	RecvTimestamp int64   `json:"RecvTimestamp"` // in nanoseconds
}

func main() {
	// Look for ticks.txt in the current directory (since we run it from the fyers root)
	filePath := "ticks.txt"
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening file %s: %v\n", filePath, err)
		os.Exit(1)
	}
	defer file.Close()

	// Load India Standard Time (IST)
	istLoc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		fmt.Printf("Warning: Could not load Asia/Kolkata timezone, using local time: %v\n", err)
		istLoc = time.Local
	}

	var totalRecords int64
	var totalLatency time.Duration
	var minLatency time.Duration = time.Hour
	var maxLatency time.Duration = -1
	var skippedRecords int64
	var symbolCounts = make(map[string]int64)

	// Inter-arrival tracking
	var lastRecvTime = make(map[string]time.Time)
	var totalInterArrivalTime time.Duration
	var minInterArrivalTime time.Duration = time.Hour
	var maxInterArrivalTime time.Duration = -1
	var interArrivalCount int64

	// Time window tracking
	var exchStartTime time.Time
	var exchEndTime time.Time
	var localStartTime time.Time
	var localEndTime time.Time

	// ATM Switch tracking (from Nifty index ticks)
	type ATMSwitch struct {
		Time   time.Time
		ATM    int
		GapSec float64 // seconds since last switch
	}
	var atmSwitches []ATMSwitch
	var currentATM int
	var lastSwitchTime time.Time

	// Since exchange timestamp is in seconds and receive timestamp in nanoseconds,
	// we will calculate latency down to the millisecond.
	// Note: Because Fyers API provides Exchange timestamp only to the second resolution (e.g., 1773654310),
	// this latency calculation is an approximation bound by 1-second precision on the Exchange side.

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var tick TickEvent
		err := json.Unmarshal([]byte(line), &tick)
		if err != nil {
			fmt.Printf("Error parsing JSON on line %d: %v\n", totalRecords+skippedRecords+1, err)
			skippedRecords++
			continue
		}

		totalRecords++
		symbolCounts[tick.Symbol]++

		// Detect ATM switches from Nifty index ticks
		if tick.Symbol == "NSE:NIFTY50-INDEX" && tick.LTP > 0 && tick.ExchTimestamp > 0 {
			newATM := roundToATM(tick.LTP)
			if newATM != currentATM {
				t := time.Unix(tick.ExchTimestamp, 0)
				var gapSec float64
				if !lastSwitchTime.IsZero() {
					gapSec = t.Sub(lastSwitchTime).Seconds()
				}
				atmSwitches = append(atmSwitches, ATMSwitch{Time: t, ATM: newATM, GapSec: gapSec})
				currentATM = newATM
				lastSwitchTime = t
			}
		}

		if tick.ExchTimestamp > 0 && tick.RecvTimestamp > 0 {
			exchTime := time.Unix(tick.ExchTimestamp, 0)
			recvTime := time.Unix(0, tick.RecvTimestamp)

			// Update total time windows
			if exchStartTime.IsZero() || exchTime.Before(exchStartTime) {
				exchStartTime = exchTime
			}
			if exchTime.After(exchEndTime) {
				exchEndTime = exchTime
			}
			
			if localStartTime.IsZero() || recvTime.Before(localStartTime) {
				localStartTime = recvTime
			}
			if recvTime.After(localEndTime) {
				localEndTime = recvTime
			}

			// Calculate Inter-arrival time for this symbol
			if lastTime, exists := lastRecvTime[tick.Symbol]; exists {
				interArrivalTime := recvTime.Sub(lastTime)
				if interArrivalTime > 0 { // Ignore out of order or simultaneous ticks for avg
					totalInterArrivalTime += interArrivalTime
					interArrivalCount++
					
					if interArrivalTime < minInterArrivalTime {
						minInterArrivalTime = interArrivalTime
					}
					if interArrivalTime > maxInterArrivalTime {
						maxInterArrivalTime = interArrivalTime
					}
				}
			}
			lastRecvTime[tick.Symbol] = recvTime

			latency := recvTime.Sub(exchTime)
			
			totalLatency += latency
			if latency < minLatency {
				minLatency = latency
			}
			if latency > maxLatency {
				maxLatency = latency
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
	}

	if totalRecords == 0 {
		fmt.Println("No valid records generated/found in ticks.txt")
		return
	}

	fmt.Println("==================================================")
	fmt.Println(" Tick Data Analytics Report ")
	fmt.Println("==================================================")
	fmt.Printf("Total Records Processed : %d\n", totalRecords)
	
	if !exchStartTime.IsZero() {
		fmt.Printf("Exchange Time Window    : %s to %s (IST)\n", exchStartTime.In(istLoc).Format("15:04:05"), exchEndTime.In(istLoc).Format("15:04:05"))
		fmt.Printf("Local App Time Window   : %s to %s (IST) (Duration: %v)\n", 
			localStartTime.In(istLoc).Format("15:04:05"), 
			localEndTime.In(istLoc).Format("15:04:05"),
			localEndTime.Sub(localStartTime).Round(time.Second))
	}

	if skippedRecords > 0 {
		fmt.Printf("Skipped Invalid Records : %d\n", skippedRecords)
	}

	// ATM Switch report
	if len(atmSwitches) > 0 {
		fmt.Printf("\n--- ATM Strike Switches (from Nifty Index Ticks) ---\n")
		for i, sw := range atmSwitches {
			if i == 0 {
				fmt.Printf("  [%s IST]  Initial ATM : %d\n", sw.Time.In(istLoc).Format("15:04:05"), sw.ATM)
			} else {
				fmt.Printf("  [%s IST]  ATM → %d  (+%.0fs since last)\n", sw.Time.In(istLoc).Format("15:04:05"), sw.ATM, sw.GapSec)
			}
		}
		fmt.Printf("  Total ATM Switches : %d\n", len(atmSwitches)-1)
	}

	niftyCount := symbolCounts["NSE:NIFTY50-INDEX"]
	var baseCount float64
	if niftyCount > 0 {
		baseCount = float64(niftyCount)
		fmt.Printf("\n--- Breakdown by Symbol (Relative to Nifty Index %d ticks) ---\n", niftyCount)
	} else {
		baseCount = float64(totalRecords)
		fmt.Println("\n--- Breakdown by Symbol (Relative to Total Records) ---")
	}

	for sym, count := range symbolCounts {
		fmt.Printf("  %s : %d ticks (%.1f%%)\n", sym, count, float64(count)/baseCount*100)
	}

	fmt.Println("\n--- Latency Metrics ---")
	fmt.Println(" * Note: Exchange time is provided with 1-second precision.")
	fmt.Printf("Average Latency         : %v\n", time.Duration(int64(totalLatency)/totalRecords))
	if minLatency != time.Hour {
		fmt.Printf("Minimum Latency         : %v\n", minLatency)
		fmt.Printf("Maximum Latency         : %v\n", maxLatency)
	}

	fmt.Println("\n--- Frequency / Throughput ---")
	if interArrivalCount > 0 {
		avgInterArrival := time.Duration(int64(totalInterArrivalTime) / interArrivalCount)
		fmt.Printf("Avg Time Between Ticks  : %v\n", avgInterArrival)
		
		if minInterArrivalTime != time.Hour {
			fmt.Printf("Min Time Between Ticks  : %v\n", minInterArrivalTime)
			fmt.Printf("Max Time Between Ticks  : %v\n", maxInterArrivalTime)
		}

		var ticksPerSecond float64 = 0
		if avgInterArrival.Seconds() > 0 {
			ticksPerSecond = 1.0 / avgInterArrival.Seconds()
		}
		fmt.Printf("Implied Updates/Sec     : %.2f ticks/sec per symbol\n", ticksPerSecond)
	}

	fmt.Println("==================================================")
}
