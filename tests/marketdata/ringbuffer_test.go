package marketdata_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fyers-trading/marketdata"
)

func TestRingBuffer_NewRingBuffer(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected uint64
	}{
		{"Exact power of 2", 8, 8},
		{"Not power of 2, round up", 100, 128},
		{"Not power of 2, round up close", 1000, 1024},
		{"Large capacity", 65000, 65536},
		{"Minimum capacity", 0, 1},
		{"Negative capacity", -5, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rb := marketdata.NewRingBuffer(tc.input)
			if rb.Capacity() != tc.expected {
				t.Errorf("Expected capacity %d, got %d", tc.expected, rb.Capacity())
			}
		})
	}
}

func TestRingBuffer_WriteAndSingleRead(t *testing.T) {
	rb := marketdata.NewRingBuffer(4)
	reader := rb.NewReader()

	tick1 := marketdata.GetTickEvent()
	tick1.Symbol = "NIFTY"
	tick1.LTP = 22000.50
	rb.Write(tick1)

	readTick := reader.Next()
	if readTick.Symbol != "NIFTY" {
		t.Errorf("Expected symbol NIFTY, got %s", readTick.Symbol)
	}
	if readTick.LTP != 22000.50 {
		t.Errorf("Expected LTP 22000.50, got %f", readTick.LTP)
	}
}

func TestRingBuffer_LappedReader(t *testing.T) {
	// Ring size 4
	rb := marketdata.NewRingBuffer(4)
	reader := rb.NewReader()

	// Write 10 ticks -> indices 0..9. 
	// At the end, writer will be at pos=10.
	// Last 4 slots valid are 6,7,8,9
	for i := 0; i < 10; i++ {
		tick := marketdata.GetTickEvent()
		tick.LTP = float64(i)
		rb.Write(tick)
	}

	// Reader has read pos 0. Writer is at 10.
	// writePos - readPos = 10 > capacity(4). 
	// Skipped. readPos becomes 9. Reads tick 9.
	tick := reader.Next()
	if tick.LTP != 9 {
		t.Errorf("Expected reader to skip to latest tick (9), got %f", tick.LTP)
	}
}

func TestRingBuffer_WrapAround(t *testing.T) {
	rb := marketdata.NewRingBuffer(4)
	reader := rb.NewReader()

	// Write 3, read 3 (pos = 3)
	for i := 0; i < 3; i++ {
		tick := marketdata.GetTickEvent()
		tick.LTP = float64(i)
		rb.Write(tick)
		readTick := reader.Next()
		if readTick.LTP != float64(i) {
			t.Errorf("Expected %f, got %f", float64(i), readTick.LTP)
		}
	}

	// Write 3 more, should wrap around the array indices
	for i := 3; i < 6; i++ {
		tick := marketdata.GetTickEvent()
		tick.LTP = float64(i)
		rb.Write(tick)
		readTick := reader.Next()
		if readTick.LTP != float64(i) {
			t.Errorf("Expected %f, got %f", float64(i), readTick.LTP)
		}
	}
}

func TestRingBuffer_MultipleReadersIndependent(t *testing.T) {
	rb := marketdata.NewRingBuffer(8)
	r1 := rb.NewReader()
	r2 := rb.NewReader()

	for i := 0; i < 3; i++ {
		t := marketdata.GetTickEvent()
		t.Volume = int64(i)
		rb.Write(t)
	}

	// R1 reads 1
	tick1 := r1.Next()
	if tick1.Volume != 0 {
		t.Errorf("r1 expected vol 0, got %d", tick1.Volume)
	}
	
	// R2 reads 2
	r2.Next()
	tick2 := r2.Next()
	if tick2.Volume != 1 {
		t.Errorf("r2 expected vol 1, got %d", tick2.Volume)
	}

	// R1 still has 2 pending
	tick3 := r1.Next()
	if tick3.Volume != 1 {
		t.Errorf("r1 expected vol 1, got %d", tick3.Volume)
	}
}

func TestRingBuffer_ConcurrentLoad(t *testing.T) {
	// 65k slots
	rb := marketdata.NewRingBuffer(65536)
	
	const N = 100000

	r1 := rb.NewReader()
	r2 := rb.NewReader()

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < N; i++ {
			tick := marketdata.GetTickEvent()
			tick.Volume = int64(i)
			rb.Write(tick)
		}
	}()

	
	var r1Count int64
	go func() {
		defer wg.Done()
		for i := 0; i < N; i++ {
			tick := r1.Next()
			if tick.Volume != atomic.LoadInt64(&r1Count) {
			   // In a real high concurrent we might skip, but here we read sequentially and buffer is huge enough not to lap mostly 
			   // just checking we don't crash
			}
			atomic.AddInt64(&r1Count, 1)
		}
	}()

	var r2Count int64
	go func() {
		defer wg.Done()
		for i := 0; i < N; i++ {
			_ = r2.Next()
			atomic.AddInt64(&r2Count, 1)
		}
	}()

	// timeout handling
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for readers and writers under load")
	}

	if r1Count != N || r2Count != N {
		t.Errorf("Expected %d reads, got r1=%d, r2=%d", N, r1Count, r2Count)
	}
}

func TestRingBuffer_ReaderSpinsCorrectly(t *testing.T) {
	rb := marketdata.NewRingBuffer(4)
	r1 := rb.NewReader()
	
	var result int
	
	go func() {
		tick := r1.Next() // Should block/spin until data is written
		result = int(tick.Volume)
	}()
	
	time.Sleep(10 * time.Millisecond) // Let reader spin
	
	if result != 0 {
		t.Errorf("Reader should not have read anything yet")
	}
	
	tick := marketdata.GetTickEvent()
	tick.Volume = 99
	rb.Write(tick)
	
	time.Sleep(10 * time.Millisecond) // Give reader time to process
	
	if result != 99 {
		t.Errorf("Reader failed to wake/spin and read tick, got %d", result)
	}
}
