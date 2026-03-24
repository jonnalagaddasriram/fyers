package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"time"
)

// roundToATM rounds the price to the nearest 50
func roundToATM(price float64) int {
	return int(math.Round(price/50.0)) * 50
}

type TickEvent struct {
	Symbol        string  `json:"Symbol"`
	LTP           float64 `json:"LTP"`
	ExchTimestamp int64   `json:"ExchTimestamp"`
	RecvTimestamp int64   `json:"RecvTimestamp"`
}

type BucketStats struct {
	TotalRecords      int64
	TotalLatency      time.Duration
	SymbolCounts      map[string]int64
	InterArrivalCount int64
	TotalInterArrival time.Duration
}

func main() {
	// The interval for chunking the analysis (e.g., 15 minutes)
	chunkMinutes := 15.0 

	filePath := "ticks.txt"
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening file %s: %v\n", filePath, err)
		os.Exit(1)
	}
	defer file.Close()

	istLoc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		fmt.Printf("Warning: Could not load Asia/Kolkata timezone, using local time: %v\n", err)
		istLoc = time.Local
	}

	// Global variables 
	var totalRecords int64
	var totalLatency time.Duration
	var minLatency time.Duration = time.Hour
	var maxLatency time.Duration = -1
	var skippedRecords int64
	var symbolCounts = make(map[string]int64)

	var lastRecvTime = make(map[string]time.Time)
	var totalInterArrivalTime time.Duration
	var minInterArrivalTime time.Duration = time.Hour
	var maxInterArrivalTime time.Duration = -1
	var interArrivalCount int64

	var localStartTime time.Time
	var localEndTime time.Time

	// Batch analytics tracking per symbol
	type BatchStats struct {
		BatchSizes       []int
		TimeGaps         []time.Duration
		CurrentBatchSize int
		LastBatchTime    time.Time
	}
	batchInfo := make(map[string]*BatchStats)

	type ATMSwitch struct {
		Time   time.Time
		ATM    int
		GapSec float64
	}
	var atmSwitches []ATMSwitch
	var currentATM int
	var lastSwitchTime time.Time

	// Time Buckets
	buckets := make(map[int64]*BucketStats)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var tick TickEvent
		err := json.Unmarshal([]byte(line), &tick)
		if err != nil {
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

			// Assign to 15-minute bucket
			bucketKey := exchTime.Truncate(time.Duration(chunkMinutes) * time.Minute).Unix()
			b, ok := buckets[bucketKey]
			if !ok {
				b = &BucketStats{SymbolCounts: make(map[string]int64)}
				buckets[bucketKey] = b
			}

			b.TotalRecords++
			b.SymbolCounts[tick.Symbol]++

			if localStartTime.IsZero() || recvTime.Before(localStartTime) {
				localStartTime = recvTime
			}
			if recvTime.After(localEndTime) {
				localEndTime = recvTime
			}

			// Batch tracking
			if _, ok := batchInfo[tick.Symbol]; !ok {
				batchInfo[tick.Symbol] = &BatchStats{}
			}
			bStats := batchInfo[tick.Symbol]

			if bStats.LastBatchTime.IsZero() {
				bStats.LastBatchTime = recvTime
				bStats.CurrentBatchSize = 1
			} else {
				// Ticks arriving within 10ms of each other belong to the same network batch
				gap := recvTime.Sub(bStats.LastBatchTime)
				if gap < 10*time.Millisecond {
					bStats.CurrentBatchSize++
				} else {
					bStats.BatchSizes = append(bStats.BatchSizes, bStats.CurrentBatchSize)
					bStats.TimeGaps = append(bStats.TimeGaps, gap)
					bStats.CurrentBatchSize = 1
					bStats.LastBatchTime = recvTime
				}
			}

			// Calculate Inter-arrival time
			if lastTime, exists := lastRecvTime[tick.Symbol]; exists {
				interArrivalTime := recvTime.Sub(lastTime)
				if interArrivalTime > 0 {
					totalInterArrivalTime += interArrivalTime
					interArrivalCount++
					b.TotalInterArrival += interArrivalTime
					b.InterArrivalCount++
					
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
			b.TotalLatency += latency

			if latency < minLatency {
				minLatency = latency
			}
			if latency > maxLatency {
				maxLatency = latency
			}
		}
	}

	// Flush final batches
	for _, bStats := range batchInfo {
		if bStats.CurrentBatchSize > 0 {
			bStats.BatchSizes = append(bStats.BatchSizes, bStats.CurrentBatchSize)
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
	fmt.Println(" Tick Data Analytics Report (Global) ")
	fmt.Println("==================================================")
	fmt.Printf("Total Records Processed : %d\n", totalRecords)
	
	if !localStartTime.IsZero() {
		fmt.Printf("Local Test Time Window  : %s to %s (IST)\n", localStartTime.In(istLoc).Format("15:04:05"), localEndTime.In(istLoc).Format("15:04:05"))
		fmt.Printf("Total Test Duration     : %s\n", localEndTime.Sub(localStartTime).Round(time.Second))
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
		
		// Print batch stats if it's an option (has batch sizes)
		bStats := batchInfo[sym]
		if bStats != nil && len(bStats.BatchSizes) > 0 {
			var totalBatchSize int
			for _, bs := range bStats.BatchSizes {
				totalBatchSize += bs
			}
			avgBatch := float64(totalBatchSize) / float64(len(bStats.BatchSizes))

			var totalGap time.Duration
			var minGap time.Duration = time.Hour
			var maxGap time.Duration = -1
			for _, g := range bStats.TimeGaps {
				totalGap += g
				if g < minGap {
					minGap = g
				}
				if g > maxGap {
					maxGap = g
				}
			}
			
			var avgGap time.Duration
			if len(bStats.TimeGaps) > 0 {
				avgGap = totalGap / time.Duration(len(bStats.TimeGaps))
			} else {
				minGap = 0
				maxGap = 0
			}

			fmt.Printf("    => Batches: %d | Avg per batch: %.2f ticks\n", len(bStats.BatchSizes), avgBatch)
			if len(bStats.TimeGaps) > 0 {
				fmt.Printf("    => Wait Gaps: Avg=%v | Min=%v | Max=%v\n", avgGap.Round(time.Millisecond), minGap.Round(time.Millisecond), maxGap.Round(time.Millisecond))
			}
		}
	}


	fmt.Println("\n--- Frequency / Throughput (Global) ---")
	if interArrivalCount > 0 {
		avgInterArrival := time.Duration(int64(totalInterArrivalTime) / interArrivalCount)
		fmt.Printf("Global Avg Time Between Ticks  : %v\n", avgInterArrival)

		var ticksPerSecond float64 = 0
		if avgInterArrival.Seconds() > 0 {
			ticksPerSecond = 1.0 / avgInterArrival.Seconds()
		}
		fmt.Printf("Global Implied Updates/Sec     : %.2f ticks/sec per symbol\n", ticksPerSecond)
	}

	fmt.Println("\n==================================================")
	fmt.Printf(" Time-Interval Breakdown (%.0f-Minute Buckets)\n", chunkMinutes)
	fmt.Println("==================================================")

	var keys []int64
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	for _, k := range keys {
		b := buckets[k]
		windowStart := time.Unix(k, 0).In(istLoc)
		windowEnd := windowStart.Add(time.Duration(chunkMinutes) * time.Minute)
		
		fmt.Printf("\n[%s - %s IST]\n", windowStart.Format("15:04"), windowEnd.Format("15:04"))
		fmt.Printf("  Total Ticks : %d\n", b.TotalRecords)
		
		if b.TotalRecords > 0 {
			avgLat := time.Duration(int64(b.TotalLatency) / b.TotalRecords)
			fmt.Printf("  Avg Latency : %v\n", avgLat)
		}

		if b.InterArrivalCount > 0 {
			avgArrival := time.Duration(int64(b.TotalInterArrival) / b.InterArrivalCount)
			updatesPerSec := 0.0
			if avgArrival.Seconds() > 0 {
				updatesPerSec = 1.0 / avgArrival.Seconds()
			}
			fmt.Printf("  Throughput  : %.2f ticks/sec\n", updatesPerSec)
		} else {
			fmt.Printf("  Throughput  : 0.00 ticks/sec\n")
		}
	}
	fmt.Println("\n==================================================")
}
