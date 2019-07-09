package balancer

import (
	"math"
	"sync/atomic"
)

// Backend describe one service instance info
type Backend struct {
	skipCounter    int32
	requestCounter int32
	weight         int32
	hostaddress    string
	address        string
}

// Skip activates skip counter
func (b *Backend) Skip() {
	atomic.StoreInt32(&b.skipCounter, 7)
}

// DoSkip reduce counter of skips and returns if it necessary to skip this step
func (b *Backend) DoSkip() bool {
	counter := atomic.AddInt32(&b.skipCounter, -1)
	if counter < math.MaxInt32 {
		// Reset the counter
		if old := atomic.SwapInt32(&b.skipCounter, 0); old > 0 {
			atomic.CompareAndSwapInt32(&b.skipCounter, 0, old)
		}
	}
	return counter >= 0
}

// ConcurrentRequestCount returns current amount of concurent requests
func (b *Backend) ConcurrentRequestCount() int {
	return (int)(atomic.LoadInt32(&b.requestCounter))
}

// IncConcurrentRequest increments current request counter
func (b *Backend) IncConcurrentRequest(v int32) int32 {
	return atomic.AddInt32(&b.requestCounter, v)
}

// Address of the backend returns the IP address
func (b *Backend) Address() string {
	return b.address
}

// Hostname of the backend
func (b *Backend) Hostname() string {
	return b.hostaddress
}

type backends []*Backend

func (b backends) maxWeight() int32 {
	maxWeight := int32(-1)
	for _, backend := range b {
		if backend.weight > maxWeight {
			maxWeight = backend.weight
		}
	}
	return maxWeight
}

func (b backends) gcd() int32 {
	divisor := int32(-1)
	for _, backend := range b {
		if divisor == -1 {
			divisor = backend.weight
		} else {
			divisor = gcd(divisor, backend.weight)
		}
	}
	return divisor
}

func gcd(a, b int32) int32 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
