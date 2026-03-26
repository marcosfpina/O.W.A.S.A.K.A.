package stream

import (
	"sync"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
)

// circularBuffer is a fixed-capacity, thread-safe ring buffer for NetworkEvents
type circularBuffer struct {
	mu       sync.RWMutex
	data     []models.NetworkEvent
	head     int
	count    int
	capacity int
}

func newCircularBuffer(capacity int) *circularBuffer {
	if capacity <= 0 {
		capacity = 10000
	}
	return &circularBuffer{
		data:     make([]models.NetworkEvent, capacity),
		capacity: capacity,
	}
}

// push adds an event, overwriting the oldest if full
func (b *circularBuffer) push(e models.NetworkEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.data[b.head] = e
	b.head = (b.head + 1) % b.capacity
	if b.count < b.capacity {
		b.count++
	}
}

// len returns the number of stored events
func (b *circularBuffer) len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.count
}

// snapshot returns a copy of all stored events, oldest first
func (b *circularBuffer) snapshot() []models.NetworkEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]models.NetworkEvent, b.count)
	start := (b.head - b.count + b.capacity) % b.capacity
	for i := 0; i < b.count; i++ {
		out[i] = b.data[(start+i)%b.capacity]
	}
	return out
}
