package ratelimit

import (
	"sync"
	"time"
)

// Bucket is a simple token bucket
type Bucket struct {
	mu         sync.Mutex
	tokens     float64
	last       time.Time
	capacity   float64
	refillPerS float64
}

func NewBucket(capacity, refillPerSecond float64) *Bucket {
	return &Bucket{
		tokens:     capacity,
		last:       time.Now(),
		capacity:   capacity,
		refillPerS: refillPerSecond,
	}
}

func (b *Bucket) Allow(n float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * b.refillPerS
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.last = now

	if b.tokens >= n {
		b.tokens -= n
		return true
	}
	return false
}
