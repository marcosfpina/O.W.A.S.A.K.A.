package stream

import (
	"sync"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
)

const (
	bucketDuration = time.Minute // each bucket covers 1 minute
	totalBuckets   = 15          // covers the 15-minute window
)

// bucket holds per-IP, per-type event counts for a 1-minute slice
type bucket struct {
	ts     time.Time
	counts map[string]map[models.EventType]int // ip → type → count
}

func newBucket(ts time.Time) *bucket {
	return &bucket{
		ts:     ts,
		counts: make(map[string]map[models.EventType]int),
	}
}

func (b *bucket) record(ip string, t models.EventType) {
	if _, ok := b.counts[ip]; !ok {
		b.counts[ip] = make(map[models.EventType]int)
	}
	b.counts[ip][t]++
}

// WindowStats holds aggregated counts for a given IP across sliding windows
type WindowStats struct {
	IP       string `json:"ip"`
	Count1m  int    `json:"count_1m"`
	Count5m  int    `json:"count_5m"`
	Count15m int    `json:"count_15m"`
	Rate1m   float64 `json:"rate_1m"` // events per second over 1m window
}

// windower maintains 15 time-bucketed counters for sliding window analysis
type windower struct {
	mu      sync.RWMutex
	buckets []*bucket // ring of totalBuckets buckets
	current int       // index of the active bucket
}

func newWindower() *windower {
	now := time.Now().Truncate(bucketDuration)
	buckets := make([]*bucket, totalBuckets)
	for i := range buckets {
		buckets[i] = newBucket(now)
	}
	return &windower{buckets: buckets}
}

// record adds an IP+type observation into the current bucket, rotating if needed
func (w *windower) record(ip string, t models.EventType) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now().Truncate(bucketDuration)
	cur := w.buckets[w.current]

	// Rotate to a new bucket if the minute has ticked
	if now.After(cur.ts) {
		w.current = (w.current + 1) % totalBuckets
		w.buckets[w.current] = newBucket(now)
	}

	w.buckets[w.current].record(ip, t)
}

// stats returns WindowStats for a given IP, summing across N most-recent buckets
func (w *windower) stats(ip string) WindowStats {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var c1, c5, c15 int
	for i := 0; i < totalBuckets; i++ {
		idx := (w.current - i + totalBuckets) % totalBuckets
		b := w.buckets[idx]
		if typeCounts, ok := b.counts[ip]; ok {
			total := 0
			for _, n := range typeCounts {
				total += n
			}
			c15 += total
			if i < 5 {
				c5 += total
			}
			if i < 1 {
				c1 += total
			}
		}
	}

	rate := float64(c1) / 60.0
	return WindowStats{IP: ip, Count1m: c1, Count5m: c5, Count15m: c15, Rate1m: rate}
}

// topIPs returns the N IPs with the most events in the 5m window
func (w *windower) topIPs(n int) []WindowStats {
	w.mu.RLock()
	totals := make(map[string]int)
	for i := 0; i < 5 && i < totalBuckets; i++ {
		idx := (w.current - i + totalBuckets) % totalBuckets
		for ip, typeCounts := range w.buckets[idx].counts {
			for _, cnt := range typeCounts {
				totals[ip] += cnt
			}
		}
	}
	w.mu.RUnlock()

	// Collect into slice, simple insertion sort for small N
	result := make([]WindowStats, 0, len(totals))
	for ip := range totals {
		result = append(result, w.stats(ip))
	}
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j].Count5m > result[j-1].Count5m; j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}
	if n > 0 && len(result) > n {
		return result[:n]
	}
	return result
}
