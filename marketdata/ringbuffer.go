package marketdata

import (
	"runtime"
	"sync/atomic"
)

// RingBuffer is a lock-free Single-Producer, Multiple-Consumer (SPMC) ring buffer.
// It is designed for microsecond-latency tick ingestion.
type RingBuffer struct {
	buffer   []*TickEvent
	capacity uint64
	mask     uint64
	closed   uint32

	// CPU Cache Line padding (64 bytes) to prevent False Sharing 
	// between the producer's writePos and the consumer's readPos.
	_        [7]uint64
	writePos uint64 // Modified ONLY by the single Producer
}

// NewRingBuffer creates a new lock-free ring buffer
func NewRingBuffer(size int) *RingBuffer {
	if size <= 0 {
		size = 1
	}
	// Round up to next power of two for fast bitmask modulo
	capacity := uint64(1)
	for capacity < uint64(size) {
		capacity <<= 1
	}

	return &RingBuffer{
		buffer:   make([]*TickEvent, capacity),
		capacity: capacity,
		mask:     capacity - 1,
		writePos: 0,
		closed:   0,
	}
}

// Close signals all waiting and future readers that no more data will arrive.
func (rb *RingBuffer) Close() {
	atomic.StoreUint32(&rb.closed, 1)
}

// Write publishes a tick to the ring buffer. Called by exactly ONE goroutine (producer).
func (rb *RingBuffer) Write(tick *TickEvent) {
	pos := atomic.LoadUint64(&rb.writePos)
	idx := pos & rb.mask
	rb.buffer[idx] = tick
	atomic.StoreUint64(&rb.writePos, pos+1) // publish the new tail location
}

// Reader is a cursor into the RingBuffer
type Reader struct {
	rb      *RingBuffer

	// CPU Cache Line padding (64 bytes) to prevent False Sharing
	// between different Readers updating their own readPos.
	_       [7]uint64
	readPos uint64
}

// NewReader creates a new independent consumer cursor
func (rb *RingBuffer) NewReader() *Reader {
	return &Reader{
		rb:      rb,
		readPos: atomic.LoadUint64(&rb.writePos), // start from the current head
	}
}

// Next reads the next available tick.
// If reader is caught up with writer, it spin-waits using Gosched.
// If reader is too far behind (writer lapped the reader), it skips to latest.
func (r *Reader) Next() *TickEvent {
	for {
		writePos := atomic.LoadUint64(&r.rb.writePos)

		if r.readPos == writePos {
			if atomic.LoadUint32(&r.rb.closed) == 1 {
				return nil // Buffer empty and closed, signal reader to exit
			}
			// Buffered caught up, spin wait
			runtime.Gosched()
			continue
		}

		if writePos-r.readPos > r.rb.capacity {
			// Writer lapped the reader. Skip to latest to avoid stale data.
			// This violates exact once delivery but prioritizes low latency.
			r.readPos = writePos - 1
		}

		idx := r.readPos & r.rb.mask
		tick := r.rb.buffer[idx]
		r.readPos++
		return tick
	}
}

// Capacity returns the size of the ring buffer
func (rb *RingBuffer) Capacity() uint64 {
	return rb.capacity
}
