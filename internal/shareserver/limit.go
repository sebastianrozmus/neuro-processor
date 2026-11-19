package shareserver

import (
	"sync"
	"time"
)

// Limiter is a token bucket per address.
type Limiter struct {
	rate  float64 // tokens a second
	burst float64
	limit int
	now   func() time.Time

	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewLimiter lets each address spend burst requests at once and rate a
// second after that, keeping at most limit addresses.
func NewLimiter(rate float64, burst, limit int, now func() time.Time) *Limiter {
	return &Limiter{rate: rate, burst: float64(burst), limit: limit, now: now, buckets: map[string]*bucket{}}
}

// overflow is the bucket all new addresses share while the table is full:
// a flood from many addresses then slows the newcomers, as one generous
// bucket, instead of shutting every one of them out.
const overflow = "*"

// overflowScale is how much more the overflow bucket holds.
const overflowScale = 20

// Allow spends a token of address, reporting whether there was one.
func (l *Limiter) Allow(address string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	rate, burst := l.rate, l.burst

	b, ok := l.buckets[address]
	if !ok && len(l.buckets) >= l.limit {
		address, rate, burst = overflow, l.rate*overflowScale, l.burst*overflowScale
		b, ok = l.buckets[address]
	}

	if !ok {
		b = &bucket{tokens: burst, last: now}
		l.buckets[address] = b
	}

	b.tokens = min(burst, b.tokens+now.Sub(b.last).Seconds()*rate)
	b.last = now

	if b.tokens < 1 {
		return false
	}

	b.tokens--

	return true
}

// Sweep forgets addresses idle for longer than idle (their buckets are
// full again anyway).
func (l *Limiter) Sweep(idle time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	for address, b := range l.buckets {
		if now.Sub(b.last) > idle {
			delete(l.buckets, address)
		}
	}
}
